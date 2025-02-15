---
title: AI Token Rate Limiting
keywords: [ AI Gateway, AI Token Rate Limiting ]
description: AI Token Rate Limiting Plugin Configuration Reference
---
## Function Description
The `ai-token-ratelimit` plugin implements token rate limiting based on specific key values. The key values can come from URL parameters, HTTP request headers, client IP addresses, consumer names, or key names in cookies.

## Runtime Attributes
Plugin execution phase: `default phase`  
Plugin execution priority: `600`

## Configuration Description
| Configuration Item      | Type              | Required | Default Value | Description                                                                   |
| ----------------------- | ----------------- | -------- | ------------- | ----------------------------------------------------------------------------- |
| rule_name               | string            | Yes      | -             | Name of the rate limiting rule, used to assemble the redis key based on the rule name + rate limiting type + rate limiting key name + actual value corresponding to the rate limiting key |
| rule_items              | array of object   | Yes      | -             | Rate limiting rule items. After matching the first rule_item, subsequent rules will be ignored based on the order in `rule_items` |
| rejected_code           | int               | No       | 429           | The HTTP status code returned when the request is rate limited               |
| rejected_msg            | string            | No       | Too many requests | The response body returned when the request is rate limited                 |
| redis                   | object            | Yes      | -             | Redis related configuration                                                   |

Field descriptions for each item in `rule_items`
| Configuration Item       | Type              | Required                    | Default Value | Description                                                    |
| ------------------------ | ----------------- | --------------------------- | ------------- | ------------------------------------------------------------ |
| limit_by_header          | string            | No, optionally select one in `limit_by_*` | -             | Configure the source HTTP header name for obtaining the rate limiting key value |
| limit_by_param           | string            | No, optionally select one in `limit_by_*` | -             | Configure the source URL parameter name for obtaining the rate limiting key value |
| limit_by_consumer        | string            | No, optionally select one in `limit_by_*` | -             | Rate limit by consumer name, no actual value needs to be added |
| limit_by_cookie          | string            | No, optionally select one in `limit_by_*` | -             | Configure the source key name in cookies for obtaining the rate limiting key value |
| limit_by_per_header      | string            | No, optionally select one in `limit_by_*` | -             | Match specific HTTP request headers according to rules and calculate rate limiting separately for each header. Configure the source HTTP header name for obtaining the rate limiting key value. Supports regular expressions or `*` when configuring `limit_keys` |
| limit_by_per_param       | string            | No, optionally select one in `limit_by_*` | -             | Match specific URL parameters according to rules and calculate rate limiting separately for each parameter. Configure the source URL parameter name for obtaining the rate limiting key value. Supports regular expressions or `*` when configuring `limit_keys` |
| limit_by_per_consumer    | string            | No, optionally select one in `limit_by_*` | -             | Match specific consumers according to rules and calculate rate limiting separately for each consumer. Rate limit by consumer name, no actual value needs to be added. Supports regular expressions or `*` when configuring `limit_keys` |
| limit_by_per_cookie      | string            | No, optionally select one in `limit_by_*` | -             | Match specific cookies according to rules and calculate rate limiting separately for each cookie. Configure the source key name in cookies for obtaining the rate limiting key value. Supports regular expressions or `*` when configuring `limit_keys` |
| limit_by_per_ip          | string            | No, optionally select one in `limit_by_*` | -             | Match specific IPs according to rules and calculate rate limiting separately for each IP. Configure the source IP parameter name for obtaining the rate limiting key value from request headers, `from-header-<header name>`, such as `from-header-x-forwarded-for`. Directly get the remote socket IP by configuring `from-remote-addr` |
| limit_keys               | array of object    | Yes                         | -             | Configure the number of rate limit requests after matching keys                                   |

Field descriptions for each item in `limit_keys`
| Configuration Item      | Type              | Required                                    | Default Value | Description                                     |
| ----------------------- | ----------------- | ------------------------------------------- | ------------- | ----------------------------------------------- |
| key                     | string            | Yes                                         | -             | Matched key value. Types `limit_by_per_header`, `limit_by_per_param`, `limit_by_per_consumer`, `limit_by_per_cookie` support configuring regular expressions (beginning with regexp: followed by the regex) or `*` (representing all). Example regex: `regexp:^d.*` (all strings starting with d); `limit_by_per_ip` supports configuring IP addresses or IP segments |
| token_per_second        | int               | No, optionally select one in `token_per_second`, `token_per_minute`, `token_per_hour`, `token_per_day` | -             | Allowed number of token requests per second     |
| token_per_minute       | int               | No, optionally select one in `token_per_second`, `token_per_minute`, `token_per_hour`, `token_per_day` | -             | Allowed number of token requests per minute     |
| token_per_hour         | int               | No, optionally select one in `token_per_second`, `token_per_minute`, `token_per_hour`, `token_per_day` | -             | Allowed number of token requests per hour       |
| token_per_day          | int               | No, optionally select one in `token_per_second`, `token_per_minute`, `token_per_hour`, `token_per_day` | -             | Allowed number of token requests per day        |

