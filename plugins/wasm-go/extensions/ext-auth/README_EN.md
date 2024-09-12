---
title: External Authentication
keywords: [higress, auth]
description: The Ext authentication plugin implements the functionality to call external authorization services for authentication and authorization.
---
## Function Description
The `ext-auth` plugin implements the ability to send authorization requests to external authorization services to check whether client requests are authorized. This plugin is based on the [ext_authz filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter) of Envoy, and implements part of the capability of connecting HTTP services found in the native filter.

## Runtime Attributes
Plugin Execution Phase: `Authentication Phase`
Plugin Execution Priority: `360`

## Configuration Fields
| Name                            | Data Type | Required | Default Value | Description                                                                                                                                                     |
| ------------------------------- | --------- | -------- | ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `http_service`                  | object    | Yes      | -             | Configuration for the external authorization service                                                                                                                                                   |
| `failure_mode_allow`            | bool      | No       | false         | When set to true, client requests will still be accepted even if communication with the authorization service fails, or if the authorization service returns an HTTP 5xx error.                                   |
| `failure_mode_allow_header_add` | bool      | No       | false         | When both `failure_mode_allow` and `failure_mode_allow_header_add` are set to true, if communication with the authorization service fails or if the authorization service returns an HTTP 5xx error, the request header will add `x-envoy-auth-failure-mode-allowed: true`. |
| `status_on_error`               | int       | No       | 403           | Sets the HTTP status code returned to the client when the authorization service is unreachable or returns a status code of 5xx. The default status code is `403`.                                        |

Description of each configuration field under `http_service`
| Name                     | Data Type | Required | Default Value | Description                                  |
| ------------------------ | --------- | -------- | ------------- | -------------------------------------------- |
| `endpoint_mode`          | string    | No       | envoy         | Choose one from `envoy`, `forward_auth`.   |
| `endpoint`               | object    | Yes      | -             | HTTP service information for sending authorization requests.          |
| `timeout`                | int       | No       | 1000          | Timeout for connecting to `ext-auth` service, in milliseconds. |
| `authorization_request`  | object    | No       | -             | Configuration for sending authorization requests.                     |
| `authorization_response` | object    | No       | -             | Configuration for processing authorization responses.                  |

Description of each configuration field under `endpoint`
| Name       | Data Type | Required | Default Value | Description                                                                                      |
|------------|-----------|----------|---------------|----------------------------------------------------------------------------------------------|
| `service_name` | string | Required | -             | The full FQDN name of the authorization service, e.g., `ext-auth.dns`, `ext-auth.my-ns.svc.cluster.local`.         |
| `service_port` | int    | No       | 80            | The service port of the authorization service.                                                                       |
| `path_prefix`    | string   | Required if `endpoint_mode` is `envoy` |           | When `endpoint_mode` is `envoy`, this is the request path prefix sent by the client to the authorization service. |
| `request_method` | string   | No                                   | GET       | When `endpoint_mode` is `forward_auth`, this is the HTTP Method sent by the client to the authorization service. |
| `path`           | string   | Required if `endpoint_mode` is `forward_auth` | -      | When `endpoint_mode` is `forward_auth`, this is the request path sent by the client to the authorization service. |

Description of each configuration field under `authorization_request`
| Name                     | Data Type               | Required | Default Value | Description                                                                                                                                                                                                      |
|-------------------------|------------------------|----------|---------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `allowed_headers`       | array of StringMatcher | No       | -             | When set, client request headers that match will be added to the request to the authorization service. In addition to user-defined header matching rules, the `Authorization` HTTP header will automatically be included in the authorization service request. (If `endpoint_mode` is `forward_auth`, the original request's path will be set to `X-Original-Uri`, and the original request's HTTP Method will be set to `X-Original-Method`.) |
| `headers_to_add`        | `map[string]string`    | No       | -             | Sets the list of request headers to include in the request to the authorization service. Note that headers with the same name from the client request will be overwritten.                                          |
| `with_request_body`     | bool                   | No       | false         | Buffers the client request body and sends it in the authorization request (does not take effect for HTTP Methods GET, OPTIONS, HEAD).                                                                           |
| `max_request_body_bytes`| int                    | No       | 10MB          | Sets the maximum size of the client request body to be stored in memory. When the request body reaches the value set in this field, an HTTP 413 status code will be returned, and the authorization process will not be initiated. Note that this setting takes precedence over the configuration of `failure_mode_allow`.                                     |

Description of each configuration field under `authorization_response`
| Name                       | Data Type               | Required | Default Value | Description                                                                              |
|---------------------------|------------------------|----------|---------------|-----------------------------------------------------------------------------------------|
| `allowed_upstream_headers` | array of StringMatcher | No       | -             | When set, response headers from the authorization request that match will be added to the original client request headers. Note that headers with the same name will be overwritten.                              |
| `allowed_client_headers`   | array of StringMatcher | No       | -             | If not set, when a request is denied, all response headers from the authorization request will be added to the client response. When set, in the case of a denied request, response headers from the authorization request that match will be added to the client response. |

