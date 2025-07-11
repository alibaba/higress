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
	"encoding/json"
	"testing"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	. "github.com/alibaba/higress/registry"
)

func TestMCPInstanceJSONParsing(t *testing.T) {
	// Test JSON parsing functionality without kubernetes dependencies
	t.Run("ValidJSONParsing", func(t *testing.T) {
		jsonData := `[
			{
				"domain": "nacos-1.example.com",
				"port": 8848,
				"weight": 100
			},
			{
				"domain": "nacos-2.example.com", 
				"port": 8848,
				"weight": 50
			}
		]`

		var instances []MCPInstance
		err := json.Unmarshal([]byte(jsonData), &instances)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(instances) != 2 {
			t.Errorf("Expected 2 instances, got: %d", len(instances))
		}

		if instances[0].Domain != "nacos-1.example.com" {
			t.Errorf("Expected domain 'nacos-1.example.com', got: %s", instances[0].Domain)
		}

		if instances[0].Port != 8848 {
			t.Errorf("Expected port 8848, got: %d", instances[0].Port)
		}

		if instances[0].Weight != 100 {
			t.Errorf("Expected weight 100, got: %d", instances[0].Weight)
		}

		if instances[1].Domain != "nacos-2.example.com" {
			t.Errorf("Expected domain 'nacos-2.example.com', got: %s", instances[1].Domain)
		}

		if instances[1].Weight != 50 {
			t.Errorf("Expected weight 50, got: %d", instances[1].Weight)
		}
	})

	t.Run("InvalidJSONParsing", func(t *testing.T) {
		invalidJsonData := `{ invalid json }`

		var instances []MCPInstance
		err := json.Unmarshal([]byte(invalidJsonData), &instances)
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})

	t.Run("EmptyArrayParsing", func(t *testing.T) {
		emptyJsonData := `[]`

		var instances []MCPInstance
		err := json.Unmarshal([]byte(emptyJsonData), &instances)
		if err != nil {
			t.Errorf("Expected no error for empty array, got: %v", err)
		}

		if len(instances) != 0 {
			t.Errorf("Expected 0 instances, got: %d", len(instances))
		}
	})
}

func TestMcpConfigRefField(t *testing.T) {
	// Test that the McpConfigRef field is properly accessible
	t.Run("McpConfigRefFieldAccess", func(t *testing.T) {
		registry := &apiv1.RegistryConfig{
			Type:         "nacos2",
			Name:         "test-registry",
			Domain:       "nacos.example.com",
			Port:         8848,
			McpConfigRef: "test-config-map",
		}

		if registry.McpConfigRef != "test-config-map" {
			t.Errorf("Expected McpConfigRef 'test-config-map', got: %s", registry.McpConfigRef)
		}

		// Test getter method
		if registry.GetMcpConfigRef() != "test-config-map" {
			t.Errorf("Expected GetMcpConfigRef() 'test-config-map', got: %s", registry.GetMcpConfigRef())
		}
	})

	t.Run("EmptyMcpConfigRef", func(t *testing.T) {
		registry := &apiv1.RegistryConfig{
			Type:   "nacos2",
			Name:   "test-registry",
			Domain: "nacos.example.com",
			Port:   8848,
		}

		if registry.McpConfigRef != "" {
			t.Errorf("Expected empty McpConfigRef, got: %s", registry.McpConfigRef)
		}

		if registry.GetMcpConfigRef() != "" {
			t.Errorf("Expected empty GetMcpConfigRef(), got: %s", registry.GetMcpConfigRef())
		}
	})
}