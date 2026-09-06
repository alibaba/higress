// Copyright (c) 2026 Alibaba Group Holding Ltd.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	extensions "istio.io/api/extensions/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	istiotype "istio.io/api/type/v1beta1"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"

	"github.com/alibaba/higress/v2/pkg/ingress/kube/common"
	. "github.com/alibaba/higress/v2/pkg/ingress/log"
)

// A2A routes use the same Kubernetes Service and McpBridge service sources as
// ordinary Ingress routes. Only protocol processing is generated here; endpoint
// watching, TLS origination and load balancing stay in the existing controllers.
func (m *IngressConfig) convertA2APlugins(configs []common.WrapperConfig) []config.Config {
	var out []config.Config
	for _, wrapped := range configs {
		c := wrapped.Config
		raw := c.Annotations["higress.io/a2a-config"]
		if raw == "" {
			continue
		}
		var rule map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &rule); err != nil || rule == nil {
			// Retain a matching invalid configuration so a typo cannot silently
			// turn an explicitly protected A2A route into transparent proxying.
			rule = map[string]interface{}{}
		}
		// Operator-supplied JSON cannot override the generated route scope.
		for key := range rule {
			if strings.HasPrefix(key, "_") {
				delete(rule, key)
			}
		}
		rule["_match_route_"] = []string{c.Namespace + "/" + c.Name}
		data, _ := json.Marshal(map[string]interface{}{"_rules_": []interface{}{rule}})
		pluginConfig := &_struct.Struct{}
		if err := jsonpb.UnmarshalString(string(data), pluginConfig); err != nil {
			continue
		}
		pluginURL := c.Annotations["higress.io/a2a-plugin-url"]
		if pluginURL == "" {
			pluginURL = "oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/a2a-protocol:1.0.0-alpha"
		}
		digest := sha256.Sum256([]byte(string(wrapped.AnnotationsConfig.ClusterId) + "/" + c.Namespace + "/" + c.Name))
		out = append(out, config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.WasmPlugin,
				Name:             fmt.Sprintf("a2a-%x", digest[:16]),
				Namespace:        m.namespace,
			},
			Spec: &extensions.WasmPlugin{
				Selector: &istiotype.WorkloadSelector{MatchLabels: map[string]string{
					m.commonOptions.GatewaySelectorKey: m.commonOptions.GatewaySelectorValue,
				}},
				Url:          pluginURL,
				PluginConfig: pluginConfig,
				Phase:        extensions.PluginPhase_AUTHN,
				Priority:     &wrappers.Int32Value{Value: 300},
			},
		})
	}
	return out
}

// The WASM host override API is non-strict. Use Envoy's native strict session
// filter after protocol processing, so endpoint removal cannot cause fallback.
func (m *IngressConfig) a2aAffinityFilter(route *common.WrapperHTTPRoute, first bool) *config.Config {
	raw := route.WrapperConfig.Config.Annotations["higress.io/a2a-config"]
	var cfg struct {
		Affinity struct {
			Enabled bool `json:"enabled"`
		} `json:"affinity"`
	}
	if json.Unmarshal([]byte(raw), &cfg) != nil || !cfg.Affinity.Enabled {
		return nil
	}
	// Set the VirtualService policy before Istio translates it. Protobuf
	// MERGE cannot clear a pre-existing Envoy retry policy with zero values.
	route.HTTPRoute.Retries = &networking.HTTPRetry{Attempts: 0}
	patches := []interface{}{}
	const filterName = "higress.a2a.stateful_session"
	if first {
		patches = append(patches, map[string]interface{}{
			"applyTo": "HTTP_FILTER",
			"match":   map[string]interface{}{"context": "GATEWAY", "listener": map[string]interface{}{"filterChain": map[string]interface{}{"filter": map[string]interface{}{"name": "envoy.filters.network.http_connection_manager", "subFilter": map[string]interface{}{"name": "envoy.filters.http.router"}}}}},
			"patch":   map[string]interface{}{"operation": "INSERT_BEFORE", "value": map[string]interface{}{"name": filterName, "typed_config": map[string]interface{}{"@type": "type.googleapis.com/envoy.extensions.filters.http.stateful_session.v3.StatefulSession"}}},
		})
	}
	patches = append(patches, map[string]interface{}{
		"applyTo": "HTTP_ROUTE",
		"match":   map[string]interface{}{"context": "GATEWAY", "routeConfiguration": map[string]interface{}{"vhost": map[string]interface{}{"route": map[string]interface{}{"name": route.HTTPRoute.Name}}}},
		"patch": map[string]interface{}{"operation": "MERGE", "value": map[string]interface{}{
			"typed_per_filter_config": map[string]interface{}{filterName: map[string]interface{}{
				"@type": "type.googleapis.com/envoy.extensions.filters.http.stateful_session.v3.StatefulSessionPerRoute",
				"stateful_session": map[string]interface{}{"strict": true, "session_state": map[string]interface{}{
					"name":         "envoy.http.stateful_session.header",
					"typed_config": map[string]interface{}{"@type": "type.googleapis.com/envoy.extensions.http.stateful_session.header.v3.HeaderBasedSessionState", "name": "x-higress-a2a-affinity-endpoint"},
				}},
			}},
		}},
	})
	data, _ := json.Marshal(map[string]interface{}{"workloadSelector": map[string]interface{}{"labels": map[string]string{m.commonOptions.GatewaySelectorKey: m.commonOptions.GatewaySelectorValue}}, "configPatches": patches})
	spec := &networking.EnvoyFilter{}
	if err := jsonpb.UnmarshalString(string(data), spec); err != nil {
		IngressLog.Errorf("construct A2A affinity filter: %v", err)
		return nil
	}
	hash := sha256.Sum256([]byte(route.HTTPRoute.Name))
	return &config.Config{Meta: config.Meta{GroupVersionKind: gvk.EnvoyFilter, Name: fmt.Sprintf("a2a-affinity-%x", hash[:16]), Namespace: m.namespace}, Spec: spec}
}
