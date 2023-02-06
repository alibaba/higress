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

	"github.com/alibaba/higress/test/ingress/conformance/utils/http"
	"github.com/alibaba/higress/test/ingress/conformance/utils/suite"
)

func init() {
	HigressConformanceTests = append(HigressConformanceTests, HTTPRouteHostNameSameNamespace)
}

var HTTPRouteHostNameSameNamespace = suite.ConformanceTest{
	ShortName:   "HTTPRouteHostNameSameNamespace",
	Description: "A Ingress in the higress-conformance-infra namespace demonstrates host match ability",
	Manifests:   []string{"tests/httproute-hostname-same-namespace.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {

		t.Run("Simple HTTP request should reach infra-backend", func(t *testing.T) {
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, http.ExpectedResponse{
				Request:   http.Request{Path: "/foo", Host: "foo.com"},
				Response:  http.Response{StatusCode: 200},
				Backend:   "infra-backend-v1",
				Namespace: "higress-conformance-infra",
			})

			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, http.ExpectedResponse{
				Request:   http.Request{Path: "/foo", Host: "bar.com"},
				Response:  http.Response{StatusCode: 200},
				Backend:   "infra-backend-v2",
				Namespace: "higress-conformance-infra",
			})

			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, http.ExpectedResponse{
				Request:   http.Request{Path: "/bar", Host: "foo.com"},
				Response:  http.Response{StatusCode: 200},
				Backend:   "infra-backend-v2",
				Namespace: "higress-conformance-infra",
			})

			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, http.ExpectedResponse{
				Request:   http.Request{Path: "/bar", Host: "bar.com"},
				Response:  http.Response{StatusCode: 200},
				Backend:   "infra-backend-v3",
				Namespace: "higress-conformance-infra",
			})

			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, http.ExpectedResponse{
				Request:   http.Request{Path: "/any", Host: "any.bar.com"},
				Response:  http.Response{StatusCode: 200},
				Backend:   "infra-backend-v1",
				Namespace: "higress-conformance-infra",
			})
		})
	},
}
