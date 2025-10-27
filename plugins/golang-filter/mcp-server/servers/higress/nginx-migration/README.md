# Nginx Migration MCP Server

一个用于将 Nginx 配置和 Lua 插件迁移到 Higress 的 MCP 服务器。

## 功能特性

### 配置转换工具
- **parse_nginx_config** - 解析和分析 Nginx 配置文件
- **convert_to_higress** - 将 Nginx 配置转换为 Higress Ingress（主要方式）或 HTTPRoute（可选）

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

### 基础配置（无需知识库）

**默认模式**：工具可以直接使用，基于内置规则生成转换建议。

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

### 进阶配置（启用 RAG 知识库）

RAG（检索增强生成）功能通过阿里云百炼集成 Higress 官方文档知识库，提供更准确的转换建议和 API 映射。

#### 适用场景

启用 RAG 后，以下工具将获得增强：
- **generate_conversion_hints** - 提供基于官方文档的 API 映射和代码示例
- **validate_wasm_code** - 基于最佳实践验证代码质量
- **query_knowledge_base** - 直接查询 Higress 官方文档

#### 配置步骤

**步骤 1：获取阿里云百炼凭证**

1. 访问 [阿里云百炼控制台](https://bailian.console.aliyun.com/)
2. 创建或选择一个应用空间，获取 **业务空间 ID** (`workspace_id`)
3. 创建知识库并导入 Higress 文档，获取 **知识库 ID** (`knowledge_base_id`)
4. 在 [阿里云 RAM 控制台](https://ram.console.aliyun.com/manage/ak) 创建 AccessKey
   - 获取 **AccessKey ID** (`access_key_id`)
   - 获取 **AccessKey Secret** (`access_key_secret`)

> **安全提示**：请妥善保管 AccessKey，避免泄露。建议使用 RAM 子账号并授予最小权限。

**步骤 2：复制配置文件**
```bash
cp config/rag.json.example config/rag.json
```

**步骤 3：编辑配置文件**

有两种配置方式：

**方式 1：配置 rag.config**
```json


```json
{
  "rag": {
    "enabled": true,
    "provider": "bailian",
    "endpoint": "bailian.cn-beijing.aliyuncs.com",
    "workspace_id": "${WORKSPACE_ID}",
    "knowledge_base_id": "${INDEX_ID}",
    "access_key_id": "${ALIBABA_CLOUD_ACCESS_KEY_ID}",
    "access_key_secret": "${ALIBABA_CLOUD_ACCESS_KEY_SECRET}"
  }
}
```

#### 高级配置项

完整的配置选项（可选）：

```json
{
  "rag": {
    "enabled": true,
    
    // === 必填：API 配置 ===
    "provider": "bailian",
    "endpoint": "bailian.cn-beijing.aliyuncs.com",
    "workspace_id": "llm-xxx",
    "knowledge_base_id": "idx-xxx",
    "access_key_id": "LTAI5t...",
    "access_key_secret": "your-secret",
    
    // === 可选：检索配置 ===
    "context_mode": "full",           // 上下文模式: full | summary | highlights
    "max_context_length": 4000,       // 最大上下文长度（字符）
    "default_top_k": 3,               // 默认返回文档数量
    "similarity_threshold": 0.7,      // 相似度阈值（0-1）
    
    // === 可选：性能配置 ===
    "enable_cache": true,             // 启用查询缓存
    "cache_ttl": 3600,                // 缓存过期时间（秒）
    "cache_max_size": 1000,           // 最大缓存条目数
    "timeout": 10,                    // 请求超时（秒）
    "max_retries": 3,                 // 最大重试次数
    "retry_delay": 1,                 // 重试间隔（秒）
    
    // === 可选：降级策略 ===
    "fallback_on_error": true,        // RAG 失败时降级到基础模式
    
    // === 可选：工具级配置 ===
    "tools": {
      "generate_conversion_hints": {
        "use_rag": true,              // 为此工具启用 RAG
        "context_mode": "full",
        "top_k": 3
      },
      "validate_wasm_code": {
        "use_rag": true,
        "context_mode": "highlights",
        "top_k": 2
      }
    },
    
    // === 可选：调试配置 ===
    "debug": true,                    // 启用调试日志
    "log_queries": true               // 记录所有查询
  }
}
```

#### 验证配置

启动服务后，检查日志输出：

```bash
# 正常启用 RAG
✓ RAG enabled: bailian (endpoint: bailian.cn-beijing.aliyuncs.com)
✓ Knowledge base: idx-xxx
✓ Cache enabled (TTL: 3600s, Max: 1000)

# RAG 未启用
⚠ RAG disabled, using rule-based conversion only
```

## 使用示例

### 转换 Nginx 配置

使用 `convert_to_higress` 工具，传入 Nginx 配置内容：
- **默认**：生成 Kubernetes Ingress 和 Service 资源
- **可选**：设置 `use_gateway_api=true` 生成 Gateway API HTTPRoute（需确认已启用）


### 迁移 Lua 插件

**方式一：快速转换**

使用 `convert_lua_to_wasm` 工具一键转换 Lua 脚本为 WASM 插件。

**方式二：AI 辅助工具链**

1. 使用 `analyze_lua_plugin` 分析 Lua 代码
2. 使用 `generate_conversion_hints` 获取转换提示和 API 映射（可启用 RAG 增强）
3. AI 根据提示生成 Go WASM 代码
4. 使用 `validate_wasm_code` 验证生成的代码（可启用 RAG 增强）
5. 使用 `generate_deployment_config` 生成部署配置

推荐使用工具链方式处理复杂插件，可获得更好的转换质量和 AI 辅助。

### 查询知识库（需启用 RAG）

使用 `query_knowledge_base` 工具直接查询 Higress 文档：

```javascript
// 查询 Lua API 迁移方法
query_knowledge_base({
  "query": "ngx.req.get_headers 在 Higress 中怎么实现？",
  "scenario": "lua_migration",
  "top_k": 5
})

// 查询插件配置方法
query_knowledge_base({
  "query": "Higress 限流插件配置",
  "scenario": "config_conversion",
  "top_k": 3
})
```


## 项目结构

```
nginx-migration/
├── config/                     # 配置文件
│   ├── rag.json.example       # RAG 配置示例
│   └── rag.json               # RAG 配置（需自行创建）
│
├── integration/                # Higress 集成模式（MCP 集成）
│   ├── server.go              # MCP 服务器注册与初始化
│   └── mcptools/              # MCP 工具实现
│       ├── adapter.go         # MCP 工具适配器
│       ├── context.go         # 迁移上下文管理
│       ├── nginx_tools.go     # Nginx 配置转换工具
│       ├── lua_tools.go       # Lua 插件迁移工具
│       ├── tool_chain.go      # 工具链实现（分析、验证、部署）
│       └── rag_integration.go # RAG 知识库集成
│
├── standalone/                 # 独立模式（可独立运行）
│   ├── cmd/
│   │   └── main.go            # 独立模式入口
│   ├── server.go              # 独立模式 MCP 服务器
│   ├── config.go              # 配置加载
│   └── types.go               # 类型定义
│
├── internal/                   # 内部实现包
│   ├── rag/                   # RAG 功能实现
│   │   ├── config.go          # RAG 配置结构
│   │   ├── client.go          # 百炼 API 客户端
│   │   └── manager.go         # RAG 管理器（查询、缓存）
│   └── standalone/            # 独立模式内部实现
│       └── server.go          # 独立服务器逻辑
│
├── tools/                      # 核心转换逻辑（共享库）
│   ├── mcp_tools.go           # MCP 工具定义和注册
│   ├── nginx_parser.go        # Nginx 配置解析器
│   ├── lua_converter.go       # Lua 到 WASM 转换器
│   └── tool_chain.go          # 工具链核心实现
│
├── docs/                       # 文档目录
│
├── mcp-tools.json              # MCP 工具元数据定义
├── go.mod                      # Go 模块依赖
├── go.sum                      # Go 模块校验和
├── Makefile                    # 构建脚本
│
├── README.md                   # 项目说明文档
├── QUICKSTART.md               # 快速开始指南
├── QUICK_TEST.md               # 快速测试指南
├── TEST_EXAMPLES.md            # 测试示例
└── CHANGELOG_INGRESS_PRIORITY.md  # Ingress 优先级变更记录
```

### 目录说明

#### 核心模块

- **`integration/`** - Higress 集成模式
  - 作为 Higress MCP 服务器的一部分运行
  - 提供完整的 MCP 工具集成
  - 支持 RAG 知识库增强

- **`standalone/`** - 独立模式
  - 可独立运行的 MCP 服务器
  - 适合本地开发和测试
  - 支持相同的工具功能

- **`tools/`** - 核心转换逻辑
  - 独立于运行模式的转换引擎
  - 包含 Nginx 解析、Lua 转换等核心功能
  - 可被集成模式和独立模式复用

- **`internal/rag/`** - RAG 功能实现
  - 阿里云百炼 API 客户端
  - 知识库查询和结果处理
  - 缓存管理和性能优化


#### 配置文件

- **`config/rag.json`** - RAG 功能配置（需从 example 复制并填写凭证）
- **`mcp-tools.json`** - MCP 工具元数据定义（工具描述、参数 schema）

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


