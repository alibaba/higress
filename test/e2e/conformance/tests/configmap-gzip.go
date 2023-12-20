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

var ConfigmapGzip = suite.ConformanceTest{
	ShortName:   "ConfigmapGzip",
	Description: "The Ingress in the higress-conformance-infra namespace uses the configmap gzip.",
	Manifests:   []string{"tests/configmap-gzip.yaml"},
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []struct {
			higressConfig *configmap.HigressConfig
			httpAssert    http.Assertion
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
			},
		}

		t.Run("Configmap Gzip", func(t *testing.T) {
			for _, testcase := range testcases {
				err := kubernetes.ApplyConfigmapDataWithYaml(suite.Client, "higress-system", "higress-config", "higress", testcase.higressConfig)
				if err != nil {
					t.Fatalf("can't apply conifgmap %s in namespace %s for data key %s", "higress-config", "higress-system", "higress")
				}
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase.httpAssert)
			}
		})
	},
}

var ConfigMapGzipEnvoy = suite.ConformanceTest{
	ShortName:   "ConfigMapGzipEnvoy",
	Description: "The Envoy config should contain gzip config",
	Manifests:   []string{"tests/configmap-gzip.yaml"},
	Features:    []suite.SupportedFeature{suite.EnvoyConfigConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testCase := []envoy.Assertion{
			{
				//Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#",
				Path:            "configs",
				TargetNamespace: "higress-system",
				CheckType:       envoy.CheckTypeExist,
				ExpectEnvoyConfig: map[string]interface{}{
					"name": "envoy.filters.network.http_connection_manager",
					"typed_config": map[string]interface{}{
						"@type":       "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
						"stat_prefix": "outbound_0.0.0.0_80",
					},
					"memory_level":       5,
					"compression_level":  "COMPRESSION_LEVEL_9",
					"window_bits":        12,
					"min_content_length": 100,
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
		}
		for _, test := range testCase {
			if err := envoy.AssertEnvoyConfig(t, test); err != nil {
				t.Errorf("failed to assert envoy config: %v", err)
			}
		}
	},
}
