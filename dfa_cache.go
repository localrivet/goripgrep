package goripgrep

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// DFACache provides thread-safe caching of compiled regular expressions
type DFACache struct {
	cache   map[string]*CachedRegex
	mutex   sync.RWMutex
	maxSize int
	ttl     time.Duration
	hits    int64
	misses  int64
	evicted int64
}

// CachedRegex represents a cached compiled regex with metadata
type CachedRegex struct {
	regex     *regexp.Regexp
	pattern   string
	flags     string
	createdAt time.Time
	lastUsed  time.Time
	useCount  int64
}

// NewDFACache creates a new DFA cache with specified parameters
func NewDFACache(maxSize int, ttl time.Duration) *DFACache {
	if maxSize <= 0 {
		maxSize = 1000 // Default cache size
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute // Default TTL
	}

	cache := &DFACache{
		cache:   make(map[string]*CachedRegex),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// GetOrCompile retrieves a cached regex or compiles and caches a new one
func (c *DFACache) GetOrCompile(pattern string, flags string) (*regexp.Regexp, error) {
	key := c.generateKey(pattern, flags)

	// Try to get from cache first
	c.mutex.RLock()
	cached, exists := c.cache[key]
	if exists && !c.isExpired(cached) {
		cached.lastUsed = time.Now()
		cached.useCount++
		c.hits++
		c.mutex.RUnlock()
		return cached.regex, nil
	}
	c.mutex.RUnlock()

	// Cache miss - need to compile
	c.misses++

	// Compile the regex
	fullPattern := flags + pattern
	regex, err := regexp.Compile(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex pattern %q: %w", pattern, err)
	}

	// Cache the compiled regex
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we need to evict entries
	if len(c.cache) >= c.maxSize {
		c.evictLRU()
	}

	now := time.Now()
	c.cache[key] = &CachedRegex{
		regex:     regex,
		pattern:   pattern,
		flags:     flags,
		createdAt: now,
		lastUsed:  now,
		useCount:  1,
	}

	return regex, nil
}

// generateKey creates a unique key for the pattern and flags combination
func (c *DFACache) generateKey(pattern string, flags string) string {
	// Use SHA256 hash to create a consistent key
	hasher := sha256.New()
	hasher.Write([]byte(flags))
	hasher.Write([]byte(pattern))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// isExpired checks if a cached entry has expired
func (c *DFACache) isExpired(cached *CachedRegex) bool {
	return time.Since(cached.createdAt) > c.ttl
}

// evictLRU removes the least recently used entry from the cache
func (c *DFACache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, cached := range c.cache {
		if first || cached.lastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.lastUsed
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
		c.evicted++
	}
}

// cleanupExpired periodically removes expired entries
func (c *DFACache) cleanupExpired() {
	ticker := time.NewTicker(c.ttl / 4) // Clean up 4 times per TTL period
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		for key, cached := range c.cache {
			if c.isExpired(cached) {
				delete(c.cache, key)
				c.evicted++
			}
		}
		c.mutex.Unlock()
	}
}

// Clear removes all entries from the cache
func (c *DFACache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache = make(map[string]*CachedRegex)
}

// Size returns the current number of cached entries
func (c *DFACache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.cache)
}

// Stats returns cache statistics
func (c *DFACache) Stats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	total := c.hits + c.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return CacheStats{
		Size:    len(c.cache),
		MaxSize: c.maxSize,
		Hits:    c.hits,
		Misses:  c.misses,
		Evicted: c.evicted,
		HitRate: hitRate,
		TTL:     c.ttl,
	}
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	Size    int           `json:"size"`
	MaxSize int           `json:"max_size"`
	Hits    int64         `json:"hits"`
	Misses  int64         `json:"misses"`
	Evicted int64         `json:"evicted"`
	HitRate float64       `json:"hit_rate"`
	TTL     time.Duration `json:"ttl"`
}

// String returns a human-readable representation of cache stats
func (s CacheStats) String() string {
	return fmt.Sprintf("Cache Stats: Size=%d/%d, Hits=%d, Misses=%d, Evicted=%d, Hit Rate=%.2f%%, TTL=%v",
		s.Size, s.MaxSize, s.Hits, s.Misses, s.Evicted, s.HitRate*100, s.TTL)
}

// GetCachedPatterns returns information about all cached patterns
func (c *DFACache) GetCachedPatterns() []PatternInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	patterns := make([]PatternInfo, 0, len(c.cache))
	for _, cached := range c.cache {
		patterns = append(patterns, PatternInfo{
			Pattern:   cached.pattern,
			Flags:     cached.flags,
			CreatedAt: cached.createdAt,
			LastUsed:  cached.lastUsed,
			UseCount:  cached.useCount,
			Age:       time.Since(cached.createdAt),
		})
	}

	return patterns
}

// PatternInfo provides information about a cached pattern
type PatternInfo struct {
	Pattern   string        `json:"pattern"`
	Flags     string        `json:"flags"`
	CreatedAt time.Time     `json:"created_at"`
	LastUsed  time.Time     `json:"last_used"`
	UseCount  int64         `json:"use_count"`
	Age       time.Duration `json:"age"`
}

// Global DFA cache instance
var globalDFACache *DFACache
var cacheOnce sync.Once

// GetGlobalDFACache returns the global DFA cache instance
func GetGlobalDFACache() *DFACache {
	cacheOnce.Do(func() {
		globalDFACache = NewDFACache(1000, 30*time.Minute)
	})
	return globalDFACache
}

// CompileWithCache compiles a regex using the global cache
func CompileWithCache(pattern string, ignoreCase bool) (*regexp.Regexp, error) {
	flags := ""
	if ignoreCase {
		flags = "(?i)"
	}

	cache := GetGlobalDFACache()
	return cache.GetOrCompile(pattern, flags)
}

// MustCompileWithCache compiles a regex using the global cache and panics on error
func MustCompileWithCache(pattern string, ignoreCase bool) *regexp.Regexp {
	regex, err := CompileWithCache(pattern, ignoreCase)
	if err != nil {
		panic(err)
	}
	return regex
}
