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

package ingress

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/kube/controllers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	ingress "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/workqueue"

	"github.com/alibaba/higress/pkg/ingress/kube/annotations"
	"github.com/alibaba/higress/pkg/ingress/kube/common"
	"github.com/alibaba/higress/pkg/ingress/kube/secret"
	"github.com/alibaba/higress/pkg/kube"
	"github.com/stretchr/testify/require"
)

func TestIngressControllerApplies(t *testing.T) {
	fakeClient := kube.NewFakeClient()
	localKubeClient, client := fakeClient, fakeClient

	options := common.Options{IngressClass: "mse", ClusterId: ""}

	secretController := secret.NewController(localKubeClient, options.ClusterId)
	ingressController := NewController(localKubeClient, client, options, secretController)

	testcases := map[string]func(*testing.T, common.IngressController){
		"test apply canary ingress":  testApplyCanaryIngress,
		"test apply default backend": testApplyDefaultBackend,
	}
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			tc(t, ingressController)
		})
	}
}

func testApplyCanaryIngress(t *testing.T, c common.IngressController) {
	testcases := []struct {
		description string
		input       struct {
			options       *common.ConvertOptions
			wrapperConfig *common.WrapperConfig
		}
		expectNoError bool
	}{
		{
			description: "convertOptions is nil",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options:       nil,
				wrapperConfig: nil,
			},
			expectNoError: false,
		}, {
			description: "convertOptions is not nil but empty",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{},
				wrapperConfig: &common.WrapperConfig{
					Config:            &config.Config{},
					AnnotationsConfig: &annotations.Ingress{},
				},
			},
			expectNoError: false,
		},
		{
			description: "valid canary ingress",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{
					IngressDomainCache: &common.IngressDomainCache{
						Valid:   make(map[string]*common.IngressDomainBuilder),
						Invalid: make([]model.IngressDomain, 0),
					},
					Route2Ingress:     map[string]*common.WrapperConfigWithRuleKey{},
					VirtualServices:   make(map[string]*common.WrapperVirtualService),
					Gateways:          make(map[string]*common.WrapperGateway),
					IngressRouteCache: &common.IngressRouteCache{},
					HTTPRoutes: map[string][]*common.WrapperHTTPRoute{
						"test1": make([]*common.WrapperHTTPRoute, 0),
					},
				},
				wrapperConfig: &common.WrapperConfig{Config: &config.Config{
					Spec: ingress.IngressSpec{Rules: []ingress.IngressRule{
						{
							Host: "test1",
							IngressRuleValue: ingress.IngressRuleValue{
								HTTP: &ingress.HTTPIngressRuleValue{
									Paths: []ingress.HTTPIngressPath{
										{
											Path:     "/test",
											PathType: &defaultPathType,
										},
									},
								},
							},
						},
					},
						Backend: &ingress.IngressBackend{},
						TLS: []ingress.IngressTLS{
							{
								Hosts:      []string{"test1", "test2"},
								SecretName: "test",
							},
						}},
				}, AnnotationsConfig: &annotations.Ingress{}},
			},
			expectNoError: true,
		},
	}

	for _, testcase := range testcases {
		err := c.ApplyCanaryIngress(testcase.input.options, testcase.input.wrapperConfig)
		if err != nil {
			require.Equal(t, testcase.expectNoError, false)
		} else {
			require.Equal(t, testcase.expectNoError, true)
		}
	}
}

