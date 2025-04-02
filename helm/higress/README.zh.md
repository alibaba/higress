## Higress for Kubernetes

Higress 是基于阿里巴巴内部网关实践构建的云原生 API 网关。

依托 Istio 和 Envoy，Higress 实现了流量网关、微服务网关和安全网关三重架构的融合，从而大幅降低了部署、运维成本。

## 设置仓库信息

```console
helm repo add higress.io https://higress.io/helm-charts
helm repo update
```

## 安装

以 `higress` 为发布名称安装 chart：

```console
helm install higress -n higress-system higress.io/higress --create-namespace --render-subchart-notes
```

## 卸载

要卸载/删除 higress 部署：

```console
helm delete higress -n higress-system
```

该命令会移除与 chart 相关的所有 Kubernetes 组件，并删除发布。

## 参数

## 值

| 键 | 类型 | 默认值 | 描述 |
|-----|------|---------|-------------|
| clusterName | 字符串 | `""` |  |
| controller.affinity | 对象 | `{}` |  |
| controller.automaticHttps.email | 字符串 | `""` |  |
| controller.automaticHttps.enabled | 布尔值 | `true` |  |
| controller.autoscaling.enabled | 布尔值 | `false` |  |
| controller.autoscaling.maxReplicas | 整数 | `5` |  |
| controller.autoscaling.minReplicas | 整数 | `1` |  |
| controller.autoscaling.targetCPUUtilizationPercentage | 整数 | `80` |  |
| controller.env | 对象 | `{}` |  |
| controller.hub | 字符串 | `"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress"` |  |
| controller.image | 字符串 | `"higress"` |  |
| controller.imagePullSecrets | 列表 | `[]` |  |
| controller.labels | 对象 | `{}` |  |
| controller.name | 字符串 | `"higress-controller"` |  |
| controller.nodeSelector | 对象 | `{}` |  |
| controller.podAnnotations | 对象 | `{}` |  |
| controller.podSecurityContext | 对象 | `{}` |  |
| controller.ports[0].name | 字符串 | `"http"` |  |
| controller.ports[0].port | 整数 | `8888` |  |
| controller.ports[0].protocol | 字符串 | `"TCP"` |  |
| controller.ports[0].targetPort | 整数 | `8888` |  |
| controller.ports[1].name | 字符串 | `"http-solver"` |  |
| controller.ports[1].port | 整数 | `8889` |  |
| controller.ports[1].protocol | 字符串 | `"TCP"` |  |
| controller.ports[1].targetPort | 整数 | `8889` |  |
| controller.ports[2].name | 字符串 | `"grpc"` |  |
| controller.ports[2].port | 整数 | `15051` |  |
| controller.ports[2].protocol | 字符串 | `"TCP"` |  |
| controller.ports[2].targetPort | 整数 | `15051` |  |
| controller.probe.httpGet.path | 字符串 | `"/ready"` |  |
| controller.probe.httpGet.port | 整数 | `8888` |  |
| controller.probe.initialDelaySeconds | 整数 | `1` |  |
| controller.probe.periodSeconds | 整数 | `3` |  |
| controller.probe.timeoutSeconds | 整数 | `5` |  |
| controller.rbac.create | 布尔值 | `true` |  |
| controller.replicas | 整数 | `1` | Higress Controller 的 Pod 数量 |
| controller.resources.limits.cpu | 字符串 | `"1000m"` |  |
| controller.resources.limits.memory | 字符串 | `"2048Mi"` |  |
| controller.resources.requests.cpu | 字符串 | `"500m"` |  |
| controller.resources.requests.memory | 字符串 | `"2048Mi"` |  |
| controller.securityContext | 对象 | `{}` |  |
| controller.service.type | 字符串 | `"ClusterIP"` |  |
| controller.serviceAccount.annotations | 对象 | `{}` | 添加到服务账户的注解 |
| controller.serviceAccount.create | 布尔值 | `true` | 指定是否创建服务账户 |
| controller.serviceAccount.name | 字符串 | `""` | 如果未设置且 create 为 true，则使用 fullname 模板生成名称 |
| controller.tag | 字符串 | `""` |  |
| controller.tolerations | 列表 | `[]` |  |
| downstream | 对象 | `{"connectionBufferLimits":32768,"http2":{"initialConnectionWindowSize":1048576,"initialStreamWindowSize":65535,"maxConcurrentStreams":100},"idleTimeout":180,"maxRequestHeadersKb":60,"routeTimeout":0}` | 下游配置设置 |
| gateway.affinity | 对象 | `{}` |  |
| gateway.annotations | 对象 | `{}` | 应用到所有资源的注解 |
| gateway.autoscaling.enabled | 布尔值 | `false` |  |
| gateway.autoscaling.maxReplicas | 整数 | `5` |  |
| gateway.autoscaling.minReplicas | 整数 | `1` |  |
| gateway.autoscaling.targetCPUUtilizationPercentage | 整数 | `80` |  |
| gateway.containerSecurityContext | 字符串 | `nil` |  |
| gateway.env | 对象 | `{}` | Pod 环境变量 |
| gateway.hostNetwork | 布尔值 | `false` |  |
| gateway.httpPort | 整数 | `80` |  |
| gateway.httpsPort | 整数 | `443` |  |
| gateway.hub | 字符串 | `"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress"` |  |
| gateway.image | 字符串 | `"gateway"` |  |
| gateway.kind | 字符串 | `"Deployment"` | 使用 `DaemonSet` 或 `Deployment` |
| gateway.labels | 对象 | `{}` | 应用到所有资源的标签 |
| gateway.metrics.enabled | 布尔值 | `false` | 如果为 true，则为网关创建 PodMonitor 或 VMPodScrape |
| gateway.metrics.honorLabels | 布尔值 | `false` |  |
| gateway.metrics.interval | 字符串 | `""` |  |
| gateway.metrics.metricRelabelConfigs | 列表 | `[]` | 用于 operator.victoriametrics.com/v1beta1.VMPodScrape |
| gateway.metrics.metricRelabelings | 列表 | `[]` | 用于 monitoring.coreos.com/v1.PodMonitor |
| gateway.metrics.provider | 字符串 | `"monitoring.coreos.com"` | CustomResourceDefinition 的提供者组名，可以是 monitoring.coreos.com 或 operator.victoriametrics.com |
| gateway.metrics.rawSpec | 对象 | `{}` | 更多原始的 podMetricsEndpoints 规范 |
| gateway.metrics.relabelConfigs | 列表 | `[]` |  |
| gateway.metrics.relabelings | 列表 | `[]` |  |
| gateway.metrics.scrapeTimeout | 字符串 | `""` |  |
| gateway.name | 字符串 | `"higress-gateway"` |  |
| gateway.networkGateway | 字符串 | `""` | 如果指定，网关将作为给定网络的网络网关。 |
| gateway.nodeSelector | 对象 | `{}` |  |
| gateway.podAnnotations."prometheus.io/path" | 字符串 | `"/stats/prometheus"` |  |
| gateway.podAnnotations."prometheus.io/port" | 字符串 | `"15020"` |  |
| gateway.podAnnotations."prometheus.io/scrape" | 字符串 | `"true"` |  |
| gateway.podAnnotations."sidecar.istio.io/inject" | 字符串 | `"false"` |  |
| gateway.rbac.enabled | 布尔值 | `true` | 如果启用，将创建角色以启用从网关访问证书。当使用 http://gateway-api.org/ 时不需要。 |
| gateway.readinessFailureThreshold | 整数 | `30` | 指示准备失败前的连续失败探测次数。 |
| gateway.readinessInitialDelaySeconds | 整数 | `1` | 准备探测的初始延迟秒数。 |
| gateway.readinessPeriodSeconds | 整数 | `2` | 准备探测之间的间隔。 |
| gateway.readinessSuccessThreshold | 整数 | `1` | 指示准备成功前的连续成功探测次数。 |
| gateway.readinessTimeoutSeconds | 整数 | `3` | 准备探测的超时秒数 |
| gateway.replicas | 整数 | `2` | Higress Gateway 的 Pod 数量 |
| gateway.resources.limits.cpu | 字符串 | `"2000m"` |  |
| gateway.resources.limits.memory | 字符串 | `"2048Mi"` |  |
| gateway.resources.requests.cpu | 字符串 | `"2000m"` |  |
| gateway.resources.requests.memory | 字符串 | `"2048Mi"` |  |
| gateway.revision | 字符串 | `""` | 修订声明此网关属于哪个修订 |
| gateway.rollingMaxSurge | 字符串 | `"100%"` |  |
| gateway.rollingMaxUnavailable | 字符串 | `"25%"` |  |
| gateway.securityContext | 字符串 | `nil` | 定义 Pod 的安全上下文。如果未设置，将自动设置为绑定到端口 80 和 443 所需的最小权限。在 Kubernetes 1.22+ 上，这只需要 `net.ipv4.ip_unprivileged_port_start` 系统调用。 |
| gateway.service.annotations | 对象 | `{}` |  |
| gateway.service.externalTrafficPolicy | 字符串 | `""` |  |
| gateway.service.loadBalancerClass | 字符串 | `""` |  |
| gateway.service.loadBalancerIP | 字符串 | `""` |  |
| gateway.service.loadBalancerSourceRanges | 列表 | `[]` |  |
| gateway.service.ports[0].name | 字符串 | `"http2"` |  |
| gateway.service.ports[0].port | 整数 | `80` |  |
| gateway.service.ports[0].protocol | 字符串 | `"TCP"` |  |
| gateway.service.ports[0].targetPort | 整数 | `80` |  |
| gateway.service.ports[1].name | 字符串 | `"https"` |  |
| gateway.service.ports[1].port | 整数 | `443` |  |
| gateway.service.ports[1].protocol | 字符串 | `"TCP"` |  |
| gateway.service.ports[1].targetPort | 整数 | `443` |  |
| gateway.service.type | 字符串 | `"LoadBalancer"` | 服务类型。设置为 "None" 以完全禁用服务 |
| gateway.serviceAccount.annotations | 对象 | `{}` | 添加到服务账户的注解 |
| gateway.serviceAccount.create | 布尔值 | `true` | 如果设置，将创建服务账户。否则，使用默认值 |
| gateway.serviceAccount.name | 字符串 | `""` | 要使用的服务账户名称。如果未设置，则使用发布名称 |
| gateway.tag | 字符串 | `""` |  |
| gateway.tolerations | 列表 | `[]` |  |
| gateway.unprivilegedPortSupported | 字符串 | `nil` |  |
| global.autoscalingv2API | 布尔值 | `true` | 是否使用 autoscaling/v2 模板进行 HPA 设置，仅供内部使用，用户不应配置。 |
| global.caAddress | 字符串 | `""` | 自定义的 CA 地址，用于为集群中的 Pod 检索证书。CSR 客户端（如 Istio Agent 和 ingress gateways）可以使用此地址指定 CA 端点。如果未明确设置，则默认为 Istio 发现地址。 |
| global.caName | 字符串 | `""` | 工作负载证书的 CA 名称。例如，当 caName=GkeWorkloadCertificate 时，GKE 工作负载证书将用作工作负载的证书。默认值为 ""，当 caName="" 时，CA 将通过其他机制（如环境变量 CA_PROVIDER）配置。 |
| global.configCluster | 布尔值 | `false` | 将远程集群配置为外部 istiod 的配置集群。 |
| global.defaultPodDisruptionBudget | 对象 | `{"enabled":false}` | 为控制平面启用 Pod 中断预算，用于确保 Istio 控制平面组件逐步升级或恢复。 |
| global.defaultResources | 对象 | `{"requests":{"cpu":"10m"}}` | 应用于所有部署的最小请求资源集，以便 Horizontal Pod Autoscaler 能够正常工作（如果设置）。每个组件可以通过在相关部分添加自己的资源块并设置所需的资源值来覆盖这些默认值。 |
| global.defaultUpstreamConcurrencyThreshold | 整数 | `10000` |  |
| global.disableAlpnH2 | 布尔值 | `false` | 是否在 ALPN 中禁用 HTTP/2 |
| global.enableGatewayAPI | 布尔值 | `false` | 如果为 true，Higress Controller 还将监控 Gateway API 资源 |
| global.enableH3 | 布尔值 | `false` |  |
| global.enableIPv6 | 布尔值 | `false` |  |
| global.enableIstioAPI | 布尔值 | `true` | 如果为 true，Higress Controller 还将监控 istio 资源 |
| global.enableLDSCache | 布尔值 | `true` |  |
| global.enableProxyProtocol | 布尔值 | `false` |  |
| global.enablePushAllMCPClusters | 布尔值 | `true` |  |
| global.enableSRDS | 布尔值 | `true` |  |
| global.enableStatus | 布尔值 | `true` | 如果为 true，Higress Controller 将更新 Ingress 资源的状态字段。从 Nginx Ingress 迁移时，为了避免 Ingress 对象的状态字段被覆盖，需要将此参数设置为 false，以便 Higress 不会将入口 IP 写入相应 Ingress 对象的状态字段。 |
| global.externalIstiod | 布尔值 | `false` | 配置由外部 istiod 控制的远程集群数据平面。当设置为 true 时，本地不部署 istiod，仅启用其他发现 chart 的子集。 |
| global.hostRDSMergeSubset | 布尔值 | `false` |  |
| global.hub | 字符串 | `"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress"` | Istio 镜像的默认仓库。发布版本发布到 docker hub 的 'istio' 项目下。来自 prow 的开发构建位于 gcr.io |
| global.imagePullPolicy | 字符串 | `""` | 如果不需要默认行为，则指定镜像拉取策略。默认行为：最新镜像将始终拉取，否则 IfNotPresent。 |
| global.imagePullSecrets | 列表 | `[]` | 所有 ServiceAccount 的 ImagePullSecrets，用于引用此 ServiceAccount 的 Pod 拉取任何镜像的同一命名空间中的秘密列表。对于不使用 ServiceAccount 的组件（即 grafana、servicegraph、tracing），ImagePullSecrets 将添加到相应的 Deployment(StatefulSet) 对象中。对于配置了私有 docker 注册表的任何集群，必须设置。 |
| global.ingressClass | 字符串 | `"higress"` | IngressClass 过滤 higress controller 监听的 ingress 资源。默认的 ingress class 是 higress。有一些特殊情况用于特殊的 ingress class。1. 当 ingress class 设置为 nginx 时，higress controller 将监听带有 nginx ingress class 或没有任何 ingress class 的 ingress 资源。2. 当 ingress class 设置为空时，higress controller 将监听 k8s 集群中的所有 ingress 资源。 |
| global.istioNamespace | 字符串 | `"istio-system"` | 用于定位 istiod。 |
| global.istiod | 对象 | `{"enableAnalysis":false}` | 默认在主分支中启用以最大化测试。 |
| global.jwtPolicy | 字符串 | `"third-party-jwt"` | 配置验证 JWT 的策略。目前支持两个选项："third-party-jwt" 和 "first-party-jwt"。 |
| global.kind | 布尔值 | `false` |  |
| global.liteMetrics | 布尔值 | `false` |  |
| global.local | 布尔值 | `false` | 当部署到本地集群（如：kind 集群）时，将此设置为 true。 |
| global.logAsJson | 布尔值 | `false` |  |
| global.logging | 对象 | `{"level":"default:info"}` | 以逗号分隔的每个范围的最小日志级别，格式为 <scope>:<level>,<scope>:<level> 控制平面根据组件不同有不同的范围，但可以配置所有组件的默认日志级别 如果为空，将使用代码中配置的默认范围和级别 |
| global.meshID | 字符串 | `""` | 如果网格管理员未指定值，Istio 将使用网格的信任域的值。最佳实践是选择一个合适的信任域值。 |
| global.meshNetworks | 对象 | `{}` |  |
| global.mountMtlsCerts | 布尔值 | `false` | 使用用户指定的、挂载的密钥和证书用于 Pilot 和工作负载。 |
| global.multiCluster.clusterName | 字符串 | `""` | 应设置为此安装运行的集群的名称。这是为了正确标记代理的 sidecar 注入所必需的 |
| global.multiCluster.enabled | 布尔值 | `true` | 设置为 true 以通过各自的 ingressgateway 服务连接两个 kubernetes 集群，当每个集群中的 Pod 无法直接相互通信时。
