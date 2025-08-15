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
	"k8s.io/client-go/kubernetes"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/hashicorp/go-multierror"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
)

// Prometheus metrics
var (
	configSourceWatchErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "higress_config_source_watch_errors_total",
			Help: "Total number of config source watch errors",
		},
		[]string{"source"},
	)
	
	configSourceWatchSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "higress_config_source_watch_success_total", 
			Help: "Total number of successful config source watch startups",
		},
		[]string{"source"},
	)
	
	configManagerStartDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name: "higress_config_manager_start_duration_seconds",
			Help: "Duration of config manager start operations",
		},
	)
	
	configManagerStartSuccessRate = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "higress_config_manager_start_success_rate",
			Help: "Success rate of config manager start operations",
		},
	)
	
	configCacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "higress_config_cache_hits_total",
			Help: "Total number of config cache hits",
		},
		[]string{"source", "type"},
	)
	
	circuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "higress_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"source"},
	)
)

// CircuitBreakerState represents the state for manager circuit breaker
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

// ManagerCircuitState represents a circuit state for the manager
type ManagerCircuitState struct {
	State       CircuitBreakerState
	FailureCount int
	LastFailure  time.Time
	LastSuccess  time.Time
}

// ManagerCircuitBreaker is a simple circuit breaker implementation for manager
type ManagerCircuitBreaker struct {
	states        map[string]*ManagerCircuitState
	mutex         sync.RWMutex
	failureThreshold int
	recoveryTimeout  time.Duration
}

// StaleCache provides fallback cached configurations
type StaleCache struct {
	items map[string]*StaleCacheItem
	mutex sync.RWMutex
	maxAge time.Duration
}

type StaleCacheItem struct {
	Config    *apiv1.MCPConfig
	Source    ConfigSource
	ConfigRef string
	CachedAt  time.Time
}

// NewManagerCircuitBreaker creates a new circuit breaker for manager
func NewManagerCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration) *ManagerCircuitBreaker {
	return &ManagerCircuitBreaker{
		states:          make(map[string]*ManagerCircuitState),
		failureThreshold: failureThreshold,
		recoveryTimeout:  recoveryTimeout,
	}
}

// IsOpen checks if circuit breaker is open for a source
func (cb *ManagerCircuitBreaker) IsOpen(source string) bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	state, exists := cb.states[source]
	if !exists {
		return false
	}
	
	switch state.State {
	case CircuitOpen:
		// Check if recovery timeout has passed
		if time.Since(state.LastFailure) > cb.recoveryTimeout {
			state.State = CircuitHalfOpen
			circuitBreakerState.WithLabelValues(source).Set(2)
			return false
		}
		return true
	case CircuitHalfOpen:
		return false
	default:
		return false
	}
}

// RecordFailure records a failure for the circuit breaker
func (cb *ManagerCircuitBreaker) RecordFailure(source string) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	state, exists := cb.states[source]
	if !exists {
		state = &ManagerCircuitState{}
		cb.states[source] = state
	}
	
	state.FailureCount++
	state.LastFailure = time.Now()
	
	if state.FailureCount >= cb.failureThreshold {
		state.State = CircuitOpen
		circuitBreakerState.WithLabelValues(source).Set(1)
		log.Warnf("Circuit breaker opened for source %s after %d failures", source, state.FailureCount)
	}
}

// RecordSuccess records a success for the circuit breaker
func (cb *ManagerCircuitBreaker) RecordSuccess(source string) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	state, exists := cb.states[source]
	if !exists {
		state = &ManagerCircuitState{}
		cb.states[source] = state
	}
	
	state.FailureCount = 0
	state.LastSuccess = time.Now()
	state.State = CircuitClosed
	circuitBreakerState.WithLabelValues(source).Set(0)
}

// NewStaleCache creates a new stale cache
func NewStaleCache(maxAge time.Duration) *StaleCache {
	return &StaleCache{
		items:  make(map[string]*StaleCacheItem),
		maxAge: maxAge,
	}
}

// GetStale retrieves a stale cached configuration
func (sc *StaleCache) GetStale(source ConfigSource, configRef string) *apiv1.MCPConfig {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	
	key := fmt.Sprintf("%s:%s", source, configRef)
	item, exists := sc.items[key]
	if !exists {
		return nil
	}
	
	// Check if cache is too old
	if time.Since(item.CachedAt) > sc.maxAge {
		configCacheHits.WithLabelValues(string(source), "expired").Inc()
		return nil
	}
	
	configCacheHits.WithLabelValues(string(source), "stale_hit").Inc()
	return item.Config
}

