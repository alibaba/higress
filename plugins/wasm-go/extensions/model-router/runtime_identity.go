package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	resolvedModelContextProperty      = "apig_resolved_model_context_v1"
	resolvedModelContextSchemaVersion = 1
	runtimeIdentityMode               = "ChooseModelV1"
	runtimeIdentityJSONBody           = "json_body"
	// runtimeIdentityTargetClusterHeader 是 strict DynamicClusterHeader Route 的唯一上游选择头。
	// Direct/Auto/Mapper 都必须由冻结 target 覆盖它，不能保留客户端传入值。
	runtimeIdentityTargetClusterHeader = "x-envoy-target-cluster"
	// runtimeIdentityRequestPropertyDeclarationContextKey 防止同一个 Wasm 根上下文重复声明同一 property。
	// 每个严格 carrier 都有独立根上下文，因此该标记只在自身的重载生命周期内生效。
	runtimeIdentityRequestPropertyDeclarationContextKey = "model-router-runtime-identity-request-property-declared"
)

// runtimeIdentityScope 定义控制面冻结到数据面的单条 ModelCard 选择作用域。
type runtimeIdentityScope struct {
	GatewayID          string `json:"gatewayId"`
	APIID              string `json:"apiId"`
	RouteID            string `json:"routeId"`
	DataPlaneRouteName string `json:"dataPlaneRouteName"`
}

// runtimeIdentityParser 定义当前 strict route 唯一允许的请求 Body 选择器来源。
type runtimeIdentityParser struct {
	Source                     string `json:"source"`
	ModelKey                   string `json:"modelKey"`
	MinimumDataPlaneCapability string `json:"minimumDataPlaneCapability,omitempty"`
}

// runtimeIdentityTarget 表示已在控制面解析完成的实际 ModelCard 身份、上游表现形式和执行目标。
type runtimeIdentityTarget struct {
	ModelCardID       string `json:"modelCardId"`
	Provider          string `json:"provider"`
	UpstreamModelName string `json:"upstreamModelName"`
	ServiceID         string `json:"serviceId"`
	TargetCluster     string `json:"targetCluster"`
}

// runtimeIdentityRule 是 model-router 新模式的不可变选择器和闭包配置。
type runtimeIdentityRule struct {
	Mode                  string                           `json:"mode"`
	ConfigRevision        string                           `json:"configRevision"`
	Scope                 runtimeIdentityScope             `json:"scope"`
	Parser                runtimeIdentityParser            `json:"parser"`
	SelectorTargets       map[string]runtimeIdentityTarget `json:"selectorTargets"`
	ReservedAutoSelectors []string                         `json:"reservedAutoSelectors"`
	TargetClosure         map[string]runtimeIdentityTarget `json:"targetClosure"`
}

// resolvedModelContext 是跨内置插件传播的 request-lifespan 可信身份，而不是客户端可写 Header。
type resolvedModelContext struct {
	SchemaVersion       int    `json:"schemaVersion"`
	GatewayID           string `json:"gatewayId"`
	APIID               string `json:"apiId"`
	RouteID             string `json:"routeId"`
	ConfigRevision      string `json:"configRevision"`
	ResolvedModelCardID string `json:"resolvedModelCardId"`
	TransitionSeq       int64  `json:"transitionSeq"`
	Source              string `json:"source"`
}

// ensureRuntimeIdentityRequestProperty 在 strict 配置的 Wasm 根上下文预声明 context 为 DownstreamRequest 生命周期。
// 输入约束：调用时正处于 OnPluginStart/Reload，配置仅来自控制面；普通 legacy 配置不得触发 foreign function。
// 输出语义：Direct 写入与后续 internal redirect 重入共享同一 request FilterState；声明失败使插件启动失败，避免退回 FilterChain 生命周期。
// 边界场景：该声明属于当前 model-router 根上下文；Auto 的首次 writer 由 ai-auto-router 在其独立根上下文作同类声明，二者不共享 declaration map。
func ensureRuntimeIdentityRequestProperty(ctx wrapper.PluginContext) error {
	raw, err := proxywasm.GetPluginConfiguration()
	if err != nil {
		if errors.Is(err, types.ErrorStatusNotFound) {
			return nil
		}
		return fmt.Errorf("read runtime identity plugin configuration: %w", err)
	}
	if !gjson.ValidBytes(raw) || !runtimeIdentityConfigRequiresRequestProperty(gjson.ParseBytes(raw)) {
		return nil
	}
	if ctx.GetContext(runtimeIdentityRequestPropertyDeclarationContextKey) != nil {
		return nil
	}
	if err := declareRuntimeIdentityRequestProperty(); err != nil {
		return err
	}
	ctx.SetContext(runtimeIdentityRequestPropertyDeclarationContextKey, true)
	return nil
}

