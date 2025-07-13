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
	"fmt"
	"runtime"
	"sync"
	"time"

	"istio.io/pkg/log"
	"k8s.io/client-go/kubernetes"
)

// ProviderRegistrationConfig configures provider registration behavior
type ProviderRegistrationConfig struct {
	MaxRetries      int            // Maximum number of retry attempts
	RetryDelay      time.Duration  // Delay between retry attempts
	EnableSnapshot  bool           // Whether to capture system snapshot before panic
	CriticalSources []ConfigSource // Sources that are considered critical
}

// DefaultProviderRegistrationConfig returns default registration configuration
func DefaultProviderRegistrationConfig() *ProviderRegistrationConfig {
	return &ProviderRegistrationConfig{
		MaxRetries:     3,
		RetryDelay:     time.Second * 2,
		EnableSnapshot: true,
		CriticalSources: []ConfigSource{
			ConfigSourceConfigMap,
			ConfigSourceSecret,
		},
	}
}

// SystemSnapshot captures system state for debugging
type SystemSnapshot struct {
	Timestamp           time.Time                    `json:"timestamp"`
	GoVersion           string                       `json:"goVersion"`
	NumGoroutines       int                          `json:"numGoroutines"`
	MemStats            runtime.MemStats             `json:"memStats"`
	RegisteredFactories []ConfigSource               `json:"registeredFactories"`
	RegistrationAttempt int                          `json:"registrationAttempt"`
	LastError           string                       `json:"lastError"`
	KubeClientReady     bool                         `json:"kubeClientReady"`
	RegistryState       map[ConfigSource]interface{} `json:"registryState"`
}

// captureSystemSnapshot captures current system state
func (f *ExtendedProviderFactory) captureSystemSnapshot(attempt int, lastErr error, source ConfigSource) *SystemSnapshot {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get currently registered factories
	registeredFactories := f.registry.GetSupportedSources()

	// Check if kube client is accessible
	kubeClientReady := f.kubeClient != nil
	if kubeClientReady {
		// Try a simple operation to verify client health
		_, err := f.kubeClient.Discovery().ServerVersion()
		kubeClientReady = err == nil
	}

	// Build registry state information
	registryState := make(map[ConfigSource]interface{})
	for _, src := range registeredFactories {
		factory, err := f.registry.GetFactory(src)
		if err == nil && factory != nil {
			registryState[src] = map[string]interface{}{
				"type":             fmt.Sprintf("%T", factory),
				"supportedSources": factory.SupportedSources(),
			}
		} else {
			registryState[src] = map[string]interface{}{
				"error": err.Error(),
			}
		}
	}

	lastErrorMsg := ""
	if lastErr != nil {
		lastErrorMsg = lastErr.Error()
	}

	return &SystemSnapshot{
		Timestamp:           time.Now(),
		GoVersion:           runtime.Version(),
		NumGoroutines:       runtime.NumGoroutine(),
		MemStats:            memStats,
		RegisteredFactories: registeredFactories,
		RegistrationAttempt: attempt,
		LastError:           lastErrorMsg,
		KubeClientReady:     kubeClientReady,
		RegistryState:       registryState,
	}
}

// logSystemSnapshot logs system snapshot with detailed context
func (f *ExtendedProviderFactory) logSystemSnapshot(snapshot *SystemSnapshot, source ConfigSource) {
	log.Errorf("=== SYSTEM SNAPSHOT BEFORE CRITICAL FAILURE ===")
	log.Errorf("Timestamp: %v", snapshot.Timestamp.Format(time.RFC3339))
	log.Errorf("Failed Source: %s", source)
	log.Errorf("Registration Attempt: %d", snapshot.RegistrationAttempt)
	log.Errorf("Last Error: %s", snapshot.LastError)
	log.Errorf("Go Version: %s", snapshot.GoVersion)
	log.Errorf("Goroutines: %d", snapshot.NumGoroutines)
	log.Errorf("Memory Usage: Alloc=%d KB, Sys=%d KB",
		snapshot.MemStats.Alloc/1024, snapshot.MemStats.Sys/1024)
	log.Errorf("Kube Client Ready: %v", snapshot.KubeClientReady)
	log.Errorf("Currently Registered Factories: %v", snapshot.RegisteredFactories)

	// Log detailed registry state
	for src, state := range snapshot.RegistryState {
		log.Errorf("Factory[%s]: %+v", src, state)
	}
	log.Errorf("=== END SYSTEM SNAPSHOT ===")
}

