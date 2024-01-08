package main

import (
	"encoding/json"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/zmap/go-iptree/iptree"
	"net"
)

const (
	DefaultRealIpHeader string = "X-Real-IP"
	DefaultDenyStatus   uint32 = 403
	DefaultDenyMessage  string = "Your IP address is not allowed."
)

type RestrictionConfig struct {
	RealIPHeader string         `json:"real_ip_header"` //真实IP头
	Allow        *iptree.IPTree `json:"allow"`          //允许的IP
	Deny         *iptree.IPTree `json:"deny"`           //拒绝的IP
	Status       uint32         `json:"status"`         //被拒绝时返回的状态码
	Message      string         `json:"message"`        //被拒绝时返回的消息
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
	}
	denyNets, err := parseIPNets(json.Get("deny").Array())
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if allowNets != nil && denyNets != nil {
		log.Warn("allow and deny cannot be set at the same time")
		return fmt.Errorf("allow and deny cannot be set at the same time")
	}
	config.Allow = allowNets
	config.Deny = denyNets
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
	if allow != nil {
		if realIp == nil {
			log.Error("realIp is nil, blocked")
			return deniedUnauthorized(config)
		}
		if _, found, _ := allow.Get(realIp); !found {
			return deniedUnauthorized(config)
		}
	}
	if deny != nil {
		if realIp == nil {
			log.Error("realIp is nil, continue")
			return types.ActionContinue
		}
		if _, found, _ := deny.Get(realIp); found {
			return deniedUnauthorized(config)
		}
	}
	return types.ActionContinue
}

func deniedUnauthorized(config RestrictionConfig) types.Action {
	body, _ := json.Marshal(map[string]string{
		"message": config.Message,
	})
	_ = proxywasm.SendHttpResponse(config.Status, nil, body, -1)
	return types.ActionContinue
}
