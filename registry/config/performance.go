// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"istio.io/pkg/log"
	
	apiv1 "github.com/alibaba/higress/api/networking/v1"
)

const (
	// Default CircuitBreaker configuration
	DefaultMaxFailures = 5
	DefaultResetTimeout = 2 * time.Minute
	
	// Default RateLimiter configuration
	DefaultRateLimiterCapacity = 100.0
	DefaultRateLimiterRefillRate = 10.0
	
	// Cache optimization intervals
	DefaultOptimizationInterval = 5 * time.Minute
)

// CircuitBreaker implements circuit breaker pattern for provider operations
type CircuitBreaker struct {
	maxFailures   int
	resetTimeout  time.Duration
	currentState  CircuitState
	failures      int
	lastFailTime  time.Time
	mutex         sync.RWMutex
	onStateChange func(from, to CircuitState)
}

// CircuitState represents circuit breaker state
type CircuitState int

const (
	CircuitStateClosed CircuitState = iota
	CircuitStateOpen
	CircuitStateHalfOpen
)

// String returns string representation of circuit state
func (s CircuitState) String() string {
	switch s {
	case CircuitStateClosed:
		return "Closed"
	case CircuitStateOpen:
		return "Open"
	case CircuitStateHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		currentState: CircuitStateClosed,
	}
}

// Execute executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	state := cb.GetState()
	
	switch state {
	case CircuitStateOpen:
		return fmt.Errorf("circuit breaker is open")
	case CircuitStateHalfOpen:
		// Allow one request to test if service is healthy
		if err := operation(); err != nil {
			cb.recordFailure()
			return err
		}
		cb.recordSuccess()
		return nil
	default: // CircuitStateClosed
		if err := operation(); err != nil {
			cb.recordFailure()
			return err
		}
		cb.recordSuccess()
		return nil
	}
}

// GetState returns current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	if cb.currentState == CircuitStateOpen {
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.currentState = CircuitStateHalfOpen
			log.Debugf("Circuit breaker transitioning to HalfOpen state")
		}
	}
	
	return cb.currentState
}

// recordFailure records a failure and updates circuit breaker state
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.failures++
	cb.lastFailTime = time.Now()
	
	oldState := cb.currentState
	if cb.failures >= cb.maxFailures && cb.currentState == CircuitStateClosed {
		cb.currentState = CircuitStateOpen
		log.Warnf("Circuit breaker opened after %d failures", cb.failures)
		if cb.onStateChange != nil {
			cb.onStateChange(oldState, cb.currentState)
		}
	}
}

// recordSuccess records a success and resets failure count
func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	oldState := cb.currentState
	cb.failures = 0
	
	if cb.currentState == CircuitStateHalfOpen {
		cb.currentState = CircuitStateClosed
		log.Infof("Circuit breaker closed after successful operation")
		if cb.onStateChange != nil {
			cb.onStateChange(oldState, cb.currentState)
		}
	}
}

// SetStateChangeCallback sets callback for state changes
func (cb *CircuitBreaker) SetStateChangeCallback(callback func(from, to CircuitState)) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.onStateChange = callback
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens    float64
	capacity  float64
	refillRate float64
	lastRefill time.Time
	mutex     sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(capacity, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if an operation is allowed based on rate limiting
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	
	// Refill tokens based on elapsed time
	rl.tokens = min(rl.capacity, rl.tokens+rl.refillRate*elapsed)
	rl.lastRefill = now
	
	if rl.tokens >= 1.0 {
		rl.tokens--
		return true
	}
	
	return false
}

// min helper function
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// EnhancedConfigCache provides advanced caching with metrics and TTL optimization
type EnhancedConfigCache struct {
	*ConfigCache
	metrics     *CacheMetrics
	optimizer   *TTLOptimizer
}

// CacheMetrics tracks cache performance metrics
type CacheMetrics struct {
	hits        int64
	misses      int64
	evictions   int64
	totalOps    int64
	mutex       sync.RWMutex
}

// TTLOptimizer dynamically adjusts TTL based on access patterns
type TTLOptimizer struct {
	baseTTL       time.Duration
	maxTTL        time.Duration
	minTTL        time.Duration
	adjustmentRate float64
	mutex         sync.RWMutex
}

// NewEnhancedConfigCache creates an enhanced configuration cache
func NewEnhancedConfigCache(config CacheConfig) *EnhancedConfigCache {
	return &EnhancedConfigCache{
		ConfigCache: NewConfigCache(config),
		metrics:     &CacheMetrics{},
		optimizer: &TTLOptimizer{
			baseTTL:        config.TTL,
			maxTTL:         config.TTL * 4,
			minTTL:         config.TTL / 4,
			adjustmentRate: 0.1,
		},
	}
}

// Get retrieves a configuration from cache with metrics tracking
func (ec *EnhancedConfigCache) Get(key string) *apiv1.MCPConfig {
	config := ec.ConfigCache.Get(key)
	
	ec.metrics.mutex.Lock()
	ec.metrics.totalOps++
	if config != nil {
		ec.metrics.hits++
	} else {
		ec.metrics.misses++
	}
	ec.metrics.mutex.Unlock()
	
	return config
}

// GetMetrics returns current cache metrics
func (ec *EnhancedConfigCache) GetMetrics() CacheMetrics {
	ec.metrics.mutex.RLock()
	defer ec.metrics.mutex.RUnlock()
	// 返回值拷贝，避免mutex拷贝
	return CacheMetrics{
		hits:       ec.metrics.hits,
		misses:     ec.metrics.misses,
		evictions:  ec.metrics.evictions,
		totalOps:   ec.metrics.totalOps,
		// mutex不拷贝，使用零值
	}
}

