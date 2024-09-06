## 简介
通过对接阿里云内容安全检测大模型的输入输出，保障AI应用内容合法合规。

## 配置说明
| Name | Type | Requirement | Default | Description |
| ------------ | ------------ | ------------ | ------------ | ------------ |
| `serviceName` | string | requried | - | 服务名 |
| `servicePort` | string | requried | - | 服务端口 |
| `serviceHost` | string | requried | - | 阿里云内容安全endpoint的域名 |
| `accessKey` | string | requried | - | 阿里云AK |
| `secretKey` | string | requried | - | 阿里云SK |
| `checkRequest` | bool | optional | false | 检查提问内容是否合规 |
| `checkResponse` | bool | optional | false | 检查大模型的回答内容是否合规，生效时会使流式响应变为非流式 |


## 配置示例
### 检测输入内容是否合规

```yaml
serviceName: safecheck.dns
servicePort: 443
serviceHost: "green-cip.cn-shanghai.aliyuncs.com"
accessKey: "XXXXXXXXX"
secretKey: "XXXXXXXXXXXXXXX"
checkRequest: true
```

### 检测输入与输出是否合规

```yaml
serviceName: safecheck.dns
servicePort: 443
serviceHost: green-cip.cn-shanghai.aliyuncs.com
accessKey: "XXXXXXXXX"
secretKey: "XXXXXXXXXXXXXXX"
checkRequest: true
checkResponse: true
```

## 可观测
### Metric
ai-security-guard 插件提供了以下监控指标：
- `ai_sec_request_deny`: 请求内容安全检测失败请求数
- `ai_sec_response_deny`: 模型回答安全检测失败请求数

### Trace
如果开启了链路追踪，ai-security-guard 插件会在请求 span 中添加以下 attributes:
- `ai_sec_risklabel`: 表示请求命中的风险类型
- `ai_sec_deny_phase`: 表示请求被检测到风险的阶段（取值为request或者response）

## 请求示例
```bash
curl http://localhost/v1/chat/completions \
-H "Content-Type: application/json" \
-d '{
  "model": "gpt-4o-mini",
  "messages": [
    {
      "role": "user",
      "content": "这是一段非法内容"
    }
  ]
}'
```

请求内容会被发送到阿里云内容安全服务进行检测，如果请求内容检测结果为非法，网关将返回形如以下的回答：

```json
{
    "id": "chatcmpl-123",
    "object": "chat.completion",
    "created": 1677652288,
    "model": "gpt-4o-mini",
    "system_fingerprint": "fp_44709d6fcb",
    "choices": [
        {
            "index": 0,
            "message": {
                "role": "assistant",
                "content": "作为一名人工智能助手，我不能提供涉及色情、暴力、政治等敏感话题的内容。如果您有其他相关问题，欢迎您提问。",
            },
            "logprobs": null,
            "finish_reason": "stop"
        }
    ]
}
```