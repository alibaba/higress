---
title: External Authentication
keywords: [higress, auth]
description: The Ext Authentication plugin implements the capability to call external authorization services for authentication and authorization.
---
## Function Description
The `ext-auth` plugin implements sending authentication requests to an external authorization service to check whether the client request is authorized. This plugin is implemented with reference to Envoy's native [ext_authz filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter), which covers some capabilities for connecting to HTTP services.

## Execution Properties
Plugin Execution Phase: `Authentication Phase`  
Plugin Execution Priority: `360`

## Configuration Fields
| Name                            | Data Type | Required | Default Value | Description                                                                                                                                                            |
| ------------------------------- | --------- | -------- | ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `http_service`                  | object    | Yes      | -             | Configuration for the external authorization service                                                                                                                   |
| `failure_mode_allow`            | bool      | No       | false         | When set to true, client requests will still be accepted even if communication with the authorization service fails or the authorization service returns an HTTP 5xx error |
| `failure_mode_allow_header_add` | bool      | No       | false         | When both `failure_mode_allow` and `failure_mode_allow_header_add` are set to true, if communication with the authorization service fails or returns an HTTP 5xx error, the request header will include `x-envoy-auth-failure-mode-allowed: true` |
| `status_on_error`               | int       | No       | 403           | Sets the HTTP status code returned to the client when the authorization service is unreachable or returns a 5xx status code. The default status code is `403`          |

### Configuration Fields for Each Item in `http_service`
| Name                     | Data Type | Required | Default Value | Description                                  |
| ------------------------ | --------- | -------- | ------------- | -------------------------------------------- |
| `endpoint_mode`          | string    | No       | envoy         | Select either `envoy` or `forward_auth` as an optional choice |
| `endpoint`               | object    | Yes      | -             | Information about the HTTP service for sending authentication requests |
| `timeout`                | int       | No       | 1000          | Connection timeout for `ext-auth` service, in milliseconds |
| `authorization_request`  | object    | No       | -             | Configuration for sending authentication requests |
| `authorization_response` | object    | No       | -             | Configuration for processing authentication responses |

### Configuration Fields for Each Item in `endpoint`
| Name             | Data Type | Required               | Default Value | Description                                                                                                   |
| ---------------- | --------- | ---------------------- | ------------- | ------------------------------------------------------------------------------------------------------------- |
| `service_name`   | string    | Required               | -             | Input the name of the authorization service, in complete FQDN format, e.g., `ext-auth.dns` or `ext-auth.my-ns.svc.cluster.local` |
| `service_port`   | int       | No                     | 80            | Input the port of the authorization service                                                                      |
| `service_host`   | string    | No                     | -             | The Host header set when requesting the authorization service; remains the same as FQDN if not filled          |
| `path_prefix`    | string    | Required when `endpoint_mode` is `envoy` |             | Request path prefix for the client when sending requests to the authorization service                          |
| `request_method` | string    | No                     | GET           | HTTP Method for client requests to the authorization service when `endpoint_mode` is `forward_auth`            |
| `path`           | string    | Required when `endpoint_mode` is `forward_auth` | -             | Request path for the client when sending requests to the authorization service                                   |

### Configuration Fields for Each Item in `authorization_request`
| Name                     | Data Type               | Required | Default Value | Description                                                                                                                                                            |
| ------------------------ | ---------------------- | -------- | ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `allowed_headers`        | array of StringMatcher | No       | -             | When set, client request headers with matching criteria will be added to the headers of the request to the authorization service. The `Authorization` HTTP header will be automatically included in the authorization service request, and if `endpoint_mode` is `forward_auth`, the original request path will be set to `X-Original-Uri` and the original request HTTP method will be set to `X-Original-Method`. |
| `headers_to_add`         | `map[string]string`    | No       | -             | Sets the list of request headers to include in the authorization service request. Note that headers with the same name from the client will be overwritten.              |
| `with_request_body`      | bool                   | No       | false         | Buffer the client request body and send it in the authentication request (does not take effect for HTTP Methods GET, OPTIONS, and HEAD)                               |
| `max_request_body_bytes` | int                    | No       | 10MB          | Sets the maximum size of the client request body to keep in memory. When the client request body reaches the value set in this field, an HTTP 413 status code will be returned, and the authorization process will not start. Note that this setting takes precedence over the `failure_mode_allow` configuration. |

### Configuration Fields for Each Item in `authorization_response`
| Name                       | Data Type               | Required | Default Value | Description                                                                                     |
| -------------------------- | ---------------------- | -------- | ------------- | ----------------------------------------------------------------------------------------------- |
| `allowed_upstream_headers` | array of StringMatcher | No       | -             | When set, the response headers of the authorization request with matching criteria will be added to the original client request headers. Note that headers with the same name will be overwritten. |
| `allowed_client_headers`   | array of StringMatcher | No       | -             | If not set, all response headers from authorization requests will be added to the client’s response when a request is denied. When set, response headers from authorization requests with matching criteria will be added to the client's response when a request is denied. |

### Field Descriptions for `StringMatcher` Type
When using `array of StringMatcher`, the fields are configured according to the order defined in the array.
| Name       | Data Type | Required                                            | Default Value | Description |
| ---------- | --------- | --------------------------------------------------- | ------------- | ----------- |
| `exact`    | string    | No, must select one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Exact match |
| `prefix`   | string    | No, must select one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Prefix match |
| `suffix`   | string    | No, must select one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Suffix match |
| `contains` | string    | No, must select one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Contains match |
| `regex`    | string    | No, must select one from `exact`, `prefix`, `suffix`, `contains`, `regex` | -             | Regex match |