func testApplyDefaultBackend(t *testing.T, c common.IngressController) {
	testcases := []struct {
		description string
		input       struct {
			options       *common.ConvertOptions
			wrapperConfig *common.WrapperConfig
		}
		expectNoError bool
	}{
		{
			description: "convertOptions is nil",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options:       nil,
				wrapperConfig: nil,
			},
			expectNoError: false,
		}, {
			description: "convertOptions is not nil but empty",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{},
				wrapperConfig: &common.WrapperConfig{
					Config:            &config.Config{},
					AnnotationsConfig: &annotations.Ingress{},
				},
			},
			expectNoError: false,
		}, {
			description: "valid default backend",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{
					IngressDomainCache: &common.IngressDomainCache{
						Valid:   make(map[string]*common.IngressDomainBuilder),
						Invalid: make([]model.IngressDomain, 0),
					},
					Route2Ingress:     map[string]*common.WrapperConfigWithRuleKey{},
					VirtualServices:   make(map[string]*common.WrapperVirtualService),
					Gateways:          make(map[string]*common.WrapperGateway),
					IngressRouteCache: &common.IngressRouteCache{},
					HTTPRoutes:        make(map[string][]*common.WrapperHTTPRoute),
				},
				wrapperConfig: &common.WrapperConfig{Config: &config.Config{
					Spec: ingress.IngressSpec{Rules: []ingress.IngressRule{
						{
							Host: "test1",
							IngressRuleValue: ingress.IngressRuleValue{
								HTTP: &ingress.HTTPIngressRuleValue{
									Paths: []ingress.HTTPIngressPath{
										{
											Path:     "/test",
											PathType: &defaultPathType,
										},
									},
								},
							},
						},
					},
						Backend: &ingress.IngressBackend{},
						TLS: []ingress.IngressTLS{
							{
								Hosts:      []string{"test1", "test2"},
								SecretName: "test",
							},
						}},
				}, AnnotationsConfig: &annotations.Ingress{}},
			},
			expectNoError: true,
		},
	}

	for _, testcase := range testcases {
		err := c.ApplyDefaultBackend(testcase.input.options, testcase.input.wrapperConfig)
		if err != nil {
			require.Equal(t, testcase.expectNoError, false)
		} else {
			require.Equal(t, testcase.expectNoError, true)
		}
	}
}

func TestIngressControllerConventions(t *testing.T) {
	fakeClient := kube.NewFakeClient()
	localKubeClient, client := fakeClient, fakeClient

	options := common.Options{IngressClass: "mse", ClusterId: "", EnableStatus: true}

	secretController := secret.NewController(localKubeClient, options.ClusterId)
	ingressController := NewController(localKubeClient, client, options, secretController)

	testcases := map[string]func(*testing.T, common.IngressController){
		"test convert Gateway":       testConvertGateway,
		"test convert HTTPRoute":     testConvertHTTPRoute,
		"test convert TrafficPolicy": testConvertTrafficPolicy,
	}
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			tc(t, ingressController)
		})
	}
}

func testConvertGateway(t *testing.T, c common.IngressController) {
	testcases := []struct {
		description string
		input       struct {
			options       *common.ConvertOptions
			wrapperConfig *common.WrapperConfig
		}
		expectNoError bool
	}{
		{
			description: "convertOptions is nil",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options:       nil,
				wrapperConfig: nil,
			},
			expectNoError: false,
		}, {
			description: "convertOptions is not nil but empty",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{},
				wrapperConfig: &common.WrapperConfig{
					Config:            &config.Config{},
					AnnotationsConfig: &annotations.Ingress{},
				},
			},
			expectNoError: false,
		}, {
			description: "valid gateway convention",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{
					IngressDomainCache: &common.IngressDomainCache{
						Valid:   make(map[string]*common.IngressDomainBuilder),
						Invalid: make([]model.IngressDomain, 0),
					},
					Gateways: make(map[string]*common.WrapperGateway),
				},
				wrapperConfig: &common.WrapperConfig{Config: &config.Config{
					Spec: ingress.IngressSpec{Rules: []ingress.IngressRule{
						{
							Host: "test1",
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
						Backend: &ingress.IngressBackend{},
						TLS: []ingress.IngressTLS{
							{
								Hosts:      []string{"test1", "test2"},
								SecretName: "test",
							},
						}},
				}, AnnotationsConfig: &annotations.Ingress{}},
			},
			expectNoError: true,
		},
	}

	for _, testcase := range testcases {
		err := c.ConvertGateway(testcase.input.options, testcase.input.wrapperConfig, nil)
		if err != nil {
			require.Equal(t, testcase.expectNoError, false)
		} else {
			require.Equal(t, testcase.expectNoError, true)
		}
	}
}

