# 功能说明

`ext-auth`插件实现了向授权服务发送鉴权请求，以检查客户端请求是否得到授权



# 配置字段

| 名称                            | 数据类型               | 必填 | 默认值 | 描述                                                         |
| ------------------------------- | ---------------------- | ---- | ------ | ------------------------------------------------------------ |
| `http_service`                  | object                 | 是   | -      | 外部授权服务配置                                             |
| `failure_mode_allow`            | bool                   | 否   | false  | 当设置为 true 时，即使与授权服务的通信失败，或者授权服务返回了 HTTP 5xx 错误，过滤器仍会接受客户端请求 |
| `failure_mode_allow_header_add` | bool                   | 否   | false  | 当 `failure_mode_allow` 和 `failure_mode_allow_header_add` 都设置为 true 时，若与授权服务的通信失败，或授权服务返回了 HTTP 5xx 错误，那么请求头中将会添加 `x-envoy-auth-failure-mode-allowed: true` |
| `with_request_body`             | bool                   | 否   | false  | 缓冲客户端请求体，并将其发送至鉴权请求中                     |
| `status_on_error`               | int                    | 否   | 403    | 当鉴权服务器返回错误或无法访问时，设置返回给客户端的 HTTP 状态码。默认状态码是 `403` |
| `allowed_headers`               | array of StringMatcher | 否   | -      | 当设置后，具有相应匹配项的客户端请求头将添加到鉴权服务请求中的请求头中。除了用户自定义的头部匹配规则外，鉴权服务请求中会自动包含`Host`, `Method`, `Path`, `Content-Length` 和 `Authorization`这几个关键的HTTP头 |

`http_service`中每一项的配置字段说明

| 名称                     | 数据类型 | 必填 | 默认值 | 描述                         |
| ------------------------ | -------- | ---- | ------ | ---------------------------- |
| `server_uri`             | object   | 是   | -      | 发送授权请求的 HTTP 服务 URI |
| `authorization_request`  | object   | 否   | -      | 发送授权请求配置             |
| `authorization_response` | object   | 否   | -      | 处理授权响应配置             |

`server_uri`中每一项的配置字段说明

| 名称             | 数据类型 | 必填 | 默认值 | 描述                                                     |
| ---------------- | -------- | ---- | ------ | -------------------------------------------------------- |
| `service_source` | string   | 是   | -      | 类型为固定 ip 或者 dns，输入认证 ext-auth 服务的注册来源 |
| `service_name`   | string   | 是   | -      | 输入认证 ext-auth 服务的注册名称                         |
| `service_port`   | string   | 是   | -      | 输入认证 ext-auth 服务的服务端口                         |
| `service_domain` | string   | 否   | -      | 当类型为dns时必须填写，输入认证 ext-auth 服务的domain    |
| `path`           | string   | 是   | -      | 输入认证 ext-auth 服务的请求路径                         |
| `timeout`        | int      | 否   | 200    | ext-auth 服务连接超时时间，单位毫秒                      |

`authorization_request`中每一项的配置字段说明

| 名称             | 数据类型            | 必填 | 默认值 | 描述                                                         |
| ---------------- | ------------------- | ---- | ------ | ------------------------------------------------------------ |
| `headers_to_add` | `map[string]string` | 否   | -      | 设置将包含在鉴权服务请求中的请求头列表。请注意，同名的客户端请求头将被覆盖 |

`authorization_response`中每一项的配置字段说明

| 名称                       | 数据类型               | 必填 | 默认值 | 描述                                                         |
| -------------------------- | ---------------------- | ---- | ------ | ------------------------------------------------------------ |
| `allowed_upstream_headers` | array of StringMatcher | 否   | -      | 当设置后，具有相应匹配项的鉴权请求的响应头将添加到原始的客户端请求头中。请注意，同名的请求头将被覆盖 |
| `allowed_client_headers`   | array of StringMatcher | 否   | -      | 当设置后，在请求被拒绝时，具有相应匹配项的鉴权请求的响应头将添加到客户端的响应头中 |

`StringMatcher`类型每一项的配置字段说明

| 名称       | 数据类型 | 必填                                                         | 默认值 | 描述     |
| ---------- | -------- | ------------------------------------------------------------ | ------ | -------- |
| `exact`    | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 精确匹配 |
| `prefix`   | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 前缀匹配 |
| `suffix`   | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 后缀匹配 |
| `contains` | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 是否包含 |
| `regex`    | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 正则匹配 |



# 配置示例

下面假设 `ext-auth` 服务在Kubernetes中serviceName为 `ext-auth`，端口 `8090`，路径为 `/auth`，命名空间为 `backend`

`ext-auth` 插件的配置：

```yaml
http_service:
  server_uri:
    path: /auth
    service_name: ext-auth
    namespace: backend
    service_port: 8090
    service_source: k8s
    timeout: 500
```

使用如下请求网关：

```shell
curl -i http://localhost:8082/users -X GET -H "foo: bar" -H "Authorization: xxx"
```

`ext-auth` 的服务将接收到类似如下的鉴权请求：

```
GET /auth HTTP/1.1
Host: ext-auth
Authorization: xxx
Content-Length: 0
```

