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
    Register(WasmPluginsExtAuthBlacklistForwardAuth)
    Register(WasmPluginsExtAuthBlacklistEnvoy)
    Register(WasmPluginsExtAuthWhitelistForwardAuth)
    Register(WasmPluginsExtAuthWhitelistEnvoy)
}

// WasmPluginsExtAuthBlacklistForwardAuth tests ext-auth WasmPlugin in blacklist mode with forward_auth endpoint
var WasmPluginsExtAuthBlacklistForwardAuth = suite.ConformanceTest{
    ShortName:   "WasmPluginsExtAuthBlacklistForwardAuth",
    Description: "Verify ext-auth WasmPlugin in blacklist mode with forward_auth endpoint",
    Manifests:   []string{"tests/ext_auth.yaml", "tests/ext_auth_blacklist_forward_auth.yaml"},
    Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
    Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
        runCommonBlacklistTests(t, s, "blacklist-forward-auth")
    },
}

// WasmPluginsExtAuthBlacklistEnvoy tests ext-auth WasmPlugin in blacklist mode with envoy endpoint
var WasmPluginsExtAuthBlacklistEnvoy = suite.ConformanceTest{
    ShortName:   "WasmPluginsExtAuthBlacklistEnvoy",
    Description: "Verify ext-auth WasmPlugin in blacklist mode with envoy endpoint",
    Manifests:   []string{"tests/ext_auth.yaml", "tests/ext_auth_blacklist_envoy.yaml"},
    Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
    Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
        runCommonBlacklistTests(t, s, "blacklist-envoy")
    },
}

// WasmPluginsExtAuthWhitelistForwardAuth tests ext-auth WasmPlugin in whitelist mode with forward_auth endpoint
var WasmPluginsExtAuthWhitelistForwardAuth = suite.ConformanceTest{
    ShortName:   "WasmPluginsExtAuthWhitelistForwardAuth",
    Description: "Verify ext-auth WasmPlugin in whitelist mode with forward_auth endpoint",
    Manifests:   []string{"tests/ext_auth.yaml", "tests/ext_auth_whitelist_forward_auth.yaml"},
    Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
    Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
        runCommonWhitelistTests(t, s, "whitelist-forward-auth")
    },
}

// WasmPluginsExtAuthWhitelistEnvoy tests ext-auth WasmPlugin in whitelist mode with envoy endpoint
var WasmPluginsExtAuthWhitelistEnvoy = suite.ConformanceTest{
    ShortName:   "WasmPluginsExtAuthWhitelistEnvoy",
    Description: "Verify ext-auth WasmPlugin in whitelist mode with envoy endpoint",
    Manifests:   []string{"tests/ext_auth.yaml", "tests/ext_auth_whitelist_envoy.yaml"},
    Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
    Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
        runCommonWhitelistTests(t, s, "whitelist-envoy")
    },
}

// runCommonBlacklistTests contains the common test cases for blacklist mode
func runCommonBlacklistTests(t *testing.T, s *suite.ConformanceTestSuite, prefix string) {
    cases := []struct {
        name      string
        assertion http.Assertion
    }{
        {
            name: prefix + "-allowed-root",
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
            name: prefix + "-method-get-allowed",
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
            name: prefix + "-method-post-blocked",
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
            name: prefix + "-blocked-path",
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
        {
            name: prefix + "-auth-failure",
            assertion: http.Assertion{
                Meta: http.AssertionMeta{
                    TestCaseName:    "auth server failure",
                    TargetBackend:   "infra-backend-v1",
                    TargetNamespace: "higress-conformance-infra",
                },
                Request: http.AssertionRequest{
                    ActualRequest: http.Request{
                        Host: "localhost",
                        Path: "/allowed-path",
                        Headers: map[string]string{
                            "X-Test-Auth-Fail": "true",
                        },
                    },
                },
                Response: http.AssertionResponse{
                    ExpectedResponse: http.Response{
                        StatusCode: 403,
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
}

// runCommonWhitelistTests contains the common test cases for whitelist mode
func runCommonWhitelistTests(t *testing.T, s *suite.ConformanceTestSuite, prefix string) {
    cases := []struct {
        name      string
        assertion http.Assertion
    }{
        {
            name: prefix + "-allowed-path",
            assertion: http.Assertion{
                Meta: http.AssertionMeta{
                    TestCaseName:    "allowed path",
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
            name: prefix + "-method-allowed",
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
            name: prefix + "-method-get-blocked-path",
            assertion: http.Assertion{
                Meta: http.AssertionMeta{
                    TestCaseName:    "GET non-whitelisted path",
                    TargetBackend:   "infra-backend-v1",
                    TargetNamespace: "higress-conformance-infra",
                },
                Request: http.AssertionRequest{
                    ActualRequest: http.Request{
                        Host:   "localhost",
                        Path:   "/random-path",
                        Method: "GET",
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
            name: prefix + "-method-post-blocked",
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
            name: prefix + "-auth-failure",
            assertion: http.Assertion{
                Meta: http.AssertionMeta{
                    TestCaseName:    "auth server failure",
                    TargetBackend:   "infra-backend-v1",
                    TargetNamespace: "higress-conformance-infra",
                },
                Request: http.AssertionRequest{
                    ActualRequest: http.Request{
                        Host: "localhost",
                        Path: "/allowed-path",
                        Headers: map[string]string{
                            "X-Test-Auth-Fail": "true",
                        },
                    },
                },
                Response: http.AssertionResponse{
                    ExpectedResponse: http.Response{
                        StatusCode: 403,
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
}