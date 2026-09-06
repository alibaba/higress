// Copyright (c) 2026 Alibaba Group Holding Ltd.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/alibaba/higress/v2/pkg/ingress/kube/annotations"
	"github.com/alibaba/higress/v2/pkg/ingress/kube/common"
	"github.com/stretchr/testify/require"
	extensions "istio.io/api/extensions/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config"
)

func TestA2ARoutePluginScopeAndRemoval(t *testing.T) {
	m := &IngressConfig{namespace: "higress-system"}
	c := common.WrapperConfig{Config: &config.Config{Meta: config.Meta{
		Name: "agent", Namespace: "demo", Annotations: map[string]string{
			"higress.io/a2a-config":     `{"agent":{"id":"demo","externalBaseURL":"https://agent.example.com"},"_match_route_":["other/route"]}`,
			"higress.io/a2a-plugin-url": "file:///opt/a2a/fixed.wasm",
		},
	}}, AnnotationsConfig: &annotations.Ingress{}}
	plugins := m.convertA2APlugins([]common.WrapperConfig{c})
	require.Len(t, plugins, 1)
	require.LessOrEqual(t, len(plugins[0].Name), 63)
	p := plugins[0].Spec.(*extensions.WasmPlugin)
	require.Equal(t, "file:///opt/a2a/fixed.wasm", p.Url)
	rule := p.PluginConfig.Fields["_rules_"].GetListValue().Values[0].GetStructValue()
	require.Equal(t, "demo/agent", rule.Fields["_match_route_"].GetListValue().Values[0].GetStringValue())
	require.Equal(t, "demo", rule.Fields["agent"].GetStructValue().Fields["id"].GetStringValue())
	require.Empty(t, m.convertA2APlugins(nil))
	delete(c.Config.Annotations, "higress.io/a2a-config")
	require.Empty(t, m.convertA2APlugins([]common.WrapperConfig{c}))
}

func TestA2AAffinityRouteUsesStrictNativeFilter(t *testing.T) {
	m := &IngressConfig{namespace: "higress-system"}
	route := &common.WrapperHTTPRoute{
		HTTPRoute:     &networking.HTTPRoute{Name: "demo/agent"},
		WrapperConfig: &common.WrapperConfig{Config: &config.Config{Meta: config.Meta{Annotations: map[string]string{"higress.io/a2a-config": `{"affinity":{"enabled":true}}`}}}},
	}
	ef := m.a2aAffinityFilter(route, true)
	require.NotNil(t, ef)
	spec := ef.Spec.(*networking.EnvoyFilter)
	require.Len(t, spec.ConfigPatches, 2)
	require.Equal(t, "envoy.filters.http.router", spec.ConfigPatches[0].Match.GetListener().FilterChain.Filter.SubFilter.Name)
	routePatch := spec.ConfigPatches[1]
	require.Equal(t, "demo/agent", routePatch.Match.GetRouteConfiguration().Vhost.Route.Name)
	value := routePatch.Patch.Value.AsMap()
	filter := value["typed_per_filter_config"].(map[string]interface{})["higress.a2a.stateful_session"].(map[string]interface{})
	require.Equal(t, true, filter["stateful_session"].(map[string]interface{})["strict"])
	require.NotNil(t, route.HTTPRoute.Retries)
	require.Zero(t, route.HTTPRoute.Retries.Attempts)
	require.Len(t, m.a2aAffinityFilter(route, false).Spec.(*networking.EnvoyFilter).ConfigPatches, 1)
	delete(route.WrapperConfig.Config.Annotations, "higress.io/a2a-config")
	require.Nil(t, m.a2aAffinityFilter(route, true))
}
