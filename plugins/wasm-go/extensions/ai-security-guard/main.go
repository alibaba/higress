package main

import (
	"errors"
	"fmt"
	"regexp"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/request_handler"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/response_handler"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"ai-security-guard",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
		wrapper.ProcessResponseBody(onHttpResponseBody),
	)
}

const (
	OpenAIResponseFormat       = `{"id": "%s","object":"chat.completion","model":"from-security-guard","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	OpenAIStreamResponseChunk  = `data:{"id":"%s","object":"chat.completion.chunk","model":"from-security-guard","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]}`
	OpenAIStreamResponseEnd    = `data:{"id":"%s","object":"chat.completion.chunk","model":"from-security-guard","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	OpenAIStreamResponseFormat = OpenAIStreamResponseChunk + "\n\n" + OpenAIStreamResponseEnd + "\n\n" + `data: [DONE]`

	DefaultRequestCheckService       = "llm_query_moderation"
	DefaultResponseCheckService      = "llm_response_moderation"
	DefaultRequestJsonPath           = "messages.@reverse.0.content"
	DefaultResponseJsonPath          = "choices.0.message.content"
	DefaultStreamingResponseJsonPath = "choices.0.delta.content"
	DefaultDenyCode                  = 200
	DefaultDenyMessage               = "很抱歉，我无法回答您的问题"
	DefaultTimeout                   = 2000

	AliyunUserAgent = "CIPFrom/AIGateway"
	LengthLimit     = 1800
)

func parseConfig(json gjson.Result, config *cfg.AISecurityConfig) error {
	serviceName := json.Get("serviceName").String()
	servicePort := json.Get("servicePort").Int()
	serviceHost := json.Get("serviceHost").String()
	if serviceName == "" || servicePort == 0 || serviceHost == "" {
		return errors.New("invalid service config")
	}
	config.Host = serviceHost
	config.AK = json.Get("accessKey").String()
	config.SK = json.Get("secretKey").String()
	if config.AK == "" || config.SK == "" {
		return errors.New("invalid AK/SK config")
	}
	if obj := json.Get("riskLevelBar"); obj.Exists() {
		config.RiskLevelBar = obj.String()
	} else {
		config.RiskLevelBar = cfg.HighRisk
	}
	config.Token = json.Get("securityToken").String()
	if obj := json.Get("action"); obj.Exists() {
		config.Action = json.Get("action").String()
	} else {
		config.Action = "TextModerationPlus"
	}
	config.CheckRequest = json.Get("checkRequest").Bool()
	config.CheckResponse = json.Get("checkResponse").Bool()
	config.ProtocolOriginal = json.Get("protocol").String() == "original"
	config.DenyMessage = json.Get("denyMessage").String()
	if obj := json.Get("denyCode"); obj.Exists() {
		config.DenyCode = obj.Int()
	} else {
		config.DenyCode = DefaultDenyCode
	}
	if obj := json.Get("requestCheckService"); obj.Exists() {
		config.RequestCheckService = obj.String()
	} else {
		config.RequestCheckService = DefaultRequestCheckService
	}
	if obj := json.Get("responseCheckService"); obj.Exists() {
		config.ResponseCheckService = obj.String()
	} else {
		config.ResponseCheckService = DefaultResponseCheckService
	}
	if obj := json.Get("requestContentJsonPath"); obj.Exists() {
		config.RequestContentJsonPath = obj.String()
	} else {
		config.RequestContentJsonPath = DefaultRequestJsonPath
	}
	if obj := json.Get("responseContentJsonPath"); obj.Exists() {
		config.ResponseContentJsonPath = obj.String()
	} else {
		config.ResponseContentJsonPath = DefaultResponseJsonPath
	}
	if obj := json.Get("responseStreamContentJsonPath"); obj.Exists() {
		config.ResponseStreamContentJsonPath = obj.String()
	} else {
		config.ResponseStreamContentJsonPath = DefaultStreamingResponseJsonPath
	}
	if obj := json.Get("contentModerationLevelBar"); obj.Exists() {
		config.ContentModerationLevelBar = obj.String()
		if cfg.LevelToInt(config.ContentModerationLevelBar) <= 0 {
			return errors.New("invalid contentModerationLevelBar, value must be one of [max, high, medium, low]")
		}
	} else {
		config.ContentModerationLevelBar = cfg.MaxRisk
	}
	if obj := json.Get("promptAttackLevelBar"); obj.Exists() {
		config.PromptAttackLevelBar = obj.String()
		if cfg.LevelToInt(config.PromptAttackLevelBar) <= 0 {
			return errors.New("invalid promptAttackLevelBar, value must be one of [max, high, medium, low]")
		}
	} else {
		config.PromptAttackLevelBar = cfg.MaxRisk
	}
	if obj := json.Get("sensitiveDataLevelBar"); obj.Exists() {
		config.SensitiveDataLevelBar = obj.String()
		if cfg.LevelToInt(config.SensitiveDataLevelBar) <= 0 {
			return errors.New("invalid sensitiveDataLevelBar, value must be one of [S4, S3, S2, S1]")
		}
	} else {
		config.SensitiveDataLevelBar = cfg.S4Sensitive
	}
	if obj := json.Get("modelHallucinationLevelBar"); obj.Exists() {
		config.ModelHallucinationLevelBar = obj.String()
		if cfg.LevelToInt(config.ModelHallucinationLevelBar) <= 0 {
			return errors.New("invalid modelHallucinationLevelBar, value must be one of [max, high, medium, low]")
		}
	} else {
		config.ModelHallucinationLevelBar = cfg.MaxRisk
	}
	if obj := json.Get("maliciousUrlLevelBar"); obj.Exists() {
		config.MaliciousUrlLevelBar = obj.String()
		if cfg.LevelToInt(config.MaliciousUrlLevelBar) <= 0 {
			return errors.New("invalid maliciousUrlLevelBar, value must be one of [max, high, medium, low]")
		}
	} else {
		config.MaliciousUrlLevelBar = cfg.MaxRisk
	}
	if obj := json.Get("timeout"); obj.Exists() {
		config.Timeout = uint32(obj.Int())
	} else {
		config.Timeout = DefaultTimeout
	}
	if obj := json.Get("bufferLimit"); obj.Exists() {
		config.BufferLimit = int(obj.Int())
	} else {
		config.BufferLimit = 1000
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
				m["matcher"] = cfg.Matcher{Exact: fmt.Sprint(consumerName)}
			case "prefix":
				m["matcher"] = cfg.Matcher{Prefix: fmt.Sprint(consumerName)}
			case "regexp":
				m["matcher"] = cfg.Matcher{Re: regexp.MustCompile(fmt.Sprint(consumerName))}
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
				m["matcher"] = cfg.Matcher{Exact: fmt.Sprint(consumerName)}
			case "prefix":
				m["matcher"] = cfg.Matcher{Prefix: fmt.Sprint(consumerName)}
			case "regexp":
				m["matcher"] = cfg.Matcher{Re: regexp.MustCompile(fmt.Sprint(consumerName))}
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
				m["matcher"] = cfg.Matcher{Exact: fmt.Sprint(consumerName)}
			case "prefix":
				m["matcher"] = cfg.Matcher{Prefix: fmt.Sprint(consumerName)}
			case "regexp":
				m["matcher"] = cfg.Matcher{Re: regexp.MustCompile(fmt.Sprint(consumerName))}
			}
			config.ConsumerRiskLevel = append(config.ConsumerRiskLevel, m)
		}
	}
	config.Client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
		Host: serviceHost,
	})
	config.Metrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config cfg.AISecurityConfig) types.Action {
	consumer, _ := proxywasm.GetHttpRequestHeader("x-mse-consumer")
	ctx.SetContext("consumer", consumer)
	ctx.DisableReroute()
	log.Infof("config: %+v", config)
	if !config.CheckRequest {
		log.Debugf("request checking is disabled")
		ctx.DontReadRequestBody()
	}
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	log.Debugf("checking request body...")
	switch config.Action {
	case cfg.MultiModalGuard:
		return request_handler.HandleTextAndImageRequestBody(ctx, config, body)
	case cfg.TextModerationPlus:
		return request_handler.HandleTextRequestBody(ctx, config, body)
	default:
		log.Warnf("Unknown action %s", config.Action)
		return types.ActionContinue
	}
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config cfg.AISecurityConfig) types.Action {
	return response_handler.HandleTextResponseHeader(ctx, config)
}

func onHttpStreamingResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, data []byte, endOfStream bool) []byte {
	log.Debugf("checking streaming response body...")
	return response_handler.HandleTextStreamingResponseBody(ctx, config, data, endOfStream)
}

func onHttpResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	log.Debugf("checking response body...")
	return response_handler.HandleTextResponseBody(ctx, config, body)
}
