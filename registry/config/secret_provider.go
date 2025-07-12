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
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"istio.io/pkg/log"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
)

// SecretProvider implements ConfigProvider for Kubernetes Secrets
type SecretProvider struct {
	client kubernetes.Interface
	config *ProviderConfig
	cache  *ConfigCache
}

// NewSecretProvider creates a new Secret provider
func NewSecretProvider(client kubernetes.Interface, config *ProviderConfig) *SecretProvider {
	if config == nil {
		config = DefaultSecretProviderConfig("default")
	}
	
	return &SecretProvider{
		client: client,
		config: config,
		cache:  NewConfigCache(config.CacheConfig),
	}
}

// Name returns the provider name
func (p *SecretProvider) Name() string {
	return string(ConfigSourceSecret)
}

// GetMCPConfig retrieves MCP configuration from Secret with retry and cache
func (p *SecretProvider) GetMCPConfig(ctx context.Context, configRef string) (*apiv1.MCPConfig, error) {
	if configRef == "" {
		return nil, fmt.Errorf("config reference cannot be empty")
	}
	
	// Try cache first
	if p.config.CacheConfig.Enabled {
		if cached := p.cache.Get(configRef); cached != nil {
			log.Debugf("Secret provider: cache hit for %s", configRef)
			return cached, nil
		}
	}
	
	// Fetch from Kubernetes
	config, err := p.getMCPConfigFromSecret(ctx, configRef)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	if p.config.CacheConfig.Enabled {
		p.cache.Set(configRef, config)
	}
	
	return config, nil
}

// Watch starts watching for Secret changes (placeholder for future implementation)
func (p *SecretProvider) Watch(ctx context.Context, handler ConfigUpdateHandler) error {
	log.Info("Secret provider: watching not implemented yet")
	return nil
}

// Stop stops the provider and cleans up resources
func (p *SecretProvider) Stop() error {
	if p.cache != nil {
		p.cache.Clear()
	}
	log.Info("Secret provider: stopped")
	return nil
}

// getMCPConfigFromSecret fetches configuration from Kubernetes Secret
func (p *SecretProvider) getMCPConfigFromSecret(ctx context.Context, configRef string) (*apiv1.MCPConfig, error) {
	secret, err := p.client.CoreV1().Secrets(p.config.Namespace).Get(ctx, configRef, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("secret %s not found in namespace %s", configRef, p.config.Namespace)
		}
		return nil, fmt.Errorf("failed to get Secret %s in namespace %s: %w", configRef, p.config.Namespace, err)
	}
	
	config, err := p.parseMCPConfigFromSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Secret %s: %w", configRef, err)
	}
	
	if err := p.validateMCPConfig(config); err != nil {
		return nil, fmt.Errorf("invalid MCP config in Secret %s: %w", configRef, err)
	}
	
	return config, nil
}

// parseMCPConfigFromSecret parses MCP configuration from Secret
func (p *SecretProvider) parseMCPConfigFromSecret(secret *corev1.Secret) (*apiv1.MCPConfig, error) {
	// Support both new structured format and legacy format
	if configData, ok := secret.Data["config"]; ok {
		// New structured format
		var config apiv1.MCPConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, fmt.Errorf("failed to parse structured MCP config: %w", err)
		}
		return &config, nil
	}
	
	if instancesData, ok := secret.Data["instances"]; ok {
		// Legacy format - instances only
		var instances []*apiv1.MCPInstance
		if err := json.Unmarshal(instancesData, &instances); err != nil {
			return nil, fmt.Errorf("failed to parse legacy MCP instances: %w", err)
		}
		
		return &apiv1.MCPConfig{
			Instances:       instances,
			LoadBalanceMode: apiv1.LoadBalanceModeRoundRobin,
		}, nil
	}
	
	return nil, fmt.Errorf("Secret missing both 'config' and 'instances' keys")
}

// validateMCPConfig validates MCP configuration
func (p *SecretProvider) validateMCPConfig(config *apiv1.MCPConfig) error {
	if len(config.Instances) == 0 {
		return fmt.Errorf("at least one instance is required")
	}
	
	for i, instance := range config.Instances {
		if err := p.validateMCPInstance(instance, i); err != nil {
			return fmt.Errorf("instance %d validation failed: %w", i, err)
		}
	}
	
	// Validate load balance mode
	if config.LoadBalanceMode != "" {
		switch config.LoadBalanceMode {
		case apiv1.LoadBalanceModeRoundRobin, apiv1.LoadBalanceModeWeighted, apiv1.LoadBalanceModeRandom:
			// Valid modes
		default:
			return fmt.Errorf("invalid load balance mode: %s", config.LoadBalanceMode)
		}
	}
	
	return nil
}

// validateMCPInstance validates a single MCP instance
func (p *SecretProvider) validateMCPInstance(instance *apiv1.MCPInstance, index int) error {
	if strings.TrimSpace(instance.Domain) == "" {
		return fmt.Errorf("domain is required")
	}
	
	if instance.Port <= 0 || instance.Port > MaxPort {
		return fmt.Errorf("port must be between %d and %d, got %d", MinPort, MaxPort, instance.Port)
	}
	
	if instance.Weight < MinWeight || instance.Weight > MaxWeight {
		return fmt.Errorf("weight must be between %d and %d, got %d", MinWeight, MaxWeight, instance.Weight)
	}
	
	if instance.Priority < MinPriority {
		return fmt.Errorf("priority must be non-negative (>= %d), got %d", MinPriority, instance.Priority)
	}
	
	return nil
}