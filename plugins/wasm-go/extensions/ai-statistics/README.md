---
title: AI可观测
keywords: [higress, AI, observability]
description: AI可观测配置参考
---

## 介绍

提供 AI 可观测基础能力，包括 metric, log, trace，其后需接 ai-proxy 插件，如果不接 ai-proxy 插件的话，则需要用户进行相应配置才可生效。

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`200`

## 配置说明

插件默认请求符合 openai 协议格式，并提供了以下基础可观测值，用户无需特殊配置：

- metric：提供了输入 token、输出 token、首个 token 的 rt（流式请求）、请求总 rt 等指标，支持在网关、路由、服务、模型四个维度上进行观测
- log：提供了 input_token, output_token, model, llm_service_duration, llm_first_token_duration 等字段

用户还可以通过配置的方式对可观测的值进行扩展：

| 名称             | 数据类型  | 填写要求 | 默认值 | 描述                     |
|----------------|-------|------|-----|------------------------|
| `attributes` | []Attribute | 非必填  | -   | 用户希望记录在log/span中的信息 |
| `disable_openai_usage` | bool | 非必填  | false   | 非openai兼容协议时，model、token的支持非标，配置为true时可以避免报错 |
| `value_length_limit` | int | 非必填  | 4000   | 记录的单个value的长度限制 |
| `enable_path_suffixes` | []string    | 非必填   | []     | 只对这些特定路径后缀的请求生效，可以配置为 "\*" 以匹配所有路径（通配符检查会优先进行以提高性能）。如果为空数组，则对所有路径生效 |
| `enable_content_types` | []string    | 非必填   | []     | 只对这些内容类型的响应进行缓冲处理。如果为空数组，则对所有内容类型生效                                                           |
| `session_id_header` | string | 非必填  | -   | 指定读取 session ID 的 header 名称。如果不配置，将按以下优先级自动查找：`x-openclaw-session-key`、`x-clawdbot-session-key`、`x-moltbot-session-key`、`x-agent-session`。session ID 可用于追踪多轮 Agent 对话 |

Attribute 配置说明:

| 名称                    | 数据类型 | 填写要求 | 默认值 | 描述                                                                                                                                        |
| ----------------------- | -------- | -------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `key`                   | string   | 必填     | -      | attribute 名称                                                                                                                              |
| `value_source`          | string   | 必填     | -      | attribute 取值来源，可选值为 `fixed_value`, `request_header`, `request_body`, `response_header`, `response_body`, `response_streaming_body` |
| `value`                 | string   | 必填     | -      | attribute 取值 key value/path                                                                                                               |
| `default_value`         | string   | 非必填   | -      | attribute 默认值                                                                                                                            |
| `rule`                  | string   | 非必填   | -      | 从流式响应中提取 attribute 的规则，可选值为 `first`, `replace`, `append`                                                                    |
| `apply_to_log`          | bool     | 非必填   | false  | 是否将提取的信息记录在日志中                                                                                                                |
| `apply_to_span`         | bool     | 非必填   | false  | 是否将提取的信息记录在链路追踪 span 中                                                                                                      |
| `trace_span_key`        | string   | 非必填   | -      | 链路追踪 attribute key，默认会使用`key`的设置                                                                                               |
| `as_separate_log_field` | bool     | 非必填   | false  | 记录日志时是否作为单独的字段，日志字段名使用`key`的设置                                                                                     |

`value_source` 的各种取值含义如下：

- `fixed_value`：固定值
- `request_header` ： attribute 值通过 http 请求头获取，value 配置为 header key
- `request_body` ：attribute 值通过请求 body 获取，value 配置格式为 gjson 的 jsonpath
- `response_header` ：attribute 值通过 http 响应头获取，value 配置为 header key
- `response_body` ：attribute 值通过响应 body 获取，value 配置格式为 gjson 的 jsonpath
- `response_streaming_body` ：attribute 值通过流式响应 body 获取，value 配置格式为 gjson 的 jsonpath

当 `value_source` 为 `response_streaming_body` 时，应当配置 `rule`，用于指定如何从流式 body 中获取指定值，取值含义如下：

- `first`：多个 chunk 中取第一个有效 chunk 的值
- `replace`：多个 chunk 中取最后一个有效 chunk 的值
- `append`：拼接多个有效 chunk 中的值，可用于获取回答内容

## 配置示例

如果希望在网关访问日志中记录 ai-statistic 相关的统计值，需要修改 log_format，在原 log_format 基础上添加一个新字段，示例如下：

```yaml
'{"ai_log":"%FILTER_STATE(wasm.ai_log:PLAIN)%"}'
```

如果字段设置了 `as_separate_log_field`，例如：

```yaml
attributes:
  - key: consumer
    value_source: request_header
    value: x-mse-consumer
    apply_to_log: true
    as_separate_log_field: true
```

那么要在日志中打印，需要额外设置 log_format：

```
'{"consumer":"%FILTER_STATE(wasm.consumer:PLAIN)%"}'
```

