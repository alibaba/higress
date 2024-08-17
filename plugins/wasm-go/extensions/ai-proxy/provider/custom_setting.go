package provider

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	settingNameMaxTokens   = "max_tokens"
	settingNameTemperature = "temperature"
	settingNameTopP        = "top_p"
	settingNameTopK        = "top_k"
	settingNameSeed        = "seed"
)

var maxTokensMapping = map[string]string{
	"openai":  "max_tokens",
	"baidu":   "max_output_tokens",
	"spark":   "max_tokens",
	"qwen":    "max_tokens",
	"gemini":  "maxOutputTokens",
	"claude":  "max_tokens",
	"minimax": "tokens_to_generate",
}

var temperatureMapping = map[string]string{
	"openai":  "temperature",
	"baidu":   "temperature",
	"spark":   "temperature",
	"qwen":    "temperature",
	"gemini":  "temperature",
	"hunyuan": "Temperature",
	"claude":  "temperature",
	"minimax": "temperature",
}

var topPMapping = map[string]string{
	"openai":  "top_p",
	"baidu":   "top_p",
	"qwen":    "top_p",
	"gemini":  "topP",
	"hunyuan": "TopP",
	"claude":  "top_p",
	"minimax": "top_p",
}

var topKMapping = map[string]string{
	"spark":  "top_k",
	"gemini": "topK",
	"claude": "top_k",
}

var seedMapping = map[string]string{
	"openai": "seed",
	"qwen":   "seed",
}

var settingMapping = map[string]map[string]string{
	settingNameMaxTokens:   maxTokensMapping,
	settingNameTemperature: temperatureMapping,
	settingNameTopP:        topPMapping,
	settingNameTopK:        topKMapping,
	settingNameSeed:        seedMapping,
}

type CustomSetting struct {
	// @Title zh-CN 参数名称
	// @Description zh-CN 想要设置的参数的名称，例如max_tokens
	settingName string
	// @Title zh-CN 参数值
	// @Description zh-CN 想要设置的参数的值，例如0
	settingValue string
	// @Title zh-CN 设置模式
	// @Description zh-CN 参数设置的模式，可以设置为"fill"或者"overwrite"，如果为"fill"则只在用户没有设置这个参数时填充参数，如果为"overwrite"则会直接覆盖用户原有的参数设置
	settingMode string
	// @Title zh-CN json edit 模式
	// @Description zh-CN 是否启用json edit模式。如果启用，会直接用输入的settingName和settingValue去更改请求中的json内容，而不对参数名称做任何限制和修改。
	enableJsonEdit bool
}

func (c *CustomSetting) FromJson(json gjson.Result) {
	c.settingName = json.Get("settingName").String()
	c.settingValue = json.Get("settingValue").Raw
	c.settingMode = json.Get("settingMode").String()
	c.enableJsonEdit = json.Get("enableJsonEdit").Bool()
}

func (c *CustomSetting) Validate() bool {
	return c.settingName != ""
}

func (c *CustomSetting) setInvalid() {
	c.settingName = "" // set empty to represent invalid
}

func (c *CustomSetting) AdjustWithProtocol(protocol string) {
	if !c.enableJsonEdit {
		mapping, ok := settingMapping[c.settingName]
		if ok {
			c.settingName, ok = mapping[protocol]
		}
		if !ok {
			c.setInvalid()
			return
		}
	}

	if protocol == providerTypeQwen {
		c.settingName = "parameters." + c.settingName
	}
	if protocol == providerTypeGemini {
		c.settingName = "generation_config." + c.settingName
	}
}

func ReplaceByCustomSettings(body []byte, settings []CustomSetting) ([]byte, error) {
	var err error
	strBody := string(body)
	for _, setting := range settings {
		strBody, err = sjson.SetRaw(strBody, setting.settingName, setting.settingValue)
		if err != nil {
			break
		}
	}
	return []byte(strBody), err
}