func testConvertHTTPRoute(t *testing.T, c common.IngressController) {
	testcases := []struct {
		description string
		input       struct {
			options       *common.ConvertOptions
			wrapperConfig *common.WrapperConfig
		}
		expectNoError bool
	}{
		{
			description: "convertOptions is nil",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options:       nil,
				wrapperConfig: nil,
			},
			expectNoError: false,
		}, {
			description: "convertOptions is not nil but empty",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{},
				wrapperConfig: &common.WrapperConfig{
					Config:            &config.Config{},
					AnnotationsConfig: &annotations.Ingress{},
				},
			},
			expectNoError: false,
		}, {
			description: "valid httpRoute convention",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{
					IngressDomainCache: &common.IngressDomainCache{
						Valid:   make(map[string]*common.IngressDomainBuilder),
						Invalid: make([]model.IngressDomain, 0),
					},
					Route2Ingress:     map[string]*common.WrapperConfigWithRuleKey{},
					VirtualServices:   make(map[string]*common.WrapperVirtualService),
					Gateways:          make(map[string]*common.WrapperGateway),
					IngressRouteCache: &common.IngressRouteCache{},
					HTTPRoutes:        make(map[string][]*common.WrapperHTTPRoute),
				},
				wrapperConfig: &common.WrapperConfig{Config: &config.Config{
					Spec: ingress.IngressSpec{Rules: []ingress.IngressRule{
						{
							Host: "test1",
							IngressRuleValue: ingress.IngressRuleValue{
								HTTP: &ingress.HTTPIngressRuleValue{
									Paths: []ingress.HTTPIngressPath{
										{
											Path:     "/test",
											PathType: &defaultPathType,
										},
									},
								},
							},
						},
					},
						Backend: &ingress.IngressBackend{},
						TLS: []ingress.IngressTLS{
							{
								Hosts:      []string{"test1", "test2"},
								SecretName: "test",
							},
						}},
				}, AnnotationsConfig: &annotations.Ingress{},
				},
			},
			expectNoError: true,
		},
	}

	for _, testcase := range testcases {
		err := c.ConvertHTTPRoute(testcase.input.options, testcase.input.wrapperConfig)
		if err != nil {
			require.Equal(t, testcase.expectNoError, false)
		} else {
			require.Equal(t, testcase.expectNoError, true)
		}
	}
}

func testConvertTrafficPolicy(t *testing.T, c common.IngressController) {
	testcases := []struct {
		description string
		input       struct {
			options       *common.ConvertOptions
			wrapperConfig *common.WrapperConfig
		}
		expectNoError bool
	}{
		{
			description: "convertOptions is nil",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options:       nil,
				wrapperConfig: nil,
			},
			expectNoError: false,
		}, {
			description: "convertOptions is not nil but empty",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{},
				wrapperConfig: &common.WrapperConfig{
					Config:            &config.Config{},
					AnnotationsConfig: &annotations.Ingress{},
				},
			},
			expectNoError: true,
		}, {
			description: "valid trafficPolicy convention",
			input: struct {
				options       *common.ConvertOptions
				wrapperConfig *common.WrapperConfig
			}{
				options: &common.ConvertOptions{
					IngressDomainCache: &common.IngressDomainCache{
						Valid:   make(map[string]*common.IngressDomainBuilder),
						Invalid: make([]model.IngressDomain, 0),
					},
					Route2Ingress:         map[string]*common.WrapperConfigWithRuleKey{},
					VirtualServices:       make(map[string]*common.WrapperVirtualService),
					Gateways:              make(map[string]*common.WrapperGateway),
					IngressRouteCache:     &common.IngressRouteCache{},
					Service2TrafficPolicy: make(map[common.ServiceKey]*common.WrapperTrafficPolicy),
					HTTPRoutes:            make(map[string][]*common.WrapperHTTPRoute),
				},
				wrapperConfig: &common.WrapperConfig{Config: &config.Config{
					Spec: ingress.IngressSpec{Rules: []ingress.IngressRule{
						{
							Host: "test1",
							IngressRuleValue: ingress.IngressRuleValue{
								HTTP: &ingress.HTTPIngressRuleValue{
									Paths: []ingress.HTTPIngressPath{
										{
											Path:     "/test",
											PathType: &defaultPathType,
											Backend: ingress.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(8080),
											},
										},
									},
								},
							},
						},
					},
						Backend: &ingress.IngressBackend{
							ServiceName: "test",
						},
						TLS: []ingress.IngressTLS{
							{
								Hosts:      []string{"test1", "test2"},
								SecretName: "test",
							},
						}},
				}, AnnotationsConfig: &annotations.Ingress{
					LoadBalance: &annotations.LoadBalanceConfig{},
				}},
			},
			expectNoError: true,
		},
	}

	for _, testcase := range testcases {
		err := c.ConvertTrafficPolicy(testcase.input.options, testcase.input.wrapperConfig)
		if err != nil {
			require.Equal(t, testcase.expectNoError, false)
		} else {
			require.Equal(t, testcase.expectNoError, true)
		}
	}
}

