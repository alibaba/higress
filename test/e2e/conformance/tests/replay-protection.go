package tests

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

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
	Manifests:   []string{"tests/replay-protection.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		replayNonce := generateBase64Nonce(32)
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Missing nonce header",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/",
						Method: "GET",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 400,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Invalid nonce not base64 encoded",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/",
						Method: "GET",
						Headers: map[string]string{
							"X-Higress-Nonce": "invalid-nonce",
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 400,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "First request with unique nonce returns 200",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/",
						Method: "GET",
						Headers: map[string]string{
							"X-Higress-Nonce": replayNonce,
						},
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
					TestCaseName:    "Second request with repeated nonce returns 429",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/",
						Method: "GET",
						Headers: map[string]string{
							"X-Higress-Nonce": replayNonce,
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 429,
					},
				},
			},
		}

		t.Run("WasmPlugins replay-protection", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
