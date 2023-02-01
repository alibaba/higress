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

// nolint
import (
	"path"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	mcp "istio.io/api/mcp/v1alpha1"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/xds"
	cfg "istio.io/istio/pkg/config"
)

type ServiceEntryGenerator struct {
	Server *xds.DiscoveryServer
}

func (c ServiceEntryGenerator) Generate(proxy *model.Proxy, push *model.PushContext, w *model.WatchedResource,
	updates *model.PushRequest) ([]*any.Any, model.XdsLogDetails, error) {
	return generate(proxy, push.AllServiceEntries, w, updates)
}

func (c ServiceEntryGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) ([]*any.Any, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type VirtualServiceGenerator struct {
	Server *xds.DiscoveryServer
}

func (c VirtualServiceGenerator) Generate(proxy *model.Proxy, push *model.PushContext, w *model.WatchedResource,
	updates *model.PushRequest) ([]*any.Any, model.XdsLogDetails, error) {
	return generate(proxy, push.AllVirtualServices, w, updates)
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
	return generate(proxy, push.AllDestinationRules, w, updates)
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
	return generate(proxy, push.AllEnvoyFilters, w, updates)
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
	return generate(proxy, push.AllGateways, w, updates)
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
	return generate(proxy, push.AllWasmplugins, w, updates)
}

func (c WasmpluginGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) ([]*any.Any, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

func generate(proxy *model.Proxy, configs []cfg.Config, w *model.WatchedResource,
	updates *model.PushRequest) ([]*any.Any, model.XdsLogDetails, error) {
	resources := make([]*any.Any, 0)
	for _, config := range configs {
		body, err := cfg.ToProtoGogo(config.Spec)
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
		// nolint
		mcpAny, err := ptypes.MarshalAny(resource)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resources = append(resources, mcpAny)
	}
	return resources, model.DefaultXdsLogDetails, nil
}
