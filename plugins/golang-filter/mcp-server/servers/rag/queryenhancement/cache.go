package queryenhancement

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// InMemoryCache implements a simple in-memory cache for query enhancements
type InMemoryCache struct {
	cache        map[string]*CacheEntry
	mutex        sync.RWMutex
	maxSize      int
	accessed     map[string]time.Time
	hits         int64  // Cache hit counter
	misses       int64  // Cache miss counter
	evictions    int64  // Cache eviction counter
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// CacheEntry represents a cached query enhancement
type CacheEntry struct {
	Query    string         `json:"query"`
	Options  string         `json:"options"` // Serialized options
	Enhanced *EnhancedQuery `json:"enhanced"`
	Expires  time.Time      `json:"expires"`
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache(maxSize int) *InMemoryCache {
	if maxSize <= 0 {
		maxSize = 1000 // Default size
	}
	
	cache := &InMemoryCache{
		cache:       make(map[string]*CacheEntry),
		maxSize:     maxSize,
		accessed:    make(map[string]time.Time),
		stopCleanup: make(chan struct{}),
	}
	
	// Start background cleanup goroutine
	cache.cleanupTicker = time.NewTicker(5 * time.Minute)
	go cache.backgroundCleanup()
	
	return cache
}

// Get retrieves a cached query enhancement
func (c *InMemoryCache) Get(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	key := c.generateKey(query, options)
	entry, exists := c.cache[key]
	
	if !exists {
		atomic.AddInt64(&c.misses, 1)
		return nil, ErrCacheMiss
	}
	
	// Check expiration
	if time.Now().After(entry.Expires) {
		atomic.AddInt64(&c.misses, 1)
		// Don't delete here to avoid deadlock, will be cleaned up later
		return nil, ErrCacheExpired
	}
	
	// Update access time
	c.accessed[key] = time.Now()
	atomic.AddInt64(&c.hits, 1)
	
	return entry.Enhanced, nil
}

// Set stores a query enhancement in cache
func (c *InMemoryCache) Set(ctx context.Context, query string, options *EnhancementOptions, enhanced *EnhancedQuery, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	key := c.generateKey(query, options)
	
	// Clean up if cache is full
	if len(c.cache) >= c.maxSize {
		c.evictLRU()
		atomic.AddInt64(&c.evictions, 1)
	}
	
	entry := &CacheEntry{
		Query:    query,
		Enhanced: enhanced,
		Expires:  time.Now().Add(ttl),
	}
	
	// Serialize options
	if optionsBytes, err := json.Marshal(options); err == nil {
		entry.Options = string(optionsBytes)
	}
	
	c.cache[key] = entry
	c.accessed[key] = time.Now()
	
	return nil
}

// Delete removes a cached entry
func (c *InMemoryCache) Delete(ctx context.Context, query string, options *EnhancementOptions) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	key := c.generateKey(query, options)
	delete(c.cache, key)
	delete(c.accessed, key)
	
	return nil
}

// Clear removes all cached entries
func (c *InMemoryCache) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache = make(map[string]*CacheEntry)
	c.accessed = make(map[string]time.Time)
	
	return nil
}

// Close stops the cache and cleanup goroutines
func (c *InMemoryCache) Close() error {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}
	close(c.stopCleanup)
	return nil
}

// Stats returns cache statistics
func (c *InMemoryCache) Stats(ctx context.Context) (*CacheStats, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	expired := 0
	now := time.Now()
	
	for _, entry := range c.cache {
		if now.After(entry.Expires) {
			expired++
		}
	}
	
	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)
	evictions := atomic.LoadInt64(&c.evictions)
	
	var hitRate float64
	if hits+misses > 0 {
		hitRate = float64(hits) / float64(hits+misses) * 100
	}
	
	return &CacheStats{
		Size:        len(c.cache),
		MaxSize:     c.maxSize,
		Expired:     expired,
		UsedPercent: float64(len(c.cache)) / float64(c.maxSize) * 100,
		HitRate:     hitRate,
		Hits:        hits,
		Misses:      misses,
		Evictions:   evictions,
	}, nil
}

// generateKey creates a cache key from query and options
func (c *InMemoryCache) generateKey(query string, options *EnhancementOptions) string {
	if options == nil {
		// Use MD5 hash for long queries to avoid memory issues
		if len(query) > 256 {
			hasher := md5.New()
			hasher.Write([]byte(query))
			return hex.EncodeToString(hasher.Sum(nil))
		}
		return query
	}
	
	// Create a deterministic key by combining query with key options
	key := query
	if options.EnableRewrite {
		key += "|rewrite"
	}
	if options.EnableExpansion {
		key += "|expansion"
	}
	if options.EnableDecomposition {
		key += "|decomposition"
	}
	if options.EnableIntentClassification {
		key += "|intent"
	}
	
	// Use MD5 hash for long keys to avoid memory issues
	if len(key) > 512 {
		hasher := md5.New()
		hasher.Write([]byte(key))
		return hex.EncodeToString(hasher.Sum(nil))
	}
	
	return key
}

