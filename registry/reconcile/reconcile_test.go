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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	"github.com/alibaba/higress/pkg/kube"
	. "github.com/alibaba/higress/registry"
)

func TestGetMCPConfig(t *testing.T) {
	// Create fake kubernetes client
	fakeClient := fake.NewSimpleClientset()

	// Create test ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcp-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"instances": `[
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
			]`,
		},
	}

	_, err := fakeClient.CoreV1().ConfigMaps("default").Create(
		context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test ConfigMap: %v", err)
	}

	// Create mock kube client
	kubeClient := &kube.Client{}
	kubeClient.SetKubeClient(fakeClient)

	// Create reconciler
	reconciler := &Reconciler{
		client:    kubeClient,
		namespace: "default",
	}

	// Test case 1: Valid ConfigMap reference
	t.Run("ValidConfigMapReference", func(t *testing.T) {
		registry := &apiv1.RegistryConfig{
			McpConfigRef: "test-mcp-config",
		}

		instances, err := reconciler.getMCPConfig(registry)
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
	})

	// Test case 2: Empty ConfigMap reference
	t.Run("EmptyConfigMapReference", func(t *testing.T) {
		registry := &apiv1.RegistryConfig{
			McpConfigRef: "",
		}

		instances, err := reconciler.getMCPConfig(registry)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if instances != nil {
			t.Errorf("Expected nil instances, got: %v", instances)
		}
	})

	// Test case 3: Non-existent ConfigMap reference
	t.Run("NonExistentConfigMapReference", func(t *testing.T) {
		registry := &apiv1.RegistryConfig{
			McpConfigRef: "non-existent-config",
		}

		instances, err := reconciler.getMCPConfig(registry)
		if err == nil {
			t.Error("Expected error for non-existent ConfigMap, got nil")
		}

		if instances != nil {
			t.Errorf("Expected nil instances, got: %v", instances)
		}
	})

	// Test case 4: ConfigMap without instances key
	t.Run("ConfigMapWithoutInstancesKey", func(t *testing.T) {
		// Create ConfigMap without instances key
		invalidConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalid-mcp-config",
				Namespace: "default",
			},
			Data: map[string]string{
				"other-key": "some-value",
			},
		}

		_, err := fakeClient.CoreV1().ConfigMaps("default").Create(
			context.Background(), invalidConfigMap, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create invalid ConfigMap: %v", err)
		}

		registry := &apiv1.RegistryConfig{
			McpConfigRef: "invalid-mcp-config",
		}

		instances, err := reconciler.getMCPConfig(registry)
		if err == nil {
			t.Error("Expected error for ConfigMap without instances key, got nil")
		}

		if instances != nil {
			t.Errorf("Expected nil instances, got: %v", instances)
		}
	})
}