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

**提示**

携带请求头`x-higress-skip-ai-cache: on`时，当前请求将不会使用缓存中的内容，而是直接转发给后端服务，同时也不会缓存该请求返回响应的内容


## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`10`

## 配置说明
配置分为 3 个部分：向量数据库（vector）；文本向量化接口（embedding）；缓存数据库（cache），同时也提供了细粒度的 LLM 请求/响应提取参数配置等。

## 配置说明

本插件同时支持基于向量数据库的语义化缓存和基于字符串匹配的缓存方法，如果同时配置了向量数据库和缓存数据库，优先使用向量数据库。

*Note*: 向量数据库(vector) 和 缓存数据库(cache) 不能同时为空，否则本插件无法提供缓存服务。

| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| vector | string | optional | "" | 向量存储服务提供者类型，例如 dashvector |
| embedding | string | optional | "" | 请求文本向量化服务类型，例如 dashscope |
| cache | string | optional | "" | 缓存服务类型，例如 redis |
| cacheKeyStrategy | string | optional | "lastQuestion" | 决定如何根据历史问题生成缓存键的策略。可选值: "lastQuestion" (使用最后一个问题), "allQuestions" (拼接所有问题) 或 "disabled" (禁用缓存) |
| enableSemanticCache | bool | optional | true | 是否启用语义化缓存, 若不启用，则使用字符串匹配的方式来查找缓存，此时需要配置cache服务 |

根据是否需要启用语义缓存，可以只配置组件的组合为:
1. `cache`: 仅启用字符串匹配缓存
3. `vector (+ embedding)`: 启用语义化缓存, 其中若 `vector` 未提供字符串表征服务，则需要自行配置 `embedding` 服务
2. `vector (+ embedding) + cache`: 启用语义化缓存并用缓存服务存储LLM响应以加速

注意若不配置相关组件，则可以忽略相应组件的`required`字段。


## 向量数据库服务（vector）
| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| vector.type | string | required | "" | 向量存储服务提供者类型，例如 dashvector |
| vector.serviceName | string | required | "" | 向量存储服务名称 |
| vector.serviceHost | string | required | "" | 向量存储服务域名 |
| vector.servicePort | int64 | optional | 443 | 向量存储服务端口 |
| vector.apiKey | string | optional | ""  | 向量存储服务 API Key |
| vector.topK | int | optional | 1 | 返回TopK结果，默认为 1 |
| vector.timeout | uint32 | optional | 10000 | 请求向量存储服务的超时时间，单位为毫秒。默认值是10000，即10秒 |
| vector.collectionID | string | optional | "" |  dashvector 向量存储服务 Collection ID |
| vector.threshold | float64 | optional | 1000 | 向量相似度度量阈值 |
| vector.thresholdRelation | string | optional | lt | 相似度度量方式有 `Cosine`, `DotProduct`, `Euclidean` 等，前两者值越大相似度越高，后者值越小相似度越高。对于 `Cosine` 和 `DotProduct` 选择 `gt`，对于 `Euclidean` 则选择 `lt`。默认为 `lt`，所有条件包括 `lt` (less than，小于)、`lte` (less than or equal to，小等于)、`gt` (greater than，大于)、`gte` (greater than or equal to，大等于) |

## 文本向量化服务（embedding）
| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| embedding.type | string | required | "" | 请求文本向量化服务类型，例如 dashscope |
| embedding.serviceName | string | required | "" | 请求文本向量化服务名称 |
| embedding.serviceHost | string | optional | "" | 请求文本向量化服务域名 |
| embedding.servicePort | int64 | optional | 443 | 请求文本向量化服务端口 |
| embedding.apiKey | string | optional | ""  | 请求文本向量化服务的 API Key |
| embedding.timeout | uint32 | optional | 10000 | 请求文本向量化服务的超时时间，单位为毫秒。默认值是10000，即10秒 |
| embedding.model | string | optional | "" | 请求文本向量化服务的模型名称 |


## 缓存服务（cache）
| cache.type | string | required | "" | 缓存服务类型，例如 redis |
| --- | --- | --- | --- | --- |
| cache.serviceName | string | required | "" | 缓存服务名称 |
| cache.serviceHost | string | required | "" | 缓存服务域名 |
| cache.servicePort | int64 | optional | 6379 | 缓存服务端口 |
| cache.username | string | optional | ""  | 缓存服务用户名 |
| cache.password | string | optional | "" | 缓存服务密码 |
| cache.timeout | uint32 | optional | 10000 | 缓存服务的超时时间，单位为毫秒。默认值是10000，即10秒 |
| cache.cacheTTL | int | optional | 0 | 缓存过期时间，单位为秒。默认值是 0，即 永不过期|
| cacheKeyPrefix | string | optional | "higress-ai-cache:" | 缓存 Key 的前缀，默认值为 "higress-ai-cache:" |