// registerProviderWithRetry registers a provider with retry mechanism and enhanced error context
func (f *ExtendedProviderFactory) registerProviderWithRetry(
	source ConfigSource,
	factory ProviderFactory,
	config *ProviderRegistrationConfig,
) error {
	var lastErr error

	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		// Add delay for retry attempts
		if attempt > 1 {
			log.Warnf("Retrying registration for %s provider (attempt %d/%d) after %v",
				source, attempt, config.MaxRetries, config.RetryDelay)
			time.Sleep(config.RetryDelay)
		}

		// Attempt registration
		err := f.registry.RegisterFactory(source, factory)
		if err == nil {
			if attempt > 1 {
				log.Infof("Successfully registered %s provider after %d attempts", source, attempt)
			}
			return nil
		}

		lastErr = err

		// Enhanced error logging with context
		log.Warnf("Registration attempt %d/%d failed for %s provider: %v",
			attempt, config.MaxRetries, source, err)

		// Log additional context for debugging
		log.Warnf("Registration context - Source: %s, Factory Type: %T, Client Ready: %v",
			source, factory, f.kubeClient != nil)

		// For critical sources, capture more detailed state
		if f.isCriticalSource(source, config.CriticalSources) {
			supportedSources := factory.SupportedSources()
			log.Warnf("Critical source %s registration failed - Factory supports: %v", source, supportedSources)

			// Check if there's a conflict
			if existingFactory, getErr := f.registry.GetFactory(source); getErr == nil {
				log.Warnf("Conflict detected: Factory for %s already exists: %T", source, existingFactory)
			}
		}
	}

	// All retry attempts failed - prepare for critical failure
	if f.isCriticalSource(source, config.CriticalSources) {
		// Capture system snapshot before panic
		if config.EnableSnapshot {
			snapshot := f.captureSystemSnapshot(config.MaxRetries, lastErr, source)
			f.logSystemSnapshot(snapshot, source)
		}

		// Enhanced error message with all context
		criticalErr := fmt.Errorf(
			"CRITICAL: Failed to register essential %s provider after %d attempts. "+
				"Last error: %w. This will cause system initialization to fail. "+
				"Check kubeconfig, RBAC permissions, and network connectivity. "+
				"Registered factories: %v",
			source, config.MaxRetries, lastErr, f.registry.GetSupportedSources())

		log.Errorf("=== CRITICAL SYSTEM FAILURE ===")
		log.Errorf("%v", criticalErr)
		log.Errorf("System will now panic to prevent running in degraded state")
		log.Errorf("==================================")

		panic(criticalErr.Error())
	}

	// Non-critical source - return error for caller to handle
	return fmt.Errorf("failed to register %s provider after %d attempts: %w",
		source, config.MaxRetries, lastErr)
}

// isCriticalSource checks if a source is considered critical
func (f *ExtendedProviderFactory) isCriticalSource(source ConfigSource, criticalSources []ConfigSource) bool {
	for _, critical := range criticalSources {
		if source == critical {
			return true
		}
	}
	return false
}

// isCriticalSourceInList checks if a source is in the critical sources list
func isCriticalSourceInList(source ConfigSource, criticalSources []ConfigSource) bool {
	for _, critical := range criticalSources {
		if source == critical {
			return true
		}
	}
	return false
}

type ProviderFactoryRegistry struct {
	factories map[ConfigSource]ProviderFactory
	mutex     sync.RWMutex
}

var (
	// Global registry instance
	globalRegistry = &ProviderFactoryRegistry{
		factories: make(map[ConfigSource]ProviderFactory),
	}
)

// GetGlobalRegistry returns the global provider factory registry
func GetGlobalRegistry() *ProviderFactoryRegistry {
	return globalRegistry
}

