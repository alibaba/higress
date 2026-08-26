// Copyright (c) 2026 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"testing"

	extensions "istio.io/api/extensions/v1alpha1"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/config/gateway/kube"
	"istio.io/istio/pkg/config/schema/gvk"

	"github.com/alibaba/higress/v2/pkg/ingress/kube/common"
)

func TestConvertBuiltinInferenceEndpointPicker(t *testing.T) {
	ingressConfig := &IngressConfig{
		namespace: "higress-system",
		commonOptions: common.Options{
			GatewaySelectorKey:   "app",
			GatewaySelectorValue: "higress-gateway",
		},
	}
	virtualService := config.Config{
		Meta: config.Meta{GroupVersionKind: gvk.VirtualService, Name: "mixed"},
		Extra: map[string]any{constants.ConfigExtraPerRouteRuleInferencePoolConfigs: map[string]kube.InferencePoolRouteRuleConfig{
			"builtin-route":  {Mode: kube.InferencePoolEndpointPickerModeBuiltin},
			"external-route": {Mode: kube.InferencePoolEndpointPickerModeExternal, FQDN: "epp.default.svc.cluster.local", Port: "9002"},
		}},
	}

	got := ingressConfig.convertBuiltinInferenceEndpointPicker([]config.Config{virtualService})
	if len(got) != 1 {
		t.Fatalf("expected one internal WasmPlugin, got %d", len(got))
	}
	plugin := got[0].Spec.(*extensions.WasmPlugin)
	if plugin.PluginName != "ai-endpoint-picker" || plugin.FailStrategy != extensions.FailStrategy_FAIL_OPEN {
		t.Fatalf("unexpected internal plugin contract: %+v", plugin)
	}
	rules := plugin.PluginConfig.Fields["_rules_"].GetListValue().Values
	if len(rules) != 1 {
		t.Fatalf("expected one route match rule, got %d", len(rules))
	}
	matches := rules[0].GetStructValue().Fields["_match_route_"].GetListValue().Values
	if len(matches) != 1 || matches[0].GetStringValue() != "builtin-route" {
		t.Fatalf("expected only builtin-route to be bound, got %v", matches)
	}

	virtualService.Extra = nil
	if got := ingressConfig.convertBuiltinInferenceEndpointPicker([]config.Config{virtualService}); len(got) != 0 {
		t.Fatalf("expected plugin removal after builtin routes disappear, got %d configs", len(got))
	}
}
