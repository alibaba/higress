---
title: AI Statistics
keywords: [higress, AI, observability]
description: AI Statistics plugin configuration reference
---

## Introduction
Provides basic AI observability capabilities, including metric, log, and trace. The ai-proxy plug-in needs to be connected afterwards. If the ai-proxy plug-in is not connected, the user needs to configure it accordingly to take effect.

## Runtime Properties

Plugin Phase: `CUSTOM`
Plugin Priority: `200`

## Configuration instructions
The default request of the plug-in conforms to the openai protocol format and provides the following basic observable values. Users do not need special configuration:

- metric: It provides indicators such as input token, output token, rt of the first token (streaming request), total request rt, etc., and supports observation in the four dimensions of gateway, routing, service, and model.
- log: Provides input_token, output_token, model, llm_service_duration, llm_first_token_duration and other fields

Users can also expand observable values ​​through configuration:

| Name             | Type  | Required | Default | Description |
|----------------|-------|------|-----|------------------------|
| `attributes` | []Attribute | required  | -   | Information that the user wants to record in log/span |

Attribute Configuration instructions:

| Name             | Type  | Required | Default | Description |
|----------------|-------|-----|-----|------------------------|
| `key`         | string | required  | -   | attrribute key           |
| `value_source` | string | required  | -   | attrribute value source, optional values ​​are `fixed_value`, `request_header`, `request_body`, `response_header`, `response_body`, `response_streaming_body`             |
| `value`      | string | required  | -   | how to get attrribute value |
| `default_value`      | string | optional  | -   | default value for attribute |
| `rule`      | string | optional  | -   | Rule to extract attribute from streaming response, optional values ​​are `first`, `replace`, `append`|
| `apply_to_log`      | bool | optional  | false  | Whether to record the extracted information in the log |
| `apply_to_span`      | bool | optional  | false  | Whether to record the extracted information in the link tracking span |

The meanings of various values for `value_source` ​​are as follows:

- `fixed_value`: fixed value
- `request_header`: The attrribute is obtained through the http request header
- `request_body`: The attrribute is obtained through the http request body
- `response_header`: The attrribute is obtained through the http response header
- `response_body`: The attrribute is obtained through the http response body
- `response_streaming_body`: The attrribute is obtained through the http streaming response body


When `value_source` is `response_streaming_body`, `rule` should be configured to specify how to obtain the specified value from the streaming body. The meaning of the value is as follows:

- `first`: extract value from the first valid chunk
- `replace`: extract value from the last valid chunk
- `append`: join value pieces from all valid chunks

## Configuration example
If you want to record ai-statistic related statistical values in the gateway access log, you need to modify log_format and add a new field based on the original log_format. The example is as follows:

```yaml
'{"ai_log":"%FILTER_STATE(wasm.ai_log:PLAIN)%"}'
```

### Empty
#### Metric

```
# counter, cumulative count of input tokens
route_upstream_model_consumer_metric_input_token{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 24

# counter, cumulative count of output tokens
route_upstream_model_consumer_metric_output_token{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 507

# counter, cumulative total duration of both streaming and non-streaming requests
route_upstream_model_consumer_metric_llm_service_duration{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 6470

# counter, cumulative count of both streaming and non-streaming requests
route_upstream_model_consumer_metric_llm_duration_count{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 2

# counter, cumulative latency of the first token in streaming requests
route_upstream_model_consumer_metric_llm_first_token_duration{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 340

# counter, cumulative count of streaming requests
route_upstream_model_consumer_metric_llm_stream_duration_count{ai_route="ai-route-aliyun.internal",ai_cluster="outbound|443||llm-aliyun.internal.dns",ai_model="qwen-turbo",ai_consumer="none"} 1
```

Below are some example usages of these metrics:

Average latency of the first token in streaming requests:

```
irate(route_upstream_model_consumer_metric_llm_first_token_duration[2m])
/
irate(route_upstream_model_consumer_metric_llm_stream_duration_count[2m])
```

Average process duration of both streaming and non-streaming requests:

```
irate(route_upstream_model_consumer_metric_llm_service_duration[2m])
/
irate(route_upstream_model_consumer_metric_llm_duration_count[2m])
```

#### Log
```json
{
  "ai_log":"{\"model\":\"qwen-turbo\",\"input_token\":\"10\",\"output_token\":\"69\",\"llm_first_token_duration\":\"309\",\"llm_service_duration\":\"1955\"}"
}
```

#### Trace
When the configuration is empty, no additional attributes will be added to the span.

### Extract token usage information from non-openai protocols
When setting the protocol to original in ai-proxy, taking Alibaba Cloud Bailian as an example, you can make the following configuration to specify how to extract `model`, `input_token`, `output_token`

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
#### Metric

```
route_upstream_model_consumer_metric_input_token{ai_route="bailian",ai_cluster="qwen",ai_model="qwen-max"} 343
route_upstream_model_consumer_metric_output_token{ai_route="bailian",ai_cluster="qwen",ai_model="qwen-max"} 153
route_upstream_model_consumer_metric_llm_service_duration{ai_route="bailian",ai_cluster="qwen",ai_model="qwen-max"} 3725
route_upstream_model_consumer_metric_llm_duration_count{ai_route="bailian",ai_cluster="qwen",ai_model="qwen-max"} 1
```

#### Log
```json
{
  "ai_log": "{\"model\":\"qwen-max\",\"input_token\":\"343\",\"output_token\":\"153\",\"llm_service_duration\":\"19110\"}"
}
```

#### Trace
Three additional attributes `model`, `input_token`, and `output_token` can be seen in the trace spans.

### Cooperate with authentication and authentication record consumer
```yaml
attributes:
  - key: consumer
    value_source: request_header
    value: x-mse-consumer
    apply_to_log: true
```

### Record questions and answers
```yaml
attributes:
  - key: question
    value_source: request_body
    value: messages.@reverse.0.content
    apply_to_log: true
  - key: answer
    value_source: response_streaming_body
    value: choices.0.delta.content
    rule: append
    apply_to_log: true
  - key: answer
    value_source: response_body
    value: choices.0.message.content
    apply_to_log: true
```