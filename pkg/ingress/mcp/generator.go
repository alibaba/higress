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
	"encoding/json"
	"path"
	"sort"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/known/anypb"
	mcp "istio.io/api/mcp/v1alpha1"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/xds"
	cfg "istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/log"
)

var (
	_ model.XdsResourceGenerator      = ServiceEntryGenerator{}
	_ model.XdsDeltaResourceGenerator = ServiceEntryGenerator{}
)

type GeneratorOptions struct {
	KeepConfigLabels      bool
	KeepConfigAnnotations bool
}

type ServiceEntryGenerator struct {
	Environment      *model.Environment
	Server           *xds.DiscoveryServer
	GeneratorOptions GeneratorOptions
}

func (c ServiceEntryGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource,
	updates *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	serviceEntries := c.Environment.List(gvk.ServiceEntry, model.NamespaceAll)
	if serviceEntries != nil {
		// To ensure the ip allocation logic deterministically
		// allocates the same IP to a service entry.
		sort.Slice(serviceEntries, func(i, j int) bool {
			// If creation time is the same, then behavior is nondeterministic. In this case, we can
			// pick an arbitrary but consistent ordering based on name and namespace, which is unique.
			// CreationTimestamp is stored in seconds, so this is not uncommon.
			if serviceEntries[i].CreationTimestamp == serviceEntries[j].CreationTimestamp {
				in := serviceEntries[i].Name + "." + serviceEntries[i].Namespace
				jn := serviceEntries[j].Name + "." + serviceEntries[j].Namespace
				return in < jn
			}
			return serviceEntries[i].CreationTimestamp.Before(serviceEntries[j].CreationTimestamp)
		})
	}
	return generate(proxy, serviceEntries, w, updates, c.GeneratorOptions.KeepConfigLabels, c.GeneratorOptions.KeepConfigAnnotations)
}

