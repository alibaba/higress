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
	"istio.io/istio/pkg/kube"
	kubeVersion "k8s.io/apimachinery/pkg/version"
	"testing"
	"time"

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
				OriginPathType: PrefixRegex,
				OriginPath:     "/test/(.*)/?[0-9]",
				HTTPRoute:      &networking.HTTPRoute{},
			},
			expect: "*.test.com-prefixRegex-/test/(.*)/?[0-9]",
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
			OriginPathType: PrefixRegex,
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

// TestSortHTTPRoutesWithMoreRules include headers, query params, methods
func TestSortHTTPRoutesWithMoreRules(t *testing.T) {
	input := []struct {
		order      string
		pathType   PathType
		path       string
		method     *networking.StringMatch
		header     map[string]*networking.StringMatch
		queryParam map[string]*networking.StringMatch
	}{
		{
			order:    "1",
			pathType: Exact,
			path:     "/bar",
		},
		{
			order:    "2",
			pathType: Prefix,
			path:     "/bar",
		},
		{
			order:    "3",
			pathType: Prefix,
			path:     "/bar",
			method: &networking.StringMatch{
				MatchType: &networking.StringMatch_Regex{Regex: "GET|PUT"},
			},
		},
		{
			order:    "4",
			pathType: Prefix,
			path:     "/bar",
			method: &networking.StringMatch{
				MatchType: &networking.StringMatch_Regex{Regex: "GET"},
			},
		},
		{
			order:    "5",
			pathType: Prefix,
			path:     "/bar",
			header: map[string]*networking.StringMatch{
				"foo": {
					MatchType: &networking.StringMatch_Exact{Exact: "bar"},
				},
			},
		},
		{
			order:    "6",
			pathType: Prefix,
			path:     "/bar",
			header: map[string]*networking.StringMatch{
				"foo": {
					MatchType: &networking.StringMatch_Exact{Exact: "bar"},
				},
				"bar": {
					MatchType: &networking.StringMatch_Exact{Exact: "foo"},
				},
			},
		},
		{
			order:    "7",
			pathType: Prefix,
			path:     "/bar",
			queryParam: map[string]*networking.StringMatch{
				"foo": {
					MatchType: &networking.StringMatch_Exact{Exact: "bar"},
				},
			},
		},
		{
			order:    "8",
			pathType: Prefix,
			path:     "/bar",
			queryParam: map[string]*networking.StringMatch{
				"foo": {
					MatchType: &networking.StringMatch_Exact{Exact: "bar"},
				},
				"bar": {
					MatchType: &networking.StringMatch_Exact{Exact: "foo"},
				},
			},
		},
		{
			order:    "9",
			pathType: Prefix,
			path:     "/bar",
			method: &networking.StringMatch{
				MatchType: &networking.StringMatch_Regex{Regex: "GET"},
			},
			queryParam: map[string]*networking.StringMatch{
				"foo": {
					MatchType: &networking.StringMatch_Exact{Exact: "bar"},
				},
			},
		},
		{
			order:    "10",
			pathType: Prefix,
			path:     "/bar",
			method: &networking.StringMatch{
				MatchType: &networking.StringMatch_Regex{Regex: "GET"},
			},
			queryParam: map[string]*networking.StringMatch{
				"bar": {
					MatchType: &networking.StringMatch_Exact{Exact: "foo"},
				},
			},
		},
	}

	origin := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	expect := []string{"1", "9", "10", "4", "3", "6", "5", "8", "7", "2"}

	var list []*WrapperHTTPRoute
	for idx, val := range input {
		list = append(list, &WrapperHTTPRoute{
			OriginPath:     val.path,
			OriginPathType: val.pathType,
			HTTPRoute: &networking.HTTPRoute{
				Name: origin[idx],
				Match: []*networking.HTTPMatchRequest{
					{
						Method:      val.method,
						Headers:     val.header,
						QueryParams: val.queryParam,
					},
				},
			},
		})
	}

	SortHTTPRoutes(list)

	for idx, val := range list {
		if val.HTTPRoute.Name != expect[idx] {
			t.Fatalf("should be %s, but got %s", expect[idx], val.HTTPRoute.Name)
		}
	}
}

type supportV1Client struct {
	kube.Client
}

func (s *supportV1Client) GetKubernetesVersion() (*kubeVersion.Info, error) {
	return &kubeVersion.Info{
		GitVersion: "v1.28.3-aliyun.1",
	}, nil
}

type unSupportV1Client struct {
	kube.Client
}

func (u *unSupportV1Client) GetKubernetesVersion() (*kubeVersion.Info, error) {
	return &kubeVersion.Info{
		GitVersion: "v1.18.0",
	}, nil
}

type supportNetworkingClient struct {
	kube.Client
}

func (s *supportNetworkingClient) GetKubernetesVersion() (*kubeVersion.Info, error) {
	return &kubeVersion.Info{
		GitVersion: "v1.18.0-aliyun.1",
	}, nil
}

type unSupportNetworkingClient struct {
	kube.Client
}

func (u *unSupportNetworkingClient) GetKubernetesVersion() (*kubeVersion.Info, error) {
	return &kubeVersion.Info{
		GitVersion: "v1.17.0-aliyun.1",
	}, nil
}

type errorClient struct {
	kube.Client
}

func (e *errorClient) GetKubernetesVersion() (*kubeVersion.Info, error) {
	return &kubeVersion.Info{
		GitVersion: "error",
	}, nil
}

func TestV1Available(t *testing.T) {
	fakeClient := kube.NewFakeClient()

	v1Client := &supportV1Client{
		fakeClient,
	}

	if !V1Available(v1Client) {
		t.Fatal("should support v1")
	}

	v1Beta1Client := &unSupportV1Client{
		fakeClient,
	}
	if V1Available(v1Beta1Client) {
		t.Fatal("should not support v1")
	}

	errorClient := &errorClient{
		fakeClient,
	}
	// will fallback to v1
	if !V1Available(errorClient, WithInterval(1*time.Second), WithTimeout(3*time.Second)) {
		t.Fatal("should fallback to v1")
	}
}

func TestNetworkingIngressAvailable(t *testing.T) {
	fakeClient := kube.NewFakeClient()

	networkingClient := &supportNetworkingClient{
		fakeClient,
	}

	if !NetworkingIngressAvailable(networkingClient) {
		t.Fatal("should support networking")
	}

	notNetworkingClient := &unSupportNetworkingClient{
		fakeClient,
	}
	if NetworkingIngressAvailable(notNetworkingClient) {
		t.Fatal("should not support networking")
	}

	errorClient := &errorClient{
		fakeClient,
	}
	// will fallback to networking
	if !NetworkingIngressAvailable(errorClient, WithInterval(1*time.Second), WithTimeout(3*time.Second)) {
		t.Fatal("should fallback to networking")
	}
}
