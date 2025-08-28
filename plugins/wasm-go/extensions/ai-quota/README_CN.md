---
title: AI 配额管理
keywords: [ AI网关, AI配额 ]
description: AI 配额管理插件配置参考
---

## 功能说明

`ai-quota` 插件实现基于用户身份的AI配额管理，支持JWT token身份验证和精确的配额控制。插件采用双Redis Key架构设计，分别存储配额总数和已使用量，能够精确跟踪和控制用户的配额使用情况。

插件从请求头中获取JWT token，解码后提取用户ID作为配额限制的key。管理操作需要通过指定的请求头和密钥进行验证。

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`750`

## 核心特性

- **双Redis Key架构**：分别存储配额总数和已使用量，计算剩余配额
- **JWT身份验证**：从JWT token中提取用户身份信息
- **灵活的配额扣减机制**：基于请求头触发配额扣减
- **完整的管理接口**：支持配额总数和已使用量的查询、刷新、增减操作
- **Redis集群支持**：兼容Redis单机和集群模式
- **GitHub项目关注管理**：支持多项目关注状态管理和验证
- **模型列表展示**：支持通过 `/ai-gateway/api/v1/models` 端点展示可配置provider的可用模型列表，包含详细的模型属性信息
- **模型权限管理**（新增）：支持基于用户身份的细粒度模型访问控制
- **受限模型配置**：可配置需要特殊权限的模型列表
- **智能权限缓存**：高性能的权限验证，支持可配置TTL的内存缓存，确保权限变更及时生效
- **权限管理接口**：提供权限设置和查询的管理接口

## 工作原理

### 配额计算逻辑
```
剩余配额 = 配额总数 - 已使用量
```

### Redis Key结构
- `{redis_key_prefix}{user_id}` - 存储用户的配额总数
- `{redis_used_prefix}{user_id}` - 存储用户的已使用量
- `{redis_star_prefix}{employee_number}` - 存储用户的GitHub关注项目列表（当启用star_check_management时）

### 缓存机制
插件采用带TTL的内存缓存机制来优化性能和确保数据一致性：
- **缓存读取**：首次读取权限时从Redis获取并缓存到内存中，后续请求直接使用内存缓存
- **TTL控制**：通过 `cache_ttl_seconds` 配置缓存过期时间，默认60秒
- **缓存失效**：权限设置后立即清除相关缓存，确保下次访问获取最新数据
- **一致性保证**：最坏情况下权限变更会在TTL时间内生效，平衡了性能和一致性需求

### 配额扣减机制
插件从请求体中提取模型名称，根据 `model_quota_weights` 配置确定扣减额度：
- 如果模型在 `model_quota_weights` 中配置了权重值，则按权重扣减配额
- 如果模型未在 `model_quota_weights` 中配置，则扣减额度为 0（不扣减配额）
- 只有当请求包含指定的请求头和值时，才会真正扣减配额

## 配置说明

| 名称                    | 数据类型   | 填写要求 | 默认值                 | 描述                           |
|------------------------|-----------|----------|------------------------|--------------------------------|
| `quota_management`     | object    | 选填     | -                      | 配额管理配置                    |
| `star_check_management` | object   | 选填     | -                      | GitHub项目关注检查配置          |
| `token_header`         | string    | 选填     | authorization          | 存储JWT token的请求头名称       |
| `admin_header`         | string    | 选填     | x-admin-key            | 管理操作验证用的请求头名称       |
| `admin_key`            | string    | 必填     | -                      | 管理操作验证用的密钥            |
| `admin_path`           | string    | 选填     | /quota                 | 管理quota请求path前缀           |
| `restricted_models`    | array     | 选填     | []                     | 需要权限控制的模型列表（新增）       |
| `permission_management`| object    | 选填     | -                      | 权限管理配置（新增）                |
| `providers`            | array     | 选填     | -                      | 多provider配置，用于模型列表展示 |
| `redis`                | object    | 是       | -                      | redis相关配置                  |