func TestIngressControllerGenerations(t *testing.T) {
	c := &controller{
		options: common.Options{
			IngressClass:    "mse",
			SystemNamespace: "higress-system",
		},
		ingresses: make(map[string]*v1beta1.Ingress),
	}

	testcases := map[string]func(*testing.T, *controller){
		"test create DefaultRoute":         testcreateDefaultRoute,
		"test create ServiceKey":           testcreateServiceKey,
		"test backend to RouteDestination": testbackendToRouteDestination,
	}
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			tc(t, c)
		})
	}
}

func testcreateDefaultRoute(t *testing.T, c *controller) {
	testcases := []struct {
		input struct {
			wrapper *common.WrapperConfig
			backend *ingress.IngressBackend
			host    string
		}
		description string
		expect      *common.WrapperHTTPRoute
	}{
		{
			input: struct {
				wrapper *common.WrapperConfig
				backend *ingress.IngressBackend
				host    string
			}{
				wrapper: nil,
				backend: nil,
				host:    "",
			},
			description: "wrapperConfig is nil",
			expect:      nil,
		},
		{
			input: struct {
				wrapper *common.WrapperConfig
				backend *ingress.IngressBackend
				host    string
			}{
				wrapper: &common.WrapperConfig{},
				backend: &ingress.IngressBackend{},
				host:    "test",
			},
			description: "wrapperConfig is not nil but empty",
			expect:      nil,
		},
		{
			input: struct {
				wrapper *common.WrapperConfig
				backend *ingress.IngressBackend
				host    string
			}{
				wrapper: &common.WrapperConfig{
					Config: &config.Config{
						Meta: config.Meta{
							Namespace: "higress-system",
							Name:      "test",
						},
					},
					AnnotationsConfig: &annotations.Ingress{}},
				backend: &ingress.IngressBackend{
					ServiceName: "test",
					ServicePort: intstr.FromInt(8088),
				},
				host: "test",
			},
			description: "create expected httpRoute",
			expect: &common.WrapperHTTPRoute{
				WrapperConfig: &common.WrapperConfig{
					Config: &config.Config{
						Meta: config.Meta{
							Name:      "test",
							Namespace: "higress-system",
						},
					},
					AnnotationsConfig: &annotations.Ingress{},
				},
				RawClusterId:     "",
				ClusterId:        "",
				ClusterName:      "",
				Host:             "test",
				OriginPath:       "/",
				OriginPathType:   "prefix",
				WeightTotal:      0,
				IsDefaultBackend: true,
				HTTPRoute: &v1alpha3.HTTPRoute{
					Name: "test-default",
					Route: []*v1alpha3.HTTPRouteDestination{
						{
							Weight: 100,
							Destination: &v1alpha3.Destination{
								Port: &v1alpha3.PortSelector{
									Number: 8088,
								},
								Host: "test.higress-system.svc.cluster.local",
							},
						},
					},
				},
			},
		},
	}

	for _, testcase := range testcases {
		httpRoute := c.createDefaultRoute(testcase.input.wrapper, testcase.input.backend, testcase.input.host)
		require.Equal(t, testcase.expect, httpRoute)
	}
}

func testcreateServiceKey(t *testing.T, c *controller) {
	testcases := []struct {
		input struct {
			backend   *ingress.IngressBackend
			namespace string
		}
		expectNoError bool
		description   string
	}{
		{
			description:   "nil",
			expectNoError: false,
			input: struct {
				backend   *ingress.IngressBackend
				namespace string
			}{
				backend:   nil,
				namespace: "",
			},
		},
		{
			description:   "nil",
			expectNoError: false,
			input: struct {
				backend   *ingress.IngressBackend
				namespace string
			}{
				backend:   &ingress.IngressBackend{},
				namespace: "",
			},
		},
		{
			description:   "create success",
			expectNoError: true,
			input: struct {
				backend   *ingress.IngressBackend
				namespace string
			}{
				backend: &ingress.IngressBackend{
					ServiceName: "test",
					ServicePort: intstr.FromInt(8080),
				},
				namespace: "default",
			},
		},
	}

	for _, testcase := range testcases {
		_, err := c.createServiceKey(testcase.input.backend, testcase.input.namespace)
		if err != nil {
			require.Equal(t, testcase.expectNoError, false)
		} else {
			require.Equal(t, testcase.expectNoError, true)
		}
	}
}

