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

	"github.com/alibaba/higress/test/e2e/conformance/utils/cert"
	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/kubernetes"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	Register(WasmPluginsSniMisdirect)
}

var WasmPluginsSniMisdirect = suite.ConformanceTest{
	ShortName:   "WasmPluginsSniMisdirect",
	Description: "The Ingress in the higress-conformance-infra namespace test the sni-misdirect wasmplugins.",
	Manifests:   []string{"tests/go-wasm-sni-misdirect.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		// Prepare certificates and secrets for testcases
		caCertOut, _, caCert, caKey := cert.MustGenerateCaCert(t)
		svcCertOut, svcKeyOut := cert.MustGenerateCertWithCA(t, cert.ServerCertType, caCert, caKey, []string{"foo.com"})
		cliCertOut, cliKeyOut := cert.MustGenerateCertWithCA(t, cert.ClientCertType, caCert, caKey, nil)
		fooSecret := kubernetes.ConstructTLSSecret("higress-conformance-infra", "foo-secret", svcCertOut.Bytes(), svcKeyOut.Bytes())
		fooSecretCACert := kubernetes.ConstructCASecret("higress-conformance-infra", "foo-secret-cacert", caCertOut.Bytes())
		suite.Applier.MustApplyObjectsWithCleanup(t, suite.Client, suite.TimeoutConfig, []client.Object{fooSecret, fooSecretCACert}, suite.Cleanup)

		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: http1.1 request",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
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
					TestCaseName:    "case 2: https/1.1 request with sni and same with host",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						TLSConfig: &http.TLSConfig{
							SNI: "foo.com",
							Certificates: http.Certificates{
								CACerts: [][]byte{caCertOut.Bytes()},
								ClientKeyPairs: []http.ClientKeyPair{{
									ClientCert: cliCertOut.Bytes(),
									ClientKey:  cliKeyOut.Bytes()},
								},
							},
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
					TestCaseName:    "case 3: https/2.0 request with sni and same with host",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Protocol: "HTTP/2.0",
						Host:     "foo.com",
						Path:     "/foo",
						Headers: map[string]string{
							"Content-Type": "text/plain",
						},
						TLSConfig: &http.TLSConfig{
							SNI: "foo.com",
							Certificates: http.Certificates{
								CACerts: [][]byte{caCertOut.Bytes()},
								ClientKeyPairs: []http.ClientKeyPair{{
									ClientCert: cliCertOut.Bytes(),
									ClientKey:  cliKeyOut.Bytes()},
								},
							},
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
					TestCaseName:    "case 4: https/2.0 request with sni and not same with host",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Protocol: "HTTP/2.0",
						Host:     "bar.com",
						Path:     "/foo",
						Headers: map[string]string{
							"Content-Type": "text/plain",
						},
						TLSConfig: &http.TLSConfig{
							SNI: "foo.com",
							Certificates: http.Certificates{
								CACerts: [][]byte{caCertOut.Bytes()},
								ClientKeyPairs: []http.ClientKeyPair{{
									ClientCert: cliCertOut.Bytes(),
									ClientKey:  cliKeyOut.Bytes()},
								},
							},
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 421,
					},
				},
			},
		}

		t.Run("WasmPlugin sni-misdirect", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
