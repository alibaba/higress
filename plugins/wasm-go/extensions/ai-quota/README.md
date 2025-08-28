---
title: AI Quota Management
keywords: [AI Gateway, AI Quota]
description: AI quota management plugin configuration reference
---

## Function Description

The `ai-quota` plugin implements AI quota management based on user identity with JWT token authentication and precise quota control. It features a dual Redis key architecture that stores total quota and used quota separately, enabling precise tracking and control of user quota consumption.

The plugin extracts JWT token from request headers, decodes it to extract user ID as the key for quota limiting. Administrative operations require verification through specified request headers and secret keys.

## Runtime Properties

Plugin execution phase: `default phase`
Plugin execution priority: `750`

## Key Features

- **Dual Redis Key Architecture**: Stores total quota and used quota separately, calculates remaining quota
- **JWT Authentication**: Extracts user identity information from JWT tokens
- **Flexible Quota Deduction**: Header-based quota deduction triggering
- **Complete Management APIs**: Supports query, refresh, and adjustment of total and used quotas
- **Redis Cluster Support**: Compatible with both Redis standalone and cluster modes
- **GitHub Project Star Management**: Supports multi-project star status management and verification
- **Model List Display**: Supports displaying available model lists via `/ai-gateway/api/v1/models` endpoint with configurable providers, including detailed model attribute information
- **Model Permission Management** (New): Supports fine-grained model access control based on user identity
- **Restricted Models Configuration**: Configurable list of models requiring special permissions
- **Intelligent Permission Caching**: High-performance permission validation with configurable TTL memory caching, ensuring timely effect of permission changes
- **Permission Management APIs**: Provides management APIs for permission setting and query

## How It Works

### Quota Calculation Logic
```
Remaining Quota = Total Quota - Used Quota
```

### Redis Key Structure
- `{redis_key_prefix}{user_id}` - Stores user's total quota
- `{redis_used_prefix}{user_id}` - Stores user's used quota
- `{redis_star_prefix}{employee_number}` - Stores user's GitHub starred projects list (when star_check_management is enabled)

### Cache Mechanism
The plugin uses a memory caching mechanism with TTL to optimize performance and ensure data consistency:
- **Cache Read**: On first permission read, fetch from Redis and cache in memory; subsequent reads use memory cache
- **TTL Control**: Configure cache expiration time via `cache_ttl_seconds` (default: 60 seconds)
- **Cache Invalidation**: Clear relevant cache immediately after permission changes, ensuring next read gets fresh data
- **Consistency Guarantee**: Permission changes take effect within TTL at worst, balancing performance and consistency

### Quota Deduction Mechanism
The plugin extracts model name from request body and determines deduction amount based on `model_quota_weights` configuration:
- If model has weight configured in `model_quota_weights`, deduct corresponding amount
- If model is not configured, deduction amount is 0 (no deduction)
- Quota is only deducted when request contains specified header and value

## Configuration Description

| Name                    | Data Type | Required | Default Value       | Description                                    |
|-------------------------|-----------|----------|---------------------|------------------------------------------------|
| `quota_management`      | object    | Optional | -                   | Quota management configuration                 |
| `star_check_management` | object    | Optional | -                   | GitHub star checking configuration             |
| `token_header`          | string    | Optional | authorization       | Request header name storing JWT token          |
| `admin_header`          | string    | Optional | x-admin-key         | Request header name for admin verification     |
| `admin_key`             | string    | Required | -                   | Secret key for admin operation verification    |
| `admin_path`            | string    | Optional | /quota              | Prefix for quota management request paths      |
| `restricted_models`     | array     | Optional | []                  | List of models requiring permission control (New) |
| `permission_management` | object    | Optional | -                   | Permission management configuration (New)      |
| `providers`             | array     | Optional | -                   | Multi-provider configuration for model lists   |
| `redis`                 | object    | Yes      | -                   | Redis related configuration                    |

Explanation of each configuration field in `quota_management`

