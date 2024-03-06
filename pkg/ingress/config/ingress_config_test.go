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

package config

import (
	"testing"

	httppb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/config/xds"
	ingress "k8s.io/api/networking/v1"
	ingressv1beta1 "k8s.io/api/networking/v1beta1"

	"github.com/alibaba/higress/pkg/ingress/kube/annotations"
	"github.com/alibaba/higress/pkg/ingress/kube/common"
	controllerv1beta1 "github.com/alibaba/higress/pkg/ingress/kube/ingress"
	controllerv1 "github.com/alibaba/higress/pkg/ingress/kube/ingressv1"
	"github.com/alibaba/higress/pkg/kube"
)

func TestNormalizeWeightedCluster(t *testing.T) {
	validate := func(route *common.WrapperHTTPRoute) int32 {
		var total int32
		for _, routeDestination := range route.HTTPRoute.Route {
			total += routeDestination.Weight
		}

		return total
	}

	var testCases []*common.WrapperHTTPRoute
	testCases = append(testCases, &common.WrapperHTTPRoute{
		HTTPRoute: &networking.HTTPRoute{
			Route: []*networking.HTTPRouteDestination{
				{
					Weight: 100,
				},
			},
		},
	})
	testCases = append(testCases, &common.WrapperHTTPRoute{
		HTTPRoute: &networking.HTTPRoute{
			Route: []*networking.HTTPRouteDestination{
				{
					Weight: 98,
				},
			},
		},
	})

	testCases = append(testCases, &common.WrapperHTTPRoute{
		HTTPRoute: &networking.HTTPRoute{
			Route: []*networking.HTTPRouteDestination{
				{
					Weight: 0,
				},
				{
					Weight: 48,
				},
				{
					Weight: 48,
				},
			},
		},
		WeightTotal: 100,
	})

	testCases = append(testCases, &common.WrapperHTTPRoute{
		HTTPRoute: &networking.HTTPRoute{
			Route: []*networking.HTTPRouteDestination{
				{
					Weight: 0,
				},
				{
					Weight: 48,
				},
				{
					Weight: 48,
				},
			},
		},
		WeightTotal: 80,
	})

	for _, route := range testCases {
		t.Run("", func(t *testing.T) {
			normalizeWeightedCluster(nil, route)
			if validate(route) != 100 {
				t.Fatalf("Weight sum should be 100, but actual is %d", validate(route))
			}
		})
	}
}

