---
title: Response Cache
keywords: [higress,response cache]
description: Response Cache Plugin Configuration Reference
---
## Function Description
Response caching plugin supports extracting keys from request headers/request bodies and caching values extracted from response bodies. On subsequent requests, if the request headers/request bodies contain the same key, it directly returns the cached value without forwarding the request to the backend service.

**Hint**

When carrying the request header `x-higress-skip-response-cache: on`, the current request will not use content from the cache but will be directly forwarded to the backend service. Additionally, the response content from this request will not be cached.

## Runtime Properties
Plugin Execution Phase: `Authentication Phase`
Plugin Execution Priority: `10`

## Configuration Description

### Cache Service (cache)
| Property | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| cache.type | string | required | "" | Cache service type, e.g., redis |
| cache.serviceName | string | required | "" | Cache service name |
| cache.serviceHost | string | required | "" | Cache service domain |
| cache.servicePort | int64 | optional | 6379 | Cache service port |
| cache.username | string | optional | "" | Cache service username |
| cache.password | string | optional | "" | Cache service password |
| cache.timeout | uint32 | optional | 10000 | Timeout for cache service in milliseconds. Default is 10000, i.e., 10 seconds |
| cache.cacheTTL | int | optional | 0 | Cache expiration time in seconds. Default is 0, meaning never expires |
| cacheKeyPrefix | string | optional | "higress-response-cache:" | Prefix for cache keys, default is "higress-response-cache:" |                 |

### Other Configurations
| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| cacheResponseCode | array of number | optional | 200 | Indicates the list of response status codes that support caching; the default is 200.|
| cacheKeyFromHeader | string | required | "" | Extracts a fixed field's value from headers as the cache key; **only one of cacheKeyFromHeader and cacheKeyFromBody can be configured when both are non-empty**|
| cacheKeyFromBody | string | required | "" | If empty, extracts all body as the cache key; otherwise, extracts a string from the request body based on [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) |
| cacheValueFromBodyType | string | optional | "application/json" | Indicates the type of cached body; the content-type returned on cache hit will be this value; default is JSON |
| cacheValueFromBody | string | optional | "" | If empty, caches all body; when cacheValueFromBodyType is JSON, supports extracting a string from the response body based on [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) |


The logic for concatenating the cache key is one of the following:

1. `cacheKeyPrefix` + content extracted from the field corresponding to `cacheKeyFromHeader` in the request header
2. `cacheKeyPrefix` + content extracted from the field corresponding to `cacheKeyFromBody` in the request body

In the case of hitting the cache plugin, there are three statuses in the returned response headers:

- `x-cache-status: hit` , indicating a cache hit and cached content is returned directly
- `x-cache-status: miss` , indicating a cache miss and backend response results are returned
- `x-cache-status: skip` , indicating skipping the cache check

## Configuration Example
### Basic Configuration
```yaml
cache:
  type: redis
  serviceName: my-redis.dns
  servicePort: 6379
  timeout: 2000
  
cacheKeyFromHeader: "x-http-cache-key"

cacheValueFromBodyType: "application/json"
cacheValueFromBody: "messages.@reverse.0.content"
```

Assumed Request

```bash
# Request
curl -H "x-http-cache-key: abcd" <url>

# Response
{"messages":[{"content":"1"}, {"content":"2"}, {"content":"3"}]}
```

In this case, the cache key would be `higress-response-cache:abcd`, and the cached value would be `3`.

For subsequent requests that hit the cache, the response Content-Type returned is `application/json`.

### Response Body as Cache Value
To cache all response bodies, configure as follows:

```yaml
cacheValueFromBodyType: "text/html"
cacheValueFromBody: ""
```
For subsequent requests that hit the cache, the response Content-Type returned is `text/html`.


### Request Body as Cache Key
To use the request body as the key, configure as follows:

```yaml

cacheKeyFromBody: ""
```

The configuration supports GJSON PATH syntax.


## Advanced Usage
When the body is `application/json`, GJSON PATH syntax is supported:

For example, the expression `messages.@reverse.0.content` means taking the content of the first item after reversing the messages array.

GJSON PATH also supports conditional syntax. For instance, to take the content of the last message where role is "user", you can write: `messages.@reverse.#(role=="user").content`.

To concatenate all contents where role is "user" into an array, you can write: `messages.@reverse.#(role=="user")#.content`.

Pipeline syntax is also supported. For example, to take the second content where role is "user", you can write: `messages.@reverse.#(role=="user")#.content|1`.

Refer to the [official documentation](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) for more usage examples, and test the syntax using the [GJSON Playground](https://gjson.dev/).

## Common Issues
If the error `error status returned by host: bad argument occurs`, check whether `serviceName` correctly includes the service type suffix (.dns, etc.).