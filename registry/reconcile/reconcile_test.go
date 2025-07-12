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

package reconcile

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	"github.com/alibaba/higress/pkg/kube"
	"github.com/alibaba/higress/registry/config"
)

func TestGetMCPConfig(t *testing.T) {
	// Create test ConfigMap with new structured format
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcp-config",
			Namespace: "default",
			Labels: map[string]string{
				"app.higress.io/mcp-config": "true",
			},
		},
		Data: map[string]string{
			"config": `{
				"instances": [
					{
						"domain": "nacos-1.example.com",
						"port": 8848,
						"weight": 100,
						"priority": 1,
						"healthPath": "/nacos/health"
					},
					{
						"domain": "nacos-2.example.com", 
						"port": 8848,
						"weight": 50,
						"priority": 2,
						"healthPath": "/nacos/health"
					}
				],
				"loadBalanceMode": "weighted",
				"healthCheck": {
					"enabled": true,
					"interval": "30s",
					"timeout": "5s",
					"unhealthyThreshold": 3
				}
			}`,
		},
	}

	// Create mock kube client  
	kubeClient := kube.NewFakeClient()
	
	// Add the ConfigMap to the fake client
	_, err := kubeClient.Kube().CoreV1().ConfigMaps("default").Create(
		context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test ConfigMap in fake client: %v", err)
	}

	// Create configuration manager
	configManager, err := config.SetupExtendedConfigManager(kubeClient.Kube(), "default")
	if err != nil {
		t.Fatalf("Failed to setup config manager: %v", err)
	}

	// Test case 1: Valid ConfigMap reference
	t.Run("ValidConfigMapReference", func(t *testing.T) {
		ctx := context.Background()
		config, err := configManager.GetMCPConfig(ctx, config.ConfigSourceConfigMap, "test-mcp-config")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(config.Instances) != 2 {
			t.Errorf("Expected 2 instances, got: %d", len(config.Instances))
		}

		if config.Instances[0].Domain != "nacos-1.example.com" {
			t.Errorf("Expected domain 'nacos-1.example.com', got: %s", config.Instances[0].Domain)
		}

		if config.Instances[0].Port != 8848 {
			t.Errorf("Expected port 8848, got: %d", config.Instances[0].Port)
		}

		if config.Instances[0].Weight != 100 {
			t.Errorf("Expected weight 100, got: %d", config.Instances[0].Weight)
		}

		if config.LoadBalanceMode != "weighted" {
			t.Errorf("Expected load balance mode 'weighted', got: %s", config.LoadBalanceMode)
		}
	})

	// Test case 2: Non-existent ConfigMap reference
	t.Run("NonExistentConfigMapReference", func(t *testing.T) {
		ctx := context.Background()
		config, err := configManager.GetMCPConfig(ctx, config.ConfigSourceConfigMap, "non-existent-config")
		if err == nil {
			t.Error("Expected error for non-existent ConfigMap, got nil")
		}

		if config != nil {
			t.Errorf("Expected nil config, got: %v", config)
		}
	})

	// Test case 3: ConfigMap with invalid JSON
	t.Run("ConfigMapWithInvalidJSON", func(t *testing.T) {
		// Create ConfigMap with invalid JSON
		invalidConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalid-json-config",
				Namespace: "default",
				Labels: map[string]string{
					"app.higress.io/mcp-config": "true",
				},
			},
			Data: map[string]string{
				"config": `{"invalid": json}`, // Invalid JSON
			},
		}

		_, err := kubeClient.Kube().CoreV1().ConfigMaps("default").Create(
			context.Background(), invalidConfigMap, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create invalid ConfigMap: %v", err)
		}

		ctx := context.Background()
		config, err := configManager.GetMCPConfig(ctx, config.ConfigSourceConfigMap, "invalid-json-config")
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}

		if config != nil {
			t.Errorf("Expected nil config, got: %v", config)
		}
	})

	// Test case 4: Legacy format compatibility
	t.Run("LegacyFormatCompatibility", func(t *testing.T) {
		// Create ConfigMap with legacy format
		legacyConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "legacy-format-config",
				Namespace: "default",
				Labels: map[string]string{
					"app.higress.io/mcp-config": "true",
				},
			},
			Data: map[string]string{
				"instances": `[
					{
						"domain": "nacos-legacy.example.com",
						"port": 8848,
						"weight": 100
					}
				]`,
			},
		}

		_, err := kubeClient.Kube().CoreV1().ConfigMaps("default").Create(
			context.Background(), legacyConfigMap, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create legacy ConfigMap: %v", err)
		}

		ctx := context.Background()
		config, err := configManager.GetMCPConfig(ctx, config.ConfigSourceConfigMap, "legacy-format-config")
		if err != nil {
			t.Errorf("Expected no error for legacy format, got: %v", err)
		}

		if len(config.Instances) != 1 {
			t.Errorf("Expected 1 instance, got: %d", len(config.Instances))
		}

		if config.LoadBalanceMode != apiv1.LoadBalanceModeRoundRobin {
			t.Errorf("Expected default load balance mode, got: %s", config.LoadBalanceMode)
		}
	})
}