// GetHitRatio returns cache hit ratio
func (ec *EnhancedConfigCache) GetHitRatio() float64 {
	ec.metrics.mutex.RLock()
	defer ec.metrics.mutex.RUnlock()
	
	if ec.metrics.totalOps == 0 {
		return 0.0
	}
	
	return float64(ec.metrics.hits) / float64(ec.metrics.totalOps)
}

// OptimizeTTL optimizes TTL based on access patterns
func (ec *EnhancedConfigCache) OptimizeTTL() {
	hitRatio := ec.GetHitRatio()
	
	ec.optimizer.mutex.Lock()
	defer ec.optimizer.mutex.Unlock()
	
	// Adjust TTL based on hit ratio
	if hitRatio > 0.8 {
		// High hit ratio, increase TTL
		newTTL := time.Duration(float64(ec.optimizer.baseTTL) * (1.0 + ec.optimizer.adjustmentRate))
		if newTTL <= ec.optimizer.maxTTL {
			ec.optimizer.baseTTL = newTTL
		}
	} else if hitRatio < 0.5 {
		// Low hit ratio, decrease TTL
		newTTL := time.Duration(float64(ec.optimizer.baseTTL) * (1.0 - ec.optimizer.adjustmentRate))
		if newTTL >= ec.optimizer.minTTL {
			ec.optimizer.baseTTL = newTTL
		}
	}
	
	log.Debugf("TTL optimized: hit ratio %.2f, new TTL %v", hitRatio, ec.optimizer.baseTTL)
}

// Enhanced provider wrapper with performance optimizations
type EnhancedProvider struct {
	ConfigProvider
	circuitBreaker *CircuitBreaker
	rateLimiter    *RateLimiter
	cache          *EnhancedConfigCache
}

// NewEnhancedProvider wraps a provider with performance enhancements
func NewEnhancedProvider(provider ConfigProvider, config *ProviderConfig) *EnhancedProvider {
	// Use configurable parameters instead of hardcoded values
	maxFailures := DefaultMaxFailures
	resetTimeout := DefaultResetTimeout
	capacity := DefaultRateLimiterCapacity
	refillRate := DefaultRateLimiterRefillRate
	
	// Allow override from config if specified
	if config.CircuitBreaker != nil {
		if config.CircuitBreaker.MaxFailures > 0 {
			maxFailures = config.CircuitBreaker.MaxFailures
		}
		if config.CircuitBreaker.ResetTimeout > 0 {
			resetTimeout = config.CircuitBreaker.ResetTimeout
		}
	}
	
	if config.RateLimiter != nil {
		if config.RateLimiter.Capacity > 0 {
			capacity = config.RateLimiter.Capacity
		}
		if config.RateLimiter.RefillRate > 0 {
			refillRate = config.RateLimiter.RefillRate
		}
	}
	
	circuitBreaker := NewCircuitBreaker(maxFailures, resetTimeout)
	rateLimiter := NewRateLimiter(capacity, refillRate)
	cache := NewEnhancedConfigCache(config.CacheConfig)
	
	return &EnhancedProvider{
		ConfigProvider: provider,
		circuitBreaker: circuitBreaker,
		rateLimiter:    rateLimiter,
		cache:          cache,
	}
}

// GetMCPConfig retrieves configuration with performance optimizations
func (ep *EnhancedProvider) GetMCPConfig(ctx context.Context, configRef string) (*apiv1.MCPConfig, error) {
	// Rate limiting check
	if !ep.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded")
	}
	
	// Try enhanced cache first
	if cached := ep.cache.Get(configRef); cached != nil {
		return cached, nil
	}
	
	// Circuit breaker protection
	var config *apiv1.MCPConfig
	var err error
	
	err = ep.circuitBreaker.Execute(ctx, func() error {
		config, err = ep.ConfigProvider.GetMCPConfig(ctx, configRef)
		return err
	})
	
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	ep.cache.Set(configRef, config)
	
	return config, nil
}

// GetMetrics returns performance metrics
func (ep *EnhancedProvider) GetMetrics() ProviderMetrics {
	cacheMetrics := ep.cache.GetMetrics()
	circuitState := ep.circuitBreaker.GetState()
	
	return ProviderMetrics{
		CacheHitRatio:    ep.cache.GetHitRatio(),
		CacheHits:        cacheMetrics.hits,
		CacheMisses:      cacheMetrics.misses,
		CircuitState:     circuitState.String(),
		TotalOperations:  cacheMetrics.totalOps,
	}
}

// ProviderMetrics contains provider performance metrics
type ProviderMetrics struct {
	CacheHitRatio   float64 `json:"cacheHitRatio"`
	CacheHits       int64   `json:"cacheHits"`
	CacheMisses     int64   `json:"cacheMisses"`
	CircuitState    string  `json:"circuitState"`
	TotalOperations int64   `json:"totalOperations"`
}

// StartOptimizationLoop starts a background loop for cache optimization
func (ep *EnhancedProvider) StartOptimizationLoop(ctx context.Context) {
	optimizationInterval := DefaultOptimizationInterval
	// Allow configuration override
	if ep.cache.config.OptimizationInterval > 0 {
		optimizationInterval = ep.cache.config.OptimizationInterval
	}
	
	ticker := time.NewTicker(optimizationInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ep.cache.OptimizeTTL()
		}
	}
}