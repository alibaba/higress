# 介绍
提供AI可观测基础能力，包括 metric, log, trace，其后需接ai-proxy插件，如果不接ai-proxy插件的话，则只支持openai协议。

# 配置说明
插件提供了以下基础可观测值，用户无需配置：
- metric：提供了输入token、输出token、首个token的rt（流式请求）、请求总rt等指标，分别在网关、路由、服务、模型四个维度上生效
- log：提供了 input_token, output_token, model, cluster, route, llm_service_duration, llm_first_token_duration 等字段
- trace：提供了 input_token, output_token, model, cluster, route, llm_service_duration, llm_first_token_duration 等字段

用户还可以通过配置的方式对可观测的值进行扩展：

| 名称             | 数据类型  | 填写要求 | 默认值 | 描述                     |
|----------------|-------|------|-----|------------------------|
| `spanAttributes` | []Attribute | 非必填  | -   | 自定义 ai 请求中日志字段 |
| `logAttributes` | []Attribute | 非必填  | -   | 自定义 ai 请求中链路追踪 span attrribute |

## Attribute 配置说明
| 名称             | 数据类型  | 填写要求 | 默认值 | 描述                     |
|----------------|-------|-----|-----|------------------------|
| `key`         | string | 必填  | -   | attrribute 名称           |
| `value_source` | string | 必填  | -   | attrribute 取值来源，可选值为 `fixed_value`, `request_header`, `request_body`, `response_header`, `response_body`, `response_streaming_body`             |
| `value`      | string | 必填  | -   | attrribute 取值 key value/path |
| `rule`      | string | 非必填  | -   | 从流式响应中提取 attrribute 的规则，可选值为 `first`, `replace`, `append`|

`value_source` 的各种取值含义如下：
- `fixed_value`：固定值
- `requeset_header` ： attrribute 值通过 http 请求头获取，value 配置为 header key
- `request_body` ：attrribute 值通过请求 body 获取，value 配置格式为 gjson 的 jsonpath
- `response_header` ：attrribute 值通过 http 响应头获取，value 配置为header key
- `response_body` ：attrribute 值通过响应 body 获取，value 配置格式为 gjson 的 jsonpath
- `response_streaming_body` ：attrribute 值通过流式响应 body 获取，value 配置格式为 gjson 的 jsonpath


当 `value_source` 为 `response_streaming_body` 时，应当配置 `rule`，用于指定如何从流式body中获取指定值，取值含义如下：
- `first`：（多个chunk中取第一个chunk的值），
- `replace`：（多个chunk中取最后一个chunk的值），
- `append`：（拼接多个chunk中的值，可用于获取回答内容）

## 配置示例
举例如下： 
```yaml
logAttributes:
  - key: consumer # 配合认证鉴权记录consumer
    value_source: request_header
    value: x-mse-consumer
  - key: question # 记录问题
    value_source: request_body
    value: messages.@reverse.0.content
  - key: answer   # 在流式响应中提取大模型的回答
    value_source: response_streaming_body
    value: choices.0.delta.content
    rule: append
  - key: answer   # 在非流式响应中提取大模型的回答
    value_source: response_body
    value: choices.0.message.content
spanAttributes:
  - key: consumer
    value_source: request_header
    value: x-mse-consumer
```

## 可观测指标示例
### Metric
开启后 metrics 示例：
```
route_upstream_model_input_token{ai_route="llm",ai_cluster="outbound|443||qwen.dns",ai_model="qwen-max"} 21
route_upstream_model_output_token{ai_route="llm",ai_cluster="outbound|443||qwen.dns",ai_model="qwen-max"} 17
```

### Log
要想在日志中看到相关统计信息，需要在meshconfig中修改log_format，添加以下字段
```yaml
access_log:
- name: envoy.access_loggers.file
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog
    log_format:
      text_format_source:
        inline_string: '{"ai-statistics":"%FILTER_STATE(wasm.ai_log:PLAIN)%"}'
    path: /dev/stdout
```

日志示例：

```json
{
  "ai-statistics": {
    "consumer": "21321r9fncsb2dq",
    "route": "llm",
    "output_token": "17",
    "llm_service_duration": "3518",
    "answer": "我是来自阿里云的超大规模语言模型，我叫通义千问。",
    "request_id": "2d8ffda2-dc43-933d-ad72-7679cfbbaf15",
    "question": "你是谁",
    "cluster": "outbound|443||qwen.dns",
    "model": "qwen-max",
    "input_token": "10",
    "llm_first_token_duration": "676"
  }
}
```