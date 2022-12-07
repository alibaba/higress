package mcp

import (
	"testing"

	extensions "istio.io/api/extensions/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config"
)

func Test_generate(t *testing.T) {
	tests := []struct {
		Name  string
		Spec  config.Spec
		IsErr bool
	}{
		{
			Name:  "VirtualService",
			Spec:  &networking.VirtualService{},
			IsErr: false,
		},
		{
			Name:  "Gateway",
			Spec:  &networking.Gateway{},
			IsErr: false,
		},
		{
			Name:  "EnvoyFilter",
			Spec:  &networking.EnvoyFilter{},
			IsErr: false,
		},
		{
			Name:  "DestinationRule",
			Spec:  &networking.DestinationRule{},
			IsErr: false,
		},
		{
			Name:  "WasmPlugin",
			Spec:  extensions.WasmPlugin{},
			IsErr: false,
		},
		{
			Name:  "string",
			Spec:  "test",
			IsErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			cfg := config.Config{
				Spec: test.Spec,
			}
			_, _, err := generate(nil, []config.Config{cfg}, nil, nil)
			if (err != nil && !test.IsErr) || (err == nil && test.IsErr) {
				t.Errorf("Failed to generate config: %v", err)
			}
		})
	}
}
