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
    "github.com/alibaba/higress/test/e2e/conformance/utils/http"
    "github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
    Register(WasmPluginsExtAuth)
}

var WasmPluginsExtAuth = suite.ConformanceTest{
    ShortName:   "WasmPluginsExtAuth",
    Description: "E2E tests for ext‑auth plugin in envoy & forward_auth modes using mock-auth and echo-server",
    Manifests:   []string{"tests/go-wasm-ext-auth.yaml"},
    Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
    Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
        cases := []struct {
            name       string
            path       string
            method     string
            body       []byte
            expectCode int
        }{
            // Envoy mode (mock-auth uses exact `/prefix/always-...`)
            {"Envoy 200", "/prefix/always-200/test", "GET", nil, 200},
            {"Envoy 500", "/prefix/always-500/test", "GET", nil, 500},
            {"Envoy Body 200", "/prefix/require-request-body-200", "POST", []byte(`{"k":"v"}`), 200},
            {"Envoy Body 400", "/prefix/require-request-body-200", "POST", nil, 400},

            // Forward_auth mode
            {"Forward 200", "/always-200", "GET", nil, 200},
            {"Forward 500", "/always-500", "GET", nil, 500},
            {"Forward Body 200", "/require-request-body-200", "POST", []byte(`{"k":"v"}`), 200},
            {"Forward Body 400", "/require-request-body-200", "POST", nil, 400},
        }

        for _, tc := range cases {
            tc := tc
            t.Run(tc.name, func(t *testing.T) {
                req := http.Request{
                    Host:             "test-auth.com",
                    Path:             tc.path,
                    Method:           tc.method,
                    Headers:          map[string]string{"Authorization": "Bearer valid-token"},
                    Body:             tc.body,
                    UnfollowRedirect: true,
                }
                if tc.body != nil {
                    req.Headers["Content-Type"] = "application/json"
                }

                resp := http.Response{StatusCode: tc.expectCode}
                if tc.expectCode == 200 {
                    resp.Headers = map[string]string{"X-User-ID": "123456"}
                }

                assertion := http.Assertion{
                    Meta:     http.AssertionMeta{TestCaseName: tc.name, TargetBackend: "echo-server", TargetNamespace: "higress-conformance-infra"},
                    Request:  http.AssertionRequest{ActualRequest: req},
                    Response: http.AssertionResponse{ExpectedResponse: resp},
                }

                http.MakeRequestAndExpectEventuallyConsistentResponse(
                    t, s.RoundTripper, s.TimeoutConfig, s.GatewayAddress, assertion,
                )
            })
        }
    },
}