// runtimeIdentityConfigRequiresRequestProperty 判断全局或 route-scoped 配置是否可能由本插件首次写入严格身份。
// 输入约束：raw 必须是已通过 JSON 语法校验的 plugin config；只检查控制面字段，不读取请求输入。
// 输出语义：存在 object 形态的 modelRuntimeIdentity 时返回 true，其他 legacy 配置保持 false。
// 边界场景：仅有 strictRuntimeIdentityOnly 而没有 rule 时不声明，避免空 carrier 对不支持 foreign function 的历史链路产生副作用。
func runtimeIdentityConfigRequiresRequestProperty(raw gjson.Result) bool {
	if raw.Get("modelRuntimeIdentity").IsObject() {
		return true
	}
	for _, rule := range raw.Get("_rules_").Array() {
		if rule.Get("modelRuntimeIdentity").IsObject() {
			return true
		}
	}
	return false
}

// declareRuntimeIdentityRequestProperty 以 Envoy declare_property 协议注册可变 Bytes request property。
// 输入约束：只能从对应 identity writer 的 PluginContext 启动回调调用；属性名为跨插件固定协议，不能由配置覆盖。
// 输出语义：成功后该根上下文的首次 SetProperty 创建 DownstreamRequest FilterState，能跨 internal redirect 保留。
// 边界场景：显式写入 Bytes=0 与 DownstreamRequest=1，避免依赖 Envoy 默认 FilterChain 生命周期；hostcall 失败必须向上返回并阻止 strict 插件启用。
func declareRuntimeIdentityRequestProperty() error {
	payload := runtimeIdentityRequestPropertyDeclarationPayload()
	if _, err := proxywasm.CallForeignFunction("declare_property", payload); err != nil {
		return fmt.Errorf("declare runtime identity request property: %w", err)
	}
	return nil
}

// runtimeIdentityRequestPropertyDeclarationPayload 编码 Envoy DeclarePropertyArguments 的最小 protobuf wire payload。
// 输入约束：property 名称和枚举值均为本协议常量，不接收运行时外部输入。
// 输出语义：payload 表示 name、mutable、Bytes 和 DownstreamRequest，可直接传给 declare_property。
// 边界场景：protobuf 的默认字段也显式编码 Bytes=0，确保协议读取方不会把类型解释为字符串或默认生命周期。
func runtimeIdentityRequestPropertyDeclarationPayload() []byte {
	payload := make([]byte, 0, len(resolvedModelContextProperty)+8)
	payload = append(payload, 0x0a)
	payload = binary.AppendUvarint(payload, uint64(len(resolvedModelContextProperty)))
	payload = append(payload, resolvedModelContextProperty...)
	payload = append(payload, 0x18, 0x00) // type = Bytes
	payload = append(payload, 0x28, 0x01) // span = DownstreamRequest
	return payload
}

// parseRuntimeIdentityRule 解析并验证控制面下发的 strict identity rule。
// 输入约束：raw 仅来自服务端 rule config；缺失字段或任意未闭合 target 都视为配置错误。
// 输出语义：不存在字段返回 nil；存在时返回可用于 Body resolver 的不可变规则。
// 边界场景：不接受 Header/query/path 推断，避免客户端输入绕过 frozen selector map。
func parseRuntimeIdentityRule(raw gjson.Result) (*runtimeIdentityRule, error) {
	if !raw.Exists() || strings.TrimSpace(raw.Raw) == "" || raw.Raw == "null" {
		return nil, nil
	}
	var rule runtimeIdentityRule
	if err := json.Unmarshal([]byte(raw.Raw), &rule); err != nil {
		return nil, fmt.Errorf("decode model runtime identity: %w", err)
	}
	if rule.Mode != runtimeIdentityMode || strings.TrimSpace(rule.ConfigRevision) == "" || rule.Scope.GatewayID == "" || rule.Scope.APIID == "" || rule.Scope.RouteID == "" || rule.Scope.DataPlaneRouteName == "" {
		return nil, errors.New("model runtime identity scope or revision is incomplete")
	}
	if rule.Parser.Source != runtimeIdentityJSONBody || strings.TrimSpace(rule.Parser.ModelKey) == "" {
		return nil, errors.New("model runtime identity parser is unsupported")
	}
	if len(rule.SelectorTargets) == 0 || len(rule.TargetClosure) == 0 {
		return nil, errors.New("model runtime identity targets are empty")
	}
	for selector, target := range rule.SelectorTargets {
		if strings.TrimSpace(selector) == "" || !validRuntimeIdentityTarget(target) {
			return nil, errors.New("model runtime identity selector target is invalid")
		}
		if closure, exists := rule.TargetClosure[target.ModelCardID]; !exists || !sameRuntimeIdentityTarget(closure, target) {
			return nil, errors.New("model runtime identity selector is outside target closure")
		}
	}
	for modelCardID, target := range rule.TargetClosure {
		if modelCardID == "" || modelCardID != target.ModelCardID || !validRuntimeIdentityTarget(target) {
			return nil, errors.New("model runtime identity closure target is invalid")
		}
	}
	return &rule, nil
}