func testbackendToRouteDestination(t *testing.T, c *controller) {
	testcases := []struct {
		input struct {
			backend   *ingress.IngressBackend
			namespace string
			builder   *common.IngressRouteBuilder
			config    *annotations.DestinationConfig
		}
		expectNoError bool
		description   string
	}{
		{
			description:   "nil",
			expectNoError: false,
			input: struct {
				backend   *ingress.IngressBackend
				namespace string
				builder   *common.IngressRouteBuilder
				config    *annotations.DestinationConfig
			}{
				backend:   nil,
				namespace: "",
				builder:   nil,
				config:    nil,
			},
		},
		{
			description:   "nil",
			expectNoError: false,
			input: struct {
				backend   *ingress.IngressBackend
				namespace string
				builder   *common.IngressRouteBuilder
				config    *annotations.DestinationConfig
			}{
				backend:   &ingress.IngressBackend{ServiceName: ""},
				namespace: "",
				builder:   nil,
				config:    nil,
			},
		},
		{
			description:   "create success",
			expectNoError: true,
			input: struct {
				backend   *ingress.IngressBackend
				namespace string
				builder   *common.IngressRouteBuilder
				config    *annotations.DestinationConfig
			}{
				backend: &ingress.IngressBackend{
					ServiceName: "test",
					ServicePort: intstr.FromInt(8080),
				},
				namespace: "default",
				builder:   &common.IngressRouteBuilder{},
				config:    nil,
			},
		},
	}

	for _, testcase := range testcases {
		_, err := c.backendToRouteDestination(
			testcase.input.backend,
			testcase.input.namespace,
			testcase.input.builder,
			testcase.input.config,
		)

		if err == common.InvalidBackendService {
			require.Equal(t, testcase.expectNoError, false)
		} else {
			require.Equal(t, testcase.expectNoError, true)
		}
	}
}

func TestIsCanaryRoute(t *testing.T) {
	testcases := []struct {
		input struct {
			canary *common.WrapperHTTPRoute
			route  *common.WrapperHTTPRoute
		}
		expect      bool
		description string
	}{
		{
			input: struct {
				canary *common.WrapperHTTPRoute
				route  *common.WrapperHTTPRoute
			}{
				canary: nil,
				route:  nil,
			},
			expect:      false,
			description: "both are nil",
		}, {
			input: struct {
				canary *common.WrapperHTTPRoute
				route  *common.WrapperHTTPRoute
			}{
				canary: &common.WrapperHTTPRoute{
					OriginPathType: common.Exact,
					OriginPath:     "/test",
				},
				route: &common.WrapperHTTPRoute{
					WrapperConfig: &common.WrapperConfig{
						AnnotationsConfig: &annotations.Ingress{
							Canary: nil,
						},
					},
					OriginPathType: common.Exact,
					OriginPath:     "/test",
				},
			},
			expect:      true,
			description: "canary is nil",
		}, {
			input: struct {
				canary *common.WrapperHTTPRoute
				route  *common.WrapperHTTPRoute
			}{
				canary: &common.WrapperHTTPRoute{
					OriginPathType: common.Exact,
					OriginPath:     "/test",
				},
				route: &common.WrapperHTTPRoute{
					WrapperConfig: &common.WrapperConfig{
						AnnotationsConfig: &annotations.Ingress{
							Canary: &annotations.CanaryConfig{
								Enabled: true,
							},
						},
					},
					OriginPathType: common.Exact,
					OriginPath:     "/test",
				},
			},
			expect:      false,
			description: "canary is not nil",
		},
	}
	for _, testcase := range testcases {
		actual := isCanaryRoute(testcase.input.canary, testcase.input.route)
		require.Equal(t, testcase.expect, actual)
	}
}

