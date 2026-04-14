# Requirements Document

## Introduction

当前 Higress AI 安全护栏 WASM Go 插件中，`evaluateRiskMultiModal()` 函数在处理 `sensitiveDataAction=mask` 场景时，只要检测到 `detail.Suggestion == "mask"` 就会触发脱敏操作，而不检查风险等级是否达到用户配置的 `sensitiveDataLevelBar` 阈值。这导致低风险等级的敏感数据也会被不必要地脱敏。

本需求修改该逻辑：仅当 `detail.Suggestion == "mask"` **且** 风险等级 `>=` 用户配置的 `sensitiveDataLevelBar` 阈值时，才触发脱敏（返回 `RiskMask`）；否则仅记录日志并放行（返回 `RiskPass`）。

## Glossary

- **Plugin**: Higress AI 安全护栏 WASM Go 插件（ai-security-guard），负责对 AI 请求和响应进行安全检查
- **EvaluateRiskMultiModal**: `config/config.go` 中的 `evaluateRiskMultiModal()` 函数，实现 MultiModalGuard 模式下的逐维度风险评估逻辑
- **Detail**: 安全检查 API 返回的逐维度检测结果，包含 `Type`（维度类型）、`Suggestion`（建议动作）、`Level`（风险等级）等字段
- **DimAction**: 通过 `ResolveRiskActionByType()` 解析得到的维度级别动作配置，可为 `"block"` 或 `"mask"`
- **Exceeds**: `detailExceedsThreshold()` 函数的返回值，表示某条 Detail 的风险等级是否达到或超过用户配置的对应维度阈值
- **SensitiveDataLevelBar**: 用户配置的敏感数据维度风险等级阈值（如 `S1`、`S2`、`S3`、`S4`），用于判断是否需要触发动作
- **RiskMask**: `EvaluateRisk()` 的返回值之一，表示需要对内容进行脱敏处理
- **RiskPass**: `EvaluateRisk()` 的返回值之一，表示放行，不做任何处理

## Requirements

### Requirement 1: Mask 动作需检查风险等级阈值

**User Story:** 作为插件使用者，我希望当 `sensitiveDataAction` 配置为 `mask` 时，只有风险等级达到我配置的 `sensitiveDataLevelBar` 阈值的敏感数据才会被脱敏，这样低风险的敏感数据不会被不必要地修改。

#### Acceptance Criteria

1. WHEN a Detail with `Type=sensitiveData` and `Suggestion=mask` is evaluated AND the Detail's Level is greater than or equal to the configured `sensitiveDataLevelBar` threshold, THE EvaluateRiskMultiModal SHALL set the mask candidate flag to true (contributing to a `RiskMask` result).
2. WHEN a Detail with `Type=sensitiveData` and `Suggestion=mask` is evaluated AND the Detail's Level is less than the configured `sensitiveDataLevelBar` threshold, THE EvaluateRiskMultiModal SHALL skip the mask candidate flag and log the event (contributing to a `RiskPass` result if no other dimensions trigger block or mask).
3. WHEN a Detail with `Type=sensitiveData`, `Suggestion=mask`, and Level below threshold is evaluated alongside other Details that do not trigger block, THE EvaluateRiskMultiModal SHALL return `RiskPass`.
4. WHEN multiple sensitiveData Details exist with `Suggestion=mask`, THE EvaluateRiskMultiModal SHALL apply the threshold check to each Detail independently, setting the mask candidate flag only for those Details whose Level meets or exceeds the threshold.

### Requirement 2: 低于阈值的 Mask 建议需记录日志

**User Story:** 作为运维人员，我希望当敏感数据的风险等级低于阈值而被跳过脱敏时，系统能记录日志，这样我可以追踪被放行的敏感数据检测事件。

#### Acceptance Criteria

1. WHEN a Detail with `Type=sensitiveData` and `Suggestion=mask` is evaluated AND the Detail's Level is less than the configured `sensitiveDataLevelBar` threshold, THE EvaluateRiskMultiModal SHALL log an informational message containing the detail type, suggestion, level, and the configured threshold.

### Requirement 3: 保持现有 Block 逻辑不变

**User Story:** 作为插件使用者，我希望此次修改不影响现有的 block 判定逻辑，这样其他安全检查行为保持一致。

#### Acceptance Criteria

1. WHEN a Detail has `Suggestion=block`, THE EvaluateRiskMultiModal SHALL return `RiskBlock` regardless of the dimension action or threshold configuration.
2. WHEN `dimAction=block` and the Detail's Level exceeds the configured threshold, THE EvaluateRiskMultiModal SHALL return `RiskBlock`.
3. WHEN the top-level `Data.RiskLevel` exceeds the `contentModerationLevelBar` threshold, THE EvaluateRiskMultiModal SHALL return `RiskBlock` before evaluating any Detail.
4. WHEN the top-level `Data.AttackLevel` exceeds the `promptAttackLevelBar` threshold, THE EvaluateRiskMultiModal SHALL return `RiskBlock` before evaluating any Detail.
5. WHEN the `Data.Suggestion` is `block` and no Detail triggers block, THE EvaluateRiskMultiModal SHALL return `RiskBlock` as a fallback.

### Requirement 4: 阈值达标时的 Mask 行为保持不变

**User Story:** 作为插件使用者，我希望当敏感数据风险等级达到阈值时，脱敏行为与修改前完全一致，这样现有的脱敏流程不受影响。

#### Acceptance Criteria

1. WHEN `sensitiveDataAction=mask`, `detail.Suggestion=mask`, and the Detail's Level meets or exceeds `sensitiveDataLevelBar`, THE EvaluateRiskMultiModal SHALL return `RiskMask` (assuming no other dimension triggers block).
2. WHEN `RiskMask` is returned, THE Plugin SHALL extract the desensitization content from the Detail's `Result[].Ext.Desensitization` field and replace the original content in the request body, consistent with existing behavior.

### Requirement 5: 更新现有测试用例以反映新行为

**User Story:** 作为开发者，我希望现有测试用例能正确反映新的阈值检查逻辑，这样测试套件能准确验证修改后的行为。

#### Acceptance Criteria

1. FOR ALL existing test cases where `sensitiveDataAction=mask` and `Suggestion=mask`, THE test suite SHALL verify that the result depends on whether the Detail's Level meets or exceeds the configured `sensitiveDataLevelBar` threshold.
2. THE test suite SHALL include a test case where `sensitiveDataAction=mask`, `Suggestion=mask`, and the Detail's Level is below `sensitiveDataLevelBar`, verifying that the result is `RiskPass`.
3. THE test suite SHALL include a test case where `sensitiveDataAction=mask`, `Suggestion=mask`, and the Detail's Level meets `sensitiveDataLevelBar`, verifying that the result is `RiskMask`.
4. THE test suite SHALL include a test case with multiple sensitiveData Details where some are above and some are below the threshold, verifying that only the above-threshold Details contribute to `RiskMask`.
