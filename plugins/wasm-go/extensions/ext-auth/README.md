# 功能说明

`ext-auth` 插件实现了向外部授权服务发送鉴权请求，以检查客户端请求是否得到授权。该插件实现时参考了Envoy原生的[ext_authz filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter)，实现了原生filter中对接HTTP服务的部分能力



# 配置字段

| 名称                            | 数据类型 | 必填 | 默认值 | 描述                                                                                                                                                         |
| ------------------------------- | -------- | ---- | ------ |------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `http_service`                  | object   | 是   | -      | 外部授权服务配置                                                                                                                                                   |
| `failure_mode_allow`            | bool     | 否   | false  | 当设置为 true 时，即使与授权服务的通信失败，或者授权服务返回了 HTTP 5xx 错误，仍会接受客户端请求                                                                                                   |
| `failure_mode_allow_header_add` | bool     | 否   | false  | 当 `failure_mode_allow` 和 `failure_mode_allow_header_add` 都设置为 true 时，若与授权服务的通信失败，或授权服务返回了 HTTP 5xx 错误，那么请求头中将会添加 `x-envoy-auth-failure-mode-allowed: true` |
| `status_on_error`               | int      | 否   | 403    | 当授权服务无法访问或状态码为 5xx 时，设置返回给客户端的 HTTP 状态码。默认状态码是 `403`                                        |

`http_service`中每一项的配置字段说明

| 名称                     | 数据类型 | 必填 | 默认值 | 描述                                  |
| ------------------------ | -------- | ---- | ------ | ------------------------------------- |
| `endpoint`               | object   | 是   | -      | 发送鉴权请求的 HTTP 服务信息          |
| `timeout`                | int      | 否   | 200    | `ext-auth` 服务连接超时时间，单位毫秒 |
| `authorization_request`  | object   | 否   | -      | 发送鉴权请求配置                      |
| `authorization_response` | object   | 否   | -      | 处理鉴权响应配置                      |

`endpoint`中每一项的配置字段说明

| 名称             | 数据类型 | 必填 | 默认值 | 描述                                                |
| ---------------- | -------- | ---- | ------ | --------------------------------------------------- |
| `service_source` | string   | 是   | -      | 类型为固定 ip 或者 dns，输入授权服务的注册来源 |
| `service_name`   | string   | 是   | -      | 输入授权服务的注册名称                      |
| `service_port`   | string   | 是   | -      | 输入授权服务的服务端口                      |
| `service_domain` | string   | 否   | -      | 当类型为dns时必须填写，输入 `ext-auth` 服务的domain |
| `request_method` | string   | 否   | GET    | 客户端向授权服务发送请求的HTTP Method        |
| `path`           | string   | 是   | -      | 输入授权服务的请求路径                       |

`authorization_request`中每一项的配置字段说明

| 名称                | 数据类型               | 必填 | 默认值 | 描述                                                         |
| ------------------- | ---------------------- | ---- | ------ | ------------------------------------------------------------ |
| `allowed_headers`   | array of StringMatcher | 否   | -      | 当设置后，具有相应匹配项的客户端请求头将添加到授权服务请求中的请求头中。除了用户自定义的头部匹配规则外，授权服务请求中会自动包含`Host`, `Method`, `Path`, `Content-Length` 和 `Authorization`这几个关键的HTTP头 |
| `headers_to_add`    | `map[string]string`    | 否   | -      | 设置将包含在授权服务请求中的请求头列表。请注意，同名的客户端请求头将被覆盖 |
| `with_request_body` | bool                   | 否   | false  | 缓冲客户端请求体，并将其发送至鉴权请求中（HTTP Method为GET、OPTIONS、HEAD请求时不生效） |

`authorization_response`中每一项的配置字段说明

