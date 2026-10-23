package text_moderation_plus

import (
	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	common_text "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/common/text"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/text_moderation_plus/text"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func OnHttpRequestHeaders(ctx wrapper.HttpContext, config cfg.AISecurityConfig) types.Action {
	return types.ActionContinue
}

func OnHttpRequestBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	return text.HandleTextGenerationRequestBody(ctx, config, body)
}

func OnHttpResponseHeaders(ctx wrapper.HttpContext, config cfg.AISecurityConfig) types.Action {
	switch config.ApiType {
	case cfg.ApiTextGeneration:
		return common_text.HandleTextGenerationResponseHeader(ctx, config)
	default:
		log.Errorf("text_moderation_plus don't support api: %s", config.ApiType)
		return types.ActionContinue
	}
}

func OnHttpStreamingResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, data []byte, endOfStream bool) []byte {
	switch config.ApiType {
	case cfg.ApiTextGeneration:
		return common_text.HandleTextGenerationStreamingResponseBody(ctx, config, data, endOfStream)
	default:
		log.Errorf("text_moderation_plus don't support api: %s", config.ApiType)
		return data
	}
}

func OnHttpResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	switch config.ApiType {
	case cfg.ApiTextGeneration:
		return common_text.HandleTextGenerationResponseBody(ctx, config, body)
	default:
		log.Errorf("text_moderation_plus don't support api: %s", config.ApiType)
		return types.ActionContinue
	}
}