`quota_management`中每一项的配置字段说明

| 配置项                 | 类型       | 必填 | 默认值                 | 说明                           |
|------------------------|-----------|------|------------------------|--------------------------------|
| `user_level_enabled`   | boolean   | 选填 | false                  | 是否启用针对单个用户的配额控制    |
| `deduct_header`        | string    | 选填 | x-quota-identity       | 扣减配额的触发请求头名称        |
| `deduct_header_value`  | string    | 选填 | user                   | 扣减配额的触发请求头值          |
| `redis_key_prefix`     | string    | 选填 | chat_quota:            | 配额总数的redis key前缀         |
| `redis_used_prefix`    | string    | 选填 | chat_quota_used:       | 已使用量的redis key前缀         |
| `admin_quota_path`     | string    | 选填 | /check-quota           | 配额权限管理接口路径前缀         |
| `redis_quota_prefix`   | string    | 选填 | quota_check:           | 配额权限的redis key前缀         |
| `model_quota_weights`  | object    | 选填 | {}                     | 模型配额权重配置，指定每个模型的扣减额度 |
| `cache_ttl_seconds`    | integer   | 选填 | 60                     | 权限缓存过期时间（秒），用于控制缓存一致性 |

`redis`中每一项的配置字段说明

| 配置项       | 类型   | 必填 | 默认值                                                     | 说明                                                                                         |
| ------------ | ------ | ---- | ---------------------------------------------------------- | ---------------------------                                                                  |
| service_name | string | 必填 | -                                                          | redis服务名，带服务类型的完整 FQDN 名称，如my-redis.dns，redis.my-ns.svc.cluster.local |
| service_port | int    | 选填 | 静态服务默认值80；其他服务默认值6379                             | redis服务端口                                                                               |
| username     | string | 选填 | -                                                          | redis 用户名                                                                                 |
| password     | string | 选填 | -                                                          | redis 密码                                                                                   |
| timeout      | int    | 选填 | 1000                                                       | redis连接超时时间，单位毫秒                                                                     |
| database     | int    | 选填 | 0                                                          | 使用的数据库 ID，例如，配置为1，对应`SELECT 1`                                                    |

### GitHub项目关注检查配置

| 配置项        | 类型    | 必填 | 默认值 | 说明                                           |
|---------------|---------|------|--------|------------------------------------------------|
| `enabled`     | boolean | 选填 | false  | 是否启用GitHub项目关注检查                     |
| `user_level_enabled` | boolean | 选填 | false  | 是否启用针对单个用户的独立控制                     |
| `redis_star_prefix` | string | 选填 | chat_quota_star: | GitHub关注项目的redis key前缀（存储employee_number -> 项目列表） |
| `admin_stargazer_path` | string | 选填 | /check-star | star检查权限管理接口路径前缀                     |
| `redis_stargazer_prefix` | string | 选填 | star_check: | star检查权限的redis key前缀（存储employee_number -> enabled状态） |
| `target_repo` | string  | 选填 | -      | 目标检查的仓库（例如："zgsm-ai.zgsm"）         |

## 配置示例

### 基本配置
```yaml
quota_management:
  user_level_enabled: false
  deduct_header: "x-quota-identity"
  deduct_header_value: "user"
  redis_key_prefix: "chat_quota:"
  redis_used_prefix: "chat_quota_used:"
  admin_quota_path: "/check-quota"
  redis_quota_prefix: "quota_check:"
  model_quota_weights:
    'gpt-3.5-turbo': 1
    'gpt-4': 2
    'gpt-4-turbo': 3
    'gpt-4o': 4
  cache_ttl_seconds: 60
star_check_management:
  enabled: false
  user_level_enabled: false
  redis_star_prefix: "chat_quota_star:"
  admin_stargazer_path: "/check-star"
  redis_stargazer_prefix: "star_check:"
  target_repo: "zgsm-ai.zgsm"
token_header: "authorization"
admin_header: "x-admin-key"
admin_key: "your-admin-secret"
admin_path: "/quota"
# 权限管理配置（新增）
restricted_models:
  - "gpt-4"
  - "gpt-4-turbo"
  - "claude-3-opus"
  - "deepseek-v3"
permission_management:
  redis_permission_prefix: "model_perm:"
  admin_permission_path: "/model-permission"
redis:
  service_name: redis-service.default.svc.cluster.local
  service_port: 6379
  timeout: 2000
```

