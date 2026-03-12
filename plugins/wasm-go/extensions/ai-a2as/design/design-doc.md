# AI A2AS 插件设计文档

## Overview

### 插件目标
实现 OWASP A2AS (Agent-to-Agent Security) 框架的核心功能，在 API 网关层为 AI 应用提供安全防护，防范提示注入攻击和未授权操作。

### 解决的问题
1. **提示注入攻击** - 恶意用户通过 prompt 注入攻击控制 AI Agent
2. **未授权工具调用** - AI Agent 调用超出授权范围的工具
3. **Prompt 篡改** - 中间人攻击篡改 Agent 发送的内容
4. **缺乏安全策略** - AI 应用缺乏统一的安全策略管理

### 目标用户
- AI 应用开发者
- 企业 AI 安全团队
- 使用 Higress 网关的 LLM 应用

## Functional Design

### 为什么选择网关层实现？

| 方案 | 优点 | 缺点 |
|------|------|------|
| Agent 侧实现 | 更灵活 | 每个 Agent 需单独实现，难以统一管控 |
| **网关层实现** | **统一管控，无侵入** | 功能受限于请求/响应处理 |
| LLM 侧实现 | 最接近模型 | 需要 LLM 厂商支持 |

**选择网关层的原因**：
1. 统一安全策略管理，无需修改每个 Agent
2. 与现有 Higress 基础设施无缝集成
3. 支持 Per-Consumer 差异化策略
4. 无需 Agent 侧代码改造（除 Authenticated Prompts）

### 功能取舍

根据 OWASP A2AS 论文，我们选择实现**网关适合实现**的功能：

| 功能 | 是否实现 | 原因 |
|------|----------|------|
| Behavior Certificates | ✅ | 网关可检查请求中的 `tools` 字段 |
| Authenticated Prompts | ✅ | 网关可验证嵌入式签名 |
| In-Context Defenses | ✅ | 网关可注入系统消息 |
| Codified Policies | ✅ | 网关可注入策略消息 |
| Security Boundaries | ❌ | 应由 Agent 侧实现数据标记 |
| Audit Logs | ❌ | 应使用专用审计系统 |

## Core Function Design

### 1. `onHttpRequestBody` - 请求处理主入口

```go
func onHttpRequestBody(ctx wrapper.HttpContext, globalConfig A2ASConfig, body []byte) types.Action
```

**设计思考**：

- **处理流程设计**：采用"验证 → 转换 → 检查"的流水线模式
  ```
  请求 → 签名验证 → A2AS转换 → 工具权限检查 → LLM
  ```

- **为什么先验签后转换？**
  - 签名验证是身份认证，必须最先执行
  - 验签失败直接返回 403，避免不必要的处理开销
  - 验签成功后移除标签，后续处理干净的原始内容

- **错误处理策略**：任何环节失败都返回明确的错误响应
  - 签名失败：403 + "Invalid or missing prompt signature"
  - 转换失败：500 + "A2AS transformation failed"
  - 工具拒绝：403 + 配置的 `denyMessage`

- **Per-Consumer 配置合并**：
  ```go
  config := globalConfig.MergeConsumerConfig(consumer)
  ```
  允许不同消费者有不同的安全策略，实现差异化管控。

### 2. `verifyAndRemoveEmbeddedHashes` - 签名验证

```go
func verifyAndRemoveEmbeddedHashes(config AuthenticatedPromptsConfig, body []byte) ([]byte, error)
```

**设计思考**：

- **嵌入式签名 vs HTTP Header 签名**：
  
  | 方案 | 优点 | 缺点 |
  |------|------|------|
  | HTTP Header (RFC 9421) | 标准化 | 复杂，需要 Agent 大改 |
  | **嵌入式签名** | **简单直观** | 需要内容解析 |

  选择嵌入式签名原因：
  - Agent 侧实现简单（几行代码即可）
  - 无需修改 HTTP 客户端
  - 签名与内容紧密绑定，不易分离

- **签名格式设计**：`<a2as:TYPE:HASH>content</a2as:TYPE:HASH>`
  - `TYPE` 表示消息类型（user/tool/system）
  - `HASH` 为 HMAC-SHA256 截取值
  - 闭合标签验证防止标签嵌套攻击

- **为什么验签后移除标签？**
  - LLM 不需要看到安全标签
  - 避免标签干扰模型输出
  - 保持请求内容简洁

