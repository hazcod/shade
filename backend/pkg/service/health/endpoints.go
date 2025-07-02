package health

import (
	"encoding/json"
	"github.com/hazcod/shade/pkg/storage"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

func HandleHealthCheck(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")

		if token != "foo" {
			logger.WithField("ip", r.RemoteAddr).Warn("invalid token")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{}); err != nil {
			logger.WithError(err).Error("Failed to write response")
		}
	}
}
