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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
	"k8s.io/apimachinery/pkg/util/intstr"
	ingress "knative.dev/networking/pkg/apis/networking/v1alpha1"

	"github.com/alibaba/higress/pkg/ingress/kube/annotations"
	"github.com/alibaba/higress/pkg/ingress/kube/common"
	kcontrollerv1 "github.com/alibaba/higress/pkg/ingress/kube/kingress"
	"github.com/alibaba/higress/pkg/kube"
)

func TestNormalizeKWeightedCluster(t *testing.T) {
	validate := func(route *common.WrapperHTTPRoute) int32 {
		var total int32
		fmt.Print("----------------------------")
		for _, routeDestination := range route.HTTPRoute.Route {
			total += routeDestination.Weight
			fmt.Print(routeDestination.Weight)

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
			normalizeWeightedKCluster(nil, route)
			if validate(route) != 100 {
				t.Fatalf("Weight sum should be 100, but actual is %d", validate(route))
			}
		})
	}
}

func TestConvertGatewaysForKIngress(t *testing.T) {
	fake := kube.NewFakeClient()
	v1Options := common.Options{
		Enable:       true,
		ClusterId:    "kingress",
		RawClusterId: "kingress__",
	}
	kingressV1Controller := kcontrollerv1.NewController(fake, fake, v1Options, nil)
	m := NewKIngressConfig(fake, nil, "wakanda", "gw-123-istio")
	m.remoteIngressControllers = map[string]common.KIngressController{
		"kingress": kingressV1Controller,
	}

	testCases := []struct {
		name        string
		inputConfig []common.WrapperConfig
		expect      map[string]config.Config
	}{
		{
			name: "kingress",
			inputConfig: []common.WrapperConfig{
				{
					Config: &config.Config{
						Meta: config.Meta{
							Name:      "test-1",
							Namespace: "wakanda",
							Annotations: map[string]string{
								common.ClusterIdAnnotation: "kingress",
							},
						},
						Spec: ingress.IngressSpec{
							HTTPOption: ingress.HTTPOptionEnabled,
							TLS: []ingress.IngressTLS{
								{
									Hosts:      []string{"test.com"},
									SecretName: "test-com",
								},
							},
							Rules: []ingress.IngressRule{
								{
									Hosts: []string{"foo.com"},
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path: "/test",
												Splits: []ingress.IngressBackendSplit{{
													IngressBackend: ingress.IngressBackend{
														ServiceNamespace: "wakanda",
														ServiceName:      "test-service",
														ServicePort:      intstr.FromInt(80),
													},
													Percent: 100,
												}},
											},
										},
									},
									Visibility: ingress.IngressVisibilityExternalIP,
								},
								{
									Hosts: []string{"test.com"},
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path: "/test",
												Splits: []ingress.IngressBackendSplit{{
													IngressBackend: ingress.IngressBackend{
														ServiceNamespace: "wakanda",
														ServiceName:      "test-service",
														ServicePort:      intstr.FromInt(80),
													},
													Percent: 100,
												}},
											},
										},
									},
									Visibility: ingress.IngressVisibilityExternalIP,
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
								common.ClusterIdAnnotation: "kingress",
							},
						},
						Spec: ingress.IngressSpec{
							HTTPOption: ingress.HTTPOptionRedirected,
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
									Hosts: []string{"foo.com"},
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path: "/test",
												Splits: []ingress.IngressBackendSplit{{
													IngressBackend: ingress.IngressBackend{
														ServiceNamespace: "wakanda",
														ServiceName:      "test-service",
														ServicePort:      intstr.FromInt(80),
													},
													Percent: 100,
												}},
											},
										},
									},
									Visibility: ingress.IngressVisibilityExternalIP,
								},
								{
									Hosts: []string{"bar.com"},
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path: "/test",
												Splits: []ingress.IngressBackendSplit{{
													IngressBackend: ingress.IngressBackend{
														ServiceNamespace: "wakanda",
														ServiceName:      "test-service",
														ServicePort:      intstr.FromInt(80),
													},
													Percent: 100,
												}},
											},
										},
									},
									Visibility: ingress.IngressVisibilityExternalIP,
								},
								{
									Hosts: []string{"test.com"},
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path: "/test",
												Splits: []ingress.IngressBackendSplit{{
													IngressBackend: ingress.IngressBackend{
														ServiceNamespace: "wakanda",
														ServiceName:      "test-service",
														ServicePort:      intstr.FromInt(80),
													},
													Percent: 100,
												}},
											},
										},
									},
									Visibility: ingress.IngressVisibilityExternalIP,
								},
							},
						},
					},
					AnnotationsConfig: &annotations.Ingress{},
				},
				{
					Config: &config.Config{
						Meta: config.Meta{
							Name:      "test-3",
							Namespace: "wakanda",
							Annotations: map[string]string{
								common.ClusterIdAnnotation: "kingress",
							},
						},
						Spec: ingress.IngressSpec{
							HTTPOption: ingress.HTTPOptionEnabled,
							TLS: []ingress.IngressTLS{
								{
									Hosts:      []string{"foo.com"},
									SecretName: "foo-com",
								},
								{
									Hosts:      []string{"test.com"},
									SecretName: "test-com-3",
								},
							},
							Rules: []ingress.IngressRule{
								{
									Hosts: []string{"foo.com"},
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path: "/test",
												Splits: []ingress.IngressBackendSplit{{
													IngressBackend: ingress.IngressBackend{
														ServiceNamespace: "wakanda",
														ServiceName:      "test-service",
														ServicePort:      intstr.FromInt(80),
													},
													Percent: 100,
												}},
											},
										},
									},
									Visibility: ingress.IngressVisibilityExternalIP,
								},
								{
									Hosts: []string{"bar.com"},
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path: "/test",
												Splits: []ingress.IngressBackendSplit{{
													IngressBackend: ingress.IngressBackend{
														ServiceNamespace: "wakanda",
														ServiceName:      "test-service",
														ServicePort:      intstr.FromInt(80),
													},
													Percent: 100,
												}},
											},
										},
									},
									Visibility: ingress.IngressVisibilityExternalIP,
								},
								{
									Hosts: []string{"test.com"},
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path: "/test",
												Splits: []ingress.IngressBackendSplit{{
													IngressBackend: ingress.IngressBackend{
														ServiceNamespace: "wakanda",
														ServiceName:      "test-service",
														ServicePort:      intstr.FromInt(80),
													},
													Percent: 100,
												}},
											},
										},
									},
									Visibility: ingress.IngressVisibilityExternalIP,
								},
							},
						},
					},
					AnnotationsConfig: &annotations.Ingress{},
				},
			},
			expect: map[string]config.Config{
				"foo.com": {
					Meta: config.Meta{
						GroupVersionKind: gvk.Gateway,
						Name:             "istio-autogenerated-k8s-ingress-" + common.CleanHost("foo.com"),
						Namespace:        "wakanda",
						Annotations: map[string]string{
							common.ClusterIdAnnotation: "kingress",
							common.HostAnnotation:      "foo.com",
						},
					},
					Spec: &networking.Gateway{
						Servers: []*networking.Server{
							{
								Port: &networking.Port{
									Number:   80,
									Protocol: "HTTP",
									Name:     "http-80-ingress-kingress",
								},
								Hosts: []string{"foo.com"},
								//Tls: &networking.ServerTLSSettings{
								//	HttpsRedirect: true,
								//},
							},
							{
								Port: &networking.Port{
									Number:   443,
									Protocol: "HTTPS",
									Name:     "https-443-ingress-kingress",
								},
								Hosts: []string{"foo.com"},
								Tls: &networking.ServerTLSSettings{
									Mode:           networking.ServerTLSSettings_SIMPLE,
									CredentialName: "kubernetes-ingress://kingress__/wakanda/foo-com",
									//CipherSuites:   []string{"ECDHE-RSA-AES128-GCM-SHA256", "AES256-SHA"},
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
							common.ClusterIdAnnotation: "kingress",
							common.HostAnnotation:      "test.com",
						},
					},
					Spec: &networking.Gateway{
						Servers: []*networking.Server{
							{
								Port: &networking.Port{
									Number:   80,
									Protocol: "HTTP",
									Name:     "http-80-ingress-kingress",
								},
								Hosts: []string{"test.com"},
								//Tls: &networking.ServerTLSSettings{
								//	HttpsRedirect: true,
								//},
							},
							{
								Port: &networking.Port{
									Number:   443,
									Protocol: "HTTPS",
									Name:     "https-443-ingress-kingress",
								},
								Hosts: []string{"test.com"},
								Tls: &networking.ServerTLSSettings{
									Mode:           networking.ServerTLSSettings_SIMPLE,
									CredentialName: "kubernetes-ingress://kingress__/wakanda/test-com",
									//CipherSuites:   []string{"ECDHE-RSA-AES128-GCM-SHA256", "AES256-SHA"},
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
							common.ClusterIdAnnotation: "kingress",
							common.HostAnnotation:      "bar.com",
						},
					},
					Spec: &networking.Gateway{
						Servers: []*networking.Server{
							{
								Port: &networking.Port{
									Number:   80,
									Protocol: "HTTP",
									Name:     "http-80-ingress-kingress",
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
				fmt.Print(item)
				target[host] = item
			}
			assert.Equal(t, testCase.expect, target)
		})
	}
}
