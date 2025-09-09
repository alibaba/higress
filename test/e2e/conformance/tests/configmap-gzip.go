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
	"testing"

	"github.com/alibaba/higress/pkg/ingress/kube/configmap"
	"github.com/alibaba/higress/test/e2e/conformance/utils/envoy"
	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/kubernetes"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
	Register(ConfigmapGzip)
	Register(ConfigMapGzipEnvoy)
}

var testCases = []struct {
	higressConfig  *configmap.HigressConfig
	envoyAssertion envoy.Assertion
	httpAssert     http.Assertion
}{
	{
		higressConfig: &configmap.HigressConfig{
			Gzip: &configmap.Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
		},
		httpAssert: http.Assertion{
			Meta: http.AssertionMeta{
				TestCaseName:    "case1: disable gzip output",
				TargetBackend:   "web-backend",
				TargetNamespace: "higress-conformance-infra",
			},
			Request: http.AssertionRequest{
				ActualRequest: http.Request{
					Host:   "foo.com",
					Path:   "/foo",
					Method: "GET",
					Headers: map[string]string{
						"Accept-Encoding": "*",
					},
				},
			},
			Response: http.AssertionResponse{
				ExpectedResponseNoRequest: true,
				ExpectedResponse: http.Response{
					StatusCode:    200,
					AbsentHeaders: []string{"content-encoding"},
				},
			},
		},
		envoyAssertion: envoy.Assertion{
			Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config.http_filters",
			TargetNamespace: "higress-system",
			CheckType:       envoy.CheckTypeNotExist,
			ExpectEnvoyConfig: map[string]interface{}{
				"name": "envoy.filters.http.gzip",
				"@type": "type.googleapis.com/envoy.extensions.filters.http.gzip.v3.Gzip",
			},
		},
	},
	{
		higressConfig: &configmap.HigressConfig{
			Gzip: &configmap.Gzip{
				Enable:              true,
				MinContentLength:    100,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
		},
		httpAssert: http.Assertion{
			Meta: http.AssertionMeta{
				TestCaseName:    "case2: enable gzip output",
				TargetBackend:   "web-backend",
				TargetNamespace: "higress-conformance-infra",
			},
			Request: http.AssertionRequest{
				ActualRequest: http.Request{
					Host:   "foo.com",
					Path:   "/foo",
					Method: "GET",
					Headers: map[string]string{
						"Accept-Encoding": "*",
					},
				},
			},
			Response: http.AssertionResponse{
				ExpectedResponseNoRequest: true,
				ExpectedResponse: http.Response{
					StatusCode: 200,
				},
				AdditionalResponseHeaders: map[string]string{"content-encoding": "gzip"},
			},
		},
		envoyAssertion: envoy.Assertion{
			Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains",
			TargetNamespace: "higress-system",
			CheckType:       envoy.CheckTypeExist,
			ExpectEnvoyConfig: map[string]interface{}{
				"name":                   "envoy.filters.network.http_connection_manager",
				"@type":                  "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
				"stat_prefix":            "outbound_0.0.0.0_80",
				"memory_level":           5,
				"compression_level":      "COMPRESSION_LEVEL_9",
				"window_bits":            12,
				"min_content_length":     100,
				"disable_on_etag_header": true,
				"content_type": []interface{}{
					"text/html",
					"text/css",
					"text/plain",
					"text/xml",
					"application/json",
					"application/javascript",
					"application/xhtml+xml",
					"image/svg+xml",
				},
			},
		},
	},
	{
		higressConfig: &configmap.HigressConfig{
			Gzip: &configmap.Gzip{
				Enable:              true,
				MinContentLength:    4096,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
		},
		httpAssert: http.Assertion{
			Meta: http.AssertionMeta{
				TestCaseName:    "case3: disable gzip output because content length less hhan 4096 ",
				TargetBackend:   "web-backend",
				TargetNamespace: "higress-conformance-infra",
			},
			Request: http.AssertionRequest{
				ActualRequest: http.Request{
					Host:   "foo.com",
					Path:   "/foo",
					Method: "GET",
					Headers: map[string]string{
						"Accept-Encoding": "*",
					},
				},
			},
			Response: http.AssertionResponse{
				ExpectedResponseNoRequest: true,
				ExpectedResponse: http.Response{
					StatusCode:    200,
					AbsentHeaders: []string{"content-encoding"},
				},
			},
		},
		envoyAssertion: envoy.Assertion{
			Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains",
			TargetNamespace: "higress-system",
			CheckType:       envoy.CheckTypeExist,
			ExpectEnvoyConfig: map[string]interface{}{
				"name":                   "envoy.filters.network.http_connection_manager",
				"@type":                  "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
				"stat_prefix":            "outbound_0.0.0.0_80",
				"memory_level":           5,
				"compression_level":      "COMPRESSION_LEVEL_9",
				"window_bits":            12,
				"min_content_length":     4096,
				"disable_on_etag_header": true,
				"content_type": []interface{}{
					"text/html",
					"text/css",
					"text/plain",
					"text/xml",
					"application/json",
					"application/javascript",
					"application/xhtml+xml",
					"image/svg+xml",
				},
			},
		},
	},
	{
		higressConfig: &configmap.HigressConfig{
			Gzip: &configmap.Gzip{
				Enable:              true,
				MinContentLength:    100,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
		},
		httpAssert: http.Assertion{
			Meta: http.AssertionMeta{
				TestCaseName:    "case4: disable gzip output because application/json missed in content types ",
				TargetBackend:   "web-backend",
				TargetNamespace: "higress-conformance-infra",
			},
			Request: http.AssertionRequest{
				ActualRequest: http.Request{
					Host:   "foo.com",
					Path:   "/foo",
					Method: "GET",
					Headers: map[string]string{
						"Accept-Encoding": "*",
					},
				},
			},
			Response: http.AssertionResponse{
				ExpectedResponseNoRequest: true,
				ExpectedResponse: http.Response{
					StatusCode:    200,
					AbsentHeaders: []string{"content-encoding"},
				},
			},
		},
		envoyAssertion: envoy.Assertion{
			Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains",
			TargetNamespace: "higress-system",
			CheckType:       envoy.CheckTypeExist,
			ExpectEnvoyConfig: map[string]interface{}{
				"name":                   "envoy.filters.network.http_connection_manager",
				"@type":                  "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
				"stat_prefix":            "outbound_0.0.0.0_80",
				"memory_level":           5,
				"compression_level":      "COMPRESSION_LEVEL_9",
				"window_bits":            12,
				"min_content_length":     100,
				"disable_on_etag_header": true,
				"content_type": []interface{}{
					"text/html",
					"text/css",
					"text/plain",
					"text/xml",
					"application/javascript",
					"application/xhtml+xml",
					"image/svg+xml",
				},
			},
		},
	},
}

var ConfigmapGzip = suite.ConformanceTest{
	ShortName:   "ConfigmapGzip",
	Description: "The Ingress in the higress-conformance-infra namespace uses the configmap gzip.",
	Manifests:   []string{"tests/configmap-gzip.yaml"},
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Parallel:    false,
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		t.Log("🚀 ConfigmapGzip: Test started")
		t.Run("Configmap Gzip", func(t *testing.T) {
			t.Log("📍 ConfigmapGzip: Processing", len(testCases), "test cases")
			for i, testcase := range testCases {
				t.Logf("📍 ConfigmapGzip: Processing test case %d/%d: %s", i+1, len(testCases), testcase.httpAssert.Meta.TestCaseName)
				
				err := kubernetes.ApplyConfigmapDataWithYaml(t, suite.Client, "higress-system", "higress-config", "higress", testcase.higressConfig)
				if err != nil {
					t.Logf("❌ ConfigmapGzip: Failed to apply config for test case %d: %v", i+1, err)
					t.Logf("📍 ConfigmapGzip: Failed to apply configmap %s in namespace %s for data key %s", "higress-config", "higress-system", "higress")
					t.FailNow()
				}
				t.Logf("✅ ConfigmapGzip: Config applied for test case %d", i+1)
				
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase.httpAssert)
				t.Logf("✅ ConfigmapGzip: HTTP assertion passed for test case %d", i+1)
			}
			t.Log("🎉 ConfigmapGzip: All test cases completed successfully")
		})
	},
}

var ConfigMapGzipEnvoy = suite.ConformanceTest{
	ShortName:   "ConfigMapGzipEnvoy",
	Description: "The Envoy config should contain gzip config",
	Manifests:   []string{"tests/configmap-gzip.yaml"},
	Features:    []suite.SupportedFeature{suite.EnvoyConfigConformanceFeature},
	Parallel:    false,
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		t.Log("🚀 ConfigMapGzipEnvoy: Test started")
		t.Run("ConfigMap Gzip Envoy", func(t *testing.T) {
			t.Log("📍 ConfigMapGzipEnvoy: Processing", len(testCases), "test cases")
			for i, testcase := range testCases {
				t.Logf("📍 ConfigMapGzipEnvoy: Processing test case %d/%d", i+1, len(testCases))
				
				// apply config
				err := kubernetes.ApplyConfigmapDataWithYaml(t, suite.Client, "higress-system", "higress-config", "higress", testcase.higressConfig)
				if err != nil {
					t.Logf("❌ ConfigMapGzipEnvoy: Failed to apply config for test case %d: %v", i+1, err)
					t.Logf("📍 ConfigMapGzipEnvoy: Failed to apply configmap %s in namespace %s for data key %s", "higress-config", "higress-system", "higress")
					t.FailNow()
				}
				t.Logf("✅ ConfigMapGzipEnvoy: Config applied for test case %d", i+1)
				
				envoy.AssertEnvoyConfig(t, suite.TimeoutConfig, testcase.envoyAssertion)
				t.Logf("✅ ConfigMapGzipEnvoy: Envoy assertion passed for test case %d", i+1)
			}
			t.Log("🎉 ConfigMapGzipEnvoy: All test cases completed successfully")
		})
	},
}
