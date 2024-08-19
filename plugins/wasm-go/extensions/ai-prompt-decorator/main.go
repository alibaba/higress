package main

import (
	"encoding/json"
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

func onHttpRequestBody(ctx wrapper.HttpContext, config AIPromptDecoratorConfig, body []byte, log wrapper.Log) types.Action {
	messageJson := `{"messages":[]}`

	for _, entry := range config.Prepend {
		if strings.Contains(entry.Content, "${geo-country}") {
			country, err := proxywasm.GetProperty([]string{"geo-country"})
			if err != nil {
				log.Errorf("get property geo-country for prepend failed.%v", err)
				return types.ActionContinue
			}
			entry.Content = strings.ReplaceAll(entry.Content, "${geo-country}", string(country))
		}

		if strings.Contains(entry.Content, "${geo-province}") {
			province, err := proxywasm.GetProperty([]string{"geo-province"})
			if err != nil {
				log.Errorf("get property geo-province for prepend failed.%v", err)
				return types.ActionContinue
			}
			entry.Content = strings.ReplaceAll(entry.Content, "${geo-province}", string(province))
		}

		if strings.Contains(entry.Content, "${geo-city}") {
			city, err := proxywasm.GetProperty([]string{"geo-city"})
			if err != nil {
				log.Errorf("get property geo-city for prepend failed.%v", err)
				return types.ActionContinue
			}
			entry.Content = strings.ReplaceAll(entry.Content, "${geo-city}", string(city))
		}

		if strings.Contains(entry.Content, "${geo-isp}") {
			isp, err := proxywasm.GetProperty([]string{"geo-isp"})
			if err != nil {
				log.Errorf("get property geo-isp for prepend failed.%v", err)
				return types.ActionContinue
			}
			entry.Content = strings.ReplaceAll(entry.Content, "${geo-isp}", string(isp))
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
		if strings.Contains(entry.Content, "${geo-country}") {
			country, err := proxywasm.GetProperty([]string{"geo-country"})
			if err != nil {
				log.Errorf("get property geo-country for append failed.%v", err)
				return types.ActionContinue
			}
			entry.Content = strings.ReplaceAll(entry.Content, "${geo-country}", string(country))
		}

		if strings.Contains(entry.Content, "${geo-province}") {
			province, err := proxywasm.GetProperty([]string{"geo-province"})
			if err != nil {
				log.Errorf("get property geo-province for append failed.%v", err)
				return types.ActionContinue
			}
			entry.Content = strings.ReplaceAll(entry.Content, "${geo-province}", string(province))
		}

		if strings.Contains(entry.Content, "${geo-city}") {
			city, err := proxywasm.GetProperty([]string{"geo-city"})
			if err != nil {
				log.Errorf("get property geo-city for append failed.%v", err)
				return types.ActionContinue
			}
			entry.Content = strings.ReplaceAll(entry.Content, "${geo-city}", string(city))
		}

		if strings.Contains(entry.Content, "${geo-isp}") {
			isp, err := proxywasm.GetProperty([]string{"geo-isp"})
			if err != nil {
				log.Errorf("get property geo-isp for append failed.%v", err)
				return types.ActionContinue
			}
			entry.Content = strings.ReplaceAll(entry.Content, "${geo-isp}", string(isp))
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
