# Higress Rust Wasm 插件开发框架

## 介绍

此 SDK 用于使用 Rust 语言开发 Higress 的 Wasm 插件。基于 [proxy-wasm-rust-sdk](https://github.com/higress-group/proxy-wasm-rust-sdk) 构建，提供了丰富的开发工具和示例。

## 特性

- 🚀 **高性能**: 基于 Rust 和 WebAssembly，提供接近原生的性能
- 🛠️ **易开发**: 提供完整的开发框架和丰富的示例
- 🔧 **可扩展**: 支持自定义配置、规则匹配、HTTP 调用等功能
- 📦 **容器化**: 支持 Docker 构建和 OCI 镜像发布
- 🧪 **测试友好**: 内置测试框架和 lint 工具

## 快速开始

### 环境要求

- Rust 1.80+
- Docker
- Make
- WASI 目标支持：`rustup target add wasm32-wasip1`

**重要提示**：确保使用 rustup 管理的 Rust 工具链，避免与 Homebrew 安装的 Rust 冲突。如果遇到 WASI 目标问题，请确保：

1. **使用 rustup 管理 Rust**：

   ```bash
   # 安装 rustup（如果还没有）
   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

   # 安装 WASI 目标
   rustup target add wasm32-wasip1
   ```

2. **确保 shell 配置正确**：
   ```bash
   # 检查 ~/.zshrc 或 ~/.bashrc 是否包含
   source "$HOME/.cargo/env"
   ```

### 构建插件

**执行路径**: 在 `plugins/wasm-rust/` 目录下执行

```bash
# 进入项目目录
cd plugins/wasm-rust/

# 构建默认插件 (say-hello)
make build

# 构建指定插件
make build PLUGIN_NAME=say-hello

# 构建并指定版本
make build PLUGIN_NAME=say-hello PLUGIN_VERSION=1.0.0

# 注意：由于 Makefile 中的 .DEFAULT 目标，需要明确指定目标
# 如果遇到 "Nothing to be done" 错误，请确保使用正确的语法
```

**重要提示**：

- 某些插件（如 `ai-data-masking`）依赖 C 库，可能需要额外的配置才能成功构建
- 建议先使用简单的插件（如 `say-hello`）测试构建环境
- 构建成功后会生成 `extensions/<plugin-name>/plugin.wasm` 文件

### 运行测试

**执行路径**: 在 `plugins/wasm-rust/` 目录下执行

```bash
# 进入项目目录
cd plugins/wasm-rust/

# 运行所有测试
make test-base

# 运行指定插件测试
make test PLUGIN_NAME=say-hello
```

### 代码检查

**执行路径**: 在 `plugins/wasm-rust/` 目录下执行

```bash
# 进入项目目录
cd plugins/wasm-rust/

# 运行所有 lint 检查
make lint-base

# 运行指定插件 lint 检查
make lint PLUGIN_NAME=say-hello
```

### Makefile 说明

当前 Makefile 包含以下可用目标：

- `build` - 构建插件（默认插件为 say-hello）
- `lint-base` - 对所有代码进行 lint 检查
- `lint` - 对指定插件进行 lint 检查
- `test-base` - 运行所有测试
- `test` - 运行指定插件测试
- `builder` - 构建构建器镜像

**重要提示**：Makefile 中的 `.DEFAULT:` 目标可能会影响某些命令的执行。如果遇到 "Nothing to be done" 错误，请确保：

1. 正确指定了目标名称（如 `build`、`lint`、`test`）
2. 使用了正确的参数格式
3. 插件目录存在且包含有效的 Cargo.toml 文件

## 插件开发

### 项目结构

```
wasm-rust/
├── src/                    # SDK 核心代码
│   ├── cluster_wrapper.rs  # 集群包装器
│   ├── error.rs           # 错误处理
│   ├── event_stream.rs    # 事件流处理
│   ├── internal.rs        # 内部 API
│   ├── log.rs             # 日志系统
│   ├── plugin_wrapper.rs  # 插件包装器
│   ├── redis_wrapper.rs   # Redis 包装器
│   ├── request_wrapper.rs # 请求包装器
│   └── rule_matcher.rs    # 规则匹配器
├── extensions/            # 插件示例
│   ├── say-hello/        # 基础示例
│   ├── ai-data-masking/  # AI 数据脱敏
│   ├── request-block/    # 请求拦截
│   ├── ai-intent/        # AI 意图识别
│   └── demo-wasm/        # 演示插件
├── example/              # 完整示例
│   ├── wrapper-say-hello/ # 包装器示例
│   └── sse-timing/       # SSE 时序示例
└── Makefile              # 构建脚本
```

### 创建新插件

**执行路径**: 在 `plugins/wasm-rust/` 目录下执行

1. **创建插件目录**

```bash
# 进入项目目录
cd plugins/wasm-rust/

# 创建插件目录
mkdir extensions/my-plugin
cd extensions/my-plugin
```

2. **创建 Cargo.toml**

```toml
[package]
name = "my-plugin"
version = "0.1.0"
edition = "2021"
publish = false

[lib]
crate-type = ["cdylib"]

[dependencies]
higress-wasm-rust = { path = "../../", version = "0.1.0" }
proxy-wasm = { git="https://github.com/higress-group/proxy-wasm-rust-sdk", branch="main", version="0.2.2" }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
```

3. **创建插件代码**

```rust
use higress_wasm_rust::*;
use proxy_wasm::traits::*;
use proxy_wasm::types::*;
use serde::{Deserialize, Serialize};

#[derive(Default, Clone, Serialize, Deserialize)]
struct MyPluginConfig {
    name: String,
}

struct MyPluginRoot {
    log: Log,
    rule_matcher: SharedRuleMatcher<MyPluginConfig>,
}

impl MyPluginRoot {
    fn new() -> Self {
        Self {
            log: Log::new("my-plugin".to_string()),
            rule_matcher: Rc::new(RefCell::new(RuleMatcher::new())),
        }
    }
}

impl Context for MyPluginRoot {}

impl RootContext for MyPluginRoot {
    fn on_configure(&mut self, plugin_configuration_size: usize) -> bool {
        on_configure(self, plugin_configuration_size, &mut self.rule_matcher.borrow_mut(), &self.log)
    }

    fn create_http_context(&self, context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(MyPlugin {
            log: self.log.clone(),
            rule_matcher: self.rule_matcher.clone(),
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpFilter)
    }
}

struct MyPlugin {
    log: Log,
    rule_matcher: SharedRuleMatcher<MyPluginConfig>,
}

impl Context for MyPlugin {}

impl HttpContext for MyPlugin {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> HeaderAction {
        self.log.info("Processing request headers");
        HeaderAction::Continue
    }
}

proxy_wasm::main! {|_| -> Box<dyn RootContext> {
    Box::new(MyPluginRoot::new())
}}
```

### 插件配置

插件支持全局配置和规则配置：

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: my-plugin
  namespace: higress-system
spec:
  selector:
    matchLabels:
      higress: higress-system-higress-gateway
  defaultConfig:
    name: "default"
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/my-plugin:1.0.0
  rules:
    - match:
        - route:
            - "my-route"
      config:
        name: "route-specific"
```

## 内置插件

### 基础插件

- **say-hello**: 基础示例插件，演示插件开发流程 ✅
- **demo-wasm**: 完整演示插件，包含 Redis 集成等功能

### 功能插件

- **ai-data-masking**: AI 数据脱敏插件 ⚠️

  - 支持敏感词拦截和替换
  - 支持 OpenAI 协议和自定义 JSONPath
  - 内置敏感词库和自定义规则
  - **注意**: 依赖 C 库，可能需要额外配置

- **request-block**: 请求拦截插件 ✅

  - 支持 URL、Header、Body 拦截
  - 支持正则表达式匹配
  - 可配置拦截状态码和消息

- **ai-intent**: AI 意图识别插件
  - 支持 LLM 调用和意图分类
  - 可配置代理服务和模型参数

**构建状态说明**：

- ✅ 已验证可成功构建
- ⚠️ 可能需要额外配置
- 未标记的插件需要进一步测试

### 故障排除

**问题**: `error[E0463]: can't find crate for 'core'`

**原因**: 系统中有多个 Rust 安装，Homebrew 的 Rust 优先于 rustup 的 Rust

**解决方案**:

```bash
# 移除 Homebrew 的 Rust
brew uninstall rust

# 确保使用 rustup 的 Rust
rustup default nightly
rustup target add wasm32-wasip1

# 确保 shell 配置正确
echo 'source "$HOME/.cargo/env"' >> ~/.zshrc
source ~/.zshrc
```

## 构建和部署

### 本地构建

**执行路径**: 在 `plugins/wasm-rust/` 目录下执行

```bash
# 进入项目目录
cd plugins/wasm-rust/

# 使用 Makefile 构建插件（推荐）
make build PLUGIN_NAME=my-plugin

# 直接使用 Cargo 构建 WASM 文件
cd extensions/my-plugin
cargo build --target wasm32-wasip1 --release

# 构建 Docker 镜像
cd plugins/wasm-rust/
docker build -t my-plugin:latest --build-arg PLUGIN_NAME=my-plugin .
```

````

### Docker 构建说明

**重要提示**：Dockerfile 需要指定 `PLUGIN_NAME` 参数来构建特定插件。

```bash
# 构建 say-hello 插件
docker build -t say-hello:latest --build-arg PLUGIN_NAME=say-hello .

# 构建 ai-data-masking 插件
docker build -t ai-data-masking:latest --build-arg PLUGIN_NAME=ai-data-masking .

# 构建 request-block 插件
docker build -t request-block:latest --build-arg PLUGIN_NAME=request-block .

# 构建自定义插件
docker build -t my-custom-plugin:latest --build-arg PLUGIN_NAME=my-custom-plugin .
```

**Dockerfile 特性**：
- 基于 `rust:1.80` 构建环境
- 自动安装 WASI 目标和 clang 编译器
- 多阶段构建，最终镜像基于 `scratch`
- 最小化镜像大小（约 300-400KB）
- 只包含编译后的 WASM 文件

**常见问题**：
- **错误**: `failed to read dockerfile: open Dockerfile: no such file or directory`
  - **解决**: 确保在 `plugins/wasm-rust/` 目录下执行命令
- **错误**: `failed to solve: failed to compute cache key`
  - **解决**: 确保指定了正确的 `PLUGIN_NAME` 参数
- **错误**: `can't find crate for 'core'`
  - **解决**: Docker 构建环境会自动安装 WASI 目标，无需手动配置

### 发布到镜像仓库

**执行路径**: 在 `plugins/wasm-rust/` 目录下执行

```bash
# 进入项目目录
cd plugins/wasm-rust/

# 构建插件（会自动推送到官方仓库）
make build PLUGIN_NAME=my-plugin PLUGIN_VERSION=1.0.0

# 构建构建器镜像
make builder
````

### 在 Higress 中使用

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: my-plugin
  namespace: higress-system
spec:
  selector:
    matchLabels:
      higress: higress-system-higress-gateway
  defaultConfig:
    # 插件配置
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/my-plugin:1.0.0
```

## 开发工具

### 路径说明

不同命令需要在不同的目录下执行：

- **Makefile 命令**（如 `make build`、`make test`、`make lint`）：在 `plugins/wasm-rust/` 目录下执行
- **Cargo 命令**（如 `cargo build`、`cargo test`）：在具体的插件目录下执行（如 `plugins/wasm-rust/extensions/my-plugin/`）
- **Docker 命令**：在 `plugins/wasm-rust/` 目录下执行，需要指定 `PLUGIN_NAME` 参数

### 调试

插件支持详细的日志输出：

```rust
self.log.info("Processing request");
self.log.debugf(format_args!("Request headers: {:?}", headers));
self.log.error("Error occurred");
```

### 测试

**执行路径**: 在插件目录下执行（如 `plugins/wasm-rust/extensions/my-plugin/`）

```bash
# 进入插件目录
cd plugins/wasm-rust/extensions/my-plugin/

# 运行单元测试
cargo test

# 运行集成测试
cargo test --test integration
```

### 性能优化

- 使用 `--release` 模式构建
- 避免不必要的内存分配
- 合理使用缓存机制

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交代码变更
4. 运行测试和 lint 检查
5. 提交 Pull Request

## 相关链接

- [Higress 官方文档](https://higress.io/)
- [proxy-wasm-rust-sdk](https://github.com/higress-group/proxy-wasm-rust-sdk)
- [WebAssembly 规范](https://webassembly.org/)
- [Rust 官方文档](https://doc.rust-lang.org/)

## 许可证

本项目采用 Apache 2.0 许可证。
