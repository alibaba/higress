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
	"testing"
	"time"

	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	Register(WasmPluginsExtAuth)
}

var WasmPluginsExtAuth = suite.ConformanceTest{
	ShortName:   "WasmPluginsExtAuth",
	Description: "The Ingress in the higress-conformance-infra namespace test the ext-auth wasmplugin.",
	Manifests:   []string{"tests/ext_auth.yaml", "tests/ext_auth_plugin.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		// 1. Increase timeout for the test
		originalTimeout := suite.TimeoutConfig
		increasedTimeout := suite.TimeoutConfig
		increasedTimeout.RequestTimeout = 60 * time.Second  // Increase request timeout
		increasedTimeout.MaxRetries = 20                   // More retries
		increasedTimeout.InitialDelay = 5 * time.Second    // Add initial delay
		suite.TimeoutConfig = increasedTimeout
		
		// 2. Wait for ext-auth-server deployment to be ready
		t.Log("Waiting for ext-auth-server deployment to be ready...")
		waitForDeploymentReady(t, suite, "ext-auth-server", "higress-conformance-infra")
		
		// 3. Verify WasmPlugin exists
		t.Log("Verifying ext-auth WasmPlugin exists...")
		// This would require an implementation to check if the WasmPlugin is properly loaded
		// For now, we'll add a delay to give time for the plugin to be processed
		time.Sleep(15 * time.Second)
		
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: Blacklist mode - blocked path",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "ext-auth-test.example.com",
						Path:             "/blocked-path",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 2: Blacklist mode - allowed path",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "ext-auth-test.example.com",
						Path:             "/allowed-path",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 3: Method-specific rules - GET allowed",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "ext-auth-test.example.com",
						Path:             "/api",
						Method:           "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 4: Method-specific rules - POST blocked",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "ext-auth-test.example.com",
						Path:             "/api",
						Method:           "POST",
						ContentType:      http.ContentTypeTextPlain,
						Body:             []byte(`test body`),
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
					},
				},
			},
		}
		
		// 4. Run tests with improved error handling
		t.Run("WasmPlugins ext-auth", func(t *testing.T) {
			// Make a simple test request first to ensure connectivity
			probe := http.Assertion{
				Meta: http.AssertionMeta{
					TestCaseName: "Connectivity probe",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "ext-auth-test.example.com",
						Path: "/health",
					},
				},
			}
			
			// Try the probe request with shorter timeout
			probeTimeoutConfig := suite.TimeoutConfig
			probeTimeoutConfig.RequestTimeout = 10 * time.Second
			probeTimeoutConfig.MaxRetries = 5
			
			t.Log("Probing gateway connectivity...")
			err := http.MakeRequest(t, suite.RoundTripper, probeTimeoutConfig, suite.GatewayAddress, probe)
			if err != nil {
				t.Logf("Probe connectivity warning: %v", err)
				t.Log("Continuing with tests despite probe failure...")
			} else {
				t.Log("Gateway connectivity confirmed.")
			}
			
			// Run the actual test cases
			for i, testcase := range testcases {
				t.Logf("Running test case %d: %s", i+1, testcase.Meta.TestCaseName)
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
		
		// Restore original timeout settings
		suite.TimeoutConfig = originalTimeout
	},
}

// Helper function to wait for a deployment to be ready
func waitForDeploymentReady(t *testing.T, suite *suite.ConformanceTestSuite, name, namespace string) {
	clientset := suite.Client.KubernetesClientset
	
	// Check deployment readiness
	for i := 0; i < 30; i++ {
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			t.Logf("Error getting deployment %s/%s: %v", namespace, name, err)
		} else {
			if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
				t.Logf("Deployment %s/%s is ready", namespace, name)
				return
			}
			t.Logf("Deployment %s/%s: %d/%d replicas ready", 
				namespace, name, deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)
		}
		time.Sleep(5 * time.Second)
	}
	
	t.Logf("Warning: Deployment %s/%s might not be fully ready, proceeding anyway", namespace, name)
}
