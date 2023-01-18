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
	HigressConformanceTests = append(HigressConformanceTests, HTTPRouteSimpleSameNamespace)
}

var HTTPRouteSimpleSameNamespace = suite.ConformanceTest{
	ShortName:   "HTTPRouteSimpleSameNamespace",
	Description: "A single Ingress in the higress-conformance-infra namespace attaches to a Gateway in the same namespace",
	Manifests:   []string{"tests/httproute-simple-same-namespace.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {

		t.Run("Simple HTTP request should reach infra-backend", func(t *testing.T) {
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, http.ExpectedResponse{
				Request:   http.Request{Path: "/hello-world"},
				Response:  http.Response{StatusCode: 200},
				Backend:   "infra-backend-v1",
				Namespace: "higress-conformance-infra",
			})
		})
	},
}
