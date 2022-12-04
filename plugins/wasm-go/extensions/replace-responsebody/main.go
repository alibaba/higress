package main

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"replace-responsebody",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeader),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

type MyConfig struct {
	rules []gjson.Result
}

func parseConfig(json gjson.Result, config *MyConfig, log wrapper.Log) error {
	if json.Get("rules").IsArray() == false {
		return errors.New("config rules Not a correct Array of object")
	}
	config.rules = json.Get("rules").Array()
	for _, item := range config.rules {
		// judge config is empty
		if item.Get("match.statuscode").String() == "" {
			return errors.New("missing config_statuscode in config")
		}
		if item.Get("match.responsebody").String() == "" {
			return errors.New("missing config_responsebody in config")
		}
		if item.Get("match.replace.statuscode").String() == "" {
			return errors.New("missing config_replace_statuscode in config")
		}
		if item.Get("match.replace.responsebody").String() == "" {
			return errors.New("missing config_replace_responsebody in config")
		}
		if item.Get("match.replace.responseheader").IsArray() == false {
			return errors.New("config rules.match.replace.responseheader Not a correct Array of object")
		}
	}

	return nil
}

func onHttpResponseHeader(ctx wrapper.HttpContext, config MyConfig, log wrapper.Log) types.Action {
	//proxywasm.LogInfo("step onHttpResponseHeader 1")

	var DontReadResponseBody = false
	//path, _ := proxywasm.GetHttpRequestHeader(":path")
	//proxywasm.LogInfo("path: " + path)
	//////// judge statuscode
	for _, item := range config.rules {
		status, err := proxywasm.GetHttpResponseHeader(":status")
		if err != nil {
			proxywasm.LogCritical("failed GetHttpResponseHeader :status")
		}
		//proxywasm.LogInfo("status:" + status)
		config_statuscode := item.Get("match.statuscode").String()
		//proxywasm.LogInfo("config_statuscode:" + config_statuscode)
		config_replace_statuscode := item.Get("match.replace.statuscode").String()
		//proxywasm.LogInfo("config_replace_statuscode:" + config_replace_statuscode)
		config_replace_responseheader := item.Get("match.replace.responseheader").Array()
		switch status {
		//case "403", "503":
		case config_statuscode:
			// X-enge-upward-service-time If the ResponseHeader is not found, it is not forwarded to the back-end service
			x_envoy_upstream_service_time, err := proxywasm.GetHttpResponseHeader("x-envoy-upstream-service-time")
			if x_envoy_upstream_service_time == "" || len(x_envoy_upstream_service_time) < 1 || err != nil {
				//proxywasm.LogInfo("not find ResponseHeader x-envoy-upstream-service-time going to set the header")
				proxywasm.RemoveHttpResponseHeader("content-length")
				// replace statuscode
				err = proxywasm.ReplaceHttpResponseHeader(":status", config_replace_statuscode)
				if err != nil {
					proxywasm.LogCritical("failed ReplaceHttpResponseHeader :status")
				}
				// Replace ResponseHeader
				if item.Get("match.replace.responseheader").Exists() {
					for _, item := range config_replace_responseheader {
						item.ForEach(func(key, value gjson.Result) bool {
							//proxywasm.LogInfo("key.String(), value.String()" + key.String() + ":" + value.String())
							err = proxywasm.ReplaceHttpResponseHeader(key.String(), value.String())
							if err != nil {
								proxywasm.LogCritical("failed ReplaceHttpResponseHeader" + item.String())
							}
							return true
						})
					}
				}
			}
			return types.ActionContinue
		default:
			//proxywasm.LogInfo("DontReadResponseBody = true")
			DontReadResponseBody = true
		}
	}

	if DontReadResponseBody == true {
		ctx.DontReadResponseBody()
	}
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config MyConfig, body []byte, log wrapper.Log) types.Action {
	//proxywasm.LogInfo("step onHttpResponseBody 1")
	bodyStr := string(body)

	//////// judge responsebody
	for _, item := range config.rules {
		config_responsebody := item.Get("match.responsebody").String()
		config_replace_responsebody := item.Get("match.replace.responsebody").String()
		//proxywasm.LogInfo("bodyStr:" + bodyStr)
		//proxywasm.LogInfo("config_responsebody:" + config_responsebody)
		if bodyStr == config_responsebody {
			log.Warn(bodyStr)
			// Replace ResponseBody
			//proxywasm.LogInfo("config_replace_responsebody:" + config_replace_responsebody)
			err := proxywasm.ReplaceHttpResponseBody([]byte(config_replace_responsebody))
			if err != nil {
				proxywasm.LogCritical("failed config_replace_responsebody" + config_replace_responsebody)
			}
			return types.ActionContinue
		}
	}

	return types.ActionContinue
}
