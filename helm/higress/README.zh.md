## Higress for Kubernetes

Higress 是基于阿里巴巴内部网关实践的云原生 API 网关。

通过基于 Istio 和 Envoy，Higress 实现了流量网关、微服务网关和安全网关的三重网关架构的集成，从而大大降低了部署、运维的成本。

## 设置仓库信息

```console
helm repo add higress.io https://higress.io/helm-charts
helm repo update
```

## 安装

使用名为 `higress` 的版本来安装 chart：

```console
helm install higress -n higress-system higress.io/higress --create-namespace --render-subchart-notes
```

## 卸载

卸载删除 higress 部署：

```console
helm delete higress -n higress-system
```

该命令会删除与 chart 相关的所有 Kubernetes 组件并删除发行版。

## 参数

## 配置值

| 键名 | 类型 | 默认值 | 描述 |
|------|------|---------|-------------|
| clusterName | string | `""` |  |
| controller.affinity | object | `{}` |  |
| controller.automaticHttps.email | string | `""` |  |
| controller.automaticHttps.enabled | bool | `true` |  |
| controller.autoscaling.enabled | bool | `false` |  |
| controller.autoscaling.maxReplicas | int | `5` |  |
| controller.autoscaling.minReplicas | int | `1` |  |
| controller.autoscaling.targetCPUUtilizationPercentage | int | `80` |  |
| controller.env | object | `{}` |  |
| controller.hub | string | `"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress"` |  |
| controller.image | string | `"higress"` |  |
| controller.imagePullSecrets | list | `[]` |  |
| controller.labels | object | `{}` |  |
| controller.name | string | `"higress-controller"` |  |
| controller.nodeSelector | object | `{}` |  |
| controller.podAnnotations | object | `{}` |  |
| controller.podLabels | object | `{}` | 应用到 pod 上的标签 |
| controller.podSecurityContext | object | `{}` |  |
| controller.ports[0].name | string | `"http"` |  |
| controller.ports[0].port | int | `8888` |  |
| controller.ports[0].protocol | string | `"TCP"` |  |
| controller.ports[0].targetPort | int | `8888` |  |
| controller.probe.httpGet.path | string | `"/ready"` |  |
| controller.probe.httpGet.port | int | `8888` |  |
| controller.probe.initialDelaySeconds | int | `1` |  |
| controller.probe.periodSeconds | int | `3` |  |
| controller.probe.timeoutSeconds | int | `5` |  |
| controller.rbac.create | bool | `true` |  |
| controller.replicas | int | `1` | Higress Controller pods 的数量 |
| controller.resources.limits.cpu | string | `"1000m"` |  |
| controller.resources.limits.memory | string | `"2048Mi"` |  |
| controller.resources.requests.cpu | string | `"500m"` |  |
| controller.resources.requests.memory | string | `"2048Mi"` |  |
| gateway.metrics.enabled | bool | `false` | 如果为 true，则为gateway创建PodMonitor或VMPodScrape |
| gateway.metrics.provider | string | `monitoring.coreos.com` | CustomResourceDefinition 的提供商组名，可以是 monitoring.coreos.com 或 operator.victoriametrics.com |
| gateway.readinessFailureThreshold | int | `30` | 成功进行探针测试前连续失败探针的最大次数。 |
| global MeshNetworks | object | `{}` |  |
| global.tracer.datadog.address | string | `"$(HOST_IP):8126"` | 提交给 Datadog agent 的 Host:Port 。|
| redis.redis.persistence.enabled | bool | `false` | 启用 Redis 持久性，默认为 false |
| redis.redis.persistence.size | string | `"1Gi"` | Persistent Volume 大小 |
| redis.redis.service.port | int | `6379` | Exporter service 端口 |
| tracing.skywalking.port | int | `11800` |  |
| upstream.connectionBufferLimits | int | `10485760` | 上游连接缓冲限制（字节）|