Configuration of the `StringMatcher` type, when using `array of StringMatcher`, will be configured in the order defined in the array
| Name       | Data Type | Required                                                         | Default Value | Description     |
|------------|-----------|------------------------------------------------------------------|---------------|------------------|
| `exact`    | string    | No, choose one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Exact match      |
| `prefix`   | string    | No, choose one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Prefix match     |
| `suffix`   | string    | No, choose one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Suffix match     |
| `contains` | string    | No, choose one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Contains         |
| `regex`    | string    | No, choose one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Regex match      |

## Configuration Example
Assuming the `ext-auth` service in Kubernetes has a serviceName of `ext-auth`, port `8090`, path of `/auth`, and namespace of `backend`, it supports two types of `endpoint_mode`:

- When `endpoint_mode` is `envoy`, the authorization request will use the original HTTP Method and the configured `path_prefix` as a prefix combined with the original request path.
- When `endpoint_mode` is `forward_auth`, the authorization request will use the configured `request_method` as the HTTP Method and the configured `path` as the request path.

### Example for endpoint_mode being envoy
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

Using the following request through the gateway, when the `ext-auth` plugin is enabled:
```shell
curl -X POST http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx"
```

**Request to `ext-auth` service successful:**
The `ext-auth` service will receive the following authorization request:
```shell
POST /auth/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 HTTP/1.1
Host: ext-auth
Authorization: xxx
Content-Length: 0
```

**Request to `ext-auth` service failed:**
When calling the `ext-auth` service gets a 5xx response, the client will receive an HTTP response code of 403 along with all response headers returned by the `ext-auth` service.

If the `ext-auth` service returns response headers like `x-auth-version: 1.0` and `x-auth-failed: true`, these will be passed to the client:
```shell
HTTP/1.1 403 Forbidden
x-auth-version: 1.0
x-auth-failed: true
date: Tue, 16 Jul 2024 00:19:41 GMT
server: istio-envoy
content-length: 0
```

When `ext-auth` is unreachable or returns a status code of 5xx, the client request will be denied with the status code configured in `status_on_error`. If the `ext-auth` service returns other HTTP status codes, the client request will be denied with the returned status code. If `allowed_client_headers` is configured, response headers with corresponding matching items will be added to the client response.

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
    service_port: 8090
    path_prefix: /auth
  timeout: 1000
```

Using the following request through the gateway, when the `ext-auth` plugin is enabled:
```shell
curl -X POST http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx"
```

The `ext-auth` service will receive the following authorization request:
```shell
POST /auth/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 HTTP/1.1
Host: ext-auth
Authorization: xxx
X-Auth-Version: 1.0
x-envoy-header: true
Content-Length: 0
```

If the response headers from the `ext-auth` service contain `x-user-id` and `x-auth-version`, these two headers will be included in the upstream request when the gateway calls upstream.

### Example for endpoint_mode being forward_auth
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

Using the following request through the gateway, when the `ext-auth` plugin is enabled:
```shell
curl -i http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx"
```

**Request to `ext-auth` service successful:**
The `ext-auth` service will receive the following authorization request:
```shell
POST /auth HTTP/1.1
Host: ext-auth
Authorization: xxx
X-Original-Uri: /users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5
X-Original-Method: GET
Content-Length: 0
```

**Request to `ext-auth` service failed:**
When calling the `ext-auth` service gets a 5xx response, the client will receive an HTTP response code of 403 along with all response headers returned by the `ext-auth` service.

If the `ext-auth` service returns response headers like `x-auth-version: 1.0` and `x-auth-failed: true`, these will be passed to the client:
```shell
HTTP/1.1 403 Forbidden
x-auth-version: 1.0
x-auth-failed: true
date: Tue, 16 Jul 2024 00:19:41 GMT
server: istio-envoy
content-length: 0
```

When `ext-auth` is unreachable or returns a status code of 5xx, the client request will be denied with the status code configured in `status_on_error`. If the `ext-auth` service returns other HTTP status codes, the client request will be denied with the returned status code. If `allowed_client_headers` is configured, response headers with corresponding matching items will be added to the client response.

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
    service_port: 8090
    path: /auth
    request_method: POST
  timeout: 1000
```

Using the following request through the gateway, when the `ext-auth` plugin is enabled:
```shell
curl -i http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx" -H "X-Auth-Version: 1.0"
```

The `ext-auth` service will receive the following authorization request:
```shell
POST /auth HTTP/1.1
Host: ext-auth
Authorization: xxx
X-Original-Uri: /users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5
X-Original-Method: GET
X-Auth-Version: 1.0
x-envoy-header: true
Content-Length: 0
```

If the response headers from the `ext-auth` service contain `x-user-id` and `x-auth-version`, these two headers will be included in the upstream request when the gateway calls upstream.

#### x-forwarded-* header
When `endpoint_mode` is `forward_auth`, higress will automatically generate and send the following headers to the authorization service.
| Header             | Description                                  |
|--------------------|------------------------------------------|
| x-forwarded-proto  | The original request scheme, e.g., http/https            |
| x-forwarded-method | The original request method, e.g., get/post/delete/patch     |
| x-forwarded-host   | The original request host                           |
| x-forwarded-uri    | The original request path, including path parameters, e.g., /v1/app?test=true |
| x-forwarded-for    | The original request client IP address                        |
