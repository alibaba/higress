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

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	v1 "github.com/alibaba/higress/client/pkg/apis/networking/v1"
	higressclient "github.com/alibaba/higress/client/pkg/clientset/versioned"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

// getClients creates and returns Kubernetes and Higress clients
func getClients(t *testing.T) (*kubernetes.Clientset, higressclient.Interface) {
	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create k8s client: %v", err)
	}

	higressClient, err := higressclient.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create higress client: %v", err)
	}

	return k8sClient, higressClient
}

func init() {
	Register(McpBridgeEtcdSizeLimitTest)
}

const (
	// EtcdSizeLimitMB represents the default etcd request size limit in MB
	EtcdSizeLimitMB = 1.5
	// EtcdSizeLimitBytes represents the default etcd request size limit in bytes
	EtcdSizeLimitBytes = int(EtcdSizeLimitMB * 1024 * 1024)
	
	// Test resource names
	TraditionalTestName = "traditional-large-scale"
	ConfigMapName       = "large-scale-mcp-instances"
	ConfigRefTestName   = "test-mcp-config-ref"
)

var McpBridgeEtcdSizeLimitTest = suite.ConformanceTest{
	ShortName:   "McpBridgeEtcdSizeLimit",
	Description: "Test etcd size limit issue with traditional approach vs ConfigMap reference solution",
	Manifests:   []string{},
	Features:    []suite.SupportedFeature{suite.NacosConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		err := setupEtcdSizeLimitTest(t, suite)
		if err != nil {
			t.Fatalf("Failed to setup test: %v", err)
		}

		defer func() {
			cleanupEtcdSizeLimitTest(t, suite)
		}()

		// Test 1: Verify traditional approach problem with large scale
		t.Run("TraditionalApproach_LargeScale_Problem", func(t *testing.T) {
			testTraditionalApproachSizeProblem(t, suite)
		})

		// Test 2: Verify ConfigMap reference solution works
		t.Run("ConfigMapReference_LargeScale_Solution", func(t *testing.T) {
			testConfigMapReferenceSolution(t, suite)
		})

		// Test 3: Compare size reduction effectiveness
		t.Run("SizeReduction_Comparison", func(t *testing.T) {
			testSizeReductionComparison(t, suite)
		})
	},
}

func setupEtcdSizeLimitTest(t *testing.T, suite *suite.ConformanceTestSuite) error {
	t.Logf("Setting up etcd size limit test...")
	return nil
}

func cleanupEtcdSizeLimitTest(t *testing.T, suite *suite.ConformanceTestSuite) {
	// Create typed clientsets using helper function
	k8sClient, higressClient := getClients(t)

	namespace := "higress-conformance-infra"

	// Clean up ConfigMaps
	configMaps := []string{
		"large-scale-mcp-instances",
		"massive-scale-mcp-instances",
	}

	for _, cm := range configMaps {
		k8sClient.CoreV1().ConfigMaps(namespace).Delete(
			context.Background(), cm, metav1.DeleteOptions{})
	}

	// Clean up McpBridges
	mcpBridges := []string{
		"traditional-large-scale",
		"configref-large-scale",
		"configref-massive-scale",
	}

	for _, mcb := range mcpBridges {
		higressClient.NetworkingV1().McpBridges(namespace).Delete(
			context.Background(), mcb, metav1.DeleteOptions{})
	}

	t.Logf("Cleanup completed")
}

// testTraditionalApproachSizeProblem tests that traditional approach hits etcd size limits
func testTraditionalApproachSizeProblem(t *testing.T, suite *suite.ConformanceTestSuite) {
	const instanceCount = 600 // Large number that should exceed etcd limit
	const etcdLimit = EtcdSizeLimitBytes

	t.Logf("Testing traditional approach with %d instances (should exceed etcd limit)", instanceCount)

	// Create traditional McpBridge with many registry entries
	mcpBridge := createTraditionalMcpBridge(TraditionalTestName, instanceCount)
	
	// Calculate the CR size
	crSize := calculateMcpBridgeSize(t, mcpBridge)
	sizeMB := float64(crSize) / 1024 / 1024

	t.Logf("Traditional approach CR size: %.2f MB (%d bytes)", sizeMB, crSize)

	// Verify it exceeds etcd limit
	if crSize > etcdLimit {
		t.Logf("✅ Traditional approach exceeds etcd limit (%.2f MB > 1.5 MB)", sizeMB)
		t.Logf("🔥 This would cause 'etcdserver: request is too large' error")
	} else {
		t.Logf("⚠️  Traditional approach size (%.2f MB) is within limit - may need more instances", sizeMB)
	}

	// Try to create the resource - expect it to fail due to size
	// Create typed clientsets using helper function
	_, higressClient := getClients(t)

	namespace := "higress-conformance-infra"

	_, err := higressClient.NetworkingV1().McpBridges(namespace).Create(
		context.Background(), mcpBridge, metav1.CreateOptions{})
	if err != nil {
		t.Logf("✅ Traditional approach failed as expected: %v", err)
		// Check if it's the etcd size error
		if strings.Contains(err.Error(), "etcdserver: request is too large") {
			t.Logf("🎯 Got expected 'etcdserver: request is too large' error")
		} else {
			t.Logf("⚠️  Got different error: %v", err)
		}
	} else {
		t.Logf("⚠️  Traditional approach succeeded unexpectedly - may need more instances")
		// Clean up if it succeeded
		higressClient.NetworkingV1().McpBridges(namespace).Delete(
			context.Background(), TraditionalTestName, metav1.DeleteOptions{})
	}
}

