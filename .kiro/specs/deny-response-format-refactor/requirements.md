# Requirements Document

## Introduction

当前 Higress AI 安全护栏插件（ai-security-guard）在拦截请求/响应时，`BuildDenyResponseBody` 生成的 `DenyResponseBody` 包含完整的 `Detail` 结构（含 `Suggestion`、`Result` 等冗余字段）、`requestId` 和 `guardCode`，信息过多且缺少用户自定义的拦截提示。

本需求对 `DenyResponseBody` 进行精简重构：
1. 只保留防护维度（`type`）、风险等级（`level`）和错误码（`code`）
2. 新增 `denyMessage` 字段，将用户配置的 `denyMessage` 合并到拦截返回体中
3. 所有协议路径（MCP JSON / MCP SSE / LLM OpenAI / LLM Original）统一生效

## Glossary

- **DenyResponseBody**: `config/config.go` 中定义的拦截返回体结构，由 `BuildDenyResponseBody()` 构建，被所有拦截路径使用
- **BuildDenyResponseBody**: 构建拦截返回体的函数，接收安全服务响应、插件配置和 consumer 参数
- **BlockedDetail**: 精简后的拦截详情结构，仅包含 `type`（防护维度）和 `level`（风险等级）
- **DenyMessage**: 用户在插件配置中通过 `denyMessage` 字段设置的自定义拦截提示文本
- **GetUnacceptableDetail**: 从安全服务返回的 Detail 列表中筛选出触发拦截的条目的函数

## Requirements

### Requirement 1: 精简 DenyResponseBody 结构

**User Story:** 作为插件使用者，我希望拦截返回体只包含必要的信息（防护维度、风险等级、错误码），去掉冗余字段，使返回数据更简洁易读。

#### Acceptance Criteria

1. THE `DenyResponseBody` struct SHALL contain exactly three fields: `code` (int), `denyMessage` (string, omitempty), and `blockedDetails` (array of `BlockedDetail`).
2. THE `BlockedDetail` struct SHALL contain exactly two fields: `type` (string) and `level` (string).
3. THE `DenyResponseBody` SHALL NOT contain `requestId` field.
4. THE `blockedDetails` array SHALL NOT contain `Suggestion`, `Result`, or any other fields from the original `Detail` struct.

### Requirement 2: 新增 denyMessage 字段

**User Story:** 作为插件使用者，我希望配置的 `denyMessage` 能出现在拦截返回体中，这样下游客户端可以直接展示用户友好的拦截提示。

#### Acceptance Criteria

1. WHEN the user has configured a non-empty `denyMessage` in the plugin config, THE `BuildDenyResponseBody` SHALL include the `denyMessage` value in the `DenyResponseBody.DenyMessage` field.
2. WHEN the user has NOT configured `denyMessage` (empty string), THE `DenyResponseBody` JSON output SHALL NOT contain the `denyMessage` key (via `omitempty`).
3. THE `denyMessage` field SHALL be read from `config.DenyMessage`, which is already parsed from the `denyMessage` configuration key.

### Requirement 3: code 字段使用安全服务业务码

**User Story:** 作为插件使用者，我希望拦截返回体中的 `code` 字段反映安全服务的业务状态码，便于程序化判断拦截原因。

#### Acceptance Criteria

1. THE `DenyResponseBody.Code` field SHALL be populated with `response.Code` (the business code from the security service response, typically 200 when a risk was detected).
2. THE field name in JSON serialization SHALL be `"code"`.

### Requirement 4: blockedDetails 仅包含 type 和 level

**User Story:** 作为插件使用者，我希望 `blockedDetails` 中每个条目只展示防护维度和对应的风险等级，不暴露内部检测细节。

#### Acceptance Criteria

1. FOR EACH unacceptable Detail returned by `GetUnacceptableDetail`, THE `BuildDenyResponseBody` SHALL extract only `Type` and `Level` fields and map them to a `BlockedDetail` struct.
2. THE existing `GetUnacceptableDetail` filtering logic (including fallback synthesis for top-level RiskLevel/AttackLevel) SHALL remain unchanged.
3. THE `BlockedDetail.Type` SHALL use lowercase JSON key `"type"` and `BlockedDetail.Level` SHALL use lowercase JSON key `"level"`.

### Requirement 5: 所有协议路径统一生效

**User Story:** 作为插件使用者，我希望无论使用 MCP 协议还是 LLM API 协议，拦截返回体的内容结构都是一致的。

#### Acceptance Criteria

1. THE MCP JSON response path (`HandleMcpResponseBody`), MCP SSE response path (`HandleMcpStreamingResponseBody`), MCP request path (`HandleMcpRequestBody`), LLM OpenAI request/response paths, and LLM Original protocol path SHALL all use the same `BuildDenyResponseBody` output.
2. No handler code changes SHALL be required — the format change SHALL be fully contained within `BuildDenyResponseBody` and the struct definitions.

### Requirement 6: 向后兼容性

**User Story:** 作为插件使用者，我希望未配置 `denyMessage` 时，返回体结构与精简前的核心信息保持一致（仅去掉冗余字段），不影响已有的客户端解析逻辑。

#### Acceptance Criteria

1. WHEN `denyMessage` is not configured, THE JSON output SHALL contain only `code` and `blockedDetails` fields (no `denyMessage` key).
2. THE `blockedDetails` array SHALL contain the same set of entries as before (same filtering logic), only with fewer fields per entry.
