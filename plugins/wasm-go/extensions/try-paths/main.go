package main

import (
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

// 默认超时时间为60s
var defaultTimeout uint32 = 60000

const VariableReplacedStr = "$uri"

func main() {
	wrapper.SetCtx(
		"try-paths",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

// 自定义插件配置
type TryPathsConfig struct {
	tryPaths []string
	code     []int // 支持多个返回码
	client   wrapper.HttpClient
}

func parseConfig(json gjson.Result, config *TryPathsConfig, log wrapper.Log) error {
	// 解析出配置，更新到config中
	for _, result := range json.Get("tryPaths").Array() {
		config.tryPaths = append(config.tryPaths, result.String())
	}

	// code默认值为["404", "403"]
	if json.Get("code").String() == "" {
		config.code = []int{http.StatusNotFound, http.StatusForbidden}
	} else {
		for _, result := range json.Get("code").Array() {
			config.code = append(config.code, int(result.Int()))
		}
	}
	client, err := Client(json)
	if err != nil {
		return err
	}
	config.client = client
	return nil
}

func tryHttpCall(ctx wrapper.HttpContext, config TryPathsConfig, index int, path string, log wrapper.Log) {
	if len(config.tryPaths) == index {
		proxywasm.ResumeHttpRequest()
		return
	}
	requestPath := strings.Replace(config.tryPaths[index], VariableReplacedStr, path, -1)
	config.client.Get(requestPath, nil,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if !contains(config.code, statusCode) {
				proxywasm.SendHttpResponse(uint32(statusCode), convertHttpHeadersToStruct(responseHeaders), responseBody, -1)
				return
			}
			tryHttpCall(ctx, config, index+1, path, log)
		}, defaultTimeout)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config TryPathsConfig, log wrapper.Log) types.Action {
	path := ctx.Path()
	if len(config.tryPaths) == 0 {
		return types.ActionContinue
	}

	tryHttpCall(ctx, config, 0, path, log)
	// 需要等待异步回调完成，返回Pause状态，可以被ResumeHttpRequest恢复
	return types.ActionPause
}
