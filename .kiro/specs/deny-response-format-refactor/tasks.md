# Implementation Plan: Deny Response Format Refactor

## Overview

精简 `DenyResponseBody` 结构体，新增 `BlockedDetail` 类型，修改 `BuildDenyResponseBody` 函数，将用户配置的 `denyMessage` 合并到拦截返回体中。改动集中在 `config/config.go` 一个文件，不需要修改任何 handler 代码。

## Tasks

- [x] 1. 新增 `BlockedDetail` 结构体并重构 `DenyResponseBody`
  - [x] 1.1 在 `config/config.go` 中新增 `BlockedDetail` 结构体
    - 在 `DenyResponseBody` 定义之前新增：
      ```go
      type BlockedDetail struct {
          Type  string `json:"type"`
          Level string `json:"level"`
      }
      ```
    - _Requirements: 1.2, 4.3_

  - [x] 1.2 修改 `DenyResponseBody` 结构体
    - 将当前定义：
      ```go
      type DenyResponseBody struct {
          BlockedDetails []Detail `json:"blockedDetails"`
          RequestId      string   `json:"requestId"`
          GuardCode      int      `json:"guardCode"`
      }
      ```
    - 改为：
      ```go
      type DenyResponseBody struct {
          Code           int              `json:"code"`
          DenyMessage    string           `json:"denyMessage,omitempty"`
          BlockedDetails []BlockedDetail  `json:"blockedDetails"`
      }
      ```
    - _Requirements: 1.1, 1.3, 2.1, 2.2, 3.1, 3.2_

  - [x] 1.3 修改 `BuildDenyResponseBody` 函数
    - 将当前实现改为：
      ```go
      func BuildDenyResponseBody(response Response, config AISecurityConfig, consumer string) ([]byte, error) {
          details := GetUnacceptableDetail(response.Data, config, consumer)
          blocked := make([]BlockedDetail, 0, len(details))
          for _, d := range details {
              blocked = append(blocked, BlockedDetail{
                  Type:  d.Type,
                  Level: d.Level,
              })
          }
          body := DenyResponseBody{
              Code:           response.Code,
              DenyMessage:    config.DenyMessage,
              BlockedDetails: blocked,
          }
          return json.Marshal(body)
      }
      ```
    - _Requirements: 2.1, 2.3, 3.1, 4.1, 4.2_

- [x] 2. 更新现有测试
  - [x] 2.1 更新 `config/` 目录下引用 `DenyResponseBody` 的测试
    - 搜索现有测试中对 `DenyResponseBody`、`BuildDenyResponseBody` 输出的断言
    - 更新对 `requestId`、`guardCode` 字段的断言为 `code` 字段
    - 更新对 `blockedDetails` 中 `Suggestion`、`Result` 等字段的断言，改为只检查 `type` 和 `level`
    - _Requirements: 6.1, 6.2_

- [x] 3. 新增测试用例
  - [x] 3.1 TestBuildDenyResponseBody_WithDenyMessage
    - 配置 `config.DenyMessage = "很抱歉，我无法回答您的问题"`，构造一个包含 contentModeration block 的 Response，调用 `BuildDenyResponseBody`，验证输出 JSON 包含 `"denyMessage":"很抱歉，我无法回答您的问题"`
    - _Requirements: 2.1_

  - [x] 3.2 TestBuildDenyResponseBody_WithoutDenyMessage
    - 不配置 `config.DenyMessage`（空字符串），调用 `BuildDenyResponseBody`，验证输出 JSON 不包含 `"denyMessage"` key
    - _Requirements: 2.2_

  - [x] 3.3 TestBuildDenyResponseBody_BlockedDetailsOnlyTypeAndLevel
    - 构造包含多个 Detail（含 Result、Suggestion 等字段）的 Response，调用 `BuildDenyResponseBody`，反序列化输出 JSON，验证 `blockedDetails` 中每个条目只有 `type` 和 `level` 两个字段
    - _Requirements: 1.2, 1.4, 4.1_

  - [x] 3.4 TestBuildDenyResponseBody_CodeField
    - 构造 `response.Code = 200` 的 Response，调用 `BuildDenyResponseBody`，验证输出 JSON 中 `"code"` 字段值为 200
    - _Requirements: 3.1, 3.2_

  - [x] 3.5 TestBuildDenyResponseBody_NoRequestId
    - 调用 `BuildDenyResponseBody`，验证输出 JSON 不包含 `"requestId"` key
    - _Requirements: 1.3_

  - [x] 3.6 TestBuildDenyResponseBody_FallbackSynthesis
    - 构造无 Detail 但有 top-level RiskLevel 超过阈值的 Response，调用 `BuildDenyResponseBody`，验证 fallback 合成的条目也被正确精简为只有 `type` 和 `level`
    - _Requirements: 4.2_

- [x] 4. 验证
  - Run `go test ./plugins/wasm-go/extensions/ai-security-guard/config/...` 确保所有测试通过。

## Notes

- 改动完全集中在 `config/config.go`，不需要修改任何 handler 代码
- `GetUnacceptableDetail()` 过滤逻辑不变，仍返回 `[]Detail`，映射在 `BuildDenyResponseBody` 中完成
- `DenyMessage` 使用 `omitempty`，未配置时 JSON 中不出现该字段，保持向后兼容
