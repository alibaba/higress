## 简介
---
title: AI 缓存
keywords: [higress,ai cache]
description: AI 缓存插件配置参考
---

**Note**

> 需要数据面的proxy wasm版本大于等于0.2.100
> 编译时，需要带上版本的tag，例如：`tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags="custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100" ./`
>

## 功能说明

LLM 结果缓存插件，默认配置方式可以直接用于 openai 协议的结果缓存，同时支持流式和非流式响应的缓存。

## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`10`

## 配置说明
配置分为 3 个部分：向量数据库（vector）；文本向量化接口（embedding）；缓存数据库（cache），同时也提供了细粒度的 LLM 请求/响应提取参数配置等。

## 配置说明

本插件需要配置向量数据库服务（vector），根据所选的向量数据库服务类型，您可以决定是否配置文本向量化接口（embedding）以将问题转换为向量。最后，根据所选的缓存服务类型，您可以决定是否配置缓存服务（cache）以存储LLM的响应结果。

| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| vector.type | string | required | "" | 向量存储服务提供者类型，例如 DashVector |
| embedding.type | string | optional | "" | 请求文本向量化服务类型，例如 DashScope |
| cache.type | string | optional | "" | 缓存服务类型，例如 redis |
| cacheKeyStrategy | string | optional | "lastQuestion" | 决定如何根据历史问题生成缓存键的策略。可选值: "lastQuestion" (使用最后一个问题), "allQuestions" (拼接所有问题) 或 "disable" (禁用缓存) |
| enableSemanticCache | bool | optional | true | 是否启用语义化缓存, 若不启用，则使用逐字匹配的方式来查找缓存，此时需要配置cache服务 |

以下是vector、embedding、cache的具体配置说明，注意若不配置embedding或cache服务，则可忽略以下相应配置中的 `required` 字段。

## 向量数据库服务（vector）
| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| vector.type | string | required | "" | 向量存储服务提供者类型，例如 DashVector |
| vector.serviceName | string | required | "" | 向量存储服务名称 |
| vector.serviceDomain | string | required | "" | 向量存储服务域名 |
| vector.servicePort | int64 | optional | 443 | 向量存储服务端口 |
| vector.apiKey | string | optional | ""  | 向量存储服务 API Key |
| vector.topK | int | optional | 1 | 返回TopK结果，默认为 1 |
| vector.timeout | uint32 | optional | 10000 | 请求向量存储服务的超时时间，单位为毫秒。默认值是10000，即10秒 |
| vector.collectionID | string | optional | "" |  DashVector 向量存储服务 Collection ID |


## 文本向量化服务（embedding）
| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| embedding.type | string | required | "" | 请求文本向量化服务类型，例如 DashScope |
| embedding.serviceName | string | required | "" | 请求文本向量化服务名称 |
| embedding.serviceDomain | string | required | "" | 请求文本向量化服务域名 |
| embedding.servicePort | int64 | optional | 443 | 请求文本向量化服务端口 |
| embedding.apiKey | string | optional | ""  | 请求文本向量化服务的 API Key |
| embedding.timeout | uint32 | optional | 10000 | 请求文本向量化服务的超时时间，单位为毫秒。默认值是10000，即10秒 |
| embedding.model | string | optional | "" | 请求文本向量化服务的模型名称 |


## 缓存服务（cache）
| cache.type | string | required | "" | 缓存服务类型，例如 redis |
| --- | --- | --- | --- | --- |
| cache.serviceName | string | required | "" | 缓存服务名称 |
| cache.serviceDomain | string | required | "" | 缓存服务域名 |
| cache.servicePort | int64 | optional | 6379 | 缓存服务端口 |
| cache.username | string | optional | ""  | 缓存服务用户名 |
| cache.password | string | optional | "" | 缓存服务密码 |
| cache.timeout | uint32 | optional | 10000 | 缓存服务的超时时间，单位为毫秒。默认值是10000，即10秒 |
| cache.cacheTTL | int | optional | 0 | 缓存过期时间，单位为秒。默认值是 0，即 永不过期|
| cacheKeyPrefix | string | optional | "higressAiCache:" | 缓存 Key 的前缀，默认值为 "higressAiCache:" |


## 其他配置
| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| cacheKeyFrom | string | optional | "messages.@reverse.0.content" | 从请求 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |
| cacheValueFrom | string | optional | "choices.0.message.content" | 从响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |
| cacheStreamValueFrom | string | optional | "choices.0.delta.content" | 从流式响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |
| cacheToolCallsFrom | string | optional | "choices.0.delta.content.tool_calls" | 从请求 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |
| responseTemplate | string | optional | `{"id":"ai-cache.hit","choices":[{"index":0,"message":{"role":"assistant","content":%s},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}` | 返回 HTTP 响应的模版，用 %s 标记需要被 cache value 替换的部分 |
| streamResponseTemplate | string | optional | `data:{"id":"ai-cache.hit","choices":[{"index":0,"delta":{"role":"assistant","content":%s},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}\n\ndata:[DONE]\n\n` | 返回流式 HTTP 响应的模版，用 %s 标记需要被 cache value 替换的部分 |


## 配置示例
### 基础配置
```yaml
embedding:
  type: dashscope
  serviceName: [Your Service Name]
  apiKey: [Your Key]

vector:
  type: dashvector
  serviceName: [Your Service Name]
  collectionID: [Your Collection ID]
  serviceDomain: [Your domain]
  apiKey: [Your key]

cache:
  type: redis
  serviceName: [Your Service Name]
  servicePort: 6379
  timeout: 100

```

## 进阶用法
当前默认的缓存 key 是基于 GJSON PATH 的表达式：`messages.@reverse.0.content` 提取，含义是把 messages 数组反转后取第一项的 content；

GJSON PATH 支持条件判断语法，例如希望取最后一个 role 为 user 的 content 作为 key，可以写成： `messages.@reverse.#(role=="user").content`；

如果希望将所有 role 为 user 的 content 拼成一个数组作为 key，可以写成：`messages.@reverse.#(role=="user")#.content`；

还可以支持管道语法，例如希望取到数第二个 role 为 user 的 content 作为 key，可以写成：`messages.@reverse.#(role=="user")#.content|1`。

更多用法可以参考[官方文档](https://github.com/tidwall/gjson/blob/master/SYNTAX.md)，可以使用 [GJSON Playground](https://gjson.dev/) 进行语法测试。

