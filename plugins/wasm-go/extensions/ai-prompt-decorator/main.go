package main

import (
	"errors"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func main() {
	wrapper.SetCtx(
		"ai-prompt-decorator",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

type AIPromptDecoratorConfig struct {
	decorators map[string]string
}

func removeBrackets(raw string) (string, error) {
	startIndex := strings.Index(raw, "{")
	endIndex := strings.LastIndex(raw, "}")
	if startIndex == -1 || endIndex == -1 {
		return raw, errors.New("message format is wrong!")
	} else {
		return raw[startIndex : endIndex+1], nil
	}
}

func parseConfig(json gjson.Result, config *AIPromptDecoratorConfig, log wrapper.Log) error {
	config.decorators = make(map[string]string)
	for _, v := range json.Get("decorators").Array() {
		config.decorators[v.Get("name").String()] = v.Get("decorator").Raw
		// log.Info(v.Get("decorator").Raw)
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIPromptDecoratorConfig, log wrapper.Log) types.Action {
	decorator, _ := proxywasm.GetHttpRequestHeader("decorator")
	if decorator == "" {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	ctx.SetContext("decorator", decorator)
	proxywasm.RemoveHttpRequestHeader("decorator")
	proxywasm.RemoveHttpRequestHeader("content-length")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIPromptDecoratorConfig, body []byte, log wrapper.Log) types.Action {
	decoratorName := ctx.GetContext("decorator").(string)
	decorator := config.decorators[decoratorName]

	messageJson := `{"messages":[]}`

	prependMessage := gjson.Get(decorator, "prepend")
	if prependMessage.Exists() {
		for _, entry := range prependMessage.Array() {
			messageJson, _ = sjson.SetRaw(messageJson, "messages.-1", entry.Raw)
		}
	}

	rawMessage := gjson.GetBytes(body, "messages")
	if rawMessage.Exists() {
		for _, entry := range rawMessage.Array() {
			messageJson, _ = sjson.SetRaw(messageJson, "messages.-1", entry.Raw)
		}
	}

	appendMessage := gjson.Get(decorator, "append")
	if appendMessage.Exists() {
		for _, entry := range appendMessage.Array() {
			messageJson, _ = sjson.SetRaw(messageJson, "messages.-1", entry.Raw)
		}
	}

	newbody, err := sjson.SetRaw(string(body), "messages", gjson.Get(messageJson, "messages").Raw)
	if err != nil {
		log.Error("modify body failed")
	}
	if err = proxywasm.ReplaceHttpRequestBody([]byte(newbody)); err != nil {
		log.Error("rewrite body failed")
	}

	return types.ActionContinue
}