// evictLRU removes least recently used entries
func (c *InMemoryCache) evictLRU() {
	if len(c.cache) == 0 {
		return
	}
	
	// Find least recently used key
	var oldestKey string
	var oldestTime time.Time
	first := true
	
	for key, accessTime := range c.accessed {
		if first || accessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = accessTime
			first = false
		}
	}
	
	// Remove oldest entry
	if oldestKey != "" {
		delete(c.cache, oldestKey)
		delete(c.accessed, oldestKey)
	}
}

// backgroundCleanup runs periodic cleanup of expired entries
func (c *InMemoryCache) backgroundCleanup() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.CleanupExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// CleanupExpired removes expired entries
func (c *InMemoryCache) CleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now()
	var expiredKeys []string
	
	for key, entry := range c.cache {
		if now.After(entry.Expires) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	for _, key := range expiredKeys {
		delete(c.cache, key)
		delete(c.accessed, key)
	}
}

// DistributedCache implements a distributed cache interface
type DistributedCache struct {
	client   DistributedCacheClient
	prefix   string
	timeout  time.Duration
}

// DistributedCacheClient interface for distributed cache implementations
type DistributedCacheClient interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// NewDistributedCache creates a new distributed cache
func NewDistributedCache(client DistributedCacheClient, prefix string, timeout time.Duration) *DistributedCache {
	return &DistributedCache{
		client:  client,
		prefix:  prefix,
		timeout: timeout,
	}
}

// Get retrieves a cached query enhancement from distributed cache
func (d *DistributedCache) Get(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	
	key := d.generateKey(query, options)
	data, err := d.client.Get(ctx, key)
	if err != nil {
		return nil, ErrCacheMiss
	}
	
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	
	// Check expiration
	if time.Now().After(entry.Expires) {
		_ = d.client.Delete(ctx, key) // Cleanup expired entry
		return nil, ErrCacheExpired
	}
	
	return entry.Enhanced, nil
}

// Set stores a query enhancement in distributed cache
func (d *DistributedCache) Set(ctx context.Context, query string, options *EnhancementOptions, enhanced *EnhancedQuery, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	
	key := d.generateKey(query, options)
	
	entry := &CacheEntry{
		Query:    query,
		Enhanced: enhanced,
		Expires:  time.Now().Add(ttl),
	}
	
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	
	return d.client.Set(ctx, key, data, ttl)
}

// Delete removes a cached entry from distributed cache
func (d *DistributedCache) Delete(ctx context.Context, query string, options *EnhancementOptions) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	
	key := d.generateKey(query, options)
	return d.client.Delete(ctx, key)
}

// Clear is not implemented for distributed cache (too expensive)
func (d *DistributedCache) Clear(ctx context.Context) error {
	return ErrNotSupported
}

// Stats returns basic cache statistics for distributed cache
func (d *DistributedCache) Stats(ctx context.Context) (*CacheStats, error) {
	// Distributed cache stats would require implementation-specific logic
	return &CacheStats{
		Size:        -1, // Unknown
		MaxSize:     -1, // Unknown
		Expired:     -1, // Unknown
		UsedPercent: -1, // Unknown
	}, nil
}

// generateKey creates a cache key with prefix
func (d *DistributedCache) generateKey(query string, options *EnhancementOptions) string {
	baseKey := query
	if options != nil {
		// Create options signature
		optionsStr := ""
		if options.EnableRewrite {
			optionsStr += "R"
		}
		if options.EnableExpansion {
			optionsStr += "E"
		}
		if options.EnableDecomposition {
			optionsStr += "D"
		}
		if options.EnableIntentClassification {
			optionsStr += "I"
		}
		baseKey += "|" + optionsStr
	}
	
	return d.prefix + ":" + baseKey
}

// CacheStats contains cache statistics
type CacheStats struct {
	Size        int     `json:"size"`
	MaxSize     int     `json:"max_size"`
	Expired     int     `json:"expired"`
	UsedPercent float64 `json:"used_percent"`
	HitRate     float64 `json:"hit_rate"`
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Evictions   int64   `json:"evictions"`
}

// Cache errors
var (
	ErrCacheMiss     = NewCacheError("cache miss")
	ErrCacheExpired  = NewCacheError("cache expired")
	ErrNotSupported  = NewCacheError("operation not supported")
)

