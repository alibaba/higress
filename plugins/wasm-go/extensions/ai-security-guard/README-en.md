## Introduction
Integrate with Aliyun content security service for detections of input and output of LLMs, ensuring that application content is legal and compliant.

## Configuration
| Name | Type | Requirement | Default | Description |
| ------------ | ------------ | ------------ | ------------ | ------------ |
| `serviceName` | string | requried | - | service name |
| `servicePort` | string | requried | - | service port |
| `serviceHost` | string | requried | - | Host of Aliyun content security service endpoint |
| `accessKey` | string | requried | - | Aliyun accesskey |
| `secretKey` | string | requried | - | Aliyun secretkey |
| `checkRequest` | bool | optional | false | check if the input is leagal |
| `checkResponse` | bool | optional | false | check if the output is leagal |


## Examples of configuration
### Check if the input is leagal

```yaml
serviceName: safecheck.dns
servicePort: 443
serviceHost: "green-cip.cn-shanghai.aliyuncs.com"
accessKey: "XXXXXXXXX"
secretKey: "XXXXXXXXXXXXXXX"
checkRequest: true
```

### Check if both the input and output are leagal

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