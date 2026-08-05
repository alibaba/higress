# dev-contract 上传大小限制变更记录

## 需求

为 `dev-contract.qtech.cn` 域名限制请求上传大小为 200M，并写入 gateway chart。

## 变更

- 在 `helm/core/templates/request-body-size-limit-envoyfilter.yaml` 新增基于 `gateway.requestBodySizeLimits` 的 EnvoyFilter 模板。
- 在 `helm/core/values.yaml` 和 `helm/core/default-2.2.1.yaml` 中配置 `dev-contract.qtech.cn:443` 的请求体上限为 `209715200` 字节。
- 超过该限制的请求将由 Envoy buffer filter 返回 HTTP 413。

## 验证策略

- 使用 `helm template` 渲染 `helm/core` chart。
- 检查渲染结果中 EnvoyFilter 的 VirtualHost 是否为 `dev-contract.qtech.cn:443`，以及 `max_request_bytes` 是否为 `209715200`。
