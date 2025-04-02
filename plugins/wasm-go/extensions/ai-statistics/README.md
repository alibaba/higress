---
title: AI可观测
keywords: [higress, AI, observability]
description: AI可观测配置参考
---

## 介绍
提供AI可观测基础能力，包括 metric, log, trace，其后需接ai-proxy插件，如果不接ai-proxy插件的话，则需要用户进行相应配置才可生效。

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`200`

## 配置说明
插件默认请求符合openai协议格式，并提供了以下基础可观测值，用户无需特殊配置：

- metric：提供了输入token、输出token、首个token的rt（流式请求）、请求总rt等指标，支持在网关、路由、服务、模型四个维度上进行观测
- log：提供了 input_token, output_token, model, llm_service_duration, llm_first_token_duration 等字段

用户还可以通过配置的方式对可观测的值进行扩展：

| 名称             | 数据类型  | 填写要求 | 默认值 | 描述                     |
|----------------|-------|------|-----|------------------------|
| `attributes` | []Attribute | 非必填  | -   | 用户希望记录在log/span中的信息 |

Attribute 配置说明:

| 名称             | 数据类型  | 填写要求 | 默认值 | 描述                     |
|----------------|-------|-----|-----|------------------------|
| `key`         | string | 必填  | -   | attrribute 名称           |
| `value_source` | string | 必填  | -   | attrribute 取值来源，可选值为 `fixed_value`, `request_header`, `request_body`, `response_header`, `response_body`, `response_streaming_body`             |
| `value`      | string | 必填  | -   | attrribute 取值 key value/path |
| `default_value`      | string | 非必填  | -   | attrribute 默认值 |
| `rule`      | string | 非必填  | -   | 从流式响应中提取 attrribute 的规则，可选值为 `first`, `replace`, `append`|
| `apply_to_log`      | bool | 非必填  | false  | 是否将提取的信息记录在日志中 |
| `apply_to_span`      | bool | 非必填  | false  | 是否将提取的信息记录在链路追踪span中 |

`value_source` 的各种取值含义如下：

- `fixed_value`：固定值
- `request_header` ： attrribute 值通过 http 请求头获取，value 配置为 header key
- `request_body` ：attrribute 值通过请求 body 获取，value 配置格式为 gjson 的 jsonpath
- `response_header` ：attrribute 值通过 http 响应头获取，value 配置为header key
- `response_body` ：attrribute 值通过响应 body 获取，value 配置格式为 gjson 的 jsonpath
- `response_streaming_body` ：attrribute 值通过流式响应 body 获取，value 配置格式为 gjson 的 jsonpath


当 `value_source` 为 `response_streaming_body` 时，应当配置 `rule`，用于指定如何从流式body中获取指定值，取值含义如下：

- `first`：多个chunk中取第一个有效chunk的值
- `replace`：多个chunk中取最后一个有效chunk的值
- `append`：拼接多个有效chunk中的值，可用于获取回答内容

## 配置示例
如果希望在网关访问日志中记录ai-statistic相关的统计值，需要修改log_format，在原log_format基础上添加一个新字段，示例如下：

```yaml
'{"ai_log":"%FILTER_STATE(wasm.ai_log:PLAIN)%"}'
```

### 空配置
#### 监控
```
route_upstream_model_metric_input_token{ai_route="llm",ai_cluster="outbound|443||qwen.dns",ai_model="qwen-turbo"} 10
route_upstream_model_metric_llm_duration_count{ai_route="llm",ai_cluster="outbound|443||qwen.dns",ai_model="qwen-turbo"} 1
route_upstream_model_metric_llm_first_token_duration{ai_route="llm",ai_cluster="outbound|443||qwen.dns",ai_model="qwen-turbo"} 309
route_upstream_model_metric_llm_service_duration{ai_route="llm",ai_cluster="outbound|443||qwen.dns",ai_model="qwen-turbo"} 1955
route_upstream_model_metric_output_token{ai_route="llm",ai_cluster="outbound|443||qwen.dns",ai_model="qwen-turbo"} 69
```

#### 日志
```json
{
  "ai_log":"{\"model\":\"qwen-turbo\",\"input_token\":\"10\",\"output_token\":\"69\",\"llm_first_token_duration\":\"309\",\"llm_service_duration\":\"1955\"}"
}
```

#### 链路追踪
配置为空时，不会在span中添加额外的attribute

### 从非openai协议提取token使用信息
在ai-proxy中设置协议为original时，以百炼为例，可作如下配置指定如何提取model, input_token, output_token

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
此配置下日志效果如下：
```json
{
  "ai_log": "{\"model\":\"qwen-max\",\"input_token\":\"343\",\"output_token\":\"153\",\"llm_service_duration\":\"19110\"}"  
}
```

#### 链路追踪
链路追踪的 span 中可以看到 model, input_token, output_token 三个额外的 attribute

### 配合认证鉴权记录consumer
举例如下： 
```yaml
attributes:
  - key: consumer # 配合认证鉴权记录consumer
    value_source: request_header
    value: x-mse-consumer
    apply_to_log: true
```

### 记录问题与回答
```yaml
attributes:
  - key: question # 记录问题
    value_source: request_body
    value: messages.@reverse.0.content
    apply_to_log: true
  - key: answer   # 在流式响应中提取大模型的回答
    value_source: response_streaming_body
    value: choices.0.delta.content
    rule: append
    apply_to_log: true
  - key: answer   # 在非流式响应中提取大模型的回答
    value_source: response_body
    value: choices.0.message.content
    apply_to_log: true
```

## 进阶
配合阿里云SLS数据加工，可以将ai相关的字段进行提取加工，例如原始日志为：

```
ai_log:{"question":"用python计算2的3次方","answer":"你可以使用 Python 的乘方运算符 `**` 来计算一个数的次方。计算2的3次方，即2乘以自己2次，可以用以下代码表示：\n\n```python\nresult = 2 ** 3\nprint(result)\n```\n\n运行这段代码，你会得到输出结果为8，因为2乘以自己两次等于8。","model":"qwen-max","input_token":"16","output_token":"76","llm_service_duration":"5913"}
```

使用如下数据加工脚本，可以提取出question和answer：

```
e_regex("ai_log", grok("%{EXTRACTJSON}"))
e_set("question", json_select(v("json"), "question", default="-"))
e_set("answer", json_select(v("json"), "answer", default="-"))
```

提取后，SLS中会添加question和answer两个字段，示例如下：

```
ai_log:{"question":"用python计算2的3次方","answer":"你可以使用 Python 的乘方运算符 `**` 来计算一个数的次方。计算2的3次方，即2乘以自己2次，可以用以下代码表示：\n\n```python\nresult = 2 ** 3\nprint(result)\n```\n\n运行这段代码，你会得到输出结果为8，因为2乘以自己两次等于8。","model":"qwen-max","input_token":"16","output_token":"76","llm_service_duration":"5913"}

question:用python计算2的3次方

answer:你可以使用 Python 的乘方运算符 `**` 来计算一个数的次方。计算2的3次方，即2乘以自己2次，可以用以下代码表示：

result = 2 ** 3
print(result)

运行这段代码，你会得到输出结果为8，因为2乘以自己两次等于8。

```