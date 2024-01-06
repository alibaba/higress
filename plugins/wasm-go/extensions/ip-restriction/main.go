package main

import (
	"encoding/json"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"net"
)

const (
	DefaultRealIpHeader string = "X-Real-IP"
	DefaultDenyStatus   uint32 = 403
	DefaultDenyMessage  string = "Your IP address is not allowed."
)

type RestrictionConfig struct {
	RealIPHeader string      `json:"real_ip_header"` //真实IP头
	Allow        []net.IPNet `json:"allow"`          //允许的IP
	Deny         []net.IPNet `json:"deny"`           //拒绝的IP
	Status       uint32      `json:"status"`         //被拒绝时返回的状态码
	Message      string      `json:"message"`        //被拒绝时返回的消息
}

func main() {
	wrapper.SetCtx(
		"ip-restriction",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders))
}

func parseConfig(json gjson.Result, config *RestrictionConfig, log wrapper.Log) error {
	header := json.Get("real-ip-header")
	if header.Exists() && header.String() != "" {
		config.RealIPHeader = header.String()
	} else {
		config.RealIPHeader = DefaultRealIpHeader
	}
	status := json.Get("status")
	if status.Exists() && status.Uint() > 1 {
		config.Status = uint32(header.Uint())
	} else {
		config.Status = DefaultDenyStatus
	}
	message := json.Get("message")
	if message.Exists() && message.String() != "" {
		config.Message = message.String()
	} else {
		config.Message = DefaultDenyMessage
	}
	allowNets, err := parseIPNets(json.Get("allow").Array())
	if err != nil {
		log.Error(err.Error())
		return err
	} else {
		config.Allow = allowNets
	}
	denyNets, err := parseIPNets(json.Get("deny").Array())
	if err != nil {
		log.Error(err.Error())
		return err
	} else {
		config.Deny = denyNets
	}
	return nil
}

func onHttpRequestHeaders(context wrapper.HttpContext, config RestrictionConfig, log wrapper.Log) types.Action {
	s, err := proxywasm.GetHttpRequestHeader(config.RealIPHeader)
	if err != nil {
		return deniedUnauthorized(config)
	}
	realIp := net.ParseIP(s)
	allow := config.Allow
	deny := config.Deny
	if len(deny) != 0 {
		for _, ipNet := range deny {
			if ipNet.Contains(realIp) {
				log.Debugf("request from %s denied", s)
				return deniedUnauthorized(config)
			}
		}
	}
	if len(allow) != 0 {
		for _, ipNet := range allow {
			if ipNet.Contains(realIp) {
				return types.ActionContinue
			}
		}
		return deniedUnauthorized(config)
	} else {
		// 空白名单,直接放行
		return types.ActionContinue
	}
}

func deniedUnauthorized(config RestrictionConfig) types.Action {
	body, _ := json.Marshal(map[string]string{
		"message": config.Message,
	})
	_ = proxywasm.SendHttpResponse(config.Status, nil, body, -1)
	return types.ActionContinue
}