// SetStale stores a configuration in stale cache
func (sc *StaleCache) SetStale(source ConfigSource, configRef string, config *apiv1.MCPConfig) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	key := fmt.Sprintf("%s:%s", source, configRef)
	sc.items[key] = &StaleCacheItem{
		Config:    config,
		Source:    source,
		ConfigRef: configRef,
		CachedAt:  time.Now(),
	}
}

// Manager manages multiple configuration providers
type Manager struct {
	providers      map[ConfigSource]ConfigProvider
	factory        ProviderFactory
	circuitBreaker *ManagerCircuitBreaker
	staleCache     *StaleCache
	mu             sync.RWMutex
}

// NewManager creates a new configuration manager
func NewManager(factory ProviderFactory) *Manager {
	return &Manager{
		providers:      make(map[ConfigSource]ConfigProvider),
		factory:        factory,
		circuitBreaker: NewManagerCircuitBreaker(3, 30*time.Second), // 3 failures, 30s recovery
		staleCache:     NewStaleCache(5 * time.Minute),       // 5 minute stale cache
	}
}

// RegisterProvider registers a configuration provider
func (m *Manager) RegisterProvider(source ConfigSource, provider ConfigProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.providers[source]; exists {
		return fmt.Errorf("provider for source %s already registered", source)
	}
	
	m.providers[source] = provider
	log.Infof("Configuration manager: registered provider for source %s", source)
	return nil
}

// GetProvider returns a provider for the specified source
func (m *Manager) GetProvider(source ConfigSource) (ConfigProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	provider, exists := m.providers[source]
	if !exists {
		return nil, fmt.Errorf("no provider registered for source %s", source)
	}
	
	return provider, nil
}

// GetMCPConfig retrieves MCP configuration using the appropriate provider
func (m *Manager) GetMCPConfig(ctx context.Context, source ConfigSource, configRef string) (*apiv1.MCPConfig, error) {
	provider, err := m.GetProvider(source)
	if err != nil {
		return nil, err
	}
	
	config, err := provider.GetMCPConfig(ctx, configRef)
	if err == nil {
		// Cache successful results
		m.staleCache.SetStale(source, configRef, config)
		configCacheHits.WithLabelValues(string(source), "fresh").Inc()
		return config, nil
	}
	
	// Record failure in circuit breaker
	m.circuitBreaker.RecordFailure(string(source))
	
	return config, err
}

// GetMCPConfigWithFallback retrieves MCP configuration with stale cache fallback
func (m *Manager) GetMCPConfigWithFallback(ctx context.Context, source ConfigSource, configRef string) (*apiv1.MCPConfig, error) {
	config, err := m.GetMCPConfig(ctx, source, configRef)
	if err != nil {
		// Try stale cache fallback
		if cached := m.staleCache.GetStale(source, configRef); cached != nil {
			log.Warnf("Using stale cached config for %s:%s due to error: %v", source, configRef, err)
			return cached, nil
		}
	}
	return config, err
}

// StartWatching starts watching for configuration changes
func (m *Manager) StartWatching(ctx context.Context, handler ConfigUpdateHandler) error {
	startTime := time.Now()
	defer func() {
		configManagerStartDuration.Observe(time.Since(startTime).Seconds())
	}()
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var result *multierror.Error
	successCount := 0
	
	for source, provider := range m.providers {
		sourceStr := string(source)
		
		// Check circuit breaker state
		if m.circuitBreaker.IsOpen(sourceStr) {
			log.Warnf("Circuit breaker open for source %s, skipping watch startup", source)
			continue
		}
		
		if err := provider.Watch(ctx, handler); err != nil {
			// Record failure in circuit breaker and metrics
			m.circuitBreaker.RecordFailure(sourceStr)
			configSourceWatchErrors.WithLabelValues(sourceStr).Inc()
			
			wrappedErr := fmt.Errorf("failed to start watching for source %s: %w", source, err)
			log.Warnf("Configuration manager: %v", wrappedErr)
			result = multierror.Append(result, wrappedErr)
		} else {
			// Record success in circuit breaker and metrics
			m.circuitBreaker.RecordSuccess(sourceStr)
			configSourceWatchSuccess.WithLabelValues(sourceStr).Inc()
			successCount++
		}
	}
	
	// Record overall success rate
	totalSources := len(m.providers)
	if totalSources > 0 {
		successRate := float64(successCount) / float64(totalSources)
		configManagerStartSuccessRate.Set(successRate)
	}
	
	return result.ErrorOrNil()
}

