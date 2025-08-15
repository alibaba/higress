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
	"strings"
	"time"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
)

// ConfigProvider defines the interface for configuration providers
type ConfigProvider interface {
	// GetMCPConfig retrieves MCP configuration by reference
	GetMCPConfig(ctx context.Context, configRef string) (*apiv1.MCPConfig, error)
	
	// Watch starts watching for configuration changes
	Watch(ctx context.Context, handler ConfigUpdateHandler) error
	
	// Stop stops the provider and cleans up resources
	Stop() error
	
	// Name returns the provider name
	Name() string
}

// ConfigUpdateHandler handles configuration update events
type ConfigUpdateHandler func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error

// ConfigEventType represents the type of configuration event
type ConfigEventType string

const (
	ConfigEventTypeAdded    ConfigEventType = "Added"
	ConfigEventTypeModified ConfigEventType = "Modified"
	ConfigEventTypeDeleted  ConfigEventType = "Deleted"
)

// ConfigSource represents a configuration source type
type ConfigSource string

const (
	ConfigSourceConfigMap ConfigSource = "configmap"
	ConfigSourceSecret    ConfigSource = "secret"
	ConfigSourceEtcd      ConfigSource = "etcd"
	ConfigSourceConsul    ConfigSource = "consul"
)

// ProviderConfig holds common configuration for providers
// It provides comprehensive configuration options for various aspects of provider behavior
// including caching, retry mechanisms, circuit breaking, and rate limiting.
type ProviderConfig struct {
	// Source specifies the configuration source type (configmap, secret, etcd, consul)
	Source ConfigSource `json:"source" yaml:"source"`
	
	// Namespace specifies the Kubernetes namespace for the configuration source
	Namespace string `json:"namespace" yaml:"namespace"`
	
	// RetryConfig configures retry behavior for failed operations
	// Default: 3 retries with exponential backoff starting at 1s
	RetryConfig RetryConfig `json:"retryConfig" yaml:"retryConfig"`
	
	// CacheConfig configures caching behavior to improve performance
	// Default: enabled with 5-minute TTL and LRU eviction
	CacheConfig CacheConfig `json:"cacheConfig" yaml:"cacheConfig"`
	
	// WatchConfig configures real-time monitoring of configuration changes
	// Default: enabled with 5-minute resync period
	WatchConfig WatchConfig `json:"watchConfig" yaml:"watchConfig"`
	
	// CircuitBreaker configures circuit breaker pattern for fault tolerance
	// Automatically opens circuit when failure rate exceeds threshold
	// Default: 5 failures trigger circuit open, 2-minute reset timeout
	CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty" yaml:"circuitBreaker,omitempty"`
	
	// RateLimiter configures rate limiting using token bucket algorithm
	// Prevents overwhelming the configuration source with too many requests
	// Default: 100 requests capacity, 10 requests/second refill rate
	RateLimiter *RateLimiterConfig `json:"rateLimiter,omitempty" yaml:"rateLimiter,omitempty"`
	
	// ExtraConfig allows provider-specific configuration options
	// Key-value pairs for custom provider implementations
	ExtraConfig map[string]string `json:"extraConfig,omitempty" yaml:"extraConfig,omitempty"`
}

