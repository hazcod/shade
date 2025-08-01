package password

import (
	"encoding/json"
	"github.com/asaskevich/govalidator"
	"github.com/hazcod/shade/pkg/storage"
	"github.com/sirupsen/logrus"
	"net/http"
)

type duplicatePasswordData struct {
	Username string `json:"username" valid:"required"`
}

func CheckDuplicatePassword(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var data duplicatePasswordData
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

		// Store the login data
		dupes, err := store.GetDuplicatePasswordsForUser(data.Username)
		if err != nil {
			logger.WithError(err).WithField("body", data).Error("store add failed")
			http.Error(w, "failed to store", http.StatusInternalServerError)
			return
		}

		if logger.IsLevelEnabled(logrus.DebugLevel) {
			logger.Debugf("%+v", dupes)
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(&dupes); err != nil {
			logger.WithError(err).Error("Failed to write response")
		}
	}
}

func CheckCompromisedPasswords(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}
