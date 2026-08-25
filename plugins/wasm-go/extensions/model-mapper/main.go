package main

import (
	"encoding/json"
	"errors"
	"sort"
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
)

func main() {}

func init() {
	wrapper.SetCtx(
		"model-mapper",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.WithRebuildMaxMemBytes[Config](200*1024*1024),
	)
}

type ModelMapping struct {
	Prefix string
	Target string
}

type Config struct {
	modelKey           string
	exactModelMapping  map[string]string
	prefixModelMapping []ModelMapping
	defaultModel       string
	enableOnPathSuffix []string
	modelToHeader      string
	// runtimeIdentity 仅由 APIG C1 的冻结 projection 下发；nil 时完整保留 legacy mapper 语义。
	runtimeIdentity *runtimeIdentityRule
	// runtimeTarget 与 runtimeTargetKey 是当前映射规则的唯一跨卡身份目标，禁止由请求模型名反推。
	runtimeTarget    *runtimeIdentityTarget
	runtimeTargetKey string
}

func parseConfig(json gjson.Result, config *Config) error {
	runtimeIdentity, runtimeTarget, runtimeTargetKey, err := parseMapperRuntimeIdentity(
		json.Get("modelRuntimeIdentity"),
		json.Get("modelRuntimeIdentityTarget"),
		json.Get("modelRuntimeIdentityTargetKey"),
	)
	if err != nil {
		return err
	}
	config.runtimeIdentity = runtimeIdentity
	config.runtimeTarget = runtimeTarget
	config.runtimeTargetKey = runtimeTargetKey
	config.modelKey = json.Get("modelKey").String()
	if config.runtimeIdentity != nil {
		// APIGO-CONTRACT: modelcard-runtime-identity
		// strict parser 的模型字段必须来自同一冻结 rule，不能被 mapper 的 legacy 默认值覆盖。
		config.modelKey = config.runtimeIdentity.Parser.ModelKey
	}
	if config.modelKey == "" {
		config.modelKey = "model"
	}

	config.modelToHeader = json.Get("modelToHeader").String()
	if config.modelToHeader == "" {
		config.modelToHeader = "x-higress-llm-model-final"
	}

	modelMapping := json.Get("modelMapping")
	if modelMapping.Exists() && !modelMapping.IsObject() {
		return errors.New("modelMapping must be an object")
	}

	config.exactModelMapping = make(map[string]string)
	config.prefixModelMapping = make([]ModelMapping, 0)

	// To replicate C++ behavior (nlohmann::json iterates keys alphabetically),
	// we collect entries and sort them by key.
	type mappingEntry struct {
		key   string
		value string
	}
	var entries []mappingEntry
	modelMapping.ForEach(func(key, value gjson.Result) bool {
		entries = append(entries, mappingEntry{
			key:   key.String(),
			value: value.String(),
		})
		return true
	})
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	for _, entry := range entries {
		key := entry.key
		value := entry.value
		if key == "*" {
			config.defaultModel = value
		} else if strings.HasSuffix(key, "*") {
			prefix := strings.TrimSuffix(key, "*")
			config.prefixModelMapping = append(config.prefixModelMapping, ModelMapping{
				Prefix: prefix,
				Target: value,
			})
		} else {
			config.exactModelMapping[key] = value
		}
	}

	enableOnPathSuffix := json.Get("enableOnPathSuffix")
	if enableOnPathSuffix.Exists() {
		if !enableOnPathSuffix.IsArray() {
			return errors.New("enableOnPathSuffix must be an array")
		}
		for _, item := range enableOnPathSuffix.Array() {
			config.enableOnPathSuffix = append(config.enableOnPathSuffix, item.String())
		}
	} else {
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

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config) types.Action {
	if config.runtimeIdentity != nil {
		return onRuntimeIdentityRequestHeaders(ctx, config)
	}
	// Check path suffix
	path, err := proxywasm.GetHttpRequestHeader(":path")
	if err != nil {
		return types.ActionContinue
	}

	// Strip query parameters
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	matched := false
	for _, suffix := range config.enableOnPathSuffix {
		if strings.HasSuffix(path, suffix) {
			matched = true
			break
		}
	}

	if !matched || !ctx.HasRequestBody() {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	// Disable re-route since the plugin may modify some headers related to the chosen route.
	ctx.DisableReroute()
	// Prepare for body processing
	proxywasm.RemoveHttpRequestHeader("content-length")
	// 100MB buffer limit
	ctx.SetRequestBodyBufferLimit(DefaultMaxBodyBytes)

	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, config Config, body []byte) types.Action {
	if config.runtimeIdentity != nil {
		return handleRuntimeIdentityMapping(ctx, config, body)
	}
	if len(body) == 0 {
		return types.ActionContinue
	}

	if !json.Valid(body) {
		log.Error("invalid json body")
		return types.ActionContinue
	}

	oldModel := gjson.GetBytes(body, config.modelKey).String()

	newModel := config.defaultModel
	if newModel == "" {
		newModel = oldModel
	}

	// Exact match
	if target, ok := config.exactModelMapping[oldModel]; ok {
		newModel = target
	} else {
		// Prefix match
		for _, mapping := range config.prefixModelMapping {
			if strings.HasPrefix(oldModel, mapping.Prefix) {
				newModel = mapping.Target
				break
			}
		}
	}

	// update x-higress-llm-model-final header
	proxywasm.ReplaceHttpRequestHeader(config.modelToHeader, newModel)
	log.Debugf("set header %s: %s", config.modelToHeader, newModel)
	if newModel != "" && newModel != oldModel {
		newBody, err := sjson.SetBytes(body, config.modelKey, newModel)
		if err != nil {
			log.Errorf("failed to update model: %v", err)
			return types.ActionContinue
		}
		proxywasm.ReplaceHttpRequestBody(newBody)
		log.Debugf("model mapped, before: %s, after: %s", oldModel, newModel)
	}

	return types.ActionContinue
}

// onRuntimeIdentityRequestHeaders 为 strict mapper 配置准备唯一允许的 JSON Body，并阻止 legacy suffix 规则降级。
// 输入约束：runtimeIdentity/runtimeTarget 已由同一发布快照完整验证；客户端 Header 不是身份来源。
// 输出语义：合法请求只进入 Body 检查；是否执行映射及 DisableReroute 必须等 Body 中的 Auto/Direct 事实确定后再决定。
// 边界场景：reserved Auto 在 ai-auto-router 建立 seq=1 前必须无副作用透传；普通 mapper/fallback 仍要求已有 context。
func onRuntimeIdentityRequestHeaders(ctx wrapper.HttpContext, config Config) types.Action {
	if config.runtimeIdentity == nil || config.runtimeTarget == nil {
		return rejectRuntimeIdentityRequest(ctx, "invalid_runtime_identity_config")
	}
	if !ctx.HasRequestBody() {
		return rejectRuntimeIdentityRequest(ctx, "body_required")
	}
	contentType, err := proxywasm.GetHttpRequestHeader("content-type")
	if err != nil || !strings.Contains(strings.ToLower(contentType), "application/json") {
		return rejectRuntimeIdentityRequest(ctx, "json_body_required")
	}
	ctx.SetRequestBodyBufferLimit(DefaultMaxBodyBytes)
	return types.HeaderStopIteration
}

// handleRuntimeIdentityMapping 在 Body/Header 改写全部成功后，提交 mapper 或 fallback 的下一代可信身份。
// 输入约束：existing context 必须由上游 Direct/Auto writer 写入且与当前冻结 rule 同 scope/revision；target 只能来自 compiler closure。
// 输出语义：跨卡写入 sequence+1，同卡仅改写表示而保持 sequence；任一冲突或 hostcall 失败均在上游前拒绝。
// 边界场景：不得通过旧模型名、service matcher 或客户端字段恢复身份，避免同名 ModelCard 的错误授权继承。
func handleRuntimeIdentityMapping(ctx wrapper.HttpContext, config Config, body []byte) types.Action {
	rule := config.runtimeIdentity
	target := config.runtimeTarget
	if rule == nil || target == nil || len(body) == 0 || !json.Valid(body) {
		return rejectRuntimeIdentityRequest(ctx, "invalid_json_body")
	}
	modelValue := gjson.GetBytes(body, rule.Parser.ModelKey)
	if !modelValue.Exists() || modelValue.Type != gjson.String || strings.TrimSpace(modelValue.String()) == "" {
		return rejectRuntimeIdentityRequest(ctx, "model_selector_required")
	}
	existing, err := loadResolvedModelContext()
	if err != nil {
		return rejectRuntimeIdentityRequest(ctx, "context_read_failed")
	}
	oldModel := modelValue.String()
	if existing == nil && isReservedAutoSelector(rule, oldModel) && mapsModelToPassthrough(config, oldModel) {
		// APIGO-CONTRACT: modelcard-runtime-identity
		// Auto 的首次可信身份只能由掌握最终 dispatched candidate 的 ai-auto-router 建立。
		// 此处既不禁用 reroute，也不改 Body/Header/property，避免 attached ModelMapper 在 seq=1 前截断 Auto 链路。
		return types.ActionContinue
	}
	if reason := validateResolvedModelContextForRule(existing, rule); reason != "" {
		return rejectRuntimeIdentityRequest(ctx, reason)
	}

	newModel := mappedModel(config, oldModel)
	// APIGO-CONTRACT: modelcard-runtime-identity
	// 每条 strict mapper/fallback rule 都只能实际改写到 compiler 已解析的 target，避免运行期把字符串映射成另一张卡。
	if newModel != target.UpstreamModelName {
		return rejectRuntimeIdentityRequest(ctx, "target_model_mismatch")
	}
	// APIGO-CONTRACT: modelcard-runtime-identity
	// 只有已经确认要执行的严格跨卡规则才能冻结当前路由；Auto passthrough 已在上方无副作用返回。
	ctx.DisableReroute()
	if config.modelToHeader != "" {
		if err = proxywasm.ReplaceHttpRequestHeader(config.modelToHeader, newModel); err != nil {
			return rejectRuntimeIdentityRequest(ctx, "model_rewrite_failed")
		}
	}
	if newModel != oldModel {
		if err = proxywasm.RemoveHttpRequestHeader("content-length"); err != nil {
			return rejectRuntimeIdentityRequest(ctx, "request_prepare_failed")
		}
		updatedBody, rewriteErr := sjson.SetBytes(body, rule.Parser.ModelKey, newModel)
		if rewriteErr != nil {
			return rejectRuntimeIdentityRequest(ctx, "model_rewrite_failed")
		}
		if err = proxywasm.ReplaceHttpRequestBody(updatedBody); err != nil {
			return rejectRuntimeIdentityRequest(ctx, "model_rewrite_failed")
		}
	}
	// APIGO-CONTRACT: modelcard-runtime-identity
	// 无论映射是否改写模型文本，都以当前冻结 target 覆盖执行 cluster。这样跨卡 fallback
	// 与同卡表示改写均不会继承客户端伪造或前一跳残留的 x-envoy-target-cluster。
	if err = proxywasm.ReplaceHttpRequestHeader(runtimeIdentityTargetClusterHeader, target.TargetCluster); err != nil {
		return rejectRuntimeIdentityRequest(ctx, "target_cluster_rewrite_failed")
	}
	if existing.ResolvedModelCardID == target.ModelCardID {
		return types.ActionContinue
	}
	source := "mapper"
	if strings.HasPrefix(config.runtimeTargetKey, "fallback:") {
		source = "fallback"
	}
	if err = writeResolvedModelContext(rule, *target, existing.TransitionSeq+1, source); err != nil {
		return rejectRuntimeIdentityRequest(ctx, "context_write_failed")
	}
	return types.ActionContinue
}

// mappedModel 按 legacy 相同的 exact、prefix、default 顺序计算当前映射结果。
// 输入约束：config 已由 parseConfig 排序 prefix；oldModel 来自合法 JSON string。
// 输出语义：没有映射时返回原模型；显式空 target 保持空值，供 Auto passthrough 判定区分默认改写。
// 边界场景：本函数只计算字符串结果，不改 Header/Body/context。
func mappedModel(config Config, oldModel string) string {
	newModel := config.defaultModel
	if newModel == "" {
		newModel = oldModel
	}
	if mapped, ok := config.exactModelMapping[oldModel]; ok {
		return mapped
	}
	for _, mapping := range config.prefixModelMapping {
		if strings.HasPrefix(oldModel, mapping.Prefix) {
			return mapping.Target
		}
	}
	return newModel
}

// mapsModelToPassthrough 确认 reserved Auto 命中了服务端显式下发的空 exact/prefix 保护规则。
// 输入约束：调用方已确认 selector 位于 ReservedAutoSelectors；普通 `*` 默认映射不构成 Auto 保护。
// 输出语义：仅首个实际命中的 exact/prefix target 为空时返回 true。
// 边界场景：缺少保护规则或非空目标保持严格拒绝，不能借 Auto 名义绕过真实 mapper transition。
func mapsModelToPassthrough(config Config, model string) bool {
	if target, ok := config.exactModelMapping[model]; ok {
		return target == ""
	}
	for _, mapping := range config.prefixModelMapping {
		if strings.HasPrefix(model, mapping.Prefix) {
			return mapping.Target == ""
		}
	}
	return false
}
