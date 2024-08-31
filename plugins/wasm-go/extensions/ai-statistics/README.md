# 介绍
提供AI可观测基础能力，其后需接ai-proxy插件，如果不接ai-proxy插件的话，则只支持openai协议。

# 配置说明

| 名称             | 数据类型  | 填写要求 | 默认值 | 描述                     |
|----------------|-------|------|-----|------------------------|
| `enable`       | bool  | 必填   | -   | 是否开启ai统计功能             |
| `tracing_span` | array | 非必填  | -   | 自定义tracing span tag 配置 |

## tracing_span 配置说明
| 名称             | 数据类型  | 填写要求 | 默认值 | 描述                     |
|----------------|-------|-----|-----|------------------------|
| `key`         | string | 必填  | -   | tracing tag 名称           |
| `value_source`        | string | 必填  | -   | tag 取值来源             |
| `value`      | string | 必填  | -   | tag 取值 key value/path           |

value_source为 tag 值的取值来源，可选配置值有 4 个：
- property ： tag 值通过proxywasm.GetProperty()方法获取，value配置GetProperty()方法要提取的key名
- requeset_header ： tag 值通过http请求头获取，value配置为header key
- request_body ：tag 值通过请求body获取，value配置格式为 gjson的 GJSON PATH 语法
- response_header ： tag 值通过http响应头获取，value配置为header key

举例如下： 
```yaml
tracing_label:
- key: "session_id"
  value_source: "requeset_header"
  value: "session_id"
- key: "user_content"
  value_source: "request_body"
  value: "input.messages.1.content"
```

开启后 metrics 示例：
```
route_upstream_model_input_token{ai_route="openai",ai_cluster="qwen",ai_model="qwen-max"} 21
route_upstream_model_output_token{ai_route="openai",ai_cluster="qwen",ai_model="qwen-max"} 17
```

日志示例：

```json
{
    "model": "qwen-max",
    "input_token": "21",
    "output_token": "17",
    "authority": "dashscope.aliyuncs.com",
    "bytes_received": "336",
    "bytes_sent": "1675",
    "duration": "1590",
    "istio_policy_status": "-",
    "method": "POST",
    "path": "/v1/chat/completions",
    "protocol": "HTTP/1.1",
    "request_id": "5895f5a9-e4e3-425b-98db-6c6a926195b7",
    "requested_server_name": "-",
    "response_code": "200",
    "response_flags": "-",
    "route_name": "openai",
    "start_time": "2024-06-18T09:37:14.078Z",
    "trace_id": "-",
    "upstream_cluster": "qwen",
    "upstream_service_time": "496",
    "upstream_transport_failure_reason": "-",
    "user_agent": "PostmanRuntime/7.37.3",
    "x_forwarded_for": "-"
}
```