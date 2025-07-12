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
	"sync"

	"k8s.io/client-go/kubernetes"
	"istio.io/pkg/log"
)

// ProviderFactoryRegistry manages registration of provider factories
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

// registerBuiltinProviders registers the built-in configuration providers
func (f *ExtendedProviderFactory) registerBuiltinProviders() {
	// ConfigMap provider factory
	configMapFactory := &ConfigMapProviderFactory{kubeClient: f.kubeClient}
	if err := f.registry.RegisterFactory(ConfigSourceConfigMap, configMapFactory); err != nil {
		log.Warnf("Failed to register ConfigMap provider factory: %v", err)
	}
	
	// Secret provider factory
	secretFactory := &SecretProviderFactory{kubeClient: f.kubeClient}
	if err := f.registry.RegisterFactory(ConfigSourceSecret, secretFactory); err != nil {
		log.Warnf("Failed to register Secret provider factory: %v", err)
	}
	
	// Add more providers as needed
	// etcdFactory := &EtcdProviderFactory{...}
	// consulFactory := &ConsulProviderFactory{...}
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
	
	// Register all available providers
	supportedSources := factory.SupportedSources()
	for _, source := range supportedSources {
		providerConfig := DefaultProviderConfig(namespace)
		providerConfig.Source = source
		
		provider, err := factory.CreateProvider(providerConfig)
		if err != nil {
			log.Warnf("Failed to create provider for source %s: %v", source, err)
			continue
		}
		
		if err := manager.RegisterProvider(source, provider); err != nil {
			log.Warnf("Failed to register provider for source %s: %v", source, err)
			continue
		}
	}
	
	return manager, nil
}

// RegisterCustomProvider registers a custom provider factory
func RegisterCustomProvider(source ConfigSource, factory ProviderFactory) error {
	return GetGlobalRegistry().RegisterFactory(source, factory)
}