| Configuration Item     | Data Type | Required | Default Value       | Description                                    |
|------------------------|-----------|----------|---------------------|------------------------------------------------|
| `user极 level_enabled` | boolean   | Optional | false               | Whether to enable per-user quota control       |
| `deduct_header`        | string    | Optional | x-quota-identity    | Header name triggering quota deduction         |
| `deduct_header_value`  | string    | Optional | user                | Header value triggering quota deduction        |
| `redis_key_prefix`     | string    | Optional | chat_quota:         | Redis key prefix for total quota               |
| `redis_used_prefix`    | string    | Optional | chat_quota_used:    | Redis key prefix for used quota                |
| `admin_quota_path`     | string    | Optional | /check-quota        | Path prefix for quota permission management APIs |
| `redis_quota_prefix`   | string    | Optional | quota_check:        | Redis key prefix for quota permissions         |
| `model_quota_weights`  | object    | Optional | {}                  | Model quota weight configuration               |
| `cache_ttl_seconds`    | integer   | Optional | 60                  | Permission cache expiration time (seconds) for cache consistency control |

Explanation of each configuration field in `redis`

| Configuration Item | Type   | Required | Default Value                                           | Explanation                                                                                             |
|--------------------|--------|----------|---------------------------------------------------------|---------------------------------------------------------------------------------------------------------|
| service_name       | string | Required | -                                                       | Redis service name, full FQDN name with service type, e.g., my-redis.dns, redis.my-ns.svc.cluster.local |
| service_port       | int    | No       | Static service default: 80; others: 6379                | Service port for the redis service                                                                      |
| username           | string | No       | -                                                       | Redis username                                                                                          |
| password           | string | No       | -                                                       | Redis password                                                                                          |
| timeout            | int    | No极     | 1000                                                    | Redis connection timeout in milliseconds                                                                |
| database           | int    | No       | 0                                                       | The database ID used, for example, configured as 1, corresponds to `SELECT 1`.                          |

### GitHub Star Check Management Configuration

| Name        | Data Type | Required | Default Value | Description                                      |
|-------------|-----------|----------|---------------|--------------------------------------------------|
| `enabled`   | boolean   | No       | false         | Whether to enable GitHub star checking           |
| `user_level_enabled` | boolean | No | false    | Whether to enable per-user control               |
| `redis_star_prefix` | string | No | chat_quota_star: | Redis key prefix for GitHub star projects (employee_number -> starred projects) |
| `admin_stargazer_path` | string | No | /check-star | Path prefix for star check permission management APIs |
| `redis_stargazer_prefix` | string | No | star_check: | Redis key prefix for star check permissions (employee_number -> enabled status) |
| `target_repo` | string  | No       | -             | Target repository for star checking (e.g., "zgsm-ai.zgsm") |

## Configuration Examples

### Basic Configuration
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
# Permission management configuration (New)
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

### Configuration with GitHub Star Check Enabled
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
    'deepseek-chat': 极
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
# Multi-provider configuration for model list display
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
        description: "Powerful multimodal AI model with image understanding and complex reasoning capabilities"
      - name: "gpt-3.5-turbo"
        maxTokens: 4096
        contextWindow: 16385
        supportsImages: false
        supportsComputerUse: false
        supportsPromptCache: false
        supportsReasoningBudget: false
        description: "Cost-effective conversational model suitable for general text processing tasks"
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
        description: "Deep reasoning model with powerful logical analysis capabilities"
      - name: "deepseek-chat"
        maxTokens: 16384
        contextWindow: 32768
        supportsImages: false
        supportsComputerUse: false
        supportsPromptCache: false
        supportsReasoningBudget: false
        description: "Optimized conversational model providing smooth interactive experience"
redis:
  service_name: "local-redis.static"
  service_port: 80
  timeout: 2000
