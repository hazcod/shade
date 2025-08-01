package hibp

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Service represents the HIBP service with caching
type Service struct {
	client *Client
	cache  *Cache
	logger *logrus.Logger
}

// NewService creates a new HIBP service with client and cache
func NewService(logger *logrus.Logger) *Service {
	return &Service{
		client: NewClient(logger),
		cache:  NewCache(logger),
		logger: logger,
	}
}

// CheckPassword checks if a password has been compromised, using cache when possible
func (s *Service) CheckPassword(password string) (int, error) {
	// Generate SHA-1 hash of the password
	hash := sha1.Sum([]byte(password))
	hashStr := strings.ToUpper(hex.EncodeToString(hash[:]))
	
	return s.CheckPasswordHash(hashStr)
}

// CheckPasswordHash checks if a password hash has been compromised, using cache when possible
func (s *Service) CheckPasswordHash(passwordHash string) (int, error) {
	// Check cache first
	if breachCount, found := s.cache.Get(passwordHash); found {
		return breachCount, nil
	}
	
	// Cache miss - check with HIBP API
	s.logger.WithField("hash_prefix", passwordHash[:5]).Debug("cache miss, checking HIBP API")
	
	breachCount, err := s.client.CheckPasswordHash(passwordHash)
	if err != nil {
		return 0, err
	}
	
	// Cache the result
	s.cache.Set(passwordHash, breachCount)
	
	return breachCount, nil
}

// CheckPasswordWithResult returns detailed information about the password check
type CheckResult struct {
	PasswordHash string
	BreachCount  int
	IsBreached   bool
	CheckedAt    time.Time
	FromCache    bool
}

// CheckPasswordWithDetails checks a password and returns detailed results
func (s *Service) CheckPasswordWithDetails(password string) (*CheckResult, error) {
	// Generate SHA-1 hash of the password
	hash := sha1.Sum([]byte(password))
	hashStr := strings.ToUpper(hex.EncodeToString(hash[:]))
	
	return s.CheckPasswordHashWithDetails(hashStr)
}

// CheckPasswordHashWithDetails checks a password hash and returns detailed results
func (s *Service) CheckPasswordHashWithDetails(passwordHash string) (*CheckResult, error) {
	result := &CheckResult{
		PasswordHash: passwordHash,
		CheckedAt:    time.Now(),
	}
	
	// Check cache first
	if breachCount, found := s.cache.Get(passwordHash); found {
		result.BreachCount = breachCount
		result.IsBreached = breachCount > 0
		result.FromCache = true
		return result, nil
	}
	
	// Cache miss - check with HIBP API
	s.logger.WithField("hash_prefix", passwordHash[:5]).Debug("cache miss, checking HIBP API")
	
	breachCount, err := s.client.CheckPasswordHash(passwordHash)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	s.cache.Set(passwordHash, breachCount)
	
	result.BreachCount = breachCount
	result.IsBreached = breachCount > 0
	result.FromCache = false
	
	return result, nil
}

// BatchCheckPasswordHashes checks multiple password hashes
func (s *Service) BatchCheckPasswordHashes(passwordHashes []string) (map[string]*CheckResult, error) {
	results := make(map[string]*CheckResult)
	
	for _, hash := range passwordHashes {
		result, err := s.CheckPasswordHashWithDetails(hash)
		if err != nil {
			s.logger.WithError(err).WithField("hash_prefix", hash[:5]).Error("failed to check password hash")
			// Continue with other hashes even if one fails
			results[hash] = &CheckResult{
				PasswordHash: hash,
				BreachCount:  -1, // Indicate error
				IsBreached:   false,
				CheckedAt:    time.Now(),
				FromCache:    false,
			}
			continue
		}
		results[hash] = result
	}
	
	return results, nil
}

// GetCacheStats returns cache statistics
func (s *Service) GetCacheStats() map[string]interface{} {
	return s.cache.Stats()
}

// ClearCache clears the HIBP cache
func (s *Service) ClearCache() {
	s.cache.Clear()
}

// IsPasswordBreached is a convenience method that returns true if password is breached
func (s *Service) IsPasswordBreached(password string) (bool, error) {
	count, err := s.CheckPassword(password)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsPasswordHashBreached is a convenience method that returns true if password hash is breached
func (s *Service) IsPasswordHashBreached(passwordHash string) (bool, error) {
	count, err := s.CheckPasswordHash(passwordHash)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}