// CacheError represents a cache-related error
type CacheError struct {
	Message string
}

func NewCacheError(message string) *CacheError {
	return &CacheError{Message: message}
}

func (e *CacheError) Error() string {
	return e.Message
}

// LRUCache implements an LRU (Least Recently Used) cache
type LRUCache struct {
	cache     map[string]*LRUNode
	head      *LRUNode
	tail      *LRUNode
	maxSize   int
	mutex     sync.RWMutex
}

// LRUNode represents a node in the LRU cache
type LRUNode struct {
	Key      string
	Entry    *CacheEntry
	Previous *LRUNode
	Next     *LRUNode
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(maxSize int) *LRUCache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	
	cache := &LRUCache{
		cache:   make(map[string]*LRUNode),
		maxSize: maxSize,
	}
	
	// Initialize dummy head and tail nodes
	cache.head = &LRUNode{}
	cache.tail = &LRUNode{}
	cache.head.Next = cache.tail
	cache.tail.Previous = cache.head
	
	return cache
}

// Get retrieves a cached query enhancement from LRU cache
func (l *LRUCache) Get(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	key := l.generateKey(query, options)
	node, exists := l.cache[key]
	
	if !exists {
		return nil, ErrCacheMiss
	}
	
	// Check expiration
	if time.Now().After(node.Entry.Expires) {
		l.removeNode(node)
		delete(l.cache, key)
		return nil, ErrCacheExpired
	}
	
	// Move to front (most recently used)
	l.moveToFront(node)
	
	return node.Entry.Enhanced, nil
}

// Set stores a query enhancement in LRU cache
func (l *LRUCache) Set(ctx context.Context, query string, options *EnhancementOptions, enhanced *EnhancedQuery, ttl time.Duration) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	key := l.generateKey(query, options)
	
	if node, exists := l.cache[key]; exists {
		// Update existing entry
		node.Entry = &CacheEntry{
			Query:    query,
			Enhanced: enhanced,
			Expires:  time.Now().Add(ttl),
		}
		l.moveToFront(node)
	} else {
		// Add new entry
		entry := &CacheEntry{
			Query:    query,
			Enhanced: enhanced,
			Expires:  time.Now().Add(ttl),
		}
		
		node := &LRUNode{
			Key:   key,
			Entry: entry,
		}
		
		l.cache[key] = node
		l.addToFront(node)
		
		// Evict if over capacity
		if len(l.cache) > l.maxSize {
			l.evictLast()
		}
	}
	
	return nil
}

// Delete removes a cached entry from LRU cache
func (l *LRUCache) Delete(ctx context.Context, query string, options *EnhancementOptions) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	key := l.generateKey(query, options)
	if node, exists := l.cache[key]; exists {
		l.removeNode(node)
		delete(l.cache, key)
	}
	
	return nil
}

// Clear removes all cached entries from LRU cache
func (l *LRUCache) Clear(ctx context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	l.cache = make(map[string]*LRUNode)
	l.head.Next = l.tail
	l.tail.Previous = l.head
	
	return nil
}

// Stats returns LRU cache statistics
func (l *LRUCache) Stats(ctx context.Context) (*CacheStats, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	expired := 0
	now := time.Now()
	
	for _, node := range l.cache {
		if now.After(node.Entry.Expires) {
			expired++
		}
	}
	
	return &CacheStats{
		Size:        len(l.cache),
		MaxSize:     l.maxSize,
		Expired:     expired,
		UsedPercent: float64(len(l.cache)) / float64(l.maxSize) * 100,
	}, nil
}

// Helper methods for LRU cache

func (l *LRUCache) generateKey(query string, options *EnhancementOptions) string {
	if options == nil {
		return query
	}
	
	key := query
	if options.EnableRewrite {
		key += "|rewrite"
	}
	if options.EnableExpansion {
		key += "|expansion"
	}
	if options.EnableDecomposition {
		key += "|decomposition"
	}
	if options.EnableIntentClassification {
		key += "|intent"
	}
	
	return key
}

func (l *LRUCache) addToFront(node *LRUNode) {
	node.Previous = l.head
	node.Next = l.head.Next
	l.head.Next.Previous = node
	l.head.Next = node
}

func (l *LRUCache) removeNode(node *LRUNode) {
	node.Previous.Next = node.Next
	node.Next.Previous = node.Previous
}

func (l *LRUCache) moveToFront(node *LRUNode) {
	l.removeNode(node)
	l.addToFront(node)
}

func (l *LRUCache) evictLast() {
	last := l.tail.Previous
	if last != l.head {
		l.removeNode(last)
		delete(l.cache, last.Key)
	}
}