package hibp

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	HIBPAPIBaseURL = "https://api.pwnedpasswords.com/range/"
	UserAgent      = "shade-password-monitor"
)

// Client represents a HIBP API client
type Client struct {
	httpClient *http.Client
	logger     *logrus.Logger
	userAgent  string
}

// NewClient creates a new HIBP client
func NewClient(logger *logrus.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:    logger,
		userAgent: UserAgent,
	}
}

// CheckPassword checks if a password has been compromised using HIBP API
// Returns the number of times the password has been seen in breaches, or 0 if not found
func (c *Client) CheckPassword(password string) (int, error) {
	// Generate SHA-1 hash of the password
	hash := sha1.Sum([]byte(password))
	hashStr := strings.ToUpper(hex.EncodeToString(hash[:]))
	
	// Use k-anonymity: send only first 5 characters of hash
	prefix := hashStr[:5]
	suffix := hashStr[5:]
	
	// Make request to HIBP API
	url := HIBPAPIBaseURL + prefix
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", c.userAgent)
	
	c.logger.WithField("prefix", prefix).Debug("checking password hash prefix with HIBP")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request to HIBP: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HIBP API returned status %d", resp.StatusCode)
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Parse response to find our hash suffix
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		
		if strings.EqualFold(parts[0], suffix) {
			count, err := strconv.Atoi(parts[1])
			if err != nil {
				c.logger.WithError(err).WithField("count_str", parts[1]).Warn("failed to parse breach count")
				return 0, fmt.Errorf("failed to parse breach count: %w", err)
			}
			
			c.logger.WithFields(logrus.Fields{
				"prefix": prefix,
				"count":  count,
			}).Debug("password found in breaches")
			
			return count, nil
		}
	}
	
	// Hash not found in breaches
	c.logger.WithField("prefix", prefix).Debug("password not found in breaches")
	return 0, nil
}

// CheckPasswordHash checks if a password hash has been compromised
// Expects a SHA-1 hash in uppercase hex format
func (c *Client) CheckPasswordHash(hashStr string) (int, error) {
	if len(hashStr) != 40 {
		return 0, fmt.Errorf("invalid hash length: expected 40 characters, got %d", len(hashStr))
	}
	
	hashStr = strings.ToUpper(hashStr)
	prefix := hashStr[:5]
	suffix := hashStr[5:]
	
	// Make request to HIBP API
	url := HIBPAPIBaseURL + prefix
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", c.userAgent)
	
	c.logger.WithField("prefix", prefix).Debug("checking password hash prefix with HIBP")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request to HIBP: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HIBP API returned status %d", resp.StatusCode)
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Parse response to find our hash suffix
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		
		if strings.EqualFold(parts[0], suffix) {
			count, err := strconv.Atoi(parts[1])
			if err != nil {
				c.logger.WithError(err).WithField("count_str", parts[1]).Warn("failed to parse breach count")
				return 0, fmt.Errorf("failed to parse breach count: %w", err)
			}
			
			c.logger.WithFields(logrus.Fields{
				"prefix": prefix,
				"count":  count,
			}).Debug("password hash found in breaches")
			
			return count, nil
		}
	}
	
	// Hash not found in breaches
	c.logger.WithField("prefix", prefix).Debug("password hash not found in breaches")
	return 0, nil
}