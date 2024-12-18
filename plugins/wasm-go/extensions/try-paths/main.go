package main

import (
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

// 默认超时时间为1s
var defaultTimeout uint32 = 1000

const VariableReplacedStr = "$uri"

func main() {
	wrapper.SetCtx(
		"try-paths",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type TryPathsConfig struct {
	tryPaths []string
	timeout  uint32
	tryCodes []int // 支持多个返回码
	client   wrapper.HttpClient
}

func parseConfig(json gjson.Result, config *TryPathsConfig, log wrapper.Log) error {
	for _, result := range json.Get("tryPaths").Array() {
		config.tryPaths = append(config.tryPaths, result.String())
	}

	if json.Get("tryCodes").String() == "" {
		// tryCodes默认值为["404", "403"]
		config.tryCodes = []int{http.StatusNotFound, http.StatusForbidden}
	} else {
		for _, result := range json.Get("code").Array() {
			config.tryCodes = append(config.tryCodes, int(result.Int()))
		}
	}

	timeout := json.Get("timeout").Int()
	if timeout == 0 {
		// tryPaths的timeout默认值为1s
		config.timeout = defaultTimeout
	} else {
		config.timeout = uint32(timeout)
	}

	client, err := Client(json)
	if err != nil {
		return err
	}
	config.client = client
	return nil
}

func tryHttpCall(config TryPathsConfig, index int, path string, log wrapper.Log) {
	if index >= len(config.tryPaths) {
		proxywasm.ResumeHttpRequest()
		return
	}

	requestPath := strings.Replace(config.tryPaths[index], VariableReplacedStr, path, -1)
	log.Debugf("try path start, path: %s", requestPath)
	err := config.client.Get(requestPath, nil,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if !contains(config.tryCodes, statusCode) {
				proxywasm.SendHttpResponse(uint32(statusCode), convertHttpHeadersToStruct(responseHeaders), responseBody, -1)
				return
			}
			tryHttpCall(config, index+1, path, log)
		}, config.timeout)

	if err != nil {
		log.Errorf("try path failed, path %s, error: %s", requestPath, err.Error())
		tryHttpCall(config, index+1, path, log)
	}
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config TryPathsConfig, log wrapper.Log) types.Action {
	log.Debugf("try path config: %+v", config)
	path := ctx.Path()
	if len(config.tryPaths) == 0 {
		return types.ActionContinue
	}

	tryHttpCall(config, 0, path, log)
	return types.ActionPause
}
