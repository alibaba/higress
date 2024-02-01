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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	networking "istio.io/api/networking/v1alpha3"
)

func TestMatch_ParseMethods(t *testing.T) {
	parser := match{}
	testCases := []struct {
		input  Annotations
		expect *MatchConfig
	}{
		{
			input:  Annotations{},
			expect: &MatchConfig{},
		},
		{
			input: Annotations{
				buildHigressAnnotationKey(MatchMethod): "PUT POST PATCH",
			},
			expect: &MatchConfig{
				Methods: []string{"PUT", "POST", "PATCH"},
			},
		},
		{
			input: Annotations{
				buildHigressAnnotationKey(MatchMethod): "PUT PUT",
			},
			expect: &MatchConfig{
				Methods: []string{"PUT"},
			},
		},
		{
			input: Annotations{
				buildHigressAnnotationKey(MatchMethod): "put post patch",
			},
			expect: &MatchConfig{
				Methods: []string{"PUT", "POST", "PATCH"},
			},
		},
		{
			input: Annotations{
				buildHigressAnnotationKey(MatchMethod): "geet",
			},
			expect: &MatchConfig{},
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{}
			_ = parser.Parse(tt.input, config, nil)
			if diff := cmp.Diff(tt.expect, config.Match); diff != "" {
				t.Fatalf("TestMatch_Parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMatch_ParseHeaders(t *testing.T) {
	parser := match{}
	testCases := []struct {
		typ    string
		key    string
		value  string
		expect map[string]map[string]string
	}{
		{
			typ:   "exact",
			key:   "abc",
			value: "123",
			expect: map[string]map[string]string{
				exact: {
					"abc": "123",
				},
			},
		},
		{
			typ:   "prefix",
			key:   "user-id",
			value: "10086-1",
			expect: map[string]map[string]string{
				prefix: {
					"user-id": "10086-1",
				},
			},
		},
		{
			typ:   "regex",
			key:   "content-type",
			value: "application/(json|xml)",
			expect: map[string]map[string]string{
				regex: {
					"content-type": "application/(json|xml)",
				},
			},
		},
		{
			typ:   "exact",
			key:   ":method",
			value: "GET",
			expect: map[string]map[string]string{
				exact: {
					":method": "GET",
				},
			},
		},
		{
			typ:   "prefix",
			key:   ":path",
			value: "/foo",
			expect: map[string]map[string]string{
				prefix: {
					":path": "/foo",
				},
			},
		},
		{
			typ:   "regex",
			key:   ":authority",
			value: "test\\d+\\.com",
			expect: map[string]map[string]string{
				regex: {
					":authority": "test\\d+\\.com",
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			matchKeyword := MatchHeader
			headerKey := tt.key
			if strings.HasPrefix(headerKey, ":") {
				headerKey = strings.TrimPrefix(headerKey, ":")
				matchKeyword = MatchPseudoHeader
			}
			key := buildHigressAnnotationKey(tt.typ + "-" + matchKeyword + "-" + headerKey)
			input := Annotations{key: tt.value}
			config := &Ingress{}
			_ = parser.Parse(input, config, nil)
			if diff := cmp.Diff(tt.expect, config.Match.Headers); diff != "" {
				t.Fatalf("TestMatch_ParseHeaders() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMatch_ParseQueryParams(t *testing.T) {
	parser := match{}
	testCases := []struct {
		typ    string
		key    string
		value  string
		expect map[string]map[string]string
	}{
		{
			typ:   "exact",
			key:   "abc",
			value: "123",
			expect: map[string]map[string]string{
				exact: {
					"abc": "123",
				},
			},
		},
		{
			typ:   "prefix",
			key:   "age",
			value: "2",
			expect: map[string]map[string]string{
				prefix: {
					"age": "2",
				},
			},
		},
		{
			typ:   "regex",
			key:   "name",
			value: "B.*",
			expect: map[string]map[string]string{
				regex: {
					"name": "B.*",
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			key := buildHigressAnnotationKey(tt.typ + "-" + MatchQuery + "-" + tt.key)
			input := Annotations{key: tt.value}
			config := &Ingress{}
			_ = parser.Parse(input, config, nil)
			if diff := cmp.Diff(tt.expect, config.Match.QueryParams); diff != "" {
				t.Fatalf("TestMatch_ParseQueryParams() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMatch_ApplyRoute(t *testing.T) {
	handler := match{}
	testCases := []struct {
		input  *networking.HTTPRoute
		config *Ingress
		expect *networking.HTTPRoute
	}{
		// methods
		{
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
			config: &Ingress{
				Match: &MatchConfig{
					Methods: []string{"PUT", "GET", "POST"},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Method: &networking.StringMatch{
							MatchType: &networking.StringMatch_Regex{Regex: "PUT|GET|POST"},
						},
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
		},
		// headers
		{
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
			config: &Ingress{
				Match: &MatchConfig{
					Headers: map[string]map[string]string{
						exact: {"new": "new"},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Headers: map[string]*networking.StringMatch{
							"new": {
								MatchType: &networking.StringMatch_Exact{Exact: "new"},
							},
						},
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
		},
		{
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Headers: map[string]*networking.StringMatch{
							"origin": {
								MatchType: &networking.StringMatch_Exact{Exact: "origin"},
							},
						},
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
			config: &Ingress{
				Match: &MatchConfig{
					Headers: map[string]map[string]string{
						exact: {"new": "new"},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Headers: map[string]*networking.StringMatch{
							"origin": {
								MatchType: &networking.StringMatch_Exact{Exact: "origin"},
							},
							"new": {
								MatchType: &networking.StringMatch_Exact{Exact: "new"},
							},
						},
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
		},
		// queryParams
		{
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
			config: &Ingress{
				Match: &MatchConfig{
					QueryParams: map[string]map[string]string{
						exact: {"new": "new"},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						QueryParams: map[string]*networking.StringMatch{
							"new": {
								MatchType: &networking.StringMatch_Exact{Exact: "new"},
							},
						},
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
		},
		{
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						QueryParams: map[string]*networking.StringMatch{
							"origin": {
								MatchType: &networking.StringMatch_Exact{Exact: "origin"},
							},
						},
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
			config: &Ingress{
				Match: &MatchConfig{
					QueryParams: map[string]map[string]string{
						exact: {"new": "new"},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						QueryParams: map[string]*networking.StringMatch{
							"origin": {
								MatchType: &networking.StringMatch_Exact{Exact: "origin"},
							},
							"new": {
								MatchType: &networking.StringMatch_Exact{Exact: "new"},
							},
						},
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			handler.ApplyRoute(tt.input, tt.config)
			if diff := cmp.Diff(tt.expect, tt.input); diff != "" {
				t.Fatalf("TestMatch_Parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
