// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
    "testing"

    "github.com/alibaba/higress/test/e2e/conformance/utils/http"
    "github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
    Register(WasmPluginsExtAuth)
}

var WasmPluginsExtAuth = suite.ConformanceTest{
    ShortName:   "WasmPluginsExtAuth",
    Description: "Verify ext-auth WasmPlugin in blacklist mode with header propagation",
    Manifests:   []string{"tests/ext_auth.yaml", "tests/ext_auth_plugin.yaml"},
    Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
    Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
        cases := []struct {
            name      string
            assertion http.Assertion
        }{
            {
                name: "blacklist-allowed-root",
                assertion: http.Assertion{
                    Meta: http.AssertionMeta{
                        TestCaseName:    "root path allowed",
                        TargetBackend:   "infra-backend-v1",
                        TargetNamespace: "higress-conformance-infra",
                    },
                    Request: http.AssertionRequest{
                        ActualRequest: http.Request{
                            Host: "localhost",
                            Path: "/allowed-path",
                        },
                    },
                    Response: http.AssertionResponse{
                        ExpectedResponse: http.Response{
                            StatusCode: 200,
                            Headers: map[string]string{
                                "X-Auth-Status": "OK",
                            },
                        },
                    },
                },
            },
            {
                name: "method-get-allowed",
                assertion: http.Assertion{
                    Meta: http.AssertionMeta{
                        TestCaseName:    "GET /api allowed",
                        TargetBackend:   "infra-backend-v1",
                        TargetNamespace: "higress-conformance-infra",
                    },
                    Request: http.AssertionRequest{
                        ActualRequest: http.Request{
                            Host:   "localhost",
                            Path:   "/api",
                            Method: "GET",
                        },
                    },
                    Response: http.AssertionResponse{
                        ExpectedResponse: http.Response{
                            StatusCode: 200,
                            Headers: map[string]string{
                                "X-Auth-Status": "OK",
                            },
                        },
                    },
                },
            },
            {
                name: "method-post-blocked",
                assertion: http.Assertion{
                    Meta: http.AssertionMeta{
                        TestCaseName:    "POST /api blocked",
                        TargetBackend:   "infra-backend-v1",
                        TargetNamespace: "higress-conformance-infra",
                    },
                    Request: http.AssertionRequest{
                        ActualRequest: http.Request{
                            Host:        "localhost",
                            Path:        "/api",
                            Method:      "POST",
                            Body:        []byte("test-body"),
                            ContentType: http.ContentTypeTextPlain,
                        },
                    },
                    Response: http.AssertionResponse{
                        ExpectedResponse: http.Response{
                            StatusCode: 403,
                            Headers: map[string]string{
                                "X-Auth-Status": "DENIED",
                            },
                        },
                    },
                },
            },
            {
                name: "blacklist-blocked-path",
                assertion: http.Assertion{
                    Meta: http.AssertionMeta{
                        TestCaseName:    "blocked-path denied",
                        TargetBackend:   "infra-backend-v1",
                        TargetNamespace: "higress-conformance-infra",
                    },
                    Request: http.AssertionRequest{
                        ActualRequest: http.Request{
                            Host: "localhost",
                            Path: "/blocked-path",
                        },
                    },
                    Response: http.AssertionResponse{
                        ExpectedResponse: http.Response{
                            StatusCode: 403,
                            Headers: map[string]string{
                                "X-Auth-Status": "DENIED",
                            },
                        },
                    },
                },
            },
        }

        for _, c := range cases {
            c := c // capture
            t.Run(c.name, func(t *testing.T) {
                t.Parallel()
                http.MakeRequestAndExpectEventuallyConsistentResponse(
                    t, s.RoundTripper, s.TimeoutConfig, s.GatewayAddress, c.assertion,
                )
            })
        }
    },
}
