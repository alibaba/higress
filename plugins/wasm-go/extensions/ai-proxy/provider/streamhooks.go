package provider

import (
	"errors"
	"sort"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/streamxform"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
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
	// Passthrough：官方对 body 一个字节都不动（generic），直接逐块放行，不经转换器。
	Passthrough bool
	// ApplyStream：官方路径会依据 stream 改 Accept 头并写 isStreaming
	// （默认路径的 chat / videos / videoremix 为真；Qwen 兼容模式与非流式接口为假）。
	ApplyStream bool
	// ApplyModel：官方路径会写 originalRequestModel / finalRequestModel 上下文键。
	ApplyModel bool
	// NoAcceptHeader：官方路径虽然调了 parseRequestAndMapModel，但随后用 body 阶段开始时的
	// 请求头快照整体覆盖（ReplaceRequestHeaders），Accept 的改写实际不生效——Gemini 就是这样。
	NoAcceptHeader bool
	// RequireModelBeforeCommit：放行请求头之前必须已经见到 model（请求路径依赖它）。
	// 提交点到了还没见到就回落——这是"有界前瞻，超出窗口只能回落"的落点。
	RequireModelBeforeCommit bool
	// RequireStreamBeforeCommit：同上，请求路径依赖 stream（Gemini 的 generateContent / streamGenerateContent）。
	// 整份 body 都到了还没见到则视为 false，与官方一致。
	RequireStreamBeforeCommit bool
	// AfterPrelude 在上下文键写好、请求头放行之前调用：用于依赖 body 事实改请求头（Azure 路径）。
	AfterPrelude func(ctx wrapper.HttpContext)
	// OnFinish 在整份 body 扫完后调用：写只有到末尾才能确定、且只有响应侧才用的上下文键。
	OnFinish func(ctx wrapper.HttpContext)
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
func (c *ProviderConfig) NewStreamPlan(ctx wrapper.HttpContext, apiName ApiName, prov Provider) (plan *StreamPlan, why string) {
	if c.typ == providerTypeGeneric {
		// generic 的 OnRequestBody 只是把 body 原样写回，不经 handleRequestBody：
		// 只有 main.go 里对所有 provider 生效的两项配置会碰 body / 上下文
		if len(c.customSettings) > 0 {
			return nil, "customSettings 会改写 body"
		}
		if c.IsRetryOnFailureEnabled() {
			return nil, "retryOnFailure 需要把全量 body 存进上下文"
		}
		return &StreamPlan{Passthrough: true}, ""
	}
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
	if !c.needToProcessRequestBody(apiName) {
		return nil, "该 apiName 官方不处理请求体"
	}
	if ct, _ := proxywasm.GetHttpRequestHeader("content-type"); !strings.Contains(ct, "application/json") {
		return nil, "非 JSON 请求体（multipart 等）走官方路径"
	}
	isChat := apiName == ApiNameChatCompletion
	// defaultTransformRequestBody 只对这三类接口读 stream
	detectStream := isChat || apiName == ApiNameVideos || apiName == ApiNameVideoRemix
	mapLenient := func(m string) string { return getMappedModel(m, c.modelMapping) }
	normalize := c.IsOpenAIProtocol() && !c.IsGeneric() && (isChat || apiName == ApiNameCompletion) && !c.disableStreamUsageStats
	defaultOpts := func(v streamxform.OpenAIVariant) streamxform.OpenAIOptions {
		return streamxform.OpenAIOptions{
			MapModel:               mapLenient,
			DetectStream:           detectStream,
			NormalizeUsage:         normalize,
			DeveloperRoleSupported: isDeveloperRoleSupported(c.typ),
			CheckMessages:          isChat,
			Variant:                v,
		}
	}
	defaultPlan := func(v streamxform.OpenAIVariant) *StreamPlan {
		return &StreamPlan{Tr: streamxform.NewOpenAI(defaultOpts(v)), ApplyStream: detectStream, ApplyModel: true}
	}
	// 走默认路径的 provider 里，这几类接口官方另有处理
	inDefaultApis := true
	if c.typ == providerTypeDoubao && (apiName == ApiNameResponses || apiName == ApiNameImageGeneration) {
		inDefaultApis = false
	}

	switch {
	case c.typ == providerTypeClaude && !isChat:
		// /v1/messages（原生 Claude 协议）、/v1/complete、embeddings：官方走 defaultTransformRequestBody
		return defaultPlan(nil), ""

	case c.typ == providerTypeClaude:
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

	case c.typ == providerTypeQwen && !c.qwenEnableCompatible:
		// DashScope 原生协议：官方 onChatCompletionRequestBody 在 body 阶段按 model / stream 改路径与请求头
		if !isChat {
			return nil, "qwen 原生协议仅 chat completion 走流式"
		}
		if c.providerBasePath != "" {
			return nil, "providerBasePath 需要在 body 阶段改 :path"
		}
		if len(c.qwenFileIds) > 0 {
			return nil, "qwenFileIds 要往 messages 里插入文件消息"
		}
		mapStrict := func(m string) (string, error) {
			if m == "" {
				return "", errors.New("missing model in request")
			}
			mapped := getMappedModel(m, c.modelMapping)
			if mapped == "" {
				return "", errors.New("model becomes empty after applying the configured mapping")
			}
			return mapped, nil
		}
		tr := streamxform.NewQwenNative(streamxform.QwenNativeOptions{
			MapModel:                 mapStrict,
			SupportsPreserveThinking: qwenSupportsPreserveThinking,
			EnableSearch:             c.qwenEnableSearch,
			DeveloperToSystem:        !isDeveloperRoleSupported(c.typ),
		})
		p := &StreamPlan{Tr: tr, ApplyStream: true, ApplyModel: true}
		p.RequireModelBeforeCommit = true
		p.RequireStreamBeforeCommit = true
		p.AfterPrelude = func(ctx wrapper.HttpContext) {
			// 复刻 onChatCompletionRequestBody 对请求头 / 路径的处理
			model := ctx.GetStringContext(ctxKeyFinalRequestModel, "")
			if strings.HasPrefix(model, qwenVlModelPrefixName) {
				_ = util.OverwriteRequestPath(qwenMultimodalGenerationPath)
			}
			if stream, _ := ctx.GetContext(ctxKeyIsStreaming).(bool); stream {
				_ = proxywasm.ReplaceHttpRequestHeader("Accept", "text/event-stream")
				_ = proxywasm.ReplaceHttpRequestHeader("X-DashScope-SSE", "enable")
			} else {
				_ = proxywasm.ReplaceHttpRequestHeader("Accept", "*/*")
				_ = proxywasm.RemoveHttpRequestHeader("X-DashScope-SSE")
			}
		}
		p.OnFinish = func(ctx wrapper.HttpContext) {
			if stream, _ := ctx.GetContext(ctxKeyIsStreaming).(bool); stream {
				if q, ok := tr.Protocol().(interface{ IncrementalOutput() bool }); ok {
					ctx.SetContext(ctxKeyIncrementalStreaming, q.IncrementalOutput())
				}
			}
		}
		return p, ""

	case c.typ == providerTypeQwen:
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

	case c.typ == providerTypeMinimax:
		// V2 接口（默认）：官方 handleRequestBodyByChatCompletionV2 只改 model 并把路径固定为 chatcompletion_v2；
		// Pro 接口另有一套请求结构，不走流式。
		if c.minimaxApiType == minimaxApiTypePro {
			return nil, "minimax Pro 接口未纳入流式"
		}
		if c.providerBasePath != "" {
			return nil, "providerBasePath 需要在 body 阶段改 :path"
		}
		if !isChat {
			return nil, "minimax 仅 chat completion"
		}
		opts := defaultOpts(nil)
		opts.DetectStream = false // 官方这条分支不设 Accept / isStreaming，也不写 model 上下文键
		p := &StreamPlan{Tr: streamxform.NewOpenAI(opts), ApplyStream: false, ApplyModel: false}
		p.AfterPrelude = func(ctx wrapper.HttpContext) {
			// 官方在 body 阶段才把路径改成 v2 接口（header 阶段不改）；路径固定，不依赖 body 字段
			if err := util.OverwriteRequestPath(minimaxChatCompletionV2Path); err != nil {
				log.Errorf("minimaxProvider: overwrite request path failed: %v", err)
			}
		}
		return p, ""

	case c.typ == providerTypeZhipuAi:
		if !inDefaultApis {
			return nil, "该 apiName 未纳入流式"
		}
		var v streamxform.OpenAIVariant
		if isChat {
			v = &streamxform.ZhipuVariant{}
		}
		return defaultPlan(v), ""

	case c.typ == providerTypeOpenRouter:
		if !inDefaultApis {
			return nil, "该 apiName 未纳入流式"
		}
		var v streamxform.OpenAIVariant
		if isChat {
			v = &streamxform.OpenRouterVariant{}
		}
		return defaultPlan(v), ""

	case c.typ == providerTypeGemini:
		gp, ok := prov.(*geminiProvider)
		if !ok {
			return nil, "gemini provider 实例类型异常"
		}
		if !isChat {
			return nil, "gemini 仅 chat completion 走流式"
		}
		var ss []streamxform.GeminiSafetySetting
		for k, v := range c.geminiSafetySetting {
			ss = append(ss, streamxform.GeminiSafetySetting{Category: k, Threshold: v})
		}
		sort.Slice(ss, func(i, j int) bool { return ss[i].Category < ss[j].Category })
		mapStrict := func(m string) (string, error) {
			if m == "" {
				return "", errors.New("missing model in request")
			}
			mapped := getMappedModel(m, c.modelMapping)
			if mapped == "" {
				return "", errors.New("model becomes empty after applying the configured mapping")
			}
			return mapped, nil
		}
		p := &StreamPlan{Tr: streamxform.NewGemini(streamxform.GeminiOptions{
			MapModel:       mapStrict,
			ThinkingModel:  func(m string) bool { return geminiThinkingModels[m] },
			ThinkingBudget: c.geminiThinkingBudget,
			SafetySettings: ss,
		}), ApplyStream: true, ApplyModel: true, NoAcceptHeader: true}
		// 官方 onChatCompletionRequestBody：路径 = /{version}/models/{映射后 model}:{generateContent|streamGenerateContent}
		p.RequireModelBeforeCommit = true
		p.RequireStreamBeforeCommit = true
		p.AfterPrelude = func(ctx wrapper.HttpContext) {
			model := ctx.GetStringContext(ctxKeyFinalRequestModel, "")
			stream, _ := ctx.GetContext(ctxKeyIsStreaming).(bool)
			if err := util.OverwriteRequestPath(gp.getRequestPath(ApiNameChatCompletion, model, stream)); err != nil {
				log.Errorf("geminiProvider: overwrite request path failed: %v", err)
			}
		}
		return p, ""

	case c.typ == providerTypeAzure:
		ap, ok := prov.(*azureProvider)
		if !ok {
			return nil, "azure provider 实例类型异常"
		}
		if !inDefaultApis {
			return nil, "该 apiName 未纳入流式"
		}
		// 官方 TransformRequestBody：默认转换后再按上下文里的最终 model 改写 :path。
		// serviceUrl 里没有部署名（DomainOnly / OpenAI v1 base）时路径含 {model} 占位，
		// 必须在放行请求头前知道 model；其余两种形态路径与 body 无关。
		p := defaultPlan(nil)
		p.RequireModelBeforeCommit = !azureModelIrrelevantApis[apiName] &&
			(ap.serviceUrlType == azureServiceUrlTypeDomainOnly || ap.serviceUrlType == azureServiceUrlTypeOpenAIV1Base)
		p.AfterPrelude = func(ctx wrapper.HttpContext) {
			if path := ap.transformRequestPath(ctx, apiName); path != "" {
				if err := util.OverwriteRequestPath(path); err != nil {
					log.Errorf("azureProvider: overwrite request path to %s failed: %v", path, err)
				}
			}
		}
		return p, ""

	case c.typ == providerTypeOpenAI || c.typ == providerTypeLongcat || c.typ == providerTypeDoubao || streamDefaultProviders[c.typ]:
		if (c.typ == providerTypeOpenAI || c.typ == providerTypeLongcat) && c.responseJsonSchema != nil {
			return nil, "responseJsonSchema 会经 struct 重新序列化"
		}
		if !inDefaultApis {
			return nil, "该 apiName 的默认路径未纳入流式"
		}
		return defaultPlan(nil), ""
	}
	return nil, "provider " + c.typ + " 的流式协议尚未实现"
}

// StreamApplyPrelude 施加官方全量路径里的副作用：
//   - 请求头 Accept: text/event-stream（仅 stream 为真；只在请求头尚未放行时有效）
//   - 上下文键 isStreaming / originalRequestModel / finalRequestModel
//
// headersMutable 为 false 说明请求头已经下发（越过提交点后），此时只写上下文键。
func (c *ProviderConfig) StreamApplyPrelude(ctx wrapper.HttpContext, apiName ApiName, plan *StreamPlan, pre streamxform.Prelude, headersMutable bool) {
	if plan.ApplyStream && pre.StreamSeen {
		if pre.Stream && !plan.NoAcceptHeader {
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
	if plan.ApplyStream && !pre.StreamSeen {
		ctx.SetContext(ctxKeyIsStreaming, false)
	}
}
