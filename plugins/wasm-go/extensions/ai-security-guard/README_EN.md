---
title: AI Content Security
keywords: [higress, AI, security]
description: Alibaba Cloud content security
---


## Introduction
Integrate with Aliyun content security service for detections of input and output of LLMs, ensuring that application content is legal and compliant.

## Runtime Properties

Plugin Phase: `CUSTOM`
Plugin Priority: `300`

## Configuration
| Name | Type | Requirement | Default | Description |
| ------------ | ------------ | ------------ | ------------ | ------------ |
| `serviceName` | string | requried | - | service name |
| `servicePort` | string | requried | - | service port |
| `serviceHost` | string | requried | - | Host of Aliyun content security service endpoint |
| `accessKey` | string | requried | - | Aliyun accesskey |
| `secretKey` | string | requried | - | Aliyun secretkey |
| `checkRequest` | bool | optional | false | check if the input is legal |
| `checkResponse` | bool | optional | false | check if the output is legal |
| `requestCheckService` | string | optional | llm_query_moderation | Aliyun yundun service name for input check |
| `responseCheckService` | string | optional | llm_response_moderation | Aliyun yundun service name for output check |
| `requestContentJsonPath` | string | optional | `messages.@reverse.0.content` | Specify the jsonpath of the content to be detected in the request body |
| `responseContentJsonPath` | string | optional | `choices.0.message.content` | Specify the jsonpath of the content to be detected in the response body |
| `responseStreamContentJsonPath` | string | optional | `choices.0.delta.content` | Specify the jsonpath of the content to be detected in the streaming response body |
| `denyCode` | int | optional | 200 | Response status code when the specified content is illegal |
| `denyMessage` | string | optional | Drainage/non-streaming response in openai format, the answer content is the suggested answer from Alibaba Cloud content security | Response content when the specified content is illegal |
| `protocol` | string | optional | openai | protocol format, `openai` or `original` |


## Examples of configuration
### Check if the input is legal

```yaml
serviceName: safecheck.dns
servicePort: 443
serviceHost: "green-cip.cn-shanghai.aliyuncs.com"
accessKey: "XXXXXXXXX"
secretKey: "XXXXXXXXXXXXXXX"
checkRequest: true
```

### Check if both the input and output are legal

```yaml
serviceName: safecheck.dns
servicePort: 443
serviceHost: green-cip.cn-shanghai.aliyuncs.com
accessKey: "XXXXXXXXX"
secretKey: "XXXXXXXXXXXXXXX"
checkRequest: true
checkResponse: true
```

## Observability
### Metric
ai-security-guard plugin provides following metrics:
- `ai_sec_request_deny`: count of requests denied at request phase
- `ai_sec_response_deny`: count of requests denied at response phase

### Trace
ai-security-guard plugin provides following span attributes:
- `ai_sec_risklabel`: risk type of this request
- `ai_sec_deny_phase`: denied phase of this request, value can be request/response