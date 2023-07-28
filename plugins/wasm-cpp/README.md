[English](./README_EN.md)

## 介绍

此 SDK 用于使用 CPP 语言开发 Higress 的 Wasm 插件。

## 使用 Higress wasm-cpp builder 快速构建

使用以下命令可以快速构建 wasm-cpp 插件:

```bash
$ PLUGIN_NAME=request_block make build
```

<details>
<summary>输出结果</summary>
<pre><code>
DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=request_block \
                                    -t request_block:20230721-141120-aa17e95 \
                                    --output extensions/request_block \
                                    .
[+] Building 2.3s (10/10) FINISHED 

output wasm file: extensions/request_block/plugin.wasm
</code></pre>
</details>

该命令最终构建出一个 wasm 文件和一个 Docker image。
这个本地的 wasm 文件被输出到了指定的插件的目录下，可以直接用于调试。

### 参数说明

| 参数名称          | 可选/必须 | 默认值                                       | 含义                                                          |
|---------------|-------|-------------------------------------------|----------------------------------------------------------------------|
| `PLUGIN_NAME` | 可选的   | hello-world                               | 要构建的插件名称。                                                    |
| `IMG`         | 可选的   | 如不设置则根据仓库地址、插件名称、构建时间以及 git commit id 生成。 | 生成的镜像名称。如非空，则会覆盖`REGISTRY` 参           |

## 创建 WasmPlugin 资源使插件生效

编写 WasmPlugin 资源如下：

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
  url: oci://<your_registry_hub>/request_block:1.0.0  # 之前构建和推送的 image 地址
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
  url: oci://<your_registry_hub>/request_block:1.0.0
```

所有规则会按上面配置的顺序一次执行匹配，当有一个规则匹配时，就停止匹配，并选择匹配的配置执行插件逻辑。

## E2E测试

当你完成一个GO语言的插件功能时, 可以同时创建关联的e2e test cases, 并在本地对插件功能完成测试验证。

### step1. 编写 test cases
在目录./test/e2e/conformance下面, 分别添加xxx.yaml文件和xxx.go文件, 比如测试插件request-block

./test/e2e/conformance/tests/cpp-request_block.yaml
```
apiVersion: networking.k8s.io/v1
kind: Ingress
...
...
spec:
  defaultConfig:
    block_urls:
    - "swagger.html"
  url: file:///opt/plugins/wasm-cpp/extensions/request_block/plugin.wasm
```
`其中url中extensions后面的'request-block'为插件所在文件夹名称`

./test/e2e/conformance/tests/cpp-request_block.go

### step2. 添加 test cases
将上述所写test cases添加到e2e测试列表中,

./test/e2e/e2e_test.go

```
...
cSuite.Setup(t)
	var higressTests []suite.ConformanceTest

	if *isWasmPluginTest {
		if strings.Compare(*wasmPluginType, "CPP") == 0 {
			m := make(map[string]suite.ConformanceTest)
			m["request_block"] = tests.CPPWasmPluginsRequestBlock
			m["key_auth"] = tests.CPPWasmPluginsKeyAuth
        //这里新增你新写的case方法名称

			higressTests = []suite.ConformanceTest{
				m[*wasmPluginName],
			}
		} else {
			higressTests = []suite.ConformanceTest{
				tests.WasmPluginsRequestBlock,
			}
		}
	} else {
...
```

### step3. 编译插件并执行 test cases
考虑到本地构建wasm比较耗时, 我们支持只构建需要测试的插件(同时你也可以临时修改上面第二小步的测试cases列表, 只执行你新写的case)。

```bash
PLUGIN_TYPE=CPP PLUGIN_NAME=request_block make higress-wasmplugin-test
```