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
	"path"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	extensions "istio.io/api/extensions/v1alpha1"
	mcp "istio.io/api/mcp/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/xds"
)

type VirtualServiceGenerator struct {
	Server *xds.DiscoveryServer
}

func (c VirtualServiceGenerator) Generate(proxy *model.Proxy, push *model.PushContext, w *model.WatchedResource,
	updates *model.PushRequest) ([]*any.Any, model.XdsLogDetails, error) {
	resources := make([]*any.Any, 0)
	configs := push.AllVirtualServices
	for _, config := range configs {
		body, err := types.MarshalAny(config.Spec.(*networking.VirtualService))
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		createTime, err := types.TimestampProto(config.CreationTimestamp)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resource := &mcp.Resource{
			Body: body,
			Metadata: &mcp.Metadata{
				Name:       path.Join(config.Namespace, config.Name),
				CreateTime: createTime,
			},
		}
		mcpAny, err := ptypes.MarshalAny(resource)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resources = append(resources, mcpAny)
	}
	return resources, model.DefaultXdsLogDetails, nil
}

func (c VirtualServiceGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) ([]*any.Any, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type DestinationRuleGenerator struct {
	Server *xds.DiscoveryServer
}

func (c DestinationRuleGenerator) Generate(proxy *model.Proxy, push *model.PushContext, w *model.WatchedResource,
	updates *model.PushRequest) ([]*any.Any, model.XdsLogDetails, error) {
	resources := make([]*any.Any, 0)
	configs := push.AllDestinationRules
	for _, config := range configs {
		body, err := types.MarshalAny(config.Spec.(*networking.DestinationRule))
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		createTime, err := types.TimestampProto(config.CreationTimestamp)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resource := &mcp.Resource{
			Body: body,
			Metadata: &mcp.Metadata{
				Name:       path.Join(config.Namespace, config.Name),
				CreateTime: createTime,
			},
		}
		mcpAny, err := ptypes.MarshalAny(resource)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resources = append(resources, mcpAny)
	}
	return resources, model.DefaultXdsLogDetails, nil
}

func (c DestinationRuleGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) ([]*any.Any, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type EnvoyFilterGenerator struct {
	Server *xds.DiscoveryServer
}

func (c EnvoyFilterGenerator) Generate(proxy *model.Proxy, push *model.PushContext, w *model.WatchedResource,
	updates *model.PushRequest) ([]*any.Any, model.XdsLogDetails, error) {
	resources := make([]*any.Any, 0)
	configs := push.AllEnvoyFilters
	for _, config := range configs {
		body, err := types.MarshalAny(config.Spec.(*networking.EnvoyFilter))
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		createTime, err := types.TimestampProto(config.CreationTimestamp)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resource := &mcp.Resource{
			Body: body,
			Metadata: &mcp.Metadata{
				Name:       path.Join(config.Namespace, config.Name),
				CreateTime: createTime,
			},
		}
		mcpAny, err := ptypes.MarshalAny(resource)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resources = append(resources, mcpAny)
	}
	return resources, model.DefaultXdsLogDetails, nil
}

func (c EnvoyFilterGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) ([]*any.Any, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type GatewayGenerator struct {
	Server *xds.DiscoveryServer
}

func (c GatewayGenerator) Generate(proxy *model.Proxy, push *model.PushContext, w *model.WatchedResource,
	updates *model.PushRequest) ([]*any.Any, model.XdsLogDetails, error) {
	resources := make([]*any.Any, 0)
	configs := push.AllGateways
	for _, config := range configs {
		body, err := types.MarshalAny(config.Spec.(*networking.Gateway))
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		createTime, err := types.TimestampProto(config.CreationTimestamp)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resource := &mcp.Resource{
			Body: body,
			Metadata: &mcp.Metadata{
				Name:       path.Join(config.Namespace, config.Name),
				CreateTime: createTime,
			},
		}
		mcpAny, err := ptypes.MarshalAny(resource)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resources = append(resources, mcpAny)
	}
	return resources, model.DefaultXdsLogDetails, nil
}

func (c GatewayGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) ([]*any.Any, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type WasmpluginGenerator struct {
	Server *xds.DiscoveryServer
}

func (c WasmpluginGenerator) Generate(proxy *model.Proxy, push *model.PushContext, w *model.WatchedResource,
	updates *model.PushRequest) ([]*any.Any, model.XdsLogDetails, error) {
	resources := make([]*any.Any, 0)
	configs := push.AllWasmplugins
	for _, config := range configs {
		body, err := types.MarshalAny(config.Spec.(*extensions.WasmPlugin))
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		createTime, err := types.TimestampProto(config.CreationTimestamp)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resource := &mcp.Resource{
			Body: body,
			Metadata: &mcp.Metadata{
				Name:       path.Join(config.Namespace, config.Name),
				CreateTime: createTime,
			},
		}
		mcpAny, err := ptypes.MarshalAny(resource)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resources = append(resources, mcpAny)
	}
	return resources, model.DefaultXdsLogDetails, nil
}

func (c WasmpluginGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) ([]*any.Any, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}
