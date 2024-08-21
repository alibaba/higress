<h1 align="center">
    <img src="https://img.alicdn.com/imgextra/i2/O1CN01NwxLDd20nxfGBjxmZ_!!6000000006895-2-tps-960-290.png" alt="Higress" width="240" height="72.5">
  <br>
  AI Gateway
</h1>
<h4 align="center"> AI Native API Gateway </h4>

[![Build Status](https://github.com/alibaba/higress/actions/workflows/build-and-test.yaml/badge.svg?branch=main)](https://github.com/alibaba/higress/actions)
[![license](https://img.shields.io/github/license/alibaba/higress.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

[**官网**](https://higress.io/) &nbsp; |
&nbsp; [**文档**](https://higress.io/docs/latest/user/quickstart/) &nbsp; |
&nbsp; [**博客**](https://higress.io/blog/) &nbsp; |
&nbsp; [**开发指引**](https://higress.io/docs/latest/dev/architecture/) &nbsp; |
&nbsp; [**AI插件**](https://higress.io/plugin/) &nbsp;


<p>
   <a href="README_EN.md"> English <a/> | 中文
</p>


Higress 是基于阿里内部多年的 Envoy Gateway 实践沉淀，以开源 [Istio](https://github.com/istio/istio) 与 [Envoy](https://github.com/envoyproxy/envoy) 为核心构建的云原生 API 网关。

Higress 在阿里内部作为 AI 网关，承载了通义千问 APP、百炼大模型 API、机器学习 PAI 平台等 AI 业务的流量。

Higress 能够用统一的协议对接国内外所有 LLM 模型厂商，同时具备丰富的 AI 可观测、多模型负载均衡/fallback、AI token 流控、AI 缓存等能力：

![](https://img.alicdn.com/imgextra/i1/O1CN01fNnhCp1cV8mYPRFeS_!!6000000003605-0-tps-1080-608.jpg)



## Summary

- [**快速开始**](#快速开始)    
- [**功能展示**](#功能展示)
- [**使用场景**](#使用场景)
- [**核心优势**](#核心优势)
- [**社区**](#社区)

## 快速开始

Higress 只需 Docker 即可启动，方便个人开发者在本地搭建学习，或者用于搭建简易站点:

```bash
# 创建一个工作目录
mkdir higress; cd higress
# 启动 higress，配置文件会写到工作目录下
docker run -d --rm --name higress-ai -v ${PWD}:/data \
        -p 8001:8001 -p 8080:8080 -p 8443:8443  \
        higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one:latest
```

监听端口说明如下：

- 8001 端口：Higress UI 控制台入口
- 8080 端口：网关 HTTP 协议入口
- 8443 端口：网关 HTTPS 协议入口

**Higress 的所有 Docker 镜像都一直使用自己独享的仓库，不受 Docker Hub 境内不可访问的影响**

K8s 下使用 Helm 部署等其他安装方式可以参考官网 [Quick Start 文档](https://higress.io/docs/latest/user/quickstart/)。


## 使用场景

- **AI 网关**:

  Higress 提供了一站式的 AI 插件集，可以增强依赖 AI 能力业务的稳定性、灵活性、可观测性，使得业务与 AI 的集成更加便捷和高效。

- **Kubernetes Ingress 网关**:

  Higress 可以作为 K8s 集群的 Ingress 入口网关, 并且兼容了大量 K8s Nginx Ingress 的注解，可以从 K8s Nginx Ingress 快速平滑迁移到 Higress。
  
  支持 [Gateway API](https://gateway-api.sigs.k8s.io/) 标准，支持用户从 Ingress API 平滑迁移到 Gateway API。
  
- **微服务网关**:

  Higress 可以作为微服务网关, 能够对接多种类型的注册中心发现服务配置路由，例如 Nacos, ZooKeeper, Consul, Eureka 等。
  
  并且深度集成了 [Dubbo](https://github.com/apache/dubbo), [Nacos](https://github.com/alibaba/nacos), [Sentinel](https://github.com/alibaba/Sentinel) 等微服务技术栈，基于 Envoy C++ 网关内核的出色性能，相比传统 Java 类微服务网关，可以显著降低资源使用率，减少成本。
  
- **安全防护网关**:

  Higress 可以作为安全防护网关， 提供 WAF 的能力，并且支持多种认证鉴权策略，例如 key-auth, hmac-auth, jwt-auth, basic-auth, oidc 等。  

## 核心优势

- **生产等级**

  脱胎于阿里巴巴2年多生产验证的内部产品，支持每秒请求量达数十万级的大规模场景。

  彻底摆脱 Nginx reload 引起的流量抖动，配置变更毫秒级生效且业务无感。对 AI 业务等长连接场景特别友好。

- **流式处理**

  支持真正的完全流式处理请求/响应 Body，Wasm 插件很方便地自定义处理 SSE （Server-Sent Events）等流式协议的报文。

  在 AI 业务等大带宽场景下，可以显著降低内存开销。  
    
- **便于扩展**
  
  提供丰富的官方插件库，涵盖 AI、流量管理、安全防护等常用功能，满足90%以上的业务场景需求。

  主打 Wasm 插件扩展，通过沙箱隔离确保内存安全，支持多种编程语言，允许插件版本独立升级，实现流量无损热更新网关逻辑。

- **安全易用**
  
  基于 Ingress API 和 Gateway API 标准，提供开箱即用的 UI 控制台，WAF 防护插件、IP/Cookie CC 防护插件开箱即用。

  支持对接 Let's Encrypt 自动签发和续签免费证书，并且可以脱离 K8s 部署，一行 Docker 命令即可启动，方便个人开发者使用。


## 功能展示

### AI 网关 Demo 展示

[从 OpenAI 到其他大模型，30 秒完成迁移
](https://www.bilibili.com/video/BV1dT421a7w7/?spm_id_from=333.788.recommend_more_video.14)


### Higress UI 控制台
    
- **丰富的可观测**

  提供开箱即用的可观测，Grafana&Prometheus 可以使用内置的也可对接自建的

  ![](./docs/images/monitor.gif)
    

- **插件扩展机制**

  官方提供了多种插件，用户也可以[开发](./plugins/wasm-go)自己的插件，构建成 docker/oci 镜像后在控制台配置，可以实时变更插件逻辑，对流量完全无损。

  ![](./docs/images/plugin.gif)


- **多种服务发现**

  默认提供 K8s Service 服务发现，通过配置可以对接 Nacos/ZooKeeper 等注册中心实现服务发现，也可以基于静态 IP 或者 DNS 来发现

  ![](./docs/images/service-source.gif)
    

- **域名和证书**

  可以创建管理 TLS 证书，并配置域名的 HTTP/HTTPS 行为，域名策略里支持对特定域名生效插件

  ![](./docs/images/domain.gif)


- **丰富的路由能力**

  通过上面定义的服务发现机制，发现的服务会出现在服务列表中；创建路由时，选择域名，定义路由匹配机制，再选择目标服务进行路由；路由策略里支持对特定路由生效插件

  ![](./docs/images/route-service.gif)


## 社区

### 感谢

如果没有 Envoy 和 Istio 的开源工作，Higress 就不可能实现，在这里向这两个项目献上最诚挚的敬意。

### 交流群

![image](https://img.alicdn.com/imgextra/i2/O1CN01qPd7Ix1uZPVEsWjWp_!!6000000006051-0-tps-720-405.jpg)

### 技术分享

微信公众号：

![](https://img.alicdn.com/imgextra/i1/O1CN01WnQt0q1tcmqVDU73u_!!6000000005923-0-tps-258-258.jpg)

