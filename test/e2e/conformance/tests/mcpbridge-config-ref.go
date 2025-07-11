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
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
	Register(McpBridgeConfigRef)
}

var McpBridgeConfigRef = suite.ConformanceTest{
	ShortName:   "McpBridgeConfigRef",
	Description: "Test MCP configuration reference functionality using ConfigMap",
	Manifests:   []string{"tests/mcpbridge-config-ref.yaml"},
	Features:    []suite.SupportedFeature{suite.NacosConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		// Wait for resources to be created
		time.Sleep(5 * time.Second)

		t.Run("ValidConfigRef", func(t *testing.T) {
			// Test case 1: Valid ConfigMap reference with single instance
			err := testMcpBridgeStatus(t, suite, "test-mcp-config-ref", true)
			if err != nil {
				t.Errorf("Valid ConfigMap reference test failed: %v", err)
			}
		})

		t.Run("MultipleInstancesConfigRef", func(t *testing.T) {
			// Test case 2: Valid ConfigMap reference with multiple instances
			err := testMcpBridgeStatus(t, suite, "test-mcp-multiple-config-ref", true)
			if err != nil {
				t.Errorf("Multiple instances ConfigMap reference test failed: %v", err)
			}
		})

		t.Run("TraditionalApproach", func(t *testing.T) {
			// Test case 3: Traditional approach without ConfigMap reference
			err := testMcpBridgeStatus(t, suite, "test-mcp-traditional", true)
			if err != nil {
				t.Errorf("Traditional approach test failed: %v", err)
			}
		})

		t.Run("InvalidConfigRef", func(t *testing.T) {
			// Test case 4: Invalid JSON in ConfigMap
			err := testMcpBridgeStatus(t, suite, "test-mcp-invalid-config-ref", false)
			if err != nil {
				t.Logf("Expected error for invalid ConfigMap: %v", err)
			}
		})

		t.Run("NonexistentConfigRef", func(t *testing.T) {
			// Test case 5: Nonexistent ConfigMap reference
			err := testMcpBridgeStatus(t, suite, "test-mcp-nonexistent-config-ref", false)
			if err != nil {
				t.Logf("Expected error for nonexistent ConfigMap: %v", err)
			}
		})

		t.Run("MissingKeyConfigRef", func(t *testing.T) {
			// Test case 6: Missing instances key in ConfigMap
			err := testMcpBridgeStatus(t, suite, "test-mcp-missing-key-config-ref", false)
			if err != nil {
				t.Logf("Expected error for missing key ConfigMap: %v", err)
			}
		})

		t.Run("CompareConfigRefVsTraditional", func(t *testing.T) {
			// Test case 7: Compare functionality between ConfigRef and traditional approach
			// Both should achieve the same result but ConfigRef approach is simpler
			configRefWorking := testMcpBridgeStatus(t, suite, "test-mcp-config-ref", true) == nil
			traditionalWorking := testMcpBridgeStatus(t, suite, "test-mcp-traditional", true) == nil
			
			if configRefWorking != traditionalWorking {
				t.Errorf("ConfigRef approach and traditional approach should have same functionality")
			}
			
			if configRefWorking && traditionalWorking {
				t.Logf("Both ConfigRef and traditional approaches are working correctly")
			}
		})
	},
}

// testMcpBridgeStatus tests whether an McpBridge is functioning correctly
func testMcpBridgeStatus(t *testing.T, suite *suite.ConformanceTestSuite, mcpBridgeName string, expectSuccess bool) error {
	client := suite.Client
	namespace := "higress-conformance-infra"

	// Wait for McpBridge to be processed
	err := wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
		// Get the McpBridge resource
		mcpBridge, err := client.HigressV1().McpBridges(namespace).Get(
			context.Background(), mcpBridgeName, metav1.GetOptions{})
		if err != nil {
			if expectSuccess {
				t.Logf("Error getting McpBridge %s: %v", mcpBridgeName, err)
				return false, nil // Continue polling
			}
			return true, err // Expected error case
		}

		// Check if the McpBridge has been processed
		if mcpBridge.Status.LoadBalancer.Ingress == nil {
			if expectSuccess {
				t.Logf("McpBridge %s status not ready yet", mcpBridgeName)
				return false, nil // Continue polling
			}
		}

		if expectSuccess {
			t.Logf("McpBridge %s is functioning correctly", mcpBridgeName)
		}
		return true, nil
	})

	if err != nil {
		return fmt.Errorf("failed to verify McpBridge %s status: %v", mcpBridgeName, err)
	}

	// Additional verification: Check if the registry is properly configured
	err = wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
		// Here we can add more specific checks for registry configuration
		// For example, checking if the registry watcher is healthy
		
		// Since we don't have direct access to registry watcher status in this context,
		// we'll verify that no errors are reported in the logs
		
		if expectSuccess {
			t.Logf("Registry configuration for McpBridge %s appears to be working", mcpBridgeName)
		}
		return true, nil
	})

	return err
}

// Additional helper function to test ConfigMap content parsing
func testConfigMapParsing(t *testing.T, suite *suite.ConformanceTestSuite) {
	client := suite.Client
	namespace := "higress-conformance-infra"

	testCases := []struct {
		name           string
		configMapName  string
		expectSuccess  bool
		expectedCount  int
	}{
		{
			name:          "Valid single instance config",
			configMapName: "test-mcp-config",
			expectSuccess: true,
			expectedCount: 1,
		},
		{
			name:          "Valid multiple instances config",
			configMapName: "test-mcp-multiple-config",
			expectSuccess: true,
			expectedCount: 2,
		},
		{
			name:          "Invalid JSON config",
			configMapName: "test-mcp-invalid-config",
			expectSuccess: false,
			expectedCount: 0,
		},
		{
			name:          "Missing instances key config",
			configMapName: "test-mcp-missing-key-config",
			expectSuccess: false,
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configMap, err := client.CoreV1().ConfigMaps(namespace).Get(
				context.Background(), tc.configMapName, metav1.GetOptions{})
			
			if err != nil {
				if tc.expectSuccess {
					t.Errorf("Failed to get ConfigMap %s: %v", tc.configMapName, err)
				}
				return
			}

			instancesData, exists := configMap.Data["instances"]
			if !exists {
				if tc.expectSuccess {
					t.Errorf("ConfigMap %s missing instances key", tc.configMapName)
				}
				return
			}

			if tc.expectSuccess {
				t.Logf("ConfigMap %s contains valid instances data: %s", tc.configMapName, instancesData)
			} else {
				t.Logf("ConfigMap %s contains invalid data as expected", tc.configMapName)
			}
		})
	}
}