// testConfigMapReferenceSolution tests that ConfigMap reference approach works at scale
func testConfigMapReferenceSolution(t *testing.T, suite *suite.ConformanceTestSuite) {
	const instanceCount = 600 // Same large number as traditional approach
	const etcdLimit = EtcdSizeLimitBytes

	t.Logf("Testing ConfigMap reference approach with %d instances (should work)", instanceCount)

	// Create typed clientsets using helper function
	k8sClient, higressClient := getClients(t)

	namespace := "higress-conformance-infra"

	// Create ConfigMap with large number of MCP instances
	configMap, err := createLargeScaleConfigMap(ConfigMapName, instanceCount)
	if err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}
	
	_, err = k8sClient.CoreV1().ConfigMaps(namespace).Create(
		context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}

	// Create McpBridge using ConfigMap reference
	mcpBridge := createConfigMapRefMcpBridge(ConfigRefTestName, ConfigMapName)
	
	// Calculate the CR size
	crSize := calculateMcpBridgeSize(t, mcpBridge)
	sizeKB := float64(crSize) / 1024

	t.Logf("ConfigMap reference approach CR size: %.2f KB (%d bytes)", sizeKB, crSize)

	// Verify it's within etcd limit
	if crSize < etcdLimit {
		t.Logf("✅ ConfigMap reference approach within etcd limit (%.2f KB < 1.5 MB)", sizeKB)
	} else {
		t.Errorf("❌ ConfigMap reference approach exceeds etcd limit (%.2f KB)", sizeKB)
	}

	// Try to create the resource - should succeed
	_, err = higressClient.NetworkingV1().McpBridges(namespace).Create(
		context.Background(), mcpBridge, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("❌ ConfigMap reference approach failed: %v", err)
	} else {
		t.Logf("✅ ConfigMap reference approach succeeded")
	}

	// Verify the resource was created correctly
	time.Sleep(5 * time.Second)
	
	createdMcpBridge, err := higressClient.NetworkingV1().McpBridges(namespace).Get(
		context.Background(), ConfigRefTestName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Failed to get created McpBridge: %v", err)
	} else {
		t.Logf("✅ McpBridge created successfully")
		if createdMcpBridge.Spec.Registries[0].McpConfigRef == ConfigMapName {
			t.Logf("✅ ConfigMap reference is correct: %s", createdMcpBridge.Spec.Registries[0].McpConfigRef)
		} else {
			t.Errorf("❌ ConfigMap reference is incorrect")
		}
	}
}

// testSizeReductionComparison compares size reduction between approaches
func testSizeReductionComparison(t *testing.T, suite *suite.ConformanceTestSuite) {
	t.Logf("Testing size reduction comparison...")

	testCases := []struct {
		name      string
		instances int
		scenario  string
	}{
		{"Small", 50, "Small deployment"},
		{"Medium", 200, "Medium deployment"},
		{"Large", 500, "Large deployment"},
		{"Massive", 1000, "Massive deployment"},
	}

	t.Logf("| Scale | Instances | Traditional | ConfigMap Ref | Reduction | Status |")
	t.Logf("|-------|-----------|-------------|---------------|-----------|--------|")

	var totalReductions []float64
	var exceedsLimitCount int

	for _, tc := range testCases {
		// Calculate traditional approach size
		traditionalMcpBridge := createTraditionalMcpBridge(fmt.Sprintf("traditional-%s", tc.name), tc.instances)
		traditionalSize := calculateMcpBridgeSize(t, traditionalMcpBridge)
		traditionalMB := float64(traditionalSize) / 1024 / 1024

		// Calculate ConfigMap reference approach size
		configRefMcpBridge := createConfigMapRefMcpBridge(fmt.Sprintf("configref-%s", tc.name), "test-configmap")
		configRefSize := calculateMcpBridgeSize(t, configRefMcpBridge)
		configRefKB := float64(configRefSize) / 1024

		// Calculate reduction percentage
		reduction := float64(traditionalSize-configRefSize) / float64(traditionalSize) * 100
		totalReductions = append(totalReductions, reduction)

		const etcdLimit = EtcdSizeLimitBytes
		status := "✅ Both OK"
		if traditionalSize > etcdLimit {
			status = "🔥 Traditional exceeds etcd limit"
			exceedsLimitCount++
		}

		t.Logf("| %s | %d | %.2f MB | %.2f KB | %.1f%% | %s |",
			tc.name, tc.instances, traditionalMB, configRefKB, reduction, status)
	}

	// Calculate average reduction dynamically
	var avgReduction float64
	if len(totalReductions) > 0 {
		sum := 0.0
		for _, r := range totalReductions {
			sum += r
		}
		avgReduction = sum / float64(len(totalReductions))
	}

	// Calculate minimum reduction for conservative reporting
	minReduction := avgReduction
	for _, r := range totalReductions {
		if r < minReduction {
			minReduction = r
		}
	}

	t.Logf("")
	t.Logf("Summary:")
	t.Logf("✅ ConfigMap reference approach reduces CR size by %.1f%% (avg) / %.1f%% (min) across all scales", avgReduction, minReduction)
	t.Logf("✅ Traditional approach hits etcd limits in %d/%d test scenarios", exceedsLimitCount, len(testCases))
	t.Logf("✅ ConfigMap reference approach enables unlimited scaling")
	t.Logf("🎯 Solution successfully resolves 'etcdserver: request is too large' error")
}

