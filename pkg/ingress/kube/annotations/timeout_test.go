package annotations

import (
	"reflect"
	"testing"

	types "github.com/gogo/protobuf/types"

	networking "istio.io/api/networking/v1alpha3"
)

func TestTimeoutParse(t *testing.T) {
	timeout := timeout{}
	inputCases := []struct {
		input  map[string]string
		expect *TimeoutConfig
	}{
		{},
		{
			input: map[string]string{
				HigressAnnotationsPrefix + "/" + timeoutAnnotation: "",
			},
		},
		{
			input: map[string]string{
				HigressAnnotationsPrefix + "/" + timeoutAnnotation: "0",
			},
			expect: &TimeoutConfig{
				time: &types.Duration{},
			},
		},
		{
			input: map[string]string{
				HigressAnnotationsPrefix + "/" + timeoutAnnotation: "10",
			},
			expect: &TimeoutConfig{
				time: &types.Duration{
					Seconds: 10,
				},
			},
		},
	}

	for _, c := range inputCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{}
			_ = timeout.Parse(c.input, config, nil)
			if !reflect.DeepEqual(c.expect, config.Timeout) {
				t.Fatalf("Should be equal.")
			}
		})
	}
}

func TestTimeoutApplyRoute(t *testing.T) {
	timeout := timeout{}
	inputCases := []struct {
		config *Ingress
		input  *networking.HTTPRoute
		expect *networking.HTTPRoute
	}{
		{
			config: &Ingress{},
			input:  &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{},
		},
		{
			config: &Ingress{
				Timeout: &TimeoutConfig{},
			},
			input:  &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{},
		},
		{
			config: &Ingress{
				Timeout: &TimeoutConfig{
					time: &types.Duration{},
				},
			},
			input:  &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{},
		},
		{
			config: &Ingress{
				Timeout: &TimeoutConfig{
					time: &types.Duration{
						Seconds: 10,
					},
				},
			},
			input: &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{
				Timeout: &types.Duration{
					Seconds: 10,
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			timeout.ApplyRoute(inputCase.input, inputCase.config)
			if !reflect.DeepEqual(inputCase.input, inputCase.expect) {
				t.Fatalf("Should be equal")
			}
		})
	}
}
