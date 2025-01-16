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
| nonce_header          | string | No       | X-Mse-Nonce   | Request header name for the nonce              |
| nonce_ttl             | int    | No       | 900           | Nonce validity period (seconds)                 |
| nonce_min_length | int | No | 8 | Minimum nonce length |
| nonce_max_length | int | No | 128 | Maximum nonce length |
| reject_code       | int | No | 429 | error code when request rejected |
| reject_msg        | string | No | "Duplicate nonce" | error massage when request rejected  |
| validate_base64 | bool    | No   | false  | Whether to validate the base64 encoding format of the nonce. |
| redis.serviceName | string | Yes | - | Redis service name |
| redis.servicePort | int | No | 6379 | Redis service port |
| redis.timeout | int | No | 1000 | Redis operation timeout (ms) |
| redis.keyPrefix | string | No | "replay-protection" | Redis key prefix |

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
    nonce_header:""
    nonce_min_length: 8
    nonce_max_length: 128
    validate_base64: true
    reject_code: 429
    reject_msg: "Duplicate nonce" 
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
| `X-Mse-Nonce` | Depends on force_nonce | Random generated nonce value in base64 format |

>Note: The default nonce header is X-Mse-Nonce. You can customize it using the nonce_header configuration.

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
## Error Response Examples

| Error Scenario              | Status Code | Error Message               |
|-----------------------------|-------------|-----------------------------|
| Missing nonce header         | `400`       | `Missing nonce header`        |
| Nonce length not valid       | `400`       | `Invalid nonce length`        |
| Nonce not Base64-encoded     | `400`       | `Invalid nonce format`        |
| Duplicate nonce (replay attack) | `429`       | `Duplicate nonce`             |