### 3. `applyA2ASTransformations` - 消息转换

```go
func applyA2ASTransformations(config A2ASConfig, body []byte) ([]byte, error)
```

**设计思考**：

- **消息注入位置设计**：
  ```
  [系统消息] ← In-Context Defense (as_system)
  [系统消息] ← Codified Policies (as_system)
  [原始消息...]
  [防御消息] ← In-Context Defense (before_user)
  [用户消息]
  ```

  两种位置的权衡：
  - `as_system`：作为系统级指令，优先级高
  - `before_user`：紧贴用户输入，针对性更强

- **为什么重建整个 messages 数组？**
  - 需要在特定位置插入消息
  - 保留原始消息的所有字段（tool_calls, name 等）
  - 使用 `sjson.SetRaw` 高效替换

### 4. `checkToolPermissions` - 工具权限检查

```go
func checkToolPermissions(config BehaviorCertificatesConfig, body []byte) (bool, string)
```

**设计思考**：

- **白名单 vs 黑名单**：选择白名单模式
  - 安全性更高（默认拒绝未知工具）
  - 配置更直观（列出允许的工具）
  - 空列表 = 拒绝所有（最安全默认值）

- **工具名称匹配**：支持多种格式
  ```go
  toolName := tool.Get("function.name").String()
  if toolName == "" {
      toolName = tool.Get("name").String()
  }
  ```
  兼容 OpenAI 和其他 LLM API 格式。

- **返回值设计**：`(denied bool, toolName string)`
  - 返回被拒绝的工具名，便于日志和错误消息

### 5. `computeContentHash` - 哈希计算

```go
func computeContentHash(config AuthenticatedPromptsConfig, content string) string
```

**设计思考**：

- **算法选择**：HMAC-SHA256
  - 业界标准，安全性高
  - Go 标准库原生支持
  - 与常见签名库兼容

- **密钥格式支持**：
  ```go
  secretBytes, err := base64.StdEncoding.DecodeString(config.SharedSecret)
  if err != nil {
      secretBytes = []byte(config.SharedSecret)
  }
  ```
  同时支持 Base64 和原始字符串，提高易用性。

- **哈希长度可配**：
  - 默认 8 字符（32 bit 安全性）
  - 可配置到 64 字符（256 bit 完整哈希）
  - 权衡：安全性 vs 标签长度

### 6. `MergeConsumerConfig` - 配置合并

```go
func (config A2ASConfig) MergeConsumerConfig(consumer string) A2ASConfig
```

**设计思考**：

- **合并策略**：Consumer 配置完全覆盖全局配置
  - 不是字段级合并，而是模块级覆盖
  - 简化逻辑，避免复杂的深度合并

- **为什么返回新配置而非修改原配置？**
  - 避免并发问题（多个请求共享全局配置）
  - 函数式风格，无副作用
  - 便于测试和调试

## Configuration Parameters

### Behavior Certificates

| Parameter | Type | Required | Description | Default |
|-----------|------|----------|-------------|---------|
| `enabled` | bool | Yes | 是否启用行为证书 | `false` |
| `allowedTools` | []string | No | 允许的工具白名单 | `[]` |
| `denyMessage` | string | No | 拒绝时的错误消息 | `"Tool call not permitted"` |

### Authenticated Prompts

| Parameter | Type | Required | Description | Default |
|-----------|------|----------|-------------|---------|
| `enabled` | bool | Yes | 是否启用签名验证 | `false` |
| `sharedSecret` | string | Yes* | HMAC-SHA256 密钥 | - |
| `hashLength` | int | No | 哈希截取长度 (4-64) | `8` |

### In-Context Defenses

| Parameter | Type | Required | Description | Default |
|-----------|------|----------|-------------|---------|
| `enabled` | bool | Yes | 是否启用上下文防御 | `false` |
| `template` | string | No | 模板类型: `default`/`custom` | `"default"` |
| `customPrompt` | string | No | 自定义防御指令 | `""` |
| `position` | string | No | 注入位置: `as_system`/`before_user` | `"as_system"` |

### Codified Policies

| Parameter | Type | Required | Description | Default |
|-----------|------|----------|-------------|---------|
| `enabled` | bool | Yes | 是否启用编码策略 | `false` |
| `policies` | []Policy | No | 策略列表 | `[]` |
| `position` | string | No | 注入位置: `as_system`/`before_user` | `"as_system"` |

