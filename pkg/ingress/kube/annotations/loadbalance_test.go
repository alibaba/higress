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

	"github.com/gogo/protobuf/types"
	networking "istio.io/api/networking/v1alpha3"
)

func TestLoadBalanceParse(t *testing.T) {
	loadBalance := loadBalance{}
	inputCases := []struct {
		input  map[string]string
		expect *LoadBalanceConfig
	}{
		{},
		{
			input: map[string]string{
				buildNginxAnnotationKey(affinity):     "cookie",
				buildNginxAnnotationKey(affinityMode): "balanced",
			},
			expect: &LoadBalanceConfig{
				cookie: &consistentHashByCookie{
					name: defaultAffinityCookieName,
					path: defaultAffinityCookiePath,
					age:  &types.Duration{},
				},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(affinity):            "cookie",
				buildNginxAnnotationKey(affinityMode):        "balanced",
				buildNginxAnnotationKey(sessionCookieName):   "test",
				buildNginxAnnotationKey(sessionCookiePath):   "/test",
				buildNginxAnnotationKey(sessionCookieMaxAge): "100",
			},
			expect: &LoadBalanceConfig{
				cookie: &consistentHashByCookie{
					name: "test",
					path: "/test",
					age: &types.Duration{
						Seconds: 100,
					},
				},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(affinity):             "cookie",
				buildNginxAnnotationKey(affinityMode):         "balanced",
				buildNginxAnnotationKey(sessionCookieName):    "test",
				buildNginxAnnotationKey(sessionCookieExpires): "10",
			},
			expect: &LoadBalanceConfig{
				cookie: &consistentHashByCookie{
					name: "test",
					path: defaultAffinityCookiePath,
					age: &types.Duration{
						Seconds: 10,
					},
				},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(upstreamHashBy): "$request_uri",
			},
			expect: &LoadBalanceConfig{
				other: &consistentHashByOther{
					header: ":path",
				},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(upstreamHashBy): "$host",
			},
			expect: &LoadBalanceConfig{
				other: &consistentHashByOther{
					header: ":authority",
				},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(upstreamHashBy): "$remote_addr",
			},
			expect: &LoadBalanceConfig{
				other: &consistentHashByOther{
					header: "x-envoy-external-address",
				},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(upstreamHashBy): "$http_test",
			},
			expect: &LoadBalanceConfig{
				other: &consistentHashByOther{
					header: "test",
				},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(upstreamHashBy): "$arg_query",
			},
			expect: &LoadBalanceConfig{
				other: &consistentHashByOther{
					queryParam: "query",
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{}
			_ = loadBalance.Parse(inputCase.input, config, nil)
			if !reflect.DeepEqual(inputCase.expect, config.LoadBalance) {
				t.Fatal("Should be equal")
			}
		})
	}
}

func TestLoadBalanceApplyTrafficPolicy(t *testing.T) {
	loadBalance := loadBalance{}
	inputCases := []struct {
		config *Ingress
		input  *networking.TrafficPolicy_PortTrafficPolicy
		expect *networking.TrafficPolicy_PortTrafficPolicy
	}{
		{
			config: &Ingress{},
			input:  &networking.TrafficPolicy_PortTrafficPolicy{},
			expect: &networking.TrafficPolicy_PortTrafficPolicy{},
		},
		{
			config: &Ingress{
				LoadBalance: &LoadBalanceConfig{
					cookie: &consistentHashByCookie{
						name: "test",
						path: "/",
						age: &types.Duration{
							Seconds: 100,
						},
					},
				},
			},
			input: &networking.TrafficPolicy_PortTrafficPolicy{},
			expect: &networking.TrafficPolicy_PortTrafficPolicy{
				LoadBalancer: &networking.LoadBalancerSettings{
					LbPolicy: &networking.LoadBalancerSettings_ConsistentHash{
						ConsistentHash: &networking.LoadBalancerSettings_ConsistentHashLB{
							HashKey: &networking.LoadBalancerSettings_ConsistentHashLB_HttpCookie{
								HttpCookie: &networking.LoadBalancerSettings_ConsistentHashLB_HTTPCookie{
									Name: "test",
									Path: "/",
									Ttl: &types.Duration{
										Seconds: 100,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			config: &Ingress{
				LoadBalance: &LoadBalanceConfig{
					other: &consistentHashByOther{
						header: ":authority",
					},
				},
			},
			input: &networking.TrafficPolicy_PortTrafficPolicy{},
			expect: &networking.TrafficPolicy_PortTrafficPolicy{
				LoadBalancer: &networking.LoadBalancerSettings{
					LbPolicy: &networking.LoadBalancerSettings_ConsistentHash{
						ConsistentHash: &networking.LoadBalancerSettings_ConsistentHashLB{
							HashKey: &networking.LoadBalancerSettings_ConsistentHashLB_HttpHeaderName{
								HttpHeaderName: ":authority",
							},
						},
					},
				},
			},
		},
		{
			config: &Ingress{
				LoadBalance: &LoadBalanceConfig{
					other: &consistentHashByOther{
						queryParam: "query",
					},
				},
			},
			input: &networking.TrafficPolicy_PortTrafficPolicy{},
			expect: &networking.TrafficPolicy_PortTrafficPolicy{
				LoadBalancer: &networking.LoadBalancerSettings{
					LbPolicy: &networking.LoadBalancerSettings_ConsistentHash{
						ConsistentHash: &networking.LoadBalancerSettings_ConsistentHashLB{
							HashKey: &networking.LoadBalancerSettings_ConsistentHashLB_HttpQueryParameterName{
								HttpQueryParameterName: "query",
							},
						},
					},
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			loadBalance.ApplyTrafficPolicy(nil, inputCase.input, inputCase.config)
			if !reflect.DeepEqual(inputCase.input, inputCase.expect) {
				t.Fatal("Should be equal")
			}
		})
	}
}
