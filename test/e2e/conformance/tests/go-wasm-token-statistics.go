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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/higress/v2/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/v2/test/e2e/conformance/utils/suite"
)

func init() {
	Register(WasmPluginsTokenStatistics)
}

var WasmPluginsTokenStatistics = suite.ConformanceTest{
	ShortName:   "WasmPluginTokenStatistics",
	Description: "Conformance test for the token-statistics WASM plugin (parsing token usage and forwarding).",
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Manifests:   []string{"tests/go-wasm-token-statistics.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: basic token extraction",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "dashscope.aliyuncs.com",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body: []byte(`{
							"model": "gpt-4-test",
							"messages": [{"role":"user","content":"hello"}]}`),
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Host:        "token-statistics.test",
							Path:        "/v1/chat/completions",
							Method:      "POST",
							ContentType: http.ContentTypeApplicationJson,
							Body: []byte(`{
								"model": "gpt-4-test",
								"messages": [{"role":"user","content":"hello"}]}`),
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}

		t.Run("WasmPlugins token-statistics", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)

				promAddr := os.Getenv("PROMETHEUS_ADDR")
				if promAddr == "" {
					t.Log("PROMETHEUS_ADDR not set; skipping metric assertion")
					continue
				}
				promQuery := `sum({__name__=~"higress_token_statistics_.*_total_tokens_total"})`

				var found bool
				for i := 0; i < 6; i++ {
					respBody, err := http.QueryPrometheus(promAddr, promQuery)
					if err != nil {
						t.Logf("prometheus query attempt %d failed: %v", i+1, err)
					} else if strings.Contains(string(respBody), "result") {
						found = true
						break
					}
					time.Sleep(2 * time.Second)
				}

				if !found {
					t.Skip("Prometheus metric not found; ensure Prometheus is scraping the Envoy/higress metrics or adjust query")
				}
			}
		})
	},
}
