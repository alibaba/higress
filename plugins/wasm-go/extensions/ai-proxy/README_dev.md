## 构建方法

确认本机已安装 Docker，然后根据操作系统选择对应的构建命令，并在 `ai-proxy` 目录下执行。构建产物将输出至 `out` 目录。

***Linux/macOS:***

```shell
DOCKER_BUILDKIT=1; docker build --build-arg PLUGIN_NAME=ai-proxy --build-arg EXTRA_TAGS=proxy_wasm_version_0_2_100 --build-arg BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.19-tinygo0.28.1-oras1.0.0 -t ai-proxy:0.0.1 --output ./out ../..
```

***Windows:***

```powershell
$env:DOCKER_BUILDKIT=1; docker build --build-arg PLUGIN_NAME=ai-proxy --build-arg EXTRA_TAGS=proxy_wasm_version_0_2_100 --build-arg BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.19-tinygo0.28.1-oras1.0.0 -t ai-proxy:0.0.1 --output .\out ..\..
```

## 本地运行
参考：https://higress.io/zh-cn/docs/user/wasm-go
需要注意的是，higress/plugins/wasm-go/extensions/ai-proxy/envoy.yaml中的clusters字段，记得改成你需要地址，比如混元的话：就会有如下的一个cluster的配置：
```yaml
<省略>
static_resources:
<省略>
  clusters:
      load_assignment:
        cluster_name: moonshot
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: hunyuan.tencentcloudapi.com
                      port_value: 443
      transport_socket:
        name: envoy.transport_sockets.tls
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
          "sni": "hunyuan.tencentcloudapi.com"
```

而后你就可以在本地的pod中查看相应的输出，请求样例如下：
```sh
curl --location 'http://127.0.0.1:10000/v1/chat/completions' \
--header 'Content-Type:  application/json' \
--data '{
  "model": "gpt-3",
  "messages": [
    {
      "role": "system",
      "content": "你是一个名专业的开发人员！"
    },
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ],
  "temperature": 0.3,
  "stream": false
}'
```

## 测试须知

由于 `ai-proxy` 插件使用了 Higress 对数据面定制的特殊功能，因此在测试时需要使用版本不低于 1.4.0-rc.1 的 Higress Gateway 镜像。