## Configuration Example
Assuming the `ext-auth` service has a serviceName of `ext-auth`, port `8090`, path `/auth`, and namespace `backend` in Kubernetes.

Two types of `endpoint_mode` are supported:
- When `endpoint_mode` is `envoy`, the authentication request will use the original request HTTP Method, and the configured `path_prefix` will be concatenated with the original request path.
- When `endpoint_mode` is `forward_auth`, the authentication request will use the configured `request_method` as the HTTP Method and the configured `path` as the request path.

### Example 1: `endpoint_mode` is `envoy`
#### Configuration of `ext-auth` Plugin:
```yaml
http_service:
  endpoint_mode: envoy
  endpoint:
    service_name: ext-auth.backend.svc.cluster.local
    service_port: 8090
    path_prefix: /auth
  timeout: 1000
```

Using the following request to the gateway, after enabling the `ext-auth` plugin:
```shell
curl -X POST http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx"
```

**Successful request to the `ext-auth` service:**
The `ext-auth` service will receive the following authentication request:
```
POST /auth/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 HTTP/1.1
Host: ext-auth.backend.svc.cluster.local
Authorization: xxx
Content-Length: 0
```

**Failed request to the `ext-auth` service:**
When the `ext-auth` service responds with a 5xx error, the client will receive an HTTP response code of 403 along with all response headers returned by the `ext-auth` service.

If the `ext-auth` service returns `x-auth-version: 1.0` and `x-auth-failed: true` headers, these will be conveyed to the client:
```
HTTP/1.1 403 Forbidden
x-auth-version: 1.0
x-auth-failed: true
date: Tue, 16 Jul 2024 00:19:41 GMT
server: istio-envoy
content-length: 0
```

When the `ext-auth` service is inaccessible or returns a status code of 5xx, the client request will be denied with the status code configured in `status_on_error`. When the `ext-auth` service returns other HTTP status codes, the client request will be denied with the returned status code. If `allowed_client_headers` is configured, the matching response headers will be added to the client's response.

#### Example 2: `ext-auth` Plugin Configuration:
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

Using the following request to the gateway after enabling the `ext-auth` plugin:
```shell
curl -X POST http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx"
```

The `ext-auth` service will receive the following authentication request:
```
POST /auth/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 HTTP/1.1
Host: my-domain.local
Authorization: xxx
X-Auth-Version: 1.0
x-envoy-header: true
Content-Length: 0
```

If the `ext-auth` service returns headers containing `x-user-id` and `x-auth-version`, these two request headers will be included in requests to the upstream when the gateway calls it.

### Example 1: `endpoint_mode` is `forward_auth`
`ext-auth` Plugin Configuration:
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

Using the following request to the gateway after enabling the `ext-auth` plugin:
```shell
curl -i http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx"
```

**Successful request to the `ext-auth` service:**
The `ext-auth` service will receive the following authentication request:
```
POST /auth HTTP/1.1
Host: ext-auth.backend.svc.cluster.local
Authorization: xxx
X-Original-Uri: /users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5
X-Original-Method: GET
Content-Length: 0
```

**Failed request to the `ext-auth` service:**
When the `ext-auth` service responds with a 5xx error, the client will receive an HTTP response code of 403 along with all response headers returned by the `ext-auth` service.

If the `ext-auth` service returns `x-auth-version: 1.0` and `x-auth-failed: true` headers, these will be conveyed to the client:
```
HTTP/1.1 403 Forbidden
x-auth-version: 1.0
x-auth-failed: true
date: Tue, 16 Jul 2024 00:19:41 GMT
server: istio-envoy
content-length: 0
```

When the `ext-auth` service is inaccessible or returns a status code of 5xx, the client request will be denied with the status code configured in `status_on_error`. When the `ext-auth` service returns other HTTP status codes, the client request will be denied with the returned status code. If `allowed_client_headers` is configured, the matching response headers will be added to the client's response.

#### Example 2: `ext-auth` Plugin Configuration:
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

Using the following request to the gateway after enabling the `ext-auth` plugin:
```shell
curl -i http://localhost:8082/users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5 -X GET -H "foo: bar" -H "Authorization: xxx" -H "X-Auth-Version: 1.0"
```

The `ext-auth` service will receive the following authentication request:
```
POST /auth HTTP/1.1
Host: my-domain.local
Authorization: xxx
X-Original-Uri: /users?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5
X-Original-Method: GET
X-Auth-Version: 1.0
x-envoy-header: true
Content-Length: 0
```

If the `ext-auth` service returns headers containing `x-user-id` and `x-auth-version`, these two request headers will be included in requests to the upstream when the gateway calls it.

#### x-forwarded-* Header
When `endpoint_mode` is `forward_auth`, Higress will automatically generate and send the following headers to the authorization service.
| Header             | Description                                   |
|--------------------|-----------------------------------------------|
| x-forwarded-proto  | The scheme of the original request, e.g., http/https |
| x-forwarded-method | The method of the original request, e.g., get/post/delete/patch |
| x-forwarded-host   | The host of the original request               |
| x-forwarded-uri    | The path of the original request, including path parameters, e.g., /v1/app?test=true |
| x-forwarded-for    | The client IP address of the original request   |