### 启用GitHub关注检查的配置
```yaml
quota_management:
  user_level_enabled: true
  deduct_header: "x-quota-identity"
  deduct_header_value: "user"
  redis_key_prefix: "chat_quota:"
  redis_used_prefix: "chat_quota_used:"
  admin_quota_path: "/check-quota"
  redis_quota_prefix: "quota_check:"
  model_quota_weights:
    'deepseek-chat': 1
    'deepseek-r1': 3
    'gpt-4': 10
    'gpt-3.5-turbo': 2
  cache_ttl_seconds: 30
star_check_management:
  enabled: true
  user_level_enabled: true
  redis_star_prefix: "chat_quota_star:"
  admin_stargazer_path: "/check-star"
  redis_stargazer_prefix: "star_check:"
  target_repo: "zgsm-ai.zgsm"
token_header: "authorization"
admin_header: "x-admin-key"
admin_key: "your-admin-secret"
admin_path: "/quota"
# 多provider配置，用于模型列表展示
providers:
  - id: openai-provider
    type: openai
    models:
      - name: "gpt-4"
        maxTokens: 8192
        contextWindow: 128000
        supportsImages: true
        supportsComputerUse: false
        supportsPromptCache: true
        supportsReasoningBudget: false
        description: "强大的多模态AI模型，支持图像理解和复杂推理"
      - name: "gpt-3.5-turbo"
        maxTokens: 4096
        contextWindow: 16385
        supportsImages: false
        supportsComputerUse: false
        supportsPromptCache: false
        supportsReasoningBudget: false
        description: "经济高效的对话模型，适合一般文本处理任务"
  - id: deepseek-provider
    type: deepseek
    models:
      - name: "deepseek-r1"
        maxTokens: 32768
        contextWindow: 65536
        supportsImages: false
        supportsComputerUse: false
        supportsPromptCache: true
        supportsReasoningBudget: true
        requiredReasoningBudget: true
        maxThinkingTokens: 32000
        description: "支持深度推理的模型，具备强大的逻辑分析能力"
      - name: "deepseek-chat"
        maxTokens: 16384
        contextWindow: 32768
        supportsImages: false
        supportsComputerUse: false
        supportsPromptCache: false
        supportsReasoningBudget: false
        description: "优化的对话模型，提供流畅的交互体验"
redis:
  service_name: "local-redis.static"
  service_port: 80
  timeout: 2000
```

**说明**: 当 `star_check_management.enabled` 设置为 `true` 时，用户必须先关注指定的 GitHub 项目才能使用AI服务。系统会检查用户的星标项目列表中是否包含 `target_repo` 配置的项目。

### 用户级别配额控制说明

当启用用户级别配额控制时 (`quota_management.user_level_enabled: true`)，系统将为每个用户提供独立的配额控制开关：

- **全局默认**: 默认情况下，所有用户都禁用配额控制
- **个人控制**: 管理员可以为特定用户启用配额控制，该用户的请求将进行配额检查和扣减
- **管理接口**: 提供API接口用于查询和设置用户的配额控制状态

### 模型权重配置说明

`model_quota_weights` 配置项用于指定不同模型的配额扣减权重：

- **键**: 模型名称（如 'gpt-3.5-turbo', 'gpt-4' 等）
- **值**: 扣减权重（正整数）