func TestExtractTLSSecretName(t *testing.T) {
	testcases := []struct {
		input struct {
			host string
			tls  []ingress.IngressTLS
		}
		expect      string
		description string
	}{
		{
			input: struct {
				host string
				tls  []ingress.IngressTLS
			}{
				host: "",
				tls:  nil,
			},
			expect:      "",
			description: "both are nil",
		},
		{
			input: struct {
				host string
				tls  []ingress.IngressTLS
			}{
				host: "test",
				tls: []ingress.IngressTLS{
					{
						Hosts:      []string{"test"},
						SecretName: "test-secret",
					},
					{
						Hosts:      []string{"test1"},
						SecretName: "test1-secret",
					},
				},
			},
			expect:      "test-secret",
			description: "found secret name",
		},
	}

	for _, testcase := range testcases {
		actual := extractTLSSecretName(testcase.input.host, testcase.input.tls)
		require.Equal(t, testcase.expect, actual)
	}
}

func TestSetDefaultMSEIngressOptionalField(t *testing.T) {
	pathType := ingress.PathTypeImplementationSpecific
	testcases := []struct {
		input struct {
			ing *ingress.Ingress
		}
		expect      *ingress.Ingress
		description string
	}{
		{
			input: struct{ ing *ingress.Ingress }{
				ing: nil,
			},
			expect:      nil,
			description: "nil",
		},
		{
			input: struct{ ing *ingress.Ingress }{
				ing: &ingress.Ingress{},
			},
			expect:      &ingress.Ingress{},
			description: "nil",
		}, {
			input: struct{ ing *ingress.Ingress }{
				ing: &ingress.Ingress{
					Spec: ingress.IngressSpec{
						TLS: []ingress.IngressTLS{
							{
								SecretName: "test",
							},
						},
					},
				},
			},
			expect: &ingress.Ingress{
				Spec: ingress.IngressSpec{
					TLS: []ingress.IngressTLS{
						{
							SecretName: "test",
							Hosts:      []string{"*"},
						},
					},
				},
			},
			description: "tls host is empty",
		}, {
			input: struct{ ing *ingress.Ingress }{
				ing: &ingress.Ingress{
					Spec: ingress.IngressSpec{
						TLS: []ingress.IngressTLS{
							{
								SecretName: "test",
								Hosts:      []string{"www.example.com"},
							},
						},
					},
				},
			},
			expect: &ingress.Ingress{
				Spec: ingress.IngressSpec{
					TLS: []ingress.IngressTLS{
						{
							SecretName: "test",
							Hosts:      []string{"www.example.com"},
						},
					},
				},
			},
			description: "tls host is not empty",
		}, {
			input: struct{ ing *ingress.Ingress }{
				ing: &ingress.Ingress{
					Spec: ingress.IngressSpec{
						Rules: []ingress.IngressRule{
							{
								IngressRuleValue: ingress.IngressRuleValue{
									HTTP: nil,
								},
							},
						},
						TLS: []ingress.IngressTLS{
							{
								SecretName: "test",
								Hosts:      []string{"www.example.com"},
							},
						},
					},
				},
			},
			expect: &ingress.Ingress{
				Spec: ingress.IngressSpec{
					Rules: []ingress.IngressRule{
						{
							IngressRuleValue: ingress.IngressRuleValue{
								HTTP: nil,
							},
						},
					},
					TLS: []ingress.IngressTLS{
						{
							SecretName: "test",
							Hosts:      []string{"www.example.com"},
						},
					},
				},
			},
			description: "http is nil",
		}, {
			input: struct{ ing *ingress.Ingress }{
				ing: &ingress.Ingress{
					Spec: ingress.IngressSpec{
						Rules: []ingress.IngressRule{
							{
								IngressRuleValue: ingress.IngressRuleValue{
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path:     "/test",
												PathType: &defaultPathType,
												Backend:  ingress.IngressBackend{},
											},
										},
									},
								},
							},
						},
						TLS: []ingress.IngressTLS{
							{
								SecretName: "test",
								Hosts:      []string{"www.example.com"},
							},
						},
					},
				},
			},
			expect: &ingress.Ingress{
				Spec: ingress.IngressSpec{
					Rules: []ingress.IngressRule{
						{
							Host: "*",
							IngressRuleValue: ingress.IngressRuleValue{
								HTTP: &ingress.HTTPIngressRuleValue{
									Paths: []ingress.HTTPIngressPath{
										{
											Path:     "/test",
											PathType: &defaultPathType,
											Backend:  ingress.IngressBackend{},
										},
									},
								},
							},
						},
					},
					TLS: []ingress.IngressTLS{
						{
							SecretName: "test",
							Hosts:      []string{"www.example.com"},
						},
					},
				},
			},
			description: "http is not nil but host is empty",
		}, {
			input: struct{ ing *ingress.Ingress }{
				ing: &ingress.Ingress{
					Spec: ingress.IngressSpec{
						Rules: []ingress.IngressRule{
							{
								IngressRuleValue: ingress.IngressRuleValue{
									HTTP: &ingress.HTTPIngressRuleValue{
										Paths: []ingress.HTTPIngressPath{
											{
												Path:     "/test",
												PathType: &pathType,
												Backend:  ingress.IngressBackend{},
											},
										},
									},
								},
							},
						},
						TLS: []ingress.IngressTLS{
							{
								SecretName: "test",
								Hosts:      []string{"www.example.com"},
							},
						},
					},
				},
			},
			expect: &ingress.Ingress{
				Spec: ingress.IngressSpec{
					Rules: []ingress.IngressRule{
						{
							Host: "*",
							IngressRuleValue: ingress.IngressRuleValue{
								HTTP: &ingress.HTTPIngressRuleValue{
									Paths: []ingress.HTTPIngressPath{
										{
											Path:     "/test",
											PathType: &defaultPathType,
											Backend:  ingress.IngressBackend{},
										},
									},
								},
							},
						},
					},
					TLS: []ingress.IngressTLS{
						{
							SecretName: "test",
							Hosts:      []string{"www.example.com"},
						},
					},
				},
			},
			description: "http path type is ImplementationSpecific",
		},
	}

	for _, testcase := range testcases {
		setDefaultMSEIngressOptionalField(testcase.input.ing)
		require.Equal(t, testcase.expect, testcase.input.ing)
	}
}

