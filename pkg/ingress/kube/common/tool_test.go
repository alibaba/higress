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

package common

import (
	"testing"

	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/higress/pkg/ingress/kube/annotations"
	"github.com/stretchr/testify/assert"
)

func TestConstructRouteName(t *testing.T) {
	testCases := []struct {
		input  *WrapperHTTPRoute
		expect string
	}{
		{
			input: &WrapperHTTPRoute{
				Host:           "test.com",
				OriginPathType: Exact,
				OriginPath:     "/test",
				HTTPRoute:      &networking.HTTPRoute{},
			},
			expect: "test.com-exact-/test",
		},
		{
			input: &WrapperHTTPRoute{
				Host:           "*.test.com",
				OriginPathType: Regex,
				OriginPath:     "/test/(.*)/?[0-9]",
				HTTPRoute:      &networking.HTTPRoute{},
			},
			expect: "*.test.com-regex-/test/(.*)/?[0-9]",
		},
		{
			input: &WrapperHTTPRoute{
				Host:           "test.com",
				OriginPathType: Exact,
				OriginPath:     "/test",
				HTTPRoute: &networking.HTTPRoute{
					Match: []*networking.HTTPMatchRequest{
						{
							Headers: map[string]*networking.StringMatch{
								"b": {
									MatchType: &networking.StringMatch_Regex{
										Regex: "a?c.*",
									},
								},
								"a": {
									MatchType: &networking.StringMatch_Exact{
										Exact: "hello",
									},
								},
							},
						},
					},
				},
			},
			expect: "test.com-exact-/test-exact-a-hello-regex-b-a?c.*",
		},
		{
			input: &WrapperHTTPRoute{
				Host:           "test.com",
				OriginPathType: Prefix,
				OriginPath:     "/test",
				HTTPRoute: &networking.HTTPRoute{
					Match: []*networking.HTTPMatchRequest{
						{
							QueryParams: map[string]*networking.StringMatch{
								"b": {
									MatchType: &networking.StringMatch_Regex{
										Regex: "a?c.*",
									},
								},
								"a": {
									MatchType: &networking.StringMatch_Exact{
										Exact: "hello",
									},
								},
							},
						},
					},
				},
			},
			expect: "test.com-prefix-/test-exact:a:hello-regex:b:a?c.*",
		},
		{
			input: &WrapperHTTPRoute{
				Host:           "test.com",
				OriginPathType: Prefix,
				OriginPath:     "/test",
				HTTPRoute: &networking.HTTPRoute{
					Match: []*networking.HTTPMatchRequest{
						{
							Headers: map[string]*networking.StringMatch{
								"f": {
									MatchType: &networking.StringMatch_Regex{
										Regex: "abc?",
									},
								},
								"e": {
									MatchType: &networking.StringMatch_Exact{
										Exact: "bye",
									},
								},
							},
							QueryParams: map[string]*networking.StringMatch{
								"b": {
									MatchType: &networking.StringMatch_Regex{
										Regex: "a?c.*",
									},
								},
								"a": {
									MatchType: &networking.StringMatch_Exact{
										Exact: "hello",
									},
								},
							},
						},
					},
				},
			},
			expect: "test.com-prefix-/test-exact-e-bye-regex-f-abc?-exact:a:hello-regex:b:a?c.*",
		},
	}

	for _, c := range testCases {
		t.Run("", func(t *testing.T) {
			out := constructRouteName(c.input)
			if out != c.expect {
				t.Fatalf("Expect %s, but is %s", c.expect, out)
			}
		})
	}
}

