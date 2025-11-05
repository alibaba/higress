package provider

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	nameMaxTokens   = "max_tokens"
	nameTemperature = "temperature"
	nameTopP        = "top_p"
	nameTopK        = "top_k"
	nameSeed        = "seed"
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
	nameMaxTokens:   maxTokensMapping,
	nameTemperature: temperatureMapping,
	nameTopP:        topPMapping,
	nameTopK:        topKMapping,
	nameSeed:        seedMapping,
}

type CustomSetting struct {
	// @Title zh-CN 参数名称
	// @Description zh-CN 想要设置的参数的名称，例如max_tokens
	name string
	// @Title zh-CN 参数值
	// @Description zh-CN 想要设置的参数的值，例如0
	value string
	// @Title zh-CN 设置模式
	// @Description zh-CN 参数设置的模式，可以设置为"auto"或者"raw"，如果为"auto"则会根据 /plugins/wasm-go/extensions/ai-proxy/README.md中关于custom-setting部分的表格自动按照协议对参数名做改写，如果为"raw"则不会有任何改写和限制检查
	mode string
	// @Title zh-CN json edit 模式
	// @Description zh-CN 如果为false则只在用户没有设置这个参数时填充参数，否则会直接覆盖用户原有的参数设置
	overwrite bool
}

func (c *CustomSetting) FromJson(json gjson.Result) {
	c.name = json.Get("name").String()
	c.value = json.Get("value").Raw
	if obj := json.Get("mode"); obj.Exists() {
		c.mode = obj.String()
	} else {
		c.mode = "auto"
	}
	if obj := json.Get("overwrite"); obj.Exists() {
		c.overwrite = obj.Bool()
	} else {
		c.overwrite = true
	}
}

func (c *CustomSetting) Validate() bool {
	return c.name != ""
}

func (c *CustomSetting) setInvalid() {
	c.name = "" // set empty to represent invalid
}

func (c *CustomSetting) AdjustWithProtocol(protocol string) {
	if !(c.mode == "raw") {
		mapping, ok := settingMapping[c.name]
		if ok {
			c.name, ok = mapping[protocol]
		}
		if !ok {
			c.setInvalid()
			return
		}
	}

	if protocol == providerTypeQwen {
		c.name = "parameters." + c.name
	}
	if protocol == providerTypeGemini {
		c.name = "generation_config." + c.name
	}
}

func ReplaceByCustomSettings(body []byte, settings []CustomSetting) ([]byte, error) {
	var err error
	strBody := string(body)
	for _, setting := range settings {
		if !setting.overwrite && gjson.Get(strBody, setting.name).Exists() {
			continue
		}
		strBody, err = sjson.SetRaw(strBody, setting.name, setting.value)
		if err != nil {
			break
		}
	}
	return []byte(strBody), err
}
