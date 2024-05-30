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
	newMessage := "["
	rawPrependMessage := gjson.Get(decorator, "prepend")
	if rawPrependMessage.Exists() {
		prependMessage, err := removeBrackets(rawPrependMessage.Raw)
		if err != nil {
			log.Errorf("%v", err)
			return types.ActionContinue
		}
		newMessage += prependMessage + ","
	}
	rawMessage := gjson.GetBytes(body, "messages")
	if rawMessage.Exists() {
		message, err := removeBrackets(rawMessage.Raw)
		if err != nil {
			log.Errorf("%v", err)
			return types.ActionContinue
		}
		newMessage += message
	}
	rawAppendMessage := gjson.Get(decorator, "append")
	if rawAppendMessage.Exists() {
		appendMessage, err := removeBrackets(rawAppendMessage.Raw)
		if err != nil {
			log.Errorf("%v", err)
			return types.ActionContinue
		}
		newMessage += "," + appendMessage
	}
	newMessage += "]"
	body, err := sjson.SetBytes(body, "messages", []byte(newMessage))
	if err != nil {
		log.Error("modify body failed")
	}
	// var js json.RawMessage
	// log.Infof("valid format: %d", json.Unmarshal(body, &js) == nil)
	// log.Info(string(body))
	if err = proxywasm.ReplaceHttpRequestBody(body); err != nil {
		log.Error("rewrite body failed")
	}

	return types.ActionContinue
}
