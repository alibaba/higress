package configmap

import (
    "encoding/json"
    "fmt"

    networking "istio.io/api/networking/v1alpha3"
    "istio.io/istio/pkg/config"
    "istio.io/istio/pkg/config/schema/gvk"

    "github.com/alibaba/higress/v2/pkg/ingress/kube/util"
)

const (
    higressMcpGuardEnvoyFilterName = "mcp-guard"
)

// McpGuard configures the MCP capability authorization guard plugin.
// This MVP uses a global filter with per-path rules for allowed capabilities
// and a simple subjectPolicy map. In a full implementation, per-route config
// and ECDS distribution should be used.
type McpGuard struct {
    Enable  bool              `json:"enable,omitempty"`
    // Header to read requested capability, e.g. X-MCP-Capability
    RequestedCapabilityHeader string            `json:"requestedCapabilityHeader,omitempty"`
    // Subject -> capabilities mapping
    SubjectPolicy             map[string][]string `json:"subjectPolicy,omitempty"`
    // Optional path-based rules to derive allowed capabilities per API
    Rules []McpGuardRule `json:"rules,omitempty"`
    // Shadow mode: only log decision, never block
    Shadow bool `json:"shadow,omitempty"`
}

type McpGuardRule struct {
    PathPrefix         string   `json:"pathPrefix,omitempty"`
    AllowedCapabilities []string `json:"allowedCapabilities,omitempty"`
}

func NewDefaultMcpGuard() *McpGuard {
    return &McpGuard{Enable: false}
}

type McpGuardController struct {
    Namespace   string
    guard       *McpGuard
    eventHandler ItemEventHandler
}

func NewMcpGuardController(namespace string) *McpGuardController {
    return &McpGuardController{Namespace: namespace, guard: NewDefaultMcpGuard()}
}

func (m *McpGuardController) GetName() string { return "mcp-guard" }

func (m *McpGuardController) ValidHigressConfig(higressConfig *HigressConfig) error {
    // No complex validation for MVP
    return nil
}

func (m *McpGuardController) RegisterItemEventHandler(h ItemEventHandler) { m.eventHandler = h }

func (m *McpGuardController) AddOrUpdateHigressConfig(_name util.ClusterNamespacedName, _old *HigressConfig, new *HigressConfig) error {
    // utilClusterNamespacedName is a lightweight alias to avoid import cycle here; defined below.
    m.guard = new.McpGuard
    if m.guard != nil && m.guard.Enable && m.eventHandler != nil {
        m.eventHandler(higressMcpGuardEnvoyFilterName)
    }
    return nil
}

// ConstructEnvoyFilters builds an EnvoyFilter that inserts the mcp-guard Wasm filter
// before router and provides a static configuration for the demo.
func (m *McpGuardController) ConstructEnvoyFilters() ([]*config.Config, error) {
    if m.guard == nil || !m.guard.Enable {
        return nil, nil
    }

    // Build plugin configuration JSON (compatible with mcp-guard plugin)
    // The plugin expects: allowedCapabilities (derived at runtime from rules), subjectPolicy, requestedCapabilityFrom.header, shadow
    // For MVP we pass rules and policy in a single JSON blob; the plugin can decide using pathPrefix rules.
    conf := map[string]any{
        "requestedCapabilityFrom": map[string]any{"header": m.guard.RequestedCapabilityHeader},
        "subjectPolicy":            m.guard.SubjectPolicy,
        "rules":                    m.guard.Rules,
        "shadow":                   m.guard.Shadow,
    }
    b, _ := json.Marshal(conf)

    // Extension config for Wasm filter
    // Note: the real deployment needs a valid wasm module URL; here we use a placeholder name
    // so that unit tests can assert structure without running Envoy.
    wasmExtension := fmt.Sprintf(`{
      "name": "envoy.filters.http.wasm",
      "typed_config": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm",
        "config": {
          "name": "mcp-guard",
          "vm_config": {"runtime": "envoy.wasm.runtime.v8"},
          "configuration": {"@type": "type.googleapis.com/google.protobuf.StringValue", "value": %q}
        }
      }
    }`, string(b))

    // Insert HTTP filter before router
    httpFilter := `{
      "name": "envoy.filters.http.wasm",
      "typed_config": {
        "@type": "type.googleapis.com/udpa.type.v1.TypedStruct",
        "type_url": "type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm"
      }
    }`

    ef := &config.Config{
        Meta: config.Meta{
            GroupVersionKind: gvk.EnvoyFilter,
            Name:             higressMcpGuardEnvoyFilterName,
            Namespace:        m.Namespace,
        },
        Spec: &networking.EnvoyFilter{
            ConfigPatches: []*networking.EnvoyFilter_EnvoyConfigObjectPatch{
                {
                    ApplyTo: networking.EnvoyFilter_HTTP_FILTER,
                    Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
                        Context: networking.EnvoyFilter_GATEWAY,
                        ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
                            Listener: &networking.EnvoyFilter_ListenerMatch{
                                FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
                                    Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
                                        Name: "envoy.filters.network.http_connection_manager",
                                        SubFilter: &networking.EnvoyFilter_ListenerMatch_SubFilterMatch{
                                            Name: "envoy.filters.http.router",
                                        },
                                    },
                                },
                            },
                        },
                    },
                    Patch: &networking.EnvoyFilter_Patch{
                        Operation: networking.EnvoyFilter_Patch_INSERT_BEFORE,
                        Value:     util.BuildPatchStruct(httpFilter),
                    },
                },
                {
                    ApplyTo: networking.EnvoyFilter_EXTENSION_CONFIG,
                    Patch: &networking.EnvoyFilter_Patch{
                        Operation: networking.EnvoyFilter_Patch_ADD,
                        Value:     util.BuildPatchStruct(wasmExtension),
                    },
                },
            },
        },
    }
    return []*config.Config{ef}, nil
}

// no-op
