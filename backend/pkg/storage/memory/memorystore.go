package memory

import (
	"errors"
	"github.com/hazcod/shade/pkg/events"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

type InMemoryStore struct {
	logger *logrus.Logger

	mutex sync.RWMutex
	data  map[string][]events.LoginEvent
	token string
}

func (s *InMemoryStore) Init(logger *logrus.Logger, settings map[string]string) error {
	s.data = make(map[string][]events.LoginEvent)
	s.logger = logger

	token, ok := settings["token"]
	if !ok || token == "" {
		return errors.New("token required for memory store")
	}

	s.token = token

	return nil
}

func (s *InMemoryStore) GetAllDomains() ([]string, error) {
	domains := make(map[string]struct{})

	for _, deviceID := range s.data {
		for _, eventEntry := range deviceID {
			domain := strings.ToLower(eventEntry.Domain)

			_, found := domains[domain]
			if found {
				continue
			}

			domains[domain] = struct{}{}
		}
	}

	var allDomains []string
	for k := range domains {
		allDomains = append(allDomains, k)
	}

	return allDomains, nil
}

func (s *InMemoryStore) GetDomainsForUser(username string) ([]string, error) {
	domains := make(map[string]struct{})

	for _, deviceID := range s.data {
		for _, eventEntry := range deviceID {
			eventUsername := strings.ToLower(eventEntry.User)

			if !strings.EqualFold(username, eventUsername) {
				continue
			}

			domain := strings.ToLower(eventEntry.Domain)

			_, found := domains[domain]
			if found {
				continue
			}

			domains[domain] = struct{}{}
		}
	}

	var allDomains []string
	for k := range domains {
		allDomains = append(allDomains, k)
	}

	return allDomains, nil
}

func (s *InMemoryStore) GetDuplicatePasswordsForUser(username string) ([][]string, error) {
	return nil, nil
}

func (s *InMemoryStore) GetDuplicatePasswords() (map[string]map[string]string, error) {
	// Map of password hash -> username -> domain
	passwordMap := make(map[string]map[string]string)

	for _, deviceData := range s.data {
		for _, eventEntry := range deviceData {
			eventUsername := strings.ToLower(eventEntry.User)
			eventDomain := strings.ToLower(eventEntry.Domain)
			passwordHash := eventEntry.Hash

			if _, ok := passwordMap[passwordHash]; !ok {
				passwordMap[passwordHash] = make(map[string]string)
			}

			// Store the domain if this password has already been seen for this user with a different domain
			if existingDomain, exists := passwordMap[passwordHash][eventUsername]; exists {
				if existingDomain != eventDomain {
					// Duplicate password found for the same user across different domains
					// keep both
					passwordMap[passwordHash][eventUsername] = existingDomain + "," + eventDomain
				}
			} else {
				passwordMap[passwordHash][eventUsername] = eventDomain
			}
		}
	}

	// Flatten only entries where the password is reused (i.e. multiple domains per user)
	duplicates := make(map[string]map[string]string)
	for hash, userDomains := range passwordMap {
		for user, domainStr := range userDomains {
			if strings.Contains(domainStr, ",") {
				if _, ok := duplicates[user]; !ok {
					duplicates[user] = make(map[string]string)
				}
				duplicates[user][hash] = domainStr
			}
		}
	}

	return duplicates, nil
}

func (s *InMemoryStore) IsValidToken(token string) (bool, error) {
	if s.token != token {
		return false, nil
	}

	return true, nil
}

func (s *InMemoryStore) IsDuplicatePassword(username, passwordHash string) ([]string, error) {
	domains := make(map[string]struct{})

	for _, deviceID := range s.data {
		for _, eventEntry := range deviceID {
			eventUsername := strings.ToLower(eventEntry.User)

			if !strings.EqualFold(username, eventUsername) {
				continue
			}

			if eventEntry.Hash != passwordHash {
				continue
			}

			domain := strings.ToLower(eventEntry.Domain)

			_, exists := domains[domain]
			if exists {
				continue
			}

			domains[domain] = struct{}{}
		}
	}

	var allDomains []string
	for k := range domains {
		allDomains = append(allDomains, k)
	}

	return allDomains, nil
}

func (s *InMemoryStore) AddLoginEvent(data events.LoginEvent) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.data[data.DeviceID] = append(s.data[data.User], data)

	s.logger.WithFields(logrus.Fields{
		"device_id": data.DeviceID,
		"username":  data.User,
		"timestamp": data.Timestamp.Format(time.DateTime),
		"domain":    data.Domain,
	}).Debug("captured login event")

	return nil
}
