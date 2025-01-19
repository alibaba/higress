---
title: External Authentication
keywords: [higress, auth]
description: The Ext Authentication plugin implements the capability to call external authorization services for authentication and authorization.
---

## Feature Description

The `ext-auth` plugin sends an authorization request to an external authorization service to check if the client request is authorized. When implementing this plugin, it refers to the native [ext_authz filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter) of Envoy, and realizes part of the capabilities of the native filter to connect to an HTTP service.

## Operating Attributes

Plugin Execution Phase: `Authentication Phase`
Plugin Execution Priority: `360`


## Configuration Fields

| Name                            | Data Type         | Required | Default Value | Description                                                         |
| ------------------------------- | ---------------- | ---- | ------ | ------------------------------------------------------------ |
| `http_service`                  | object           | Yes   | -      | Configuration of the external authorization service                                             |
| `skipped_path_prefixes`         | array of strings | No   | -      | Used to exclude specific paths from authorization                                     |
| `failure_mode_allow`            | bool             | No   | false  | When set to true, the client request will be accepted even if the communication with the authorization service fails or the authorization service returns an HTTP 5xx error |
| `failure_mode_allow_header_add` | bool             | No   | false  | When both `failure_mode_allow` and `failure_mode_allow_header_add` are set to true, if the communication with the authorization service fails or the authorization service returns an HTTP 5xx error, the request header will add `x-envoy-auth-failure-mode-allowed: true` |
| `status_on_error`               | int              | No   | 403    | When the authorization service is inaccessible or the status code is 5xx, set the HTTP status code returned to the client. The default status code is `403` |

Configuration field description for each item in `http_service`

| Name                     | Data Type | Required | Default Value | Description                                  |
| ------------------------ | -------- | ---- | ------ | ------------------------------------- |
| `endpoint_mode`          | string   | No   | envoy  | Optional `envoy` or `forward_auth`        |
| `endpoint`               | object   | Yes   | -      | HTTP service information for sending authorization requests          |
| `timeout`                | int      | No   | 1000   | Connection timeout for the `ext-auth` service, in milliseconds |
| `authorization_request`  | object   | No   | -      | Configuration for sending authorization requests                      |
| `authorization_response` | object   | No   | -      | Configuration for processing authorization responses                      |

Configuration field description for each item in `endpoint`

| Name             | Data Type | Required                                     | Default Value | Description                                                         |
| ---------------- | -------- | ---------------------------------------- | ------ | ------------------------------------------------------------ |
| `service_name`   | string   | Yes                                       | -      | Enter the authorization service name, the full FQDN name with the service type, such as `ext-auth.dns`, `ext-auth.my-ns.svc.cluster.local` |
| `service_port`   | int      | No                                       | 80     | Enter the service port of the authorization service                                       |
| `service_host`   | string   | No                                       | -      | The Host header set when requesting the authorization service. If not filled, it is consistent with the FQDN         |
| `path_prefix`    | string   | Required when `endpoint_mode` is `envoy`        | -      | When `endpoint_mode` is `envoy`, the request path prefix for the client to send requests to the authorization service |
| `request_method` | string   | No                                       | GET    | When `endpoint_mode` is `forward_auth`, the HTTP Method for the client to send requests to the authorization service |
| `path`           | string   | Required when `endpoint_mode` is `forward_auth` | -      | When `endpoint_mode` is `forward_auth`, the request path for the client to send requests to the authorization service |

Configuration field description for each item in `authorization_request`

| Name                     | Data Type               | Required | Default Value | Description                                                         |
| ------------------------ | ---------------------- | ---- | ------ | ------------------------------------------------------------ |
| `allowed_headers`        | array of StringMatcher | No   | -      | After setting, the client request headers of the matching items will be added to the request headers in the authorization service request. In addition to the user - defined header matching rules, the `Authorization` HTTP header will be automatically included in the authorization service request (when `endpoint_mode` is `forward_auth`, the `X - Forwarded - *` request headers will be added) |
| `headers_to_add`         | map[string]string      | No   | -      | Set the list of request headers to be included in the authorization service request. Note that client request headers with the same name will be overwritten |
| `with_request_body`      | bool                   | No   | false  | Buffer the client request body and send it to the authorization request (not effective for HTTP Method GET, OPTIONS, HEAD requests) |
| `max_request_body_bytes` | int                    | No   | 10MB   | Set the maximum size of the client request body to be saved in memory. When the client request body reaches the value set in this field, an HTTP 413 status code will be returned, and the authorization process will not be started. Note that this setting takes precedence over the `failure_mode_allow` configuration |

