## Intro

This SDK is used to develop the WASM Plugins of Higress.
## Requirements

(need support Go's type parameters)

Go version: >= 1.18

TinyGo version: >= 0.25.0

## Quick Examples

Use the [request-block](example/request-block) as an example

### step1. compile to wasm

```bash
tinygo build -o main.wasm -scheduler=none -target=wasi ./example/request-block/main.go
```

### step2. build&push docker image

Use this [dockerfile](./Dockerfile).

```bash
docker build -t <your_registry_hub>/request-block:1.0.0 .
docker push <your_registry_hub>/request-block:1.0.0
```

### step3. create WasmPlugin resource

Read this [document](https://istio.io/latest/docs/reference/config/proxy_extensions/wasm-plugin/) to learn more about wasmplugin.

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

If the url in request contains the `swagger.html`, the request will be blocked.

```bash
curl <your_gateway_address>/api/user/swagger.html
```

```text
HTTP/1.1 403 Forbidden
date: Wed, 09 Nov 2022 12:12:32 GMT
server: istio-envoy
content-length: 0
```

## route-level & domain-level takes effect

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
   # this config will take effect globally (all incoming requests not matched by rules below)
   block_urls:
   - "swagger.html"
   _rules_:
   # route-level takes effect
   - _match_route_:
     - default/foo
     # the ingress foo in namespace default will use this config
     block_bodys:
     - "foo"
   - _match_route_:
     - default/bar
     # the ingress bar in namespace default will use this config
     block_bodys:
     - "bar"
   # domain-level takes effect
   - _match_domain_:
     - "*.example.com"
     # if the request's domain matched, this config will be used
     block_bodys:
     - "foo"
     - "bar"
  url: oci://<your_registry_hub>/request-block:1.0.0
```

The rules will be matched in the order of configuration. If one match is found, it will stop, and the matching configuration will take effect.