func TestIngressControllerProcessing(t *testing.T) {
	fakeClient := kube.NewFakeClient()
	localKubeClient, _ := fakeClient, fakeClient

	options := common.Options{IngressClass: "mse", ClusterId: "", EnableStatus: true}

	secretController := secret.NewController(localKubeClient, options.ClusterId)
	q := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())

	ingressInformer := fakeClient.KubeInformer().Networking().V1beta1().Ingresses()
	serviceInformer := fakeClient.KubeInformer().Core().V1().Services()

	ingressController := &controller{
		options:          options,
		queue:            q,
		ingresses:        make(map[string]*ingress.Ingress),
		ingressInformer:  ingressInformer.Informer(),
		ingressLister:    ingressInformer.Lister(),
		serviceInformer:  serviceInformer.Informer(),
		serviceLister:    serviceInformer.Lister(),
		secretController: secretController,
	}

	handler := controllers.LatestVersionHandlerFuncs(controllers.EnqueueForSelf(q))
	ingressController.ingressInformer.AddEventHandler(handler)

	stopChan := make(chan struct{})
	t.Cleanup(func() {
		time.Sleep(3 * time.Second)
		stopChan <- struct{}{}
	})

	go ingressController.ingressInformer.Run(stopChan)
	go ingressController.serviceInformer.Run(stopChan)
	go ingressController.secretController.Informer().Run(stopChan)

	go ingressController.Run(stopChan)
	go secretController.Run(stopChan)

	ingressController.RegisterEventHandler(gvk.VirtualService, func(c1, c2 config.Config, e model.Event) {})
	ingressController.RegisterEventHandler(gvk.DestinationRule, func(c1, c2 config.Config, e model.Event) {})
	ingressController.RegisterEventHandler(gvk.EnvoyFilter, func(c1, c2 config.Config, e model.Event) {})
	ingressController.RegisterEventHandler(gvk.Gateway, func(c1, c2 config.Config, e model.Event) {})

	serviceLister := ingressController.ServiceLister()
	svcObj, err := fakeClient.CoreV1().Services("default").Create(context.Background(), &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, metav1.CreateOptions{})
	require.NoError(t, err)
	err = serviceInformer.Informer().GetStore().Add(svcObj)
	require.NoError(t, err)
	services, err := serviceLister.List(labels.Everything())
	require.NoError(t, err)
	require.Equal(t, 1, len(services))

	ingress1 := &ingress.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-1",
		},
		Spec: v1beta1.IngressSpec{
			IngressClassName: &options.IngressClass,
			Rules: []v1beta1.IngressRule{
				{
					Host: "test.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/test",
								},
							},
						},
					},
				},
			},
		},
	}
	ingressObj, err := fakeClient.NetworkingV1beta1().Ingresses("default").Create(context.Background(), ingress1, metav1.CreateOptions{})
	require.NoError(t, err)
	err = ingressController.ingressInformer.GetStore().Add(ingressObj)
	require.NoError(t, err)
	ingresses := ingressController.List()
	require.Equal(t, 1, len(ingresses))

	ingress2 := &ingress.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-2",
			Namespace: "test-2",
		},
		Spec: v1beta1.IngressSpec{
			IngressClassName: &options.IngressClass,
			Rules: []v1beta1.IngressRule{
				{
					Host: "test.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/test",
								},
							},
						},
					},
				},
			},
		},
	}
	err = ingressController.ingressInformer.GetStore().Add(ingress2)
	require.NoError(t, err)
	ingresses = ingressController.List()
	require.Equal(t, 2, len(ingresses))
}