### Policy Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | 策略名称 |
| `content` | string | Yes | 策略内容 |
| `severity` | string | No | 严重程度: `high`/`medium`/`low` |

### Per-Consumer Configuration

支持为不同消费者配置不同的安全策略：

```yaml
consumerConfigs:
  premium_user:
    behaviorCertificates:
      allowedTools: ["read", "write", "delete"]
  basic_user:
    behaviorCertificates:
      allowedTools: ["read"]
```

合并规则：Consumer 配置完全覆盖全局配置（模块级，非字段级）

## Technical Implementation

### 配置结构设计

```go
type A2ASConfig struct {
    AuthenticatedPrompts AuthenticatedPromptsConfig
    InContextDefenses    InContextDefensesConfig
    BehaviorCertificates BehaviorCertificatesConfig
    CodifiedPolicies     CodifiedPoliciesConfig
    ConsumerConfigs      map[string]*ConsumerA2ASConfig
}
```

**设计思考**：

- **模块化设计**：每个功能独立配置，可单独启用/禁用
- **Per-Consumer 支持**：`ConsumerConfigs` 支持差异化策略
- **默认值处理**：在 `ParseConfig` 中设置合理默认值

### 依赖库选择

| 库 | 用途 | 选择原因 |
|----|------|----------|
| `gjson` | JSON 读取 | 高性能，无需反序列化 |
| `sjson` | JSON 写入 | 与 gjson 配合，原地修改 |
| `proxy-wasm-go-sdk` | WASM 运行时 | Higress 标准 SDK |
| `wasm-go/pkg/wrapper` | 插件封装 | Higress 插件开发框架 |

### 配置验证

```go
func (config *A2ASConfig) Validate() error
```

验证规则：
1. 启用签名时必须提供 `sharedSecret`
2. `hashLength` 必须在 4-64 范围内
3. `position` 只能是 `as_system` 或 `before_user`
4. `severity` 只能是 `high`/`medium`/`low`
5. 策略必须有名称和内容

## Test Plan

### 测试分类

| 类型 | 文件 | 覆盖范围 |
|------|------|----------|
| 单元测试 | `main_test.go` | 核心函数逻辑 |
| 集成测试 | `test/*.go` | 完整请求流程 |

### 重点测试场景

1. **Authenticated Prompts**
   - 正确签名验证通过
   - 错误签名拒绝
   - 签名缺失处理
   - 标签格式验证

2. **Behavior Certificates**
   - 白名单内工具允许
   - 白名单外工具拒绝
   - 空白名单拒绝所有

3. **消息注入**
   - `as_system` 位置正确
   - `before_user` 位置正确
   - 多种消息类型保留

### Performance Considerations

1. **JSON 处理**：使用 `gjson`/`sjson` 而非 `encoding/json`
   - 避免完整反序列化
   - 只处理需要的字段

2. **正则表达式**：预编译模式
   ```go
   pattern := regexp.MustCompile(`<a2as:(\w+):([0-9a-fA-F]+)>(.*?)</a2as:...>`)
   ```

3. **内存分配**：预分配切片容量
   ```go
   result := make([]map[string]interface{}, 0, len(messages)+1)
   ```

### Security Considerations

1. **密钥安全**：`sharedSecret` 应通过安全配置管理，不要硬编码
2. **日志脱敏**：不记录完整密钥或敏感内容
3. **错误信息**：不泄露内部实现细节

## Limitations and Notes

### 已知限制
1. **不支持重放攻击防护**：无 Nonce/时间戳验证
2. **不支持密钥轮换**：单一共享密钥
3. **依赖 Agent 配合**：Authenticated Prompts 需要 Agent 侧签名
4. **仅支持 OpenAI 兼容格式**：依赖 `messages` 数组结构

### 使用建议
1. 生产环境务必启用 Behavior Certificates 限制工具调用
2. 敏感场景建议同时启用 Authenticated Prompts
3. `sharedSecret` 应使用环境变量或密钥管理系统
4. 定期审计 `allowedTools` 白名单

## References

- [OWASP A2AS 论文](https://arxiv.org/abs/2510.13825)
- [Higress WASM 插件开发指南](https://higress.io/docs/plugins/wasm-go)
- [HMAC-SHA256 RFC 2104](https://datatracker.ietf.org/doc/html/rfc2104)
