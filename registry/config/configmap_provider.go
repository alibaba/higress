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

const (
	// Port validation constants - TCP/UDP port range (1-65535)
	MinPort = 1
	MaxPort = 65535
	// Weight validation constants - load balancing weight range (0-100)
	MinWeight = 0
	MaxWeight = 100
	// Priority validation constants - service priority (>= 0, lower values indicate higher priority)
	MinPriority = 0
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
// Uses indexMap for O(1) key lookup optimization in LRU operations
type ConfigCache struct {
	items     map[string]*CacheItem
	lruList   []string            // LRU order list (least -> most recently used)
	indexMap  map[string]int      // Key -> index mapping for O(1) position lookup
	mutex     sync.RWMutex
	config    CacheConfig
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
		config = DefaultConfigMapProviderConfig("default")
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
			LoadBalanceMode: apiv1.LoadBalanceMode_ROUND_ROBIN,
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
	switch config.LoadBalanceMode {
	case apiv1.LoadBalanceMode_ROUND_ROBIN, apiv1.LoadBalanceMode_WEIGHTED, apiv1.LoadBalanceMode_RANDOM:
		// Valid modes
	default:
		return fmt.Errorf("invalid load balance mode: %v", config.LoadBalanceMode)
	}
	
	return nil
}

// validateMCPInstance validates a single MCP instance
func (p *ConfigMapProvider) validateMCPInstance(instance *apiv1.MCPInstance, index int) error {
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

// ConfigCache implementation

// NewConfigCache creates a new configuration cache
func NewConfigCache(config CacheConfig) *ConfigCache {
	return &ConfigCache{
		items:    make(map[string]*CacheItem),
		lruList:  make([]string, 0),
		indexMap: make(map[string]int),
		config:   config,
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
	c.indexMap = make(map[string]int)
}

// updateLRU updates the LRU list efficiently using index map
// Time complexity: O(1) for lookup + O(k) for slice operations where k is elements after removed position
// Space complexity: O(n) for indexMap storage
func (c *ConfigCache) updateLRU(key string) {
	// Check if key exists in index map - O(1) operation
	if idx, exists := c.indexMap[key]; exists {
		// Remove from current position - O(k) where k = len(slice) - idx
		c.lruList = append(c.lruList[:idx], c.lruList[idx+1:]...)
		// Update indices for shifted elements - O(k) operation
		for i := idx; i < len(c.lruList); i++ {
			c.indexMap[c.lruList[i]] = i
		}
	}
	// Add to end (most recently used) - O(1) operation
	c.lruList = append(c.lruList, key)
	c.indexMap[key] = len(c.lruList) - 1
}

// removeLRU removes a key from LRU list efficiently using index map
// Time complexity: O(1) for lookup + O(k) for slice operations where k is elements after removed position
func (c *ConfigCache) removeLRU(key string) {
	if idx, exists := c.indexMap[key]; exists {
		// Remove from list - O(k) where k = len(slice) - idx
		c.lruList = append(c.lruList[:idx], c.lruList[idx+1:]...)
		// Update indices for shifted elements - O(k) operation
		for i := idx; i < len(c.lruList); i++ {
			c.indexMap[c.lruList[i]] = i
		}
		// Remove from index map - O(1) operation
		delete(c.indexMap, key)
	}
}

// evictIfNeeded evicts least recently used items if cache is full
// Time complexity: O(n) in worst case when evicting multiple items
// Only triggered when cache exceeds MaxSize limit to avoid frequent operations
func (c *ConfigCache) evictIfNeeded() {
	if c.config.MaxSize <= 0 {
		return
	}
	
	for len(c.items) > c.config.MaxSize && len(c.lruList) > 0 {
		oldest := c.lruList[0]
		delete(c.items, oldest)                    // O(1) operation
		delete(c.indexMap, oldest)                 // O(1) operation
		c.lruList = c.lruList[1:]                  // O(1) operation (slice header modification)
		// Update indices for shifted elements - O(n) operation
		for i := 0; i < len(c.lruList); i++ {
			c.indexMap[c.lruList[i]] = i
		}
	}
}

// validateConsistency performs internal consistency checks for debugging
// This method is intended for development and testing purposes
func (c *ConfigCache) validateConsistency() error {
	// Check if indexMap size matches lruList length
	if len(c.indexMap) != len(c.lruList) {
		return fmt.Errorf("indexMap size (%d) doesn't match lruList length (%d)", 
			len(c.indexMap), len(c.lruList))
	}
	
	// Check if all lruList entries have correct indices in indexMap
	for i, key := range c.lruList {
		if idx, exists := c.indexMap[key]; !exists {
			return fmt.Errorf("key %s at position %d not found in indexMap", key, i)
		} else if idx != i {
			return fmt.Errorf("key %s has incorrect index in indexMap: expected %d, got %d", 
				key, i, idx)
		}
	}
	
	// Check if all indexMap entries point to valid lruList positions
	for key, idx := range c.indexMap {
		if idx < 0 || idx >= len(c.lruList) {
			return fmt.Errorf("indexMap key %s has invalid index %d (lruList length: %d)", 
				key, idx, len(c.lruList))
		}
		if c.lruList[idx] != key {
			return fmt.Errorf("indexMap key %s at index %d doesn't match lruList entry %s", 
				key, idx, c.lruList[idx])
		}
	}
	
	return nil
}