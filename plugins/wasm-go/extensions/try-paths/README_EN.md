---
title: request retry
keywords: [higress,try-paths]
description: request retry
---

# Function Description
`try-paths` plugin supports retrying requests based on different paths until a correct response is received, similar to the try files instruction in nginx.

# Configuration Fields

| Name | Data Type | Fill Requirement |  Default Value | Description |
| -------- | -------- | -------- | -------- | -------- |
| serviceSource  | string           | Required    | -          | Support k8s,nacos,ip,dns                                 |
| domain         | string           | Optional    | -          | Service Domain (Required when serviceSource is `dns`)                 |
| host           | string           | Optional    | -          | Access host address(Required when serviceSource is `k8s,nacos,ip`) |
| serviceName    | string           | Optional    | -          | Service Name（Required when serviceSource is `k8s,nacos,ip,dns`）    |
| servicePort    | string           | Optional    | -          | Service Port（Required when serviceSource is `k8s,nacos,ip,dns`）    |
| namespace      | string           | Optional    | -          | Service Namespace（Required when serviceSource is `k8s,nacos`）           |
| tryPaths       | array of string  | Required    | -          | Try path list，`index.html`，`$uri/`, `index.html` for example        |
| tryCodes       | array of int     | Optional    | [403, 404] | Try response code，can be customized                    |
| timeout        | int              | Optional    | 1000       | The timeout for try request，unit is ms                                     |

# Configuration Example

## scene with try-paths plugin configured

```yaml
namespace: "default"
serviceName: "oss"
servicePort: 80
serviceSource: "k8s"
host: "<bucket name>.oss-cn-hangzhou.aliyuncs.com"
tryPaths:
- "$uri/"
- "$uri.html"
- "/index.html"

```

From the above configuration, the `try-paths` plugin is enabled. The request "curl http://a.com/a" will be tried with the following paths:
- http://<bucket name>.oss-cn-hangzhou.aliyuncs.com/a/
- http://<bucket name>.oss-cn-hangzhou.aliyuncs.com/a.html
- http://<bucket name>.oss-cn-hangzhou.aliyuncs.com/index.html
If the response code is not the retry status code, the response will be returned directly, otherwise the next request will be tried. If all requests are not retry status codes, the default backend service will be requested.
