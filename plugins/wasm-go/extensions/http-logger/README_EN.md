---
title: HTTP Request Logging
keywords: [higress, logging, debug]
description: HTTP request logging plugin configuration reference
---

## Feature Description

The `http-logger` plugin is used to log detailed HTTP request and response information in Higress logs, including request headers, request body, response headers, and response body, facilitating API debugging and monitoring.

## Execution Properties

Plugin execution phase: `Default Phase`
Plugin execution priority: `10`

## Configuration Fields

| Field               | Type | Required | Default | Description                                                                                           |
| ------------------- | ---- | -------- | ------- | ----------------------------------------------------------------------------------------------------- |
| log_request_headers  | bool | Optional | true    | Whether to log request headers, enabled by default                                                     |
| log_request_body     | bool | Optional | true    | Whether to log request body, enabled by default. Only logs POST, PUT, PATCH requests with supported Content-Type |
| log_response_headers | bool | Optional | true    | Whether to log response headers, enabled by default                                                  |
| log_response_body    | bool | Optional | true    | Whether to log response body, enabled by default. Only logs supported Content-Type without content-encoding compression |

## Supported Content Types

The plugin only logs request/response bodies with the following Content-Type:

- `application/x-www-form-urlencoded`
- `application/json`
- `text/plain`

## Important Notes

1. Request/response body content exceeding 1KB will be truncated and marked as `<truncated>`
2. Response bodies with `content-encoding` header will not be logged (to avoid logging compressed content)
3. Newline characters will be escaped as `\n` to ensure log readability
4. Log output contains complete request and response information, please protect sensitive data
5. This plugin is primarily for development debugging and API monitoring scenarios

## Configuration Examples

### Log all request and response information

```yaml
log_request_headers: true
log_request_body: true
log_response_headers: true
log_response_body: true
```

### Log only request and response headers

```yaml
log_request_headers: true
log_request_body: false
log_response_headers: true
log_response_body: false
```

### Log only response information

```yaml
log_request_headers: false
log_request_body: false
log_response_headers: true
log_response_body: true
```

## Log Output Example

```
[info] request Headers: [:method=POST, :path=/api/test, content-type=application/json, user-agent=curl/7.68.0]
[info] request Body: [{"name": "test", "message": "hello\nworld"}]
[info] response Headers: [:status=200, content-type=application/json]
[info] response Body: [{"result": "success"}]
```
