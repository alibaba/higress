# 功能说明
waf插件实现了基于ModSecurity的规则防护引擎，可以根据用户配置的规则屏蔽可疑请求，并支持OWASP CRS，为站点提供基础的防护功能。

# 配置字段
| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
| useCRS | bool | 选填 | false | 是否开启OWASP CRS，详情可参考[coreruleset](https://github.com/coreruleset/coreruleset/tree/v3.3.2) |
| secRules | array of string | 选填 | - | 用户自定义的waf防护规则，语法规则可参考[ModSecurity中文手册](http://www.modsecurity.cn/chm/) |

# 配置示例
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

# 对特定路由或域名开启
```yaml
useCRS: true
secRules: 
  - "SecDebugLogLevel 3"
  - "SecRuleEngine On"
  - "SecAction \"id:100,phase:1,pass\""
  - "SecRule REQUEST_URI \"@streq /admin\" \"id:101,phase:1,t:lowercase,deny\""
  - "SecRule REQUEST_BODY \"@rx maliciouspayload\" \"id:102,phase:2,t:lowercase,deny\""
_rules_:
- _match_route_:
    - "route-1"
  secRules:
    - "SecDebugLogLevel 3"
    - "SecRuleEngine On"
    - "SecAction \"id:102,phase:1,deny\""
- _match_domain_:
    - "*.example.com"
    - test.com
  secRules:
    - "SecDebugLogLevel 3"
    - "SecRuleEngine On"
    - "SecAction \"id:102,phase:1,pass\""
```

此例 `_match_route_` 中指定的 `route-1` 即在创建网关路由时填写的路由名称，当匹配到这两个路由时，将使用此段配置； 此例 `_match_domain_` 中指定的 `*.example.com` 和 `test.com` 用于匹配请求的域名，当发现域名匹配时，将使用此段配置； 配置的匹配生效顺序，将按照 `_rules_` 下规则的排列顺序，匹配第一个规则后生效对应配置，后续规则将被忽略。
