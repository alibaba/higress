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
## 测试须知

由于 `ai-proxy` 插件使用了 Higress 对数据面定制的特殊功能，因此在测试时需要使用版本不低于 1.4.0-rc.1 的 Higress Gateway 镜像。