```

**Note**: When `star_check_management.enabled` is set to `true`, users must star the specified GitHub project before using AI services. The system will check if the user's starred projects list includes the project configured in `target_repo`.

### Per-User Quota Control Explanation
When user-level quota control is enabled (`quota_management.user_level_enabled: true`), the system provides an independent quota control switch for each user:
- **Global Default**: By default, quota control is disabled for all users
- **Per-User Control**: Admins can enable quota control for specific users, whose requests will undergo quota checks and deductions
- **Management APIs**: Provides APIs for querying and setting user's quota control status

### Model Weight Configuration Explanation
The `model_quota_weights` configuration specifies quota deduction weights for different models:
- **Key**: Model name (e.g., 'gpt-3.5-turbo', 'gpt-4', etc.)
- **Value**: Deduction weight (positive integer)

Example configuration explanation:
- `gpt-3.5-turbo`: Deduct 1 quota per call
- `gpt-4`: Deduct 2 quotas per call
- `gpt-4-turbo`: Deduct 3 quotas per call
- `gpt-4o`: Deduct 4 quotas per call
- Unconfigured models (e.g., `claude-3`) deduct 0 quotas (no limitation)

## JWT Token Format
The plugin expects to obtain JWT token from specified request header. After decoding, the token should contain user ID information. Token format:
```json
{
  "id": "user123",
  "other_claims": "..."
}
```

The plugin will extract user ID from the `id` field of the token as the key for quota limiting.

**Employee number extraction in permission management (New)**:
The plugin now supports extracting employee number from the name field of JWT token. Supported formats:
- `Username (EmployeeNumber)` - e.g., `张三 (85054712)`
- `EmployeeNumber` - e.g., `85054712`

The extracted employee number will be used for permission verification.

## API Reference

### User Quota Check
#### Complete API Endpoints List

| Path                                  | Method | Usage Description                      |
|---------------------------------------|--------|----------------------------------------|
| `/model-permission/set`               | POST   | Set user model permissions             |
| `/model-permission/query`             | GET    | Query user model permissions           |
| `/check-star/set`                     | POST   | Set user star check permission         |
| `/check-star`                         | GET    | Query user star check permission       |
| `/check-quota/set`                    | POST   | Set user quota control permission      |
| `/check-quota`                        | GET    | Query user quota control permission    |
| `/quota`                              | GET    | Query total quota                      |
| `/quota/refresh`                      | POST   | Refresh total quota                    |
| `/quota/delta`                        | POST   | Adjust total quota                     |
| `/quota/used`                         | GET    | Query used quota                       |
| `/quota/used/refresh`                 | POST   | Refresh used quota                     |
| `/quota/used/delta`                   | POST   | Adjust used quota                      |
| `/quota/star`                         | GET    | Query GitHub star status               |
| `/quota/star/projects/set`            | POST   | Set user starred projects list         |

#### Request/Response Examples

##### Set Model Permissions
```bash
curl -X POST "https://example.com/model-permission/set" \
  -H "x-admin-key: your-admin-secret" \
  -d "employee_number=85054712&models=[\"gpt-4\",\"claude-3-opus\"]"
```

Response:
```json
{
  "code": "ai-quota.set_model_permission",
  "message": "set model permission successful",
  "success": true
}
```

##### Query Model Permissions
```bash
curl -X GET "https://example.com/model-permission/query?employee_number=85054712" \
  -H "x-admin-key: your-admin-secret"
```

Response:
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

##### Set Star Check Permission
```bash
curl -X POST "https://example.com/check-star/s极et" \
  -H "x-admin-key: your-admin-secret" \
  -d "employee_number=85054712&enabled=true"
```

Response:
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

##### Query Star Check Permission
```bash
curl -X GET "https://example.com/check-star?employee_number=85054712" \
  -H "x-admin-key: your-admin-secret"
```

Response:
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

##### Set Quota Control Permission
```bash
curl -X POST "https://example.com/check-quota/set" \
  -H "x-admin-key: your-admin-secret" \
  -d "employee_number=85054712&enabled=true"
```

Response:
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

##### Query Quota Control Permission
```bash
curl -X GET "https://example.com/check-quota?employee_number=85054712" \
  -H "x-admin-key: your-admin-secret"
```

Response:
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

##### Query Total Quota
```bash
curl -H "x-admin-key: your-admin-secret" \
  "https://example.com/quota?user_id=user123"
```

Response:
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

##### Refresh Total Quota
```bash
curl -X POST "https://example.com/quota/refresh" \
  -H "x-admin-key: your-admin-secret" \
  -d "user_id=user123&quota=15000"
```

Response:
```json
{
  "code": "ai-quota.refresh_quota",
  "message": "refresh total quota successful",
  "success": true
}
```

##### Adjust Total Quota
```bash
curl -X POST "https://example.com/quota/delta" \
  -H "x-admin-key: your-admin-secret" \
  -d "user_id=user123&delta=500"
```

Response:
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

##### Query Used Quota
```bash
curl -H "x-admin-key: your-admin-secret" \
  "https://example.com/quota/used?user_id=user123"
```

Response:
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

##### Refresh Used Quota
```bash
curl -X POST "https://example.com/quota/used/refresh" \
  -H "x-admin-key: your-admin-secret" \
  -d "user_id=user123&used=1000"