// RegisterFactory registers a provider factory for a specific config source
func (r *ProviderFactoryRegistry) RegisterFactory(source ConfigSource, factory ProviderFactory) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.factories[source]; exists {
		return fmt.Errorf("factory for source %s already registered", source)
	}

	r.factories[source] = factory
	log.Infof("Registered provider factory for source: %s", source)
	return nil
}

// GetFactory returns a factory for the specified source
func (r *ProviderFactoryRegistry) GetFactory(source ConfigSource) (ProviderFactory, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	factory, exists := r.factories[source]
	if !exists {
		return nil, fmt.Errorf("no factory registered for source %s", source)
	}

	return factory, nil
}

// GetSupportedSources returns all supported configuration sources
func (r *ProviderFactoryRegistry) GetSupportedSources() []ConfigSource {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	sources := make([]ConfigSource, 0, len(r.factories))
	for source := range r.factories {
		sources = append(sources, source)
	}

	return sources
}

// CreateProvider creates a provider using the registered factory
func (r *ProviderFactoryRegistry) CreateProvider(source ConfigSource, config *ProviderConfig) (ConfigProvider, error) {
	factory, err := r.GetFactory(source)
	if err != nil {
		return nil, err
	}

	return factory.CreateProvider(config)
}

// ExtendedProviderFactory implements ProviderFactory with enhanced functionality
type ExtendedProviderFactory struct {
	kubeClient kubernetes.Interface
	registry   *ProviderFactoryRegistry
}

// NewExtendedProviderFactory creates a new extended provider factory
func NewExtendedProviderFactory(kubeClient kubernetes.Interface) *ExtendedProviderFactory {
	factory := &ExtendedProviderFactory{
		kubeClient: kubeClient,
		registry:   GetGlobalRegistry(),
	}

	// Register built-in providers
	factory.registerBuiltinProviders()

	return factory
}

// registerBuiltinProviders registers the built-in configuration providers with enhanced error handling
func (f *ExtendedProviderFactory) registerBuiltinProviders() {
	// Get registration configuration
	regConfig := DefaultProviderRegistrationConfig()

	log.Infof("Starting built-in provider registration with config: MaxRetries=%d, RetryDelay=%v, Snapshot=%v",
		regConfig.MaxRetries, regConfig.RetryDelay, regConfig.EnableSnapshot)

	// ConfigMap provider factory - critical for system operation
	configMapFactory := &ConfigMapProviderFactory{kubeClient: f.kubeClient}
	if err := f.registerProviderWithRetry(ConfigSourceConfigMap, configMapFactory, regConfig); err != nil {
		// This should not reach here for critical sources as registerProviderWithRetry will panic
		log.Errorf("Unexpected: ConfigMap provider registration returned error: %v", err)
	} else {
		log.Infof("Successfully registered ConfigMap provider factory")
	}

	// Secret provider factory - critical for secure configuration
	secretFactory := &SecretProviderFactory{kubeClient: f.kubeClient}
	if err := f.registerProviderWithRetry(ConfigSourceSecret, secretFactory, regConfig); err != nil {
		// This should not reach here for critical sources as registerProviderWithRetry will panic
		log.Errorf("Unexpected: Secret provider registration returned error: %v", err)
	} else {
		log.Infof("Successfully registered Secret provider factory")
	}

	log.Infof("Built-in provider registration completed. Registered sources: %v",
		f.registry.GetSupportedSources())

	// Future providers can be added here with appropriate criticality settings
	// Example for non-critical providers:
	// etcdFactory := &EtcdProviderFactory{...}
	// nonCriticalConfig := DefaultProviderRegistrationConfig()
	// nonCriticalConfig.CriticalSources = []ConfigSource{} // Make it non-critical
	// if err := f.registerProviderWithRetry(ConfigSourceEtcd, etcdFactory, nonCriticalConfig); err != nil {
	//     log.Warnf("Failed to register etcd provider (non-critical): %v", err)
	// }
}

