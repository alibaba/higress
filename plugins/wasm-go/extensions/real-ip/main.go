package main

import (
	"net"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"real-ip",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type void struct{}

var VOID void

type RealIpConfig struct {
	realIpHeader string
	recursive    bool
	ipMap        map[string]void
	ipNets       []*net.IPNet
}

func parseConfig(json gjson.Result, config *RealIpConfig, log wrapper.Log) error {
	config.ipMap = make(map[string]void)
	config.ipNets = make([]*net.IPNet, 0)
	for _, item := range json.Get("real_ip_from").Array() {
		ip := item.String()
		if strings.Contains(ip, "/") {
			_, ipNet, err := net.ParseCIDR(ip)
			if err != nil {
				return err
			}
			config.ipNets = append(config.ipNets, ipNet)
		} else {
			config.ipMap[ip] = VOID
		}
	}

	header := json.Get("real_ip_header").String()
	if header == "" {
		config.realIpHeader = "X-Real-IP"
	} else {
		config.realIpHeader = header
	}

	config.recursive = json.Get("recursive").Bool()
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config RealIpConfig, log wrapper.Log) types.Action {
	remoteAddress, err := proxywasm.GetProperty([]string{"source", "address"})
	if err != nil {
		log.Warnf("get remote address failed: %v", err)
		return types.ActionContinue
	}

	trustedAddress := isTrustedAddress(string(remoteAddress), config)

	if !trustedAddress {
		return types.ActionContinue
	}

	realIp, err := getRealIp(config)
	if err != nil {
		log.Warnf("get real ip failed: %v", err)
		return types.ActionContinue
	}
	host, port, _ := net.SplitHostPort(realIp)
	if port != "" {
		err = proxywasm.SetProperty([]string{"remote", "port"}, []byte(port))
		if err != nil {
			log.Warnf("set property remote port failed: %v, port: %v, port byte: %v", err, port, []byte(port))
		}
	} else {
		host = realIp
	}

	err = proxywasm.SetProperty([]string{"remote", "address"}, []byte(host))
	if err != nil {
		log.Warnf("set property remote address failed: %v, host: %v, host byte: %v", err, host, []byte(host))
	}

	return types.ActionContinue
}

func isTrustedAddress(address string, config RealIpConfig) bool {
	host, _, _ := net.SplitHostPort(address)

	if host == "" {
		host = address
	}

	_, exists := config.ipMap[host]
	if !exists {
		for _, ipNet := range config.ipNets {
			if ipNet.Contains(net.ParseIP(host)) {
				exists = true
				break
			}
		}
	}

	return exists

}

func getRealIp(config RealIpConfig) (string, error) {
	h, err := proxywasm.GetHttpRequestHeader(config.realIpHeader)
	if err != nil {
		return "", err
	}

	addresses := strings.Split(h, ",")

	if !config.recursive {
		return addresses[len(addresses)-1], nil
	}

	for i := len(addresses) - 1; i >= 0; i-- {
		if !isTrustedAddress(addresses[i], config) {
			return addresses[i], nil
		}
	}

	return addresses[0], nil
}
