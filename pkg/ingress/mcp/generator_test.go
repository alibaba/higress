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

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	extensions "istio.io/api/extensions/v1alpha1"
	mcp "istio.io/api/mcp/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name      string
		fn        func() config.Config
		generator func(config.Config) model.XdsResourceGenerator
		isErr     bool
	}{
		{
			name: "VirtualService",
			fn: func() config.Config {
				return config.Config{
					Meta: config.Meta{
						GroupVersionKind: gvk.VirtualService,
					},
					Spec: &networking.VirtualService{},
				}
			},
			generator: func(c config.Config) model.XdsResourceGenerator {
				env := model.NewEnvironment()
				env.ConfigStore = model.NewFakeStore()
				_, _ = env.ConfigStore.Create(c)
				return VirtualServiceGenerator{Environment: env}
			},
			isErr: false,
		},
		{
			name: "Gateway",
			fn: func() config.Config {
				return config.Config{
					Meta: config.Meta{
						GroupVersionKind: gvk.Gateway,
					},
					Spec: &networking.Gateway{},
				}
			},
			generator: func(c config.Config) model.XdsResourceGenerator {
				env := model.NewEnvironment()
				env.ConfigStore = model.NewFakeStore()
				_, _ = env.ConfigStore.Create(c)
				return GatewayGenerator{Environment: env}
			},
			isErr: false,
		},
		{
			name: "EnvoyFilter",
			fn: func() config.Config {
				return config.Config{
					Meta: config.Meta{
						GroupVersionKind: gvk.EnvoyFilter,
					},
					Spec: &networking.EnvoyFilter{},
				}
			},
			generator: func(c config.Config) model.XdsResourceGenerator {
				env := model.NewEnvironment()
				env.ConfigStore = model.NewFakeStore()
				_, _ = env.ConfigStore.Create(c)
				return EnvoyFilterGenerator{Environment: env}
			},
			isErr: false,
		},
		{
			name: "DestinationRule",
			fn: func() config.Config {
				return config.Config{
					Meta: config.Meta{
						GroupVersionKind: gvk.DestinationRule,
					},
					Spec: &networking.DestinationRule{},
				}
			},
			generator: func(c config.Config) model.XdsResourceGenerator {
				env := model.NewEnvironment()
				env.ConfigStore = model.NewFakeStore()
				_, _ = env.ConfigStore.Create(c)
				return DestinationRuleGenerator{Environment: env}
			},
			isErr: false,
		},
		{
			name: "WasmPlugin",
			fn: func() config.Config {
				return config.Config{
					Meta: config.Meta{
						GroupVersionKind: gvk.WasmPlugin,
					},
					Spec: &extensions.WasmPlugin{},
				}
			},
			generator: func(c config.Config) model.XdsResourceGenerator {
				env := model.NewEnvironment()
				env.ConfigStore = model.NewFakeStore()
				_, _ = env.ConfigStore.Create(c)
				return WasmPluginGenerator{Environment: env}
			},
			isErr: false,
		},
		{
			name: "ServiceEntry",
			fn: func() config.Config {
				return config.Config{
					Meta: config.Meta{
						GroupVersionKind: gvk.ServiceEntry,
					},
					Spec: &networking.ServiceEntry{},
				}
			},
			generator: func(c config.Config) model.XdsResourceGenerator {
				env := model.NewEnvironment()
				env.ConfigStore = model.NewFakeStore()
				_, _ = env.ConfigStore.Create(c)
				return ServiceEntryGenerator{Environment: env}
			},
			isErr: false,
		},
		{
			name: "WasmPlugin with wrong config",
			fn: func() config.Config {
				return config.Config{
					Meta: config.Meta{
						GroupVersionKind: gvk.WasmPlugin,
					},
					Spec: "string",
				}
			},
			generator: func(c config.Config) model.XdsResourceGenerator {
				env := model.NewEnvironment()
				env.ConfigStore = model.NewFakeStore()
				_, _ = env.ConfigStore.Create(c)
				return WasmPluginGenerator{Environment: env}
			},
			isErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				err error
				val model.Resources
			)

			cfg := test.fn()
			func() {
				defer func() {
					if err := recover(); err != nil && !test.isErr {
						t.Fatalf("Failed to generate config: %v", err)
					}
				}()

				val, _, err = test.generator(cfg).Generate(nil, nil, nil)
				if (err != nil && !test.isErr) || (err == nil && test.isErr) {
					t.Fatalf("Failed to generate config: %v", err)
				}
			}()

			if test.isErr { // if the func 'Generate' should occur an error, just return
				return
			}

			resource := &mcp.Resource{}
			err = ptypes.UnmarshalAny(val[0].Resource, resource)
			if err != nil {
				t.Fatal(err)
			}

			specType := reflect.TypeOf(cfg.Spec)
			if specType.Kind() == reflect.Ptr {
				specType = specType.Elem()
			}

			target := reflect.New(specType).Interface().(proto.Message)
			if err = ptypes.UnmarshalAny(resource.Body, target); err != nil {
				t.Fatal(err)
			}

			if !test.isErr && !proto.Equal(cfg.Spec.(proto.Message), target) {
				t.Fatal("failed ")
			}
		})
	}
}