## 其他配置
| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| cacheKeyFrom | string | optional | "messages.@reverse.0.content" | 从请求 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |
| cacheValueFrom | string | optional | "choices.0.message.content" | 从响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |
| cacheStreamValueFrom | string | optional | "choices.0.delta.content" | 从流式响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |
| cacheToolCallsFrom | string | optional | "choices.0.delta.content.tool_calls" | 从请求 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |
| responseTemplate | string | optional | `{"id":"ai-cache.hit","choices":[{"index":0,"message":{"role":"assistant","content":%s},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}` | 返回 HTTP 响应的模版，用 %s 标记需要被 cache value 替换的部分 |
| streamResponseTemplate | string | optional | `data:{"id":"ai-cache.hit","choices":[{"index":0,"delta":{"role":"assistant","content":%s},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}\n\ndata:[DONE]\n\n` | 返回流式 HTTP 响应的模版，用 %s 标记需要被 cache value 替换的部分 |

# 向量数据库提供商特有配置
## Chroma
Chroma 所对应的 `vector.type` 为 `chroma`。它并无特有的配置字段。需要提前创建 Collection。

## DashVector
DashVector 所对应的 `vector.type` 为 `dashvector`。它并无特有的配置字段。需要提前创建 Collection。

## ElasticSearch
ElasticSearch 所对应的 `vector.type` 为 `elasticsearch`。需要提前创建 Index 并填入在 `vector.collectionID` 中。当前依赖于 [KNN](https://www.elastic.co/guide/en/elasticsearch/reference/current/knn-search.html) 方法，请保证 ES 版本支持 `KNN`，当前已在 `8.16` 版本测试。
它特有的配置字段如下：
| 名称              | 数据类型 | 填写要求 | 默认值 | 描述                                                                          |
|-------------------|----------|----------|--------|-------------------------------------------------------------------------------|
| `vector.esUsername` | string   | 非必填   | -      | ElasticSearch 用户名 |
| `vector.esPassword` | string | 非必填 | - | ElasticSearch 密码 |

`vector.esUsername` 和 `vector.esPassword` 用于 Basic 认证。同时也支持 Api Key 认证，当填写了 `vector.apiKey` 时，则启用 Api Key 认证，如果使用 SaaS 版本需要填写 `encoded` 的值。

## Milvus
Milvus 所对应的 `vector.type` 为 `milvus`。它并无特有的配置字段。需要提前创建 Collection。

## Pinecone
Pinecone 所对应的 `vector.type` 为 `pinecone`。它并无特有的配置字段。需要提前创建 Index，并填写 Index 访问域名至 `serviceHost`。
Pinecone 中的 `Namespace` 参数通过插件的 `vector.collectionID` 进行配置。

## Qdrant
Qdrant 所对应的 `vector.type` 为 `qdrant`。它并无特有的配置字段。需要提前创建 Collection。

## Weaviate
Weaviate 所对应的 `vector.type` 为 `weaviate`。它并无特有的配置字段。
需要提前创建 Collection。需要注意的是 Weaviate 会设置首字母自动大写，在填写配置 `collectionID` 的时候需要将首字母设置为大写。
如果使用 SaaS 需要填写 `serviceHost` 参数。

## 配置示例
### 基础配置
```yaml
embedding:
  type: dashscope
  serviceName: my_dashscope.dns
  apiKey: [Your Key]

vector:
  type: dashvector
  serviceName: my_dashvector.dns
  collectionID: [Your Collection ID]
  serviceDomain: [Your domain]
  apiKey: [Your key]

cache:
  type: redis
  serviceName: my_redis.dns
  servicePort: 6379
  timeout: 100

```

旧版本配置兼容
```yaml
redis:
  serviceName: my_redis.dns
  servicePort: 6379
  timeout: 100
```

## 进阶用法
当前默认的缓存 key 是基于 GJSON PATH 的表达式：`messages.@reverse.0.content` 提取，含义是把 messages 数组反转后取第一项的 content；

GJSON PATH 支持条件判断语法，例如希望取最后一个 role 为 user 的 content 作为 key，可以写成： `messages.@reverse.#(role=="user").content`；

如果希望将所有 role 为 user 的 content 拼成一个数组作为 key，可以写成：`messages.@reverse.#(role=="user")#.content`；

还可以支持管道语法，例如希望取到数第二个 role 为 user 的 content 作为 key，可以写成：`messages.@reverse.#(role=="user")#.content|1`。

更多用法可以参考[官方文档](https://github.com/tidwall/gjson/blob/master/SYNTAX.md)，可以使用 [GJSON Playground](https://gjson.dev/) 进行语法测试。

## 常见问题

1. 如果返回的错误为 `error status returned by host: bad argument`，请检查`serviceName`是否正确包含了服务的类型后缀(.dns等)。
