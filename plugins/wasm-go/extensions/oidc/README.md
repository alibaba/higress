# OIDC Wasm 插件

### 简介

本仓库提供了一个高度可集成的Wasm插件支持OpenID Connect（OIDC）身份认证，同时强化了对跨站请求伪造（CSRF）攻击的防御能力，支持OpenID Connect协议中的Logout Endpoint。在通过Wasm插件OIDC验证后的请求会携带 `Authorization`的标头对应ID Token。

### OIDC 流程

#### 流程图

<p align="center">
  <img src="https://gw.alicdn.com/imgextra/i4/O1CN01McjXP51EWT9dMxQm4_!!6000000000359-2-tps-1803-2018.png" alt="oidc_process" style="zoom: 33%;" />
</p>


#### OIDC 流程解析

1. 模拟用户访问对应服务api

   ```shell
   curl --url "foo.bar.com/foo"
   ```

2. Higress重定向到OIDC Provider登录页同时携带client_id、response_type、scope等OIDC认证的参数并设置csrf cookie防御CSRF攻击

   ```shell
   curl --url "https://dev-o43xb1mz7ya7ach4.us.auth0.com/authorize"\
     --data "approval_prompt=force" \
     --data "client_id=YagFqRD9tfNIaac5BamjhsSatjrAnsnZ" \
     --data "redirect_uri=http%3A%2F%2Ffoo.bar.com%2Foauth2%2Fcallback" \
     --data "response_type=code" \
     --data "scope=openid+email+offline_access" \
     --data "state=nT06xdCqn4IqemzBRV5hmO73U_hCjskrH_VupPqdcdw%3A%2Ffoo" \
    	--header "Set-Cookie: _oauth2_proxy_csrf=LPruATEDgcdmelr8zScD_ObhsbP4zSzvcgmPlcNDcJpFJ0OvhxP2hFotsU-kZnYxd5KsIjzeIXGTOjf8TKcbTHbDIt-aQoZORXI_0id3qeY0Jt78223DPeJ1xBqa8VO0UiEOUFOR53FGxirJOdKFxaAvxDFb1Ok=|1718962455|V1QGWyjQ4hMNOQ4Jtf17HeQJdVqHdt5d65uraFduMIU=; Path=/; Expires=Fri, 21 Jun 2024 08:06:20 GMT; HttpOnly"
   ```

3. 用户在登录页进行登录