func TestGenerateUniqueRouteName(t *testing.T) {
	input := &WrapperHTTPRoute{
		WrapperConfig: &WrapperConfig{
			Config: &config.Config{
				Meta: config.Meta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			AnnotationsConfig: &annotations.Ingress{},
		},
		Host:           "test.com",
		OriginPathType: Prefix,
		OriginPath:     "/test",
		ClusterId:      "cluster1",
		HTTPRoute: &networking.HTTPRoute{
			Match: []*networking.HTTPMatchRequest{
				{
					Headers: map[string]*networking.StringMatch{
						"f": {
							MatchType: &networking.StringMatch_Regex{
								Regex: "abc?",
							},
						},
						"e": {
							MatchType: &networking.StringMatch_Exact{
								Exact: "bye",
							},
						},
					},
					QueryParams: map[string]*networking.StringMatch{
						"b": {
							MatchType: &networking.StringMatch_Regex{
								Regex: "a?c.*",
							},
						},
						"a": {
							MatchType: &networking.StringMatch_Exact{
								Exact: "hello",
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t, "bar/foo", GenerateUniqueRouteName("xxx", input))
	assert.Equal(t, "foo", GenerateUniqueRouteName("bar", input))

}

func TestGetLbStatusList(t *testing.T) {
	clusterPrefix = "gw-123-"
	svcName := clusterPrefix
	svcList := []*v1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: svcName,
			},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeLoadBalancer,
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{
							IP: "2.2.2.2",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: svcName,
			},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeLoadBalancer,
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{
							Hostname: "1.1.1.1" + SvcHostNameSuffix,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: svcName,
			},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeLoadBalancer,
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{
							Hostname: "4.4.4.4" + SvcHostNameSuffix,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: svcName,
			},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeLoadBalancer,
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{
							IP: "3.3.3.3",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: svcName,
			},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeClusterIP,
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{
							IP: "5.5.5.5",
						},
					},
				},
			},
		},
	}

	lbiList := GetLbStatusList(svcList)
	if len(lbiList) != 4 {
		t.Fatal("len should be 4")
	}

	if lbiList[0].IP != "1.1.1.1" {
		t.Fatal("should be 1.1.1.1")
	}

	if lbiList[3].IP != "4.4.4.4" {
		t.Fatal("should be 4.4.4.4")
	}
}

func TestSortRoutes(t *testing.T) {
	input := []*WrapperHTTPRoute{
		{
			WrapperConfig: &WrapperConfig{
				Config: &config.Config{
					Meta: config.Meta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				AnnotationsConfig: &annotations.Ingress{},
			},
			Host:           "test.com",
			OriginPathType: Prefix,
			OriginPath:     "/",
			ClusterId:      "cluster1",
			HTTPRoute: &networking.HTTPRoute{
				Name: "test-1",
			},
		},
		{
			WrapperConfig: &WrapperConfig{
				Config: &config.Config{
					Meta: config.Meta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				AnnotationsConfig: &annotations.Ingress{},
			},
			Host:           "test.com",
			OriginPathType: Prefix,
			OriginPath:     "/a",
			ClusterId:      "cluster1",
			HTTPRoute: &networking.HTTPRoute{
				Name: "test-2",
			},
		},
		{
			WrapperConfig: &WrapperConfig{
				Config: &config.Config{
					Meta: config.Meta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				AnnotationsConfig: &annotations.Ingress{},
			},
			HTTPRoute: &networking.HTTPRoute{
				Name: "test-3",
			},
			IsDefaultBackend: true,
		},
		{
			WrapperConfig: &WrapperConfig{
				Config: &config.Config{
					Meta: config.Meta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				AnnotationsConfig: &annotations.Ingress{},
			},
			Host:           "test.com",
			OriginPathType: Exact,
			OriginPath:     "/b",
			ClusterId:      "cluster1",
			HTTPRoute: &networking.HTTPRoute{
				Name: "test-4",
			},
		},
		{
			WrapperConfig: &WrapperConfig{
				Config: &config.Config{
					Meta: config.Meta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				AnnotationsConfig: &annotations.Ingress{},
			},
			Host:           "test.com",
			OriginPathType: Regex,
			OriginPath:     "/d(.*)",
			ClusterId:      "cluster1",
			HTTPRoute: &networking.HTTPRoute{
				Name: "test-5",
			},
		},
	}

	SortHTTPRoutes(input)
	if (input[0].HTTPRoute.Name) != "test-4" {
		t.Fatal("should be test-4")
	}
	if (input[1].HTTPRoute.Name) != "test-2" {
		t.Fatal("should be test-2")
	}
	if (input[2].HTTPRoute.Name) != "test-5" {
		t.Fatal("should be test-5")
	}
	if (input[3].HTTPRoute.Name) != "test-1" {
		t.Fatal("should be test-1")
	}
	if (input[4].HTTPRoute.Name) != "test-3" {
		t.Fatal("should be test-3")
	}
}
