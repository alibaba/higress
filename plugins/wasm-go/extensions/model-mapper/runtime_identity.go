package main

import (
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
	// runtimeIdentityTargetClusterHeader 是严格 DynamicClusterHeader Route 的受控执行目标。
	// mapper/fallback 在发生或确认切换后必须覆盖它，不能继续使用请求进入时的 Header。
	runtimeIdentityTargetClusterHeader = "x-envoy-target-cluster"
)

// runtimeIdentityScope 定义 mapper 只能消费的冻结请求作用域。
type runtimeIdentityScope struct {
	GatewayID          string `json:"gatewayId"`
	APIID              string `json:"apiId"`
	RouteID            string `json:"routeId"`
	DataPlaneRouteName string `json:"dataPlaneRouteName"`
}

// runtimeIdentityParser 是控制面登记的 Body selector parser。
type runtimeIdentityParser struct {
	Source                     string `json:"source"`
	ModelKey                   string `json:"modelKey"`
	MinimumDataPlaneCapability string `json:"minimumDataPlaneCapability,omitempty"`
}

// runtimeIdentityTarget 表示一张冻结主执行 ModelCard 及其唯一 Service/Cluster 执行目标。
type runtimeIdentityTarget struct {
	ModelCardID       string `json:"modelCardId"`
	Provider          string `json:"provider"`
	UpstreamModelName string `json:"upstreamModelName"`
	ServiceID         string `json:"serviceId"`
	TargetCluster     string `json:"targetCluster"`
}

// runtimeIdentityRule 保存当前规则的 revision、scope 和可切换 target closure。
type runtimeIdentityRule struct {
	Mode                  string                           `json:"mode"`
	ConfigRevision        string                           `json:"configRevision"`
	Scope                 runtimeIdentityScope             `json:"scope"`
	Parser                runtimeIdentityParser            `json:"parser"`
	ReservedAutoSelectors []string                         `json:"reservedAutoSelectors"`
	TargetClosure         map[string]runtimeIdentityTarget `json:"targetClosure"`
}

// resolvedModelContext 是 model-router/auto/mapper 之间唯一可信的 request-lifespan 身份载体。
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

// parseMapperRuntimeIdentity 解析 ModelMapper 的严格 rule 和对应 compiler target。
// 输入约束：三个字段都只能由控制面 projection 写入；出现其中一个而缺另一个视为损坏配置。
// 输出语义：legacy 配置返回 nil；新模式返回同一 revision 内已验证 target 与稳定规则键。
// 边界场景：不允许根据 modelMapping 或 service matcher 反推 target，避免跨 Provider 同名模型错绑。
func parseMapperRuntimeIdentity(rawRule, rawTarget, rawTargetKey gjson.Result) (*runtimeIdentityRule, *runtimeIdentityTarget, string, error) {
	if !rawRule.Exists() && !rawTarget.Exists() && !rawTargetKey.Exists() {
		return nil, nil, "", nil
	}
	if !rawRule.Exists() || !rawTarget.Exists() || !rawTargetKey.Exists() {
		return nil, nil, "", errors.New("model runtime identity transition config is incomplete")
	}
	var rule runtimeIdentityRule
	if err := json.Unmarshal([]byte(rawRule.Raw), &rule); err != nil {
		return nil, nil, "", fmt.Errorf("decode model runtime identity: %w", err)
	}
	var target runtimeIdentityTarget
	if err := json.Unmarshal([]byte(rawTarget.Raw), &target); err != nil {
		return nil, nil, "", fmt.Errorf("decode model runtime identity target: %w", err)
	}
	targetKey := strings.TrimSpace(rawTargetKey.String())
	if rule.Mode != runtimeIdentityMode || rule.ConfigRevision == "" || rule.Scope.GatewayID == "" || rule.Scope.APIID == "" || rule.Scope.RouteID == "" || rule.Scope.DataPlaneRouteName == "" || rule.Parser.Source != runtimeIdentityJSONBody || rule.Parser.ModelKey == "" || len(rule.TargetClosure) == 0 || !validRuntimeIdentityTarget(target) {
		return nil, nil, "", errors.New("model runtime identity transition config is invalid")
	}
	if targetKey == "" || (!strings.HasPrefix(targetKey, "mapper:") && !strings.HasPrefix(targetKey, "fallback:")) {
		return nil, nil, "", errors.New("model runtime identity transition key is invalid")
	}
	closure, exists := rule.TargetClosure[target.ModelCardID]
	if !exists || !sameRuntimeIdentityTarget(closure, target) {
		return nil, nil, "", errors.New("model runtime identity target is outside closure")
	}
	return &rule, &target, targetKey, nil
}

