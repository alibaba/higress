package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"google.golang.org/appengine/log"
)

func main() {
	wrapper.SetCtx(
		"ai-prompt-decorator",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIPromptDecoratorConfig struct {
	Prepend []Message `json:"prepend"`
	Append  []Message `json:"append"`
}

func parseConfig(jsonConfig gjson.Result, config *AIPromptDecoratorConfig, log wrapper.Log) error {
	return json.Unmarshal([]byte(jsonConfig.Raw), config)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIPromptDecoratorConfig, log wrapper.Log) types.Action {
	proxywasm.RemoveHttpRequestHeader("content-length")
	return types.ActionContinue
}

func substituteGeographicVariable(cnt, geo string) (string, error) {
	out := ""
	key := fmt.Sprintf("${%s}", geo)
	if strings.Contains(cnt, key) {
		country, err := proxywasm.GetProperty([]string{geo})
		if err != nil {
			log.Errorf("get property geo data failed.%v", err)
			return "", err
		}
		out = strings.ReplaceAll(cnt, key, string(country))
	}
	return out, nil
}

func decorateGeographicPrompt(entry *Message, log wrapper.Log) (*Message, error) {
	cnt, err := substituteGeographicVariable(entry.Content, "geo-country")
	if err != nil {
		log.Errorf("decorateGeographicPrompt failed.%v %s", err, "geo-country")
		return nil, err
	}
	entry.Content = cnt
	cnt, err = substituteGeographicVariable(entry.Content, "geo-province")
	if err != nil {
		log.Errorf("decorateGeographicPrompt failed.%v %s", err, "geo-province")
		return nil, err
	}
	entry.Content = cnt
	cnt, err = substituteGeographicVariable(entry.Content, "geo-city")
	if err != nil {
		log.Errorf("decorateGeographicPrompt failed.%v %s", err, "geo-city")
		return nil, err
	}
	entry.Content = cnt
	cnt, err = substituteGeographicVariable(entry.Content, "geo-isp")
	if err != nil {
		log.Errorf("decorateGeographicPrompt failed.%v %s", err, "geo-isp")
		return nil, err
	}
	entry.Content = cnt

	return entry, nil
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIPromptDecoratorConfig, body []byte, log wrapper.Log) types.Action {
	messageJson := `{"messages":[]}`

	for _, entry := range config.Prepend {
		entry, err := decorateGeographicPrompt(&entry, log)
		if err != nil {
			log.Errorf("Failed to decorate geographic prompt in prepend, error: %v", err)
			return types.ActionContinue
		}

		msg, err := json.Marshal(entry)
		if err != nil {
			log.Errorf("Failed to add prepend message, error: %v", err)
			return types.ActionContinue
		}
		messageJson, _ = sjson.SetRaw(messageJson, "messages.-1", string(msg))
	}

	rawMessage := gjson.GetBytes(body, "messages")
	if !rawMessage.Exists() {
		log.Errorf("Cannot find messages field in request body")
		return types.ActionContinue
	}
	for _, entry := range rawMessage.Array() {
		messageJson, _ = sjson.SetRaw(messageJson, "messages.-1", entry.Raw)
	}

	for _, entry := range config.Append {
		entry, err := decorateGeographicPrompt(&entry, log)
		if err != nil {
			log.Errorf("Failed to decorate geographic prompt in append, error: %v", err)
			return types.ActionContinue
		}

		msg, err := json.Marshal(entry)
		if err != nil {
			log.Errorf("Failed to add prepend message, error: %v", err)
			return types.ActionContinue
		}
		messageJson, _ = sjson.SetRaw(messageJson, "messages.-1", string(msg))
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
