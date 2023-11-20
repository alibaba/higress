// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
)

const timeoutAnnotation = "timeout"

var (
	_ Parser       = timeout{}
	_ RouteHandler = timeout{}
)

type TimeoutConfig struct {
	time *types.Duration
}

type timeout struct{}

func (t timeout) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) error {
	if !needTimeoutConfig(annotations) {
		return nil
	}

	if time, err := annotations.ParseIntForHigress(timeoutAnnotation); err == nil {
		config.Timeout = &TimeoutConfig{
			time: &types.Duration{
				Seconds: int64(time),
			},
		}
	}
	return nil
}

func (t timeout) ApplyRoute(route *networking.HTTPRoute, config *Ingress) {
	timeout := config.Timeout
	if timeout == nil || timeout.time == nil || timeout.time.Seconds == 0 {
		return
	}

	route.Timeout = timeout.time
}

func needTimeoutConfig(annotations Annotations) bool {
	return annotations.HasHigress(timeoutAnnotation)
}
