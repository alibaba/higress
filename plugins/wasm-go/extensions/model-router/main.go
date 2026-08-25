package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"regexp"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	DefaultMaxBodyBytes = 100 * 1024 * 1024 // 100MB
	AutoModelPrefix     = "higress/auto"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"model-router",
		wrapper.PrePluginStartOrReload[ModelRouterConfig](ensureRuntimeIdentityRequestProperty),
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.WithRebuildMaxMemBytes[ModelRouterConfig](200*1024*1024),
	)
}

// AutoRoutingRule defines a regex-based routing rule for auto model selection
type AutoRoutingRule struct {
	Pattern *regexp.Regexp
	Model   string
}

type ModelRouterConfig struct {
	modelKey              string
	addProviderHeader     string
	modelToHeader         string
	enableOnPathSuffix    []string
	keepOriginalModelName bool
	// strictRuntimeIdentityOnly 标识该 WasmPlugin 是 C1 为单个 API 独立下发的 strict carrier。
	// 未命中 strict route 时必须 no-op，不能回落为全局 legacy model-router 并影响其他 API。
	strictRuntimeIdentityOnly bool
	// Auto routing configuration
	enableAutoRouting bool
	autoRoutingRules  []AutoRoutingRule
	defaultModel      string
	// runtimeIdentity 仅由 APIG C1 严格规则下发；nil 代表完整保留 legacy model-router 行为。
	runtimeIdentity *runtimeIdentityRule
}