// TestLoadBalancer tests load balancing functionality
func TestLoadBalancer(t *testing.T) {
	// Create test MCP config
	instances := []*apiv1.MCPInstance{
		{
			Domain:   "nacos-1.example.com",
			Port:     8848,
			Weight:   100,
			Priority: 1,
		},
		{
			Domain:   "nacos-2.example.com",
			Port:     8848,
			Weight:   50,
			Priority: 1,
		},
		{
			Domain:   "nacos-3.example.com",
			Port:     8848,
			Weight:   25,
			Priority: 2,
		},
	}

	config := &apiv1.MCPConfig{
		Instances: instances,
	}

	// Test round-robin load balancing
	t.Run("RoundRobinLoadBalancing", func(t *testing.T) {
		config.LoadBalanceMode = apiv1.LoadBalanceModeRoundRobin
		lb := &LoadBalancer{config: config}

		// Test multiple selections
		selections := make(map[string]int)
		for i := 0; i < 100; i++ {
			instance := lb.selectInstance("test-registry")
			if instance != nil {
				selections[instance.Domain]++
			}
		}

		// Should distribute evenly among all instances
		if len(selections) != 3 {
			t.Errorf("Expected 3 unique instances selected, got: %d", len(selections))
		}

		// Each instance should be selected at least once
		for domain, count := range selections {
			if count == 0 {
				t.Errorf("Instance %s was never selected", domain)
			}
		}
	})

	// Test weighted load balancing
	t.Run("WeightedLoadBalancing", func(t *testing.T) {
		config.LoadBalanceMode = apiv1.LoadBalanceModeWeighted
		lb := &LoadBalancer{config: config}

		// Test multiple selections
		selections := make(map[string]int)
		for i := 0; i < 1000; i++ {
			instance := lb.selectInstance("test-registry")
			if instance != nil {
				selections[instance.Domain]++
			}
		}

		// Higher weight instances should be selected more frequently
		count1 := selections["nacos-1.example.com"]
		count2 := selections["nacos-2.example.com"]
		count3 := selections["nacos-3.example.com"]

		// nacos-1 (weight 100) should be selected more than nacos-2 (weight 50)
		if count1 <= count2 {
			t.Errorf("Expected nacos-1 (%d) to be selected more than nacos-2 (%d)", count1, count2)
		}

		// nacos-2 (weight 50) should be selected more than nacos-3 (weight 25)
		if count2 <= count3 {
			t.Errorf("Expected nacos-2 (%d) to be selected more than nacos-3 (%d)", count2, count3)
		}
	})

	// Test random load balancing
	t.Run("RandomLoadBalancing", func(t *testing.T) {
		config.LoadBalanceMode = apiv1.LoadBalanceModeRandom
		lb := &LoadBalancer{config: config}

		// Test multiple selections
		selections := make(map[string]int)
		for i := 0; i < 100; i++ {
			instance := lb.selectInstance("test-registry")
			if instance != nil {
				selections[instance.Domain]++
			}
		}

		// Should distribute among all instances (random distribution)
		if len(selections) != 3 {
			t.Errorf("Expected 3 unique instances selected, got: %d", len(selections))
		}
	})

	// Test priority-based selection
	t.Run("PriorityBasedSelection", func(t *testing.T) {
		config.LoadBalanceMode = apiv1.LoadBalanceModeRoundRobin
		lb := &LoadBalancer{config: config}

		instances := lb.getHealthyInstances()

		// Instances should be sorted by priority (lower number = higher priority)
		if instances[0].Priority > instances[1].Priority {
			t.Errorf("Instances not sorted by priority correctly")
		}

		// Priority 1 instances should come before priority 2
		priority1Count := 0
		priority2Count := 0
		foundPriority2 := false

		for _, instance := range instances {
			if instance.Priority == 1 {
				if foundPriority2 {
					t.Errorf("Priority 1 instance found after priority 2 instance")
				}
				priority1Count++
			} else if instance.Priority == 2 {
				foundPriority2 = true
				priority2Count++
			}
		}

		if priority1Count != 2 {
			t.Errorf("Expected 2 priority 1 instances, got: %d", priority1Count)
		}

		if priority2Count != 1 {
			t.Errorf("Expected 1 priority 2 instance, got: %d", priority2Count)
		}
	})

	// Test empty instances
	t.Run("EmptyInstancesList", func(t *testing.T) {
		emptyConfig := &apiv1.MCPConfig{
			Instances:       []*apiv1.MCPInstance{},
			LoadBalanceMode: apiv1.LoadBalanceModeRoundRobin,
		}
		lb := &LoadBalancer{config: emptyConfig}

		instance := lb.selectInstance("test-registry")
		if instance != nil {
			t.Errorf("Expected nil instance for empty list, got: %v", instance)
		}
	})
}

