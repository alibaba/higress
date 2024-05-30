## 简介

**Note**

> 需要数据面的proxy wasm版本大于等于0.2.100

> 编译时，需要带上版本的tag，例如：`tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags="custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100" ./`

LLM 结果缓存插件，默认配置方式可以直接用于 openai 协议的结果缓存，同时支持流式和非流式响应的缓存。

## 配置说明

| Name                              | Type     | Requirement | Default                                                                                                                                                                                                                                                 | Description                                                                                                |
| --------                          | -------- | --------    | --------                                                                                                                                                                                                                                                | --------                                                                                                   |
| cacheKeyFrom.requestBody          | string   | optional    | "messages.@reverse.0.content"                                                                                                                                                                                                                           | 从请求 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串     |
| cacheValueFrom.responseBody       | string   | optional    | "choices.0.message.content"                                                                                                                                                                                                                             | 从响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串     |
| cacheStreamValueFrom.responseBody | string   | optional    | "choices.0.delta.content"                                                                                                                                                                                                                               | 从流式响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |
| cacheKeyPrefix                    | string   | optional    | "higress-ai-cache:"                                                                                                                                                                                                                                     | Redis缓存Key的前缀                                                                                         |
| cacheTTL                          | integer  | optional    | 0                                                                                                                                                                                                                                                       | 缓存的过期时间，单位是秒，默认值为0，即永不过期                                                            |
| redis.serviceName                 | string   | requried    | -                                                                                                                                                                                                                                                       | redis 服务名称，带服务类型的完整 FQDN 名称，例如 my-redis.dns、redis.my-ns.svc.cluster.local               |
| redis.servicePort                 | integer  | optional    | 6379                                                                                                                                                                                                                                                    | redis 服务端口                                                                                             |
| redis.timeout                     | integer  | optional    | 1000                                                                                                                                                                                                                                                    | 请求 redis 的超时时间，单位为毫秒                                                                          |
| redis.username                    | string   | optional    | -                                                                                                                                                                                                                                                       | 登陆 redis 的用户名                                                                                        |
| redis.password                    | string   | optional    | -                                                                                                                                                                                                                                                       | 登陆 redis 的密码                                                                                          |
| returnResponseTemplate            | string   | optional    | `{"id":"from-cache","choices":[%s],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`                                                                                                     | 返回 HTTP 响应的模版，用 %s 标记需要被 cache value 替换的部分                                              |
| returnStreamResponseTemplate      | string   | optional    | `data:{"id":"from-cache","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}\n\ndata:[DONE]\n\n` | 返回流式 HTTP 响应的模版，用 %s 标记需要被 cache value 替换的部分                                          |

## 配置示例

```yaml
redis:
  serviceName: my-redis.dns
  timeout: 2000
```