示例配置说明：
- `gpt-3.5-turbo` 每次调用扣减 1 个配额
- `gpt-4` 每次调用扣减 2 个配额
- `gpt-4-turbo` 每次调用扣减 3 个配额
- `gpt-4o` 每次调用扣减 4 个配额
- 未配置的模型（如 `claude-3`）扣减 0 个配额（不限制）


## JWT Token 格式

插件期望从指定的请求头中获取JWT token，token解码后应包含用户ID信息。token格式：

```json
{
  "id": "user123",
  "other_claims": "..."
}
```

插件会从token的`id`字段提取用户ID作为配额限制的key。

**权限管理中的员工号提取（新增）**：

插件现在支持从JWT token的name字段中提取员工号。支持的格式：
- `Username (EmployeeNumber)` - 如 `张三 (85054712)`
- `EmployeeNumber` - 如 `85054712`

提取的员工号将用于权限验证。


## API接口

### 用户配额检查
#### 完整的API端点列表

| 路径                                  | 方法   | 用途说明                  |
|---------------------------------------|--------|--------------------------|
| `/model-permission/set`               | POST   | 设置用户模型权限          |
| `/model-permission/query`             | GET    | 查询用户模型权限          |
| `/check-star/set`                     | POST   | 设置用户star检查权限      |
| `/check-star`                         | GET    | 查询用户star检查权限      |
| `/check-quota/set`                    | POST   | 设置用户配额控制权限      |
| `/check-quota`                        | GET    | 查询用户配额控制权限      |
| `/quota`                              | GET    | 查询配额总数              |
| `/quota/refresh`                      | POST   | 刷新配额总数              |
| `/quota/delta`                        | POST   | 增减配额总数              |
| `/quota/used`                         | GET    | 查询已使用量              |
| `/quota/used/refresh`                 | POST   | 刷新已使用量              |
| `/quota/used/delta`                   | POST   | 增减已使用量              |
| `/quota/star`                         | GET    | 查询GitHub关注状态        |
| `/quota/star/projects/set`            | POST   | 设置用户关注项目列表      |

#### 接口请求/响应示例

##### 设置模型权限
```bash
curl -X POST "https://example.com/model-permission/set" \
  -H "x-admin-key: your-admin-secret" \
  -d "employee_number=85054712&models=[\"gpt-4\",\"claude-3-opus\"]"
```

响应：
```json
{
  "code": "ai-quota.set_model_permission",
  "message": "set model permission successful",
  "success": true
}
```

##### 查询模型权限
```bash
curl -X GET "https://example.com/model-permission/query?employee_number=85054712" \
  -H "x-admin-key: your-admin-secret"
```

响应：
```json
{
  "code": "ai-quota.query_model_permission",
  "message": "query model permission successful",
  "success": true,
  "data": {
    "employee_number": "85054712",
    "models": ["gpt-4", "claude-3-opus"]
  }
}
```

##### 设置star检查权限
```bash
curl -X POST "https://example.com/check-star/set" \
  -H "x-admin-key: your-admin-secret" \
  -d "employee_number=85054712&enabled=true"
```

响应：
```json
{
  "code": "ai-quota.set_star_permission",
  "message": "set star check permission successful",
  "success": true,
  "data": {
    "employee_number": "85054712",
    "enabled": true
  }
}
```

##### 查询star检查权限
```bash
curl -X GET "https://example.com/check-star?employee_number=85054712" \
  -H "x-admin-key: your-admin-secret"
```

响应：
```json
{
  "code": "ai-quota.query_star_permission",
  "message": "query star check permission successful",
  "success": true,
  "data": {
    "employee_number": "85054712",
    "enabled": true
  }
}
```

##### 设置配额控制权限
```bash
curl -X POST "https://example.com/check-quota/set" \
  -H "x-admin-key: your-admin-secret" \
  -d "employee_number=85054712&enabled=true"
```

