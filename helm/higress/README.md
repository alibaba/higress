## Higress for Kubernetes

Higress is a cloud-native api gateway based on Alibaba's internal gateway practices.

Powered by Istio and Envoy, Higress realizes the integration of the triple gateway architecture of traffic gateway, microservice gateway and security gateway, thereby greatly reducing the costs of deployment, operation and maintenance.

## Prerequisites

* Kubernetes v1.14+
* Helm v3+

## Get Repo Info

```console
helm repo add higress.io https://higress.io/helm-charts
helm repo update
```

## Install

To install the chart with the release name `higress`:

```console
helm install higress -n higress-system higress.io/higress --create-namespace --render-subchart-notes
```

## Uninstall

To uninstall/delete the higress deployment:

```console
helm delete higress -n higress-system
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
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
| controller.podSecurityContext | object | `{}` |  |
| controller.ports[0].name | string | `"http"` |  |
| controller.ports[0].port | int | `8888` |  |
| controller.ports[0].protocol | string | `"TCP"` |  |
| controller.ports[0].targetPort | int | `8888` |  |
| controller.ports[1].name | string | `"http-solver"` |  |
| controller.ports[1].port | int | `8889` |  |
| controller.ports[1].protocol | string | `"TCP"` |  |
| controller.ports[1].targetPort | int | `8889` |  |
| controller.ports[2].name | string | `"grpc"` |  |
| controller.ports[2].port | int | `15051` |  |
| controller.ports[2].protocol | string | `"TCP"` |  |
| controller.ports[2].targetPort | int | `15051` |  |
| controller.probe.httpGet.path | string | `"/ready"` |  |
| controller.probe.httpGet.port | int | `8888` |  |
| controller.probe.initialDelaySeconds | int | `1` |  |
| controller.probe.periodSeconds | int | `3` |  |
| controller.probe.timeoutSeconds | int | `5` |  |
| controller.rbac.create | bool | `true` |  |
| controller.replicas | int | `1` |  |
| controller.resources.limits.cpu | string | `"1000m"` |  |
| controller.resources.limits.memory | string | `"2048Mi"` |  |
| controller.resources.requests.cpu | string | `"500m"` |  |
| controller.resources.requests.memory | string | `"2048Mi"` |  |
| controller.securityContext | object | `{}` |  |
| controller.service.type | string | `"ClusterIP"` |  |
| controller.serviceAccount.annotations | object | `{}` |  |
| controller.serviceAccount.create | bool | `true` |  |
| controller.serviceAccount.name | string | `""` |  |
| controller.tag | string | `""` |  |
| controller.tolerations | list | `[]` |  |
| downstream.connectionBufferLimits | int | `32768` |  |
| downstream.http2.initialConnectionWindowSize | int | `1048576` |  |
| downstream.http2.initialStreamWindowSize | int | `65535` |  |
| downstream.http2.maxConcurrentStreams | int | `100` |  |
| downstream.idleTimeout | int | `180` |  |
| downstream.maxRequestHeadersKb | int | `60` |  |
| downstream.routeTimeout | int | `0` |  |
| gateway.affinity | object | `{}` |  |
| gateway.annotations | object | `{}` |  |
| gateway.autoscaling.enabled | bool | `false` |  |
| gateway.autoscaling.maxReplicas | int | `5` |  |
| gateway.autoscaling.minReplicas | int | `1` |  |
| gateway.autoscaling.targetCPUUtilizationPercentage | int | `80` |  |
| gateway.containerSecurityContext | string | `nil` |  |
| gateway.env | object | `{}` |  |
| gateway.hostNetwork | bool | `false` |  |
| gateway.httpPort | int | `80` |  |
| gateway.httpsPort | int | `443` |  |
| gateway.hub | string | `"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress"` |  |
| gateway.image | string | `"gateway"` |  |
| gateway.kind | string | `"Deployment"` | Use a `DaemonSet` or `Deployment` |
| gateway.labels | object | `{}` |  |
| gateway.metrics.enabled | bool | `false` |  |
| gateway.metrics.honorLabels | bool | `false` |  |
| gateway.metrics.interval | string | `""` |  |
| gateway.metrics.metricRelabelConfigs | list | `[]` |  |
| gateway.metrics.metricRelabelings | list | `[]` |  |
| gateway.metrics.provider | string | `"monitoring.coreos.com"` |  |
| gateway.metrics.rawSpec | object | `{}` |  |
| gateway.metrics.relabelConfigs | list | `[]` |  |
| gateway.metrics.relabelings | list | `[]` |  |
| gateway.metrics.scrapeTimeout | string | `""` |  |
| gateway.name | string | `"higress-gateway"` |  |
| gateway.networkGateway | string | `""` |  |
| gateway.nodeSelector | object | `{}` |  |
| gateway.podAnnotations."prometheus.io/path" | string | `"/stats/prometheus"` |  |
| gateway.podAnnotations."prometheus.io/port" | string | `"15020"` |  |
| gateway.podAnnotations."prometheus.io/scrape" | string | `"true"` |  |
| gateway.podAnnotations."sidecar.istio.io/inject" | string | `"false"` |  |
| gateway.rbac.enabled | bool | `true` |  |
| gateway.readinessFailureThreshold | int | `30` |  |
| gateway.readinessInitialDelaySeconds | int | `1` |  |
| gateway.readinessPeriodSeconds | int | `2` |  |
| gateway.readinessSuccessThreshold | int | `1` |  |
| gateway.readinessTimeoutSeconds | int | `3` |  |
| gateway.replicas | int | `2` |  |
| gateway.resources.limits.cpu | string | `"2000m"` |  |
| gateway.resources.limits.memory | string | `"2048Mi"` |  |
| gateway.resources.requests.cpu | string | `"2000m"` |  |
| gateway.resources.requests.memory | string | `"2048Mi"` |  |
| gateway.revision | string | `""` |  |
| gateway.rollingMaxSurge | string | `"100%"` |  |
| gateway.rollingMaxUnavailable | string | `"25%"` |  |
| gateway.securityContext | string | `nil` |  |
| gateway.service.annotations | object | `{}` |  |
| gateway.service.externalTrafficPolicy | string | `""` |  |
| gateway.service.loadBalancerClass | string | `""` |  |
| gateway.service.loadBalancerIP | string | `""` |  |
| gateway.service.loadBalancerSourceRanges | list | `[]` |  |
| gateway.service.ports[0].name | string | `"http2"` |  |
| gateway.service.ports[0].port | int | `80` |  |
| gateway.service.ports[0].protocol | string | `"TCP"` |  |
| gateway.service.ports[0].targetPort | int | `80` |  |
| gateway.service.ports[1].name | string | `"https"` |  |
| gateway.service.ports[1].port | int | `443` |  |
| gateway.service.ports[1].protocol | string | `"TCP"` |  |
| gateway.service.ports[1].targetPort | int | `443` |  |
| gateway.service.type | string | `"LoadBalancer"` |  |
| gateway.serviceAccount.annotations | object | `{}` |  |
| gateway.serviceAccount.create | bool | `true` |  |
| gateway.serviceAccount.name | string | `""` |  |
| gateway.tag | string | `""` |  |
| gateway.tolerations | list | `[]` |  |
| global.autoscalingv2API | bool | `true` |  |
| global.caAddress | string | `""` |  |
| global.caName | string | `""` |  |
| global.configCluster | bool | `false` |  |
| global.defaultPodDisruptionBudget.enabled | bool | `false` |  |
| global.defaultResources.requests.cpu | string | `"10m"` |  |
| global.defaultUpstreamConcurrencyThreshold | int | `10000` |  |
| global.disableAlpnH2 | bool | `false` |  |
| global.enableGatewayAPI | bool | `false` |  |
| global.enableH3 | bool | `false` |  |
| global.enableHigressIstio | bool | `false` |  |
| global.enableIPv6 | bool | `false` |  |
| global.enableIstioAPI | bool | `true` |  |
| global.enableProxyProtocol | bool | `false` |  |
| global.enableSRDS | bool | `true` |  |
| global.enableStatus | bool | `true` |  |
| global.externalIstiod | bool | `false` |  |
| global.hostRDSMergeSubset | bool | `false` |  |
| global.hub | string | `"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress"` |  |
| global.imagePullPolicy | string | `""` |  |
| global.imagePullSecrets | list | `[]` |  |
| global.ingressClass | string | `"higress"` |  |
| global.istioNamespace | string | `"istio-system"` |  |
| global.istiod.enableAnalysis | bool | `false` |  |
| global.jwtPolicy | string | `"third-party-jwt"` |  |
| global.kind | bool | `false` |  |
| global.liteMetrics | bool | `true` |  |
| global.local | bool | `false` |  |
| global.logAsJson | bool | `false` |  |
| global.logging.level | string | `"default:info"` |  |
| global.meshID | string | `""` |  |
| global.meshNetworks | object | `{}` |  |
| global.mountMtlsCerts | bool | `false` |  |
| global.multiCluster.clusterName | string | `""` |  |
| global.multiCluster.enabled | bool | `true` |  |
| global.network | string | `""` |  |
| global.o11y.enabled | bool | `false` |  |
| global.o11y.promtail.image.repository | string | `"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/promtail"` |  |
| global.o11y.promtail.image.tag | string | `"2.9.4"` |  |
| global.o11y.promtail.port | int | `3101` |  |
| global.o11y.promtail.resources.limits.cpu | string | `"500m"` |  |
| global.o11y.promtail.resources.limits.memory | string | `"2Gi"` |  |
| global.o11y.promtail.securityContext | object | `{}` |  |
| global.omitSidecarInjectorConfigMap | bool | `false` |  |
| global.onDemandRDS | bool | `false` |  |
| global.oneNamespace | bool | `false` |  |
| global.onlyPushRouteCluster | bool | `true` |  |
| global.operatorManageWebhooks | bool | `false` |  |
| global.pilotCertProvider | string | `"istiod"` |  |
| global.priorityClassName | string | `""` |  |
| global.proxy.autoInject | string | `"enabled"` |  |
| global.proxy.clusterDomain | string | `"cluster.local"` |  |
| global.proxy.componentLogLevel | string | `"misc:error"` |  |
| global.proxy.enableCoreDump | bool | `false` |  |
| global.proxy.excludeIPRanges | string | `""` |  |
| global.proxy.excludeInboundPorts | string | `""` |  |
| global.proxy.excludeOutboundPorts | string | `""` |  |
| global.proxy.holdApplicationUntilProxyStarts | bool | `false` |  |
| global.proxy.image | string | `"proxyv2"` |  |
| global.proxy.includeIPRanges | string | `"*"` |  |
| global.proxy.includeInboundPorts | string | `"*"` |  |
| global.proxy.includeOutboundPorts | string | `""` |  |
| global.proxy.logLevel | string | `"warning"` |  |
| global.proxy.privileged | bool | `false` |  |
| global.proxy.readinessFailureThreshold | int | `30` |  |
| global.proxy.readinessInitialDelaySeconds | int | `1` |  |
| global.proxy.readinessPeriodSeconds | int | `2` |  |
| global.proxy.readinessSuccessThreshold | int | `30` |  |
| global.proxy.readinessTimeoutSeconds | int | `3` |  |
| global.proxy.resources.limits.cpu | string | `"2000m"` |  |
| global.proxy.resources.limits.memory | string | `"1024Mi"` |  |
| global.proxy.resources.requests.cpu | string | `"100m"` |  |
| global.proxy.resources.requests.memory | string | `"128Mi"` |  |
| global.proxy.statusPort | int | `15020` |  |
| global.proxy.tracer | string | `""` |  |
| global.proxy_init.image | string | `"proxyv2"` |  |
| global.proxy_init.resources.limits.cpu | string | `"2000m"` |  |
| global.proxy_init.resources.limits.memory | string | `"1024Mi"` |  |
| global.proxy_init.resources.requests.cpu | string | `"10m"` |  |
| global.proxy_init.resources.requests.memory | string | `"10Mi"` |  |
| global.remotePilotAddress | string | `""` |  |
| global.sds.token.aud | string | `"istio-ca"` |  |
| global.sts.servicePort | int | `0` |  |
| global.tracer.datadog.address | string | `"$(HOST_IP):8126"` |  |
| global.tracer.lightstep.accessToken | string | `""` |  |
| global.tracer.lightstep.address | string | `""` |  |
| global.tracer.stackdriver.debug | bool | `false` |  |
| global.tracer.stackdriver.maxNumberOfAnnotations | int | `200` |  |
| global.tracer.stackdriver.maxNumberOfAttributes | int | `200` |  |
| global.tracer.stackdriver.maxNumberOfMessageEvents | int | `200` |  |
| global.useMCP | bool | `false` |  |
| global.watchNamespace | string | `""` |  |
| global.xdsMaxRecvMsgSize | string | `"104857600"` |  |
| hub | string | `"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress"` |  |
| meshConfig.enablePrometheusMerge | bool | `true` |  |
| meshConfig.rootNamespace | string | `nil` |  |
| meshConfig.trustDomain | string | `"cluster.local"` |  |
| pilot.autoscaleEnabled | bool | `false` |  |
| pilot.autoscaleMax | int | `5` |  |
| pilot.autoscaleMin | int | `1` |  |
| pilot.configMap | bool | `true` |  |
| pilot.configSource.subscribedResources | list | `[]` |  |
| pilot.cpu.targetAverageUtilization | int | `80` |  |
| pilot.deploymentLabels | object | `{}` |  |
| pilot.enableProtocolSniffingForInbound | bool | `true` |  |
| pilot.enableProtocolSniffingForOutbound | bool | `true` |  |
| pilot.env.PILOT_ENABLE_CROSS_CLUSTER_WORKLOAD_ENTRY | string | `"false"` |  |
| pilot.env.PILOT_ENABLE_METADATA_EXCHANGE | string | `"false"` |  |
| pilot.env.PILOT_SCOPE_GATEWAY_TO_NAMESPACE | string | `"false"` |  |
| pilot.env.VALIDATION_ENABLED | string | `"false"` |  |
| pilot.hub | string | `"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress"` |  |
| pilot.image | string | `"pilot"` |  |
| pilot.jwksResolverExtraRootCA | string | `""` |  |
| pilot.keepaliveMaxServerConnectionAge | string | `"30m"` |  |
| pilot.nodeSelector | object | `{}` |  |
| pilot.plugins | list | `[]` |  |
| pilot.podAnnotations | object | `{}` |  |
| pilot.podLabels | object | `{}` |  |
| pilot.replicaCount | int | `1` |  |
| pilot.resources.requests.cpu | string | `"500m"` |  |
| pilot.resources.requests.memory | string | `"2048Mi"` |  |
| pilot.rollingMaxSurge | string | `"100%"` |  |
| pilot.rollingMaxUnavailable | string | `"25%"` |  |
| pilot.serviceAnnotations | object | `{}` |  |
| pilot.tag | string | `""` |  |
| pilot.traceSampling | float | `1` |  |
| revision | string | `""` |  |
| tracing.enable | bool | `false` |  |
| tracing.sampling | int | `100` |  |
| tracing.skywalking.port | int | `11800` |  |
| tracing.skywalking.service | string | `""` |  |
| tracing.timeout | int | `500` |  |
| upstream.connectionBufferLimits | int | `10485760` |  |
| upstream.idleTimeout | int | `10` |  |