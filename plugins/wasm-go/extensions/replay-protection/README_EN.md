---
title: Replay Attack Prevention
keywords: [higress, replay-protection]
description: Configuration reference for the replay attack prevention plugin
---

## Function Description

The replay prevention plugin prevents request replay attacks by verifying the one-time random number (nonce) in the request. Each request needs to carry a unique nonce value, and the server will record and verify the uniqueness of this value to prevent the request from being maliciously replayed.

It specifically includes the following functions:

- **Mandatory or optional nonce verification**: It can be determined according to the configuration whether to force the request to carry a nonce value.
- **Redis-based nonce uniqueness verification**: Store and verify the nonce value through Redis to ensure its uniqueness.
- **Configurable nonce validity period**: Supports setting the validity period of the nonce, which will automatically expire after the expiration.
- **Nonce format and length verification**: Supports verifying the format (Base64) and length of the nonce value.
- **Custom error response**: Supports configuring the status code and error message when the request is rejected.
- **Customizable nonce request header**: You can customize the name of the request header that carries the nonce.

## Configuration Fields

| Name               | Data Type | Required | Default Value                   | Description                              |
| ------------------ | --------- | -------- | ----------------------------- | ---------------------------------------- |
| `force_nonce`      | bool      | No       | true                          | Whether to force the request to carry a nonce value. |
| `nonce_header`     | string    | No       | `X-Higress-Nonce`             | Specify the name of the request header that carries the nonce value. |
| `nonce_ttl`        | int       | No       | 900                           | The validity period of the nonce (unit: seconds). |
| `nonce_min_length` | int       | No       | 8                             | The minimum length of the nonce value. |
| `nonce_max_length` | int       | No       | 128                           | The maximum length of the nonce value. |
| `reject_code`      | int       | No       | 429                           | The status code returned when the request is rejected. |
| `reject_msg`       | string    | No       | `Replay Attack Detected`      | The error message returned when the request is rejected. |
| `validate_base64`  | bool      | No       | false                         | Whether to verify the base64 encoding format of the nonce. |
| `redis`            | Object    | Yes      | -                             | Redis-related configuration. |

Configuration field description for each item in `redis`

| Name           | Data Type | Required | Default Value              | Description                                    |
| -------------- | --------- | -------- | -------------------------- | ---------------------------------------------- |
| `service_name` | string    | Yes      | -                          | The name of the Redis service, used to store nonce values. |
| `service_port` | int       | No       | 6379                       | The port of the Redis service. |
| `timeout`      | int       | No       | 1000                       | The timeout for Redis operations (unit: milliseconds). |
| `key_prefix`   | string    | No       | `replay-protection`        | The key prefix in Redis, used to distinguish different nonce keys. |

## Configuration Example

The following is a complete configuration example of the replay attack prevention plugin:

```yaml
force_nonce: true
nonce_header: "X-Higress-Nonce"    # Specify the name of the nonce request header
nonce_ttl: 900                    # The validity period of the nonce, set to 900 seconds
nonce_min_length: 8               # The minimum length of the nonce
nonce_max_length: 128             # The maximum length of the nonce
validate_base64: true             # Whether to enable base64 format verification
reject_code: 429                  # The HTTP status code returned when the request is rejected
reject_msg: "Replay Attack Detected"  # The content of the error message returned when the request is rejected
redis:
  service_name: redis.static       # The name of the Redis service
  service_port: 80                # The port used by the Redis service
  timeout: 1000                   # The timeout for Redis operations (unit: milliseconds)
  key_prefix: "replay-protection" # The prefix of the keys in Redis
```

## Usage Instructions

### Request Header Requirements

| Request Header Name        | Required or Not                    | Description                                                  |
| -------------------------- | --------------------------------- | ------------------------------------------------------------ |
| `X-Higress-Nonce`          | Determined according to the `force_nonce` configuration | The randomly generated nonce value carried in the request, which needs to conform to the Base64 format. |

> **Note**: You can customize the name of the request header through the `nonce_header` configuration, and the default value is `X-Higress-Nonce`.

### Usage Example

```bash
# Generate nonce
nonce=$(openssl rand -base64 32)

# Send request
curl -X POST 'https://api.example.com/path' \
  -H "X-Higress-Nonce: $nonce" \
  -d '{"key": "value"}'
```

## Return Results

```json
{
    "code": 429,
    "message": "Replay Attack Detected"
}
```

## Error Response Example

| Error Scenario                   | Status Code | Error Message                  |
| ------------------------------ | ----------- | ------------------------------ |
| Missing the nonce request header | 400         | `Missing Required Header`      |
| The nonce length does not meet the requirements | 400         | `Invalid Nonce`                |
| The nonce format does not conform to Base64 | 400         | `Invalid Nonce`                |
| The nonce has been used (replay attack) | 429         | `Replay Attack Detected`       | 