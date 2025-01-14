---
title: 防重放攻击
keywords: [higress,nonce-protection]
description: 防重放攻击插件配置参考
---

## 简介

Nonce (Number used ONCE) 防重放插件通过验证请求中的一次性随机数来防止请求重放攻击。每个请求都需要携带一个唯一的 nonce 值，服务器会记录并校验这个值的唯一性，从而防止请求被恶意重放。

## 功能说明

- 强制或可选的 nonce 校验
- 基于 Redis 的 nonce 唯一性验证
- 可配置的 nonce 有效期
- nonce 格式和长度校验

## 配置说明

| 配置项               | 类型 | 必填 | 默认值 | 说明 |
|-------------------|------|------|--------|-----|
| force_nonce       | bool | 否 | true | 是否强制要求 nonce |
| nonce_ttl         | int | 否 | 900 | nonce 有效期（单位：秒） |
| nonce_min_length  | int | 否 | 8 | nonce 最小长度 |
| nonce_max_length  | int | 否 | 128 | nonce 最大长度 |
| reject_code       | int | 否 | 429 | 拒绝请求时的状态码 |
| reject_msg        | string | 否 | "Duplicate nonce" | 拒绝请求时的错误信息 |
| redis.serviceName | string | 是 | - | Redis 服务名称 |
| redis.servicePort | int | 否 | 6379 | Redis 服务端口 |
| redis.timeout     | int | 否 | 1000 | Redis 操作超时时间（毫秒） |
| redis.keyPrefix   | string | 否 | "replay-protection" | Redis key 前缀 |

## 配置示例

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: replay-protection
  namespace: higress-system
spec:
  defaultConfig:
    force_nonce: true
    nonce_ttl: 900
    nonce_min_length: 8
    nonce_max_length: 128
    redis:
      serviceName: "redis.higress"
      servicePort: 6379
      timeout: 1000
      keyPrefix: "replay-protection"
url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/replay-protection:v1.0.0
```

## 使用说明

### 请求头要求

| 请求头 | 是否必须 | 说明 |
|-------|---------|------|
| x-apigw-nonce | 由 force_nonce 配置决定 | 随机生成的 nonce 值，需符合 base64 编码格式 |


### 1. 测试环境配置

```yaml
# test-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-api
  namespace: default
spec:
  rules:
  - host: test.example.com
    http:
      paths:
      - path: /api/test
        pathType: Prefix
        backend:
          service:
            name: test-service
            port:
              number: 8080
---
# test-wasmplugin.yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: replay-protection
  namespace: higress-system
spec:
  defaultConfig:
    force_nonce: true
    nonce_ttl: 60        # 测试时设置短一点，比如60秒
    nonce_min_length: 8
    nonce_max_length: 128
    redis:
      serviceName: "redis.higress"  # 确保这个 Redis 服务可用
      servicePort: 6379
      timeout: 1000
      keyPrefix: "test-replay-protection"
  matchRules:
  - ingress:
    - default/test-api   # 匹配我们的测试 Ingress
url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/replay-protection:v1.0.0
```

### 2. 测试脚本

```bash
#!/bin/bash

# 测试 API 地址
API_URL="http://test.example.com/api/test"

# 测试用例1: 正常请求
test_normal_request() {
    echo "测试用例1: 正常请求"
    nonce=$(openssl rand -base64 32)
    echo "使用 nonce: $nonce"
    
    curl -X POST "$API_URL" \
        -H "x-apigw-nonce: $nonce" \
        -H "Host: test.example.com" \
        -d '{"test": "data"}'
    echo -e "\n"
}

# 测试用例2: 重放攻击
test_replay_attack() {
    echo "测试用例2: 重放攻击"
    nonce=$(openssl rand -base64 32)
    echo "使用 nonce: $nonce"
    
    # 第一次请求
    echo "第一次请求:"
    curl -X POST "$API_URL" \
        -H "x-apigw-nonce: $nonce" \
        -H "Host: test.example.com" \
        -d '{"test": "data"}'
    echo -e "\n"
    
    # 重放请求
    echo "重放请求:"
    curl -X POST "$API_URL" \
        -H "x-apigw-nonce: $nonce" \
        -H "Host: test.example.com" \
        -d '{"test": "data"}'
    echo -e "\n"
}

# 测试用例3: 无 nonce
test_without_nonce() {
    echo "测试用例3: 无 nonce"
    curl -X POST "$API_URL" \
        -H "Host: test.example.com" \
        -d '{"test": "data"}'
    echo -e "\n"
}

# 测试用例4: nonce 太短
test_short_nonce() {
    echo "测试用例4: nonce 太短"
    curl -X POST "$API_URL" \
        -H "x-apigw-nonce: abc" \
        -H "Host: test.example.com" \
        -d '{"test": "data"}'
    echo -e "\n"
}

# 运行所有测试
run_all_tests() {
    test_normal_request
    sleep 2
    test_replay_attack
    sleep 2
    test_without_nonce
    sleep 2
    test_short_nonce
}

# 执行测试
run_all_tests
```


### 3. 预期结果

1. **正常请求**：
```json
{
  "success": true,
  "data": "..."
}
```

2. **重放攻击**：
```json
{
  "code": 429,
  "message": "Request replay detected"
}
```

3. **无 nonce**：
```json
{
  "code": 400,
  "message": "Missing nonce header"
}
```

4. **nonce 太短**：
```json
{
  "code": 400,
  "message": "Invalid nonce length"
}
```