Field descriptions for each item in `redis`
| Configuration Item      | Type              | Required | Default Value                                                     | Description                                     |
| ----------------------- | ----------------- | -------- | --------------------------------------------------------------- | ----------------------------------------------- |
| service_name            | string            | Required | -                                                               | Full FQDN name of the redis service, including service type, e.g., my-redis.dns, redis.my-ns.svc.cluster.local |
| service_port            | int               | No       | Default value for static addresses (static service) is 80; otherwise, it is 6379 | Input the service port of the redis service     |
| username                | string            | No       | -                                                               | Redis username                                  |
| password                | string            | No       | -                                                               | Redis password                                  |
| timeout                 | int               | No       | 1000                                                            | Redis connection timeout in milliseconds       |

## Configuration Examples
### Identify request parameter apikey for differentiated rate limiting
```yaml
rule_name: default_rule
rule_items:
  - limit_by_param: apikey
    limit_keys:
      - key: 9a342114-ba8a-11ec-b1bf-00163e1250b5
        token_per_minute: 10
      - key: a6a6d7f2-ba8a-11ec-bec2-00163e1250b5
        token_per_hour: 100
  - limit_by_per_param: apikey
    limit_keys:
      # Regular expression, matches all strings starting with a, each apikey corresponds to 10 qds
      - key: "regexp:^a.*"
        token_per_second: 10
      # Regular expression, matches all strings starting with b, each apikey corresponds to 100 qd
      - key: "regexp:^b.*"
        token_per_minute: 100
      # Fallback, matches all requests, each apikey corresponds to 1000 qdh
      - key: "*"
        token_per_hour: 1000
redis:
  service_name: redis.static
```
### Identify request header x-ca-key for differentiated rate limiting
```yaml
rule_name: default_rule
rule_items:
  - limit_by_header: x-ca-key
    limit_keys:
    	- key: 102234
        token_per_minute: 10
      - key: 308239
        token_per_hour: 10
  - limit_by_per_header: x-ca-key
    limit_keys:
      # Regular expression, matches all strings starting with a, each apikey corresponds to 10 qds
      - key: "regexp:^a.*"
        token_per_second: 10
      # Regular expression, matches all strings starting with b, each apikey corresponds to 100 qd
      - key: "regexp:^b.*"
        token_per_minute: 100
      # Fallback, matches all requests, each apikey corresponds to 1000 qdh
      - key: "*"
        token_per_hour: 1000
redis:
  service_name: redis.static
```
### Get the peer IP using the request header x-forwarded-for for differentiated rate limiting
```yaml
rule_name: default_rule
rule_items:
  - limit_by_per_ip: from-header-x-forwarded-for
    limit_keys:
      # Exact IP
      - key: 1.1.1.1
        token_per_day: 10
      # IP segment, matching IPs in this segment, each IP 100 qpd
      - key: 1.1.1.0/24
        token_per_day: 100
      # Fallback, i.e., default each IP 1000 qpd
      - key: 0.0.0.0/0
        token_per_day: 1000
redis:
  service_name: redis.static
```
### Identify consumer for differentiated rate limiting
```yaml
rule_name: default_rule
rule_items:
  - limit_by_consumer: ''
    limit_keys:
      - key: consumer1
        token_per_second: 10
      - key: consumer2
        token_per_hour: 100
  - limit_by_per_consumer: ''
    limit_keys:
      # Regular expression, matches all strings starting with a, each consumer corresponds to 10 qds
      - key: "regexp:^a.*"
        token_per_second: 10
      # Regular expression, matches all strings starting with b, each consumer corresponds to 100 qd
      - key: "regexp:^b.*"
        token_per_minute: 100
      # Fallback, matches all requests, each consumer corresponds to 1000 qdh
      - key: "*"
        token_per_hour: 1000
redis:
  service_name: redis.static
```
### Identify key-value pairs in cookies for differentiated rate limiting
```yaml
rule_name: default_rule
rule_items:
  - limit_by_cookie: key1
    limit_keys:
      - key: value1
        token_per_minute: 10
      - key: value2
        token_per_hour: 100
  - limit_by_per_cookie: key1
    limit_keys:
      # Regular expression, matches all strings starting with a, each value in cookie corresponds to 10 qds
      - key: "regexp:^a.*"
        token_per_second: 10
      # Regular expression, matches all strings starting with b, each value in cookie corresponds to 100 qd
      - key: "regexp:^b.*"
        token_per_minute: 100
      # Fallback, matches all requests, each value in cookie corresponds to 1000 qdh
      - key: "*"
        token_per_hour: 1000
rejected_code: 200
rejected_msg: '{"code":-1,"msg":"Too many requests"}'
redis:
  service_name: redis.static
```

