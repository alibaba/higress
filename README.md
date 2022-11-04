<h1 align="center">
    <img src="https://img.alicdn.com/imgextra/i2/O1CN01NwxLDd20nxfGBjxmZ_!!6000000006895-2-tps-960-290.png" alt="Higress" width="240" height="72.5">
  <br>
  易使用、易扩展、热更新的下一代云原生网关
</h1>

<p>
   <a href="README_EN.md"> English <a/> | 中文
</p>

Higress 是基于阿里内部两年多的 Envoy Gateway 实践沉淀，以开源 [Istio](https://github.com/istio/istio) 与 [Envoy](https://github.com/envoyproxy/envoy) 为核心构建的下一代云原生网关。Higress 实现了安全防护网关、流量网关、微服务网关三层网关合一，从而显著降低了网关的部署和运维成本。

<BR><center><img src="https://img.alicdn.com/imgextra/i4/O1CN01dqXHDi27RhjAtZyNp_!!6000000007794-0-tps-1794-1446.jpg" alt="Higress Architecture"></center>


## Summary

- [**使用场景**](#使用场景)
- [**核心优势**](#核心优势)
- [**Quick Start**](#quick-start)

## 使用场景

- **Kubernetes Ingress 网关**: 

  Higress 可以作为 K8s 集群的 Ingress 入口网关, 并且兼容了大量 K8s Nginx Ingress 的注解，可以从 K8s Nginx Ingress 快速平滑迁移到 Higress。
  
  [Gateway API](https://gateway-api.sigs.k8s.io/) 标准的支持正在开发中，将会支持用户从 Ingress API 平滑迁移到 Gateway API。
  
- **微服务网关**: 

  Higress 可以作为微服务网关, 能够对接多种类型的注册中心发现服务配置路由，例如 Nacos, ZooKeeper, Consul 等。
  
  并且深度集成了 [Dubbo](https://github.com/apache/dubbo), [Nacos](https://github.com/alibaba/nacos), [Sentinel](https://github.com/alibaba/Sentinel) 等微服务技术栈，基于 Envoy C++ 网关内核的出色性能，相比传统 Java 类微服务网关，可以显著降低资源使用率，减少成本。
  
- **安全防护网关**:

  Higress 可以作为安全防护网关， 提供 WAF 的能力，并且支持多种认证鉴权策略，例如 key-auth, hmac-auth, jwt-auth, basic-auth, oidc 等。  
  

## 核心优势

- **生产等级**

  脱胎于历经阿里巴巴2年多生产验证的内部产品，支持每秒请求量达数十万级的大规模场景，具备企业级 SLA 的开源产品
  
- **平滑演进**

  支持 Nacos/Zookeeper 等多种注册中心，可以不依赖 K8s Service 进行服务发现，支持传统非容器架构业务平滑过渡到云原生架构

  支持从 Nginx Ingress Controller 平滑迁移，支持从 Ingress API 平滑过渡到 Gateway API，基于 Istio 支持平滑演进到 ServiceMesh 架构

- **动态热更新**
  
  彻底摆脱 reload 引起的流量抖动，配置变更毫秒级生效且业务无感，Wasm 插件热更新且流量无损
  
- **便于扩展**
  
  提供 Wasm、Lua、进程外三种插件扩展机制，让多语言编写插件不再成为障碍，插件生效粒度既支持全局级、域名级，也支持细粒度的路由级
  
- **兼收并蓄**
  
  兼容 Nginx Ingress Annotation 80%+ 的使用场景，且提供功能更丰富的 Higress Annotation 注解
  
  兼容 Ingress API/Gateway API/Istio API，可以组合多种 CRD 实现流量精细化管理  


## Quick Start
- [**本地环境**](#本地环境)
- [**生产环境**](#生产环境)


### 本地环境


#### 第一步、 安装 kubectl & kind

**MacOS：**
```bash
curl -Lo ./kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/darwin/amd64/kubectl
# for Intel Macs
[ $(uname -m) = x86_64 ]&& curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.17.0/kind-darwin-amd64
# for M1 / ARM Macs
[ $(uname -m) = arm64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.17.0/kind-darwin-arm64
chmod +x ./kind ./kubectl
mv ./kind ./kubectl /some-dir-in-your-PATH/
```

**Windows 中使用 PowerShell:**
```bash
curl.exe -Lo kubectl.exe https://storage.googleapis.com/kubernetes-release/release/$(curl.exe -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/windows/amd64/kubectl.exe
curl.exe -Lo kind-windows-amd64.exe https://kind.sigs.k8s.io/dl/v0.17.0/kind-windows-amd64
Move-Item .\kind-windows-amd64.exe c:\some-dir-in-your-PATH\kind.exe
Move-Item .\kubectl.exe c:\some-dir-in-your-PATH\kubectl.exe
```

**Linux:**
```bash
curl -Lo ./kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.17.0/kind-linux-amd64
chmod +x ./kind ./kubectl
sudo mv ./kind ./kubectl /usr/local/bin/kind
```

#### 第二步、 创建并启用 kind 

首先创建一个集群配置文件: `cluster.conf`

```yaml
# cluster.conf
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
```
Mac & Linux 系统执行:
```bash
kind create cluster --name higress --config=cluster.conf
kubectl cluster-info --context higress
```
Windows 系统执行:
```bash
kind.exe create cluster --name higress --config=cluster.conf
kubectl.exe cluster-info --context higress
```

#### 第三步、 安装 istio & higress

```bash
helm install istio -n istio-system oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/charts/istio-local
helm install higress -n higress-system oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/charts/higress-local
```

#### 第四步、 创建 Ingress 资源并测试

```bash
kubectl apply -f https://kind.sigs.k8s.io/examples/ingress/usage.yaml
```

测试 Ingress 生效：

```bash
# should output "foo"
curl localhost/foo
# should output "bar"
curl localhost/bar
```


### 生产环境

#### 第一步、 安装 istio

可以选择安装 higress 发行的 istio 版本: 
```bash
kubectl create ns istio-system
helm install istio -n istio-system oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/charts/istio
```

或者选择安装官方 istio 版本 (将失去部分能力，例如通过 Ingress 注解实现限流的功能):
    
https://istio.io/latest/docs/setup/install

#### 第二步、 安装 higress

```bash
kubectl create ns higress-system
helm install higress -n higress-system oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/charts/higress 
```

#### 第三步、 创建 Ingress 资源并测试

假设在 default 命名空间下已经部署了一个 test service，服务端口为 80 ，则创建下面这个 K8s Ingress
    
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: simple-example
spec:
  rules:
  - host: foo.bar.com
    http:
      paths:
      - path: /foo
        pathType: Prefix
        backend:
          service:
            name: test
            port:
              number: 80  
```

测试能访问到该服务：
    
```bash
curl "$(k get svc -n higress-system higress-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"/foo -H 'host: foo.bar.com'
```    