// TestConfigManagerFailureScenarios tests various failure scenarios
func TestConfigManagerFailureScenarios(t *testing.T) {
	kubeClient := kube.NewFakeClient()
	configManager, err := config.SetupExtendedConfigManager(kubeClient.Kube(), "default")
	if err != nil {
		t.Fatalf("Failed to setup config manager: %v", err)
	}

	// Test timeout scenarios
	t.Run("ContextTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1)
		defer cancel()

		// Wait for context to timeout
		time.Sleep(time.Millisecond * 2)

		_, err := configManager.GetMCPConfig(ctx, config.ConfigSourceConfigMap, "test-config")
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	})

	// Test unsupported source
	t.Run("UnsupportedConfigSource", func(t *testing.T) {
		ctx := context.Background()
		_, err := configManager.GetMCPConfig(ctx, "unsupported-source", "test-config")
		if err == nil {
			t.Error("Expected error for unsupported source, got nil")
		}
	})

	// Test configuration validation failures
	t.Run("ConfigValidationFailure", func(t *testing.T) {
		// Create ConfigMap with invalid configuration
		invalidConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalid-config",
				Namespace: "default",
				Labels: map[string]string{
					"app.higress.io/mcp-config": "true",
				},
			},
			Data: map[string]string{
				"config": `{
					"instances": [
						{
							"domain": "",
							"port": 99999,
							"weight": -10
						}
					]
				}`,
			},
		}

		_, err := kubeClient.Kube().CoreV1().ConfigMaps("default").Create(
			context.Background(), invalidConfigMap, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create invalid ConfigMap: %v", err)
		}

		ctx := context.Background()
		_, err = configManager.GetMCPConfig(ctx, config.ConfigSourceConfigMap, "invalid-config")
		if err == nil {
			t.Error("Expected validation error, got nil")
		}
	})
}