## Example

The AI Token Rate Limiting Plugin relies on Redis to track the remaining available tokens, so the Redis service must be deployed first.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  labels:
    app: redis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis
        ports:
        - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  labels:
    app: redis
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
---
```

In this example, qwen is used as the AI service provider. Additionally, the AI Statistics Plugin must be configured, as the AI Token Rate Limiting Plugin depends on it to calculate the number of tokens consumed per request. The following configuration limits the total number of input and output tokens to 200 per minute.

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-proxy
  namespace: higress-system
spec:
  matchRules:
  - config:
      provider:
        type: qwen
        apiTokens:
        - "<YOUR_API_TOKEN>"
        modelMapping:
          'gpt-3': "qwen-turbo"
          'gpt-35-turbo': "qwen-plus"
          'gpt-4-turbo': "qwen-max"
          '*': "qwen-turbo"
    ingress:
    - qwen
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-proxy:1.0.0
  phase: UNSPECIFIED_PHASE
  priority: 100
---
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-token-ratelimit
  namespace: higress-system
spec:
  defaultConfig:
    rule_name: default_limit_by_param_apikey
    rule_items:
    - limit_by_param: apikey
      limit_keys:
      - key: 123456
        token_per_minute: 200
    redis:
      # By default, to reduce data plane pressure, the `global.onlyPushRouteCluster` parameter in Higress is set to true, meaning that Kubernetes Services are not automatically discovered.
      # If you need to use Kubernetes Service for service discovery, set `global.onlyPushRouteCluster` to false,
      # allowing you to directly set `service_name` to the Kubernetes Service without needing to create an McpBridge and an Ingress route for Redis.
      # service_name: redis.default.svc.cluster.local
      service_name: redis.dns
      service_port: 6379
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-token-ratelimit:1.0.0
  phase: UNSPECIFIED_PHASE
  priority: 600
```

Note that the `service_name` in the Redis configuration of the AI Token Rate Limiting Plugin is derived from the service source configured in McpBridge. Additionally, we need to configure the access address of the qnwen service in McpBridge.

```yaml
apiVersion: networking.higress.io/v1
kind: McpBridge
metadata:
  name: default
  namespace: higress-system
spec:
  registries:
  - domain: dashscope.aliyuncs.com
    name: qwen
    port: 443
    type: dns
  - domain: redis.default.svc.cluster.local # Kubernetes Service
    name: redis
    type: dns
    port: 6379
```

Create two routing rules separately.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    higress.io/backend-protocol: HTTPS
    higress.io/destination: qwen.dns
    higress.io/proxy-ssl-name: dashscope.aliyuncs.com
    higress.io/proxy-ssl-server-name: "on"
  labels:
    higress.io/resource-definer: higress
  name: qwen
  namespace: higress-system
spec:
  ingressClassName: higress
  rules:
  - host: qwen-test.com
    http:
      paths:
      - backend:
          resource:
            apiGroup: networking.higress.io
            kind: McpBridge
            name: default
        path: /
        pathType: Prefix
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    higress.io/destination: redis.dns
    higress.io/ignore-path-case: "false"
  labels:
    higress.io/resource-definer: higress
  name: redis
spec:
  ingressClassName: higress
  rules:
  - http:
      paths:
      - backend:
          resource:
            apiGroup: networking.higress.io
            kind: McpBridge
            name: default
        path: /
        pathType: Prefix
```

Forward the traffic of higress-gateway to the local, making it convenient for testing.

```bash
kubectl port-forward svc/higress-gateway -n higress-system 18000:80
```

The rate limiting effect is triggered as follows:

```bash
curl "http://localhost:18000/v1/chat/completions?apikey=123456" \
-H "Host: qwen-test.com" \
-H "Content-Type: application/json" \
-d '{
  "model": "gpt-3",
  "messages": [
    {
      "role": "user",
      "content": "Hello, who are you?"
    }
  ],
  "stream": false
}'
{"id":"88cfa80f-545d-93b4-8ff3-3f5245ca33ba","choices":[{"index":0,"message":{"role":"assistant","content":"I am Tongyi Qianwen, an AI assistant developed by Alibaba Cloud. I can answer various questions, provide information, and have conversations with users. How can I assist you?"},"finish_reason":"stop"}],"created":1719909825,"model":"qwen-turbo","object":"chat.completion","usage":{"prompt_tokens":13,"completion_tokens":33,"total_tokens":46}}
curl "http://qwen-test.com:18000/v1/chat/completions?apikey=123456" -H "Content-Type: application/json"  -d '{
  "model": "gpt-3",
  "messages": [
    {
      "role": "user",
      "content": "Hello, who are you?"
    }
  ],
  "stream": false
}'
Too many requests  # Rate limiting successful
```