func (c ServiceEntryGenerator) GenerateDeltas(proxy *model.Proxy, updates *model.PushRequest,
	w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type VirtualServiceGenerator struct {
	Environment      *model.Environment
	Server           *xds.DiscoveryServer
	GeneratorOptions GeneratorOptions
}

func (c VirtualServiceGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource,
	updates *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	virtualServices := c.Environment.List(gvk.VirtualService, model.NamespaceAll)
	return generate(proxy, virtualServices, w, updates, c.GeneratorOptions.KeepConfigLabels, c.GeneratorOptions.KeepConfigAnnotations)
}

func (c VirtualServiceGenerator) GenerateDeltas(proxy *model.Proxy, updates *model.PushRequest,
	w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type DestinationRuleGenerator struct {
	Environment      *model.Environment
	Server           *xds.DiscoveryServer
	GeneratorOptions GeneratorOptions
}

func (c DestinationRuleGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource,
	updates *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	rules := c.Environment.List(gvk.DestinationRule, model.NamespaceAll)
	return generate(proxy, rules, w, updates, c.GeneratorOptions.KeepConfigLabels, c.GeneratorOptions.KeepConfigAnnotations)
}

func (c DestinationRuleGenerator) GenerateDeltas(proxy *model.Proxy, updates *model.PushRequest,
	w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type EnvoyFilterGenerator struct {
	Environment      *model.Environment
	Server           *xds.DiscoveryServer
	GeneratorOptions GeneratorOptions
}

func (c EnvoyFilterGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource,
	updates *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	filters := c.Environment.List(gvk.EnvoyFilter, model.NamespaceAll)
	return generate(proxy, filters, w, updates, c.GeneratorOptions.KeepConfigLabels, c.GeneratorOptions.KeepConfigAnnotations)
}

func (c EnvoyFilterGenerator) GenerateDeltas(proxy *model.Proxy, updates *model.PushRequest,
	w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type GatewayGenerator struct {
	Environment      *model.Environment
	Server           *xds.DiscoveryServer
	GeneratorOptions GeneratorOptions
}

func (c GatewayGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource,
	updates *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	gateways := c.Environment.List(gvk.Gateway, model.NamespaceAll)
	return generate(proxy, gateways, w, updates, c.GeneratorOptions.KeepConfigLabels, c.GeneratorOptions.KeepConfigAnnotations)
}

func (c GatewayGenerator) GenerateDeltas(proxy *model.Proxy, updates *model.PushRequest,
	w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type WasmPluginGenerator struct {
	Environment      *model.Environment
	Server           *xds.DiscoveryServer
	GeneratorOptions GeneratorOptions
}

func (c WasmPluginGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource,
	updates *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	wasmPlugins := c.Environment.List(gvk.WasmPlugin, model.NamespaceAll)
	return generate(proxy, wasmPlugins, w, updates, c.GeneratorOptions.KeepConfigLabels, c.GeneratorOptions.KeepConfigAnnotations)
}

func (c WasmPluginGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

type FallbackGenerator struct {
	Environment      *model.Environment
	Server           *xds.DiscoveryServer
	GeneratorOptions GeneratorOptions
}

func (c FallbackGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource,
	updates *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	return make(model.Resources, 0), model.DefaultXdsLogDetails, nil
}

func (c FallbackGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails, bool, error) {
	// TODO: delta implement
	return nil, nil, model.DefaultXdsLogDetails, false, nil
}

func generate(proxy *model.Proxy, configs []cfg.Config, w *model.WatchedResource,
	updates *model.PushRequest, keepLabels, keepAnnotations bool) (model.Resources, model.XdsLogDetails, error) {
	resources := make(model.Resources, 0)
	if configs == nil {
		return resources, model.DefaultXdsLogDetails, nil
	}
	for _, config := range configs {
		body, err := cfg.ToProto(config.Spec)
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
				Name: path.Join(config.Namespace, config.Name),
				CreateTime: &timestamp.Timestamp{
					Seconds: createTime.Seconds,
					Nanos:   createTime.Nanos,
				},
			},
		}
		if keepLabels {
			resource.Metadata.Labels = config.Labels
		}
		if keepAnnotations {
			resource.Metadata.Annotations = config.Annotations
		}

		// Add config.Extra to Resource's unknown fields
		if len(config.Extra) > 0 {
			if err = addExtraToUnknownFields(resource, config.Extra); err != nil {
				log.Warnf("Failed to add Extra to unknown fields: %v, extra: %v", err, config.Extra)
			}
		}

		// nolint
		mcpAny, err := anypb.New(resource)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resources = append(resources, &discovery.Resource{
			Name:     resource.Metadata.Name,
			Resource: mcpAny,
		})
	}
	return resources, model.DefaultXdsLogDetails, nil
}

// addExtraToUnknownFields adds the Extra map to the Resource's unknown fields
// We use field number 100 (which is not defined in the proto) to store the Extra data
func addExtraToUnknownFields(resource *mcp.Resource, extra map[string]any) error {
	// Serialize Extra to JSON
	extraJSON, err := json.Marshal(extra)
	if err != nil {
		return err
	}

	// Use field number 100 (arbitrary high number not used in the proto definition)
	// Resource proto only has field 1 (metadata) and field 2 (body), so 100 is safe
	// Field 100, wire type 2 (length-delimited for bytes/string)
	const extraFieldNumber = 100

	// Encode the field: tag (field number + wire type) + length + data
	tag := protowire.EncodeTag(extraFieldNumber, protowire.BytesType)
	unknownData := protowire.AppendVarint(nil, uint64(tag))
	unknownData = protowire.AppendBytes(unknownData, extraJSON)

	// Get the ProtoReflect interface to access unknown fields
	resourceReflect := resource.ProtoReflect()

	// Append to existing unknown fields
	existingUnknown := resourceReflect.GetUnknown()
	resourceReflect.SetUnknown(append(existingUnknown, unknownData...))

	log.Debugf("[addExtraToUnknownFields] Added %d bytes to Resource unknown fields (field %d)", len(unknownData), extraFieldNumber)
	log.Debugf("[addExtraToUnknownFields] Extra JSON: %s", string(extraJSON))
	log.Debugf("[addExtraToUnknownFields] Unknown data (hex): %x", unknownData)
	return nil
}