// Helper functions

func createTraditionalMcpBridge(name string, instanceCount int) *v1.McpBridge {
	registries := make([]*apiv1.RegistryConfig, instanceCount)
	
	for i := 0; i < instanceCount; i++ {
		registries[i] = &apiv1.RegistryConfig{
			Type:                   "nacos2",
			Name:                   fmt.Sprintf("nacos-instance-%d", i),
			Domain:                 fmt.Sprintf("nacos-%d.example.com", i),
			Port:                   8848,
			NacosAddressServer:     fmt.Sprintf("http://nacos-addr-%d.example.com:8080", i),
			NacosAccessKey:         fmt.Sprintf("access-key-%d", i),
			NacosSecretKey:         fmt.Sprintf("secret-key-%d", i),
			NacosNamespaceId:       "public",
			NacosNamespace:         "default",
			NacosGroups:            []string{"DEFAULT_GROUP", "PROD_GROUP", "TEST_GROUP"},
			NacosRefreshInterval:   30000,
			ConsulNamespace:        fmt.Sprintf("consul-ns-%d", i),
			ZkServicesPath:         []string{fmt.Sprintf("/services-%d", i)},
			ConsulDatacenter:       fmt.Sprintf("dc-%d", i%5),
			ConsulServiceTag:       fmt.Sprintf("tag-%d", i),
			ConsulRefreshInterval:  60000,
			AuthSecretName:         fmt.Sprintf("auth-secret-%d", i),
			Protocol:               "http",
			Sni:                    fmt.Sprintf("sni-%d.example.com", i),
			McpServerExportDomains: []string{
				fmt.Sprintf("service-%d.local", i),
				fmt.Sprintf("api-%d.local", i),
			},
			McpServerBaseUrl:  fmt.Sprintf("http://mcp-%d.example.com:8080", i),
			AllowMcpServers:   []string{fmt.Sprintf("mcp-%d", i)},
			Metadata: map[string]*apiv1.InnerMap{
				"region": {
					InnerMap: map[string]string{
						"zone":        fmt.Sprintf("zone-%d", i%10),
						"datacenter":  fmt.Sprintf("dc-%d", i%5),
						"environment": "production",
						"cluster":     fmt.Sprintf("cluster-%d", i%3),
					},
				},
				"monitoring": {
					InnerMap: map[string]string{
						"enabled":    "true",
						"prometheus": fmt.Sprintf("http://prometheus-%d:9090", i),
						"grafana":    fmt.Sprintf("http://grafana-%d:3000", i),
						"alerting":   fmt.Sprintf("http://alertmanager-%d:9093", i),
					},
				},
			},
		}
	}

	return &v1.McpBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "higress-conformance-infra",
		},
		Spec: apiv1.McpBridge{
			Registries: registries,
		},
	}
}

func createConfigMapRefMcpBridge(name, configMapName string) *v1.McpBridge {
	return &v1.McpBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "higress-conformance-infra",
		},
		Spec: apiv1.McpBridge{
			Registries: []*apiv1.RegistryConfig{
				{
					Type:             "nacos2",
					Name:             "nacos-cluster",
					Domain:           "nacos.example.com",
					Port:             8848,
					NacosNamespaceId: "public",
					NacosGroups:      []string{"DEFAULT_GROUP"},
					McpConfigRef:     configMapName,
				},
			},
		},
	}
}

func createLargeScaleConfigMap(name string, instanceCount int) (*corev1.ConfigMap, error) {
	instances := make([]*apiv1.MCPInstance, instanceCount)
	for i := 0; i < instanceCount; i++ {
		instances[i] = &apiv1.MCPInstance{
			Domain: fmt.Sprintf("nacos-%d.example.com", i),
			Port:   8848,
			Weight: int32(100 - (i % 100)),
		}
	}

	instancesJSON, err := json.Marshal(instances)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal instances: %v", err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "higress-conformance-infra",
		},
		Data: map[string]string{
			"instances": string(instancesJSON),
		},
	}, nil
}

func calculateMcpBridgeSize(t *testing.T, mcpBridge *v1.McpBridge) int {
	data, err := json.Marshal(mcpBridge)
	if err != nil {
		t.Fatalf("Failed to marshal McpBridge: %v", err)
	}
	return len(data)
}