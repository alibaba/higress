package tests

import (
	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
	"testing"
)

func init() {
	Register(WasmPluginCacheControl)
}

var WasmPluginCacheControl = suite.ConformanceTest{
	ShortName:   "WasmPluginCacheControl",
	Description: "The Ingress in the higress-conformance-infra namespace test the cache control WASM Plugin",
	Manifests:   []string{"tests/go-wasm-cache-control.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: Test hit",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo.com",
						Path:    "/foo",
						Headers: map[string]string{"User-Agent": "BaiduMobaider/1.1.0"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"Cache-Control": "maxAge=3600",
						},
					},
				},
			},
		}
		t.Run("WasmPlugins cache-control", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