func TestConvertGatewaysForIngress(t *testing.T) {
	fake := kube.NewFakeClient()
	v1Beta1Options := common.Options{
		Enable:           true,
		ClusterId:        "ingress-v1beta1",
		RawClusterId:     "ingress-v1beta1__",
		GatewayHttpPort:  80,
		GatewayHttpsPort: 443,
	}
	v1Options := common.Options{
		Enable:           true,
		ClusterId:        "ingress-v1",
		RawClusterId:     "ingress-v1__",
		GatewayHttpPort:  80,
		GatewayHttpsPort: 443,
	}
	ingressV1Beta1Controller := controllerv1beta1.NewController(fake, fake, v1Beta1Options, nil)
	ingressV1Controller := controllerv1.NewController(fake, fake, v1Options, nil)
	m := NewIngressConfig(fake, nil, "wakanda", "gw-123-istio")
	m.remoteIngressControllers = map[string]common.IngressController{
		"ingress-v1beta1": ingressV1Beta1Controller,
		"ingress-v1":      ingressV1Controller,
	}

	testCases := []struct {
		name        string
		inputConfig []common.WrapperConfig
		expect      map[string]config.Config
	}{
		{
			name: "ingress v1beta1",
			inputConfig: []common.WrapperConfig{
				{
					Config: &config.Config{
						Meta: config.Meta{
							Name:      "test-1",
							Namespace: "wakanda",
							Annotations: map[string]string{
								common.ClusterIdAnnotation: "ingress-v1beta1",
							},
						},
						Spec: ingressv1beta1.IngressSpec{
							TLS: []ingressv1beta1.IngressTLS{
								{
									Hosts:      []string{"test.com"},
									SecretName: "test-com",
								},
							},
							Rules: []ingressv1beta1.IngressRule{
								{
									Host: "foo.com",
									IngressRuleValue: ingressv1beta1.IngressRuleValue{
										HTTP: &ingressv1beta1.HTTPIngressRuleValue{
											Paths: []ingressv1beta1.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
								{
									Host: "test.com",
									IngressRuleValue: ingressv1beta1.IngressRuleValue{
										HTTP: &ingressv1beta1.HTTPIngressRuleValue{
											Paths: []ingressv1beta1.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
							},
						},
					},
					AnnotationsConfig: &annotations.Ingress{
						DownstreamTLS: &annotations.DownstreamTLSConfig{
							CipherSuites: []string{"ECDHE-RSA-AES128-GCM-SHA256", "AES256-SHA"},
						},
					},
				},
				{
					Config: &config.Config{
						Meta: config.Meta{
							Name:      "test-2",
							Namespace: "wakanda",
							Annotations: map[string]string{
								common.ClusterIdAnnotation: "ingress-v1beta1",
							},
						},
						Spec: ingressv1beta1.IngressSpec{
							TLS: []ingressv1beta1.IngressTLS{
								{
									Hosts:      []string{"foo.com"},
									SecretName: "foo-com",
								},
								{
									Hosts:      []string{"test.com"},
									SecretName: "test-com-2",
								},
							},
							Rules: []ingressv1beta1.IngressRule{
								{
									Host: "foo.com",
									IngressRuleValue: ingressv1beta1.IngressRuleValue{
										HTTP: &ingressv1beta1.HTTPIngressRuleValue{
											Paths: []ingressv1beta1.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
								{
									Host: "bar.com",
									IngressRuleValue: ingressv1beta1.IngressRuleValue{
										HTTP: &ingressv1beta1.HTTPIngressRuleValue{
											Paths: []ingressv1beta1.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
								{
									Host: "test.com",
									IngressRuleValue: ingressv1beta1.IngressRuleValue{
										HTTP: &ingressv1beta1.HTTPIngressRuleValue{
											Paths: []ingressv1beta1.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
							},
						},
					},
					AnnotationsConfig: &annotations.Ingress{
						DownstreamTLS: &annotations.DownstreamTLSConfig{
							CipherSuites: []string{"ECDHE-RSA-AES128-GCM-SHA256"},
						},
					},
				},
			},
			expect: map[string]config.Config{
				"foo.com": {
					Meta: config.Meta{
						GroupVersionKind: gvk.Gateway,
						Name:             "istio-autogenerated-k8s-ingress-" + common.CleanHost("foo.com"),
						Namespace:        "wakanda",
						Annotations: map[string]string{
							common.ClusterIdAnnotation: "ingress-v1beta1",
							common.HostAnnotation:      "foo.com",
						},
					},
					Spec: &networking.Gateway{
						Servers: []*networking.Server{
							{
								Port: &networking.Port{
									Number:   80,
									Protocol: "HTTP",
									Name:     "http-80-ingress-ingress-v1beta1",
								},
								Hosts: []string{"foo.com"},
							},
							{
								Port: &networking.Port{
									Number:   443,
									Protocol: "HTTPS",
									Name:     "https-443-ingress-ingress-v1beta1",
								},
								Hosts: []string{"foo.com"},
								Tls: &networking.ServerTLSSettings{
									Mode:           networking.ServerTLSSettings_SIMPLE,
									CredentialName: "kubernetes-ingress://ingress-v1beta1__/wakanda/foo-com",
									CipherSuites:   []string{"ECDHE-RSA-AES128-GCM-SHA256", "AES256-SHA"},
								},
							},
						},
					},
				},
				"test.com": {
					Meta: config.Meta{
						GroupVersionKind: gvk.Gateway,
						Name:             "istio-autogenerated-k8s-ingress-" + common.CleanHost("test.com"),
						Namespace:        "wakanda",
						Annotations: map[string]string{
							common.ClusterIdAnnotation: "ingress-v1beta1",
							common.HostAnnotation:      "test.com",
						},
					},
					Spec: &networking.Gateway{
						Servers: []*networking.Server{
							{
								Port: &networking.Port{
									Number:   80,
									Protocol: "HTTP",
									Name:     "http-80-ingress-ingress-v1beta1",
								},
								Hosts: []string{"test.com"},
							},
							{
								Port: &networking.Port{
									Number:   443,
									Protocol: "HTTPS",
									Name:     "https-443-ingress-ingress-v1beta1",
								},
								Hosts: []string{"test.com"},
								Tls: &networking.ServerTLSSettings{
									Mode:           networking.ServerTLSSettings_SIMPLE,
									CredentialName: "kubernetes-ingress://ingress-v1beta1__/wakanda/test-com",
									CipherSuites:   []string{"ECDHE-RSA-AES128-GCM-SHA256", "AES256-SHA"},
								},
							},
						},
					},
				},
				"bar.com": {
					Meta: config.Meta{
						GroupVersionKind: gvk.Gateway,
						Name:             "istio-autogenerated-k8s-ingress-" + common.CleanHost("bar.com"),
						Namespace:        "wakanda",
						Annotations: map[string]string{
							common.ClusterIdAnnotation: "ingress-v1beta1",
							common.HostAnnotation:      "bar.com",
						},
					},
					Spec: &networking.Gateway{
						Servers: []*networking.Server{
							{
								Port: &networking.Port{
									Number:   80,
									Protocol: "HTTP",
									Name:     "http-80-ingress-ingress-v1beta1",
								},
								Hosts: []string{"bar.com"},
							},
						},
					},
				},
			},
		},
		{
			name: "ingress v1",
			inputConfig: []common.WrapperConfig{
				{
					Config: &config.Config{
						Meta: config.Meta{
							Name:      "test-1",
							Namespace: "wakanda",
							Annotations: map[string]string{
								common.ClusterIdAnnotation: "ingress-v1",
							},
						},
						Spec: ingress.IngressSpec{
							TLS: []ingress.IngressTLS{
								{
									Hosts:      []string{"test.com"},
									SecretName: "test-com",
								},
							},
							Rules: []ingress.IngressRule{
								{
									Host: "foo.com",
									IngressRuleValue: ingress.IngressRuleValue{
										HTTP: &ingress.HTTPIngressRuleValue{
											Paths: []ingress.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
								{
									Host: "test.com",
									IngressRuleValue: ingress.IngressRuleValue{
										HTTP: &ingress.HTTPIngressRuleValue{
											Paths: []ingress.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
							},
						},
					},
					AnnotationsConfig: &annotations.Ingress{},
				},
				{
					Config: &config.Config{
						Meta: config.Meta{
							Name:      "test-2",
							Namespace: "wakanda",
							Annotations: map[string]string{
								common.ClusterIdAnnotation: "ingress-v1",
							},
						},
						Spec: ingress.IngressSpec{
							TLS: []ingress.IngressTLS{
								{
									Hosts:      []string{"foo.com"},
									SecretName: "foo-com",
								},
								{
									Hosts:      []string{"test.com"},
									SecretName: "test-com-2",
								},
							},
							Rules: []ingress.IngressRule{
								{
									Host: "foo.com",
									IngressRuleValue: ingress.IngressRuleValue{
										HTTP: &ingress.HTTPIngressRuleValue{
											Paths: []ingress.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
								{
									Host: "bar.com",
									IngressRuleValue: ingress.IngressRuleValue{
										HTTP: &ingress.HTTPIngressRuleValue{
											Paths: []ingress.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
								{
									Host: "test.com",
									IngressRuleValue: ingress.IngressRuleValue{
										HTTP: &ingress.HTTPIngressRuleValue{
											Paths: []ingress.HTTPIngressPath{
												{
													Path: "/test",
												},
											},
										},
									},
								},
							},
						},
					},
					AnnotationsConfig: &annotations.Ingress{
						DownstreamTLS: &annotations.DownstreamTLSConfig{
							CipherSuites: []string{"ECDHE-RSA-AES128-GCM-SHA256"},
						},
					},
				},
			},
			expect: map[string]config.Config{
				"foo.com": {
					Meta: config.Meta{
						GroupVersionKind: gvk.Gateway,
						Name:             "istio-autogenerated-k8s-ingress-" + common.CleanHost("foo.com"),
						Namespace:        "wakanda",
						Annotations: map[string]string{
							common.ClusterIdAnnotation: "ingress-v1",
							common.HostAnnotation:      "foo.com",
						},
					},
					Spec: &networking.Gateway{
						Servers: []*networking.Server{
							{
								Port: &networking.Port{
									Number:   80,
									Protocol: "HTTP",
									Name:     "http-80-ingress-ingress-v1",
								},
								Hosts: []string{"foo.com"},
							},
							{
								Port: &networking.Port{
									Number:   443,
									Protocol: "HTTPS",
									Name:     "https-443-ingress-ingress-v1",
								},
								Hosts: []string{"foo.com"},
								Tls: &networking.ServerTLSSettings{
									Mode:           networking.ServerTLSSettings_SIMPLE,
									CredentialName: "kubernetes-ingress://ingress-v1__/wakanda/foo-com",
									CipherSuites:   []string{"ECDHE-RSA-AES128-GCM-SHA256"},
								},
							},
						},
					},
				},
				"test.com": {
					Meta: config.Meta{
						GroupVersionKind: gvk.Gateway,
						Name:             "istio-autogenerated-k8s-ingress-" + common.CleanHost("test.com"),
						Namespace:        "wakanda",
						Annotations: map[string]string{
							common.ClusterIdAnnotation: "ingress-v1",
							common.HostAnnotation:      "test.com",
						},
					},
					Spec: &networking.Gateway{
						Servers: []*networking.Server{
							{
								Port: &networking.Port{
									Number:   80,
									Protocol: "HTTP",
									Name:     "http-80-ingress-ingress-v1",
								},
								Hosts: []string{"test.com"},
							},
							{
								Port: &networking.Port{
									Number:   443,
									Protocol: "HTTPS",
									Name:     "https-443-ingress-ingress-v1",
								},
								Hosts: []string{"test.com"},
								Tls: &networking.ServerTLSSettings{
									Mode:           networking.ServerTLSSettings_SIMPLE,
									CredentialName: "kubernetes-ingress://ingress-v1__/wakanda/test-com",
									CipherSuites:   []string{"ECDHE-RSA-AES128-GCM-SHA256"},
								},
							},
						},
					},
				},
				"bar.com": {
					Meta: config.Meta{
						GroupVersionKind: gvk.Gateway,
						Name:             "istio-autogenerated-k8s-ingress-" + common.CleanHost("bar.com"),
						Namespace:        "wakanda",
						Annotations: map[string]string{
							common.ClusterIdAnnotation: "ingress-v1",
							common.HostAnnotation:      "bar.com",
						},
					},
					Spec: &networking.Gateway{
						Servers: []*networking.Server{
							{
								Port: &networking.Port{
									Number:   80,
									Protocol: "HTTP",
									Name:     "http-80-ingress-ingress-v1",
								},
								Hosts: []string{"bar.com"},
							},
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := m.convertGateways(testCase.inputConfig)
			target := map[string]config.Config{}
			for _, item := range result {
				host := common.GetHost(item.Annotations)
				target[host] = item
			}
			assert.Equal(t, testCase.expect, target)
		})
	}
}

func TestConstructBasicAuthEnvoyFilter(t *testing.T) {
	rules := &common.BasicAuthRules{
		Rules: []*common.Rule{
			{
				Realm:       "test",
				MatchRoute:  []string{"route"},
				Credentials: []string{"user:password"},
				Encrypted:   true,
			},
		},
	}

	config, err := constructBasicAuthEnvoyFilter(rules, "")
	if err != nil {
		t.Fatalf("construct error %v", err)
	}
	envoyFilter := config.Spec.(*networking.EnvoyFilter)
	pb, err := xds.BuildXDSObjectFromStruct(networking.EnvoyFilter_HTTP_FILTER, envoyFilter.ConfigPatches[0].Patch.Value, false)
	if err != nil {
		t.Fatalf("build object error %v", err)
	}
	target := proto.Clone(pb).(*httppb.HttpFilter)
	t.Log(target)
}
