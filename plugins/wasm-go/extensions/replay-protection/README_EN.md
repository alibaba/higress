---
title: Nonce Replay Protection 
keywords: [higress, replay-protection]
description: replay-protection config example
---


## Introduction

The Nonce (Number used ONCE) replay protection plugin prevents request replay attacks by validating a one-time random number in requests. Each request must carry a unique nonce value, which the server records and validates to prevent malicious request replay.

## Features

- Mandatory or optional nonce validation
- Redis-based nonce uniqueness verification
- Configurable nonce TTL
- Custom error responses
- Nonce format and length validation

## Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| force_nonce | bool | No | true | Whether to enforce nonce requirement |
| nonce_ttl | int | No | 900 | Nonce validity period (seconds) |
| nonce_min_length | int | No | 8 | Minimum nonce length |
| nonce_max_length | int | No | 128 | Maximum nonce length |

### Redis Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| serviceName | string | Yes | - | Redis service name |
| servicePort | int | No | 6379 | Redis service port |
| timeout | int | No | 1000 | Redis operation timeout (ms) |
| keyPrefix | string | No | "replay-protection" | Redis key prefix |

## Configuration Example

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

## Usage

### Required Headers

| Header | Required | Description |
|--------|----------|-------------|
| x-apigw-nonce | Depends on force_nonce | Random generated nonce value in base64 format |

### Usage Example

```bash
# Generate nonce
nonce=$(openssl rand -base64 32)

# Send request
curl -X POST 'https://api.example.com/path' \
  -H "x-apigw-nonce: $nonce" \
  -d '{"key": "value"}'
```

## Error Response

```json
{
    "code": 429,
    "message": "Duplicate nonce detected"
}
```

