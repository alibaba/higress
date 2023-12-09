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
	Register(HTTPRouteHttp2RpcCreate)
	Register(HTTPRouteHttp2RpcUpdate)
	Register(HTTPRouteHttp2RpcDelete)
}

var HTTPRouteHttp2RpcCreate = suite.ConformanceTest{
	ShortName:   "HTTPRouteHttp2RpcCreate",
	Description: "The Ingress in the higress-conformance-infra namespace test create the http2rpc.",
	Manifests:   []string{"tests/httproute-http2rpc-0-create.yaml"},
	Features:    []suite.SupportedFeature{suite.DubboConformanceFeature, suite.NacosConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/dubbo/hello?name=higress",
						Method: "GET",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponseNoRequest: true,
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}
		t.Run("HTTPRoute uses HTTP to RPC", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}

var HTTPRouteHttp2RpcUpdate = suite.ConformanceTest{
	ShortName:   "HTTPRouteHttp2RpcUpdate",
	Description: "The Ingress in the higress-conformance-infra namespace test delete the http2rpc.",
	Manifests:   []string{"tests/httproute-http2rpc-1-update.yaml"},
	Features:    []suite.SupportedFeature{suite.DubboConformanceFeature, suite.NacosConformanceFeature},
	NotCleanup:  true,
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/dubbo/hello?name=higress",
						Method: "GET",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponseNoRequest: true,
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/dubbo/hello_update?name=higress",
						Method: "GET",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponseNoRequest: true,
					ExpectedResponse: http.Response{
						StatusCode: 404,
					},
				},
			},
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/dubbo/health/readiness?type=readiness",
						Method: "GET",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponseNoRequest: true,
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/dubbo/health/liveness?type=liveness",
						Method: "GET",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponseNoRequest: true,
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}
		t.Run("HTTPRoute uses HTTP to RPC", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}

var HTTPRouteHttp2RpcDelete = suite.ConformanceTest{
	ShortName:   "HTTPRouteHttp2RpcDelete",
	Description: "The Ingress in the higress-conformance-infra namespace test delete the http2rpc.",
	PreDeleteRs: []string{"tests/httproute-http2rpc-2-delete.yaml"},
	Features:    []suite.SupportedFeature{suite.DubboConformanceFeature, suite.NacosConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/dubbo/hello_update?name=higress",
						Method: "GET",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponseNoRequest: true,
					ExpectedResponse: http.Response{
						StatusCode: 404,
					},
				},
			},
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/dubbo/health/readiness?type=readiness",
						Method: "GET",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponseNoRequest: true,
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:   "foo.com",
						Path:   "/dubbo/health/liveness?type=liveness",
						Method: "GET",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponseNoRequest: true,
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}
		t.Run("HTTPRoute uses HTTP to RPC", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
