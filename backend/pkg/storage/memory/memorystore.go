package memory

import (
	"errors"
	"github.com/hazcod/shade/pkg/events"
	"github.com/sirupsen/logrus"
	"sort"
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
	// Map of password hash -> domain
	domainMap := make(map[string][]string)

	for _, deviceData := range s.data {
		for _, eventEntry := range deviceData {
			eventDomain := strings.ToLower(eventEntry.Domain)
			passwordHash := eventEntry.Hash

			if _, ok := domainMap[passwordHash]; !ok {
				domainMap[passwordHash] = make([]string, 0)
			}

			found := false
			for _, domain := range domainMap[passwordHash] {
				if domain == eventDomain {
					found = true
					break
				}
			}

			if !found {
				domainMap[passwordHash] = append(domainMap[passwordHash], eventDomain)
			}
		}
	}

	dupes := make([][]string, 0)
	for _, userDomains := range domainMap {
		dupes = append(dupes, userDomains)
	}

	return dupes, nil
}

func (s *InMemoryStore) GetDuplicatePasswords() (map[string]map[string]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// user -> password hash -> set of domains
	userPasswordDomains := make(map[string]map[string]map[string]struct{})

	for _, deviceData := range s.data {
		for _, eventEntry := range deviceData {
			user := strings.ToLower(eventEntry.User)
			domain := strings.ToLower(eventEntry.Domain)
			hash := eventEntry.Hash

			if _, ok := userPasswordDomains[user]; !ok {
				userPasswordDomains[user] = make(map[string]map[string]struct{})
			}
			if _, ok := userPasswordDomains[user][hash]; !ok {
				userPasswordDomains[user][hash] = make(map[string]struct{})
			}
			userPasswordDomains[user][hash][domain] = struct{}{}
		}
	}

	// Build result: user -> hash -> comma-separated list of domains
	result := make(map[string]map[string]string)

	for user, hashMap := range userPasswordDomains {
		for hash, domainSet := range hashMap {
			if len(domainSet) < 2 {
				continue // not a duplicate use
			}

			if _, ok := result[user]; !ok {
				result[user] = make(map[string]string)
			}

			domains := make([]string, 0, len(domainSet))
			for d := range domainSet {
				domains = append(domains, d)
			}

			sort.Strings(domains) // optional, for consistency
			result[user][hash] = strings.Join(domains, ", ")
		}
	}

	return result, nil
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
