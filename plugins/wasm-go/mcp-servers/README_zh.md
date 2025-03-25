# MCP 服务器实现指南

本指南介绍如何使用 Higress WASM Go SDK 实现 Model Context Protocol (MCP) 服务器。MCP 服务器提供工具和资源，扩展 AI 助手的能力。

## 概述

MCP 服务器是一个独立的应用程序，通过 Model Context Protocol 与 AI 助手通信。它可以提供：

- **工具**：可以被 AI 调用以执行特定任务的函数
- **资源**：可以被 AI 访问的数据

> **注意**：MCP 服务器插件需要 Higress 2.1.0 或更高版本才能使用。

## 项目结构

一个典型的 MCP 服务器项目具有以下结构：

```
my-mcp-server/
├── go.mod                 # Go 模块定义
├── go.sum                 # Go 模块校验和
├── main.go                # 注册工具和资源的入口点
├── server/
│   └── server.go          # 服务器配置和解析
└── tools/
    └── my_tool.go         # 工具实现
```

## 服务器配置

服务器配置定义了服务器运行所需的参数。例如：

```go
// server/server.go
package server

import (
    "encoding/json"
    "errors"

    "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

// 定义服务器配置结构
type MyMCPServer struct {
    ApiKey string `json:"apiKey"`
    // 根据需要添加其他配置字段
}

// 验证配置
func (s MyMCPServer) ConfigHasError() error {
    if s.ApiKey == "" {
        return errors.New("missing api key")
    }
    return nil
}

// 从 JSON 解析配置
func ParseFromConfig(configBytes []byte, server *MyMCPServer) error {
    return json.Unmarshal(configBytes, server)
}

// 从 HTTP 请求解析配置
func ParseFromRequest(ctx wrapper.HttpContext, server *MyMCPServer) error {
    return ctx.ParseMCPServerConfig(server)
}
```

## 工具实现

每个工具应该实现为一个具有以下方法的结构体：

1. `Description()`：返回工具的描述
2. `InputSchema()`：返回工具输入参数的 JSON schema
3. `Create()`：使用提供的参数创建工具的新实例
4. `Call()`：执行工具的功能

示例：

```go
// tools/my_tool.go
package tools

import (
    "encoding/json"
    "fmt"
    "net/http"
    
    "my-mcp-server/server"
    
    "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

// 定义带有输入参数的工具结构
type MyTool struct {
    Param1 string `json:"param1" jsonschema_description:"参数1的描述" jsonschema:"example=示例值"`
    Param2 int    `json:"param2,omitempty" jsonschema_description:"参数2的描述" jsonschema:"default=5"`
}

// Description 返回 MCP 工具定义的描述字段。
// 这对应于 MCP 工具 JSON 响应中的 "description" 字段，
// 提供了工具目的和用法的人类可读解释。
func (t MyTool) Description() string {
    return `详细描述这个工具做什么以及何时使用它。`
}

// InputSchema 返回 MCP 工具定义的 inputSchema 字段。
// 这对应于 MCP 工具 JSON 响应中的 "inputSchema" 字段，
// 定义了工具输入参数的 JSON Schema，包括属性类型、描述和必填字段。
func (t MyTool) InputSchema() map[string]any {
    return wrapper.ToInputSchema(&MyTool{})
}

// Create 基于 MCP 工具调用的输入参数实例化一个新的工具实例。
// 它将 JSON 参数反序列化为结构体，为可选字段应用默认值，并返回配置好的工具实例。
func (t MyTool) Create(params []byte) wrapper.MCPTool[server.MyMCPServer] {
    myTool := &MyTool{
        Param2: 5, // 默认值
    }
    json.Unmarshal(params, &myTool)
    return myTool
}

// Call 实现处理 MCP 工具调用的核心逻辑。当通过 MCP 框架调用工具时，执行此方法。
// 它处理配置的参数，进行必要的 API 请求，并格式化返回给调用者的结果。
func (t MyTool) Call(ctx wrapper.HttpContext, config server.MyMCPServer) error {
    // 验证配置
    err := server.ParseFromRequest(ctx, &config)
    if err != nil {
        return err
    }
    err = config.ConfigHasError()
    if err != nil {
        return err
    }
    
    // 在这里实现工具的逻辑
    // ...
    
    // 返回结果
    ctx.SendMCPToolTextResult(fmt.Sprintf("结果: %s, %d", t.Param1, t.Param2))
    return nil
}
```

## 主入口点

main.go 文件是 MCP 服务器的入口点。它注册工具和资源：

```go
// main.go
package main

import (
    "my-mcp-server/server"
    "my-mcp-server/tools"
    
    "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

func main() {}

func init() {
    wrapper.SetCtx(
        "my-mcp-server", // 服务器名称
        wrapper.ParseRawConfig(server.ParseFromConfig),
        wrapper.AddMCPTool("my_tool", tools.MyTool{}), // 注册工具
        // 根据需要添加更多工具
    )
}
```

## 依赖项

您的 MCP 服务器必须使用支持 Go 1.24 WebAssembly 编译功能的特定版本的 wasm-go SDK：

```bash
# 添加具有特定版本标签的必需依赖项
go get github.com/alibaba/higress/plugins/wasm-go@wasm-go-1.24
```

## 构建 WASM 二进制文件

要将 Go 代码编译为 WebAssembly (WASM) 文件，请使用以下命令：

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm main.go
```

此命令将目标操作系统设置为 `wasip1`（WebAssembly 系统接口）和架构设置为 `wasm`（WebAssembly），然后将代码构建为 C 共享库并输出为 `main.wasm`。

## 使用 Makefile

提供了 Makefile 以简化构建过程。它包括以下目标：

- `make build`：为 MCP 服务器构建 WASM 二进制文件
- `make build-image`：构建包含 MCP 服务器的 Docker 镜像
- `make build-push`：构建并将 Docker 镜像推送到注册表
- `make clean`：删除构建产物
- `make help`：显示可用的目标和变量

您可以通过设置以下变量来自定义构建：

```bash
# 使用自定义服务器名称构建
make SERVER_NAME=my-mcp-server build

# 使用自定义注册表构建
make REGISTRY=my-registry.example.com/ build-image

# 使用特定版本标签构建
make SERVER_VERSION=1.0.0 build-image
```

## 测试

您可以为工具创建单元测试以验证其功能：

```go
// tools/my_tool_test.go
package tools

import (
    "encoding/json"
    "fmt"
    "testing"
)

func TestMyToolInputSchema(t *testing.T) {
    myTool := MyTool{}
    schema := myTool.InputSchema()
    
    schemaJSON, err := json.MarshalIndent(schema, "", "  ")
    if err != nil {
        t.Fatalf("无法将 schema 序列化为 JSON: %v", err)
    }
    
    fmt.Printf("MyTool InputSchema:\n%s\n", string(schemaJSON))
    
    if len(schema) == 0 {
        t.Error("InputSchema 返回了空 schema")
    }
}
