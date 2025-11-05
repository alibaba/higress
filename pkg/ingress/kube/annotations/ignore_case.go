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
	networking "istio.io/api/networking/v1alpha3"
)

const (
	enableIgnoreCase = "ignore-path-case"
)

type IgnoreCaseConfig struct {
	IgnoreUriCase bool
}

type ignoreCaseMatching struct{}

func (m ignoreCaseMatching) ApplyRoute(route *networking.HTTPRoute, config *Ingress) {
	if config == nil || config.IgnoreCase == nil || !config.IgnoreCase.IgnoreUriCase {
		return
	}

	for _, v := range route.Match {
		v.IgnoreUriCase = true
	}
}

func (m ignoreCaseMatching) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) error {
	if !needIgnoreCaseMatch(annotations) {
		return nil
	}

	config.IgnoreCase = &IgnoreCaseConfig{}
	config.IgnoreCase.IgnoreUriCase, _ = annotations.ParseBoolASAP(enableIgnoreCase)

	return nil
}

func needIgnoreCaseMatch(annotation Annotations) bool {
	return annotation.HasASAP(enableIgnoreCase)
}