// validRuntimeIdentityTarget 校验身份 target 的最小不可变字段。
func validRuntimeIdentityTarget(target runtimeIdentityTarget) bool {
	return strings.TrimSpace(target.ModelCardID) != "" && strings.TrimSpace(target.Provider) != "" && strings.TrimSpace(target.UpstreamModelName) != "" && strings.TrimSpace(target.ServiceID) != "" && strings.TrimSpace(target.TargetCluster) != ""
}

// sameRuntimeIdentityTarget 判断两个 target 是否表达同一张卡、模型表现形式和执行目标。
func sameRuntimeIdentityTarget(left, right runtimeIdentityTarget) bool {
	return left.ModelCardID == right.ModelCardID && left.Provider == right.Provider && left.UpstreamModelName == right.UpstreamModelName && left.ServiceID == right.ServiceID && left.TargetCluster == right.TargetCluster
}

// loadResolvedModelContext 读取只由内置插件写入的 request-lifespan 身份 property。
// 输入约束：调用点不得从 Header、Body 或 wrapper Context 合成替代 context。
// 输出语义：property 未写入时返回 nil；编码损坏或 hostcall 失败返回错误，调用方必须 fail closed。
// 边界场景：空字节不是合法 context，防止清除态被误解为无上下文并重新选择模型。
func loadResolvedModelContext() (*resolvedModelContext, error) {
	raw, err := proxywasm.GetProperty([]string{resolvedModelContextProperty})
	if err != nil {
		if errors.Is(err, types.ErrorStatusNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("resolved model context is empty")
	}
	var context resolvedModelContext
	if err := json.Unmarshal(raw, &context); err != nil {
		return nil, fmt.Errorf("decode resolved model context: %w", err)
	}
	return &context, nil
}

// validateResolvedModelContextForRule 校验已有 context 能否在当前 strict route 再入时继续使用。
// 输入约束：context 必须来自 loadResolvedModelContext；rule 是当前请求匹配的控制面冻结配置。
// 输出语义：返回空字符串表示上下文可保留；非空为稳定低基数拒绝原因。
// 边界场景：任何 scope/revision/target 差异都拒绝，绝不根据当前 Body 的模型字段覆盖已有可信身份。
func validateResolvedModelContextForRule(context *resolvedModelContext, rule *runtimeIdentityRule) string {
	if context == nil || rule == nil {
		return "missing_context"
	}
	if context.SchemaVersion != resolvedModelContextSchemaVersion {
		return "unsupported_context_version"
	}
	if context.ConfigRevision != rule.ConfigRevision {
		return "context_revision_mismatch"
	}
	if context.GatewayID != rule.Scope.GatewayID || context.APIID != rule.Scope.APIID || context.RouteID != rule.Scope.RouteID {
		return "context_scope_mismatch"
	}
	if context.TransitionSeq < 1 || context.Source == "" {
		return "invalid_context_generation"
	}
	if _, exists := rule.TargetClosure[context.ResolvedModelCardID]; !exists {
		return "context_target_not_in_closure"
	}
	return ""
}

// writeResolvedModelContext 将一次成功 Direct 选择写入请求生命周期 property。
// 输入约束：target 已从当前 rule 的精确 selector map 获取，所有 Body/Header 改写已成功。
// 输出语义：成功时后续 mapper/auto/ai-proxy 可读取 seq=1 的同一身份；失败由调用方拒绝请求。
// 边界场景：不写入客户端 Header，也不保留 selector 原文，避免 canonical/alias 文本成为身份来源。
func writeResolvedModelContext(rule *runtimeIdentityRule, target runtimeIdentityTarget, sequence int64, source string) error {
	context := resolvedModelContext{
		SchemaVersion:       resolvedModelContextSchemaVersion,
		GatewayID:           rule.Scope.GatewayID,
		APIID:               rule.Scope.APIID,
		RouteID:             rule.Scope.RouteID,
		ConfigRevision:      rule.ConfigRevision,
		ResolvedModelCardID: target.ModelCardID,
		TransitionSeq:       sequence,
		Source:              source,
	}
	raw, err := json.Marshal(context)
	if err != nil {
		return err
	}
	return proxywasm.SetProperty([]string{resolvedModelContextProperty}, raw)
}

// isReservedAutoSelector 判断 selector 是否必须交给 ai-auto-router 在最终候选后建立 identity。
func isReservedAutoSelector(rule *runtimeIdentityRule, selector string) bool {
	for _, reserved := range rule.ReservedAutoSelectors {
		if reserved == selector {
			return true
		}
		if strings.HasSuffix(reserved, "*") && strings.HasPrefix(selector, strings.TrimSuffix(reserved, "*")) {
			return true
		}
	}
	return false
}

// rejectRuntimeIdentityRequest 返回不携带 Body、凭据或 selector 内容的稳定本地错误。
func rejectRuntimeIdentityRequest(ctx wrapper.HttpContext, reason string) types.Action {
	ctx.DontReadResponseBody()
	body, _ := json.Marshal(map[string]any{"error": map[string]string{"type": "model_runtime_identity_error", "code": reason}})
	_ = proxywasm.SendHttpResponse(http.StatusBadRequest, [][2]string{{"content-type", "application/json"}}, body, -1)
	return types.ActionPause
}
