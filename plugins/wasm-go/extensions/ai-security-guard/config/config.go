package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	MaxRisk    = "max"
	HighRisk   = "high"
	MediumRisk = "medium"
	LowRisk    = "low"
	NoRisk     = "none"

	S4Sensitive = "s4"
	S3Sensitive = "s3"
	S2Sensitive = "s2"
	S1Sensitive = "s1"
	NoSensitive = "s0"

	ContentModerationType      = "contentModeration"
	PromptAttackType           = "promptAttack"
	SensitiveDataType          = "sensitiveData"
	MaliciousUrlDataType       = "maliciousUrl"
	ModelHallucinationDataType = "modelHallucination"

	// Default configurations
	OpenAIResponseFormat       = `{"id": "%s","object":"chat.completion","model":"from-security-guard","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	OpenAIStreamResponseChunk  = `data:{"id":"%s","object":"chat.completion.chunk","model":"from-security-guard","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]}`
	OpenAIStreamResponseEnd    = `data:{"id":"%s","object":"chat.completion.chunk","model":"from-security-guard","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	OpenAIStreamResponseFormat = OpenAIStreamResponseChunk + "\n\n" + OpenAIStreamResponseEnd + "\n\n" + `data: [DONE]`

	DefaultDenyCode    = 200
	DefaultDenyMessage = "很抱歉，我无法回答您的问题"
	DefaultTimeout     = 2000

	AliyunUserAgent = "CIPFrom/AIGateway"
	LengthLimit     = 1800

	DefaultRequestCheckService       = "llm_query_moderation"
	DefaultResponseCheckService      = "llm_response_moderation"
	DefaultRequestJsonPath           = "messages.@reverse.0.content"
	DefaultResponseJsonPath          = "choices.0.message.content"
	DefaultStreamingResponseJsonPath = "choices.0.delta.content"

	// Actions
	MultiModalGuard          = "MultiModalGuard"
	MultiModalGuardForBase64 = "MultiModalGuardForBase64"
	TextModerationPlus       = "TextModerationPlus"

	// Services
	DefaultMultiModalGuardTextInputCheckService  = "query_security_check"
	DefaultMultiModalGuardTextOutputCheckService = "response_security_check"
	DefaultMultiModalGuardImageInputCheckService = "img_query_security_check"

	DefaultTextModerationPlusTextInputCheckService  = "llm_query_moderation"
	DefaultTextModerationPlusTextOutputCheckService = "llm_response_moderation"
)

// api types

const (
	ApiTextGeneration  = "text_generation"
	ApiImageGeneration = "image_generation"
	ApiMCP             = "mcp"
)

// provider types
const (
	ProviderOpenAI  = "openai"
	ProviderQwen    = "qwen"
	ProviderComfyUI = "comfyui"
)

type Response struct {
	Code      int    `json:"Code"`
	Message   string `json:"Message"`
	RequestId string `json:"RequestId"`
	Data      Data   `json:"Data"`
}

type Data struct {
	RiskLevel   string   `json:"RiskLevel,omitempty"`
	AttackLevel string   `json:"AttackLevel,omitempty"`
	Result      []Result `json:"Result,omitempty"`
	Advice      []Advice `json:"Advice,omitempty"`
	Detail      []Detail `json:"Detail,omitempty"`
}

type Result struct {
	RiskWords   string  `json:"RiskWords,omitempty"`
	Description string  `json:"Description,omitempty"`
	Confidence  float64 `json:"Confidence,omitempty"`
	Label       string  `json:"Label,omitempty"`
}

type Advice struct {
	Answer     string `json:"Answer,omitempty"`
	HitLabel   string `json:"HitLabel,omitempty"`
	HitLibName string `json:"HitLibName,omitempty"`
}

type Detail struct {
	Suggestion string `json:"Suggestion,omitempty"`
	Type       string `json:"Type,omitempty"`
	Level      string `json:"Level,omitempty"`
}

type Matcher struct {
	Exact  string
	Prefix string
	Re     *regexp.Regexp
}

func (m *Matcher) match(consumer string) bool {
	if m.Exact != "" {
		return consumer == m.Exact
	} else if m.Prefix != "" {
		return strings.HasPrefix(consumer, m.Prefix)
	} else if m.Re != nil {
		return m.Re.MatchString(consumer)
	} else {
		return false
	}
}

type AISecurityConfig struct {
	Client                        wrapper.HttpClient
	Host                          string
	AK                            string
	SK                            string
	Token                         string
	Action                        string
	CheckRequest                  bool
	CheckRequestImage             bool
	RequestCheckService           string
	RequestImageCheckService      string
	RequestContentJsonPath        string
	CheckResponse                 bool
	ResponseCheckService          string
	ResponseImageCheckService     string
	ResponseContentJsonPath       string
	ResponseStreamContentJsonPath string
	DenyCode                      int64
	DenyMessage                   string
	ProtocolOriginal              bool
	RiskLevelBar                  string
	ContentModerationLevelBar     string
	PromptAttackLevelBar          string
	SensitiveDataLevelBar         string
	MaliciousUrlLevelBar          string
	ModelHallucinationLevelBar    string
	Timeout                       uint32
	BufferLimit                   int
	Metrics                       map[string]proxywasm.MetricCounter
	ConsumerRequestCheckService   []map[string]interface{}
	ConsumerResponseCheckService  []map[string]interface{}
	ConsumerRiskLevel             []map[string]interface{}
	// text_generation, image_generation, etc.
	ApiType string
	// openai, qwen, comfyui, etc.
	ProviderType string
}

func (config *AISecurityConfig) Parse(json gjson.Result) error {
	serviceName := json.Get("serviceName").String()
	servicePort := json.Get("servicePort").Int()
	serviceHost := json.Get("serviceHost").String()
	config.Host = serviceHost
	if serviceName == "" || servicePort == 0 || serviceHost == "" {
		return errors.New("invalid service config")
	}
	config.AK = json.Get("accessKey").String()
	config.SK = json.Get("secretKey").String()
	if config.AK == "" || config.SK == "" {
		return errors.New("invalid AK/SK config")
	}
	config.Token = json.Get("securityToken").String()
	// set action
	if obj := json.Get("action"); obj.Exists() {
		config.Action = json.Get("action").String()
	} else {
		config.Action = TextModerationPlus
	}
	// set default values
	config.SetDefaultValues()
	// set values
	if obj := json.Get("riskLevelBar"); obj.Exists() {
		config.RiskLevelBar = obj.String()
	}
	if obj := json.Get("requestCheckService"); obj.Exists() {
		config.RequestCheckService = obj.String()
	}
	if obj := json.Get("requestImageCheckService"); obj.Exists() {
		config.RequestImageCheckService = obj.String()
	}
	if obj := json.Get("responseCheckService"); obj.Exists() {
		config.ResponseCheckService = obj.String()
	}
	if obj := json.Get("responseImageCheckService"); obj.Exists() {
		config.ResponseImageCheckService = obj.String()
	}
	config.CheckRequest = json.Get("checkRequest").Bool()
	config.CheckRequestImage = json.Get("checkRequestImage").Bool()
	config.CheckResponse = json.Get("checkResponse").Bool()
	config.ProtocolOriginal = json.Get("protocol").String() == "original"
	config.DenyMessage = json.Get("denyMessage").String()
	if obj := json.Get("denyCode"); obj.Exists() {
		config.DenyCode = obj.Int()
	}
	if obj := json.Get("requestContentJsonPath"); obj.Exists() {
		config.RequestContentJsonPath = obj.String()
	}
	if obj := json.Get("responseContentJsonPath"); obj.Exists() {
		config.ResponseContentJsonPath = obj.String()
	}
	if obj := json.Get("responseStreamContentJsonPath"); obj.Exists() {
		config.ResponseStreamContentJsonPath = obj.String()
	}
	if obj := json.Get("contentModerationLevelBar"); obj.Exists() {
		config.ContentModerationLevelBar = obj.String()
		if LevelToInt(config.ContentModerationLevelBar) <= 0 {
			return errors.New("invalid contentModerationLevelBar, value must be one of [max, high, medium, low]")
		}
	}
	if obj := json.Get("promptAttackLevelBar"); obj.Exists() {
		config.PromptAttackLevelBar = obj.String()
		if LevelToInt(config.PromptAttackLevelBar) <= 0 {
			return errors.New("invalid promptAttackLevelBar, value must be one of [max, high, medium, low]")
		}
	}
	if obj := json.Get("sensitiveDataLevelBar"); obj.Exists() {
		config.SensitiveDataLevelBar = obj.String()
		if LevelToInt(config.SensitiveDataLevelBar) <= 0 {
			return errors.New("invalid sensitiveDataLevelBar, value must be one of [S4, S3, S2, S1]")
		}
	}
	if obj := json.Get("modelHallucinationLevelBar"); obj.Exists() {
		config.ModelHallucinationLevelBar = obj.String()
		if LevelToInt(config.ModelHallucinationLevelBar) <= 0 {
			return errors.New("invalid modelHallucinationLevelBar, value must be one of [max, high, medium, low]")
		}
	}
	if obj := json.Get("maliciousUrlLevelBar"); obj.Exists() {
		config.MaliciousUrlLevelBar = obj.String()
		if LevelToInt(config.MaliciousUrlLevelBar) <= 0 {
			return errors.New("invalid maliciousUrlLevelBar, value must be one of [max, high, medium, low]")
		}
	}
	if obj := json.Get("timeout"); obj.Exists() {
		config.Timeout = uint32(obj.Int())
	}
	if obj := json.Get("bufferLimit"); obj.Exists() {
		config.BufferLimit = int(obj.Int())
	}
	if obj := json.Get("consumerRequestCheckService"); obj.Exists() {
		for _, item := range json.Get("consumerRequestCheckService").Array() {
			m := make(map[string]interface{})
			for k, v := range item.Map() {
				m[k] = v.Value()
			}
			consumerName, ok1 := m["name"]
			matchType, ok2 := m["matchType"]
			if !ok1 || !ok2 {
				continue
			}
			switch fmt.Sprint(matchType) {
			case "exact":
				m["matcher"] = Matcher{Exact: fmt.Sprint(consumerName)}
			case "prefix":
				m["matcher"] = Matcher{Prefix: fmt.Sprint(consumerName)}
			case "regexp":
				m["matcher"] = Matcher{Re: regexp.MustCompile(fmt.Sprint(consumerName))}
			}
			config.ConsumerRequestCheckService = append(config.ConsumerRequestCheckService, m)
		}
	}
	if obj := json.Get("consumerResponseCheckService"); obj.Exists() {
		for _, item := range json.Get("consumerResponseCheckService").Array() {
			m := make(map[string]interface{})
			for k, v := range item.Map() {
				m[k] = v.Value()
			}
			consumerName, ok1 := m["name"]
			matchType, ok2 := m["matchType"]
			if !ok1 || !ok2 {
				continue
			}
			switch fmt.Sprint(matchType) {
			case "exact":
				m["matcher"] = Matcher{Exact: fmt.Sprint(consumerName)}
			case "prefix":
				m["matcher"] = Matcher{Prefix: fmt.Sprint(consumerName)}
			case "regexp":
				m["matcher"] = Matcher{Re: regexp.MustCompile(fmt.Sprint(consumerName))}
			}
			config.ConsumerResponseCheckService = append(config.ConsumerResponseCheckService, m)
		}
	}
	if obj := json.Get("consumerRiskLevel"); obj.Exists() {
		for _, item := range json.Get("consumerRiskLevel").Array() {
			m := make(map[string]interface{})
			for k, v := range item.Map() {
				m[k] = v.Value()
			}
			consumerName, ok1 := m["name"]
			matchType, ok2 := m["matchType"]
			if !ok1 || !ok2 {
				continue
			}
			switch fmt.Sprint(matchType) {
			case "exact":
				m["matcher"] = Matcher{Exact: fmt.Sprint(consumerName)}
			case "prefix":
				m["matcher"] = Matcher{Prefix: fmt.Sprint(consumerName)}
			case "regexp":
				m["matcher"] = Matcher{Re: regexp.MustCompile(fmt.Sprint(consumerName))}
			}
			config.ConsumerRiskLevel = append(config.ConsumerRiskLevel, m)
		}
	}
	if obj := json.Get("apiType"); obj.Exists() {
		config.ApiType = obj.String()
	}
	if obj := json.Get("providerType"); obj.Exists() {
		config.ProviderType = obj.String()
	}
	config.Client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
		Host: serviceHost,
	})
	config.Metrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func (config *AISecurityConfig) SetDefaultValues() {
	switch config.Action {
	case TextModerationPlus:
		config.RequestCheckService = DefaultTextModerationPlusTextInputCheckService
		config.ResponseCheckService = DefaultTextModerationPlusTextOutputCheckService
	case MultiModalGuard:
		config.RequestCheckService = DefaultMultiModalGuardTextInputCheckService
		config.RequestImageCheckService = DefaultMultiModalGuardImageInputCheckService
		config.ResponseCheckService = DefaultMultiModalGuardTextOutputCheckService
	}
	config.RiskLevelBar = HighRisk
	config.DenyCode = DefaultDenyCode
	config.RequestContentJsonPath = DefaultRequestJsonPath
	config.ResponseContentJsonPath = DefaultResponseJsonPath
	config.ResponseStreamContentJsonPath = DefaultStreamingResponseJsonPath
	config.ContentModerationLevelBar = MaxRisk
	config.PromptAttackLevelBar = MaxRisk
	config.SensitiveDataLevelBar = S4Sensitive
	config.ModelHallucinationLevelBar = MaxRisk
	config.MaliciousUrlLevelBar = MaxRisk
	config.Timeout = DefaultTimeout
	config.BufferLimit = 1000
	config.ApiType = ApiTextGeneration
	config.ProviderType = ProviderOpenAI
}

func (config *AISecurityConfig) IncrementCounter(metricName string, inc uint64) {
	counter, ok := config.Metrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.Metrics[metricName] = counter
	}
	counter.Increment(inc)
}

func (config *AISecurityConfig) GetRequestCheckService(consumer string) string {
	result := config.RequestCheckService
	for _, obj := range config.ConsumerRequestCheckService {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if requestCheckService, ok := obj["requestCheckService"]; ok {
					result, _ = requestCheckService.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetRequestImageCheckService(consumer string) string {
	result := config.RequestImageCheckService
	for _, obj := range config.ConsumerRequestCheckService {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if requestCheckService, ok := obj["requestImageCheckService"]; ok {
					result, _ = requestCheckService.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetResponseCheckService(consumer string) string {
	result := config.ResponseCheckService
	for _, obj := range config.ConsumerResponseCheckService {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if responseCheckService, ok := obj["responseCheckService"]; ok {
					result, _ = responseCheckService.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetResponseImageCheckService(consumer string) string {
	result := config.ResponseImageCheckService
	for _, obj := range config.ConsumerResponseCheckService {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if responseCheckService, ok := obj["responseImageCheckService"]; ok {
					result, _ = responseCheckService.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetRiskLevelBar(consumer string) string {
	result := config.RiskLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if riskLevelBar, ok := obj["riskLevelBar"]; ok {
					result, _ = riskLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetContentModerationLevelBar(consumer string) string {
	result := config.ContentModerationLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if contentModerationLevelBar, ok := obj["contentModerationLevelBar"]; ok {
					result, _ = contentModerationLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetPromptAttackLevelBar(consumer string) string {
	result := config.PromptAttackLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if promptAttackLevelBar, ok := obj["promptAttackLevelBar"]; ok {
					result, _ = promptAttackLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetSensitiveDataLevelBar(consumer string) string {
	result := config.SensitiveDataLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if sensitiveDataLevelBar, ok := obj["sensitiveDataLevelBar"]; ok {
					result, _ = sensitiveDataLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetMaliciousUrlLevelBar(consumer string) string {
	result := config.MaliciousUrlLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if maliciousUrlLevelBar, ok := obj["maliciousUrlLevelBar"]; ok {
					result, _ = maliciousUrlLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func (config *AISecurityConfig) GetModelHallucinationLevelBar(consumer string) string {
	result := config.ModelHallucinationLevelBar
	for _, obj := range config.ConsumerRiskLevel {
		if matcher, ok := obj["matcher"].(Matcher); ok {
			if matcher.match(consumer) {
				if modelHallucinationLevelBar, ok := obj["modelHallucinationLevelBar"]; ok {
					result, _ = modelHallucinationLevelBar.(string)
				}
				break
			}
		}
	}
	return result
}

func LevelToInt(riskLevel string) int {
	// First check against our defined constants
	switch strings.ToLower(riskLevel) {
	case MaxRisk, S4Sensitive:
		return 4
	case HighRisk, S3Sensitive:
		return 3
	case MediumRisk, S2Sensitive:
		return 2
	case LowRisk, S1Sensitive:
		return 1
	case NoRisk, NoSensitive:
		return 0
	default:
		return -1
	}
}

func IsRiskLevelAcceptable(action string, data Data, config AISecurityConfig, consumer string) bool {
	if action == MultiModalGuard || action == MultiModalGuardForBase64 {
		// Check top-level risk levels for MultiModalGuard
		if LevelToInt(data.RiskLevel) >= LevelToInt(config.GetContentModerationLevelBar(consumer)) {
			return false
		}
		// Also check AttackLevel for prompt attack detection
		if LevelToInt(data.AttackLevel) >= LevelToInt(config.GetPromptAttackLevelBar(consumer)) {
			return false
		}

		// Check detailed results for backward compatibility
		for _, detail := range data.Detail {
			switch detail.Type {
			case ContentModerationType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetContentModerationLevelBar(consumer)) {
					return false
				}
			case PromptAttackType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetPromptAttackLevelBar(consumer)) {
					return false
				}
			case SensitiveDataType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetSensitiveDataLevelBar(consumer)) {
					return false
				}
			case MaliciousUrlDataType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetMaliciousUrlLevelBar(consumer)) {
					return false
				}
			case ModelHallucinationDataType:
				if LevelToInt(detail.Level) >= LevelToInt(config.GetModelHallucinationLevelBar(consumer)) {
					return false
				}
			}
		}
		return true
	} else {
		return LevelToInt(data.RiskLevel) < LevelToInt(config.GetRiskLevelBar(consumer))
	}
}
