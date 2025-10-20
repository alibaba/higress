# Nginx Migration MCP Server

一个用于将 Nginx 配置和 Lua 插件迁移到 Higress 的 MCP 服务器。

## 功能特性

### 配置转换工具
- **parse_nginx_config** - 解析和分析 Nginx 配置文件
- **convert_to_higress** - 将 Nginx 配置转换为 Higress HTTPRoute

### Lua 插件迁移工具链

#### 快速转换模式
- **convert_lua_to_wasm** - 一键将 Lua 脚本转换为 WASM 插件

#### LLM 辅助工具链（精细化控制）
1. **analyze_lua_plugin** - 分析 Lua 插件兼容性
2. **generate_conversion_hints** - 生成详细的代码转换提示和 API 映射
3. **validate_wasm_code** - 验证生成的 Go WASM 代码
4. **generate_deployment_config** - 生成完整的部署配置包

## 快速开始

### 构建

```bash
make build
```

构建后会生成 `nginx-migration-mcp` 可执行文件。

### 配置 MCP 客户端

在 MCP 客户端（如 Cursor）的配置文件中添加：

```json
{
  "mcpServers": {
    "nginx-migration": {
      "command": "/path/to/nginx-migration-mcp",
      "args": []
    }
  }
}
```

重启 MCP 客户端后即可使用。详细步骤请参考 [QUICKSTART.md](QUICKSTART.md)。

## 使用示例

### 转换 Nginx 配置

使用 `convert_to_higress` 工具，传入 Nginx 配置内容，自动生成 Higress HTTPRoute 和 Service 资源。

### 迁移 Lua 插件

**方式一：快速转换**

使用 `convert_lua_to_wasm` 工具一键转换 Lua 脚本为 WASM 插件。

**方式二：AI 辅助工具链**

1. 使用 `analyze_lua_plugin` 分析 Lua 代码
2. 使用 `generate_conversion_hints` 获取转换提示和 API 映射
3. AI 根据提示生成 Go WASM 代码
4. 使用 `validate_wasm_code` 验证生成的代码
5. 使用 `generate_deployment_config` 生成部署配置

推荐使用工具链方式处理复杂插件，可获得更好的转换质量和 AI 辅助。

## 项目结构

```
nginx-migration/
├── cmd/
│   └── standalone/          # 独立模式入口
├── integration/             # Higress 集成模式
│   ├── server.go           # MCP 服务器注册
│   └── mcptools/           # 工具实现
│       ├── context.go      # 迁移上下文
│       ├── nginx_tools.go  # Nginx 配置工具
│       ├── lua_tools.go    # Lua 插件工具
│       └── tool_chain.go   # 工具链实现
├── internal/
│   └── standalone/         # 独立模式实现
├── tools/                  # 核心转换逻辑
│   ├── mcp_tools.go        # 工具定义
│   ├── tool_chain.go       # 工具链实现
│   └── lua_converter.go    # Lua 转换器
├── examples/               # 示例代码
├── go.mod                  # Go 模块定义
├── Makefile                # 构建脚本
└── mcp-tools.json          # 工具配置
```

## 开发

### 构建命令

```bash
# 编译
make build

# 清理
make clean

# 格式化代码
make fmt

# 运行测试
make test

# 查看帮助
make help
```


