---
title: Key-based Local Rate Limiting
keywords: [higress,key rate limit]
description: Configuration reference for Key local rate limiting plugin
---

## Functional Description
The `key-rate-limit` plugin implements rate limiting based on specific key values, which can originate from URL parameters or HTTP request headers.

## Running Properties
Plugin execution phase: `default phase`
Plugin execution priority: `10`

## Configuration Fields

| Name            | Data Type       | Required                                                      | Default Value | Description                                                                            |
|-----------------|-----------------|---------------------------------------------------------------|---------------|----------------------------------------------------------------------------------------|
| limit_by_header | string          | Optional, choose one from `limit_by_header`, `limit_by_param` | -             | Configuration for the source of the rate limiting key value (HTTP request header name) |
| limit_by_param  | string          | Optional, choose one from `limit_by_header`, `limit_by_param` | -             | Configuration for the source of the rate limiting key value (URL parameter name)       |
| limit_keys      | array of object | Required                                                      | -             | Configuration for the rate limiting frequency based on matched key values              |

Explanation of each configuration field in `limit_keys`

| Name             | Data Type | Required                                                                                            | Default Value | Description                           |
|------------------|-----------|-----------------------------------------------------------------------------------------------------|---------------|---------------------------------------|
| key              | string    | Required                                                                                            | -             | Matched key value                     |
| query_per_second | number    | Optional, choose one from `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day` | -             | Allowed number of requests per second |
| query_per_minute | number    | Optional, choose one from `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day` | -             | Allowed number of requests per minute |
| query_per_hour   | number    | Optional, choose one from `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day` | -             | Allowed number of requests per hour   |
| query_per_day    | number    | Optional, choose one from `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day` | -             | Allowed number of requests per day    |

## Configuration Examples
### Identify request parameter apikey for differentiated rate limiting
```yaml
limit_by_param: apikey
limit_keys:
- key: 9a342114-ba8a-11ec-b1bf-00163e1250b5
  query_per_second: 10
- key: a6a6d7f2-ba8a-11ec-bec2-00163e1250b5
  query_per_minute: 100
```

### Identify request header x-ca-key for differentiated rate limiting
```yaml
limit_by_header: x-ca-key
limit_keys:
- key: 102234
  query_per_second: 10
- key: 308239
  query_per_hour: 10
```
