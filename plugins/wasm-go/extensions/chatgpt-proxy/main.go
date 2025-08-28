package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"chatgpt-proxy",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type MyConfig struct {
	Model       string
	ApiKey      string
	PromptParam string
	ChatgptPath string
	HumainId    string
	AIId        string
	client      wrapper.HttpClient
}

func parseConfig(json gjson.Result, config *MyConfig, log log.Log) error {
	chatgptUri := json.Get("chatgptUri").String()
	var chatgptHost string
	if chatgptUri == "" {
		config.ChatgptPath = "/v1/completions"
		chatgptHost = "api.openai.com"
	} else {
		cp, err := url.Parse(chatgptUri)
		if err != nil {
			return err
		}
		config.ChatgptPath = cp.Path
		chatgptHost = cp.Host
	}
	if config.ChatgptPath == "" {
		return errors.New("not found path in chatgptUri")
	}
	if chatgptHost == "" {
		return errors.New("not found host in chatgptUri")
	}
	config.client = wrapper.NewClusterClient(wrapper.RouteCluster{
		Host: chatgptHost,
	})
	config.Model = json.Get("model").String()
	if config.Model == "" {
		config.Model = "text-davinci-003"
	}
	config.ApiKey = json.Get("apiKey").String()
	if config.ApiKey == "" {
		return errors.New("no apiKey found in config")
	}
	config.PromptParam = json.Get("promptParam").String()
	if config.PromptParam == "" {
		config.PromptParam = "prompt"
	}
	config.HumainId = json.Get("HumainId").String()
	if config.HumainId == "" {
		config.HumainId = "Humain:"
	}
	config.AIId = json.Get("AIId").String()
	if config.AIId == "" {
		config.AIId = "AI:"
	}
	return nil
}

const bodyTemplate string = `
{
"model":"%s",
"prompt":"%s",
"temperature":0.9,
"max_tokens": 150,
"top_p": 1,
"frequency_penalty": 0.0,
"presence_penalty": 0.6,
"stop": [" %s", " %s"]
}
`

func onHttpRequestHeaders(ctx wrapper.HttpContext, config MyConfig, log log.Log) types.Action {
	pairs := strings.SplitN(ctx.Path(), "?", 2)

	if len(pairs) < 2 {
		proxywasm.SendHttpResponseWithDetail(http.StatusBadRequest, "chatgpt-proxy.empty_query_string", nil, []byte("1-need prompt param"), -1)
		return types.ActionContinue
	}
	querys, err := url.ParseQuery(pairs[1])
	if err != nil {
		proxywasm.SendHttpResponseWithDetail(http.StatusBadRequest, "chatgpt-proxy.bad_query_string", nil, []byte("2-need prompt param"), -1)
		return types.ActionContinue
	}
	var prompt []string
	var ok bool
	if prompt, ok = querys[config.PromptParam]; !ok || len(prompt) == 0 {
		proxywasm.SendHttpResponseWithDetail(http.StatusBadRequest, "chatgpt-proxy.no_prompt", nil, []byte("3-need prompt param"), -1)
		return types.ActionContinue
	}
	body := fmt.Sprintf(bodyTemplate, config.Model, prompt[0], config.HumainId, config.AIId)
	err = config.client.Post(config.ChatgptPath, [][2]string{
		{"Content-Type", "application/json"},
		{"Authorization", "Bearer " + config.ApiKey},
	}, []byte(body),
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			var headers [][2]string
			for key, value := range responseHeaders {
				headers = append(headers, [2]string{key, value[0]})
			}
			proxywasm.SendHttpResponseWithDetail(uint32(statusCode), "chatgpt-proxy.forward", headers, responseBody, -1)
		}, 10000)
	if err != nil {
		proxywasm.SendHttpResponseWithDetail(http.StatusInternalServerError, "chatgpt-proxy.request_failed", nil, []byte("Internal Error: "+err.Error()), -1)
		return types.ActionContinue
	}
	return types.ActionPause
}
