package multi_modal_guard

import (
	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	common_text "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/common/text"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/multi_modal_guard/image"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/multi_modal_guard/text"
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
	case cfg.ApiImageGeneration:
		switch config.ProviderType {
		case cfg.ProviderOpenAI, cfg.ProviderQwen:
			return image.HandleImageGenerationResponseHeader(ctx, config)
		default:
			log.Errorf("[on response header] image generation api don't support provider: %s", config.ProviderType)
			return types.ActionContinue
		}
	default:
		log.Errorf("[on response header] multi_modal_guard don't support api: %s", config.ApiType)
		return types.ActionContinue
	}
}

func OnHttpStreamingResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, data []byte, endOfStream bool) []byte {
	switch config.ApiType {
	case cfg.ApiTextGeneration:
		return common_text.HandleTextGenerationStreamingResponseBody(ctx, config, data, endOfStream)
	default:
		log.Errorf("[on streaming response body] multi_modal_guard don't support api: %s", config.ApiType)
		return data
	}
}

func OnHttpResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	switch config.ApiType {
	case cfg.ApiTextGeneration:
		return common_text.HandleTextGenerationResponseBody(ctx, config, body)
	case cfg.ApiImageGeneration:
		switch config.ProviderType {
		case cfg.ProviderOpenAI:
			return image.HandleOpenAIImageGenerationResponseBody(ctx, config, body)
		case cfg.ProviderQwen:
			return image.HandleQwenImageGenerationResponseBody(ctx, config, body)
		default:
			log.Errorf("[on response body] image generation api don't support provider: %s", config.ProviderType)
			return types.ActionContinue
		}
	default:
		log.Errorf("[on response body] multi_modal_guard don't support api: %s", config.ApiType)
		return types.ActionContinue
	}
}