### 空配置

#### 监控

```
# counter 类型，输入 token 数量的累加值
route_upstream_model_consumer_metric_input_token{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 24

# counter 类型，输出 token 数量的累加值
route_upstream_model_consumer_metric_output_token{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 507

# counter 类型，流式请求和非流式请求消耗总时间的累加值
route_upstream_model_consumer_metric_llm_service_duration{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 6470

# counter 类型，流式请求和非流式请求次数的累加值
route_upstream_model_consumer_metric_llm_duration_count{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 2

# counter 类型，流式请求首个 token 延时的累加值
route_upstream_model_consumer_metric_llm_first_token_duration{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 340

# counter 类型，流式请求次数的累加值
route_upstream_model_consumer_metric_llm_stream_duration_count{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 1
```

以下是使用指标的几个示例：

流式请求首个 token 的平均延时：

```
irate(route_upstream_model_consumer_metric_llm_first_token_duration[2m])
/
irate(route_upstream_model_consumer_metric_llm_stream_duration_count[2m])
```

流式请求和非流式请求平均消耗的总时长：

```
irate(route_upstream_model_consumer_metric_llm_service_duration[2m])
/
irate(route_upstream_model_consumer_metric_llm_duration_count[2m])
```

#### 日志

```json
{
  "ai_log": "{\"model\":\"qwen-turbo\",\"input_token\":\"10\",\"output_token\":\"69\",\"llm_first_token_duration\":\"309\",\"llm_service_duration\":\"1955\"}"
}
```

如果请求中携带了 session ID header，日志中会自动添加 `session_id` 字段：

```json
{
  "ai_log": "{\"session_id\":\"sess_abc123\",\"model\":\"qwen-turbo\",\"input_token\":\"10\",\"output_token\":\"69\",\"llm_first_token_duration\":\"309\",\"llm_service_duration\":\"1955\"}"
}
```

#### 链路追踪

配置为空时，不会在 span 中添加额外的 attribute

### 从非 openai 协议提取 token 使用信息

在 ai-proxy 中设置协议为 original 时，以百炼为例，可作如下配置指定如何提取 model, input_token, output_token

```yaml
attributes:
  - key: model
    value_source: response_body
    value: usage.models.0.model_id
    apply_to_log: true
    apply_to_span: false
  - key: input_token
    value_source: response_body
    value: usage.models.0.input_tokens
    apply_to_log: true
    apply_to_span: false
  - key: output_token
    value_source: response_body
    value: usage.models.0.output_tokens
    apply_to_log: true
    apply_to_span: false
```

#### 监控

```
route_upstream_model_consumer_metric_input_token{ai_route="bailian",ai_cluster="qwen",ai_model="qwen-max"} 343
route_upstream_model_consumer_metric_output_token{ai_route="bailian",ai_cluster="qwen",ai_model="qwen-max"} 153
route_upstream_model_consumer_metric_llm_service_duration{ai_route="bailian",ai_cluster="qwen",ai_model="qwen-max"} 3725
route_upstream_model_consumer_metric_llm_duration_count{ai_route="bailian",ai_cluster="qwen",ai_model="qwen-max"} 1
```

#### 日志

此配置下日志效果如下：

```json
{
  "ai_log": "{\"model\":\"qwen-max\",\"input_token\":\"343\",\"output_token\":\"153\",\"llm_service_duration\":\"19110\"}"
}
```

#### 链路追踪

链路追踪的 span 中可以看到 model, input_token, output_token 三个额外的 attribute

### 配合认证鉴权记录 consumer

举例如下：

```yaml
attributes:
  - key: consumer # 配合认证鉴权记录consumer
    value_source: request_header
    value: x-mse-consumer
    apply_to_log: true
```

### 记录问题与回答

#### 仅记录当前轮次的问题与回答

```yaml
attributes:
  - key: question # 记录当前轮次的问题（最后一条用户消息）
    value_source: request_body
    value: messages.@reverse.0.content
    apply_to_log: true
  - key: answer # 在流式响应中提取大模型的回答
    value_source: response_streaming_body
    value: choices.0.delta.content
    rule: append
    apply_to_log: true
  - key: answer # 在非流式响应中提取大模型的回答
    value_source: response_body
    value: choices.0.message.content
    apply_to_log: true
```

#### 记录完整的多轮对话历史（推荐配置）

对于多轮 Agent 对话场景，使用内置属性可以大幅简化配置：

```yaml
session_id_header: "x-session-id"  # 可选，指定 session ID header
attributes:
  - key: messages     # 完整对话历史
    value_source: request_body
    value: messages
    apply_to_log: true
  - key: question     # 内置属性，自动提取最后一条用户消息
    apply_to_log: true
  - key: answer       # 内置属性，自动提取回答
    apply_to_log: true
  - key: reasoning    # 内置属性，自动提取思考过程
    apply_to_log: true
  - key: tool_calls   # 内置属性，自动提取工具调用
    apply_to_log: true
```