// validRuntimeIdentityTarget 校验 target 的不可省略身份字段和执行闭包。
func validRuntimeIdentityTarget(target runtimeIdentityTarget) bool {
	return strings.TrimSpace(target.ModelCardID) != "" && strings.TrimSpace(target.Provider) != "" && strings.TrimSpace(target.UpstreamModelName) != "" && strings.TrimSpace(target.ServiceID) != "" && strings.TrimSpace(target.TargetCluster) != ""
}

// sameRuntimeIdentityTarget 判断两个 target 是否表达同一卡、同一上游模型和同一执行目标。
func sameRuntimeIdentityTarget(left, right runtimeIdentityTarget) bool {
	return left.ModelCardID == right.ModelCardID && left.Provider == right.Provider && left.UpstreamModelName == right.UpstreamModelName && left.ServiceID == right.ServiceID && left.TargetCluster == right.TargetCluster
}

// isReservedAutoSelector 判断当前 Body 字面量是否属于控制面冻结的 Auto exact/prefix 空间。
// 输入约束：rule 来自完整 strict 配置；selector 是 parser 已确认的非空字符串，仅用于决定是否延后首次身份写入。
// 输出语义：exact 相等或命中以 `*` 结尾的前缀时返回 true；普通 Direct selector 返回 false。
// 边界场景：本函数不把 Auto 字面量解释为 ModelCard，也不允许它覆盖已存在 context。
func isReservedAutoSelector(rule *runtimeIdentityRule, selector string) bool {
	if rule == nil {
		return false
	}
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

// loadResolvedModelContext 从 Envoy property 读取已有身份，拒绝损坏值。
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
		return nil, err
	}
	return &context, nil
}

// validateResolvedModelContextForRule 确认 mapper 只接受当前 scope/revision 的上游身份。
func validateResolvedModelContextForRule(context *resolvedModelContext, rule *runtimeIdentityRule) string {
	if context == nil {
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

// writeResolvedModelContext 用新的 target 和递增 sequence 提交 mapper/fallback 切换。
func writeResolvedModelContext(rule *runtimeIdentityRule, target runtimeIdentityTarget, sequence int64, source string) error {
	raw, err := json.Marshal(resolvedModelContext{
		SchemaVersion:       resolvedModelContextSchemaVersion,
		GatewayID:           rule.Scope.GatewayID,
		APIID:               rule.Scope.APIID,
		RouteID:             rule.Scope.RouteID,
		ConfigRevision:      rule.ConfigRevision,
		ResolvedModelCardID: target.ModelCardID,
		TransitionSeq:       sequence,
		Source:              source,
	})
	if err != nil {
		return err
	}
	return proxywasm.SetProperty([]string{resolvedModelContextProperty}, raw)
}

// rejectRuntimeIdentityRequest 发出稳定错误，不记录 Body、模型名或凭据。
func rejectRuntimeIdentityRequest(ctx wrapper.HttpContext, reason string) types.Action {
	ctx.DontReadResponseBody()
	body, _ := json.Marshal(map[string]any{"error": map[string]string{"type": "model_runtime_identity_error", "code": reason}})
	_ = proxywasm.SendHttpResponse(http.StatusBadRequest, [][2]string{{"content-type", "application/json"}}, body, -1)
	return types.ActionPause
}
