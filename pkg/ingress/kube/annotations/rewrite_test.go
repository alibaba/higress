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
	"reflect"
	"testing"

	networking "istio.io/api/networking/v1alpha3"
)

func TestConvertToRE2(t *testing.T) {
	useCases := []struct {
		input  string
		except string
	}{
		{
			input:  "/test",
			except: "/test",
		},
		{
			input:  "/test/app",
			except: "/test/app",
		},
		{
			input:  "/$1",
			except: "/\\1",
		},
		{
			input:  "/$2/$1",
			except: "/\\2/\\1",
		},
		{
			input:  "/$test/$a",
			except: "/$test/$a",
		},
	}

	for _, c := range useCases {
		t.Run("", func(t *testing.T) {
			if convertToRE2(c.input) != c.except {
				t.Fatalf("input %s is not equal to except %s.", c.input, c.except)
			}
		})
	}
}

func TestRewriteParse(t *testing.T) {
	rewrite := rewrite{}
	testCases := []struct {
		input  Annotations
		expect *RewriteConfig
	}{
		{
			input:  nil,
			expect: nil,
		},
		{
			input:  Annotations{},
			expect: nil,
		},
		{
			input: Annotations{
				buildNginxAnnotationKey(rewriteTarget): "/test",
			},
			expect: &RewriteConfig{
				RewriteTarget: "/test",
			},
		},
		{
			input: Annotations{
				buildNginxAnnotationKey(rewriteTarget): "",
			},
			expect: &RewriteConfig{},
		},
		{
			input: Annotations{
				buildNginxAnnotationKey(rewriteTarget): "/$2/$1",
			},
			expect: &RewriteConfig{
				RewriteTarget: "/\\2/\\1",
			},
		},
		{
			input: Annotations{
				buildNginxAnnotationKey(useRegex): "true",
			},
			expect: &RewriteConfig{
				UseRegex: true,
			},
		},
		{
			input: Annotations{
				buildNginxAnnotationKey(upstreamVhost): "test.com",
			},
			expect: &RewriteConfig{
				RewriteHost: "test.com",
			},
		},
		{
			input: Annotations{
				buildNginxAnnotationKey(useRegex):      "true",
				buildNginxAnnotationKey(rewriteTarget): "/$1",
			},
			expect: &RewriteConfig{
				UseRegex:      true,
				RewriteTarget: "/\\1",
			},
		},
		{
			input: Annotations{
				buildNginxAnnotationKey(rewriteTarget): "/$2/$1",
				buildNginxAnnotationKey(upstreamVhost): "test.com",
			},
			expect: &RewriteConfig{
				RewriteTarget: "/\\2/\\1",
				RewriteHost:   "test.com",
			},
		},
		{
			input: Annotations{
				buildHigressAnnotationKey(rewritePath): "/test",
			},
			expect: &RewriteConfig{
				RewritePath: "/test",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{}
			_ = rewrite.Parse(testCase.input, config, nil)
			if !reflect.DeepEqual(config.Rewrite, testCase.expect) {
				t.Fatalf("Must be equal.")
			}
		})
	}
}

func TestRewriteApplyRoute(t *testing.T) {
	rewrite := rewrite{}
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
				Rewrite: &RewriteConfig{},
			},
			input:  &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{},
		},
		{
			config: &Ingress{
				Rewrite: &RewriteConfig{
					RewriteTarget: "/test",
				},
			},
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Regex{
								Regex: "/hello",
							},
						},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Regex{
								Regex: "/hello",
							},
						},
					},
				},
				Rewrite: &networking.HTTPRewrite{
					UriRegex: &networking.RegexMatchAndSubstitute{
						Pattern:      "/hello",
						Substitution: "/test",
					},
				},
			},
		},
		{
			config: &Ingress{
				Rewrite: &RewriteConfig{
					RewriteHost: "test.com",
				},
			},
			input: &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{
				Rewrite: &networking.HTTPRewrite{
					Authority: "test.com",
				},
			},
		},
		{
			config: &Ingress{
				Rewrite: &RewriteConfig{
					RewriteTarget: "/test",
					RewriteHost:   "test.com",
				},
			},
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Regex{
								Regex: "/hello",
							},
						},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Regex{
								Regex: "/hello",
							},
						},
					},
				},
				Rewrite: &networking.HTTPRewrite{
					UriRegex: &networking.RegexMatchAndSubstitute{
						Pattern:      "/hello",
						Substitution: "/test",
					},
					Authority: "test.com",
				},
			},
		},
		{
			config: &Ingress{
				Rewrite: &RewriteConfig{
					RewriteTarget: "/test",
					RewritePath:   "/test",
					RewriteHost:   "test.com",
				},
			},
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Regex{
								Regex: "/hello",
							},
						},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Regex{
								Regex: "/hello",
							},
						},
					},
				},
				Rewrite: &networking.HTTPRewrite{
					Uri:       "/test",
					Authority: "test.com",
				},
			},
		},
		{
			config: &Ingress{
				Rewrite: &RewriteConfig{
					RewritePath: "/test",
				},
			},
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Prefix{
								Prefix: "/hello/",
							},
						},
					},
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{
								Exact: "/hello",
							},
						},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Prefix{
								Prefix: "/hello/",
							},
						},
					},
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{
								Exact: "/hello",
							},
						},
					},
				},
				Rewrite: &networking.HTTPRewrite{
					Uri: "/test/",
				},
			},
		},
		{
			config: &Ingress{
				Rewrite: &RewriteConfig{
					RewriteTarget: "/test",
				},
			},
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{
								Exact: "/exact",
							},
						},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{
								Exact: "/exact",
							},
						},
					},
				},
				Rewrite: &networking.HTTPRewrite{
					UriRegex: &networking.RegexMatchAndSubstitute{
						Pattern:      "/exact",
						Substitution: "/test",
					},
				},
			},
		},
		{
			config: &Ingress{
				Rewrite: &RewriteConfig{
					RewriteTarget: "/test",
				},
			},
			input: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Prefix{
								Prefix: "/prefix",
							},
						},
					},
				},
			},
			expect: &networking.HTTPRoute{
				Match: []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Prefix{
								Prefix: "/prefix",
							},
						},
					},
				},
				Rewrite: &networking.HTTPRewrite{
					UriRegex: &networking.RegexMatchAndSubstitute{
						Pattern:      "/prefix",
						Substitution: "/test",
					},
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			rewrite.ApplyRoute(inputCase.input, inputCase.config)
			if !reflect.DeepEqual(inputCase.input, inputCase.expect) {
				t.Fatal("Should be equal")
			}
		})
	}
}
