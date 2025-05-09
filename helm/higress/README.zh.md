## Higress 适用于 Kubernetes

Higress 是基于阿里巴巴内部网关实践构建的云原生 API 网关。

借助 Istio 和 Envoy，Higress 实现了流量网关、微服务网关和安全网关的三合一架构整合，从而大幅降低了部署、运维和维护的成本。

## 添加仓库信息

```console
helm repo add higress.io https://higress.io/helm-charts
helm repo update
```

## 安装

使用名为 `higress` 的发布名进行安装：

```console
helm install higress -n higress-system higress.io/higress --create-namespace --render-subchart-notes
```

## 卸载

卸载或删除 Higress 部署：

```console
helm delete higress -n higress-system
```

该命令将移除与该图表相关的所有 Kubernetes 组件，并删除发布记录。

## 参数

## 配置值

| 键 | 类型     | 默认值         | 描述 |
|---|---------|--------------|-------|
| clusterName | string   | ""       |        |
| controller.affinity | object  | {}      |           |
| controller.automaticHttps.email | string  | ""        |         |
| controller.automaticHttps.enabled | bool   | true      |        |
| controller.autoscaling.enabled | bool  | false     |        |
| controller.autoscaling.maxReplicas | int    | 5         |        |
| controller.autoscaling.minReplicas | int    | 1         |         |
| controller.autoscaling.targetCPUUtilizationPercentage | int | 80  |              |
| controller.env | object | {}       |           |
| controller.hub | string | "higress-registry.cn-hangzhou.cr.aliyuncs.com/higress" |      |
| controller.image | string | "higress" |       | 
| controller.imagePullSecrets | list | [] |            |
| controller.labels | object | {} |            |
| controller.name | string | "higress-controller" |       |
| controller.nodeSelector | object | {} |             |
| controller.podAnnotations | object | {} |        |
| controller.podLabels | object | {} | 应用于Pod的标签 |
| controller.podSecurityContext | object | {} |          |
| controller.ports[0].name | string | "http" |       |
| controller.ports[0].port | int | 8888 |       |
| controller.ports[0].protocol | string | "TCP" |       |
| controller.ports[0].targetPort | int | 8888 |       |
| controller.ports[1].name | string | "http-solver" |       |
| controller.ports[1].port | int | 8889 |        |
| controller.ports[1].protocol | string | "TCP" |        |
| controller.ports[1].targetPort | int | 8889 |        |
| controller.ports[2].name | string | "grpc" |       |
| controller.ports[2].port | int | 15051 |        |
| controller.ports[2].protocol | string | "TCP" |        |
| controller.ports[2].targetPort | int | 15051 |        |
| controller.probe.httpGet.path | string | "/ready" |        |
| controller.probe.httpGet.port | int | 8888 |        |
| controller.probe.initialDelaySeconds | int | 1 |        |
| controller.probe.periodSeconds | int | 3 |        |
| controller.probe.timeoutSeconds | int | 5 |        |
| controller.rbac.create | bool | true |        |
| controller.replicas | int | 1 | Higress 控制器 Pod 数量 |
| controller.resources.limits.cpu | string | "1000m" |        |
| controller.resources.limits.memory | string | "2048Mi" |        |
| controller.resources.requests.cpu | string | "500m" |        |
| controller.resources.requests.memory | string | "2048Mi" |        |
| controller.securityContext | object | {} |        |
| controller.service.type | string | "ClusterIP" |        |
| controller.serviceAccount.annotations | object | {} | 要添加到服务账户的注解 |
| controller.serviceAccount.create | bool | true | 指定是否应创建服务账户 |
| controller.serviceAccount.name | string | "" | 如果未设置且 create 为 true，则使用 fullname 模板生成一个名称 |
| controller.tag | string | "" |        |
| controller.tolerations | list | [] |        |
| downstream.connectionBufferLimits | int | 32768 | 下游配置设置 |
| downstream.idleTimeout | int | 180 |        |
| downstream.maxRequestHeadersKb | int | 60 |        |
| downstream.routeTimeout | int | 0 |        |
| downstream.http2.initialConnectionWindowSize | int | 1048576 |        |
| downstream.http2.initialStreamWindowSize | int | 65535 |        |
| downstream.http2.maxConcurrentStreams | int | 100 |        |
| gateway.affinity | object | {} |        |
| gateway.annotations | object | {} | 应用于所有资源的注解 |
| gateway.autoscaling.enabled | bool | false |        |
| gateway.autoscaling.maxReplicas | int | 5 |        |
| gateway.autoscaling.minReplicas | int | 1 |        |
| gateway.autoscaling.targetCPUUtilizationPercentage | int | 80 |        |
| gateway.containerSecurityContext | string | nil |        |
| gateway.env | object | {} | Pod 环境变量 |
| gateway.hostNetwork | bool | false |        |
| gateway.httpPort | int | 80 |        |
| gateway.httpsPort | int | 443 |        |
| gateway.hub | string | "higress-registry.cn-hangzhou.cr.aliyuncs.com/higress" |        |
| gateway.image | string | "gateway" |        |
| gateway.kind | string | "Deployment" | 使用 `DaemonSet` 或 `Deployment` |
| gateway.labels | object | {} | 应用于所有资源的标签 |
| gateway.metrics.enabled | bool | false | 如果为 true，将为网关创建 PodMonitor 或 VMPodScrape |
| gateway.metrics.honorLabels | bool | false |        |
| gateway.metrics.interval | string | "" |        |
| gateway.metrics.metricRelabelConfigs | list | [] | 适用于 operator.victoriametrics.com/v1beta1.VMPodScrape |
| gateway.metrics.metricRelabelings | list | [] | 适用于 monitoring.coreos.com/v1.PodMonitor |
| gateway.metrics.rawSpec | object | {} | 更多原始 podMetricsEndpoints 规范 |
| gateway.metrics.provider | string | "monitoring.coreos.com" | CustomResourceDefinition 的提供程序组名，可以是 monitoring.coreos.com 或 operator.victoriametrics.com |
| gateway.name | string | "higress-gateway" |        |
| gateway.networkGateway | string | "" | 如果指定，网关将充当给定网络的网络网关。 |
| gateway.nodeSelector | object | {} |        |
| gateway.podAnnotations."prometheus.io/path" | string | "/stats/prometheus" |        |
| gateway.podAnnotations."prometheus.io/port" | string | "15020" |        |
| gateway.podAnnotations."prometheus.io/scrape" | string | "true" |        |
| gateway.podAnnotations."sidecar.istio.io/inject" | string | "false" |        |
| gateway.podLabels | object | {} | 应用到_pod 上的标签 |
| gateway.rbac.enabled | bool | true | 如果启用，将在访问证书中创建角色。当使用 http://gateway-api.org/ 时不需要此项。 |
| gateway.readinessFailureThreshold | int | 30 | 连续失败探针的数量，直到标记为就绪状态失效 |
| gateway.readinessInitialDelaySeconds | int | 1 | 就绪检测探针的初始延迟 (以秒为单位) |
| gateway.readinessPeriodSeconds | int | 2 | 就绪性探测的时间间隔 |
| gateway.readinessSuccessThreshold | int | 1 | 在表示就绪之前连续成功探针的数量 |
| gateway.readinessTimeoutSeconds | int | 3 | 就绪状态超时时间（秒） |
| gateway.replicas | int | 2 | Higress Gateway pods 的数量 |
| gateway.resources.limits.cpu | string | "2000m" |        |
| gateway.resources.limits.memory | string | "2048Mi" |        |
| gateway.resources.requests.cpu | string | "2000m" |        |
| gateway.resources.requests.memory | string | "2048Mi" |        |
| gateway.revision | string | "" | 版本声明此网关隶属哪个版本 |
| gateway.rollingMaxSurge | string | "100%" |        |
| gateway.rollingMaxUnavailable | string | "25%" | 全局本地为真，默认值为 100%，否则为 25% |
| gateway.securityContext | string | nil | 为 pod 定义权限环境。如果未设置，会自动设定为绑定端口 80 和 443 所需的最小权限。在 Kubernetes 1.22+ 中，只需要 net.ipv4.ip_unprivileged_port_start sysctl即可 |
| gateway.service.annotations | object | {} |        |
| gateway.service.externalTrafficPolicy | string | "" |        |
| gateway.service.loadBalancerClass | string | "" |        |
| gateway.service.loadBalancerIP | string | "" |        |
| gateway.service.loadBalancerSourceRanges | list | [] |        |
| gateway.service.type | string | "LoadBalancer" | 如果设置为 "None"，则完全禁用服务类型 |
| global.imagePullSecrets | list | [] | 所有 ServiceAccount 的 ImagePullSecrets，在同一名字空间下引用这个 ServiceAccount 的 Pod 中使用的 secret 列表。对于不使用 ServiceAccount（例如 grafana, servicegraph, tracing）的组件，ImagePullSecrets 将被添加到相对应 Deployment(StatefulSet) 对象中。必须为任何配置私有 Docker 注册表的集群设置。|
| ...（继续其他参数） |

以上内容展示了对 Higress 部署的相关选项及详细的配置参数说明。