Configuration field description for each item in `authorization_response`

| Name                       | Data Type               | Required | Default Value | Description                                                         |
| -------------------------- | ---------------------- | ---- | ------ | ------------------------------------------------------------ |
| `allowed_upstream_headers` | array of StringMatcher | No   | -      | The response headers of the authorization request of the matching items will be added to the original client request headers. Note that request headers with the same name will be overwritten |
| `allowed_client_headers`   | array of StringMatcher | No   | -      | If not set, when the request is rejected, all the response headers of the authorization request will be added to the client response headers. When set, when the request is rejected, the response headers of the authorization request of the matching items will be added to the client response headers |

Configuration field description for each item of the `StringMatcher` type. When using `array of StringMatcher`, the configuration will be carried out in the order of the `StringMatcher` defined in the array

| Name       | Data Type | Required                                                         | Default Value | Description     |
| ---------- | -------- | ------------------------------------------------------------ | ------ | -------- |
| `exact`    | string   | No, choose one of `exact`, `prefix`, `suffix`, `contains`, `regex` | -      | Exact match |
| `prefix`   | string   | No, choose one of `exact`, `prefix`, `suffix`, `contains`, `regex` | -      | Prefix match |
| `suffix`   | string   | No, choose one of `exact`, `prefix`, `suffix`, `contains`, `regex` | -      | Suffix match |
| `contains` | string   | No, choose one of `exact`, `prefix`, `suffix`, `contains`, `regex` | -      | Contains |
| `regex`    | string   | No, choose one of `exact`, `prefix`, `suffix`, `contains`, `regex` | -      | Regex match |

### Differences between the Two `endpoint_mode`

When `endpoint_mode` is `envoy`, the authorization request will use the HTTP Method of the original request and the configured `path_prefix` as the request path prefix, and concatenate it with the original request path.

When `endpoint_mode` is `forward_auth`, the authorization request will use the configured `request_method` as the HTTP Method and the configured `path` as the request path. And higress will automatically generate and send the following headers to the authorization service:

| Header               | Description                                                   |
| -------------------- | ------------------------------------------------------ |
| `x-forwarded-proto`  | The scheme of the original request, such as http/https                      |
| `x-forwarded-method` | The method of the original request, such as get/post/delete/patch             |
| `x-forwarded-host`   | The host of the original request                                         |
| `x-forwarded-uri`    | The path of the original request, including path parameters, such as `/v1/app?test=true` |

## Configuration Examples

Assume that in Kubernetes, the `ext-auth` service has a `serviceName` of `ext-auth`, a port of `8090`, a path of `/auth`, and is in the `backend` namespace.

### When endpoint_mode is envoy

#### Example 1

Configuration of the `ext-auth` plugin:

```yaml
http_service:
  endpoint_mode: envoy
  endpoint:
    service_name: ext-auth.backend.svc.cluster.local
    service_port: 8090
    path_prefix: /auth
  timeout: 1000
```

When using the following request to the gateway after enabling the `ext-auth` plugin:

```shell
curl -X POST http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx"
```

**When the request to the `ext-auth` service is successful**:

The `ext-auth` service will receive the following authorization request:

```
POST /auth/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 HTTP/1.1
Host: ext-auth.backend.svc.cluster.local
Authorization: xxx
Content-Length: 0
```

**When the request to the `ext-auth` service fails**:

When the response from the `ext-auth` service is 5xx, the client will receive an HTTP response code of 403 and all the response headers returned by the `ext-auth` service.

If the `ext-auth` service returns response headers of `x-auth-version: 1.0` and `x-auth-failed: true`, they will be passed to the client.

