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

package mcp

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/anypb"
	extensions "istio.io/api/extensions/v1alpha1"
	mcp "istio.io/api/mcp/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name      string
		fn        func() (*model.PushContext, any)
		generator model.McpResourceGenerator
		isErr     bool
	}{
		{
			name: "VirtualService",
			fn: func() (*model.PushContext, any) {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &networking.VirtualService{},
				}
				ctx.AllVirtualServices = []config.Config{cfg}
				return ctx, cfg.Spec
			},
			generator: VirtualServiceGenerator{},
			isErr:     false,
		},
		{
			name: "Gateway",
			fn: func() (*model.PushContext, any) {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &networking.Gateway{},
				}
				ctx.AllGateways = []config.Config{cfg}
				return ctx, cfg.Spec
			},
			generator: GatewayGenerator{},
			isErr:     false,
		},
		{
			name: "EnvoyFilter",
			fn: func() (*model.PushContext, any) {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &networking.EnvoyFilter{},
				}
				ctx.AllEnvoyFilters = []config.Config{cfg}
				return ctx, cfg.Spec
			},
			generator: EnvoyFilterGenerator{},
			isErr:     false,
		},
		{
			name: "DestinationRule",
			fn: func() (*model.PushContext, any) {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &networking.DestinationRule{},
				}
				ctx.AllDestinationRules = []config.Config{cfg}
				return ctx, cfg.Spec
			},
			generator: DestinationRuleGenerator{},
			isErr:     false,
		},
		{
			name: "WasmPlugin",
			fn: func() (*model.PushContext, any) {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &extensions.WasmPlugin{},
				}
				ctx.AllWasmplugins = []config.Config{cfg}
				return ctx, cfg.Spec
			},
			generator: WasmpluginGenerator{},
			isErr:     false,
		},
		{
			name: "ServiceEntry",
			fn: func() (*model.PushContext, any) {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &networking.ServiceEntry{},
				}
				ctx.AllServiceEntries = []config.Config{cfg}
				return ctx, cfg.Spec
			},
			generator: ServiceEntryGenerator{},
			isErr:     false,
		},
		{
			name: "WasmPlugin with wrong config",
			fn: func() (*model.PushContext, any) {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: "string",
				}
				ctx.AllWasmplugins = []config.Config{cfg}
				return ctx, cfg.Spec
			},
			generator: WasmpluginGenerator{},
			isErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				err error
				val []*anypb.Any
			)

			pushCtx, spec := test.fn()
			func() {
				defer func() {
					if err := recover(); err != nil && !test.isErr {
						t.Fatalf("Failed to generate config: %v", err)
					}
				}()

				val, _, err = test.generator.Generate(nil, pushCtx, nil, nil)
				if (err != nil && !test.isErr) || (err == nil && test.isErr) {
					t.Fatalf("Failed to generate config: %v", err)
				}
			}()

			if test.isErr { // if the func 'Generate' should occur an error, just return
				return
			}

			resource := &mcp.Resource{}
			err = ptypes.UnmarshalAny(val[0], resource)
			if err != nil {
				t.Fatal(err)
			}

			specType := reflect.TypeOf(spec)
			if specType.Kind() == reflect.Ptr {
				specType = specType.Elem()
			}

			target := reflect.New(specType).Interface().(proto.Message)
			if err = types.UnmarshalAny(resource.Body, target); err != nil {
				t.Fatal(err)
			}

			if !test.isErr && !proto.Equal(spec.(proto.Message), target) {
				t.Fatal("failed ")
			}
		})
	}
}
