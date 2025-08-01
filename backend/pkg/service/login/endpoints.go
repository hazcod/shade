package login

import (
	"encoding/json"
	"github.com/asaskevich/govalidator"
	"github.com/hazcod/shade/pkg/events"
	"github.com/hazcod/shade/pkg/service/hibp"
	"github.com/hazcod/shade/pkg/storage"
	"github.com/sirupsen/logrus"
	"net"
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
	HasMFA       bool      `json:"hasMFA"`
	MFAType      string    `json:"mfaType"`
}

// getClientIP extracts the real client IP from the HTTP request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// getHostnameFromIP attempts to resolve hostname from IP address
func getHostnameFromIP(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ip // Return IP if hostname resolution fails
	}
	return strings.TrimSuffix(names[0], ".")
}

func HandleLoginData(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	// Initialize HIBP service
	hibpService := hibp.NewService(logger)
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

		// Extract real client IP and hostname
		clientIP := getClientIP(r)
		hostname := getHostnameFromIP(clientIP)

		loginEvent := events.LoginEvent{
			Timestamp: data.CapturedTime,
			User:      data.Username,
			Domain:    data.Domain,
			Hash:      data.Hash,
			DeviceID:  data.DeviceID,
			IP:        clientIP,
			Hostname:  hostname,
			HasMFA:    data.HasMFA,
			MFAType:   data.MFAType,
		}

		// Check password against HIBP
		breachCount, hibpErr := hibpService.CheckPasswordHash(data.Hash)
		hibpChecked := hibpErr == nil
		
		if hibpErr != nil {
			logger.WithError(hibpErr).WithField("hash_prefix", data.Hash[:5]).Warn("failed to check password against HIBP")
			// Continue processing even if HIBP check fails
		} else {
			// Store HIBP result in database
			if storeErr := store.StoreHIBPResult(data.Hash, breachCount); storeErr != nil {
				logger.WithError(storeErr).WithField("hash_prefix", data.Hash[:5]).Warn("failed to store HIBP result")
			}
			
			if breachCount > 0 {
				logger.WithFields(logrus.Fields{
					"username": data.Username,
					"domain": data.Domain,
					"breach_count": breachCount,
				}).Info("password found in HIBP database")
			}
		}

		// Store the login data
		if err := store.AddLoginEvent(loginEvent); err != nil {
			logger.WithError(err).WithField("body", data).Error("store add failed")
			http.Error(w, "failed to store", http.StatusInternalServerError)
			return
		}

		// Prepare response with HIBP information
		response := map[string]interface{}{
			"status":  "success",
			"message": "Login data stored successfully",
			"hibp": map[string]interface{}{
				"checked": hibpChecked,
				"breached": hibpChecked && breachCount > 0,
				"breach_count": breachCount,
			},
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.WithError(err).Error("Failed to write response")
		}
	}
}
