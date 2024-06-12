# 简介

# 配置说明
| Name | Type | Requirement | Default | Description |
| :-: | :-:  | :-:  | :-: | :-: |
| serviceSource | string | requried | - | 服务来源，填dns |
| serviceName | string | requried | - | 服务名 |
| servicePort | string | requried | - | 服务端口 |
| domain | string | requried | - | 阿里云内容安全endpoint |
| ak | string | requried | - | 阿里云AK |
| sk | string | requried | - | 阿里云SK |


# 配置示例
```yaml
serviceSource: "dns"
serviceName: "safecheck"
servicePort: 443
domain: "green-cip.cn-shanghai.aliyuncs.com"
ak: "XXXXXXXXX"
sk: "XXXXXXXXXXXXXXX"
```