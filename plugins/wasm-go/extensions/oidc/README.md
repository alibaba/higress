# 功能说明
`Oidc`本插件实现了 OIDC 认证插件

# 配置字段

| 字段                | 数据类型   | 填写要求 | 默认值      | 描述                                                          |
|-------------------|--------|------|----------|-------------------------------------------------------------|
| issuer            | string | 必填   | -        | 设置认证服务的 issuer ，即签发人。                                       |
| redirectUrl       | string | 必填   | -        | 输入授权成功后的重定向地址，需要与 OIDC 中配置的重定向地址保持一致。   (后缀需为oidc/callback) |
| clientId          | string | 必填   | -        | 输入服务注册的应用 ID 。                                              |
| clientSecret      | string | 必填   | -        | 输入服务注册的应用 Secret 。                                          |
| scopes            | Array  | 必填   | -        | 输入授权作用域.。                                                   |
| clientDomain      |    string  |     必填  | -        | -                                                           | 输入 Cookie 的域名，认证通过后会将 Cookie 发送到指定的域名 ，保持登录状态。                 |
| skipExpiryCheck   | bool   | 选填   | -        | 是否检测 IDToken 过期。                                            |
| skipIssuerCheck   | bool   | 选填   | -        | SkipIssuerCheck 用于特殊情况，其中调用者希望推迟对签发者的验证。当启用时，调用者必须独立验证令牌的签发者是否为已知的有效值。不匹配的签发者通常指示客户端配置错误。如果不希望发生不匹配，请检查所提供的签发者URL是否正确，而不是启用这个选项。                                           |
| timeOut           | int | 选填   | 500 （毫秒） | 控制请求的超时时长，若是一直得到超时错误，可以选择增大时长                               |
| secureCookie      | bool | 选填   | false    | cookie 是否设置 secure 参数                                       |
| serviceSource     | string | 必填   | -        | 输入认证 oidc 服务注册来源。                                           |
| serviceName       | string | 必填   | -        | 输入认证 oidc 服务注册的名称。                                          |
| servicePort       | int    | 必填   | -        | 输入认证 oidc 服务注册的服务端口。                                        |
| serviceHost       | string | 必填   | -        | 输入认证 oidc 服务注册的主机名。                                         |                                          |
# 配置示例
- 固定ip
```yaml
Issuer: "http://127.0.0.1:9090/realms/myrealm"
RedirectURL: "http://test.com/bar/oidc/callback"
Scopes:
  - "openid"
  - "email"
clientDomain: "test.com"
clientID: "myclinet"
clientSecret: "xxxxxx"
serviceHost: "127.0.0.1:9090"
serviceName: "keyclocak"
servicePort: 80
serviceSource: "ip"
```
- DNS域名
- 在服务来源中注册好服务后，创建对应的ingress
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
  annotations:
     higress.io/destination: okta.dns    
     higress.io/backend-protocol: "HTTPS"
     higress.io/ignore-path-case: "false"
spec:
  ingressClassName: higress
  rules:
    - host: foo.bar.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              resource:
                apiGroup: networking.higress.io
                kind: McpBridge
                name: default

```
- 创建wasm插件
```yaml
Issuer: "https://dev-650jsqsvuyrk4ahg.us.auth0.com/"
clientID: "xxxx"
clientSecret: "xxxxxxx"
RedirectURL: "http://foo.bar.com/a/oidc/callback"
clientDomain : "foo.bar.com"
Scopes:
  - "openid"
  - "email"
serviceSource: "dns"
serviceName: "okta"
servicePort: 443
domain: "dev-650jsqsvuyrk4ahg.us.auth0.com"
```
在通过插件验证后会携带 `X-Authorization-ID`的标头携带令牌





