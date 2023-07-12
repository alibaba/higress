<h1 align="center">
    <img src="https://img.alicdn.com/imgextra/i2/O1CN01NwxLDd20nxfGBjxmZ_!!6000000006895-2-tps-960-290.png" alt="Higress" width="240" height="72.5">
  <br>
  Next-generation Cloud Native Gateway
</h1>

[![Build Status](https://github.com/alibaba/higress/workflows/build%20and%20codecov/badge.svg?branch=main)](https://github.com/alibaba/higress/actions)
[![license](https://img.shields.io/github/license/alibaba/higress.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

[**Official Site**](https://higress.io/en-us/) &nbsp; |
&nbsp; [**Docs**](https://higress.io/en-us/docs/overview/what-is-higress) &nbsp; |
&nbsp; [**Blog**](https://higress.io/en-us/blog) &nbsp; |
&nbsp; [**Developer**](https://higress.io/en-us/docs/developers/developers_dev) &nbsp; |
&nbsp; [**Higress in Cloud**](https://www.alibabacloud.com/product/microservices-engine?spm=higress-website.topbar.0.0.0) &nbsp;


<p>
   English | <a href="README.md">中文<a/>
</p>

Higress is a next-generation cloud-native gateway based on Alibaba's internal gateway practices. 

Powered by [Istio](https://github.com/istio/istio) and [Envoy](https://github.com/envoyproxy/envoy), Higress realizes the integration of the triple gateway architecture of traffic gateway, microservice gateway and security gateway, thereby greatly reducing the costs of deployment, operation and maintenance.

<h1 align="center">
  <img src="https://img.alicdn.com/imgextra/i1/O1CN01iO9ph825juHbOIg75_!!6000000007563-2-tps-2483-2024.png" alt="Higress Architecture">
</h1>


## Summary

- [**Use Cases**](#use-cases)
- [**Higress Features**](#higress-features)
- [**Quick Start**](https://higress.io/en-us/docs/user/quickstart)
- [**Community**](#community)
- [**Thanks**](#thanks)

## Use Cases

- **Kubernetes ingress controller**:

  Higress can function as a feature-rich ingress controller, which is compatible with many annotations of K8s' nginx ingress controller.
  
  [Gateway API](https://gateway-api.sigs.k8s.io/) support is coming soon and will support smooth migration from Ingress API to Gateway API.
  
- **Microservice gateway**:

  Higress can function as a microservice gateway, which can discovery microservices from various service registries, such as Nacos, ZooKeeper, Consul, Eureka, etc.
  
  It deeply integrates of [Dubbo](https://github.com/apache/dubbo), [Nacos](https://github.com/alibaba/nacos), [Sentinel](https://github.com/alibaba/Sentinel) and other microservice technology stacks.
  
- **Security gateway**:

  Higress can be used as a security gateway, supporting WAF and various authentication strategies, such as key-auth, hmac-auth, jwt-auth, basic-auth, oidc, etc.  

## Higress Features

- **Easy to use**

  Provide one-stop gateway solutions for traffic scheduling, service management, and security protection, support Console, K8s Ingress, and Gateway API configuration methods, and also support HTTP to Dubbo protocol conversion, and easily complete protocol mapping configuration.  
  
- **Easy to expand**

  Provides Wasm, Lua, and out-of-process plug-in extension mechanisms, so that multi-language plug-in writing is no longer an obstacle. The granularity of plug-in effectiveness supports not only the global level, domain name level, but also fine-grained routing level
  
- **Dynamic hot update**
  
  Get rid of the traffic jitter caused by reload at the bottom, the configuration change takes effect in milliseconds and the business is not affected, the Wasm plug-in is hot updated and the traffic is not damaged
  
- **Smooth upgrade**

  Compatible with 80%+ usage scenarios of Nginx Ingress Annotation, and provides more feature-rich annotations, easy to handle Nginx Ingress migration in one step
  
- **Security**

  Provides JWT, OIDC, custom authentication and authentication, deeply integrates open source web application firewall.

## Community

[Slack](https://w1689142780-euk177225.slack.com/archives/C05GEL4TGTG): to get invited go [here](https://communityinviter.com/apps/w1689142780-euk177225/higress).

### Thanks

Higress would not be possible without the valuable open-source work of projects in the community. We would like to extend a special thank-you to Envoy and Istio.