**内置属性说明：**

插件提供以下内置属性 key，无需配置 `value_source` 和 `value` 字段即可自动提取：

| 内置 Key | 说明 | 默认 value_source |
|---------|------|-------------------|
| `question` | 自动提取最后一条用户消息 | `request_body` |
| `answer` | 自动提取回答内容（支持 OpenAI/Claude 协议） | `response_streaming_body` / `response_body` |
| `tool_calls` | 自动提取并拼接工具调用（流式场景自动按 index 拼接 arguments） | `response_streaming_body` / `response_body` |
| `reasoning` | 自动提取思考过程（reasoning_content，如 DeepSeek-R1） | `response_streaming_body` / `response_body` |

> **注意**：如果配置了 `value_source` 和 `value`，将优先使用配置的值，以保持向后兼容。

日志输出示例：

```json
{
  "ai_log": "{\"session_id\":\"sess_abc123\",\"messages\":[{\"role\":\"user\",\"content\":\"北京天气怎么样？\"}],\"question\":\"北京天气怎么样？\",\"reasoning\":\"用户想知道北京的天气，我需要调用天气查询工具。\",\"tool_calls\":[{\"index\":0,\"id\":\"call_abc123\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"{\\\"location\\\":\\\"Beijing\\\"}\"}}],\"model\":\"deepseek-reasoner\"}"
}
```

**流式响应中的 tool_calls 处理：**

插件会自动按 `index` 字段识别每个独立的工具调用，拼接分片返回的 `arguments` 字符串，最终输出完整的工具调用列表。

## 调试

### 验证 ai_log 内容

在测试或调试过程中，可以通过开启 Higress 的 debug 日志来验证 ai_log 的内容：

```bash
# 日志格式示例
2026/01/31 23:29:30 proxy_debug_log: [ai-statistics] [nil] [test-request-id] [ai_log] attributes to be written: {"question":"What is 2+2?","answer":"4","reasoning":"...","tool_calls":[...],"session_id":"sess_123","model":"gpt-4","input_token":20,"output_token":10}
```

通过这个debug日志可以验证：
- question/answer/reasoning 是否正确提取
- tool_calls 是否正确拼接（特别是流式场景下的arguments）
- session_id 是否正确识别
- 各个字段是否符合预期

## 进阶

配合阿里云 SLS 数据加工，可以将 ai 相关的字段进行提取加工，例如原始日志为：

````
ai_log:{"question":"用python计算2的3次方","answer":"你可以使用 Python 的乘方运算符 `**` 来计算一个数的次方。计算2的3次方，即2乘以自己2次，可以用以下代码表示：\n\n```python\nresult = 2 ** 3\nprint(result)\n```\n\n运行这段代码，你会得到输出结果为8，因为2乘以自己两次等于8。","model":"qwen-max","input_token":"16","output_token":"76","llm_service_duration":"5913"}
````

使用如下数据加工脚本，可以提取出 question 和 answer：

```
e_regex("ai_log", grok("%{EXTRACTJSON}"))
e_set("question", json_select(v("json"), "question", default="-"))
e_set("answer", json_select(v("json"), "answer", default="-"))
```

提取后，SLS 中会添加 question 和 answer 两个字段，示例如下：

````
ai_log:{"question":"用python计算2的3次方","answer":"你可以使用 Python 的乘方运算符 `**` 来计算一个数的次方。计算2的3次方，即2乘以自己2次，可以用以下代码表示：\n\n```python\nresult = 2 ** 3\nprint(result)\n```\n\n运行这段代码，你会得到输出结果为8，因为2乘以自己两次等于8。","model":"qwen-max","input_token":"16","output_token":"76","llm_service_duration":"5913"}

question:用python计算2的3次方

answer:你可以使用 Python 的乘方运算符 `**` 来计算一个数的次方。计算2的3次方，即2乘以自己2次，可以用以下代码表示：

result = 2 ** 3
print(result)

运行这段代码，你会得到输出结果为8，因为2乘以自己两次等于8。

````

### 路径和内容类型过滤配置示例

#### 只处理特定 AI 路径

```yaml
enable_path_suffixes:
  - "/v1/chat/completions"
  - "/v1/embeddings"
  - "/generateContent"
```

#### 只处理特定内容类型

```yaml
enable_content_types:
  - "text/event-stream"
  - "application/json"
```

#### 处理所有路径（通配符）

```yaml
enable_path_suffixes:
  - "*"
```

#### 处理所有内容类型（空数组）

```yaml
enable_content_types: []
```

#### 完整配置示例

```yaml
enable_path_suffixes:
  - "/v1/chat/completions"
  - "/v1/embeddings"
  - "/generateContent"
enable_content_types:
  - "text/event-stream"
  - "application/json"
attributes:
  - key: model
    value_source: request_body
    value: model
    apply_to_log: true
  - key: consumer
    value_source: request_header
    value: x-mse-consumer
    apply_to_log: true
```
