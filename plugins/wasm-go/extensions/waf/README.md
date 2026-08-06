---
title: WAF
keywords: [higress,waf]
description: WAF 插件配置参考
---

## 功能说明
waf插件实现了基于ModSecurity的规则防护引擎，可以根据用户配置的规则屏蔽可疑请求，并支持OWASP CRS，为站点提供基础的防护功能。

## 运行属性

插件执行阶段：`授权阶段`
插件执行优先级：`330`


## 配置字段
| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
| useCRS | bool | 选填 | false | 是否开启OWASP CRS，详情可参考[coreruleset](https://github.com/coreruleset/coreruleset/tree/v3.3.2) |
| secRules | array of string | 选填 | - | 用户自定义的waf防护规则，语法规则可参考[ModSecurity中文手册](http://www.modsecurity.cn/chm/) |

## 配置示例
```yaml
useCRS: true
secRules: 
  - "SecDebugLogLevel 3"
  - "SecRuleEngine On"
  - "SecAction \"id:100,phase:1,pass\""
  - "SecRule REQUEST_URI \"@streq /admin\" \"id:101,phase:1,t:lowercase,deny\""
  - "SecRule REQUEST_BODY \"@rx maliciouspayload\" \"id:102,phase:2,t:lowercase,deny\""
```

根据该配置，以下请求将被禁止访问：
```bash
curl http://example.com/admin
curl http://example.com -d "maliciouspayload"
```