响应：
```json
{
  "code": "ai-quota.set_quota_permission",
  "message": "set quota control permission successful",
  "success": true,
  "data": {
    "employee_number": "85054712",
    "enabled": true
  }
}
```

##### 查询配额控制权限
```bash
curl -X GET "https://example.com/check-quota?employee_number=85054712" \
  -H "x-admin-key: your-admin-secret"
```

响应：
```json
{
  "code": "ai-quota.query_quota_permission",
  "message": "query quota control permission successful",
  "success": true,
  "data": {
    "employee_number": "85054712",
    "enabled": true
  }
}
```

##### 查询配额总数
```bash
curl -H "x-admin-key: your-admin-secret" \
  "https://example.com/quota?user_id=user123"
```

响应：
```json
{
  "code": "ai-gateway.queryquota",
  "message": "query quota successful",
  "success": true,
  "data": {
    "user_id": "user123",
    "quota": 10000,
    "type": "total_quota"
  }
}
```

##### 刷新配额总数
```bash
curl -X POST "https://example.com/quota/refresh" \
  -H "x-admin-key: your-admin-secret" \
  -d "user_id=user123&quota=15000"
```

响应：
```json
{
  "code": "ai-quota.refresh_quota",
  "message": "refresh total quota successful",
  "success": true
}
```

##### 增减配额总数
```bash
curl -X POST "https://example.com/quota/delta" \
  -H "x-admin-key: your-admin-secret" \
  -d "user_id=user123&delta=500"
```

响应：
```json
{
  "code": "ai-quota.adjust_quota",
  "message": "adjust total quota successful",
  "success": true,
  "data": {
    "new_quota": 15500
  }
}
```

##### 查询已使用量
```bash
curl -H "x-admin-key: your-admin-secret" \
  "https://example.com/quota/used?user_id=user123"
```

响应：
```json
{
  "code": "ai-quota.query_used",
  "message": "query used quota successful",
  "success": true,
  "data": {
    "user_id": "user123",
    "used": 1200,
    "type": "used_quota"
  }
}
```

##### 刷新已使用量
```bash
curl -X POST "https://example.com/quota/used/refresh" \
  -H "x-admin-key: your-admin-secret" \
  -d "user_id=user123&used=1000"
```

响应：
```json
{
  "code": "ai-quota.refresh_used",
  "message": "refresh used quota successful",
  "success": true
}
```

##### 增减已使用量
```bash
curl -X POST "https://example.com/quota/used/delta" \
  -H "x-admin-key: your-admin-secret" \
  -d "user_id=user123&delta=200"
```

响应：
```json
{
  "code": "ai-quota.adjust_used",
  "message": "adjust used quota successful",
  "success": true,
  "data": {
    "new_used": 1200
  }
}
```

##### 查询GitHub关注状态
```bash
curl -H "x-admin-key: your-admin-secret" \
  "https://example.com/quota/star?user_id=user123"
```

响应：
```json
{
  "code": "ai-quota.query_star_status",
  "message": "query GitHub star status successful",
  "success": true,
  "data": {
    "user_id": "user123",
    "starred": true
  }
}
```

##### 设置关注项目列表
```bash
curl -X POST "https://example.com/quota/star/projects/set" \
  -H "x-admin-key: your-admin-secret" \
  -d "employee_number=85054712&projects=[\"repo1\",\"repo2\"]"
```

响应：
```json
{
  "code": "ai-quota.set_star_projects",
  "message": "set starred projects successful",
  "success": true
}
```

**行为**:
1. 从JWT token中提取用户ID
2. 如果启用了 `star_check_management`，检查用户的GitHub关注项目列表中是否包含 `target_repo`
3. 从请求体中提取模型名称
4. 根据 `model_quota_weights` 配置确定所需配额
5. 检查用户的剩余配额是否足够（总数 - 已使用量 >= 所需配额）
6. 如果配额足够且包含扣减触发头，则按模型权重扣减配额
7. 如果模型未配置权重，则不扣减配额直接放行

