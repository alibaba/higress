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
	"math"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"istio.io/pkg/log"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
)

// ConfigMapProvider implements ConfigProvider for Kubernetes ConfigMaps
type ConfigMapProvider struct {
	client    kubernetes.Interface
	config    *ProviderConfig
	cache     *ConfigCache
	watcher   watch.Interface
	stopChan  chan struct{}
	mu        sync.RWMutex
	started   bool
}

// ConfigCache provides caching functionality with TTL and LRU support
type ConfigCache struct {
	items   map[string]*CacheItem
	lruList []string
	mutex   sync.RWMutex
	config  CacheConfig
}

// CacheItem represents a cached configuration item
type CacheItem struct {
	Config    *apiv1.MCPConfig
	ExpiresAt time.Time
	AccessAt  time.Time
}

// NewConfigMapProvider creates a new ConfigMap provider
func NewConfigMapProvider(client kubernetes.Interface, config *ProviderConfig) *ConfigMapProvider {
	if config == nil {
		config = DefaultProviderConfig("default")
	}
	
	return &ConfigMapProvider{
		client:   client,
		config:   config,
		cache:    NewConfigCache(config.CacheConfig),
		stopChan: make(chan struct{}),
	}
}

// Name returns the provider name
func (p *ConfigMapProvider) Name() string {
	return string(ConfigSourceConfigMap)
}

// GetMCPConfig retrieves MCP configuration from ConfigMap with retry and cache
func (p *ConfigMapProvider) GetMCPConfig(ctx context.Context, configRef string) (*apiv1.MCPConfig, error) {
	if configRef == "" {
		return nil, fmt.Errorf("config reference cannot be empty")
	}
	
	// Try cache first
	if p.config.CacheConfig.Enabled {
		if cached := p.cache.Get(configRef); cached != nil {
			log.Debugf("ConfigMap provider: cache hit for %s", configRef)
			return cached, nil
		}
	}
	
	// Fetch from Kubernetes with retry
	config, err := p.getMCPConfigWithRetry(ctx, configRef)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	if p.config.CacheConfig.Enabled {
		p.cache.Set(configRef, config)
	}
	
	return config, nil
}

// Watch starts watching for ConfigMap changes
func (p *ConfigMapProvider) Watch(ctx context.Context, handler ConfigUpdateHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.started {
		return fmt.Errorf("provider already started")
	}
	
	if !p.config.WatchConfig.Enabled {
		log.Info("ConfigMap provider: watching disabled")
		return nil
	}
	
	// Start watcher in background
	go p.startWatcher(ctx, handler)
	p.started = true
	
	log.Infof("ConfigMap provider: started watching in namespace %s", p.config.Namespace)
	return nil
}

// Stop stops the provider and cleans up resources
func (p *ConfigMapProvider) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.started {
		return nil
	}
	
	close(p.stopChan)
	
	if p.watcher != nil {
		p.watcher.Stop()
		p.watcher = nil
	}
	
	if p.cache != nil {
		p.cache.Clear()
	}
	
	p.started = false
	log.Info("ConfigMap provider: stopped")
	return nil
}

// startWatcher starts the ConfigMap watcher
func (p *ConfigMapProvider) startWatcher(ctx context.Context, handler ConfigUpdateHandler) {
	ticker := time.NewTicker(p.config.WatchConfig.RetryInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			if err := p.doWatch(ctx, handler); err != nil {
				log.Warnf("ConfigMap provider: watch error: %v, retrying...", err)
			}
		}
	}
}

// doWatch performs the actual watching
func (p *ConfigMapProvider) doWatch(ctx context.Context, handler ConfigUpdateHandler) error {
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app.higress.io/mcp-config": "true",
		},
	}
	
	listOptions := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&labelSelector),
		FieldSelector: fields.Everything().String(),
	}
	
	watcher, err := p.client.CoreV1().ConfigMaps(p.config.Namespace).Watch(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to start ConfigMap watcher: %w", err)
	}
	
	p.mu.Lock()
	p.watcher = watcher
	p.mu.Unlock()
	
	defer func() {
		p.mu.Lock()
		if p.watcher == watcher {
			p.watcher = nil
		}
		p.mu.Unlock()
		watcher.Stop()
	}()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-p.stopChan:
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watcher channel closed")
			}
			
			if err := p.handleWatchEvent(event, handler); err != nil {
				log.Warnf("ConfigMap provider: error handling watch event: %v", err)
			}
		}
	}
}

