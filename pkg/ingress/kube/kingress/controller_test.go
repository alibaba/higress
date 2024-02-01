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

package kingress

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/networking/pkg/apis/networking"
	"knative.dev/networking/pkg/apis/networking/v1alpha1"
	ingress "knative.dev/networking/pkg/apis/networking/v1alpha1"
	"knative.dev/pkg/kmeta"

	"github.com/alibaba/higress/pkg/ingress/kube/annotations"
	"github.com/alibaba/higress/pkg/ingress/kube/common"
	"github.com/alibaba/higress/pkg/ingress/kube/secret"
	"github.com/alibaba/higress/pkg/kube"
)

const (
	testNS                    = "testNS"
	IstioIngressClassNametest = "higress"
)

var (
	ingressRules = []v1alpha1.IngressRule{{
		Hosts: []string{
			"host-tls.example.com",
		},
		HTTP: &v1alpha1.HTTPIngressRuleValue{
			Paths: []v1alpha1.HTTPIngressPath{{
				Splits: []v1alpha1.IngressBackendSplit{{
					IngressBackend: v1alpha1.IngressBackend{
						ServiceNamespace: testNS,
						ServiceName:      "test-service",
						ServicePort:      intstr.FromInt(80),
					},
					Percent: 100,
				}},
			}},
		},
		Visibility: v1alpha1.IngressVisibilityExternalIP,
	}, {
		Hosts: []string{
			"host-tls.test-ns.svc.cluster.local",
		},
		HTTP: &v1alpha1.HTTPIngressRuleValue{
			Paths: []v1alpha1.HTTPIngressPath{{
				Splits: []v1alpha1.IngressBackendSplit{{
					IngressBackend: v1alpha1.IngressBackend{
						ServiceNamespace: testNS,
						ServiceName:      "test-service",
						ServicePort:      intstr.FromInt(80),
					},
					Percent: 100,
				}},
			}},
		},
		Visibility: v1alpha1.IngressVisibilityClusterLocal,
	}}

	ingressTLS = []v1alpha1.IngressTLS{{
		Hosts:           []string{"host-tls.example.com"},
		SecretName:      "secret0",
		SecretNamespace: "istio-system",
	}}

	// The gateway server according to ingressTLS.
	ingressTLSServer = &istiov1alpha3.Server{
		Hosts: []string{"host-tls.example.com"},
		Port: &istiov1alpha3.Port{
			Name:     "test-ns/reconciling-ingress:0",
			Number:   443,
			Protocol: "HTTPS",
		},
		Tls: &istiov1alpha3.ServerTLSSettings{
			Mode:              istiov1alpha3.ServerTLSSettings_SIMPLE,
			ServerCertificate: "tls.crt",
			PrivateKey:        "tls.key",
			CredentialName:    "secret0",
		},
	}

	ingressHTTPServer = &istiov1alpha3.Server{
		Hosts: []string{"host-tls.example.com"},
		Port: &istiov1alpha3.Port{
			Name:     "http-server",
			Number:   80,
			Protocol: "HTTP",
		},
	}

	ingressHTTPRedirectServer = &istiov1alpha3.Server{
		Hosts: []string{"*"},
		Port: &istiov1alpha3.Port{
			Name:     "http-server",
			Number:   80,
			Protocol: "HTTP",
		},
		Tls: &istiov1alpha3.ServerTLSSettings{
			HttpsRedirect: true,
		},
	}

	// The gateway server irrelevant to ingressTLS.
	irrelevantServer = &istiov1alpha3.Server{
		Hosts: []string{"host-tls.example.com", "host-tls.test-ns.svc.cluster.local"},
		Port: &istiov1alpha3.Port{
			Name:     "test:0",
			Number:   443,
			Protocol: "HTTPS",
		},
		Tls: &istiov1alpha3.ServerTLSSettings{
			Mode:              istiov1alpha3.ServerTLSSettings_SIMPLE,
			ServerCertificate: "tls.crt",
			PrivateKey:        "tls.key",
			CredentialName:    "other-secret",
		},
	}
	irrelevantServer1 = &istiov1alpha3.Server{
		Hosts: []string{"*"},
		Port: &istiov1alpha3.Port{
			Name:     "http-server",
			Number:   80,
			Protocol: "HTTP",
		},
	}

	deletionTime = metav1.NewTime(time.Unix(1e9, 0))
)

func TestKIngressControllerConventions(t *testing.T) {
	fakeClient := kube.NewFakeClient()
	localKubeClient, client := fakeClient, fakeClient

	options := common.Options{IngressClass: "mse", ClusterId: "", EnableStatus: true}

	secretController := secret.NewController(localKubeClient, options.ClusterId)
	ingressController := NewController(localKubeClient, client, options, secretController)

	testcases := map[string]func(*testing.T, common.KIngressController){
		"test convert HTTPRoute": testConvertHTTPRoute,
	}
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			tc(t, ingressController)
		})
	}
}