| 名称                       | 数据类型               | 必填 | 默认值 | 描述                                                                              |
| -------------------------- | ---------------------- | ---- | ------ |---------------------------------------------------------------------------------|
| `allowed_upstream_headers` | array of StringMatcher | 否   | -      | 当设置后，具有相应匹配项的鉴权请求的响应头将添加到原始的客户端请求头中。请注意，同名的请求头将被覆盖                              |
| `allowed_client_headers`   | array of StringMatcher | 否   | -      | 如果不设置，在请求被拒绝时，所有的鉴权请求的响应头将添加到客户端的响应头中。当设置后，在请求被拒绝时，具有相应匹配项的鉴权请求的响应头将添加到客户端的响应头中 |

`StringMatcher`类型每一项的配置字段说明，在使用`array of StringMatcher`时会按照数组中定义的StringMatcher顺序依次进行配置

| 名称       | 数据类型 | 必填                                                         | 默认值 | 描述     |
| ---------- | -------- | ------------------------------------------------------------ | ------ | -------- |
| `exact`    | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 精确匹配 |
| `prefix`   | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 前缀匹配 |
| `suffix`   | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 后缀匹配 |
| `contains` | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 是否包含 |
| `regex`    | string   | 否，`exact` , `prefix` , `suffix`, `contains`, `regex` 中选填一项 | -      | 正则匹配 |



# 配置示例

下面假设 `ext-auth` 服务在Kubernetes中serviceName为 `ext-auth`，端口 `8090`，路径为 `/auth`，命名空间为 `backend`

## 示例1

`ext-auth` 插件的配置：

```yaml
http_service:
  endpoint:
    service_name: ext-auth
    namespace: backend
    service_port: 8090
    service_source: k8s
    path: /auth
    request_method: POST
  timeout: 500
```

使用如下请求网关，当开启 `ext-auth` 插件后：

```shell
curl -i http://localhost:8082/users -X GET -H "foo: bar" -H "Authorization: xxx"
```

**请求 `ext-auth` 服务成功：**

`ext-auth` 服务将接收到如下的鉴权请求：

```
POST /auth HTTP/1.1
Host: ext-auth
Authorization: xxx
Content-Length: 0
```

**请求 `ext-auth` 服务失败：**

当调用 `ext-auth` 服务响应为 5xx 时，客户端将接收到HTTP响应码403和 `ext-auth` 服务返回的全量响应头

假如 `ext-auth` 服务返回了 `x-auth-version: 1.0` 和 `x-auth-failed: true` 的响应头，会传递给客户端

```
HTTP/1.1 403 Forbidden
x-auth-version: 1.0
x-auth-failed: true
date: Tue, 16 Jul 2024 00:19:41 GMT
server: istio-envoy
content-length: 0
```

当 `ext-auth` 无法访问或状态码为 5xx 时，将以 `status_on_error` 配置的状态码拒绝客户端请求

当 `ext-auth` 服务返回其他 HTTP 状态码时，将以返回的状态码拒绝客户端请求。如果配置了 `allowed_client_headers`，具有相应匹配项的响应头将添加到客户端的响应中


## 示例2

`ext-auth` 插件的配置：

```yaml
http_service:
  authorization_request:
    allowed_headers:
    - exact: x-auth-version
    headers_to_add:
      x-envoy-header: true
  authorization_response:
    allowed_upstream_headers:
    - exact: x-user-id
    - exact: x-auth-version
  endpoint:
    service_name: ext-auth
    namespace: backend
    service_port: 8090
    service_source: k8s
    path: /auth
    request_method: POST
  timeout: 500
```

使用如下请求网关，当开启 `ext-auth` 插件后：

```shell
curl -i http://localhost:8082/users -X GET -H "foo: bar" -H "Authorization: xxx" -H "X-Auth-Version: 1.0"
```

`ext-auth` 服务将接收到如下的鉴权请求：

```
POST /auth HTTP/1.1
Host: ext-auth
Authorization: xxx
X-Auth-Version: 1.0
x-envoy-header: true
Content-Length: 0
```

`ext-auth` 服务返回响应头中如果包含 `x-user-id` 和 `x-auth-version`，网关调用upstream时的请求中会带上这两个请求头