**GitHub关注检查**:
- 当 `star_check_management.enabled` 设置为 `true` 时，会首先检查用户是否关注了配置的GitHub项目
- 如果用户的星标项目列表中不包含 `target_repo` 配置的项目，将返回403错误，提示用户需要关注对应项目
- 只有通过GitHub关注检查后，才会继续进行配额检查和扣减

### 模型列表接口

**路径**: `/ai-gateway/api/v1/models`

**方法**: GET

**描述**: 返回所有配置的provider的可用模型列表组合。此端点不需要身份验证，由插件本地处理。

**响应示例**:
```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4",
      "object": "model",
      "created": 1686935002,
      "owned_by": "openai",
      "max_tokens": 8192,
      "context_window": 128000,
      "supports_images": true,
      "supports_computer_use": false,
      "supports_prompt_cache": true,
      "supports_reasoning_budget": false,
      "description": "强大的多模态AI模型，支持图像理解和复杂推理"
    },
    {
      "id": "deepseek-r1",
      "object": "model",
      "created": 1686935002,
      "owned_by": "unknown",
      "max_tokens": 32768,
      "context_window": 65536,
      "supports_images": false,
      "supports_computer_use": false,
      "supports_prompt_cache": true,
      "supports_reasoning_budget": true,
      "required_reasoning_budget": true,
      "max_thinking_tokens": 32000,
      "description": "支持深度推理的模型，具备强大的逻辑分析能力"
    }
  ]
}
```

**说明**:
- 在多provider模式下，如果多个provider定义了相同的模型名称，第一个provider的配置优先
- `owned_by` 字段会根据provider类型自动设置（openai → "openai", qwen → "alibaba" 等）
- 模型配置支持对象格式，包含详细属性如 maxTokens、contextWindow、supportsImages 等
- 也支持简化的字符串格式，自动应用默认属性值
- 此端点由插件本地处理，不会转发请求到上游服务

## 错误处理

### 常见错误响应

| 状态码 | 错误代码 | 说明 |
|--------|----------|------|
| 401 | `ai-gateway.no_token` | 未提供JWT token |
| 401 | `ai-gateway.invalid_token` | JWT token格式无效 |
| 401 | `ai-gateway.token_parse_failed` | JWT token解析失败 |
| 401 | `ai-gateway.no_userid` | JWT token中未找到用户ID |
| 403 | `ai-gateway.unauthorized` | 管理接口认证失败 |
| 403 | `ai-gateway.star_required` | 需要先关注GitHub项目 |
| 403 | `ai-gateway.noquota` | 配额不足 |
| 400 | `ai-gateway.invalid_params` | 请求参数无效 |
| 500 | `ai-gateway.invalid_quota_format` | 配额格式无效 |
| 500 | `ai-gateway.invalid_quota_value` | 配额值无效 |
| 503 | `ai-gateway.error` | Redis连接错误 |
| 503 | `ai-gateway.redis_error` | Redis操作错误 |

**错误响应示例**:
```json
{
  "code": "ai-gateway.noquota",
  "message": "Request denied by ai quota check, insufficient quota. Required: 1, Remaining: 0",
  "success": false
}
```

**成功响应示例**:
```json
{
  "code": "ai-gateway.refreshquota",
  "message": "refresh quota successful",
  "success": true
}
```

## 注意事项

1. **JWT格式要求**: JWT token必须包含用户ID信息，插件会从token的claims中提取`id`字段
2. **Redis连接**: 确保Redis服务可用，插件依赖Redis存储配额信息
3. **管理接口安全**: 管理接口的认证密钥需要妥善保管，避免泄露
4. **配额精度**: 配额计算基于整数，不支持小数
5. **并发安全**: 插件支持高并发场景下的配额管理

注意：管理操作不需要携带JWT token，只需要在指定的请求头中提供正确的管理密钥即可。