```

Response:
```json
{
  "code": "ai-quota.refresh_used",
  "message": "refresh used quota successful",
  "success": true
}
```

##### Adjust Used Quota
```bash
curl -X POST "https://example.com/quota/used/delta" \
  -H "x-admin-key: your-admin-secret" \
  -d "user_id=user123&delta=200"
```

Response:
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

##### Query GitHub Star Status
```bash
curl -H "x-admin-key: your-admin-secret" \
  "https://example.com/quota/star?user_id=user123"
```

Response:
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

##### Set Starred Projects
```bash
curl -X POST "https://example.com/quota/star/projects/set" \
  -H "x-admin-key: your-admin-secret" \
  -d "employee_number=85054712&projects=[\"repo1\",\"repo2\"]"
```

Response:
```json
{
  "code": "ai-quota.set_star_projects",
  "message": "set starred projects successful",
  "success": true
}
```

**Behavior**:
1. Extract user ID from JWT token
2. If `star_check_management` is enabled, check if user has starred the GitHub project
3. Extract model name from request body
4. Determine required quota based on `model_quota_weights`
5. Check if user's remaining quota is sufficient (total - used >= required quota)
6. If quota is sufficient and deduction trigger header is present, deduct quota according to model weight
7. If model is not configured, allow without deduction

**GitHub Star Check**:
- When `star_check_management.enabled` is true, the system first checks if user has starred configured GitHub project
- If user's starred projects list doesn't include `target_repo`, returns 403 error asking user to star the project
- Only after passing GitHub star check does the system proceed with quota check and deduction

### Model List API

**Path**: `/ai-gateway/api/v1/models`

**Method**: GET

**Description**: Returns combined list of available models from all configured providers. This endpoint doesn't require authentication and is handled locally by the plugin.

**Response Example**:
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
      "description": "Powerful multimodal AI model with image understanding and complex reasoning capabilities"
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
      "description": "Deep reasoning model with powerful logical analysis capabilities"
    }
  ]
}
```

**Notes**:
- In multi-provider mode, if multiple providers define same model name, first provider's configuration takes precedence
- `owned_by` field is automatically set based on provider type (openai → "openai", qwen → "alibaba", etc.)
- Model configuration supports object format with detailed attributes like maxTokens, contextWindow, supportsImages, etc.
- Also supports simplified string format with default attribute values applied automatically
- This endpoint is handled locally and doesn't forward requests to upstream services

## Error Handling

### Common Error Responses

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 401 | `ai-gateway.no_token` | No JWT token provided |
| 401 | `ai-gateway.invalid_token` | Invalid JWT token format |
| 401 | `ai-gateway.token_parse_failed` | JWT token parse failed |
| 401 | `ai-gateway.no_userid` | User ID not found in JWT token |
| 403 | `ai-gateway.unauthorized` | Admin interface authentication failed |
| 403 | `ai-gateway.star_required` | GitHub project starring required |
| 403 | `ai-gateway.noquota` | Insufficient quota |
| 400 | `ai-gateway.invalid_params` | Invalid request parameters |
| 500 | `ai-gateway.invalid_quota_format` | Invalid quota format |
| 500 | `ai-gateway.invalid_quota_value` | Invalid quota value |
| 503 | `ai-gateway.error` | Redis connection error |
| 503 | `ai-gateway.redis_error` | Redis operation error |

**Error Response Example**:
```json
{
  "code": "ai-gateway.noquota",
  "message": "Request denied by ai quota check, insufficient quota. Required: 1, Remaining: 0",
  "success": false
}
```

**Success Response Example**:
```json
{
  "code": "ai-gateway.refreshquota",
  "message": "refresh quota successful",
  "success": true
}
```

## Notes

1. **JWT Format Requirements**: JWT token must contain user ID information. The plugin extracts `id` field from token claims.
2. **Redis Connection**: Ensure Redis service is available. The plugin relies on Redis for storing quota information.
3. **Management API Security**: Admin API keys should be properly secured to prevent leakage.
4. **Quota Precision**: Quota calculations are based on integers; decimal values are not supported.
5. **Concurrency Safety**: The plugin supports high-concurrency scenarios for quota management.

Note: Admin operations do not require JWT token; only provide correct admin secret in specified header.