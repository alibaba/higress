package utils

import "github.com/higress-group/proxy-wasm-go-sdk/proxywasm"

func GetRouteName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		return "", err
	} else {
		return string(raw), nil
	}
}

func GetClusterName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err != nil {
		return "", err
	} else {
		return string(raw), nil
	}
}
