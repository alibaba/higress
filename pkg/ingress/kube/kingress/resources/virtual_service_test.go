/*
Copyright 2019 The Knative Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/networking/pkg/apis/networking/v1alpha1"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

var (
	defaultIngressRuleValue = &v1alpha1.HTTPIngressRuleValue{
		Paths: []v1alpha1.HTTPIngressPath{{
			Splits: []v1alpha1.IngressBackendSplit{{
				Percent: 100,
				IngressBackend: v1alpha1.IngressBackend{
					ServiceNamespace: "test",
					ServiceName:      "test.svc.cluster.local",
					ServicePort:      intstr.FromInt(8080),
				},
			}},
		}},
	}
	defaultIngress = v1alpha1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: system.Namespace(),
		},
		Spec: v1alpha1.IngressSpec{Rules: []v1alpha1.IngressRule{{
			Hosts: []string{
				"test-route.test-ns.svc.cluster.local",
			},
			HTTP: defaultIngressRuleValue,
		}}},
	}
	defaultVSCmpOpts = protocmp.Transform()
)

func TestMakeVirtualServiceRoute_RewriteHost(t *testing.T) {
	ingressPath := &v1alpha1.HTTPIngressPath{
		RewriteHost: "the.target.host",
		Splits: []v1alpha1.IngressBackendSplit{{
			Percent: 100,
			IngressBackend: v1alpha1.IngressBackend{
				ServiceName:      "the-svc",
				ServiceNamespace: "the-ns",
				ServicePort:      intstr.FromInt(8080),
			},
		}},
	}
	route := MakeVirtualServiceRoute(sets.NewString("a.vanity.url", "another.vanity.url"), ingressPath)
	expected := &istiov1alpha3.HTTPRoute{
		Retries: &istiov1alpha3.HTTPRetry{},
		Match: []*istiov1alpha3.HTTPMatchRequest{{
			Authority: &istiov1alpha3.StringMatch{
				MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: `a.vanity.url`},
			},
		}, {
			Authority: &istiov1alpha3.StringMatch{
				MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: `another.vanity.url`},
			},
		}},
		Rewrite: &istiov1alpha3.HTTPRewrite{
			Authority: "the.target.host",
		},
		Route: []*istiov1alpha3.HTTPRouteDestination{{
			Destination: &istiov1alpha3.Destination{
				Host: "the-svc.the-ns.svc.cluster.local",
				Port: &istiov1alpha3.PortSelector{
					Number: 8080,
				},
			},
			Weight: 100,
		}},
	}
	if diff := cmp.Diff(expected, route, defaultVSCmpOpts); diff != "" {
		t.Error("Unexpected route  (-want +got):", diff)
	}
}

// One active target.
func TestMakeVirtualServiceRoute_Vanilla(t *testing.T) {
	ingressPath := &v1alpha1.HTTPIngressPath{
		Headers: map[string]v1alpha1.HeaderMatch{
			"my-header": {
				Exact: "my-header-value",
			},
		},
		Splits: []v1alpha1.IngressBackendSplit{{
			IngressBackend: v1alpha1.IngressBackend{
				ServiceNamespace: "test-ns",
				ServiceName:      "revision-service",
				ServicePort:      intstr.FromInt(80),
			},
			Percent: 100,
		}},
	}
	route := MakeVirtualServiceRoute(sets.NewString("a.com", "b.org"), ingressPath)
	expected := &istiov1alpha3.HTTPRoute{
		Retries: &istiov1alpha3.HTTPRetry{},
		Match: []*istiov1alpha3.HTTPMatchRequest{{
			Authority: &istiov1alpha3.StringMatch{
				MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: `a.com`},
			},
			Headers: map[string]*istiov1alpha3.StringMatch{
				"my-header": {
					MatchType: &istiov1alpha3.StringMatch_Exact{
						Exact: "my-header-value",
					},
				},
			},
		}, {
			Authority: &istiov1alpha3.StringMatch{
				MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: `b.org`},
			},
			Headers: map[string]*istiov1alpha3.StringMatch{
				"my-header": {
					MatchType: &istiov1alpha3.StringMatch_Exact{
						Exact: "my-header-value",
					},
				},
			},
		}},
		Route: []*istiov1alpha3.HTTPRouteDestination{{
			Destination: &istiov1alpha3.Destination{
				Host: "revision-service.test-ns.svc.cluster.local",
				Port: &istiov1alpha3.PortSelector{Number: 80},
			},
			Weight: 100,
		}},
	}
	if diff := cmp.Diff(expected, route, defaultVSCmpOpts); diff != "" {
		t.Error("Unexpected route  (-want +got):", diff)
	}
}

// One active target.
func TestMakeVirtualServiceRoute_Internal(t *testing.T) {
	ingressPath := &v1alpha1.HTTPIngressPath{
		Splits: []v1alpha1.IngressBackendSplit{{
			IngressBackend: v1alpha1.IngressBackend{
				ServiceNamespace: "test-ns",
				ServiceName:      "revision-service",
				ServicePort:      intstr.FromInt(80),
			},
			Percent: 100,
		}},
	}
	route := MakeVirtualServiceRoute(sets.NewString("a.default"), ingressPath)
	expected := &istiov1alpha3.HTTPRoute{
		Retries: &istiov1alpha3.HTTPRetry{},
		Match: []*istiov1alpha3.HTTPMatchRequest{{
			Authority: &istiov1alpha3.StringMatch{
				MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: `a.default`},
			},
		}},
		Route: []*istiov1alpha3.HTTPRouteDestination{{
			Destination: &istiov1alpha3.Destination{
				Host: "revision-service.test-ns.svc.cluster.local",
				Port: &istiov1alpha3.PortSelector{Number: 80},
			},
			Weight: 100,
		}},
	}
	if diff := cmp.Diff(expected, route, defaultVSCmpOpts); diff != "" {
		t.Error("Unexpected route  (-want +got):", diff)
	}
}

// Two active targets.
func TestMakeVirtualServiceRoute_TwoTargets(t *testing.T) {
	ingressPath := &v1alpha1.HTTPIngressPath{
		Splits: []v1alpha1.IngressBackendSplit{{
			IngressBackend: v1alpha1.IngressBackend{
				ServiceNamespace: "test-ns",
				ServiceName:      "revision-service",
				ServicePort:      intstr.FromInt(80),
			},
			Percent: 90,
		}, {
			IngressBackend: v1alpha1.IngressBackend{
				ServiceNamespace: "test-ns",
				ServiceName:      "new-revision-service",
				ServicePort:      intstr.FromInt(81),
			},
			Percent: 10,
		}},
	}
	route := MakeVirtualServiceRoute(sets.NewString("test.org"), ingressPath)
	expected := &istiov1alpha3.HTTPRoute{
		Retries: &istiov1alpha3.HTTPRetry{},
		Match: []*istiov1alpha3.HTTPMatchRequest{{
			Authority: &istiov1alpha3.StringMatch{
				MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: `test.org`},
			},
		}},
		Route: []*istiov1alpha3.HTTPRouteDestination{{
			Destination: &istiov1alpha3.Destination{
				Host: "revision-service.test-ns.svc.cluster.local",
				Port: &istiov1alpha3.PortSelector{Number: 80},
			},
			Weight: 90,
		}, {
			Destination: &istiov1alpha3.Destination{
				Host: "new-revision-service.test-ns.svc.cluster.local",
				Port: &istiov1alpha3.PortSelector{Number: 81},
			},
			Weight: 10,
		}},
	}
	if diff := cmp.Diff(expected, route, defaultVSCmpOpts); diff != "" {
		t.Error("Unexpected route  (-want +got):", diff)
	}
}

func TestGetDistinctHostPrefixes(t *testing.T) {
	cases := []struct {
		name string
		in   sets.String
		out  sets.String
	}{
		{"empty", sets.NewString(), sets.NewString()},
		{"single element", sets.NewString("a"), sets.NewString("a")},
		{"no overlap", sets.NewString("a", "b"), sets.NewString("a", "b")},
		{"overlap", sets.NewString("a", "ab", "abc"), sets.NewString("a")},
		{"multiple overlaps", sets.NewString("a", "ab", "abc", "xyz", "xy", "m"), sets.NewString("a", "xy", "m")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := getDistinctHostPrefixes(tt.in)
			if !tt.out.Equal(got) {
				t.Fatalf("Expected %v, got %v", tt.out, got)
			}
		})
	}
}
