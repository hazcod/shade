package login

import (
	"encoding/json"
	"github.com/asaskevich/govalidator"
	"github.com/hazcod/shade/pkg/events"
	"github.com/hazcod/shade/pkg/storage"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

type loginData struct {
	Domain       string    `json:"domain" valid:"required"`
	Username     string    `json:"username" valid:"required"`
	Hash         string    `json:"hash" valid:"required"`
	DeviceID     string    `json:"device_id" valid:"required"`
	CapturedTime time.Time `json:"captured_time"`
}

func HandleLoginData(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var data loginData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		valid, err := govalidator.ValidateStruct(data)
		if !valid || err != nil {
			logger.WithError(err).WithField("body", data).Error("endpoint data validation failed")
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Set capture time if not provided
		if data.CapturedTime.IsZero() {
			data.CapturedTime = time.Now()
		}

		data.Domain = strings.ToLower(data.Domain)
		data.Username = strings.ToLower(data.Username)

		loginEvent := events.LoginEvent{
			Timestamp: data.CapturedTime,
			User:      data.Username,
			Domain:    data.Domain,
			Hash:      data.Hash,
			DeviceID:  data.DeviceID,
		}

		// Store the login data
		if err := store.AddLoginEvent(loginEvent); err != nil {
			logger.WithError(err).WithField("body", data).Error("store add failed")
			http.Error(w, "failed to store", http.StatusInternalServerError)
			return
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Login data stored successfully",
		}); err != nil {
			logger.WithError(err).Error("Failed to write response")
		}
	}
}