// RetryConfig configures retry behavior for failed operations
// Uses exponential backoff strategy to gradually increase delays between retries
type RetryConfig struct {
	// MaxRetries specifies the maximum number of retry attempts
	// Valid range: 0-10, Default: 3
	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`
	
	// BaseDelay specifies the initial delay before the first retry
	// Valid range: 100ms-10s, Default: 1s
	BaseDelay time.Duration `json:"baseDelay" yaml:"baseDelay"`
	
	// MaxDelay specifies the maximum delay between retries
	// Valid range: 1s-5m, Default: 30s
	MaxDelay time.Duration `json:"maxDelay" yaml:"maxDelay"`
	
	// BackoffFactor specifies the multiplier for exponential backoff
	// Valid range: 1.0-5.0, Default: 2.0
	BackoffFactor float64 `json:"backoffFactor" yaml:"backoffFactor"`
}

// CircuitBreakerConfig configures circuit breaker behavior for fault tolerance
// Implements the circuit breaker pattern to prevent cascading failures
type CircuitBreakerConfig struct {
	// MaxFailures specifies the number of consecutive failures that trigger circuit opening
	// Valid range: 1-100, Default: 5
	MaxFailures int `json:"maxFailures" yaml:"maxFailures"`
	
	// ResetTimeout specifies how long to wait before attempting to close an open circuit
	// Valid range: 10s-30m, Default: 2m
	ResetTimeout time.Duration `json:"resetTimeout" yaml:"resetTimeout"`
}

// RateLimiterConfig configures rate limiting behavior using token bucket algorithm
// Prevents overwhelming the configuration source with excessive requests
type RateLimiterConfig struct {
	// Capacity specifies the maximum number of tokens in the bucket
	// Valid range: 1-10000, Default: 100
	Capacity float64 `json:"capacity" yaml:"capacity"`
	
	// RefillRate specifies how many tokens are added per second
	// Valid range: 0.1-1000, Default: 10
	RefillRate float64 `json:"refillRate" yaml:"refillRate"`
}

// CacheConfig configures caching behavior to improve performance
// Uses TTL and LRU strategies for efficient memory management
type CacheConfig struct {
	// Enabled determines whether caching is active
	// Default: true
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// TTL specifies how long cache entries remain valid
	// Valid range: 1m-1h, Default: 5m
	TTL time.Duration `json:"ttl" yaml:"ttl"`
	
	// MaxSize specifies the maximum number of entries in the cache
	// Valid range: 10-10000, Default: 1000 (0 means unlimited)
	MaxSize int `json:"maxSize" yaml:"maxSize"`
	
	// EnableLRU determines whether to use LRU eviction policy
	// Default: true
	EnableLRU bool `json:"enableLRU" yaml:"enableLRU"`
	
	// OptimizationInterval specifies how often to optimize cache performance
	// Valid range: 1m-1h, Default: 5m
	OptimizationInterval time.Duration `json:"optimizationInterval" yaml:"optimizationInterval"`
}

// WatchConfig configures watching behavior for real-time updates
// Monitors configuration sources for changes and triggers updates
type WatchConfig struct {
	// Enabled determines whether watching is active
	// Default: true
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// ResyncPeriod specifies how often to perform full resynchronization
	// Valid range: 1m-1h, Default: 5m
	ResyncPeriod time.Duration `json:"resyncPeriod" yaml:"resyncPeriod"`
	
	// RetryInterval specifies how long to wait before retrying failed watch operations
	// Valid range: 1s-5m, Default: 5s
	RetryInterval time.Duration `json:"retryInterval" yaml:"retryInterval"`
	
	// BufferSize specifies the size of the event buffer
	// Valid range: 10-1000, Default: 100
	BufferSize int `json:"bufferSize" yaml:"bufferSize"`
}

// ValidateProviderConfig validates provider configuration parameters
// Returns an error if any configuration values are outside valid ranges
func ValidateProviderConfig(config *ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("provider config cannot be nil")
	}
	
	var errs []error
	
	// Validate RetryConfig
	if err := validateRetryConfig(config.RetryConfig); err != nil {
		errs = append(errs, fmt.Errorf("retry config: %w", err))
	}
	
	// Validate CacheConfig
	if err := validateCacheConfig(config.CacheConfig); err != nil {
		errs = append(errs, fmt.Errorf("cache config: %w", err))
	}
	
	// Validate WatchConfig
	if err := validateWatchConfig(config.WatchConfig); err != nil {
		errs = append(errs, fmt.Errorf("watch config: %w", err))
	}
	
	// Validate CircuitBreakerConfig
	if config.CircuitBreaker != nil {
		if err := validateCircuitBreakerConfig(*config.CircuitBreaker); err != nil {
			errs = append(errs, fmt.Errorf("circuit breaker config: %w", err))
		}
	}
	
	// Validate RateLimiterConfig
	if config.RateLimiter != nil {
		if err := validateRateLimiterConfig(*config.RateLimiter); err != nil {
			errs = append(errs, fmt.Errorf("rate limiter config: %w", err))
		}
	}
	
	if len(errs) > 0 {
		var errMsgs []string
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errMsgs, "; "))
	}
	
	return nil
}

// validateRetryConfig validates retry configuration parameters
func validateRetryConfig(config RetryConfig) error {
	if config.MaxRetries < 0 || config.MaxRetries > 10 {
		return fmt.Errorf("maxRetries must be between 0 and 10, got %d", config.MaxRetries)
	}
	
	if config.BaseDelay < 100*time.Millisecond || config.BaseDelay > 10*time.Second {
		return fmt.Errorf("baseDelay must be between 100ms and 10s, got %v", config.BaseDelay)
	}
	
	if config.MaxDelay < time.Second || config.MaxDelay > 5*time.Minute {
		return fmt.Errorf("maxDelay must be between 1s and 5m, got %v", config.MaxDelay)
	}
	
	if config.BackoffFactor < 1.0 || config.BackoffFactor > 5.0 {
		return fmt.Errorf("backoffFactor must be between 1.0 and 5.0, got %f", config.BackoffFactor)
	}
	
	if config.BaseDelay > config.MaxDelay {
		return fmt.Errorf("baseDelay (%v) cannot be greater than maxDelay (%v)", config.BaseDelay, config.MaxDelay)
	}
	
	return nil
}

// validateCircuitBreakerConfig validates circuit breaker configuration parameters
func validateCircuitBreakerConfig(config CircuitBreakerConfig) error {
	if config.MaxFailures < 1 || config.MaxFailures > 100 {
		return fmt.Errorf("maxFailures must be between 1 and 100, got %d", config.MaxFailures)
	}
	
	if config.ResetTimeout < 10*time.Second || config.ResetTimeout > 30*time.Minute {
		return fmt.Errorf("resetTimeout must be between 10s and 30m, got %v", config.ResetTimeout)
	}
	
	return nil
}

// validateRateLimiterConfig validates rate limiter configuration parameters
func validateRateLimiterConfig(config RateLimiterConfig) error {
	if config.Capacity < 1 || config.Capacity > 10000 {
		return fmt.Errorf("capacity must be between 1 and 10000, got %f", config.Capacity)
	}
	
	if config.RefillRate < 0.1 || config.RefillRate > 1000 {
		return fmt.Errorf("refillRate must be between 0.1 and 1000, got %f", config.RefillRate)
	}
	
	return nil
}

// validateCacheConfig validates cache configuration parameters
func validateCacheConfig(config CacheConfig) error {
	if config.TTL < time.Minute || config.TTL > time.Hour {
		return fmt.Errorf("ttl must be between 1m and 1h, got %v", config.TTL)
	}
	
	if config.MaxSize < 0 || config.MaxSize > 10000 {
		return fmt.Errorf("maxSize must be between 0 and 10000, got %d", config.MaxSize)
	}
	
	if config.OptimizationInterval < time.Minute || config.OptimizationInterval > time.Hour {
		return fmt.Errorf("optimizationInterval must be between 1m and 1h, got %v", config.OptimizationInterval)
	}
	
	return nil
}

// validateWatchConfig validates watch configuration parameters
func validateWatchConfig(config WatchConfig) error {
	if config.ResyncPeriod < time.Minute || config.ResyncPeriod > time.Hour {
		return fmt.Errorf("resyncPeriod must be between 1m and 1h, got %v", config.ResyncPeriod)
	}
	
	if config.RetryInterval < time.Second || config.RetryInterval > 5*time.Minute {
		return fmt.Errorf("retryInterval must be between 1s and 5m, got %v", config.RetryInterval)
	}
	
	if config.BufferSize < 10 || config.BufferSize > 1000 {
		return fmt.Errorf("bufferSize must be between 10 and 1000, got %d", config.BufferSize)
	}
	
	return nil
}

// DefaultProviderConfig returns default provider configuration for the specified source.
// This is the base configuration that can be customized for different config sources.
//
// Usage examples:
//   - DefaultProviderConfig(namespace, ConfigSourceConfigMap) for ConfigMap
//   - DefaultProviderConfig(namespace, ConfigSourceEtcd) for etcd
//   - DefaultProviderConfig(namespace, ConfigSourceSecret) for Secret
//   - DefaultProviderConfig(namespace, ConfigSourceConsul) for Consul
//
// For source-specific optimizations, consider using the dedicated functions:
//   - DefaultConfigMapProviderConfig(namespace)
//   - DefaultEtcdProviderConfig(namespace) 
//   - DefaultSecretProviderConfig(namespace)
//   - DefaultConsulProviderConfig(namespace)
func DefaultProviderConfig(namespace string, source ConfigSource) *ProviderConfig {
	return &ProviderConfig{
		Source:    source,
		Namespace: namespace,
		RetryConfig: RetryConfig{
			MaxRetries:    3,
			BaseDelay:     time.Second,
			MaxDelay:      time.Minute,
			BackoffFactor: 2.0,
		},
		CacheConfig: CacheConfig{
			Enabled:              true,
			TTL:                  time.Minute * 5,
			MaxSize:              1000,
			EnableLRU:            true,
			OptimizationInterval: time.Minute * 5,
		},
		WatchConfig: WatchConfig{
			Enabled:       true,
			ResyncPeriod:  time.Minute * 5,
			RetryInterval: time.Second * 5,
			BufferSize:    100,
		},
		CircuitBreaker: &CircuitBreakerConfig{
			MaxFailures:  5,
			ResetTimeout: time.Minute * 2,
		},
		RateLimiter: &RateLimiterConfig{
			Capacity:   100.0,
			RefillRate: 10.0,
		},
	}
}

// DefaultEtcdProviderConfig returns default configuration for etcd provider
func DefaultEtcdProviderConfig(namespace string) *ProviderConfig {
	config := DefaultProviderConfig(namespace, ConfigSourceEtcd)
	// Etcd specific optimizations
	config.CacheConfig.TTL = time.Minute * 10 // Longer TTL for etcd
	config.CircuitBreaker.MaxFailures = 3     // More sensitive to failures
	return config
}

// DefaultConfigMapProviderConfig returns default configuration for ConfigMap provider
func DefaultConfigMapProviderConfig(namespace string) *ProviderConfig {
	return DefaultProviderConfig(namespace, ConfigSourceConfigMap)
}

// DefaultSecretProviderConfig returns default configuration for Secret provider
func DefaultSecretProviderConfig(namespace string) *ProviderConfig {
	return DefaultProviderConfig(namespace, ConfigSourceSecret)
}

// DefaultConsulProviderConfig returns default configuration for Consul provider
func DefaultConsulProviderConfig(namespace string) *ProviderConfig {
	config := DefaultProviderConfig(namespace, ConfigSourceConsul)
	// Consul specific optimizations
	config.CacheConfig.TTL = time.Minute * 3 // Shorter TTL for dynamic discovery
	config.WatchConfig.ResyncPeriod = time.Minute * 2
	return config
}

// ProviderFactory creates configuration providers
type ProviderFactory interface {
	CreateProvider(config *ProviderConfig) (ConfigProvider, error)
	SupportedSources() []ConfigSource
}