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
	"istio.io/istio/pilot/pkg/model"
)

var parser = downstreamTLS{}

func TestParse(t *testing.T) {
	testCases := []struct {
		input  map[string]string
		expect *DownstreamTLSConfig
	}{
		{},
		{
			input: map[string]string{
				buildNginxAnnotationKey(sslCipher): "ECDHE-RSA-AES256-GCM-SHA384:AES128-SHA",
			},
			expect: &DownstreamTLSConfig{
				Mode:         networking.ServerTLSSettings_SIMPLE,
				CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384", "AES128-SHA"},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(authTLSSecret): "test",
				buildNginxAnnotationKey(sslCipher):     "ECDHE-RSA-AES256-GCM-SHA384:AES128-SHA",
			},
			expect: &DownstreamTLSConfig{
				CASecretName: model.NamespacedName{
					Namespace: "foo",
					Name:      "test",
				},
				Mode:         networking.ServerTLSSettings_MUTUAL,
				CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384", "AES128-SHA"},
			},
		},
		{
			input: map[string]string{
				buildHigressAnnotationKey(authTLSSecret):   "test/foo",
				DefaultAnnotationsPrefix + "/" + sslCipher: "ECDHE-RSA-AES256-GCM-SHA384:AES128-SHA",
			},
			expect: &DownstreamTLSConfig{
				CASecretName: model.NamespacedName{
					Namespace: "test",
					Name:      "foo",
				},
				Mode:         networking.ServerTLSSettings_MUTUAL,
				CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384", "AES128-SHA"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{
				Meta: Meta{
					Namespace: "foo",
				},
			}
			_ = parser.Parse(testCase.input, config, nil)
			if !reflect.DeepEqual(testCase.expect, config.DownstreamTLS) {
				t.Fatalf("Should be equal")
			}
		})
	}
}

func TestApplyGateway(t *testing.T) {
	testCases := []struct {
		input  *networking.Gateway
		config *Ingress
		expect *networking.Gateway
	}{
		{
			input: &networking.Gateway{
				Servers: []*networking.Server{
					{
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
					CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384"},
				},
			},
			expect: &networking.Gateway{
				Servers: []*networking.Server{
					{
						Port: &networking.Port{
							Protocol: "HTTPS",
						},
						Tls: &networking.ServerTLSSettings{
							Mode:         networking.ServerTLSSettings_SIMPLE,
							CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384"},
						},
					},
				},
			},
		},
		{
			input: &networking.Gateway{
				Servers: []*networking.Server{
					{
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
					CASecretName: model.NamespacedName{
						Namespace: "foo",
						Name:      "bar",
					},
					Mode:         networking.ServerTLSSettings_MUTUAL,
					CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384"},
				},
			},
			expect: &networking.Gateway{
				Servers: []*networking.Server{
					{
						Port: &networking.Port{
							Protocol: "HTTPS",
						},
						Tls: &networking.ServerTLSSettings{
							CredentialName: "kubernetes-ingress://cluster/foo/bar",
							Mode:           networking.ServerTLSSettings_MUTUAL,
							CipherSuites:   []string{"ECDHE-RSA-AES256-GCM-SHA384"},
						},
					},
				},
			},
		},
		{
			input: &networking.Gateway{
				Servers: []*networking.Server{
					{
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
					CASecretName: model.NamespacedName{
						Namespace: "foo",
						Name:      "bar-cacert",
					},
					Mode:         networking.ServerTLSSettings_MUTUAL,
					CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384"},
				},
			},
			expect: &networking.Gateway{
				Servers: []*networking.Server{
					{
						Port: &networking.Port{
							Protocol: "HTTPS",
						},
						Tls: &networking.ServerTLSSettings{
							CredentialName: "kubernetes-ingress://cluster/foo/bar",
							Mode:           networking.ServerTLSSettings_MUTUAL,
							CipherSuites:   []string{"ECDHE-RSA-AES256-GCM-SHA384"},
						},
					},
				},
			},
		},
		{
			input: &networking.Gateway{
				Servers: []*networking.Server{
					{
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
					CASecretName: model.NamespacedName{
						Namespace: "bar",
						Name:      "foo",
					},
					Mode:         networking.ServerTLSSettings_MUTUAL,
					CipherSuites: []string{"ECDHE-RSA-AES256-GCM-SHA384"},
				},
			},
			expect: &networking.Gateway{
				Servers: []*networking.Server{
					{
						Port: &networking.Port{
							Protocol: "HTTPS",
						},
						Tls: &networking.ServerTLSSettings{
							CredentialName: "kubernetes-ingress://cluster/foo/bar",
							Mode:           networking.ServerTLSSettings_SIMPLE,
							CipherSuites:   []string{"ECDHE-RSA-AES256-GCM-SHA384"},
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			parser.ApplyGateway(testCase.input, testCase.config)
			if !reflect.DeepEqual(testCase.input, testCase.expect) {
				t.Fatalf("Should be equal")
			}
		})
	}
}
