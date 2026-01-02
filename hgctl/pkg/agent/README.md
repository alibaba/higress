// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

# Agent Module

`pkg/agent` 是 hgctl 中用于 Agent 生命周期管理的核心模块，提供了从创建、配置、部署到发布的完整工作流。

## 目录

- [概述](#概述)
- [架构设计](#架构设计)
- [核心功能](#核心功能)
- [主要组件](#主要组件)
- [使用方式](#使用方式)
- [配置管理](#配置管理)
- [部署方式](#部署方式)
- [集成说明](#集成说明)

## 概述

Agent 模块提供了一套完整的 AI Agent 开发和部署解决方案，支持：

- **多种 Agentic Core**：集成 Claude Code 和 Qodercli
- **本地和云端部署**：支持本地运行和 AgentRun (阿里云函数计算)
- **MCP Server 管理**：支持 HTTP 和 OpenAPI 类型的 MCP Server
- **Higress 集成**：自动发布 Agent API 到 Higress 网关
- **Himarket 发布**：支持将 Agent 发布到 Himarket 市场

## 架构设计

```
pkg/agent/
├── agent.go           # CLI 命令入口和主要业务逻辑
├── core.go           # Agentic Core (Claude/Qodercli) 封装
├── new.go            # Agent 创建流程
├── deploy.go         # Agent 部署处理（本地/云端）
├── mcp.go            # MCP Server 管理
├── config.go         # 配置管理和初始化
├── base.go           # 基础函数和环境检查
├── types.go          # 类型定义
├── utils.go          # 工具函数
├── common/           # 通用类型定义
│   └── base.go       # ProductType 等常量
├── services/         # 外部服务客户端
│   ├── client.go     # HTTP 客户端封装
│   ├── service.go    # Higress/Himarket API 封装
│   └── utils.go      # 服务工具函数
└── prompt/           # Prompt 模板和指导
    ├── base.go       # Agent 开发指南
    └── agent_guide.md
```

## 核心功能

### 1. Agent 创建 (new.go)

提供两种 Agent 创建方式：

#### 1.1 交互式创建
通过命令行交互式问答，逐步配置：
- Agent 名称和描述
- 系统 Prompt（支持直接输入、从文件导入、LLM 生成）
- AI 模型配置（DashScope、OpenAI、Anthropic 等）
- 工具集选择（AgentScope 内置工具）
- MCP Server 配置
- 部署设置

#### 1.2 从 Core 导入
从 Agentic Core 的 subagent 目录导入已有的 Agent 配置。

**关键代码位置**:
- `createAgentCmd()` (new.go:99): 创建命令定义
- `getAgentConfig()` (utils.go:289): 获取 Agent 配置
- `createAgentTemplate()` (new.go:205): 生成 Agent 模板文件

### 2. Agent 部署 (deploy.go)

支持两种部署模式：

#### 2.1 本地部署 (Local)
- 基于 AgentScope Runtime
- 自动管理 Python 虚拟环境
- 依赖管理：`agentscope`, `agentscope-runtime==1.0.0`
- 默认端口：8090

#### 2.2 云端部署 (AgentRun)
- 部署到阿里云函数计算
- 使用 Serverless Devs (s工具)
- 自动构建和部署
- 需要配置阿里云 Access Key

**关键代码位置**:
- `DeployHandler` (deploy.go:35): 部署处理器
- `HandleLocal()` (deploy.go:350): 本地部署逻辑
- `HandleAgentRun()` (deploy.go:305): AgentRun 部署逻辑

### 3. MCP Server 管理 (mcp.go)

支持两种类型的 MCP Server：

#### 3.1 HTTP MCP Server
直接通过 HTTP URL 添加：
```bash
hgctl mcp add [name] [url] --type http
```

#### 3.2 OpenAPI MCP Server
从 OpenAPI 规范文件创建：
```bash
hgctl mcp add [name] [spec-file] --type openapi
```

功能特性：
- 自动解析 OpenAPI 规范
- 转换为 MCP Server 配置
- 自动添加到 Agentic Core
- 可选发布到 Higress
- 支持发布到 Himarket 市场

**关键代码位置**:
- `handleAddMCP()` (mcp.go:183): MCP 添加主逻辑
- `publishMCPToHigress()` (mcp.go:228): 发布到 Higress
- `parseOpenapi2MCP()` (utils.go:79): OpenAPI 解析

### 4. Agentic Core 集成 (core.go)

封装了 Agentic Core（Claude Code/Qodercli）的交互：

#### 支持的 Core 类型
```go
const (
    CORE_CLAUDE   CoreType = "claude"
    CORE_QODERCLI CoreType = "qodercli"
)
```

#### 核心功能
- **Setup()**: 初始化环境和插件
- **Start()**: 启动交互式窗口
- **AddMCPServer()**: 添加 MCP Server 到 Core
- **ImproveNewAgent()**: 在特定 Agent 目录运行 Core 进行改进

**关键代码位置**:
- `AgenticCore` (core.go:32): Core 封装结构
- `Setup()` (core.go:108): 环境初始化
- `addHigressAPIMCP()` (core.go:161): 自动添加 Higress API MCP

### 5. Higress 集成

自动将 Agent API 发布到 Higress 网关：

#### 支持的 API 类型
```go
const (
    A2A   = "a2a"      // Agent-to-Agent
    REST  = "restful"  // RESTful API
    MODEL = "model"    // AI Model API
)
```

#### 发布流程
1. 创建 AI Provider Service
2. 创建 AI Route
3. 配置服务源和路由

**关键代码位置**:
- `publishAgentAPIToHigress()` (agent.go:123): 发布逻辑
- `services/service.go`: Higress API 封装

### 6. Himarket 集成

支持将 Agent 发布到 Himarket 市场：

#### 产品类型
```go
const (
    MCP_SERVER ProductType = "MCP_SERVER"
    MODEL_API  ProductType = "MODEL_API"
    REST_API   ProductType = "REST_API"
    AGENT_API  ProductType = "AGENT_API"
)
```

**关键代码位置**:
- `publishAPIToHimarket()` (base.go:128): 发布到市场
- `services/service.go`: Himarket API 封装

## 主要组件

### AgentConfig 结构

Agent 的核心配置结构：

```go
type AgentConfig struct {
    AppName         string              // 应用名称
    AppDescription  string              // 应用描述
    AgentName       string              // Agent 名称
    AvailableTools  []string            // 可用工具列表
    SysPromptPath   string              // 系统 Prompt 路径
    ChatModel       string              // 使用的模型
    Provider        string              // 模型提供商
    APIKeyEnvVar    string              // API Key 环境变量
    DeploymentPort  int                 // 部署端口
    HostBinding     string              // 主机绑定
    EnableStreaming bool                // 是否启用流式响应
    EnableThinking  bool                // 是否启用思考过程
    MCPServers      []MCPServerConfig   // MCP Server 配置
    Type            DeployType          // 部署类型
    ServerlessCfg   ServerlessConfig    // Serverless 配置
}
```

### 环境检查 (base.go)

`EnvProvisioner` 负责检查和安装必要的环境：

#### Node.js 检查
- 最低版本要求：Node.js 18+
- 支持自动安装（通过 fnm）

#### Agentic Core 检查
- 检查 claude 或 qodercli 是否安装
- 支持自动安装（通过 npm）

**关键代码位置**:
- `EnvProvisioner.check()` (base.go:221): 环境检查
- `promptNodeInstall()` (base.go:259): Node.js 安装引导
- `promptAgentInstall()` (base.go:401): Core 安装引导

## 使用方式

### 命令结构

```bash
hgctl agent                    # 启动交互式 Agent 窗口
hgctl agent new               # 创建新 Agent
hgctl agent deploy [name]     # 部署 Agent
hgctl agent add [name] [url]  # 添加 Agent API 到 Higress
hgctl mcp add [name] [url]    # 添加 MCP Server
```

### 创建 Agent

#### 本地部署的 Agent
```bash
hgctl agent new
```

交互式选择：
1. 创建方式：step by step / 从 Core 导入
2. Agent 名称和描述
3. 系统 Prompt 设置
4. 模型提供商和模型选择
5. 工具选择
6. MCP Server 配置
7. 部署设置

#### AgentRun 部署的 Agent
```bash
hgctl agent new --agent-run
```

额外配置：
- Resource Name
- Region
- Disk Size
- Timeout

### 部署 Agent

#### 部署到本地
```bash
hgctl agent deploy my-agent
```

自动处理：
- Python 环境检查
- 依赖安装
- 启动 Agent 服务

#### 部署到 AgentRun
```bash
hgctl agent deploy my-agent
```

要求：
- 已配置阿里云 Access Key
- 已安装 Docker
- 已安装 Serverless Devs CLI

### 添加 MCP Server

#### 添加 HTTP MCP Server
```bash
hgctl mcp add my-mcp http://localhost:8080/mcp \
  --type http \
  --transport streamable \
  -e API_KEY=secret \
  -H "Authorization: Bearer token"
```

参数说明：
- `--type`: MCP 类型（http/openapi）
- `--transport`: 传输类型（streamable/sse）
- `-e`: 环境变量
- `-H`: HTTP 头部

#### 从 OpenAPI 创建 MCP Server
```bash
hgctl mcp add swagger-mcp ./openapi.yaml \
  --type openapi
```

自动完成：
1. 解析 OpenAPI 规范
2. 转换为 MCP 配置
3. 发布到 Higress
4. 添加到 Agentic Core

### 发布到 Higress 和 Himarket

```bash
hgctl agent add my-agent http://my-agent.com \
  --type model \
  --as-product \
  --higress-console-url http://console.higress.io \
  --higress-console-user admin \
  --higress-console-password password \
  --himarket-admin-url http://himarket.io \
  --himarket-admin-user admin \
  --himarket-admin-password password
```

## 配置管理

### 配置文件

配置文件位置：`~/.hgctl`

```json
{
  "hgctl-agent-core": "claude",
  "agent-chat-model": "qwen-plus",
  "agent-model-provider": "DashScope",
  "higress-console-url": "http://127.0.0.1:8080",
  "higress-console-user": "admin",
  "higress-console-password": "admin",
  "higress-gateway-url": "http://127.0.0.1:80",
  "himarket-admin-url": "",
  "himarket-admin-user": "",
  "himarket-admin-password": "",
  "agentrun-model-name": "",
  "agentrun-region": "cn-hangzhou"
}
```

### 配置项说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `hgctl-agent-core` | Agentic Core 类型 | `qodercli` |
| `agent-chat-model` | 默认聊天模型 | - |
| `agent-model-provider` | 默认模型提供商 | - |
| `higress-console-url` | Higress 控制台地址 | - |
| `higress-console-user` | Higress 用户名 | - |
| `higress-console-password` | Higress 密码 | - |
| `higress-gateway-url` | Higress 网关地址 | - |
| `himarket-admin-url` | Himarket 管理地址 | - |
| `himarket-admin-user` | Himarket 用户名 | - |
| `himarket-admin-password` | Himarket 密码 | - |
| `agentrun-model-name` | AgentRun 模型名 | - |
| `agentrun-region` | AgentRun 区域 | `cn-hangzhou` |

### 环境变量

配置也可以通过环境变量设置（自动转换，用下划线替换连字符）：

```bash
export HIGRESS_CONSOLE_URL=http://127.0.0.1:8080
export HIGRESS_CONSOLE_USER=admin
export HIGRESS_CONSOLE_PASSWORD=admin
```

**代码位置**: `config.go:100` - `InitConfig()`

## 部署方式

### 本地部署 (Local)

#### 技术栈
- **Runtime**: AgentScope Runtime
- **Python**: 3.12+
- **依赖**:
  - `agentscope`
  - `agentscope-runtime==1.0.0`

#### 部署流程
1. 检查 Python 环境
2. 创建/激活虚拟环境 (`~/.hgctl/.venv`)
3. 安装依赖
4. 启动 Agent 服务

#### 生成的文件
```
~/.hgctl/agents/{agent-name}/
├── as_runtime_main.py    # AgentScope Runtime 入口
├── agent.py              # Agent 类定义
├── toolkit.py            # 工具集
├── prompt.md             # 系统 Prompt
├── CLAUDE.md             # Claude 开发指南（如果使用 Claude）
└── AGENTS.md             # Qoder 开发指南（如果使用 Qodercli）
```

**代码位置**: `deploy.go:350` - `HandleLocal()`

### 云端部署 (AgentRun)

#### 技术栈
- **平台**: 阿里云函数计算 (Function Compute)
- **SDK**: agentrun-sdk-python
- **工具**: Serverless Devs CLI

#### 部署流程
1. 检查环境（Docker、Serverless Devs）
2. 检查/配置 Access Key
3. 执行 `s build`
4. 执行 `s deploy`

#### 生成的文件
```
~/.hgctl/agents/{agent-name}/
├── agentrun_main.py      # AgentRun 入口
├── agent.py              # Agent 类定义
├── toolkit.py            # 工具集
├── prompt.md             # 系统 Prompt
├── requirements.txt      # Python 依赖
└── s.yaml                # Serverless Devs 配置
```

#### s.yaml 配置
```yaml
edition: 3.0.0
name: {agent-name}
access: hgctl-credential

resources:
  fc-agentrun-demo:
    component: fc3
    props:
      region: {region}
      description: {description}
      runtime: python3.12
      code: ./
      handler: agentrun_main.main
      timeout: {timeout}
      diskSize: {disk-size}
      environmentVariables:
        MODEL_NAME: {model-name}
        {api-key-env}: {api-key}
      customRuntimeConfig:
        command:
          - python3
        args:
          - agentrun_main.py
        port: {port}
```

**代码位置**: `deploy.go:305` - `HandleAgentRun()`

## 集成说明

### Higress 集成

#### Service Source 创建
```go
// services/utils.go
func BuildServiceBodyAndSrv(name, rawURL string) (map[string]interface{}, string, int, error)
```

创建服务源：
- 解析 URL
- 提取域名、端口
- 生成服务名称

#### AI Provider 和 Route 创建

对于 MODEL 类型的 Agent：
```go
// services/utils.go
func BuildAIProviderServiceBody(name, url string) map[string]interface{}
func BuildAddAIRouteBody(name, url string) map[string]interface{}
```

#### MCP Server 创建

支持两种类型：
- **DIRECT_ROUTE**: 直接路由到 MCP Server URL
- **OPEN_API**: 基于 OpenAPI 规范的工具配置

### Himarket 集成

#### API Product 创建
```go
// services/utils.go
func BuildAPIProductBody(name, desc string, typ string) map[string]interface{}
```

#### Product Reference
```go
func BuildRefModelAPIProductBody(gatewayId, productId, routeName string) map[string]interface{}
func BuildRefMCPAPIProductBody(gatewayId, productId, mcpServerName string) map[string]interface{}
```

### Agentic Core 集成

#### 初始化流程
1. 提取 manifest 文件到 `~/.hgctl/`
2. 提取 Core 相关文件到 `~/.claude/` 或 `~/.qoder/`
3. 添加预定义的 MCP Server
4. 自动配置 Higress API MCP Server

#### MCP Server 添加
```bash
{core} mcp add --transport {transport} {name} {url} \
  --scope {scope} \
  -e {env} \
  -H {header}
```

**代码位置**: `core.go:236` - `AddMCPServer()`

## 类型定义 (types.go)

### API 请求/响应类型

用于与 AI 模型 API 交互：

```go
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type Request struct {
    Model            string    `json:"model"`
    Messages         []Message `json:"messages"`
    FrequencyPenalty float64   `json:"frequency_penalty"`
    PresencePenalty  float64   `json:"presence_penalty"`
    Stream           bool      `json:"stream"`
    Temperature      float64   `json:"temperature"`
    Topp             int32     `json:"top_p"`
}

type Response struct {
    ID      string   `json:"id"`
    Choices []Choice `json:"choices"`
    Created int64    `json:"created"`
    Model   string   `json:"model"`
    Object  string   `json:"object"`
    Usage   Usage    `json:"usage"`
}
```

### OpenAPI 相关类型

用于 OpenAPI 规范解析：

```go
type API struct {
    OpenAPI    string     `yaml:"openapi"`
    Info       Info       `yaml:"info"`
    Servers    []Server   `yaml:"servers"`
    Paths      Paths      `yaml:"paths"`
    Components Components `yaml:"components"`
}
```

## Services 子包

### HigressClient

Higress API 客户端：

```go
type HigressClient struct {
    baseURL  string
    username string
    password string
    client   *http.Client
}
```

**主要方法**:
- `Get(path string) ([]byte, error)`
- `Post(path string, body interface{}) ([]byte, error)`
- `Put(path string, body interface{}) ([]byte, error)`

### HimarketClient

Himarket API 客户端：

```go
type HimarketClient struct {
    baseURL  string
    username string
    password string
    client   *http.Client
}
```

**主要方法**:
- `GetDevMCPServerProduct() (map[string]string, error)`
- `GetDevModelProduct() (map[string]string, error)`

## 工具函数 (utils.go)

### Kubernetes 相关

- `GetHigressGatewayServiceIP()`: 获取 Higress Gateway Service IP
- `extractServiceIP()`: 从 Service 提取 IP
- `getConsoleCredentials()`: 从 K8s Secret 获取控制台凭证

### Agent 配置

- `getAgentConfig()`: 交互式获取 Agent 配置
- `createAgentStepByStep()`: 逐步创建 Agent
- `importAgentFromCore()`: 从 Core 导入 Agent

### Query 函数

一系列用于交互式配置查询的函数：
- `queryAgentSysPrompt()`: 查询系统 Prompt
- `queryAgentTools()`: 查询工具选择
- `queryAgentModel()`: 查询模型配置
- `queryAgentMCP()`: 查询 MCP Server
- `queryDeploySettings()`: 查询部署设置

## 最佳实践

### 1. 开发流程

```bash
# 1. 创建 Agent
hgctl agent new

# 2. 使用 Core 改进和测试
# 选择 "Improve and test it using agentic core"

# 3. 部署 Agent
hgctl agent deploy my-agent

# 4. 添加到 Higress
hgctl agent add my-agent http://localhost:8090 --type model

# 5. （可选）发布到 Himarket
hgctl agent add my-agent http://localhost:8090 --type model --as-product
```

### 2. MCP Server 管理

```bash
# 添加 HTTP MCP Server
hgctl mcp add my-mcp http://mcp-server:8080/mcp

# 从 OpenAPI 创建 MCP Server
hgctl mcp add swagger-mcp ./openapi.yaml --type openapi

# 添加到 Higress 和 Himarket
hgctl mcp add my-mcp http://mcp-server:8080/mcp --as-product
```

### 3. 配置管理

```bash
# 使用配置文件
vim ~/.hgctl

# 或使用环境变量
export HIGRESS_CONSOLE_URL=http://127.0.0.1:8080
export HIGRESS_CONSOLE_USER=admin
export HIGRESS_CONSOLE_PASSWORD=admin
```

## 错误处理

### 常见错误

1. **Node.js 未安装**
   - 自动提示安装选项
   - 支持自动安装（fnm）

2. **Agentic Core 未安装**
   - 自动提示安装选项
   - 支持自动安装（npm）

3. **Python 环境问题**
   - 自动创建虚拟环境
   - 自动安装依赖

4. **Kubernetes 连接问题**
   - 提供手动输入 kubeconfig 选项
   - 支持自定义 namespace

5. **Higress/Himarket 认证失败**
   - 检查配置文件
   - 检查环境变量
   - 尝试从 K8s Secret 自动获取

## 扩展开发

### 添加新的 Agentic Core

1. 在 `config.go` 中添加新的 CoreType
2. 在 `core.go` 中实现相应的方法
3. 更新 `EnvProvisioner` 支持新的安装方式

### 添加新的部署类型

1. 在 `deploy.go` 中添加新的 DeployType
2. 实现相应的部署处理方法
3. 更新模板生成逻辑

### 添加新的 API 类型

1. 在 `agent.go` 中添加新的 API Type 常量
2. 在 `publishAgentAPIToHigress()` 中添加处理逻辑
3. 在 `services/utils.go` 中添加相应的构建函数

## 依赖说明

### Go 依赖
- `github.com/spf13/cobra`: CLI 框架
- `github.com/spf13/viper`: 配置管理
- `github.com/AlecAivazis/survey/v2`: 交互式问答
- `github.com/fatih/color`: 终端颜色输出
- `k8s.io/client-go`: Kubernetes 客户端

### 外部工具
- **Node.js 18+**: Agentic Core 运行环境
- **Claude Code / Qodercli**: Agentic Core
- **Python 3.12+**: Agent Runtime
- **Docker**: AgentRun 部署
- **Serverless Devs CLI**: AgentRun 部署工具

## 参考资源

- [AgentScope 文档](https://modelscope.github.io/agentscope/)
- [Claude Code 文档](https://docs.claude.com/en/docs/claude-code/setup)
- [Qoder 文档](https://docs.qoder.com/zh/cli/quick-start)
- [Serverless Devs 文档](https://serverless-devs.com/docs/user-guide/install)
- [Higress 文档](https://higress.io/)
- [AgentRun 文档](https://github.com/Serverless-Devs/agentrun-sdk-python)

## License

Copyright (c) 2025 Alibaba Group Holding Ltd.

Licensed under the Apache License, Version 2.0