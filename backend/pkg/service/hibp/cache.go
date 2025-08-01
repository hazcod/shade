package hibp

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CacheEntry represents a cached HIBP result
type CacheEntry struct {
	BreachCount int
	Timestamp   time.Time
}

// Cache represents an in-memory cache for HIBP results
type Cache struct {
	entries map[string]CacheEntry
	mutex   sync.RWMutex
	logger  *logrus.Logger
	ttl     time.Duration
}

// NewCache creates a new HIBP cache with 1-hour TTL
func NewCache(logger *logrus.Logger) *Cache {
	cache := &Cache{
		entries: make(map[string]CacheEntry),
		logger:  logger,
		ttl:     time.Hour, // 1 hour cache as specified
	}
	
	// Start cleanup goroutine
	go cache.cleanup()
	
	return cache
}

// Get retrieves a cached result for a password hash
func (c *Cache) Get(passwordHash string) (int, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	entry, exists := c.entries[passwordHash]
	if !exists {
		return 0, false
	}
	
	// Check if entry has expired
	if time.Since(entry.Timestamp) > c.ttl {
		c.logger.WithField("hash_prefix", passwordHash[:5]).Debug("cache entry expired")
		return 0, false
	}
	
	c.logger.WithFields(logrus.Fields{
		"hash_prefix": passwordHash[:5],
		"breach_count": entry.BreachCount,
		"age": time.Since(entry.Timestamp).String(),
	}).Debug("cache hit for password hash")
	
	return entry.BreachCount, true
}

// Set stores a result in the cache
func (c *Cache) Set(passwordHash string, breachCount int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.entries[passwordHash] = CacheEntry{
		BreachCount: breachCount,
		Timestamp:   time.Now(),
	}
	
	c.logger.WithFields(logrus.Fields{
		"hash_prefix": passwordHash[:5],
		"breach_count": breachCount,
	}).Debug("cached HIBP result")
}

// cleanup removes expired entries from the cache
func (c *Cache) cleanup() {
	ticker := time.NewTicker(30 * time.Minute) // Clean up every 30 minutes
	defer ticker.Stop()
	
	for range ticker.C {
		c.mutex.Lock()
		
		now := time.Now()
		expired := 0
		
		for hash, entry := range c.entries {
			if now.Sub(entry.Timestamp) > c.ttl {
				delete(c.entries, hash)
				expired++
			}
		}
		
		if expired > 0 {
			c.logger.WithFields(logrus.Fields{
				"expired_entries": expired,
				"remaining_entries": len(c.entries),
			}).Debug("cleaned up expired cache entries")
		}
		
		c.mutex.Unlock()
	}
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return map[string]interface{}{
		"total_entries": len(c.entries),
		"ttl_hours":     c.ttl.Hours(),
	}
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	cleared := len(c.entries)
	c.entries = make(map[string]CacheEntry)
	
	c.logger.WithField("cleared_entries", cleared).Info("cleared HIBP cache")
}