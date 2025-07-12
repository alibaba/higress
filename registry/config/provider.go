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
type ProviderConfig struct {
	Source       ConfigSource      `json:"source"`
	Namespace    string            `json:"namespace"`
	RetryConfig  RetryConfig       `json:"retryConfig"`
	CacheConfig  CacheConfig       `json:"cacheConfig"`
	WatchConfig  WatchConfig       `json:"watchConfig"`
	ExtraConfig  map[string]string `json:"extraConfig,omitempty"`
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries    int           `json:"maxRetries"`
	BaseDelay     time.Duration `json:"baseDelay"`
	MaxDelay      time.Duration `json:"maxDelay"`
	BackoffFactor float64       `json:"backoffFactor"`
}

// CacheConfig configures caching behavior
type CacheConfig struct {
	Enabled    bool          `json:"enabled"`
	TTL        time.Duration `json:"ttl"`
	MaxSize    int           `json:"maxSize"`
	EnableLRU  bool          `json:"enableLRU"`
}

// WatchConfig configures watching behavior
type WatchConfig struct {
	Enabled       bool          `json:"enabled"`
	ResyncPeriod  time.Duration `json:"resyncPeriod"`
	RetryInterval time.Duration `json:"retryInterval"`
	BufferSize    int           `json:"bufferSize"`
}

// DefaultProviderConfig returns default provider configuration
func DefaultProviderConfig(namespace string) *ProviderConfig {
	return &ProviderConfig{
		Source:    ConfigSourceConfigMap,
		Namespace: namespace,
		RetryConfig: RetryConfig{
			MaxRetries:    3,
			BaseDelay:     time.Second,
			MaxDelay:      time.Minute,
			BackoffFactor: 2.0,
		},
		CacheConfig: CacheConfig{
			Enabled:   true,
			TTL:       time.Minute * 5,
			MaxSize:   1000,
			EnableLRU: true,
		},
		WatchConfig: WatchConfig{
			Enabled:       true,
			ResyncPeriod:  time.Minute * 5,
			RetryInterval: time.Second * 5,
			BufferSize:    100,
		},
	}
}

// ProviderFactory creates configuration providers
type ProviderFactory interface {
	CreateProvider(config *ProviderConfig) (ConfigProvider, error)
	SupportedSources() []ConfigSource
}