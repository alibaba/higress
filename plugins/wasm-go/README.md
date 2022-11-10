[English](./README_EN.md)

## 介绍

此 SDK 用于开发 Higress 的 Wasm 插件

## 编译环境要求

(需要支持 go 范型特性)

Go 版本: >= 1.18

TinyGo 版本: >= 0.25.0

## Quick Examples

使用 [request-block](extensions/request-block) 作为例子

### step1. 编译 wasm

```bash
tinygo build -o main.wasm -scheduler=none -target=wasi ./extensions/request-block/main.go
```

### step2. 构建并推送插件的 docker 镜像

使用这份简单的 [dockerfile](./Dockerfile).

```bash
docker build -t <your_registry_hub>/request-block:1.0.0 .
docker push <your_registry_hub>/request-block:1.0.0
```

### step3. 创建 WasmPlugin 资源

```yaml
apiVersion: extensions.istio.io/v1alpha1
kind: WasmPlugin
metadata:
  name: request-block
  namespace: higress-system
spec:
  selector:
    matchLabels:
      higress: higress-system-higress-gateway
  pluginConfig:
    block_urls:
    - "swagger.html"
  url: oci://<your_registry_hub>/request-block:1.0.0
```

创建上述资源后，如果请求url携带 `swagger.html`, 则这个请求就会被拒绝，例如：

```bash
curl <your_gateway_address>/api/user/swagger.html
```

```text
HTTP/1.1 403 Forbidden
date: Wed, 09 Nov 2022 12:12:32 GMT
server: istio-envoy
content-length: 0
```

如果需要进一步控制插件的执行阶段和顺序

可以阅读此 [文档](https://istio.io/latest/docs/reference/config/proxy_extensions/wasm-plugin/) 了解更多关于 wasmplugin 的配置


## 路由级或域名级生效

```yaml
apiVersion: extensions.istio.io/v1alpha1
kind: WasmPlugin
metadata:
  name: request-block
  namespace: higress-system
spec:
  selector:
    matchLabels:
      higress: higress-system-higress-gateway 
  pluginConfig:
   # 跟上面例子一样，这个配置会全局生效，但如果被下面规则匹配到，则会改为执行命中规则的配置
   block_urls:
   - "swagger.html"
   _rules_:
   # 路由级生效配置
   - _match_route_:
     - default/foo
     # default 命名空间下名为 foo 的 ingress 会执行下面这个配置
     block_bodys:
     - "foo"
   - _match_route_:
     - default/bar
     # default 命名空间下名为 bar 的 ingress 会执行下面这个配置
     block_bodys:
     - "bar"
   # 域名级生效配置
   - _match_domain_:
     - "*.example.com"
     # 若请求匹配了上面的域名, 会执行下面这个配置
     block_bodys:
     - "foo"
     - "bar"
  url: oci://<your_registry_hub>/request-block:1.0.0
```

所有规则会按上面配置的顺序一次执行匹配，当有一个规则匹配时，就停止匹配，并选择匹配的配置执行插件逻辑

