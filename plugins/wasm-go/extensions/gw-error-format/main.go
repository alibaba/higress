package main

import (
	"errors"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"gw-error-format",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeader),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

type MyConfig struct {
	rules      []gjson.Result
	set_header []gjson.Result
}

func parseConfig(json gjson.Result, config *MyConfig, log log.Log) error {
	config.set_header = json.Get("set_header").Array()
	config.rules = json.Get("rules").Array()
	for _, item := range config.rules {
		log.Info("config.rules: " + item.String())
		if item.Get("match.statuscode").String() == "" {
			return errors.New("missing match.statuscode in config")
		}
		if item.Get("replace.statuscode").String() == "" {
			return errors.New("missing replace.statuscode in config")
		}
	}

	return nil
}

func onHttpResponseHeader(ctx wrapper.HttpContext, config MyConfig, log log.Log) types.Action {
	dontReadResponseBody := false
	currentStatuscode, _ := proxywasm.GetHttpResponseHeader(":status")

	for _, item := range config.rules {
		configMatchStatuscode := item.Get("match.statuscode").String()
		configReplaceStatuscode := item.Get("replace.statuscode").String()
		switch currentStatuscode {
		// configMatchStatuscode value example: "403" or "503":
		case configMatchStatuscode:
			// If the response header `x-envoy-upstream-service-time`  is not found,  the request has  not  been  forwarded to the  backend  service
			_, err := proxywasm.GetHttpResponseHeader("x-envoy-upstream-service-time")
			if err != nil {
				proxywasm.RemoveHttpResponseHeader("content-length")
				proxywasm.ReplaceHttpResponseHeader(":status", configReplaceStatuscode)
				for _, item_header := range config.set_header {
					item_header.ForEach(func(key, value gjson.Result) bool {
						err := proxywasm.ReplaceHttpResponseHeader(key.String(), value.String())
						if err != nil {
							log.Critical("failed ReplaceHttpResponseHeader" + item_header.String())
						}
						return true
					})
				}
				// goto func onHttpResponseBody
				return types.ActionContinue
			} else {
				dontReadResponseBody = true
				break
			}
		default:
			// There is no matching rule
			dontReadResponseBody = true
		}
	}

	// If there is no rule match or no header for x-envoy-upstream-service-time, the onHttpResponseBody is not exec
	if dontReadResponseBody == true {
		ctx.DontReadResponseBody()
	}
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config MyConfig, body []byte, log log.Log) types.Action {
	bodyStr := string(body)

	for _, item := range config.rules {
		configMatchResponsebody := item.Get("match.responsebody").String()
		configReplaceResponsebody := item.Get("replace.responsebody").String()
		if bodyStr == configMatchResponsebody {
			proxywasm.ReplaceHttpResponseBody([]byte(configReplaceResponsebody))
			return types.ActionContinue
		}
	}

	return types.ActionContinue
}
