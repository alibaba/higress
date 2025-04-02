<a name="readme-top"></a>
<h1 align="center">
    <img src="https://img.alicdn.com/imgextra/i2/O1CN01NwxLDd20nxfGBjxmZ_!!6000000006895-2-tps-960-290.png" alt="Higress" width="240" height="72.5">
  <br>
  AI Gateway
</h1>
<h4 align="center"> AI Native API Gateway </h4>

<div align="center">
    
[![Build Status](https://github.com/alibaba/higress/actions/workflows/build-and-test.yaml/badge.svg?branch=main)](https://github.com/alibaba/higress/actions)
[![license](https://img.shields.io/github/license/alibaba/higress.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

<a href="https://trendshift.io/repositories/10918" target="_blank"><img src="https://trendshift.io/api/badge/repositories/10918" alt="alibaba%2Fhigress | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>
</div>

[**Official Site**](https://higress.io/en-us/) &nbsp; |
&nbsp; [**Docs**](https://higress.io/en-us/docs/overview/what-is-higress) &nbsp; |
&nbsp; [**Blog**](https://higress.io/en-us/blog) &nbsp; |
&nbsp; [**Developer**](https://higress.io/en-us/docs/developers/developers_dev) &nbsp; |
&nbsp; [**Higress in Cloud**](https://www.alibabacloud.com/product/microservices-engine?spm=higress-website.topbar.0.0.0) &nbsp;


<p>
   English | <a href="README_ZH.md">中文<a/> | <a href="README_JP.md">日本語<a/>
</p>

Higress is a cloud-native API gateway based on Istio and Envoy, which can be extended with Wasm plugins written in Go/Rust/JS. It provides dozens of ready-to-use general-purpose plugins and an out-of-the-box console (try the [demo here](http://demo.higress.io/)).

Higress was born within Alibaba to solve the issues of Tengine reload affecting long-connection services and insufficient load balancing capabilities for gRPC/Dubbo.

Alibaba Cloud has built its cloud-native API gateway product based on Higress, providing 99.99% gateway high availability guarantee service capabilities for a large number of enterprise customers.

Higress's AI gateway capabilities support all [mainstream model providers](https://github.com/alibaba/higress/tree/main/plugins/wasm-go/extensions/ai-proxy/provider) both domestic and international, as well as self-built DeepSeek models based on vllm/ollama. Within Alibaba Cloud, it supports AI businesses such as Tongyi Qianwen APP, Bailian large model API, and machine learning PAI platform. It also serves leading AIGC enterprises (such as Zero One Infinite) and AI products (such as FastGPT).

## Summary

- [**Quick Start**](#quick-start)    
- [**Feature Showcase**](#feature-showcase)
- [**Use Cases**](#use-cases)
- [**Core Advantages**](#core-advantages)
- [**Community**](#community)

## Quick Start

Higress can be started with just Docker, making it convenient for individual developers to set up locally for learning or for building simple sites:

```bash
# Create a working directory
mkdir higress; cd higress
# Start higress, configuration files will be written to the working directory
docker run -d --rm --name higress-ai -v ${PWD}:/data \
        -p 8001:8001 -p 8080:8080 -p 8443:8443  \
        higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one:latest
```

Port descriptions:

- Port 8001: Higress UI console entry
- Port 8080: Gateway HTTP protocol entry
- Port 8443: Gateway HTTPS protocol entry

**All Higress Docker images use their own dedicated repository, unaffected by Docker Hub access restrictions in certain regions**

For other installation methods such as Helm deployment under K8s, please refer to the official [Quick Start documentation](https://higress.io/en-us/docs/user/quickstart).

## Use Cases

- **AI Gateway**:

  Higress can connect to all LLM model providers both domestic and international using a unified protocol, while also providing rich AI observability, multi-model load balancing/fallback, AI token rate limiting, AI caching, and other capabilities:

  ![](https://img.alicdn.com/imgextra/i2/O1CN01izmBNX1jbHT7lP3Yr_!!6000000004566-0-tps-1920-1080.jpg)

- **MCP Server Hosting**:

  Higress, as an Envoy-based API gateway, supports hosting MCP Servers through its plugin mechanism. MCP (Model Context Protocol) is essentially an AI-friendly API that enables AI Agents to more easily call various tools and services. Higress provides unified capabilities for authentication, authorization, rate limiting, and observability for tool calls, simplifying the development and deployment of AI applications.

  ![](https://img.alicdn.com/imgextra/i1/O1CN01wv8H4g1mS4MUzC1QC_!!6000000004952-2-tps-1764-597.png)

  By hosting MCP Servers with Higress, you can achieve:
  - Unified authentication and authorization mechanisms, ensuring the security of AI tool calls
  - Fine-grained rate limiting to prevent abuse and resource exhaustion
  - Comprehensive audit logs recording all tool call behaviors
  - Rich observability for monitoring the performance and health of tool calls
  - Simplified deployment and management through Higress's plugin mechanism for quickly adding new MCP Servers
  - Dynamic updates without disruption: Thanks to Envoy's friendly handling of long connections and Wasm plugin's dynamic update mechanism, MCP Server logic can be updated on-the-fly without any traffic disruption or connection drops

- **Kubernetes ingress controller**:

  Higress can function as a feature-rich ingress controller, which is compatible with many annotations of K8s' nginx ingress controller.
  
  [Gateway API](https://gateway-api.sigs.k8s.io/) support is coming soon and will support smooth migration from Ingress API to Gateway API.
  
- **Microservice gateway**:

  Higress can function as a microservice gateway, which can discovery microservices from various service registries, such as Nacos, ZooKeeper, Consul, Eureka, etc.
  
  It deeply integrates with [Dubbo](https://github.com/apache/dubbo), [Nacos](https://github.com/alibaba/nacos), [Sentinel](https://github.com/alibaba/Sentinel) and other microservice technology stacks.
  
- **Security gateway**:

  Higress can be used as a security gateway, supporting WAF and various authentication strategies, such as key-auth, hmac-auth, jwt-auth, basic-auth, oidc, etc.


## Core Advantages

- **Production Grade**

  Born from Alibaba's internal product with over 2 years of production validation, supporting large-scale scenarios with hundreds of thousands of requests per second.

  Completely eliminates traffic jitter caused by Nginx reload, configuration changes take effect in milliseconds and are transparent to business. Especially friendly to long-connection scenarios such as AI businesses.

- **Streaming Processing**

  Supports true complete streaming processing of request/response bodies, Wasm plugins can easily customize the handling of streaming protocols such as SSE (Server-Sent Events).

  In high-bandwidth scenarios such as AI businesses, it can significantly reduce memory overhead.
    
- **Easy to Extend**
  
  Provides a rich official plugin library covering AI, traffic management, security protection and other common functions, meeting more than 90% of business scenario requirements.

  Focuses on Wasm plugin extensions, ensuring memory safety through sandbox isolation, supporting multiple programming languages, allowing plugin versions to be upgraded independently, and achieving traffic-lossless hot updates of gateway logic.

- **Secure and Easy to Use**
  
  Based on Ingress API and Gateway API standards, provides out-of-the-box UI console, WAF protection plugin, IP/Cookie CC protection plugin ready to use.

  Supports connecting to Let's Encrypt for automatic issuance and renewal of free certificates, and can be deployed outside of K8s, started with a single Docker command, convenient for individual developers to use.

## Community

[Slack](https://w1689142780-euk177225.slack.com/archives/C05GEL4TGTG): to get invited go [here](https://communityinviter.com/apps/w1689142780-euk177225/higress).

### Thanks

Higress would not be possible without the valuable open-source work of projects in the community. We would like to extend a special thank you to Envoy and Istio.

### Related Repositories

- Higress Console: https://github.com/higress-group/higress-console
- Higress Standalone: https://github.com/higress-group/higress-standalone

### Contributors

<a href="https://github.com/alibaba/higress/graphs/contributors">
  <img alt="contributors" src="https://contrib.rocks/image?repo=alibaba/higress"/>
</a>

### Star History

[![Star History Chart](https://api.star-history.com/svg?repos=alibaba/higress&type=Date)](https://star-history.com/#alibaba/higress&Date)

<p align="right" style="font-size: 14px; color: #555; margin-top: 20px;">
    <a href="#readme-top" style="text-decoration: none; color: #007bff; font-weight: bold;">
        ↑ Back to Top ↑
    </a>
</p>
