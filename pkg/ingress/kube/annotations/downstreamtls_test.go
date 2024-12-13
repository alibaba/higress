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

package annotations

import (
		"reflect"
		"testing"

		networking "istio.io/api/networking/v1alpha3"
		"k8s.io/apimachinery/pkg/types"
)

var parser = downstreamTLS{}

func TestParse(t *testing.T) {
		testCases := []struct {
				name   string
				input  map[string]string
				expect *DownstreamTLSConfig
		}{
				{
						name: "empty annotations",
				},
				{
						name: "cipher suite only",
						input: map[string]string{
								buildNginxAnnotationKey(sslCipher): "ECDHE-RSA-AES256-GCM-SHA384:AES128-SHA",
						},
						expect: &DownstreamTLSConfig{
								Mode:           networking.ServerTLSSettings_SIMPLE,
								CipherSuites:   []string{"ECDHE-RSA-AES256-GCM-SHA384", "AES128-SHA"},
								RuleMinVersion: make(map[string]string),
								RuleMaxVersion: make(map[string]string),
						},
				},
				{
						name: "global TLS version only",
						input: map[string]string{
								buildNginxAnnotationKey(annotationMinTLSVersion): "TLSv1_2",
								buildNginxAnnotationKey(annotationMaxTLSVersion): "TLSv1_3",
						},
						expect: &DownstreamTLSConfig{
								Mode:           networking.ServerTLSSettings_SIMPLE,
								MinVersion:     "TLSv1_2",
								MaxVersion:     "TLSv1_3",
								RuleMinVersion: make(map[string]string),
								RuleMaxVersion: make(map[string]string),
						},
				},
				{
						name: "rule specific TLS version",
						input: map[string]string{
								buildNginxAnnotationKey(annotationMinTLSVersion + ".rule1"): "TLSv1_1",
								buildNginxAnnotationKey(annotationMaxTLSVersion + ".rule1"): "TLSv1_2",
						},
						expect: &DownstreamTLSConfig{
								Mode:           networking.ServerTLSSettings_SIMPLE,
								RuleMinVersion: map[string]string{"rule1": "TLSv1_1"},
								RuleMaxVersion: map[string]string{"rule1": "TLSv1_2"},
						},
				},
				{
						name: "global and rule specific TLS version",
						input: map[string]string{
								buildNginxAnnotationKey(annotationMinTLSVersion):           "TLSv1_2",
								buildNginxAnnotationKey(annotationMaxTLSVersion):           "TLSv1_3",
								buildNginxAnnotationKey(annotationMinTLSVersion + ".rule1"): "TLSv1_1",
								buildNginxAnnotationKey(annotationMaxTLSVersion + ".rule1"): "TLSv1_2",
						},
						expect: &DownstreamTLSConfig{
								Mode:           networking.ServerTLSSettings_SIMPLE,
								MinVersion:     "TLSv1_2",
								MaxVersion:     "TLSv1_3",
								RuleMinVersion: map[string]string{"rule1": "TLSv1_1"},
								RuleMaxVersion: map[string]string{"rule1": "TLSv1_2"},
						},
				},
				{
						name: "multiple rules TLS version",
						input: map[string]string{
								buildNginxAnnotationKey(annotationMinTLSVersion + ".rule1"): "TLSv1_1",
								buildNginxAnnotationKey(annotationMaxTLSVersion + ".rule1"): "TLSv1_2",
								buildNginxAnnotationKey(annotationMinTLSVersion + ".rule2"): "TLSv1_2",
								buildNginxAnnotationKey(annotationMaxTLSVersion + ".rule2"): "TLSv1_3",
						},
						expect: &DownstreamTLSConfig{
								Mode: networking.ServerTLSSettings_SIMPLE,
								RuleMinVersion: map[string]string{
										"rule1": "TLSv1_1",
										"rule2": "TLSv1_2",
								},
								RuleMaxVersion: map[string]string{
										"rule1": "TLSv1_2",
										"rule2": "TLSv1_3",
								},
						},
				},
				{
						name: "complete configuration",
						input: map[string]string{
								buildHigressAnnotationKey(authTLSSecret):                   "test/foo",
								buildHigressAnnotationKey(annotationMinTLSVersion):         "TLSv1_2",
								buildHigressAnnotationKey(annotationMaxTLSVersion):         "TLSv1_3",
								buildHigressAnnotationKey(annotationMinTLSVersion + ".rule1"): "TLSv1_1",
								buildHigressAnnotationKey(annotationMaxTLSVersion + ".rule1"): "TLSv1_2",
								DefaultAnnotationsPrefix + "/" + sslCipher:                 "ECDHE-RSA-AES256-GCM-SHA384",
						},
						expect: &DownstreamTLSConfig{
								CASecretName: types.NamespacedName{
										Namespace: "test",
										Name:      "foo",
								},
								Mode:         networking.ServerTLSSettings_MUTUAL,
								MinVersion:   "TLSv1_2",
								MaxVersion:   "TLSv1_3",
								CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384"},
								RuleMinVersion: map[string]string{"rule1": "TLSv1_1"},
								RuleMaxVersion: map[string]string{"rule1": "TLSv1_2"},
						},
				},
		}

		for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
						config := &Ingress{
								Meta: Meta{
										Namespace: "foo",
								},
						}
						err := parser.Parse(tc.input, config, nil)
						if err != nil {
								t.Fatalf("Parse failed: %v", err)
						}

						if !reflect.DeepEqual(tc.expect, config.DownstreamTLS) {
								t.Fatalf("Parse result mismatch:\nExpect: %+v\nGot: %+v", tc.expect, config.DownstreamTLS)
						}
				})
		}
}

