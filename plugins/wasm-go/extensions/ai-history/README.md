## 简介

**Note**

> 需要数据面的proxy wasm版本大于等于0.2.100

> 编译时，需要带上版本的tag，例如：
`tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags="custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100" ./`

LLM 对话历史保存插件，用于保存对话历史，同时支持流式和非流式响应的缓存。

## 配置说明

| Name              | Type    | Requirement | Default               | Description                                                               |
|-------------------|---------|-------------|-----------------------|---------------------------------------------------------------------------|
| identityHeader    | string  | optional    | "Authorization"       | 身份解析对应的请求头                                                                |
| fillHistoryCnt    | integer | optional    | 3                     | 默认填充历史对话轮次                                                                |
| cacheKeyPrefix    | string  | optional    | "higress-ai-history:" | Redis缓存Key的前缀                                                             |
| redis.serviceName | string  | required    | -                     | redis 服务名称，带服务类型的完整 FQDN 名称，例如 my-redis.dns、redis.my-ns.svc.cluster.local |
| redis.servicePort | integer | optional    | 6379                  | redis 服务端口                                                                |
| redis.timeout     | integer | optional    | 1000                  | 请求 redis 的超时时间，单位为毫秒                                                      |
| redis.username    | string  | optional    | -                     | 登陆 redis 的用户名                                                             |
| redis.password    | string  | optional    | -                     | 登陆 redis 的密码                                                              |

> identity_type 可选项:
>
>- Authorization
>
>- X-Mse-Consumer

## 配置示例

```yaml
redis:
  serviceName: my-redis.dns
  timeout: 2000
```

## 请求说明

**自动填充请求示例：**

```
 curl 'http://example.com/api/openai/v1/chat/completions?fill_history_cnt=3' \
  -H 'Accept: application/json, text/event-stream' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer sk-Nzf7RtkdS4s0zFyn5575124129254d9bAf9473A5D7D06dD3'
  --data-raw '{"model":"qwen-long","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[
        {
            "role": "user",
            "content": "用最简短的一句话概述Higress 的全局资源限制怎么设定？"
        }
    ],"presence_penalty":0,"temperature":0.7,"top_p":0.95}'
```

每轮请求只需要带上当前问题，以及当前需要填充的历史对话轮数，即可自动完成历史对话填充。

**获取历史数据示例：**

```
curl 'http://example.com/api/openai/v1/chat/completions/ai-history/query?cnt=3' \
  -H 'Accept: application/json, text/event-stream' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer sk-Nzf7RtkdS4s0zFyn5575124129254d9bAf9473A5D7D06dD3'
```

响应示例:

```json
[
  {
    "role": "user",
    "content": "Higress 可以替换 Nginx 吗？"
  },
  {
    "role": "assistant",
    "content": "Higress 和 Nginx 虽然都有作为网关的功能，但它们的设计理念和应用场景有所不同。Nginx 更多是作为一个高性能的 HTTP 和反向代理服务器被大家熟知，而 Higress 是一个云原生网关，除了基础的路由转发能力外，还集成了服务网格、可观测性、安全管理等众多云原生特性。\\n\\n因此，如果你想在云原生环境中部署应用，并且希望获得现代应用所需的高级功能，比如服务治理、灰度发布、熔断限流、安全认证等功能，那么 Higress 可以作为一个很好的 Nginx 替代方案。但如果是较为简单的静态网站或者仅需要基本的反向代理功能，传统的 Nginx 配置可能会更为简单直接。"
  },
  {
    "role": "user",
    "content": "SpringCloud GateWay 呢？"
  },
  {
    "role": "assistant",
    "content": "与 Spring Cloud Gateway 相比，Higress 也是一个 API 网关，但它们之间存在一些关键的区别：\\n\\n- **设计理念**：Spring Cloud Gateway 主要针对微服务架构中的服务间通信和路由，它作为 Spring Cloud 生态系统的一部分，更加专注于 Java 开发者的微服务场景。而 Higress 作为云原生网关，不仅关注服务间的通信，还提供了一系列云原生功能，如服务网格、可观测性、安全管理等。\\n- **部署方式**：Spring Cloud Gateway 通常作为微服务应用的一部分运行在应用服务器内，而 Higress 通常以独立的微服务或者容器化服务的形式部署在 Kubernetes 环境中，适用于现代云原生部署模型。\\n- **扩展性和集成**：Higress 提供了更广泛的集成和支持，例如与 Istio、Kubernetes 等生态系统的深度集成，这使得它可以更好地适应复杂的云原生环境。\\n\\n因此，如果你的应用程序是基于 Spring Cloud 构建的，并且你想要一个轻量级的、易于集成的服务网关，那么 Spring Cloud Gateway 可能是一个合适的选择。但是，如果你正在构建或重构云原生应用，并且需要更强大的路由规则、服务治理、可观测性等功能，那么 Higress 将是一个更好的选择。"
  },
  {
    "role": "user",
    "content": "Higress 可以替换 Nginx 吗？"
  },
  {
    "role": "assistant",
    "content": "Higress 和 Nginx 虽然都有作为网关的功能，但它们的设计理念和应用场景有所不同。Nginx 更多是作为一个高性能的 HTTP 和反向代理服务器被大家熟知，而 Higress 是一个云原生网关，除了基础的路由转发能力外，还集成了服务网格、可观测性、安全管理等众多云原生特性。\\n\\n因此，如果你想在云原生环境中部署应用，并且希望获得现代应用所需的高级功能，比如服务治理、灰度发布、熔断限流、安全认证等功能，那么 Higress 可以作为一个很好的 Nginx 替代方案。但如果是较为简单的静态网站或者仅需要基本的反向代理功能，传统的 Nginx 配置可能会更为简单直接。"
  }
]
```

返回三个历史对话,如果未传入 cnt 默认返回所有缓存历史对话。