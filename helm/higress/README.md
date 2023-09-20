# Higress Helm Chart

Installs the cloud-native gateway [Higress](http://higress.io/)

## Get Repo Info

```console
helm repo add higress.io https://higress.io/helm-charts
helm repo update
```

_See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation._

## Installing the Chart

To install the chart with the release name `higress`:

```console
helm install higress -n higress-system higress.io/higress --create-namespace --render-subchart-notes
```

## Uninstalling the Chart

To uninstall/delete the higress deployment:

```console
helm delete higress -n higress-system
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

| **Parameter** | **Description** | **Default** |
|---|---|---|
| **Global Parameters** |  |  |
| global.local | Set to `true` if installing to a local K8s cluster (e.g.: Kind, Rancher Desktop, etc.) | false |
| global.ingressClass | [IngressClass](https://kubernetes.io/zh-cn/docs/concepts/services-networking/ingress/#ingress-class) which is used to filter Ingress resources Higress Controller watches.<br />If there are multiple gateway instances deployed in the cluster, this parameter can be used to distinguish the scope of each gateway instance.<br />There are some special cases for special IngressClass values:<br />1. If set to "nginx", Higress Controller will watch Ingress resources with the `nginx` IngressClass or without any Ingress class.<br />2. If set to empty, Higress Controller will watch all Ingress resources in the K8s cluster. | higress |
| global.watchNamespace | If not empty, Higress Controller will only watch resources in the specified namespace. When isolating different business systems using K8s namespace, if each namespace requires a standalone gateway instance, this parameter can be used to confine the Ingress watching of Higress within the given namespace. | "" |
| global.disableAlpnH2 | Whether to disable HTTP/2 in ALPN | true |
| global.enableStatus | If `true`, Higress Controller will update the `status` field of Ingress resources.<br />When migrating from Nginx Ingress, in order to avoid `status` field of Ingress objects being overwritten, this parameter needs to be set to false, so Higress won't write the entry IP to the `status` field of the corresponding Ingress object. | true |
| global.enableIstioAPI | If `true`, Higress Controller will monitor istio resources as well | false |
| global.enableGatewayAPI | If `true`, Higress Controller will monitor Gateway API resources as well | false |
| global.istioNamespace | The namespace istio is installed to | istio-system |
| **Core Paramters** |  |  |
| higress-core.gateway.replicas | Number of Higress Gateway pods | 2 |
| higress-core.controller.replicas | Number of Higress Controller pods | 1 |
| **Console Paramters** |  |  |
| higress-console.replicaCount | Number of Higress Console pods | 1 |
| higress-console.service.type | K8s service type used by Higress Console | ClusterIP |
| higress-console.domain | Domain used to access Higress Console | console.higress.io |
| higress-console.tlsSecretName | Name of Secret resource used by TLS connections. | "" |
| higress-console.web.login.prompt | Prompt message to be displayed on the login page | "" |
| higress-console.admin.password.value | If not empty, the admin password will be configured to the specified value. | "" |
| higress-console.admin.password.length | The length of random admin password generated during installation. Only works when `higress-console.admin.password.value` is not set. | 8 |
| higress-console.o11y.enabled | If `true`, o11y suite (Grafana + Promethues) will be installed. | false |
| higress-console.pvc.rwxSupported | Set to `false` when installing to a standard K8s cluster and the target cluster doesn't support the ReadWriteMany access mode of PersistentVolumeClaim. | true |
