# log-request-response 插件

这个插件用于在 Higress 的访问日志中添加以下信息：

- HTTP 请求头（添加为 `%FILTER_STATE(wasm.log-request-headers:PLAIN)%`）
- POST、PUT 等请求的请求体内容（添加为 `%FILTER_STATE(wasm.log-request-body:PLAIN)%`）
- 响应头（添加为 `%FILTER_STATE(wasm.log-response-headers:PLAIN)%`）
- 响应体内容（添加为 `%FILTER_STATE(wasm.log-response-body:PLAIN)%`）

## 配置参数

在 Higress 控制台配置该插件时，可以使用以下 YAML 配置：

```yaml
# 是否记录请求头（默认：false）
logRequestHeaders: false

# 是否记录请求体内容（默认：false）
logRequestBody: false

# 是否记录响应头（默认：false）
logResponseHeaders: false

# 是否记录响应体内容（默认：false）
logResponseBody: false

# 需要记录请求体的内容类型（可选，默认包含常见的内容类型）
requestBodyContentTypes:
  - application/json
  - application/xml
  - application/x-www-form-urlencoded
  - text/plain

# 最大记录长度限制，单位字节（可选，默认：10KB）
maxBodySize: 10240
```

## 工作原理

1. 请求处理时，插件会根据配置决定是否记录请求头和请求体
2. 只有当请求方法为 POST、PUT 或 PATCH，且内容类型在配置的 `requestBodyContentTypes` 列表中时，才会记录请求体
3. 响应处理时，插件会根据配置决定是否记录响应头和响应体
4. 所有记录的内容都会被限制在 `maxBodySize` 指定的大小内
5. 记录的内容会被存储在 Envoy 的 Filter State 中，可以通过访问日志配置获取

## 编译方法

本插件使用了 Higress 扩展的 0.2.100 版本 ABI 功能，编译时请使用以下命令：

```bash
# 先整理依赖
go mod tidy

# 使用支持0.2.100版本ABI的编译命令
tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags="custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100" ./main.go
```

## 访问日志配置

要在 Higress 访问日志中显示插件添加的 Filter State 数据，需要修改 Higress 的访问日志配置。编辑 ConfigMap：

```bash
kubectl edit cm -n higress-system higress-config
```

在 `envoyAccessLogService.config.accessLog` 下的 `format` 字段中添加以下内容：

```json
{
  "request_headers": "%FILTER_STATE(wasm.log-request-headers:PLAIN)%",
  "request_body": "%FILTER_STATE(wasm.log-request-body:PLAIN)%",
  "response_headers": "%FILTER_STATE(wasm.log-response-headers:PLAIN)%",
  "response_body": "%FILTER_STATE(wasm.log-response-body:PLAIN)%"
}
```

完整的访问日志配置可能会像这样（添加到现有配置中）：

```yaml
mesh:
  accessLogFile: "/dev/stdout"
  accessLogFormat: |
    {
      "authority": "%REQ(:AUTHORITY)%",
      "bytes_received": "%BYTES_RECEIVED%",
      "bytes_sent": "%BYTES_SENT%",
      "downstream_local_address": "%DOWNSTREAM_LOCAL_ADDRESS%",
      "downstream_remote_address": "%DOWNSTREAM_REMOTE_ADDRESS%",
      "duration": "%DURATION%",
      "method": "%REQ(:METHOD)%",
      "path": "%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%",
      "protocol": "%PROTOCOL%",
      "request_id": "%REQ(X-REQUEST-ID)%",
      "requested_server_name": "%REQUESTED_SERVER_NAME%",
      "response_code": "%RESPONSE_CODE%",
      "response_flags": "%RESPONSE_FLAGS%",
      "route_name": "%ROUTE_NAME%",
      "start_time": "%START_TIME%",
      "trace_id": "%REQ(X-B3-TRACEID)%",
      "upstream_cluster": "%UPSTREAM_CLUSTER%",
      "upstream_host": "%UPSTREAM_HOST%",
      "upstream_local_address": "%UPSTREAM_LOCAL_ADDRESS%",
      "upstream_service_time": "%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%",
      "upstream_transport_failure_reason": "%UPSTREAM_TRANSPORT_FAILURE_REASON%",
      "user_agent": "%REQ(USER-AGENT)%",
      "x_forwarded_for": "%REQ(X-FORWARDED-FOR)%",
      "request_headers": "%FILTER_STATE(wasm.log-request-headers:PLAIN)%",
      "request_body": "%FILTER_STATE(wasm.log-request-body:PLAIN)%",
      "response_headers": "%FILTER_STATE(wasm.log-response-headers:PLAIN)%",
      "response_body": "%FILTER_STATE(wasm.log-response-body:PLAIN)%"
    }
```

## 日志输出示例

配置完成后，Higress 的访问日志中将包含这些额外的字段（取决于您的配置启用了哪些选项）：

```json
{
  "authority": "example.com",
  "method": "POST",
  "path": "/api/users",
  "response_code": 200,
  "request_headers": "{\"host\":\"example.com\",\":path\":\"/api/users\",\":method\":\"POST\",\"content-type\":\"application/json\"}",
  "request_body": "{\"name\":\"测试用户\",\"email\":\"test@example.com\"}",
  "response_headers": "{\"content-type\":\"application/json\",\"status\":\"200\"}",
  "response_body": "{\"id\":123,\"status\":\"success\"}"
}
```

## 注意事项

1. 所有日志记录选项默认都是关闭的（false），需要明确启用才会记录相应内容
2. 对于大型请求体或响应体，可以通过 `maxBodySize` 参数限制记录的长度，以避免日志过大
3. 只有指定内容类型的 POST、PUT、PATCH 请求才会记录请求体内容
4. 请确保合理配置该插件，避免记录敏感信息到日志中
5. 本插件使用了 Higress 扩展的 0.2.100 版本 ABI 功能