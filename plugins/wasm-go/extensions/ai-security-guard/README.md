# 简介
通过对接阿里云内容安全检测大模型的输入输出，保障AI应用内容合法合规。

# 配置说明
| Name | Type | Requirement | Default | Description |
| ------------ | ------------ | ------------ | ------------ | ------------ |
| `serviceSource` | string | requried | - | 服务来源，填dns |
| `serviceName` | string | requried | - | 服务名 |
| `servicePort` | string | requried | - | 服务端口 |
| `domain` | string | requried | - | 阿里云内容安全endpoint |
| `ak` | string | requried | - | 阿里云AK |
| `sk` | string | requried | - | 阿里云SK |
| `request` | string | requried | - | 请求阶段 |
| `response` | string | requried | - | 阿里云SK |


# 配置示例
```yaml
serviceSource: "dns"
serviceName: "safecheck"
servicePort: 443
domain: "green-cip.cn-shanghai.aliyuncs.com"
ak: "XXXXXXXXX"
sk: "XXXXXXXXXXXXXXX"
request: true
response: true
```