```
HTTP/1.1 403 Forbidden
x-auth-version: 1.0
x-auth-failed: true
date: Tue, 16 Jul 2024 00:19:41 GMT
server: istio-envoy
content-length: 0
```

When the `ext-auth` service is inaccessible or the status code is 5xx, the client request will be rejected with the status code configured in `status_on_error`.

When the `ext-auth` service returns other HTTP status codes, the client request will be rejected with the returned status code. If `allowed_client_headers` is configured, the response headers with corresponding matching items will be added to the client's response.

#### Example 2

Configuration of the `ext-auth` plugin:

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
  endpoint_mode: envoy
  endpoint:
    service_name: ext-auth.backend.svc.cluster.local
    service_host: my-domain.local
    service_port: 8090
    path_prefix: /auth
  timeout: 1000
```

When using the following request to the gateway after enabling the `ext-auth` plugin:

```shell
curl -X POST http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx"
```

The `ext-auth` service will receive the following authorization request:

```
POST /auth/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 HTTP/1.1
Host: my-domain.local
Authorization: xxx
X-Auth-Version: 1.0
x-envoy-header: true
Content-Length: 0
```

If the response headers returned by the `ext-auth` service contain `x-user-id` and `x-auth-version`, these two headers will be included in the request when the gateway calls the upstream.

### When endpoint_mode is forward_auth

#### Example 1

Configuration of the `ext-auth` plugin:

```yaml
http_service:
  endpoint_mode: forward_auth
  endpoint:
    service_name: ext-auth.backend.svc.cluster.local
    service_port: 8090
    path: /auth
    request_method: POST
  timeout: 1000
```

When using the following request to the gateway after enabling the `ext-auth` plugin:

```shell
curl -i http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx" -H "Host: foo.bar.com"
```

**When the request to the `ext-auth` service is successful**:

The `ext-auth` service will receive the following authorization request:

```
POST /auth HTTP/1.1
Host: ext-auth.backend.svc.cluster.local
Authorization: xxx
X-Forwarded-Proto: HTTP
X-Forwarded-Host: foo.bar.com
X-Forwarded-Uri: /users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5
X-Forwarded-Method: GET
Content-Length: 0
```

**When the request to the `ext-auth` service fails**:

When the response from the `ext-auth` service is 5xx, the client will receive an HTTP response code of 403 and all the response headers returned by the `ext-auth` service.

If the `ext-auth` service returns response headers of `x-auth-version: 1.0` and `x-auth-failed: true`, they will be passed to the client.

```
HTTP/1.1 403 Forbidden
x-auth-version: 1.0
x-auth-failed: true
date: Tue, 16 Jul 2024 00:19:41 GMT
server: istio-envoy
content-length: 0
```

When the `ext-auth` service is inaccessible or the status code is 5xx, the client request will be rejected with the status code configured in `status_on_error`.

When the `ext-auth` service returns other HTTP status codes, the client request will be rejected with the returned status code. If `allowed_client_headers` is configured, the response headers with corresponding matching items will be added to the client's response.

#### Example 2

Configuration of the `ext-auth` plugin:

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
  endpoint_mode: forward_auth
  endpoint:
    service_name: ext-auth.backend.svc.cluster.local
    service_host: my-domain.local
    service_port: 8090
    path: /auth
    request_method: POST
  timeout: 1000
```

When using the following request to the gateway after enabling the `ext-auth` plugin:

```shell
curl -i http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx" -H "X-Auth-Version: 1.0" -H "Host: foo.bar.com"
```

The `ext-auth` service will receive the following authorization request:

```
POST /auth HTTP/1.1
Host: my-domain.local
Authorization: xxx
X-Forwarded-Proto: HTTP
X-Forwarded-Host: foo.bar.com
X-Forwarded-Uri: /users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5
X-Forwarded-Method: GET
X-Auth-Version: 1.0
x-envoy-header: true
Content-Length: 0
```

If the response headers returned by the `ext-auth` service contain `x-user-id` and `x-auth-version`, these two headers will be included in the request when the gateway calls the upstream.