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
	types "github.com/gogo/protobuf/types"

	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/networking/core/v1alpha3/mseingress"
)

const (
	limitRPM             = "route-limit-rpm"
	limitRPS             = "route-limit-rps"
	limitBurstMultiplier = "route-limit-burst-multiplier"

	defaultBurstMultiplier = 5
	defaultStatusCode      = 429
)

var (
	_ Parser       = localRateLimit{}
	_ RouteHandler = localRateLimit{}

	second = &types.Duration{
		Seconds: 1,
	}

	minute = &types.Duration{
		Seconds: 60,
	}
)

type localRateLimitConfig struct {
	TokensPerFill uint32
	MaxTokens     uint32
	FillInterval  *types.Duration
}

type localRateLimit struct{}

func (l localRateLimit) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) error {
	if !needLocalRateLimitConfig(annotations) {
		return nil
	}

	var local *localRateLimitConfig
	defer func() {
		config.localRateLimit = local
	}()

	multiplier := defaultBurstMultiplier
	if m, err := annotations.ParseIntForHigress(limitBurstMultiplier); err == nil {
		multiplier = m
	}

	if rpm, err := annotations.ParseIntForHigress(limitRPM); err == nil {
		local = &localRateLimitConfig{
			MaxTokens:     uint32(rpm * multiplier),
			TokensPerFill: uint32(rpm),
			FillInterval:  minute,
		}
	} else if rps, err := annotations.ParseIntForHigress(limitRPS); err == nil {
		local = &localRateLimitConfig{
			MaxTokens:     uint32(rps * multiplier),
			TokensPerFill: uint32(rps),
			FillInterval:  second,
		}
	}

	return nil
}

func (l localRateLimit) ApplyRoute(route *networking.HTTPRoute, config *Ingress) {
	localRateLimitConfig := config.localRateLimit
	if localRateLimitConfig == nil {
		return
	}

	route.RouteHTTPFilters = append(route.RouteHTTPFilters, &networking.HTTPFilter{
		Name: mseingress.LocalRateLimit,
		Filter: &networking.HTTPFilter_LocalRateLimit{
			LocalRateLimit: &networking.LocalRateLimit{
				TokenBucket: &networking.TokenBucket{
					MaxTokens:     localRateLimitConfig.MaxTokens,
					TokensPefFill: localRateLimitConfig.TokensPerFill,
					FillInterval:  localRateLimitConfig.FillInterval,
				},
				StatusCode: defaultStatusCode,
			},
		},
	})
}

func needLocalRateLimitConfig(annotations Annotations) bool {
	return annotations.HasHigress(limitRPM) ||
		annotations.HasHigress(limitRPS)
}