![keycloak_login](https://gw.alicdn.com/imgextra/i4/O1CN01HLcl7r1boXwwnzGqA_!!6000000003512-0-tps-3840-2160.jpg)

4. 携带授权重定向到Higress并携带了state参数用于验证CSRF Cookie，授权code用于交换Token

   ```shell
   curl --url "http://foo.bar.com/oauth2/callback" \
     --data "state=nT06xdCqn4IqemzBRV5hmO73U_hCjskrH_VupPqdcdw%3A%2Ffoo" \
   	--data "code=0bdopoS2c2lx95u7iO0OH9kY1TvaEdJHo4lB6CT2_qVFm"
   ```

5. 利用授权交换id_token和access_token

   ```shell
   curl -X POST \
     --url "https://dev-o43xb1mz7ya7ach4.us.auth0.com/oauth/token" \
     --data "grant_type=authorization_code" \
     --data "client_id=YagFqRD9tfNIaac5BamjhsSatjrAnsnZ" \
     --data "client_secret=ekqv5XoZuMFtYms1NszEqRx03qct6BPvGeJUeptNG4y09PrY16BKT9IWezTrrhJJ" \
     --data "redirect_uri=http%3A%2F%2Ffoo.bar.com%2Foauth2%2Fcallback" \
     --data "code=0bdopoS2c2lx95u7iO0OH9kY1TvaEdJHo4lB6CT2_qVFm" \
   ```

   返回的请求里包含了id_token, access_token，refresh_token用于后续刷新access_token

   ```json
   {
       "access_token": "eyJhbGciOiJkaXIiLCJlbmMiOiJBMjU2R0NNIiwiaXNzIjoiaHR0cHM6Ly9kZXYtbzQzeGIxbXo3eWE3YWNoNC51cy5hdXRoMC5jb20vIn0..WP_WRVM-y3fM1sN4.fAQqtKoKZNG9Wj0OhtrMgtsjTJ2J72M2klDRd9SvUKGbiYsZNPmIl_qJUf81D3VIjD59o9xrOOJIzXTgsfFVA2x15g-jBlNh68N7dyhXu9237Tbplweu1jA25IZDSnjitQ3pbf7xJVIfPnWcrzl6uT8G1EP-omFcl6AQprV2FoKFMCGFCgeafuttppKe1a8mpJDj7AFLPs-344tT9mvCWmI4DuoLFh0PiqMMJBByoijRSxcSdXLPxZng84j8JVF7H6mFa-dj-icP-KLy6yvzEaRKz_uwBzQCzgYK434LIpqw_PRuN3ClEsenwRgIsNdVjvKcoAysfoZhmRy9BQaE0I7qTohSBFNX6A.mgGGeeWgugfXcUcsX4T5dQ",
       "refresh_token": "GrZ1f2JvzjAZQzSXmyr1ScWbv8aMFBvzAXHBUSiILcDEG",
       "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Imc1Z1ExSF9ZbTY0WUlvVkQwSVpXTCJ9.eyJlbWFpbCI6IjE2MDExNTYyNjhAcXEuY29tIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJpc3MiOiJodHRwczovL2Rldi1vNDN4YjFtejd5YTdhY2g0LnVzLmF1dGgwLmNvbS8iLCJhdWQiOiJZYWdGcVJEOXRmTklhYWM1QmFtamhzU2F0anJBbnNuWiIsImlhdCI6MTcxOTE5ODYzOCwiZXhwIjoxNzE5MjM0NjM4LCJzdWIiOiJhdXRoMHw2NjVkNzFlNzRjMTMxMTc3YmU2NmU2MDciLCJzaWQiOiJjdDJVOF9ZUS16VDdFOGkwRTNNeUstejc5ZGlWUWhhVSJ9.gfzXKJ0FeqzYqOUDLQHWcUG19IOLqkpLN09xTmIat0umrlGV5VNSumgWH3XJmmwnhdb8AThH3Jf-7kbRJzu4rM-BbGbFTRBTzNHeUajFOFrIgld5VENQ_M_sXHkTp0psWKSr9vF24kmilCfSbvC5lBKjt878ljZ7-xteWuaUYOMUdcJb4DSv0-zjX01sonJxYamTlhji3M4TAW7VwhwqyZt8dBhVSNaRw1wUKj-M1JrBDLyx65sroZtSqVA0udIrqMHEbWYb2de7JjzlqG003HRMzwOm7OXgEd5ZVFqgmBLosgixOU5DJ4A26nlqK92Sp6VqDMRvA-3ym8W_m-wJ_A",
       "scope": "openid email offline_access",
       "expires_in": 86400,
       "token_type": "Bearer"
   }
   ```

6. 将获得的id_token和access_token加密存储在Cookie _oauth2_proxy中，用于后续用户登录状态的验证，同时清除Cookie _oauth2_proxy_csrf

   ```json
   "Set-Cookie": [
       "_oauth2_proxy_csrf=; Path=/; Expires=Mon, 24 Jun 2024 02:17:39 GMT; HttpOnly",
       "_oauth2_proxy=8zM_Pcfpp_gesKFe4SMg08o5Iv0A8WAOQOmG1-vZBbQ56UggYVC0Cu-gFMEoxJZU5q1O5vqRlVBizlLetgVjRCksGVbttwl8tQ7h5YiyIubbbtvF1T4JzLh3QfzUUrwbB-VznOkh8qLbjAhddocecjBt4rMiDyceKXqMr4eO5TUEMx4vHtJYnTYalMeTYhGXk5MNSyrdZX9NnQnkdrCjiOQM13ggwob2nYwhGWaAlgzFSWkgkdtBy2Cl_YMWZ8_gKk9rDX289-JrJyGpr5k9O9RzRhZoY2iE3Mcr8-Q37RTji1Ga22QO-XkAcSaGqY1Qo7jLdmgZTYKC5JvtdLc4rj3vcbveYxU7R3Pt2vEribQjKTh4Sqb0aA03p4cxXyZN4SUfBW1NAOm4JLPUhKJy8frqC9_E0nVqPvpvnacaoQs8WkX2zp75xHoMa3SD6KZhQ5JUiPEiNkOaUsyafLvht6lLkNDhgzW3BP2czoe0DCDBLnsot0jH-qQpMZYkaGr-ZnRKI1OPl1vHls3mao5juOAW1VB2A9aughgc8SJ55IFZpMfFMdHdTDdMqPODkItX2PK44GX-pHeLxkOqrzp3GHtMInpL5QIQlTuux3erm3CG-ntlUE7JBtN2T9LEb8XfIFu58X9_vzMun4JQlje2Thi9_taI_z1DSaTtvNNb54wJfSPwYCCl4OsH-BacVmPQhH6TTZ6gP2Qsm5TR2o1U2D9fuVkSM-OPCG9l3tILambIQwC3vofMW6X8SIFSmhJUDvN7NbwxowBiZ6Y7GJRZlAk_GKDkpsdrdIvC67QqczZFphRVnm6qi-gPO41APCbcO6fgTwyOhbP3RrZZKWSIqWJYhNE3_Sfkf0565H7sC7Hc8XUUjJvP3WnjKS9x7KwzWa-dsUjV3-Q-VNl-rXTguVNAIirYK-qrMNMZGCRcJqcLnUF0V_J2lVmFyVsSlE3t0sDw2xmbkOwDptXFOjQL5Rb4esUMYdCBWFajBfvUtcZEFtYhD0kb6VcbjXO3NCVW5qKh_l9C9SRCc7TG1vcRAqUQlRXHacTGWfcWsuQkCJ3Mp_oWaDxs1GRDykQYxAn5sTICovThWEU2C6o75grWaNrkj5NU-0eHh3ryvxLmGLBOXZV9OQhtKShWmUgywSWMxOHOuZAqdAPULc8KheuGFjXYp-RnCbFYWePJmwzfQw89kSkj1KUZgMYwKEjSz62z2qc9KLczomv76ortQzvo4Hv9kaW6xVuQj5R5Oq6_WMBOqsmUMzcXpxCIOGjcdcZRBc0Fm09Uy9oV1PRqvAE4PGtfyrCaoqILBix8UIww63B07YGwzQ-hAXDysBK-Vca2x7GmGdXsNXXcTgu00bdsjtHZPDBBWGfL3g_rMAXr2vWyvK4CwNjcaPAmrlF3geHPwbIePT0hskBboX1v1bsuhzsai7rGM4r53pnb1ZEoTQDa1B-HyokFgo14XiwME0zE1ifpNzefjpkz1YY2krJlqfCydNwoKaTit4tD2yHlnxAeFF9iIrxzSKErNUFpmyLa7ge7V33vhEH-6k5oBTLE2Q2BrC6aAkLCcPwU9xv_SzBDQPRY0MEYv3kGF03Swo1crRbGh-aifYX9NiHDsmG6r1vAnx0MAOw2Jzuz2x6SSdfBrzlcoWBlrwiZzd9kAKq75n1Uy9uzZ8SRnkBrEZySHBwEbu196VklkRE0jqwC-e3wWNNuviSOfwkVeX-7QdOoO10yw9VK2sW52lFvIEf4chv_ta7bGfAZOWBjpktG6ZLD81SE6A88zpqG2SysSyNMp9hl-umG-5sFsjCn_c9E8bDvwkUOUVb9bNqhBDsZgR0BNPawiOZjmyfhzmwmWf-zgFzfFSV6BvOwNRi3sCOHTsWcuk9NBQ_YK8CpNkVl3WeIBSDfidimuC_QV9UWKs1GPk35ZRkM4zKtLY2JsBFWKaDy_P80TcOzcMBoP8gIBClXZ-WUqfE8s1yyc4jrq-qL1_wJ24ef1O9FktsbyZiDKXw2vnqsT8-g_hCeG-unrT1ZFscf8oNdqczARHX-K4vKH2k3uIqEx1M=|1719199056|2rsgdUIClHNEpxBLlHOVRYup6e4oKensQfljtmn4B80=; Path=/; Expires=Mon, 01 Jul 2024 03:17:36 GMT; HttpOnly"
   ]
   ```

7. 携带 Authorization的标头对应id_token访问对应api

   ```shell
   curl --url "foo.bar.com/foo"
     --data "Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Imc1Z1ExSF9ZbTY0WUlvVkQwSVpXTCJ9.eyJlbWFpbCI6IjE2MDExNTYyNjhAcXEuY29tIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJpc3MiOiJodHRwczovL2Rldi1vNDN4YjFtejd5YTdhY2g0LnVzLmF1dGgwLmNvbS8iLCJhdWQiOiJZYWdGcVJEOXRmTklhYWM1QmFtamhzU2F0anJBbnNuWiIsImlhdCI6MTcxOTE5ODYzOCwiZXhwIjoxNzE5MjM0NjM4LCJzdWIiOiJhdXRoMHw2NjVkNzFlNzRjMTMxMTc3YmU2NmU2MDciLCJzaWQiOiJjdDJVOF9ZUS16VDdFOGkwRTNNeUstejc5ZGlWUWhhVSJ9.gfzXKJ0FeqzYqOUDLQHWcUG19IOLqkpLN09xTmIat0umrlGV5VNSumgWH3XJmmwnhdb8AThH3Jf-7kbRJzu4rM-BbGbFTRBTzNHeUajFOFrIgld5VENQ_M_sXHkTp0psWKSr9vF24kmilCfSbvC5lBKjt878ljZ7-xteWuaUYOMUdcJb4DSv0-zjX01sonJxYamTlhji3M4TAW7VwhwqyZt8dBhVSNaRw1wUKj-M1JrBDLyx65sroZtSqVA0udIrqMHEbWYb2de7JjzlqG003HRMzwOm7OXgEd5ZVFqgmBLosgixOU5DJ4A26nlqK92Sp6VqDMRvA-3ym8W_m-wJ_A"
   ```

8. 后端服务根据id_token获取用户信息并返回对应的Http响应

   ```json
   {
       "email": "******",
       "email_verified": false,
       "iss": "https://dev-o43xb1mz7ya7ach4.us.auth0.com/",
       "aud": "YagFqRD9tfNIaac5BamjhsSatjrAnsnZ",
       "iat": 1719198638,
       "exp": 1719234638,
       "sub": "auth0|665d71e74c131177be66e607",
       "sid": "ct2U8_YQ-zT7E8i0E3MyK-z79diVQhaU"
   }
   ```

### 配置说明

| Option                        | Type         | Description                                                  | Default           |
| ----------------------------- | ------------ | ------------------------------------------------------------ | ----------------- |
| cookie_name                   | string       | the name of the cookie that the oauth_proxy creates. Should be changed to use a [cookie prefix](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#cookie_prefixes) (`__Host-` or `__Secure-`) if `--cookie-secure` is set. | `"_oauth2_proxy"` |
| cookie_secret                 | string       | the seed string for secure cookies (optionally base64 encoded) |                   |
| cookie_domains                | string\|list | Optional cookie domains to force cookies to (e.g. `.yourcompany.com`). The longest domain matching the request's host will be used (or the shortest cookie domain if there is no match). |                   |
| cookie_path                   | string       | an optional cookie path to force cookies to (e.g. `/poc/`)   | `"/"`             |
| cookie_expire                 | duration     | expire timeframe for cookie. If set to 0, cookie becomes a session-cookie which will expire when the browser is closed. | 168h0m0s          |
| cookie_refresh                | duration     | refresh the cookie after this duration; `0` to disable       |                   |
| cookie_secure                 | bool         | set [secure (HTTPS only) cookie flag](https://owasp.org/www-community/controls/SecureFlag) | true              |
| cookie_httponly               | bool         | set HttpOnly cookie flag                                     | true              |
| cookie_samesite               | string       | set SameSite cookie attribute (`"lax"`, `"strict"`, `"none"`, or `""`). | `""`              |
| cookie_csrf_per_request       | bool         | Enable having different CSRF cookies per request, making it possible to have parallel requests. | false             |
| cookie_csrf_expire            | duration     | expire timeframe for CSRF cookie                             | 15m               |
| client_id                     | string       | the OAuth Client ID                                          |                   |
| client_secret                 | string       | the OAuth Client Secret                                      |                   |
| provider                      | string       | OAuth provider                                               | oidc              |
| pass_authorization_header     | bool         | pass OIDC IDToken to upstream via Authorization Bearer header | true              |
| oidc_issuer_url               | string       | the OpenID Connect issuer URL, e.g. `"https://dev-o43xb1mz7ya7ach4.us.auth0.com"` |                   |
| oidc_verifier_request_timeout | uint32       | OIDC verifier discovery request timeout                      | 500(ms)           |
| redeem_timeout                | uint32       | OIDC redeem token request timeout                            | 500(ms)           |
| scope                         | string       | OAuth scope specification                                    |                   |
| redirect_url                  | string       | the OAuth Redirect URL, e.g. `"https://internalapp.yourcompany.com/oauth2/callback"` |                   |
| service_name                  | string       | registered name of the OIDC service, e.g. `auth.dns`, `keycloak.static` |                   |
| service_port                  | int64        | service port of the OIDC service                             |                   |
| service_host                  | string       | host of the OIDC service when type is static ip              |                   |

### 生成 Cookie Secret

``` python
python -c 'import os,base64; print(base64.urlsafe_b64encode(os.urandom(32)).decode())'
```

### 注销用户

为了注销用户，需将他们重定向到`/oauth2/sign_out`这个端点。这个端点仅移除oauth2-proxy自己设置的cookie，也就是说，用户仍然在认证提供商那里保持登录状态，并且在再次访问应用时可能会自动重新登录。之后，您还需要使用`rd`查询参数将用户重定向到认证提供商的注销页面，即重定向用户到类似如下地址（注意URL编码！）：

```
/oauth2/sign_out?rd=https%3A%2F%2Fmy-oidc-provider.example.com%2Fsign_out_page
```

或者，可以在`X-Auth-Request-Redirect`头部中包含重定向URL：

```
GET /oauth2/sign_out HTTP/1.1
X-Auth-Request-Redirect: https://my-oidc-provider/sign_out_page
...
```

（如果您的OIDC提供商支持会话管理和发现，那么"sign_out_page"应该是从[metadata](https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderConfig)中获取的`end_session_endpoint`

**注**：想要重定向到的域名（例如示例中的`my-oidc-provider.example.com`）必须添加到`whitelist_domains`配置选项中，否则重定向将会被忽略。确保包含实际的域名和端口，而不是完整的URL（例如，应使用"localhost:8081"而非"[http://localhost:8081](http://localhost:8081/)")。

### OIDC with auth0

#### 配置 auth0 账户

- 登录到开发人员 Okta 网站 [Developer Auth0 site](https://auth0.com/)
- 注册测试 web 应用程序

#### Higress 配置服务来源

* 在Higress服务来源中创建auth0 DNS来源

![auth0 service](https://gw.alicdn.com/imgextra/i1/O1CN01p9y0jF1tfzdXTzNYm_!!6000000005930-0-tps-3362-670.jpg)

#### Auth0 Ingress 配置 

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: auth0-ingress
  annotations:
    higress.io/destination: auth.dns
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

#### Wasm 插件配置

```yaml
redirect_url: 'http://foo.bar.com/oauth2/callback'
oidc_issuer_url: 'https://dev-o43xb1mz7ya7ach4.us.auth0.com/'
client_id: 'YagFqRD9tfNIaac5BamjhsSatjrAnsnZ'
client_secret: 'ekqv5XoZuMFtYms1NszEqRx03qct6BPvGeJUeptNG4y09PrY16BKT9IWezTrrhJJ'
scope: 'openid email offline_access'
cookie_secret: 'nqavJrGvRmQxWwGNptLdyUVKcBNZ2b18Guc1n_8DCfY='
cookie_secure: false
service_name: 'auth.dns'
service_port: 443
oidc_verifier_request_timeout: 2000
whitelist_domains:
    - 'dev-o43xb1mz7ya7ach4.us.auth0.com'
```

**注**：必须先配置服务来源及oidc provider ingress，wasm插件在初始化时需要访问配置的服务获取openid-configuration

#### 访问服务页面，未登陆的话进行跳转

![auth0_login](https://gw.alicdn.com/imgextra/i3/O1CN01hVNk0C1gkUWLwuC0N_!!6000000004180-0-tps-3840-2160.jpg)

#### 登陆成功跳转到服务页面

![auth0 result](https://gw.alicdn.com/imgextra/i1/O1CN01AtZ2cd1JlBxsgyCjG_!!6000000001068-0-tps-3840-2160.jpg)

#### 访问登出跳转到登出页面

```
http://foo.bar.com/oauth2/sign_out?rd=https%3A%2F%2Fdev-o43xb1mz7ya7ach4.us.auth0.com%2Foidc%2Flogout
```

![auth0 result](https://gw.alicdn.com/imgextra/i4/O1CN01zOrOm828du4C4o97U_!!6000000007956-0-tps-3840-2160.jpg)

### OIDC with keycloak

#### Get started with keycloak on docker

<https://www.keycloak.org/getting-started/getting-started-docker> 

#### Higress 配置服务来源

* 在Higress服务来源中创建Keycloak固定地址服务

![keycloak service](https://gw.alicdn.com/imgextra/i1/O1CN01p9y0jF1tfzdXTzNYm_!!6000000005930-0-tps-3362-670.jpg)

#### Keycloak Ingress 配置 

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: keycloak-ingress
  annotations:
    higress.io/destination: keycloak.static
    higress.io/backend-protocol: "HTTP"
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

#### Wasm 插件配置

```yaml
redirect_url: 'http://foo.bar.com/oauth2/callback'
oidc_issuer_url: 'http://127.0.0.1:9090/realms/myrealm'
client_id: 'myclient'
client_secret: 'PgSvuK7M4GYMPUaO0naQteKhpWYDpBYE'
scope: 'openid email'
cookie_secret: 'nqavJrGvRmQxWwGNptLdyUVKcBNZ2b18Guc1n_8DCfY='
cookie_secure: false
service_name: 'keycloak.static'
service_port: 80
service_host: '127.0.0.1:9090'
oidc_verifier_request_timeout: 2000
whitelist_domains:
    - '127.0.0.1:9090'
```

#### 访问服务页面，未登陆的话进行跳转

![keycloak_login](https://gw.alicdn.com/imgextra/i4/O1CN01HLcl7r1boXwwnzGqA_!!6000000003512-0-tps-3840-2160.jpg)

#### 登陆成功跳转到服务页面

![keycloak service](https://gw.alicdn.com/imgextra/i1/O1CN01AtZ2cd1JlBxsgyCjG_!!6000000001068-0-tps-3840-2160.jpg)

#### 访问登出跳转到登出页面

```
http://foo.bar.com/oauth2/sign_out?rd=http%3A%2F%2F127.0.0.1:9090%2Frealms%2Fmyrealm%2Fprotocol%2Fopenid-connect%2Flogout
```

![auth0 result](https://gw.alicdn.com/imgextra/i4/O1CN01JzIAvo1CqxdypfnOH_!!6000000000133-0-tps-3840-2160.jpg)