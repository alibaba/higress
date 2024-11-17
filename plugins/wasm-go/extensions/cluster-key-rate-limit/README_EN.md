---
title: Key-Based Cluster Rate Limiting
keywords: [higress, rate-limit]
description: Configuration reference for the Key-Based Cluster Rate Limiting plugin
---
## Function Description
The `cluster-key-rate-limit` plugin implements cluster rate limiting based on Redis, suitable for scenarios that require global consistent rate limiting across multiple Higress Gateway instances. 

The Key used for rate limiting can originate from URL parameters, HTTP request headers, client IP addresses, consumer names, or keys in cookies. 

## Execution Attributes
Plugin Execution Phase: `default phase`  
Plugin Execution Priority: `20` 

## Configuration Description
| Configuration Item        | Type          | Required | Default Value | Description                                                                               |
|---------------------------|---------------|----------|---------------|-------------------------------------------------------------------------------------------|
| rule_name                 | string        | Yes      | -             | The name of the rate limiting rule. The Redis key is constructed using rule name + rate limit type + limit key name + actual value of the limit key.         |
| rule_items                | array of object| Yes     | -             | Rate limiting rule items. The first matching `rule_item` based on the order under `rule_items` will trigger the rate limiting, and subsequent rules will be ignored.                 |
| show_limit_quota_header   | bool          | No       | false         | Whether to display `X-RateLimit-Limit` (total requests allowed) and `X-RateLimit-Remaining` (remaining requests that can be sent) in the response headers. |
| rejected_code             | int           | No       | 429           | HTTP status code returned when a request is rate limited.                                                          |
| rejected_msg              | string        | No       | Too many requests | Response body returned when a request is rate limited.                                                               |
| redis                     | object        | Yes      | -             | Redis related configuration.                                                                  |

Description of configuration fields for each item in `rule_items`.
| Configuration Item        | Type          | Required               | Default Value | Description                                                                                           |
|---------------------------|---------------|------------------------|---------------|-------------------------------------------------------------------------------------------------------|
| limit_by_header           | string        | No, one of `limit_by_*` | -             | The name of the HTTP request header from which to retrieve the rate limiting key value.               |
| limit_by_param            | string        | No, one of `limit_by_*` | -             | The name of the URL parameter from which to retrieve the rate limiting key value.                     |
| limit_by_consumer         | string        | No, one of `limit_by_*` | -             | Applies rate limiting based on consumer name without needing to add an actual value.                  |
| limit_by_cookie           | string        | No, one of `limit_by_*` | -             | The name of the key in the Cookie from which to retrieve the rate limiting key value.                |
| limit_by_per_header       | string        | No, one of `limit_by_*` | -             | Matches specific HTTP request headers according to the rules and calculates rate limits for each header. The `limit_keys` configuration supports regular expressions or `*`. |
| limit_by_per_param        | string        | No, one of `limit_by_*` | -             | Matches specific URL parameters according to the rules and calculates rate limits for each parameter. The `limit_keys` configuration supports regular expressions or `*`. |
| limit_by_per_consumer     | string        | No, one of `limit_by_*` | -             | Matches specific consumers according to the rules and calculates rate limits for each consumer. The `limit_keys` configuration supports regular expressions or `*`. |
| limit_by_per_cookie       | string        | No, one of `limit_by_*` | -             | Matches specific cookies according to the rules and calculates rate limits for each cookie. The `limit_keys` configuration supports regular expressions or `*`. |
| limit_by_per_ip           | string        | No, one of `limit_by_*` | -             | Matches specific IPs according to the rules and calculates rate limits for each IP. Retrieve via IP parameter name from request headers, defined as `from-header-{header name}`, e.g., `from-header-x-forwarded-for`. To get the remote socket IP directly, use `from-remote-addr`. |
| limit_keys                | array of object | Yes                    | -             | Configures the limit counts after matching key values.                                               |

Description of configuration fields for each item in `limit_keys`.
| Configuration Item        | Type          | Required                                                         | Default Value | Description                                                        |
|---------------------------|---------------|------------------------------------------------------------------|---------------|--------------------------------------------------------------------|
| key                       | string        | Yes                                                              | -             | Matched key value; types `limit_by_per_header`, `limit_by_per_param`, `limit_by_per_consumer`, `limit_by_per_cookie` support regular expression configurations (starting with regexp: followed by a regular expression) or `*` (representing all), e.g., `regexp:^d.*` (all strings starting with d); `limit_by_per_ip` supports configuring IP addresses or IP segments. |
| query_per_second          | int           | No, one of `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day` is optional. | -             | Allowed number of requests per second.                           |
| query_per_minute          | int           | No, one of `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day` is optional. | -             | Allowed number of requests per minute.                           |
| query_per_hour            | int           | No, one of `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day` is optional. | -             | Allowed number of requests per hour.                             |
| query_per_day             | int           | No, one of `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day` is optional. | -             | Allowed number of requests per day.                              |

