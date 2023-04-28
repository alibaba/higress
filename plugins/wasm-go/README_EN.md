## Intro

This SDK is used to develop the WASM Plugins for Higress in Go.

## Quick build with Higress wasm-go builder

The wasm-go plugin can be built quickly with the following command:

```bash
$ PLUGIN_NAME=request-block make build
```

<details>
<summary>Output</summary>
<pre><code>
DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=request-block \
                               -t request-block:20230223-173305-3b1a471 \
                               --output extensions/request-block .
[+] Building 67.7s (12/12) FINISHED

image:            request-block:20230223-173305-3b1a471
output wasm file: extensions/request-block/plugin.wasm
</code></pre>
</details>

This command eventually builds a wasm file and a Docker image.
This local wasm file is exported to the specified plugin's directory and can be used directly for debugging.
You can also use `make build-push` to build and push the image at the same time.

### Environmental parameters

| Name          | Optional/Required | Default                                                                                                      | 含义                                                                                                                                  |
|---------------|-------------------|--------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------|
| `PLUGIN_NAME` | Optional          | hello-world                                                                                                  | The name of the plugin to build.                                                                                                    |
| `REGISTRY`    | Optional          | empty                                                                                                        | The regitstry address of the generated image, e.g. `example.registry.io/my-name/`.  Note that the REGISTRY value should end with /. |
| `IMG`         | Optional          | If it is empty, it is generated based on the repository address, plugin name, build time, and git commit id. | The generated image tag will override the `REGISTRY` parameter if it is not empty.                                                  |

## Build on local yourself

You can also build wasm locally and copy it to a Docker image. This requires a local build environment:

Go version: >= 1.18

TinyGo version: >= 0.25.0

The following is an example of building the plugin [request-block](extensions/request-block).

### step1. build wasm

```bash
tinygo build -o main.wasm -scheduler=none -target=wasi ./extensions/request-block/main.go
```

### step2. build and push docker image

A simple Dockerfile:

```Dockerfile
FROM scratch
COPY main.wasm plugin.wasm
```

```bash
docker build -t <your_registry_hub>/request-block:1.0.0 -f <your_dockerfile> .
docker push <your_registry_hub>/request-block:1.0.0
```

## Apply WasmPlugin API

Read this [document](https://istio.io/latest/docs/reference/config/proxy_extensions/wasm-plugin/) to learn more about wasmplugin.

Create a WasmPlugin API resource:

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: request-block
  namespace: higress-system
spec:
  defaultConfig:
    block_urls:
    - "swagger.html"
  url: oci://<your_registry_hub>/request-block:1.0.0
```

When the resource is applied on the Kubernetes cluster with `kubectl apply -f <your-wasm-plugin-yaml>`,
the request will be blocked if the string `swagger.html` in the url. 

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
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: request-block
  namespace: higress-system
spec:
  defaultConfig:
   # this config will take effect globally (all incoming requests not matched by rules below)
   block_urls:
   - "swagger.html"
  matchRules:
  # ingress-level takes effect
  - ingress:
    - default/foo
    # the ingress foo in namespace default will use this config
    config:
      block_bodies:
      - "foo"
  - ingress:
    - default/bar
    # the ingress bar in namespace default will use this config
    config:
      block_bodies:
      - "bar"
  # domain-level takes effect
  - domain:
    - "*.example.com"
    # if the request's domain matched, this config will be used
    config:
      block_bodies:
       - "foo"
       - "bar"
  url: oci://<your_registry_hub>/request-block:1.0.0
```

The rules will be matched in the order of configuration. If one match is found, it will stop, and the matching configuration will take effect.

