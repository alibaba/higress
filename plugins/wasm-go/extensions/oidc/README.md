# 功能说明
`Oidc`本插件实现了 OIDC 认证插件

# 配置字段


| 字段            | 数据类型   | 填写要求 |  默认值 | 描述                                  |
|---------------|--------|------|--------|-------------------------------------|
| Issuer        | string | 必填   |    -   | 设置认证服务的Issuer，即签发人。                 |
| RedirectURL   | string | 必填   |    -   | 输入授权成功后的重定向地址，需要与OIDC中配置的重定向地址保持一致。 |
| Client-ID     | string | 必填   |    -   | 输入服务注册的应用ID。                        |
| Client-Secret | string | 必填   |    -   | 输入服务注册的应用Secret。                    |
| Scopes        | Array  | 必填   |    -   | 输入授权作用域.。                           |
| SkipExpiryCheck | bool   | 选填   |    -   | 是否检测IDToken过期。                      |
| serviceSource | string | 必填   |    -   | 输入服务注册来源。                           |
| serviceName   | string | 必填   |    -   | 输入服务注册的名称。                          |
| servicePort   | int    | 必填   |    -   | 输入服务注册的服务端口。                        |
| serviceHost   | string | 必填   |    -   | 输入服务注册的主机名。                         |
# 配置示例
- 固定ip
```yaml
Issuer: "http://127.0.0.1:9090/realms/myrealm"
clientID: "myclinet"
clientSecret: "EdKdKBX4N0jtYuPD4aGxZWiI7EVh4pr9"
RedirectURL: "http://foo.bar.com/foo/oidc/callback"
Scopes:
  - "openid"
  - "email"
serviceSource: "ip"
serviceName: "keyclocak"
servicePort: 80
serviceHost: "127.0.0.1:9090"
```
- DNS域名
```yaml
Issuer: "https://dev-650jsqsvuyrk4ahg.us.auth0.com/"
clientID: "BZQS8h0jV3pnhLcHYzRIgBLuaWj57SCu"
clientSecret: "kbaZxflcvryA1Hlua56c29rM007v1Jj8PKW5YjysoNai833DCP_EYCDJSMsoEUNZ"
RedirectURL: "http://foo.bar.com/foo/oidc/callback"
Scopes:
- "openid"
- "email"
serviceSource: "dns"
serviceName: "okta"
servicePort: 443
domain: "dev-650jsqsvuyrk4ahg.us.auth0.com"
```






