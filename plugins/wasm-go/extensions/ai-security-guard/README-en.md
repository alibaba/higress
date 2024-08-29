## 简介
Integrate with Aliyun content security service for detections of input and output of LLMs, ensuring that application content is legal and compliant.

## 配置说明
| Name | Type | Requirement | Default | Description |
| ------------ | ------------ | ------------ | ------------ | ------------ |
| `serviceSource` | string | requried | - | service source, such as `dns` |
| `serviceName` | string | requried | - | service name |
| `servicePort` | string | requried | - | service port |
| `domain` | string | requried | - | Host of Aliyun content security service endpoint |
| `ak` | string | requried | - | Aliyun accesskey |
| `sk` | string | requried | - | Aliyun secretkey |
| `checkRequest` | bool | optional | - | check if the input is leagal |
| `checkresponse` | bool | optional | - | check if the output is leagal |


## 配置示例
### check if the input is leagal

```yaml
serviceSource: "dns"
serviceName: "safecheck"
servicePort: 443
domain: "green-cip.cn-shanghai.aliyuncs.com"
ak: "XXXXXXXXX"
sk: "XXXXXXXXXXXXXXX"
checkRequest: true
```

### check if both the input and output are leagal

```yaml
serviceSource: "dns"
serviceName: "safecheck"
servicePort: 443
domain: "green-cip.cn-shanghai.aliyuncs.com"
ak: "XXXXXXXXX"
sk: "XXXXXXXXXXXXXXX"
checkRequest: true
checkresponse: true
```

## observability
### Metric
ai-security-guard plugin provides following metrics:
- `ai_sec_request_deny`: count of requests denied at request phase
- `ai_sec_response_deny`: count of requests denied at response phase

### Trace
ai-security-guard plugin provides following span attributes:
- `ai_sec_risklabel`: risk type of this request
- `ai_sec_deny_phase`: denied phase of this request, value can be request/response