// StartWatchingWithRetry starts watching with retry mechanism  
func (m *Manager) StartWatchingWithRetry(ctx context.Context, handler ConfigUpdateHandler, maxRetries int) error {
	var lastErrors []string
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			log.Infof("Retrying StartWatching after %v (attempt %d/%d)", delay, attempt, maxRetries)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		
		if err := m.StartWatching(ctx, handler); err == nil {
			if attempt > 0 {
				log.Infof("StartWatching succeeded after %d attempts", attempt+1)
			}
			return nil
		} else {
			lastErrors = m.extractErrorsFromMessage(err.Error())
			log.Warnf("StartWatching attempt %d failed: %v", attempt+1, err)
		}
	}
	
	return fmt.Errorf("StartWatching failed after %d attempts, last errors: %v", maxRetries+1, lastErrors)
}

// extractErrorsFromMessage extracts error details from formatted error message
func (m *Manager) extractErrorsFromMessage(errorMsg string) []string {
	// Simple extraction - in production you might want more sophisticated parsing
	return []string{errorMsg}
}

// Stop stops all providers
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var result *multierror.Error
	for source, provider := range m.providers {
		if err := provider.Stop(); err != nil {
			wrappedErr := fmt.Errorf("failed to stop provider for source %s: %w", source, err)
			log.Warnf("Configuration manager: %v", wrappedErr)
			result = multierror.Append(result, wrappedErr)
		}
	}
	
	return result.ErrorOrNil()
}

// DefaultProviderFactory implements ProviderFactory
type DefaultProviderFactory struct {
	kubeClient kubernetes.Interface
}

// NewDefaultProviderFactory creates a new default provider factory
func NewDefaultProviderFactory(kubeClient kubernetes.Interface) *DefaultProviderFactory {
	return &DefaultProviderFactory{
		kubeClient: kubeClient,
	}
}

// CreateProvider creates a provider based on the configuration
func (f *DefaultProviderFactory) CreateProvider(config *ProviderConfig) (ConfigProvider, error) {
	switch config.Source {
	case ConfigSourceConfigMap:
		return NewConfigMapProvider(f.kubeClient, config), nil
	case ConfigSourceSecret:
		return nil, fmt.Errorf("secret provider is not yet implemented; please use ConfigMap provider instead (supported sources: %v)", f.SupportedSources())
	case ConfigSourceEtcd:
		return nil, fmt.Errorf("etcd provider is not yet implemented; please use ConfigMap provider instead (supported sources: %v)", f.SupportedSources())
	case ConfigSourceConsul:
		return nil, fmt.Errorf("consul provider is not yet implemented; please use ConfigMap provider instead (supported sources: %v)", f.SupportedSources())
	default:
		return nil, fmt.Errorf("unsupported configuration source: %s (supported sources: %v)", config.Source, f.SupportedSources())
	}
}

// SupportedSources returns the list of supported configuration sources
func (f *DefaultProviderFactory) SupportedSources() []ConfigSource {
	return []ConfigSource{
		ConfigSourceConfigMap,
		// Add more as they are implemented
	}
}

// Helper functions for easy setup

// SetupConfigManager sets up a configuration manager with default providers
func SetupConfigManager(kubeClient kubernetes.Interface, namespace string) (*Manager, error) {
	factory := NewDefaultProviderFactory(kubeClient)
	manager := NewManager(factory)
	
	// Register ConfigMap provider
	configMapConfig := DefaultConfigMapProviderConfig(namespace)
	configMapProvider, err := factory.CreateProvider(configMapConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ConfigMap provider: %w", err)
	}
	
	if err := manager.RegisterProvider(ConfigSourceConfigMap, configMapProvider); err != nil {
		return nil, fmt.Errorf("failed to register ConfigMap provider: %w", err)
	}
	
	return manager, nil
}