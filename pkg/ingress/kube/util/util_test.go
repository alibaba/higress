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
	"istio.io/istio/pkg/cluster"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/wasm/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/types"
)

func TestString(t *testing.T) {
	assert.Equal(t, "cluster/foo/bar", ClusterNamespacedName{
		NamespacedName: types.NamespacedName{
			Name:      "bar",
			Namespace: "foo",
		},
		ClusterId: "cluster",
	}.String())
}

func TestSplitNamespacedName(t *testing.T) {
	testCases := []struct {
		input  string
		expect types.NamespacedName
	}{
		{
			input: "",
		},
		{
			input: "a/",
			expect: types.NamespacedName{
				Namespace: "a",
			},
		},
		{
			input: "a/b",
			expect: types.NamespacedName{
				Namespace: "a",
				Name:      "b",
			},
		},
		{
			input: "/b",
			expect: types.NamespacedName{
				Name: "b",
			},
		},
		{
			input: "b",
			expect: types.NamespacedName{
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
	istioCluster := cluster.ID("gw-123-istio")
	namespace := "default"
	serviceName := "go-httpbin-v1"
	t.Log(CreateDestinationRuleName(istioCluster, namespace, serviceName))
}

func TestMessageToGoGoStruct(t *testing.T) {
	testStr := "hello, world"
	testCases := []struct {
		name    string
		getMsg  func() (proto.Message, error)
		expect  *structpb.Struct
		wantErr bool
	}{
		{
			name: "message is nil",
			getMsg: func() (proto.Message, error) {
				return nil, nil
			},
			wantErr: true,
		},
		{
			name: "marshal error",
			getMsg: func() (proto.Message, error) {
				return &wasm.Wasm{
					Config: &v3.PluginConfig{
						Name: "error-config",
						Configuration: &anypb.Any{
							TypeUrl: "type.googleapis.com/google.protobuf.StringValue",
							Value:   []byte(testStr),
						},
					},
				}, nil
			},
			wantErr: true,
		},
		{
			name: "case 1",
			getMsg: func() (proto.Message, error) {
				bytesVal, err := proto.Marshal(wrappers.String(testStr))
				if err != nil {
					return nil, err
				}

				return &wasm.Wasm{
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
						Configuration: &anypb.Any{
							TypeUrl: "type.googleapis.com/google.protobuf.StringValue",
							Value:   bytesVal,
						},
					},
				}, nil
			},
			expect: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"config": {
						Kind: &structpb.Value_StructValue{
							StructValue: &structpb.Struct{
								Fields: map[string]*structpb.Value{
									"name": {
										Kind: &structpb.Value_StringValue{
											StringValue: "basic-auth",
										},
									},
									"fail_open": {
										Kind: &structpb.Value_BoolValue{
											BoolValue: true,
										},
									},
									"vm_config": {
										Kind: &structpb.Value_StructValue{
											StructValue: &structpb.Struct{
												Fields: map[string]*structpb.Value{
													"runtime": {
														Kind: &structpb.Value_StringValue{
															StringValue: "envoy.wasm.runtime.null",
														}},
													"code": {
														Kind: &structpb.Value_StructValue{
															StructValue: &structpb.Struct{
																Fields: map[string]*structpb.Value{
																	"local": {
																		Kind: &structpb.Value_StructValue{
																			StructValue: &structpb.Struct{
																				Fields: map[string]*structpb.Value{
																					"inline_string": {
																						Kind: &structpb.Value_StringValue{
																							StringValue: "envoy.wasm.basic_auth",
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
									"configuration": {
										Kind: &structpb.Value_StructValue{
											StructValue: &structpb.Struct{
												Fields: map[string]*structpb.Value{
													"@type": {
														Kind: &structpb.Value_StringValue{
															StringValue: "type.googleapis.com/google.protobuf.StringValue",
														},
													},
													"value": {
														Kind: &structpb.Value_StringValue{
															StringValue: testStr,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// get proto.Message
			msg, err := tt.getMsg()
			if err != nil {
				t.Fatalf("getMsg() error = %v", err)
			}

			got, err := MessageToStruct(msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("MessageToStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !proto.Equal(got, tt.expect) {
				t.Errorf("MessageToStruct() got = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestCreateServiceFQDN(t *testing.T) {
	namespace := "default"
	serviceName := "go-httpbin-v1"
	expect := "go-httpbin-v1.default.svc.cluster.local"

	got := CreateServiceFQDN(namespace, serviceName)
	t.Log(got)
	assert.Equal(t, got, expect)
}
