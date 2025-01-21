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
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
	Register(WasmPluginsReplayProtection)
}

func generateBase64Nonce(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

var WasmPluginsReplayProtection = suite.ConformanceTest{
	ShortName:   "WasmPluginsReplayProtection",
	Description: "The replay protection wasm plugin prevents replay attacks by validating request nonce.",
	Manifests:   []string{"tests/replay-protection.yaml"}, // Path to your YAML
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		replayNonce := generateBase64Nonce(32)
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",          // or the correct backend name
					TargetNamespace: "higress-conformance-infra", // or the correct namespace
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com", // Or your test host
						Path:             "/get",    // Or your test path
						UnfollowRedirect: true,
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/get",
							Host: "foo.com",
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 400, // Missing nonce should return 400
						Body:       []byte("Missing nonce header"),
					},
					ExpectedResponseNoRequest: false,
				},
			},
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/get",
						UnfollowRedirect: true,
						Headers: map[string]string{
							"X-Higress-Nonce": "invalid-nonce",
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/get",
							Host: "foo.com",
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 400, // Invalid nonce format should return 400
						Body:       []byte("Invalid nonce"),
					},
					ExpectedResponseNoRequest: false,
				},
			},
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/get",
						UnfollowRedirect: true,
						Headers: map[string]string{
							"X-Higress-Nonce": generateBase64Nonce(32),
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/get",
							Host: "foo.com",
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
					ExpectedResponseNoRequest: false,
				},
			},
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/get",
						UnfollowRedirect: true,
						Headers: map[string]string{
							"X-Higress-Nonce": replayNonce,
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/get",
							Host: "foo.com",
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
					ExpectedResponseNoRequest: false,
				},
			},
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/get",
						UnfollowRedirect: true,
						Headers: map[string]string{
							"X-Higress-Nonce": replayNonce,
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/get",
							Host: "foo.com",
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 429,
						Body:       []byte("Duplicate nonce"),
					},
					ExpectedResponseNoRequest: false,
				},
			},
		}
		t.Run("WasmPlugins replay-protection", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
				if strings.Contains(string(testcase.Response.ExpectedResponse.Body), "Duplicate nonce") {
					time.Sleep(time.Second)
				}
			}
		})
	},
}
