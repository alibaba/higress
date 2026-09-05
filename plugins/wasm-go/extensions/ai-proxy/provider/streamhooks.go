package provider

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/streamxform"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// 流式请求体转换的接入点。
//
// 这里回答两个问题：(1) 这个 provider + apiName + 配置能不能走流式；(2) 走流式时，
// 官方全量路径里那些"顺手做掉"的副作用（改请求头、写上下文键）由谁来补。
// 判定条件全部对照 handleRequestBody / defaultTransformRequestBody 逐行推导，
// 任何一条官方会额外动 body 的配置都直接判不适用——回落到官方路径，而不是猜。

// StreamPlan 是一个请求的流式方案。
type StreamPlan struct {
	Tr *streamxform.Transformer
	// ApplyStream：官方路径会依据 stream 改 Accept 头并写 isStreaming（默认路径为真；Qwen 兼容模式为假）。
	ApplyStream bool
	// ApplyModel：官方路径会写 originalRequestModel / finalRequestModel 上下文键。
	ApplyModel bool
}

// streamDefaultProviders 走 defaultTransformRequestBody 的 provider
// （无 TransformRequestBody*，或有但只是转调默认实现）。
var streamDefaultProviders = map[string]bool{
	providerTypeAi360: true, providerTypeBaichuan: true, providerTypeBaidu: true,
	providerTypeCloudflare: true, providerTypeDeepSeek: true, providerTypeFireworks: true,
	providerTypeGaladriel: true, providerTypeGithub: true, providerTypeGrok: true,
	providerTypeGroq: true, providerTypeMistral: true, providerTypeMoonshot: true,
	providerTypeOllama: true, providerTypeSpark: true, providerTypeStepfun: true,
	providerTypeTogetherAI: true, providerTypeYi: true,
	providerTypeVllm: true, // TransformRequestBody 只转调默认实现
}

