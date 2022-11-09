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

package util

import (
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/wasm/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"
	any "google.golang.org/protobuf/types/known/anypb"
	"istio.io/istio/pilot/pkg/model"
)

func TestSplitNamespacedName(t *testing.T) {
	testCases := []struct {
		input  string
		expect model.NamespacedName
	}{
		{
			input: "",
		},
		{
			input: "a/",
			expect: model.NamespacedName{
				Namespace: "a",
			},
		},
		{
			input: "a/b",
			expect: model.NamespacedName{
				Namespace: "a",
				Name:      "b",
			},
		},
		{
			input: "/b",
			expect: model.NamespacedName{
				Name: "b",
			},
		},
		{
			input: "b",
			expect: model.NamespacedName{
				Name: "b",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			result := SplitNamespacedName(testCase.input)
			if result != testCase.expect {
				t.Fatalf("expect is %v, but actual is %v", testCase.expect, result)
			}
		})
	}
}

func TestCreateDestinationRuleName(t *testing.T) {
	istioCluster := "gw-123-istio"
	namespace := "default"
	serviceName := "go-httpbin-v1"
	t.Log(CreateDestinationRuleName(istioCluster, namespace, serviceName))
}

func TestMessageToGoGoStruct(t *testing.T) {
	bytes := []byte("test")
	wasm := &wasm.Wasm{
		Config: &v3.PluginConfig{
			Name:     "basic-auth",
			FailOpen: true,
			Vm: &v3.PluginConfig_VmConfig{
				VmConfig: &v3.VmConfig{
					Runtime: "envoy.wasm.runtime.null",
					Code: &corev3.AsyncDataSource{
						Specifier: &corev3.AsyncDataSource_Local{
							Local: &corev3.DataSource{
								Specifier: &corev3.DataSource_InlineString{
									InlineString: "envoy.wasm.basic_auth",
								},
							},
						},
					},
				},
			},
			Configuration: &any.Any{
				TypeUrl: "type.googleapis.com/google.protobuf.StringValue",
				Value:   bytes,
			},
		},
	}

	gogoStruct, _ := MessageToGoGoStruct(wasm)
	t.Log(gogoStruct)
}