// handleWatchEvent handles a single watch event
func (p *ConfigMapProvider) handleWatchEvent(event watch.Event, handler ConfigUpdateHandler) error {
	configMap, ok := event.Object.(*corev1.ConfigMap)
	if !ok {
		return fmt.Errorf("unexpected object type: %T", event.Object)
	}
	
	var eventType ConfigEventType
	switch event.Type {
	case watch.Added:
		eventType = ConfigEventTypeAdded
	case watch.Modified:
		eventType = ConfigEventTypeModified
	case watch.Deleted:
		eventType = ConfigEventTypeDeleted
	default:
		return nil // Ignore other event types
	}
	
	configRef := configMap.Name
	
	// Handle deletion
	if eventType == ConfigEventTypeDeleted {
		p.cache.Delete(configRef)
		return handler(configRef, nil, eventType)
	}
	
	// Parse and validate configuration
	config, err := p.parseMCPConfig(configMap)
	if err != nil {
		log.Warnf("ConfigMap provider: failed to parse ConfigMap %s: %v", configRef, err)
		return nil // Don't propagate parse errors
	}
	
	if err := p.validateMCPConfig(config); err != nil {
		log.Warnf("ConfigMap provider: invalid config in ConfigMap %s: %v", configRef, err)
		return nil // Don't propagate validation errors
	}
	
	// Update cache
	if p.config.CacheConfig.Enabled {
		p.cache.Set(configRef, config)
	}
	
	return handler(configRef, config, eventType)
}

// getMCPConfigWithRetry retrieves configuration with exponential backoff retry
func (p *ConfigMapProvider) getMCPConfigWithRetry(ctx context.Context, configRef string) (*apiv1.MCPConfig, error) {
	retryConfig := p.config.RetryConfig
	var lastErr error
	
	for attempt := 0; attempt < retryConfig.MaxRetries; attempt++ {
		config, err := p.getMCPConfigFromK8s(ctx, configRef)
		if err == nil {
			return config, nil
		}
		
		lastErr = err
		
		// Don't retry on certain errors
		if errors.IsNotFound(err) || errors.IsForbidden(err) {
			break
		}
		
		if attempt < retryConfig.MaxRetries-1 {
			delay := p.calculateRetryDelay(attempt, retryConfig)
			log.Warnf("ConfigMap provider: attempt %d/%d failed for %s: %v, retrying in %v", 
				attempt+1, retryConfig.MaxRetries, configRef, err, delay)
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
	}
	
	return nil, fmt.Errorf("failed to get ConfigMap %s after %d attempts: %w", 
		configRef, retryConfig.MaxRetries, lastErr)
}

// calculateRetryDelay calculates exponential backoff delay
func (p *ConfigMapProvider) calculateRetryDelay(attempt int, config RetryConfig) time.Duration {
	delay := time.Duration(float64(config.BaseDelay) * math.Pow(config.BackoffFactor, float64(attempt)))
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}
	return delay
}

// getMCPConfigFromK8s fetches configuration from Kubernetes
func (p *ConfigMapProvider) getMCPConfigFromK8s(ctx context.Context, configRef string) (*apiv1.MCPConfig, error) {
	configMap, err := p.client.CoreV1().ConfigMaps(p.config.Namespace).Get(ctx, configRef, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap %s in namespace %s: %w", 
			configRef, p.config.Namespace, err)
	}
	
	config, err := p.parseMCPConfig(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ConfigMap %s: %w", configRef, err)
	}
	
	if err := p.validateMCPConfig(config); err != nil {
		return nil, fmt.Errorf("invalid MCP config in ConfigMap %s: %w", configRef, err)
	}
	
	return config, nil
}

