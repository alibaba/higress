[English](./README_EN.md)

## 介绍

此 SDK 用于开发 Higress 的 Wasm 插件

## 使用 Docker 快速构建

使用以下命令可以快速构建 wasm-go 插件:

```bash
$ THIS_ARCH=arm64 PLUGIN_NAME=request-block make build
```
<details>
<summary>输出结果</summary>
<pre><code>
DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=request-block \
                               --build-arg THIS_ARCH=arm64 \
                               -t request-block:20230211-184334-f402f86 \
                               -f DockerfileBuilder \
                               --output extensions/request-block .
[+] Building 11.8s (17/17) FINISHED                                                                                                                                                                                                                                                  
 => [internal] load .dockerignore                                                                                                                                                                                                                                               0.0s
 => => transferring context: 2B                                                                                                                                                                                                                                                 0.0s
 => [internal] load build definition from DockerfileBuilder                                                                                                                                                                                                                     0.0s
 => => transferring dockerfile: 843B                                                                                                                                                                                                                                            0.0s
 => [internal] load metadata for docker.io/library/ubuntu:latest                                                                                                                                                                                                                0.9s
 => [builder  1/11] FROM docker.io/library/ubuntu@sha256:9a0bdde4188b896a372804be2384015e90e3f84906b750c1a53539b585fbbe7f                                                                                                                                                       0.0s
 => [internal] load build context                                                                                                                                                                                                                                               0.0s
 => => transferring context: 6.65kB                                                                                                                                                                                                                                             0.0s
 => CACHED [builder  2/11] RUN apt-get update   && apt-get install -y wget build-essential  && rm -rf /var/lib/apt/lists/*                                                                                                                                                      0.0s
 => CACHED [builder  3/11] RUN wget https://golang.google.cn/dl/go1.19.3.linux-arm64.tar.gz                                                                                                                                                                                     0.0s
 => CACHED [builder  4/11] RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.3.linux-arm64.tar.gz                                                                                                                                                                       0.0s
 => CACHED [builder  5/11] RUN wget https://github.com/tinygo-org/tinygo/releases/download/v0.25.0/tinygo_0.25.0_arm64.deb                                                                                                                                                      0.0s
 => CACHED [builder  6/11] RUN dpkg -i tinygo_0.25.0_arm64.deb                                                                                                                                                                                                                  0.0s
 => CACHED [builder  7/11] WORKDIR /workspace                                                                                                                                                                                                                                   0.0s
 => [builder  8/11] COPY . .                                                                                                                                                                                                                                                    0.0s
 => [builder  9/11] WORKDIR /workspace/extensions/request-block                                                                                                                                                                                                                 0.0s
 => [builder 10/11] RUN go mod tidy                                                                                                                                                                                                                                             1.1s
 => [builder 11/11] RUN tinygo build -o /main.wasm -scheduler=none -target=wasi ./main.go                                                                                                                                                                                       9.5s
 => CACHED [stage-1 1/1] COPY --from=builder /main.wasm plugin.wasm                                                                                                                                                                                                             0.0s 
 => exporting to client                                                                                                                                                                                                                                                         0.0s 
 => => copying files 998.36kB                                                                                                                                                                                                                                                   0.0s 

image:            request-block:20230211-184334-f402f86
output wasm file: extensions/request-block/plugin.wasm
</code></pre>
</details>

该命令最终构建出一个 wasm 文件和一个 Docker image。
这个本地的 wasm 文件被输出到了指定的插件的目录下，可以直接用于调试。
你也可以直接使用 `make build-push` 一并构建和推送 image.

### 参数说明

| 参数名称          | 可选/必须 | 默认值                                      | 含义                                           |
|---------------|-------|------------------------------------------|----------------------------------------------|
| `THIS_ARCH`   | 可选的   | amd64                                    | 构建插件的机器的指令集架构，在非 amd64 架构的机器上构建时要手动指定。       |
| `PLUGIN_NAME` | 可选的   | hello-world                              | 要构建的插件名称。                                    |
| `REGISTRY`    | 可选的   | 空                                        | 生成的镜像的仓库地址，如 `example.registry.io/my-name/`. |
| `IMG`         | 可选的   | 如不设置则根据仓库地址、插件名称、构建时间以及 git commit id 生成 | 生成的镜像名称。                                     |

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

