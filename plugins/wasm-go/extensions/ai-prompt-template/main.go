package main

import (
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"ai-prompt-template",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

type AIPromptTemplateConfig struct {
	templates map[string]string
}

func parseConfig(json gjson.Result, config *AIPromptTemplateConfig, log wrapper.Log) error {
	config.templates = make(map[string]string)
	for _, v := range json.Get("templates").Array() {
		config.templates[v.Get("name").String()] = v.Get("template").Raw
		log.Info(v.Get("template").Raw)
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIPromptTemplateConfig, log wrapper.Log) types.Action {
	templateEnable, _ := proxywasm.GetHttpRequestHeader("template-enable")
	if templateEnable != "true" {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	proxywasm.RemoveHttpRequestHeader("content-length")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIPromptTemplateConfig, body []byte, log wrapper.Log) types.Action {
	if gjson.GetBytes(body, "template").Exists() && gjson.GetBytes(body, "properties").Exists() {
		name := gjson.GetBytes(body, "template").String()
		template := config.templates[name]
		for key, value := range gjson.GetBytes(body, "properties").Map() {
			template = strings.ReplaceAll(template, fmt.Sprintf("{{%s}}", key), value.String())
		}
		proxywasm.ReplaceHttpRequestBody([]byte(template))
	}
	return types.ActionContinue
}