Description of configuration fields for each item in `redis`.
| Configuration Item        | Type          | Required | Default Value                                               | Description                                                               |
|---------------------------|---------------|----------|------------------------------------------------------------|---------------------------------------------------------------------------|
| service_name              | string        | Required | -                                                          | Full FQDN name of the Redis service, including service type, e.g., my-redis.dns, redis.my-ns.svc.cluster.local. |
| service_port              | int           | No       | 80 for static services; otherwise 6379                     | Service port for the Redis service.                                      |
| username                  | string        | No       | -                                                          | Redis username.                                                          |
| password                  | string        | No       | -                                                          | Redis password.                                                          |
| timeout                   | int           | No       | 1000                                                       | Redis connection timeout in milliseconds.                               |

## Configuration Examples

### Distinguish rate limiting based on the request parameter apikey
```yaml
rule_name: default_rule
rule_items:
- limit_by_param: apikey
  limit_keys:
  - key: 9a342114-ba8a-11ec-b1bf-00163e1250b5
    query_per_minute: 10
  - key: a6a6d7f2-ba8a-11ec-bec2-00163e1250b5
    query_per_hour: 100
- limit_by_per_param: apikey
  limit_keys:
  # Regular expression, matches all strings starting with a, each apikey corresponds to 10qds.
  - key: "regexp:^a.*"
    query_per_second: 10
  # Regular expression, matches all strings starting with b, each apikey corresponds to 100qd.
  - key: "regexp:^b.*"
    query_per_minute: 100
  # As a fallback, matches all requests, each apikey corresponds to 1000qdh.
  - key: "*"
    query_per_hour: 1000
redis:
  service_name: redis.static
show_limit_quota_header: true
```

### Distinguish rate limiting based on the header x-ca-key
```yaml
rule_name: default_rule
rule_items:
- limit_by_header: x-ca-key
  limit_keys:
  - key: 102234
    query_per_minute: 10
  - key: 308239
    query_per_hour: 10
- limit_by_per_header: x-ca-key
  limit_keys:
  # Regular expression, matches all strings starting with a, each apikey corresponds to 10qds.
  - key: "regexp:^a.*"
    query_per_second: 10
  # Regular expression, matches all strings starting with b, each apikey corresponds to 100qd.
  - key: "regexp:^b.*"
    query_per_minute: 100
  # As a fallback, matches all requests, each apikey corresponds to 1000qdh.
  - key: "*"
    query_per_hour: 1000
redis:
  service_name: redis.static
show_limit_quota_header: true
```

### Distinguish rate limiting based on the client IP from the request header x-forwarded-for
```yaml
rule_name: default_rule
rule_items:
- limit_by_per_ip: from-header-x-forwarded-for
  limit_keys:
  # Exact IP
  - key: 1.1.1.1
    query_per_day: 10
  # IP segment, for IPs matching this segment, each IP corresponds to 100qpd.
  - key: 1.1.1.0/24
    query_per_day: 100
  # As a fallback, defaults to 1000 qpd for each IP.
  - key: 0.0.0.0/0
    query_per_day: 1000
redis:
  service_name: redis.static
show_limit_quota_header: true
```

### Distinguish rate limiting based on consumers
```yaml
rule_name: default_rule
rule_items:
- limit_by_consumer: ''
  limit_keys:
  - key: consumer1
    query_per_second: 10
  - key: consumer2
    query_per_hour: 100
- limit_by_per_consumer: ''
  limit_keys:
  # Regular expression, matches all strings starting with a, each consumer corresponds to 10qds.
  - key: "regexp:^a.*"
    query_per_second: 10
  # Regular expression, matches all strings starting with b, each consumer corresponds to 100qd.
  - key: "regexp:^b.*"
    query_per_minute: 100
  # As a fallback, matches all requests, each consumer corresponds to 1000qdh.
  - key: "*"
    query_per_hour: 1000
redis:
  service_name: redis.static
show_limit_quota_header: true
```

### Distinguish rate limiting based on key-value pairs in cookies
```yaml
rule_name: default_rule
rule_items:
  - limit_by_cookie: key1
    limit_keys:
      - key: value1
        query_per_minute: 10
      - key: value2
        query_per_hour: 100
  - limit_by_per_cookie: key1
    limit_keys:
      # Regular expression, matches all strings starting with a, each cookie's value corresponds to 10qds.
      - key: "regexp:^a.*"
        query_per_second: 10
      # Regular expression, matches all strings starting with b, each cookie's value corresponds to 100qd.
      - key: "regexp:^b.*"
        query_per_minute: 100
      # As a fallback, matches all requests, each cookie's value corresponds to 1000qdh.
      - key: "*"
        query_per_hour: 1000
rejected_code: 200
rejected_msg: '{"code":-1,"msg":"Too many requests"}'
redis:
  service_name: redis.static
show_limit_quota_header: true
```
