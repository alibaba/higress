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

	types "github.com/gogo/protobuf/types"

	networking "istio.io/api/networking/v1alpha3"
)

func TestTimeoutParse(t *testing.T) {
	timeout := timeout{}
	inputCases := []struct {
		input  map[string]string
		expect *TimeoutConfig
	}{
		{},
		{
			input: map[string]string{
				HigressAnnotationsPrefix + "/" + timeoutAnnotation: "",
			},
		},
		{
			input: map[string]string{
				HigressAnnotationsPrefix + "/" + timeoutAnnotation: "0",
			},
			expect: &TimeoutConfig{
				time: &types.Duration{},
			},
		},
		{
			input: map[string]string{
				HigressAnnotationsPrefix + "/" + timeoutAnnotation: "10",
			},
			expect: &TimeoutConfig{
				time: &types.Duration{
					Seconds: 10,
				},
			},
		},
	}

	for _, c := range inputCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{}
			_ = timeout.Parse(c.input, config, nil)
			if !reflect.DeepEqual(c.expect, config.Timeout) {
				t.Fatalf("Should be equal.")
			}
		})
	}
}

func TestTimeoutApplyRoute(t *testing.T) {
	timeout := timeout{}
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
				Timeout: &TimeoutConfig{},
			},
			input:  &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{},
		},
		{
			config: &Ingress{
				Timeout: &TimeoutConfig{
					time: &types.Duration{},
				},
			},
			input:  &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{},
		},
		{
			config: &Ingress{
				Timeout: &TimeoutConfig{
					time: &types.Duration{
						Seconds: 10,
					},
				},
			},
			input: &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{
				Timeout: &types.Duration{
					Seconds: 10,
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			timeout.ApplyRoute(inputCase.input, inputCase.config)
			if !reflect.DeepEqual(inputCase.input, inputCase.expect) {
				t.Fatalf("Should be equal")
			}
		})
	}
}