// CreateProvider creates a provider based on the configuration
func (f *ExtendedProviderFactory) CreateProvider(config *ProviderConfig) (ConfigProvider, error) {
	return f.registry.CreateProvider(config.Source, config)
}

// SupportedSources returns the list of supported configuration sources
func (f *ExtendedProviderFactory) SupportedSources() []ConfigSource {
	return f.registry.GetSupportedSources()
}

// ConfigMapProviderFactory creates ConfigMap providers
type ConfigMapProviderFactory struct {
	kubeClient kubernetes.Interface
}

func (f *ConfigMapProviderFactory) CreateProvider(config *ProviderConfig) (ConfigProvider, error) {
	return NewConfigMapProvider(f.kubeClient, config), nil
}

func (f *ConfigMapProviderFactory) SupportedSources() []ConfigSource {
	return []ConfigSource{ConfigSourceConfigMap}
}

// SecretProviderFactory creates Secret providers
type SecretProviderFactory struct {
	kubeClient kubernetes.Interface
}

func (f *SecretProviderFactory) CreateProvider(config *ProviderConfig) (ConfigProvider, error) {
	return NewSecretProvider(f.kubeClient, config), nil
}

func (f *SecretProviderFactory) SupportedSources() []ConfigSource {
	return []ConfigSource{ConfigSourceSecret}
}

// SetupExtendedConfigManager sets up a configuration manager with extended functionality
func SetupExtendedConfigManager(kubeClient kubernetes.Interface, namespace string) (*Manager, error) {
	factory := NewExtendedProviderFactory(kubeClient)
	manager := NewManager(factory)

	// Get default configuration to identify critical sources
	regConfig := DefaultProviderRegistrationConfig()
	criticalSources := regConfig.CriticalSources

	// Register all available providers
	supportedSources := factory.SupportedSources()
	var errors []string
	registeredCount := 0

	for _, source := range supportedSources {
		providerConfig := DefaultProviderConfig(namespace, source)

		provider, err := factory.CreateProvider(providerConfig)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to create provider for source %s: %v", source, err)

			if isCriticalSourceInList(source, criticalSources) {
				// Critical provider failure - terminate immediately
				log.Errorf("CRITICAL: %s", errorMsg)
				return nil, fmt.Errorf("critical provider %s creation failed: %v", source, err)
			}

			// Non-critical provider - log and continue
			log.Warnf("Non-critical provider failure: %s", errorMsg)
			errors = append(errors, errorMsg)
			continue
		}

		if err := manager.RegisterProvider(source, provider); err != nil {
			errorMsg := fmt.Sprintf("failed to register provider for source %s: %v", source, err)

			if isCriticalSourceInList(source, criticalSources) {
				// Critical provider registration failure - terminate immediately
				log.Errorf("CRITICAL: %s", errorMsg)
				return nil, fmt.Errorf("critical provider %s registration failed: %v", source, err)
			}

			// Non-critical provider - log and continue
			log.Warnf("Non-critical provider registration failure: %s", errorMsg)
			errors = append(errors, errorMsg)
			continue
		}

		registeredCount++
		log.Infof("Successfully registered provider for source: %s", source)
	}

	// Ensure at least one provider is registered
	if registeredCount == 0 {
		return nil, fmt.Errorf("failed to register any configuration providers: %v", errors)
	}

	// Verify all critical providers are registered
	for _, critical := range criticalSources {
		if !isCriticalSourceInList(critical, supportedSources) {
			return nil, fmt.Errorf("critical provider %s is not available in supported sources: %v", critical, supportedSources)
		}
	}

	if len(errors) > 0 {
		log.Warnf("Configuration manager setup completed with %d/%d providers registered. Non-critical errors: %v",
			registeredCount, len(supportedSources), errors)
	} else {
		log.Infof("Configuration manager setup completed successfully with %d/%d providers registered",
			registeredCount, len(supportedSources))
	}

	return manager, nil
}

// RegisterCustomProvider registers a custom provider factory
func RegisterCustomProvider(source ConfigSource, factory ProviderFactory) error {
	return GetGlobalRegistry().RegisterFactory(source, factory)
}
