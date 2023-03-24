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

func TestHeaderControlParse(t *testing.T) {
	headerControl := &headerControl{}
	inputCases := []struct {
		input  map[string]string
		expect *HeaderControlConfig
	}{
		{},
		{
			input: map[string]string{
				buildHigressAnnotationKey(requestHeaderAdd):  "one 1",
				buildHigressAnnotationKey(responseHeaderAdd): "A a",
			},
			expect: &HeaderControlConfig{
				Request: &HeaderOperation{
					Add: map[string]string{
						"one": "1",
					},
				},
				Response: &HeaderOperation{
					Add: map[string]string{
						"A": "a",
					},
				},
			},
		},
		{
			input: map[string]string{
				buildHigressAnnotationKey(requestHeaderAdd):     "one 1\n  two  2\nthree   3  \n",
				buildHigressAnnotationKey(requestHeaderUpdate):  "two 2",
				buildHigressAnnotationKey(requestHeaderRemove):  "one, two,three\n",
				buildHigressAnnotationKey(responseHeaderAdd):    "A a\nB b\n",
				buildHigressAnnotationKey(responseHeaderUpdate): "X x\nY y\n",
				buildHigressAnnotationKey(responseHeaderRemove): "x",
			},
			expect: &HeaderControlConfig{
				Request: &HeaderOperation{
					Add: map[string]string{
						"one":   "1",
						"two":   "2",
						"three": "3",
					},
					Update: map[string]string{
						"two": "2",
					},
					Remove: []string{"one", "two", "three"},
				},
				Response: &HeaderOperation{
					Add: map[string]string{
						"A": "a",
						"B": "b",
					},
					Update: map[string]string{
						"X": "x",
						"Y": "y",
					},
					Remove: []string{"x"},
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{}
			_ = headerControl.Parse(inputCase.input, config, nil)
			if !reflect.DeepEqual(inputCase.expect, config.HeaderControl) {
				t.Fatal("Should be equal")
			}
		})
	}
}

func TestHeaderControlApplyRoute(t *testing.T) {
	headerControl := headerControl{}
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
				HeaderControl: &HeaderControlConfig{},
			},
			input: &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{
				Headers: &networking.Headers{
					Request:  &networking.Headers_HeaderOperations{},
					Response: &networking.Headers_HeaderOperations{},
				},
			},
		},
		{
			config: &Ingress{
				HeaderControl: &HeaderControlConfig{
					Request: &HeaderOperation{
						Add: map[string]string{
							"one":   "1",
							"two":   "2",
							"three": "3",
						},
						Update: map[string]string{
							"two": "2",
						},
						Remove: []string{"one", "two", "three"},
					},
				},
			},
			input: &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{
				Headers: &networking.Headers{
					Request: &networking.Headers_HeaderOperations{
						Add: map[string]string{
							"one":   "1",
							"two":   "2",
							"three": "3",
						},
						Set: map[string]string{
							"two": "2",
						},
						Remove: []string{"one", "two", "three"},
					},
					Response: &networking.Headers_HeaderOperations{},
				},
			},
		},
		{
			config: &Ingress{
				HeaderControl: &HeaderControlConfig{
					Response: &HeaderOperation{
						Add: map[string]string{
							"A": "a",
							"B": "b",
						},
						Update: map[string]string{
							"X": "x",
							"Y": "y",
						},
						Remove: []string{"x"},
					},
				},
			},
			input: &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{
				Headers: &networking.Headers{
					Request: &networking.Headers_HeaderOperations{},
					Response: &networking.Headers_HeaderOperations{
						Add: map[string]string{
							"A": "a",
							"B": "b",
						},
						Set: map[string]string{
							"X": "x",
							"Y": "y",
						},
						Remove: []string{"x"},
					},
				},
			},
		},
		{
			config: &Ingress{
				HeaderControl: &HeaderControlConfig{
					Request: &HeaderOperation{
						Update: map[string]string{
							"two": "2",
						},
						Remove: []string{"one", "two", "three"},
					},
					Response: &HeaderOperation{
						Add: map[string]string{
							"A": "a",
							"B": "b",
						},
						Remove: []string{"x"},
					},
				},
			},
			input: &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{
				Headers: &networking.Headers{
					Request: &networking.Headers_HeaderOperations{
						Set: map[string]string{
							"two": "2",
						},
						Remove: []string{"one", "two", "three"},
					},
					Response: &networking.Headers_HeaderOperations{
						Add: map[string]string{
							"A": "a",
							"B": "b",
						},
						Remove: []string{"x"},
					},
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			headerControl.ApplyRoute(inputCase.input, inputCase.config)
			if !reflect.DeepEqual(inputCase.input, inputCase.expect) {
				t.Fatal("Should be equal")
			}
		})
	}
}
