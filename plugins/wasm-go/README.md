[English](./README_EN.md)

## 介绍

此 SDK 用于开发 Higress 的 Wasm 插件

## 使用 Docker 快速构建

使用以下命令可以快速构建 wasm-go 插件:

```bash
$ PLUGIN_NAME=request-block make build
```

<details>
<summary>输出结果</summary>
<pre><code>
DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=request-block \
                               --build-arg GO_VERSION= \
                               --build-arg TINYGO_VERSION= \
                               -t request-block:20230213-170844-ca49714 \
                               -f DockerfileBuilder \
                               --output extensions/request-block .
[+] Building 84.6s (15/15)                                                                                                                                                                                                                                                0.0s 

image:            request-block:20230211-184334-f402f86
output wasm file: extensions/request-block/plugin.wasm
</code></pre>
</details>

该命令最终构建出一个 wasm 文件和一个 Docker image。
这个本地的 wasm 文件被输出到了指定的插件的目录下，可以直接用于调试。
你也可以直接使用 `make build-push` 一并构建和推送 image.

### 参数说明

| 参数名称             | 可选/必须 | 默认值                                      | 含义                                                                   |
|------------------|-------|------------------------------------------|----------------------------------------------------------------------|
| `PLUGIN_NAME`    | 可选的   | hello-world                              | 要构建的插件名称。                                                            |
| `REGISTRY`       | 可选的   | 空                                        | 生成的镜像的仓库地址，如 `example.registry.io/my-name/`.  注意 REGISTRY 值应当以 / 结尾。 |
| `IMG`            | 可选的   | 如不设置则根据仓库地址、插件名称、构建时间以及 git commit id 生成 | 生成的镜像名称。                                                             |
| `GO_VERSION`     | 可选的   | 1.19                                     | Go 版本号。                                                              |
| `TINYGO_VERSION` | 可选的   | 0.25.0                                   | TinyGo 版本号。                                                          |

## 本地构建

你也可以选择先在本地将 wasm 构建出来，再拷贝到 Docker 镜像中。这要求你要先在本地搭建构建环境。

编译环境要求如下：

- Go 版本: >= 1.18 (需要支持范型特性)

- TinyGo 版本: >= 0.25.0

下面是本地多步骤构建 [request-block](extensions/request-block) 的例子。

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

## 创建 WasmPlugin 资源使插件生效

编写 WasmPlugin 资源如下：

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: request-block
  namespace: higress-system
spec:
  selector:
    matchLabels:
      higress: higress-system-higress-gateway
  defaultConfig:
    block_urls:
    - "swagger.html"
  url: oci://<your_registry_hub>/request-block:1.0.0  # 之前构建和推送的 image 地址
```

使用 `kubectl apply -f <your-wasm-plugin-yaml>` 使资源生效。

资源生效后，如果请求url携带 `swagger.html`, 则这个请求就会被拒绝，例如：

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
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: request-block
  namespace: higress-system
spec:
  selector:
    matchLabels:
      higress: higress-system-higress-gateway
  defaultConfig:
   # 跟上面例子一样，这个配置会全局生效，但如果被下面规则匹配到，则会改为执行命中规则的配置
   block_urls:
   - "swagger.html"
   matchRules:
   # 路由级生效配置
  - ingress:
    - default/foo
     # default 命名空间下名为 foo 的 ingress 会执行下面这个配置
    config:
      block_bodies:
      - "foo"
  - ingress:
    - default/bar
    # default 命名空间下名为 bar 的 ingress 会执行下面这个配置
    config:
      block_bodies:
      - "bar"
   # 域名级生效配置
  - domain:
    - "*.example.com"
    # 若请求匹配了上面的域名, 会执行下面这个配置
    config:
      block_bodies:
      - "foo"
      - "bar"
  url: oci://<your_registry_hub>/request-block:1.0.0
```

所有规则会按上面配置的顺序一次执行匹配，当有一个规则匹配时，就停止匹配，并选择匹配的配置执行插件逻辑。

## 更多构建细节

使用 `make build` 会从头构建 wasm-go 的 builder 镜像，然后使用 tinygo 构建 wasm。
构建过程中会下载 Go 和 Tinygo 安装包，以及执行 apt-get update，会花费大量时间在网络 IO 上。
如果不想每次构建都从头开始，可以使用分阶段构建。下面介绍分阶段构建的 make 规则。

### `make builder`

`make builder` 用来构建 wasm-go 编译环境镜像。

```bash
make builder
```
<details>
<summary>输出结果</summary>
<pre><code>You can use the following command to build the final image in the next stage
WASM_BUILDER=wasm-go-builder:go1.19-tinygo0.26.0 PLUGIN_NAME= make bb
</code></pre>
</details>

可以指定 `GO_VERSION`, `TINYGO_VERSION` 参数决定 Go 和 Tinygo 版本号;
指定 `REGISTRY` 设置要推送的镜像仓库。

一旦构建出 builder 镜像，以后就可以复用这个镜像来构建 wasm-go, 而不必每次都从头开始构建。

### `make build-on-builder` 或 `make bb`

`make build-on-builder` 是利用上一阶段输出的 builder 镜像来构建 wasm-go.
`make bb` 是 `make build-on-builder` 的别名。

```bash
WASM_BUILDER=wasm-go-builder:go1.20-tinygo0.27.0 PLUGIN_NAME=request-block make bb
```
<details>
<summary>输出结果</summary>
<pre><code>image:            request-block:20230216-143723-c5845c1
output wasm file: extensions/request-block/plugin.wasm
</code></pre>
</details>

其中 `WASM_BUILDER=wasm-go-builder:go1.20-tinygo0.27.0` 是上一阶段输出的 builder 镜像。
