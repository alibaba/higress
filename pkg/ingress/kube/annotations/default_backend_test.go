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
	"context"
	"reflect"
	"testing"
	"time"

	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	normalService = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "test",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name: "http",
				Port: 80,
			}},
		},
	}

	abnormalService = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "foo",
		},
	}
)

func TestFallbackParse(t *testing.T) {
	fallback := fallback{}
	inputCases := []struct {
		input  map[string]string
		expect *FallbackConfig
	}{
		{},
		{
			input: map[string]string{
				buildNginxAnnotationKey(annDefaultBackend): "test/app",
			},
			expect: &FallbackConfig{
				DefaultBackend: model.NamespacedName{
					Namespace: "test",
					Name:      "app",
				},
				Port: 80,
			},
		},
		{
			input: map[string]string{
				buildHigressAnnotationKey(annDefaultBackend): "app",
			},
			expect: &FallbackConfig{
				DefaultBackend: model.NamespacedName{
					Namespace: "test",
					Name:      "app",
				},
				Port: 80,
			},
		},
		{
			input: map[string]string{
				buildHigressAnnotationKey(annDefaultBackend): "foo/app",
			},
		},
		{
			input: map[string]string{
				buildHigressAnnotationKey(annDefaultBackend): "test/app",
				buildNginxAnnotationKey(customHTTPError):     "404,503",
			},
			expect: &FallbackConfig{
				DefaultBackend: model.NamespacedName{
					Namespace: "test",
					Name:      "app",
				},
				Port:             80,
				customHTTPErrors: []uint32{404, 503},
			},
		},
		{
			input: map[string]string{
				buildHigressAnnotationKey(annDefaultBackend): "test/app",
				buildNginxAnnotationKey(customHTTPError):     "404,5ac",
			},
			expect: &FallbackConfig{
				DefaultBackend: model.NamespacedName{
					Namespace: "test",
					Name:      "app",
				},
				Port:             80,
				customHTTPErrors: []uint32{404},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{
				Meta: Meta{
					Namespace: "test",
					ClusterId: "cluster",
				},
			}
			globalContext, cancel := initGlobalContextForService()
			defer cancel()

			_ = fallback.Parse(inputCase.input, config, globalContext)
			if !reflect.DeepEqual(inputCase.expect, config.Fallback) {
				t.Fatal("Should be equal")
			}
		})
	}
}

func TestFallbackApplyRoute(t *testing.T) {
	fallback := fallback{}
	inputCases := []struct {
		config *Ingress
		input  *networking.HTTPRoute
		expect *networking.HTTPRoute
	}{
		{
			config: &Ingress{},
			input:  &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{},
		},
		{
			config: &Ingress{
				Fallback: &FallbackConfig{
					DefaultBackend: model.NamespacedName{
						Namespace: "test",
						Name:      "app",
					},
					Port:             80,
					customHTTPErrors: []uint32{404, 503},
				},
			},
			input: &networking.HTTPRoute{
				Name: "route",
				Route: []*networking.HTTPRouteDestination{
					{},
				},
			},
			expect: &networking.HTTPRoute{
				Name: "route",
				InternalActiveRedirect: &networking.HTTPInternalActiveRedirect{
					MaxInternalRedirects:  1,
					RedirectResponseCodes: []uint32{404, 503},
					AllowCrossScheme:      true,
					Headers: &networking.Headers{
						Request: &networking.Headers_HeaderOperations{
							Add: map[string]string{
								FallbackInjectHeaderRouteName: "route" + FallbackRouteNameSuffix,
								FallbackInjectFallbackService: "test/app",
							},
						},
					},
					RedirectUrlRewriteSpecifier: &networking.HTTPInternalActiveRedirect_RedirectUrl{
						RedirectUrl: defaultRedirectUrl,
					},
					ForcedUseOriginalHost:             true,
					ForcedAddHeaderBeforeRouteMatcher: true,
				},
				Route: []*networking.HTTPRouteDestination{
					{
						FallbackClusters: []*networking.Destination{
							{
								Host: "app.test.svc.cluster.local",
								Port: &networking.PortSelector{
									Number: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			fallback.ApplyRoute(inputCase.input, inputCase.config)
			if !reflect.DeepEqual(inputCase.input, inputCase.expect) {
				t.Fatal("Should be equal")
			}
		})
	}
}

func initGlobalContextForService() (*GlobalContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	client := fake.NewSimpleClientset(normalService, abnormalService)
	informerFactory := informers.NewSharedInformerFactory(client, time.Hour)
	serviceInformer := informerFactory.Core().V1().Services()
	go serviceInformer.Informer().Run(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), serviceInformer.Informer().HasSynced)

	return &GlobalContext{
		ClusterServiceList: map[string]listerv1.ServiceLister{
			"cluster": serviceInformer.Lister(),
		},
	}, cancel
}
