package mcp

import (
	"path"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	extensions "istio.io/api/extensions/v1alpha1"
	mcp "istio.io/api/mcp/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name      string
		fn        func() *model.PushContext
		generator model.McpResourceGenerator
		isErr     bool
	}{
		{
			name: "VirtualService",
			fn: func() *model.PushContext {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &networking.VirtualService{},
				}
				ctx.AllVirtualServices = []config.Config{cfg}
				return ctx
			},
			generator: VirtualServiceGenerator{},
			isErr:     false,
		},
		{
			name: "Gateway",
			fn: func() *model.PushContext {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &networking.Gateway{},
				}
				ctx.AllGateways = []config.Config{cfg}
				return ctx
			},
			generator: GatewayGenerator{},
			isErr:     false,
		},
		{
			name: "EnvoyFilter",
			fn: func() *model.PushContext {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &networking.EnvoyFilter{},
				}
				ctx.AllEnvoyFilters = []config.Config{cfg}
				return ctx
			},
			generator: EnvoyFilterGenerator{},
			isErr:     false,
		},
		{
			name: "DestinationRule",
			fn: func() *model.PushContext {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &networking.DestinationRule{},
				}
				ctx.AllDestinationRules = []config.Config{cfg}
				return ctx
			},
			generator: DestinationRuleGenerator{},
			isErr:     false,
		},
		{
			name: "WasmPlugin",
			fn: func() *model.PushContext {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: &extensions.WasmPlugin{},
				}
				ctx.AllWasmplugins = []config.Config{cfg}
				return ctx
			},
			generator: WasmpluginGenerator{},
			isErr:     false,
		},
		{
			name: "WasmPlugin with wrong config",
			fn: func() *model.PushContext {
				ctx := model.NewPushContext()
				cfg := config.Config{
					Spec: "string",
				}
				ctx.AllWasmplugins = []config.Config{cfg}
				return ctx
			},
			generator: WasmpluginGenerator{},
			isErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := test.generator.Generate(nil, test.fn(), nil, nil)
			if (err != nil && !test.isErr) || (err == nil && test.isErr) {
				t.Errorf("Failed to generate config: %v", err)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	ctx := model.NewPushContext()
	cfg := config.Config{
		Spec: &networking.VirtualService{
			Hosts: []string{"127.0.0.1", "192.168.0.1"},
		},
	}
	ctx.AllVirtualServices = []config.Config{cfg}

	generator := VirtualServiceGenerator{}

	val1, _, err := generator.Generate(nil, ctx, nil, nil)
	if err != nil {
		t.Fatalf("failed to call generate: %v", err)
	}

	val2, _, err := generateVirtualService(nil, ctx, nil, nil)
	if err != nil {
		t.Fatalf("failed to call generate_VirtualService: %v", err)
	}

	c1, c2 := &mcp.Resource{}, &mcp.Resource{}
	err = ptypes.UnmarshalAny(val1[0], c1) // nolint
	if err != nil {
		t.Fatal(err)
	}

	err = ptypes.UnmarshalAny(val2[0], c2) // nolint
	if err != nil {
		t.Fatal(err)
	}

	if !c1.Body.Equal(c2.Body) {
		t.Fatalf("Marshal failed")
	}
}

func generateVirtualService(proxy *model.Proxy, push *model.PushContext, w *model.WatchedResource,
	updates *model.PushRequest) ([]*any.Any, model.XdsLogDetails, error) {
	configs := push.AllVirtualServices
	resources := make([]*any.Any, 0)
	for _, cfg := range configs {
		body, err := types.MarshalAny(cfg.Spec.(*networking.VirtualService))
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		createTime, err := types.TimestampProto(cfg.CreationTimestamp)
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resource := &mcp.Resource{
			Body: body,
			Metadata: &mcp.Metadata{
				Name:       path.Join(cfg.Namespace, cfg.Name),
				CreateTime: createTime,
			},
		}

		mcpAny, err := ptypes.MarshalAny(resource) // nolint
		if err != nil {
			return nil, model.DefaultXdsLogDetails, err
		}
		resources = append(resources, mcpAny)
	}
	return resources, model.DefaultXdsLogDetails, nil
}