func testConvertHTTPRoute(t *testing.T, c common.KIngressController) {
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
			description: "valid httpRoute convention,invalid backend",
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
							Hosts: []string{
								"host-tls.example.com",
							},
							HTTP: &ingress.HTTPIngressRuleValue{
								Paths: []ingress.HTTPIngressPath{{
									Splits: []ingress.IngressBackendSplit{{
										IngressBackend: ingress.IngressBackend{},
										Percent:        100,
									}},
								}},
							},
							Visibility: ingress.IngressVisibilityExternalIP,
						},
					},
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
		}, {
			description: "valid httpRoute convention,invalid split",
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
							Hosts: []string{
								"host-tls.example.com",
							},
							HTTP: &ingress.HTTPIngressRuleValue{
								Paths: []ingress.HTTPIngressPath{{
									Splits: []ingress.IngressBackendSplit{{}},
								}},
							},
							Visibility: ingress.IngressVisibilityExternalIP,
						},
					},
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
		{
			description: "valid httpRoute convention, vaild ingress",
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
					IngressRouteCache: common.NewIngressRouteCache(),
					HTTPRoutes:        make(map[string][]*common.WrapperHTTPRoute),
				},
				wrapperConfig: &common.WrapperConfig{Config: &config.Config{
					Meta: config.Meta{
						Name:      "host-tls-test",
						Namespace: testNS,
					},
					Spec: ingress.IngressSpec{Rules: []ingress.IngressRule{
						{
							Hosts: []string{
								"host-tls.example.com",
							},
							HTTP: &ingress.HTTPIngressRuleValue{
								Paths: []ingress.HTTPIngressPath{{
									Splits: []ingress.IngressBackendSplit{{
										IngressBackend: v1alpha1.IngressBackend{
											ServiceNamespace: testNS,
											ServiceName:      "v1-service",
											ServicePort:      intstr.FromInt(80),
										},
										Percent: 100,
									}},
								}},
							},
							Visibility: ingress.IngressVisibilityExternalIP,
						},
					},
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
		}, {
			description: "valid httpRoute convention, Spec Rule All open Ingress",
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
					IngressRouteCache: common.NewIngressRouteCache(),
					HTTPRoutes:        make(map[string][]*common.WrapperHTTPRoute),
				},
				wrapperConfig: &common.WrapperConfig{Config: &config.Config{
					Meta: config.Meta{
						Name:      "host-kingress-all-open-test",
						Namespace: "default",
					},
					Spec: ingress.IngressSpec{Rules: []ingress.IngressRule{
						{
							Hosts: []string{
								"hello.default",
								"hello.default.svc",
								"hello.default.svc.cluster.local",
							},
							HTTP: &ingress.HTTPIngressRuleValue{
								Paths: []ingress.HTTPIngressPath{{
									Path: "/pet/",
									Splits: []v1alpha1.IngressBackendSplit{{
										AppendHeaders: map[string]string{
											"Knative-Serving-Namespace": "default",
											"Knative-Serving-Revision":  "hello-00002",
										},
										IngressBackend: v1alpha1.IngressBackend{
											ServiceNamespace: "default",
											ServiceName:      "hello-00002",
											ServicePort:      intstr.FromInt(80),
										},
										Percent: 90,
									}, {
										AppendHeaders: map[string]string{
											"Knative-Serving-Namespace": "default",
											"Knative-Serving-Revision":  "hello-00001",
										},
										IngressBackend: v1alpha1.IngressBackend{
											ServiceNamespace: "default",
											ServiceName:      "hello-00001",
											ServicePort:      intstr.FromInt(80),
										},
										Percent: 10,
									}},
									AppendHeaders: map[string]string{
										"ugh": "blah",
									},
								}},
							},
							Visibility: ingress.IngressVisibilityClusterLocal,
						}, {
							Hosts: []string{
								"hello.default.zwj.com",
							},
							HTTP: &ingress.HTTPIngressRuleValue{
								Paths: []ingress.HTTPIngressPath{{
									Splits: []v1alpha1.IngressBackendSplit{{
										AppendHeaders: map[string]string{
											"Knative-Serving-Namespace": "default",
											"Knative-Serving-Revision":  "hello-00002",
										},
										IngressBackend: v1alpha1.IngressBackend{
											ServiceNamespace: "default",
											ServiceName:      "hello-00002",
											ServicePort:      intstr.FromInt(80),
										},
										Percent: 90,
									}, {
										AppendHeaders: map[string]string{
											"Knative-Serving-Namespace": "default",
											"Knative-Serving-Revision":  "hello-00001",
										},
										IngressBackend: v1alpha1.IngressBackend{
											ServiceNamespace: "default",
											ServiceName:      "hello-00001",
											ServicePort:      intstr.FromInt(80),
										},
										Percent: 10,
									}},
								}},
							},
							Visibility: ingress.IngressVisibilityExternalIP,
						},
					},
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

func TestShouldProcessIngressUpdate(t *testing.T) {
	c := controller{
		options:   common.Options{},
		ingresses: make(map[string]*ingress.Ingress),
	}
	ingress1 := &ingress.Ingress{

		ObjectMeta: metav1.ObjectMeta{
			Name: "test-1",
		},
		Spec: ingress.IngressSpec{
			Rules: []ingress.IngressRule{
				{
					Hosts: []string{
						"host-tls.example.com",
					},
					HTTP: &ingress.HTTPIngressRuleValue{
						Paths: []ingress.HTTPIngressPath{{
							Splits: []ingress.IngressBackendSplit{{
								IngressBackend: ingress.IngressBackend{
									ServiceNamespace: "testNs",
									ServiceName:      "test-service",
									ServicePort:      intstr.FromInt(80),
								},
								Percent: 100,
							}},
						}},
					},
				},
			},
		},
	}
	addAnnotations(ingress1, map[string]string{networking.IngressClassAnnotationKey: IstioIngressClassNametest})

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
	ingress4 := ingress1.DeepCopy()
	addAnnotations(ingress4, map[string]string{networking.IngressClassAnnotationKey: "fake-classname"})
	should, _ = c.shouldProcessIngressUpdate(ingress4)
	if should {
		t.Fatal("should be false")
	}
	//可能有坑，annotation更新可能会引起ingress资源的反复处理。

}

func addAnnotations(ing *ingress.Ingress, annos map[string]string) *ingress.Ingress {
	// UnionMaps(a, b) where value from b wins. Use annos for second arg.
	ing.ObjectMeta.Annotations = kmeta.UnionMaps(ing.ObjectMeta.Annotations, annos)
	return ing
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
