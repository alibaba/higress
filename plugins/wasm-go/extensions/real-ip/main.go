package main

import (
	"errors"
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
			_, ipNet, _ := net.ParseCIDR(ip)
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

	trustedAddRess := isTrustedAddRess(string(remoteAddress), config, log)

	if !trustedAddRess {
		return types.ActionContinue
	}

	realIp, err := getRealIp(config, log)
	if err != nil {
		log.Warnf("get real ip failed: %v", err)
		return types.ActionContinue
	}
	proxywasm.AddHttpRequestHeader("real_ip", realIp)
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

func isTrustedAddRess(address string, config RealIpConfig, log wrapper.Log) bool {
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

func getRealIp(config RealIpConfig, log wrapper.Log) (string, error) {
	const splitSymbol = ","
	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		return "", err
	}

	realIpHeader := make([]string, 0)

	for i, find := len(headers)-1, false; i >= 0; i-- {
		h := headers[i]
		eq := strings.EqualFold(h[0], config.realIpHeader)
		if find && !eq {
			break
		} else if !eq {
			continue
		}
		// It could be an array or a single
		headerVal := h[1]
		if strings.Contains(headerVal, splitSymbol) {
			addresses := strings.Split(headerVal, splitSymbol)
			for j := len(addresses) - 1; j >= 0; j-- {
				realIpHeader = append(realIpHeader, addresses[j])
			}
			break
		} else {
			realIpHeader = append(realIpHeader, headerVal)
		}
		if !config.recursive {
			return realIpHeader[0], nil
		}
		find = true
	}

	for _, v := range realIpHeader {
		if !isTrustedAddRess(v, config, log) {
			return v, nil
		}
	}

	return "", errors.New(config.realIpHeader + " value is empty")
}