// NewStreamPlan 为一个请求挑选流式协议。返回 nil 时 why 说明原因，调用方走官方全量路径。
func (c *ProviderConfig) NewStreamPlan(ctx wrapper.HttpContext, apiName ApiName) (plan *StreamPlan, why string) {
	if c.IsOriginal() {
		return nil, "original 协议"
	}
	if c.firstByteTimeout != 0 {
		return nil, "firstByteTimeout 需要在放行请求头前知道 stream，字段位置不可控"
	}
	if len(c.customSettings) > 0 {
		return nil, "customSettings 会改写 body"
	}
	if c.context != nil {
		return nil, "context 注入需要全量 body"
	}
	if len(c.contextCleanupCommands) > 0 {
		return nil, "contextCleanupCommands"
	}
	if c.mergeConsecutiveMessages {
		return nil, "mergeConsecutiveMessages"
	}
	if c.IsRetryOnFailureEnabled() {
		return nil, "retryOnFailure 需要把全量 body 存进上下文"
	}
	if need, _ := ctx.GetContext("needClaudeResponseConversion").(bool); need {
		return nil, "Claude 协议输入的自动转换"
	}
	if !c.isSupportedAPI(apiName) {
		return nil, "apiName 不受支持"
	}
	isChat := apiName == ApiNameChatCompletion
	mapLenient := func(m string) string { return getMappedModel(m, c.modelMapping) }
	normalize := c.IsOpenAIProtocol() && !c.IsGeneric() && (isChat || apiName == ApiNameCompletion) && !c.disableStreamUsageStats
	defaultOpts := func(v streamxform.OpenAIVariant) streamxform.OpenAIOptions {
		return streamxform.OpenAIOptions{
			MapModel:               mapLenient,
			DetectStream:           isChat,
			NormalizeUsage:         normalize,
			DeveloperRoleSupported: isDeveloperRoleSupported(c.typ),
			CheckMessages:          isChat,
			Variant:                v,
		}
	}
	inDefaultApis := isChat || apiName == ApiNameCompletion || apiName == ApiNameEmbeddings

	switch {
	case c.typ == providerTypeClaude:
		if !isChat {
			return nil, "claude 仅 chat completion 走流式"
		}
		return &StreamPlan{Tr: streamxform.NewClaude(streamxform.ClaudeOptions{
			MapModel: func(m string) (string, error) {
				if m == "" {
					return "", errors.New("missing model in request")
				}
				mapped := getMappedModel(m, c.modelMapping)
				if mapped == "" {
					return "", errors.New("model becomes empty after applying the configured mapping")
				}
				return mapped, nil
			},
			ClaudeCodeMode: c.claudeCodeMode,
		}), ApplyStream: true, ApplyModel: true}, ""

	case c.typ == providerTypeQwen:
		if !c.qwenEnableCompatible {
			return nil, "qwen 原生协议（DashScope）需要请求头承载 stream，未纳入流式"
		}
		if c.providerBasePath != "" {
			return nil, "providerBasePath 需要在 body 阶段改 :path"
		}
		if !inDefaultApis {
			return nil, "该 apiName 未纳入流式"
		}
		opts := defaultOpts(&streamxform.QwenVariant{SupportsPreserveThinking: qwenSupportsPreserveThinking})
		opts.ModelOnlyIfPresent = true
		opts.DetectStream = false // 兼容分支不调 defaultTransformRequestBody，不设 Accept / isStreaming
		return &StreamPlan{Tr: streamxform.NewOpenAI(opts), ApplyStream: false, ApplyModel: false}, ""

	case c.typ == providerTypeZhipuAi:
		if !inDefaultApis {
			return nil, "该 apiName 未纳入流式"
		}
		var v streamxform.OpenAIVariant
		if isChat {
			v = &streamxform.ZhipuVariant{}
		}
		return &StreamPlan{Tr: streamxform.NewOpenAI(defaultOpts(v)), ApplyStream: true, ApplyModel: true}, ""

	case c.typ == providerTypeOpenRouter:
		if !inDefaultApis {
			return nil, "该 apiName 未纳入流式"
		}
		var v streamxform.OpenAIVariant
		if isChat {
			v = &streamxform.OpenRouterVariant{}
		}
		return &StreamPlan{Tr: streamxform.NewOpenAI(defaultOpts(v)), ApplyStream: true, ApplyModel: true}, ""

	case c.typ == providerTypeOpenAI || c.typ == providerTypeLongcat || c.typ == providerTypeDoubao || streamDefaultProviders[c.typ]:
		if (c.typ == providerTypeOpenAI || c.typ == providerTypeLongcat) && c.responseJsonSchema != nil {
			return nil, "responseJsonSchema 会经 struct 重新序列化"
		}
		if !inDefaultApis {
			return nil, "该 apiName 的默认路径未纳入流式"
		}
		return &StreamPlan{Tr: streamxform.NewOpenAI(defaultOpts(nil)), ApplyStream: true, ApplyModel: true}, ""
	}
	return nil, "provider " + c.typ + " 的流式协议尚未实现"
}

// StreamApplyPrelude 施加官方全量路径里的副作用：
//   - 请求头 Accept: text/event-stream（仅 stream 为真；只在请求头尚未放行时有效）
//   - 上下文键 isStreaming / originalRequestModel / finalRequestModel
//
// headersMutable 为 false 说明请求头已经下发（越过提交点后），此时只写上下文键。
func (c *ProviderConfig) StreamApplyPrelude(ctx wrapper.HttpContext, apiName ApiName, plan *StreamPlan, pre streamxform.Prelude, headersMutable bool) {
	if plan.ApplyStream && pre.StreamSeen && apiName == ApiNameChatCompletion {
		if pre.Stream {
			if headersMutable {
				_ = proxywasm.ReplaceHttpRequestHeader("Accept", "text/event-stream")
			} else {
				log.Warnf("[stream-xform] stream=true 在提交点之后才出现，Accept 请求头未改写")
			}
		}
		ctx.SetContext(ctxKeyIsStreaming, pre.Stream)
	}
	if plan.ApplyModel && pre.ModelSeen {
		ctx.SetContext(ctxKeyOriginalRequestModel, pre.Model)
		ctx.SetContext(ctxKeyFinalRequestModel, getMappedModel(pre.Model, c.modelMapping))
	}
}

// StreamFinalizeContext 在整份 body 扫完后补齐"没见到"的默认值，与官方一致：
// chat 请求官方总会写 isStreaming（缺省 false）。
func (c *ProviderConfig) StreamFinalizeContext(ctx wrapper.HttpContext, apiName ApiName, plan *StreamPlan, pre streamxform.Prelude) {
	if plan.ApplyStream && apiName == ApiNameChatCompletion && !pre.StreamSeen {
		ctx.SetContext(ctxKeyIsStreaming, false)
	}
}