// parseMCPConfig parses MCP configuration from ConfigMap
func (p *ConfigMapProvider) parseMCPConfig(configMap *corev1.ConfigMap) (*apiv1.MCPConfig, error) {
	// Support both new structured format and legacy format
	if configData, ok := configMap.Data["config"]; ok {
		// New structured format
		var config apiv1.MCPConfig
		if err := json.Unmarshal([]byte(configData), &config); err != nil {
			return nil, fmt.Errorf("failed to parse structured MCP config: %w", err)
		}
		return &config, nil
	}
	
	if instancesData, ok := configMap.Data["instances"]; ok {
		// Legacy format - instances only
		var instances []*apiv1.MCPInstance
		if err := json.Unmarshal([]byte(instancesData), &instances); err != nil {
			return nil, fmt.Errorf("failed to parse legacy MCP instances: %w", err)
		}
		
		return &apiv1.MCPConfig{
			Instances:       instances,
			LoadBalanceMode: apiv1.LoadBalanceModeRoundRobin,
		}, nil
	}
	
	return nil, fmt.Errorf("ConfigMap missing both 'config' and 'instances' keys")
}

// validateMCPConfig validates MCP configuration
func (p *ConfigMapProvider) validateMCPConfig(config *apiv1.MCPConfig) error {
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
func (p *ConfigMapProvider) validateMCPInstance(instance *apiv1.MCPInstance, index int) error {
	if strings.TrimSpace(instance.Domain) == "" {
		return fmt.Errorf("domain is required")
	}
	
	if instance.Port <= 0 || instance.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", instance.Port)
	}
	
	if instance.Weight < 0 || instance.Weight > 100 {
		return fmt.Errorf("weight must be between 0 and 100, got %d", instance.Weight)
	}
	
	if instance.Priority < 0 {
		return fmt.Errorf("priority must be non-negative, got %d", instance.Priority)
	}
	
	return nil
}

// ConfigCache implementation

// NewConfigCache creates a new configuration cache
func NewConfigCache(config CacheConfig) *ConfigCache {
	return &ConfigCache{
		items:   make(map[string]*CacheItem),
		lruList: make([]string, 0),
		config:  config,
	}
}

// Get retrieves a configuration from cache
func (c *ConfigCache) Get(key string) *apiv1.MCPConfig {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	item, exists := c.items[key]
	if !exists {
		return nil
	}
	
	// Check expiration
	if time.Now().After(item.ExpiresAt) {
		delete(c.items, key)
		c.removeLRU(key)
		return nil
	}
	
	// Update access time for LRU
	if c.config.EnableLRU {
		item.AccessAt = time.Now()
		c.updateLRU(key)
	}
	
	return item.Config
}

// Set stores a configuration in cache
func (c *ConfigCache) Set(key string, config *apiv1.MCPConfig) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now()
	c.items[key] = &CacheItem{
		Config:    config,
		ExpiresAt: now.Add(c.config.TTL),
		AccessAt:  now,
	}
	
	if c.config.EnableLRU {
		c.updateLRU(key)
		c.evictIfNeeded()
	}
}

// Delete removes a configuration from cache
func (c *ConfigCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	delete(c.items, key)
	c.removeLRU(key)
}

// Clear removes all configurations from cache
func (c *ConfigCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.items = make(map[string]*CacheItem)
	c.lruList = c.lruList[:0]
}

// updateLRU updates the LRU list
func (c *ConfigCache) updateLRU(key string) {
	c.removeLRU(key)
	c.lruList = append(c.lruList, key)
}

// removeLRU removes a key from LRU list
func (c *ConfigCache) removeLRU(key string) {
	for i, k := range c.lruList {
		if k == key {
			c.lruList = append(c.lruList[:i], c.lruList[i+1:]...)
			break
		}
	}
}

// evictIfNeeded evicts least recently used items if cache is full
func (c *ConfigCache) evictIfNeeded() {
	if c.config.MaxSize <= 0 {
		return
	}
	
	for len(c.items) > c.config.MaxSize && len(c.lruList) > 0 {
		oldest := c.lruList[0]
		delete(c.items, oldest)
		c.lruList = c.lruList[1:]
	}
}