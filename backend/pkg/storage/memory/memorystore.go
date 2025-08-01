package memory

import (
	"errors"
	"fmt"
	"github.com/hazcod/shade/pkg/events"
	"github.com/hazcod/shade/pkg/models"
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
	"sync"
	"time"
)

type InMemoryStore struct {
	logger *logrus.Logger

	mutex       sync.RWMutex
	data        map[string][]events.LoginEvent
	hibpResults map[string]int // passwordHash -> breachCount
	token       string
}

func (s *InMemoryStore) Init(logger *logrus.Logger, settings map[string]string) error {
	s.data = make(map[string][]events.LoginEvent)
	s.hibpResults = make(map[string]int)
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

func (s *InMemoryStore) GetCompromisedPasswords() (map[string]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	compromised := make(map[string]string)

	// Return password hashes that have breach counts > 0
	for hash, breachCount := range s.hibpResults {
		if breachCount > 0 {
			// Map hash to breach count as string
			compromised[hash] = fmt.Sprintf("%d", breachCount)
		}
	}

	return compromised, nil
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

func (s *InMemoryStore) GetEnrolledUsers() ([]models.EnrolledUser, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	userMap := make(map[string]models.EnrolledUser)

	for deviceID, deviceEvents := range s.data {
		if len(deviceEvents) == 0 {
			continue
		}

		// Get the most recent event for this device
		latestEvent := deviceEvents[len(deviceEvents)-1]
		user := strings.ToLower(latestEvent.User)

		// Use real IP and hostname from the event data
		ip := latestEvent.IP
		hostname := latestEvent.Hostname

		// Fallback to device ID if IP/hostname are empty (for backward compatibility)
		if ip == "" {
			ip = "Unknown"
		}
		
		if hostname == "" {
			hostname = "Unknown"
		}

		userMap[user] = models.EnrolledUser{
			Username: user,
			ID:       deviceID,
			Hostname: hostname,
			IP:       ip,
			LastSeen: latestEvent.Timestamp.Format("2006-01-02 15:04:05"),
		}
	}

	users := make([]models.EnrolledUser, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}

	return users, nil
}

func (s *InMemoryStore) GetDashboardStats() (models.DashboardStats, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	userSet := make(map[string]struct{})
	domainSet := make(map[string]struct{})

	for _, events := range s.data {
		for _, event := range events {
			userSet[strings.ToLower(event.User)] = struct{}{}
			domainSet[strings.ToLower(event.Domain)] = struct{}{}
		}
	}

	duplicatePasswords, _ := s.GetDuplicatePasswords()
	duplicateCount := 0
	for _, userDupes := range duplicatePasswords {
		duplicateCount += len(userDupes)
	}

	compromisedPasswords, _ := s.GetCompromisedPasswords()

	usersWithoutMFA, _ := s.GetUsersWithoutMFA()

	return models.DashboardStats{
		TotalUsers:           len(userSet),
		TotalDomains:         len(domainSet),
		DuplicatePasswords:   duplicateCount,
		CompromisedPasswords: len(compromisedPasswords),
		UsersWithoutMFA:      len(usersWithoutMFA),
	}, nil
}

func (s *InMemoryStore) GetUsersWithoutMFA() ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	userMFAStatus := make(map[string]bool)

	// Check all events to determine MFA status for each user
	for _, events := range s.data {
		for _, event := range events {
			user := strings.ToLower(event.User)
			// If any event for this user has MFA, mark them as having MFA
			if event.HasMFA {
				userMFAStatus[user] = true
			} else if _, exists := userMFAStatus[user]; !exists {
				// Only set to false if we haven't seen MFA for this user yet
				userMFAStatus[user] = false
			}
		}
	}

	// Collect users without MFA
	users := make([]string, 0)
	for user, hasMFA := range userMFAStatus {
		if !hasMFA {
			users = append(users, user)
		}
	}

	return users, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StoreHIBPResult stores a HIBP breach count for a password hash
func (s *InMemoryStore) StoreHIBPResult(passwordHash string, breachCount int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.hibpResults[passwordHash] = breachCount

	s.logger.WithFields(logrus.Fields{
		"hash_prefix":  passwordHash[:5],
		"breach_count": breachCount,
	}).Debug("stored HIBP result")

	return nil
}

// GetHIBPResult retrieves a HIBP breach count for a password hash
func (s *InMemoryStore) GetHIBPResult(passwordHash string) (int, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	breachCount, exists := s.hibpResults[passwordHash]
	return breachCount, exists, nil
}

// GetAllPasswordHashes returns all unique password hashes from login events
func (s *InMemoryStore) GetAllPasswordHashes() ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	hashSet := make(map[string]struct{})

	for _, events := range s.data {
		for _, event := range events {
			hashSet[event.Hash] = struct{}{}
		}
	}

	hashes := make([]string, 0, len(hashSet))
	for hash := range hashSet {
		hashes = append(hashes, hash)
	}

	return hashes, nil
}
