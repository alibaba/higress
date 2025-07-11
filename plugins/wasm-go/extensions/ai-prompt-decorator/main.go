package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func main() {}

func init() {
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

func parseConfig(jsonConfig gjson.Result, config *AIPromptDecoratorConfig, log log.Log) error {
	return json.Unmarshal([]byte(jsonConfig.Raw), config)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIPromptDecoratorConfig, log log.Log) types.Action {
	proxywasm.RemoveHttpRequestHeader("content-length")
	return types.ActionContinue
}

func replaceVariable(variable string, entry *Message) (*Message, error) {
	key := fmt.Sprintf("${%s}", variable)
	if strings.Contains(entry.Content, key) {
		value, err := proxywasm.GetProperty([]string{variable})
		if err != nil {
			return nil, err
		}
		entry.Content = strings.ReplaceAll(entry.Content, key, string(value))
	}
	return entry, nil
}

func decorateGeographicPrompt(entry *Message) (*Message, error) {
	geoArr := []string{"geo-country", "geo-province", "geo-city", "geo-isp"}

	var err error
	for _, geo := range geoArr {
		entry, err = replaceVariable(geo, entry)
		if err != nil {
			return nil, err
		}
	}

	return entry, nil
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIPromptDecoratorConfig, body []byte, log log.Log) types.Action {
	messageJson := `{"messages":[]}`

	for _, entry := range config.Prepend {
		entry, err := decorateGeographicPrompt(&entry)
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
		entry, err := decorateGeographicPrompt(&entry)
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
