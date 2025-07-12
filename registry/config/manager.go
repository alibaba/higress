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

	"istio.io/pkg/log"
	"k8s.io/client-go/kubernetes"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
)

// Manager manages multiple configuration providers
type Manager struct {
	providers map[ConfigSource]ConfigProvider
	factory   ProviderFactory
	mu        sync.RWMutex
}

// NewManager creates a new configuration manager
func NewManager(factory ProviderFactory) *Manager {
	return &Manager{
		providers: make(map[ConfigSource]ConfigProvider),
		factory:   factory,
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
	
	return provider.GetMCPConfig(ctx, configRef)
}

// StartWatching starts watching for configuration changes
func (m *Manager) StartWatching(ctx context.Context, handler ConfigUpdateHandler) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for source, provider := range m.providers {
		if err := provider.Watch(ctx, handler); err != nil {
			log.Warnf("Configuration manager: failed to start watching for source %s: %v", source, err)
			// Continue with other providers
		}
	}
	
	return nil
}

// Stop stops all providers
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var lastErr error
	for source, provider := range m.providers {
		if err := provider.Stop(); err != nil {
			log.Warnf("Configuration manager: failed to stop provider for source %s: %v", source, err)
			lastErr = err
		}
	}
	
	return lastErr
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
		return nil, fmt.Errorf("secret provider not implemented yet")
	case ConfigSourceEtcd:
		return nil, fmt.Errorf("etcd provider not implemented yet")
	case ConfigSourceConsul:
		return nil, fmt.Errorf("consul provider not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported configuration source: %s", config.Source)
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
	configMapConfig := DefaultProviderConfig(namespace)
	configMapProvider, err := factory.CreateProvider(configMapConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ConfigMap provider: %w", err)
	}
	
	if err := manager.RegisterProvider(ConfigSourceConfigMap, configMapProvider); err != nil {
		return nil, fmt.Errorf("failed to register ConfigMap provider: %w", err)
	}
	
	return manager, nil
}