func TestApplyGateway(t *testing.T) {
	testCases := []struct {
			name   string
			input  *networking.Gateway
			config *Ingress
			expect *networking.Gateway
	}{
			{
					name: "global TLS version only",
					input: &networking.Gateway{
							Servers: []*networking.Server{
									{
											Name: "rule1",
											Port: &networking.Port{
													Protocol: "HTTPS",
											},
											Tls: &networking.ServerTLSSettings{
													Mode: networking.ServerTLSSettings_SIMPLE,
											},
									},
							},
					},
					config: &Ingress{
							DownstreamTLS: &DownstreamTLSConfig{
									Mode:           networking.ServerTLSSettings_SIMPLE,
									MinVersion:     "TLSv1_2",
									MaxVersion:     "TLSv1_3",
									RuleMinVersion: make(map[string]string),
									RuleMaxVersion: make(map[string]string),
							},
					},
					expect: &networking.Gateway{
							Servers: []*networking.Server{
									{
											Name: "rule1",
											Port: &networking.Port{
													Protocol: "HTTPS",
											},
											Tls: &networking.ServerTLSSettings{
													Mode:              networking.ServerTLSSettings_SIMPLE,
													MinProtocolVersion: networking.ServerTLSSettings_TLSV1_2,
													MaxProtocolVersion: networking.ServerTLSSettings_TLSV1_3,
											},
									},
							},
					},
			},
			{
					name: "rule specific TLS version",
					input: &networking.Gateway{
							Servers: []*networking.Server{
									{
											Name: "rule1",
											Port: &networking.Port{
													Protocol: "HTTPS",
											},
											Tls: &networking.ServerTLSSettings{
													Mode: networking.ServerTLSSettings_SIMPLE,
											},
									},
							},
					},
					config: &Ingress{
							DownstreamTLS: &DownstreamTLSConfig{
									Mode: networking.ServerTLSSettings_SIMPLE,
									RuleMinVersion: map[string]string{
											"rule1": "TLSv1_1",
									},
									RuleMaxVersion: map[string]string{
											"rule1": "TLSv1_2",
									},
							},
					},
					expect: &networking.Gateway{
							Servers: []*networking.Server{
									{
											Name: "rule1",
											Port: &networking.Port{
													Protocol: "HTTPS",
											},
											Tls: &networking.ServerTLSSettings{
													Mode:              networking.ServerTLSSettings_SIMPLE,
													MinProtocolVersion: networking.ServerTLSSettings_TLSV1_1,
													MaxProtocolVersion: networking.ServerTLSSettings_TLSV1_2,
											},
									},
							},
					},
			},
			{
					name: "rule override global TLS version",
					input: &networking.Gateway{
							Servers: []*networking.Server{
									{
											Name: "rule1",
											Port: &networking.Port{
													Protocol: "HTTPS",
											},
											Tls: &networking.ServerTLSSettings{
													Mode: networking.ServerTLSSettings_SIMPLE,
											},
									},
							},
					},
					config: &Ingress{
							DownstreamTLS: &DownstreamTLSConfig{
									Mode:       networking.ServerTLSSettings_SIMPLE,
									MinVersion: "TLSv1_2",
									MaxVersion: "TLSv1_3",
									RuleMinVersion: map[string]string{
											"rule1": "TLSv1_1",
									},
									RuleMaxVersion: map[string]string{
											"rule1": "TLSv1_2",
									},
							},
					},
					expect: &networking.Gateway{
							Servers: []*networking.Server{
									{
											Name: "rule1",
											Port: &networking.Port{
													Protocol: "HTTPS",
											},
											Tls: &networking.ServerTLSSettings{
													Mode:              networking.ServerTLSSettings_SIMPLE,
													MinProtocolVersion: networking.ServerTLSSettings_TLSV1_1,
													MaxProtocolVersion: networking.ServerTLSSettings_TLSV1_2,
											},
									},
							},
					},
			},
			{
					name: "complete configuration with cipher suites",
					input: &networking.Gateway{
							Servers: []*networking.Server{
									{
											Name: "rule1",
											Port: &networking.Port{
													Protocol: "HTTPS",
											},
											Tls: &networking.ServerTLSSettings{
													Mode:           networking.ServerTLSSettings_SIMPLE,
													CredentialName: "kubernetes-ingress://cluster/foo/bar",
											},
									},
							},
					},
					config: &Ingress{
							DownstreamTLS: &DownstreamTLSConfig{
									Mode:       networking.ServerTLSSettings_MUTUAL,
									MinVersion: "TLSv1_2",
									MaxVersion: "TLSv1_3",
									RuleMinVersion: map[string]string{
											"rule1": "TLSv1_1",
									},
									RuleMaxVersion: map[string]string{
											"rule1": "TLSv1_2",
									},
									CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384"},
									CASecretName: types.NamespacedName{
											Namespace: "foo",
											Name:      "bar",
									},
							},
					},
					expect: &networking.Gateway{
							Servers: []*networking.Server{
									{
											Name: "rule1",
											Port: &networking.Port{
													Protocol: "HTTPS",
											},
											Tls: &networking.ServerTLSSettings{
													Mode:              networking.ServerTLSSettings_MUTUAL,
													CredentialName:    "kubernetes-ingress://cluster/foo/bar",
													MinProtocolVersion: networking.ServerTLSSettings_TLSV1_1,
													MaxProtocolVersion: networking.ServerTLSSettings_TLSV1_2,
													CipherSuites:      []string{"ECDHE-RSA-AES256-GCM-SHA384"},
											},
									},
							},
					},
			},
	}

	for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
					parser.ApplyGateway(tc.input, tc.config)
					if !reflect.DeepEqual(tc.input, tc.expect) {
							t.Fatalf("ApplyGateway result mismatch:\nExpect: %+v\nGot: %+v", tc.expect, tc.input)
					}
			})
	}
}

	func TestConvertTLSVersion(t *testing.T) {
		testCases := []struct {
				name    string
				version string
				expect  networking.ServerTLSSettings_TLSProtocol
		}{
				{
						name:    "TLS 1.0",
						version: "TLSv1_0",
						expect:  networking.ServerTLSSettings_TLSV1_0,
				},
				{
						name:    "TLS 1.1",
						version: "TLSv1_1",
						expect:  networking.ServerTLSSettings_TLSV1_1,
				},
				{
						name:    "TLS 1.2",
						version: "TLSv1_2",
						expect:  networking.ServerTLSSettings_TLSV1_2,
				},
				{
						name:    "TLS 1.3",
						version: "TLSv1_3",
						expect:  networking.ServerTLSSettings_TLSV1_3,
				},
				{
						name:    "invalid version",
						version: "invalid",
						expect:  networking.ServerTLSSettings_TLS_AUTO,
				},
				{
						name:    "empty version",
						version: "",
						expect:  networking.ServerTLSSettings_TLS_AUTO,
				},
		}

		for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
						result := convertTLSVersion(tc.version)
						if result != tc.expect {
								t.Errorf("convertTLSVersion(%s) = %v, want %v", tc.version, result, tc.expect)
						}
				})
		}
	}

	func TestNeedDownstreamTLS(t *testing.T) {
		testCases := []struct {
				name        string
				annotations map[string]string
				expect      bool
		}{
				{
						name:        "empty annotations",
						annotations: map[string]string{},
						expect:      false,
				},
				{
						name: "ssl cipher only",
						annotations: map[string]string{
								buildNginxAnnotationKey(sslCipher): "ECDHE-RSA-AES256-GCM-SHA384",
						},
						expect: true,
				},
				{
						name: "auth TLS secret only",
						annotations: map[string]string{
								buildNginxAnnotationKey(authTLSSecret): "test/foo",
						},
						expect: true,
				},
				{
						name: "global TLS version only",
						annotations: map[string]string{
								buildNginxAnnotationKey(annotationMinTLSVersion): "TLSv1_2",
						},
						expect: true,
				},
				{
						name: "rule specific TLS version only",
						annotations: map[string]string{
								buildNginxAnnotationKey(annotationMinTLSVersion + ".rule1"): "TLSv1_2",
						},
						expect: true,
				},
				{
						name: "multiple TLS configurations",
						annotations: map[string]string{
								buildNginxAnnotationKey(sslCipher):                          "ECDHE-RSA-AES256-GCM-SHA384",
								buildNginxAnnotationKey(annotationMinTLSVersion):            "TLSv1_2",
								buildNginxAnnotationKey(annotationMinTLSVersion + ".rule1"): "TLSv1_1",
						},
						expect: true,
				},
		}

		for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
						result := needDownstreamTLS(Annotations(tc.annotations))
						if result != tc.expect {
								t.Errorf("needDownstreamTLS() = %v, want %v", result, tc.expect)
						}
				})
		}
	}

	func TestExtraSecret(t *testing.T) {
		testCases := []struct {
				name           string
				credentialName string
				expect         types.NamespacedName
		}{
				{
						name:           "valid credential name",
						credentialName: "kubernetes-ingress://cluster/foo/bar",
						expect: types.NamespacedName{
								Namespace: "foo",
								Name:      "bar",
						},
				},
				{
						name:           "invalid credential name",
						credentialName: "invalid-format",
						expect:        types.NamespacedName{},
				},
				{
						name:           "empty credential name",
						credentialName: "",
						expect:        types.NamespacedName{},
				},
		}

		for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
						result := extraSecret(tc.credentialName)
						if !reflect.DeepEqual(result, tc.expect) {
								t.Errorf("extraSecret(%s) = %v, want %v", tc.credentialName, result, tc.expect)
						}
				})
		}
	}