func TestShouldProcessIngressUpdate(t *testing.T) {
	c := controller{
		options: common.Options{
			IngressClass: "mse",
		},
		ingresses: make(map[string]*v1beta1.Ingress),
	}

	ingressClass := "mse"

	ingress1 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-1",
		},
		Spec: v1beta1.IngressSpec{
			IngressClassName: &ingressClass,
			Rules: []v1beta1.IngressRule{
				{
					Host: "test.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/test",
								},
							},
						},
					},
				},
			},
		},
	}

	should, _ := c.shouldProcessIngressUpdate(ingress1)
	if !should {
		t.Fatal("should be true")
	}

	ingress2 := *ingress1
	should, _ = c.shouldProcessIngressUpdate(&ingress2)
	if should {
		t.Fatal("should be false")
	}

	ingress3 := *ingress1
	ingress3.Annotations = map[string]string{
		"test": "true",
	}
	should, _ = c.shouldProcessIngressUpdate(&ingress3)
	if !should {
		t.Fatal("should be true")
	}
}

func TestCreateRuleKey(t *testing.T) {
	sep := "\n\n"
	wrapperHttpRoute := &common.WrapperHTTPRoute{
		Host:           "higress.com",
		OriginPathType: common.Prefix,
		OriginPath:     "/foo",
	}

	annots := annotations.Annotations{
		buildHigressAnnotationKey(annotations.MatchMethod):                                 "GET PUT",
		buildHigressAnnotationKey("exact-" + annotations.MatchHeader + "-abc"):             "123",
		buildHigressAnnotationKey("prefix-" + annotations.MatchHeader + "-def"):            "456",
		buildHigressAnnotationKey("exact-" + annotations.MatchPseudoHeader + "-authority"): "foo.bar.com",
		buildHigressAnnotationKey("prefix-" + annotations.MatchPseudoHeader + "-scheme"):   "htt",
		buildHigressAnnotationKey("exact-" + annotations.MatchQuery + "-region"):           "beijing",
		buildHigressAnnotationKey("prefix-" + annotations.MatchQuery + "-user-id"):         "user-",
	}
	expect := "higress.com-prefix-/foo" + sep + //host-pathType-path
		"GET PUT" + sep + // method
		"exact-:authority\tfoo.bar.com" + "\n" + "exact-abc\t123" + "\n" +
		"prefix-:scheme\thtt" + "\n" + "prefix-def\t456" + sep + // header
		"exact-region\tbeijing" + "\n" + "prefix-user-id\tuser-" + sep // params

	key := createRuleKey(annots, wrapperHttpRoute.PathFormat())
	if diff := cmp.Diff(expect, key); diff != "" {

		t.Errorf("CreateRuleKey() mismatch (-want +got):\n%s", diff)
	}
}

func buildHigressAnnotationKey(key string) string {
	return annotations.HigressAnnotationsPrefix + "/" + key
}
