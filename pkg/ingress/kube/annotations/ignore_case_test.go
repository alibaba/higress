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
	"testing"

	"github.com/google/go-cmp/cmp"

	networking "istio.io/api/networking/v1alpha3"
)

func TestIgnoreCaseMatching_ApplyRoute(t *testing.T) {
	handler := ignoreCaseMatching{}

	testCases := []struct {
		input  *networking.HTTPRoute
		config *Ingress
		expect *networking.HTTPRoute
	}{
		{
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
						IgnoreUriCase: false,
					},
				},
			},
			config: &Ingress{
				IgnoreCase: &IgnoreCaseConfig{IgnoreUriCase: true},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
						IgnoreUriCase: true,
					},
				},
			},
		},
		{
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
						IgnoreUriCase: false,
					},
				},
			},
			config: &Ingress{
				IgnoreCase: &IgnoreCaseConfig{IgnoreUriCase: false},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{Exact: "/abc"},
						},
						IgnoreUriCase: false,
					},
				},
			},
		},
	}

	t.Run("", func(t *testing.T) {
		for _, tt := range testCases {
			handler.ApplyRoute(tt.input, tt.config)

			if diff := cmp.Diff(tt.expect, tt.input); diff != "" {
				t.Fatalf("TestIgnoreCaseMatching_ApplyRoute() mismatch(-want +got): \n%s", diff)
			}
		}
	})
}

func TestIgnoreCaseMatching_Parse(t *testing.T) {
	parser := ignoreCaseMatching{}

	testCases := []struct {
		input  Annotations
		expect *IgnoreCaseConfig
	}{
		{
			input:  Annotations{},
			expect: nil,
		},
		{
			input: Annotations{
				buildHigressAnnotationKey(enableIgnoreCase): "true",
			},
			expect: &IgnoreCaseConfig{
				IgnoreUriCase: true,
			},
		},
		{
			input: Annotations{
				buildHigressAnnotationKey(enableIgnoreCase): "false",
			},
			expect: &IgnoreCaseConfig{
				IgnoreUriCase: false,
			},
		},
	}

	t.Run("", func(t *testing.T) {
		for _, tt := range testCases {
			config := &Ingress{}

			_ = parser.Parse(tt.input, config, nil)
			if diff := cmp.Diff(tt.expect, config.IgnoreCase); diff != "" {
				t.Fatalf("TestIgnoreCaseMatching_Parse() mismatch(-want +got): \n%s", diff)
			}
		}
	})
}
