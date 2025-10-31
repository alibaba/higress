## 构建方法

确认本机已安装 Docker，然后根据操作系统选择对应的构建命令，并在 `ai-a2as` 目录下执行。构建产物将输出至 `out` 目录。

***Linux/macOS:***

```shell
DOCKER_BUILDKIT=1; docker build --build-arg PLUGIN_NAME=ai-a2as --build-arg EXTRA_TAGS=proxy_wasm_version_0_2_100 --build-arg BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.0-oras1.0.0 -t ai-a2as:1.1.0 --output ./out ../..
```

***Windows:***

```powershell
$env:DOCKER_BUILDKIT=1; docker build --build-arg PLUGIN_NAME=ai-a2as --build-arg EXTRA_TAGS=proxy_wasm_version_0_2_100 --build-arg BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.0-oras1.0.0 -t ai-a2as:1.1.0 --output .\out ..\..
```

## 本地构建（使用 TinyGo）

如果已安装 TinyGo 和 wasm-opt，可以直接使用 Makefile 构建：

```bash
make build
```

或手动构建：

```bash
tinygo build -o ai-a2as.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100' ./main.go
```

## 测试

运行所有测试：

```bash
go test -v .
```

运行特定测试套件：

```bash
# 认证提示测试
go test -v . -run TestAuthenticatedPrompts

# 安全边界测试
go test -v . -run TestSecurityBoundaries

# 行为证书测试
go test -v . -run TestBehaviorCertificates

# Nonce 验证集成测试（v1.2.0+）
go test -v . -run TestNonceVerification

# Nonce 单元测试（v1.2.0+）
go test -v . -run TestNonceStore
go test -v . -run TestNonceLength
```

生成测试覆盖率报告：

```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## 测试覆盖率统计 (v1.2.0)

- **单元测试**: 7 个（Nonce 存储逻辑）
- **集成测试**: 14 个（完整的 Nonce 验证流程）
- **总测试用例**: 21 个
- **测试模式**: Go 模式 + WASM 模式（双模式验证）

## 测试须知

由于 `ai-a2as` 插件使用了 Higress 对数据面定制的特殊功能，因此在测试时需要使用版本不低于 1.4.0-rc.1 的 Higress Gateway 镜像。

## 相关文档

- [OWASP A2AS Framework](https://genai.owasp.org/llm-top-10-governance-doc/A2AS-Framework/)
- [RFC 9421 HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [Higress WASM 插件开发指南](https://higress.io/zh-cn/docs/user/wasm-go)