func parseConfig(json gjson.Result, config *ModelRouterConfig) error {
	runtimeIdentity, err := parseRuntimeIdentityRule(json.Get("modelRuntimeIdentity"))
	if err != nil {
		return err
	}
	config.runtimeIdentity = runtimeIdentity
	config.strictRuntimeIdentityOnly = json.Get("strictRuntimeIdentityOnly").Bool()
	config.modelKey = json.Get("modelKey").String()
	if config.runtimeIdentity != nil {
		// APIGO-CONTRACT: modelcard-runtime-identity
		// strict parser 的 modelKey 只能从已发布 rule 获得，不能被共享 legacy 配置或客户请求覆盖。
		config.modelKey = config.runtimeIdentity.Parser.ModelKey
	}
	if config.modelKey == "" {
		config.modelKey = "model"
	}
	config.addProviderHeader = json.Get("addProviderHeader").String()
	config.modelToHeader = json.Get("modelToHeader").String()
	config.keepOriginalModelName = json.Get("keepOriginalModelName").Bool()

	enableOnPathSuffix := json.Get("enableOnPathSuffix")
	if enableOnPathSuffix.Exists() && enableOnPathSuffix.IsArray() {
		for _, item := range enableOnPathSuffix.Array() {
			config.enableOnPathSuffix = append(config.enableOnPathSuffix, item.String())
		}
	} else {
		// Default suffixes if not provided
		config.enableOnPathSuffix = []string{
			"/completions",
			"/embeddings",
			"/images/generations",
			"/audio/speech",
			"/fine_tuning/jobs",
			"/moderations",
			"/image-synthesis",
			"/video-synthesis",
			"/rerank",
			"/messages",
			"/responses",
		}
	}

	// Parse auto routing configuration
	autoRouting := json.Get("autoRouting")
	if autoRouting.Exists() {
		config.enableAutoRouting = autoRouting.Get("enable").Bool()
		config.defaultModel = autoRouting.Get("defaultModel").String()

		rules := autoRouting.Get("rules")
		if rules.Exists() && rules.IsArray() {
			for _, rule := range rules.Array() {
				patternStr := rule.Get("pattern").String()
				model := rule.Get("model").String()
				if patternStr == "" || model == "" {
					log.Warnf("skipping invalid auto routing rule: pattern=%s, model=%s", patternStr, model)
					continue
				}
				compiled, err := regexp.Compile(patternStr)
				if err != nil {
					log.Warnf("failed to compile regex pattern '%s': %v", patternStr, err)
					continue
				}
				config.autoRoutingRules = append(config.autoRoutingRules, AutoRoutingRule{
					Pattern: compiled,
					Model:   model,
				})
				log.Debugf("loaded auto routing rule: pattern=%s, model=%s", patternStr, model)
			}
		}
	}

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config ModelRouterConfig) types.Action {
	if config.runtimeIdentity != nil {
		return onRuntimeIdentityRequestHeaders(ctx, config)
	}
	if config.strictRuntimeIdentityOnly {
		// APIGO-CONTRACT: modelcard-runtime-identity
		// 独立 strict carrier 只能处理自身 `_match_route_` 命中的规则；未命中时不读取 Body，
		// 让原 model-router 和其它 legacy 插件保持原有全局语义。
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	path, err := proxywasm.GetHttpRequestHeader(":path")
	if err != nil {
		return types.ActionContinue
	}

	// Remove query parameters for suffix check
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	enable := false
	for _, suffix := range config.enableOnPathSuffix {
		if suffix == "*" || strings.HasSuffix(path, suffix) {
			enable = true
			break
		}
	}

	if !enable || !ctx.HasRequestBody() {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	// Prepare for body processing
	proxywasm.RemoveHttpRequestHeader("content-length")
	// 100MB buffer limit
	ctx.SetRequestBodyBufferLimit(DefaultMaxBodyBytes)

	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, config ModelRouterConfig, body []byte) types.Action {
	if config.runtimeIdentity != nil {
		return handleRuntimeIdentityJSONBody(ctx, config, body)
	}
	if config.strictRuntimeIdentityOnly {
		return types.ActionContinue
	}
	contentType, err := proxywasm.GetHttpRequestHeader("content-type")
	if err != nil {
		return types.ActionContinue
	}

	if strings.Contains(contentType, "application/json") {
		return handleJsonBody(ctx, config, body)
	} else if strings.Contains(contentType, "multipart/form-data") {
		return handleMultipartBody(ctx, config, body, contentType)
	}

	return types.ActionContinue
}

// onRuntimeIdentityRequestHeaders 为 strict Body parser 开启缓冲，并在无 Body/非 JSON 时在任何上游前拒绝。
// 输入约束：config.runtimeIdentity 已通过控制面配置校验；该分支不使用 path suffix 或 Header 选择模型。
// 输出语义：合法 JSON Body 返回 HeaderStopIteration，非法请求立即本地拒绝，legacy 分支保持原行为。
// 边界场景：C1 不支持 multipart、WebSocket 或无 Body，不能因路径恰好命中旧 suffix 而回落到 legacy parser。
func onRuntimeIdentityRequestHeaders(ctx wrapper.HttpContext, config ModelRouterConfig) types.Action {
	if !ctx.HasRequestBody() {
		return rejectRuntimeIdentityRequest(ctx, "body_required")
	}
	contentType, err := proxywasm.GetHttpRequestHeader("content-type")
	if err != nil || !strings.Contains(strings.ToLower(contentType), "application/json") {
		return rejectRuntimeIdentityRequest(ctx, "json_body_required")
	}
	if err := proxywasm.RemoveHttpRequestHeader("content-length"); err != nil {
		return rejectRuntimeIdentityRequest(ctx, "request_prepare_failed")
	}
	ctx.SetRequestBodyBufferLimit(DefaultMaxBodyBytes)
	return types.HeaderStopIteration
}

// handleRuntimeIdentityJSONBody 以精确 selector map 建立 Direct context，或把保留 Auto selector 交给最终候选 writer。
// 输入约束：body 来自 strict header 分支缓冲，model 字段只能是 parser.modelKey 对应的非空 JSON string。
// 输出语义：Direct 成功后改写为 target 上游模型并写 seq=1 property；已有合法 context 再入时保持不变；Auto 不在此处伪造身份。
// 边界场景：任何 selector/context/property 失败都在上游前 local reject，客户端字段不会覆盖已有可信 context。
func handleRuntimeIdentityJSONBody(ctx wrapper.HttpContext, config ModelRouterConfig, body []byte) types.Action {
	rule := config.runtimeIdentity
	if rule == nil || !json.Valid(body) {
		return rejectRuntimeIdentityRequest(ctx, "invalid_json_body")
	}
	selectorValue := gjson.GetBytes(body, rule.Parser.ModelKey)
	if !selectorValue.Exists() || selectorValue.Type != gjson.String || strings.TrimSpace(selectorValue.String()) == "" {
		return rejectRuntimeIdentityRequest(ctx, "model_selector_required")
	}
	selector := selectorValue.String()
	existing, err := loadResolvedModelContext()
	if err != nil {
		return rejectRuntimeIdentityRequest(ctx, "context_read_failed")
	}
	if existing != nil {
		if reason := validateResolvedModelContextForRule(existing, rule); reason != "" {
			return rejectRuntimeIdentityRequest(ctx, reason)
		}
		return types.ActionContinue
	}
	if isReservedAutoSelector(rule, selector) {
		return types.ActionContinue
	}
	target, exists := rule.SelectorTargets[selector]
	if !exists {
		return rejectRuntimeIdentityRequest(ctx, "unknown_model_selector")
	}
	updatedBody, err := sjson.SetBytes(body, rule.Parser.ModelKey, target.UpstreamModelName)
	if err != nil {
		return rejectRuntimeIdentityRequest(ctx, "model_rewrite_failed")
	}
	// APIGO-CONTRACT: modelcard-runtime-identity
	// Direct selector 通常比实际 upstream model 更长；改写 Body 后必须移除旧 Content-Length，
	// 由 Envoy 按新 body 重新计算，避免下游等待过长请求体并返回 400。
	if selector != target.UpstreamModelName {
		if err = proxywasm.RemoveHttpRequestHeader("content-length"); err != nil {
			return rejectRuntimeIdentityRequest(ctx, "request_prepare_failed")
		}
	}
	if err = proxywasm.ReplaceHttpRequestBody(updatedBody); err != nil {
		return rejectRuntimeIdentityRequest(ctx, "model_rewrite_failed")
	}
	if config.modelToHeader != "" {
		if err = proxywasm.ReplaceHttpRequestHeader(config.modelToHeader, target.UpstreamModelName); err != nil {
			return rejectRuntimeIdentityRequest(ctx, "model_rewrite_failed")
		}
	}
	if config.addProviderHeader != "" {
		// APIGO-CONTRACT: modelcard-runtime-identity
		// strict Direct 的 Provider 必须直接取自已冻结 target，不能从可能包含多个斜杠的 selector 或 modelName 反推。
		// 该服务端重写头只承载目标 Provider 的协议/插件投影；strict 主 Route 已由 Path 唯一命中，
		// 实际上游由同一冻结 target 的 x-envoy-target-cluster 决定，不能再借 Header 选择后端。
		if err = proxywasm.ReplaceHttpRequestHeader(config.addProviderHeader, target.Provider); err != nil {
			return rejectRuntimeIdentityRequest(ctx, "model_rewrite_failed")
		}
	}
	// APIGO-CONTRACT: modelcard-runtime-identity
	// strict Direct 必须用同一冻结 target 覆盖 DynamicClusterHeader 的执行 cluster；即使客户端
	// 已携带同名 Header 也不能保留，避免命中 Path 后绕开当前 ModelCard 的 Service 授权边界。
	if err = proxywasm.ReplaceHttpRequestHeader(runtimeIdentityTargetClusterHeader, target.TargetCluster); err != nil {
		return rejectRuntimeIdentityRequest(ctx, "target_cluster_rewrite_failed")
	}
	if err = writeResolvedModelContext(rule, target, 1, "direct"); err != nil {
		return rejectRuntimeIdentityRequest(ctx, "context_write_failed")
	}
	return types.ActionContinue
}

// extractLastUserMessage extracts the content of the last message with role "user" from the messages array
func extractLastUserMessage(body []byte) string {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return ""
	}

	var lastUserContent string
	for _, msg := range messages.Array() {
		if msg.Get("role").String() == "user" {
			content := msg.Get("content")
			if content.IsArray() {
				// Handle array content (e.g., multimodal messages with text and images)
				for _, item := range content.Array() {
					if item.Get("type").String() == "text" {
						lastUserContent = item.Get("text").String()
					}
				}
			} else {
				lastUserContent = content.String()
			}
		}
	}
	return lastUserContent
}

// matchAutoRoutingRule matches the user message against auto routing rules and returns the matched model
func matchAutoRoutingRule(config ModelRouterConfig, userMessage string) (string, bool) {
	for _, rule := range config.autoRoutingRules {
		if rule.Pattern.MatchString(userMessage) {
			log.Debugf("auto routing rule matched: pattern=%s, model=%s", rule.Pattern.String(), rule.Model)
			return rule.Model, true
		}
	}
	return "", false
}

func handleJsonBody(ctx wrapper.HttpContext, config ModelRouterConfig, body []byte) types.Action {
	if !json.Valid(body) {
		log.Error("invalid json body")
		return types.ActionContinue
	}
	modelValue := gjson.GetBytes(body, config.modelKey).String()
	if modelValue == "" {
		return types.ActionContinue
	}

	// Check if auto routing should be triggered
	if config.enableAutoRouting && modelValue == AutoModelPrefix {
		userMessage := extractLastUserMessage(body)
		var targetModel string
		if userMessage != "" {
			if matchedModel, found := matchAutoRoutingRule(config, userMessage); found {
				targetModel = matchedModel
				log.Infof("auto routing: user message matched, routing to model: %s", matchedModel)
			}
		}
		// No rule matched, use default model if configured
		if targetModel == "" && config.defaultModel != "" {
			targetModel = config.defaultModel
			log.Infof("auto routing: no rule matched, using default model: %s", config.defaultModel)
		}

		if targetModel != "" {
			// Set the matched model to the header for routing
			_ = proxywasm.ReplaceHttpRequestHeader("x-higress-llm-model", targetModel)
			// Update the model field in the request body
			newBody, err := sjson.SetBytes(body, config.modelKey, targetModel)
			if err != nil {
				log.Errorf("failed to update model in auto routing json body: %v", err)
				return types.ActionContinue
			}
			_ = proxywasm.ReplaceHttpRequestBody(newBody)
			log.Debugf("auto routing: updated body model field to: %s", targetModel)
		} else {
			log.Warnf("auto routing: no rule matched and no default model configured")
		}
		return types.ActionContinue
	}

	if config.modelToHeader != "" {
		_ = proxywasm.ReplaceHttpRequestHeader(config.modelToHeader, modelValue)
	}

	if config.addProviderHeader != "" {
		parts := strings.SplitN(modelValue, "/", 2)
		if len(parts) == 2 {
			provider := parts[0]
			model := parts[1]
			_ = proxywasm.ReplaceHttpRequestHeader(config.addProviderHeader, provider)

			if !config.keepOriginalModelName {
				newBody, err := sjson.SetBytes(body, config.modelKey, model)
				if err != nil {
					log.Errorf("failed to update model in json body: %v", err)
					return types.ActionContinue
				}
				_ = proxywasm.ReplaceHttpRequestBody(newBody)
			}
			log.Debugf("model route to provider: %s, model: %s", provider, model)
		} else {
			log.Debugf("model route to provider not work, model: %s", modelValue)
		}
	}

	return types.ActionContinue
}

func handleMultipartBody(ctx wrapper.HttpContext, config ModelRouterConfig, body []byte, contentType string) types.Action {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Errorf("failed to parse content type: %v", err)
		return types.ActionContinue
	}
	boundary, ok := params["boundary"]
	if !ok {
		log.Errorf("no boundary in content type")
		return types.ActionContinue
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var newBody bytes.Buffer
	writer := multipart.NewWriter(&newBody)
	writer.SetBoundary(boundary)

	modified := false

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("failed to read multipart part: %v", err)
			return types.ActionContinue
		}

		// Read part content
		partContent, err := io.ReadAll(part)
		if err != nil {
			log.Errorf("failed to read part content: %v", err)
			return types.ActionContinue
		}

		formName := part.FormName()
		if formName == config.modelKey {
			modelValue := string(partContent)

			if config.modelToHeader != "" {
				_ = proxywasm.ReplaceHttpRequestHeader(config.modelToHeader, modelValue)
			}

			if config.addProviderHeader != "" {
				parts := strings.SplitN(modelValue, "/", 2)
				if len(parts) == 2 {
					provider := parts[0]
					model := parts[1]
					_ = proxywasm.ReplaceHttpRequestHeader(config.addProviderHeader, provider)

					if !config.keepOriginalModelName {
						// Write modified part
						h := make(http.Header)
						for k, v := range part.Header {
							h[k] = v
						}

						pw, err := writer.CreatePart(textproto.MIMEHeader(h))
						if err != nil {
							log.Errorf("failed to create part: %v", err)
							return types.ActionContinue
						}
						_, err = pw.Write([]byte(model))
						if err != nil {
							log.Errorf("failed to write part content: %v", err)
							return types.ActionContinue
						}
						modified = true
						log.Debugf("model route to provider: %s, model: %s", provider, model)
						continue
					}
					log.Debugf("model route to provider: %s, model kept: %s", provider, modelValue)
				} else {
					log.Debugf("model route to provider not work, model: %s", modelValue)
				}
			}
		}

		// Write original part
		h := make(http.Header)
		for k, v := range part.Header {
			h[k] = v
		}
		pw, err := writer.CreatePart(textproto.MIMEHeader(h))
		if err != nil {
			log.Errorf("failed to create part: %v", err)
			return types.ActionContinue
		}
		_, err = pw.Write(partContent)
		if err != nil {
			log.Errorf("failed to write part content: %v", err)
			return types.ActionContinue
		}
	}

	writer.Close()

	if modified {
		_ = proxywasm.ReplaceHttpRequestBody(newBody.Bytes())
	}

	return types.ActionContinue
}
