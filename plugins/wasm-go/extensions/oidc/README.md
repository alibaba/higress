# 功能说明
`oidc` 本插件实现了 OIDC 认证能力, 插件目前存在的 CSRF 攻击问题，不建议用于生产环境  

# 配置字段
| 字段                | 数据类型   | 填写要求 | 默认值        | 描述                                                               |
|-------------------|--------|------|------------|------------------------------------------------------------------|
| issuer            | string | 必填   | -          | 设置认证服务的 issuer ，即签发人。                                            |
| client_id         | string | 必填   | -          | 输入服务注册的应用 ID 。                                                   |
| client_secret     | string | 必填   | -          | 输入服务注册的应用 Secret 。                                               |
| redirect_url      | string | 必填   | -          | 输入授权成功后的重定向地址，需要与 OIDC 中配置的重定向地址保持一致。该地址的后缀需为 (oauth2/callback)。 |
| client_url        | string | 必填   | -          | 登陆成功跳转后的地址，如果未跳转成功，请检查设置的 cookiename 是否重复。                       |
| scopes            | Array  | 必填   | -          | 输入授权作用域的数组。                                                      |
| skip_expiry_check | bool   | 选填   | false      | 控制是否检测 IDToken 的过期状态。                                            |
| skip_nonce_check  | bool   | 选填   | true       | 控制是否检测 Nonce 值。                                                  |
| timeout_millis    | int    | 选填   | 500        | 设置请求与认证服务连接的超时时长。如果频繁遇到超时错误，建议增加该时长。                             |
| cookie_name       | string | 选填   | "_oidc_wasm" | 设置 cookie 的名称, 如果一个域名下多个路由设置不同的认证服务，建议设置不同名称。                    |
| cookie_domain     | string | 必填   | -          | 设置 cookie 的域名。                                                   |
| cookie_path       | string | 选填   | "/"        | 设置 cookie 的存储路径。                                                 |
| cookie_secure     | bool   | 选填   | false      | 控制 cookie 是否只在 HTTPS 下传输。                                        |
| cookie_httponly   | bool   | 选填   | true       | 控制 cookie 是否仅限于 HTTP 传输，禁止JavaScript访问。                          |
| cookie_samesite   | string | 选填   | "Lax"      | 设置 cookie 的 SameSite 属性，如："Lax", "none"。第三方跳转一般建议默认设置为Lax        |
| service_source    | string | 必填   | -          | 类型为固定 ip 或者 DNS ,输入认证 oidc 服务的注册来源。                              |
| service_name      | string | 必填   | -          | 输入认证 oidc 服务的注册名称。                                               |
| service_port      | int    | 必填   | -          | 输入认证 oidc 服务的服务端口。                                               |
| service_host      | string | 必填   | -          | 当类型为固定ip时必须填写,输入认证 oidc 服务的主机名。                                  |
| service_domain    | string | 必填   | -          | 当类型为DNS时必须填写，输入认证 oidc 服务的domain。                                |

这是一个用于OIDC认证配置的表格，确保在提供所有必要的信息时遵循上述指导。
# 配置示例
- 固定ip
```yaml
issuer: "http://127.0.0.1:9090/realms/myrealm"
redirect_url: "http://foo.bar.com/bar/oauth2/callback"
client_url: "http://foo.bar.com/"
scopes:
  - "openid"
  - "email"
cookie_name: "_oauth2_wasm_keyclocak"
cookie_domain: "foo.bar.com"
client_id: "xxxxxxxxxxxx"
client_secret: "xxxxxxxxxxxxxx"
service_host: "127.0.0.1:9090"
service_name: "keyclocak"
service_port: 80
service_source: "ip"
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
issuer: "https://dev-65874123.okta.com"
redirect_url: "http://foo.bar.com/a/oauth2/callback"
scopes:
  - "openid"
  - "email"
client_url: "http://foo.bar.com/a"
cookie_domain: "foo.bar.com"
client_id: "xxxxxxxxxxxxxxx"
client_secret: "xxxxxxx"
service_domain: "dev-65874123.okta.com"
service_name: "okta"
service_port: 443
service_source: "dns"
timeout_millis: 2000
```

在通过插件验证后会携带 `Authorization`的标头携带令牌





