# MCP Server
[English](./README_en.md) | 简体中文

## 概述

MCP Server 是一个基于 Envoy 的 Golang Filter 插件，用于实现服务器端事件（SSE）和消息通信功能。该插件支持多种数据库类型，并使用 Redis 作为消息队列来实现负载均衡的请求通过对应的SSE连接发送。

> **注意**：MCP Server需要 Higress 2.1.0 或更高版本才能使用。
## 项目结构
```
mcp-server/
├── config.go                # 配置解析相关代码
├── filter.go                # 请求处理相关代码
├── internal/                # 内部实现逻辑
├── servers/                 # MCP 服务器实现
├── go.mod                   # Go模块依赖定义
└── go.sum                   # Go模块依赖校验
```
## MCP Server开发指南

```go
// 在init函数中注册你的服务器
// 参数1: 服务器名称
// 参数2: 配置结构体实例
func init() {
	internal.GlobalRegistry.RegisterServer("demo", &DemoConfig{})
}

// 服务器配置结构体
type DemoConfig struct {
	helloworld string
}

// 解析配置方法
// 从配置map中解析并验证配置项
func (c *DBConfig) ParseConfig(config map[string]any) error {
	helloworld, ok := config["helloworld"].(string)
	if !ok { return errors.New("missing helloworld")}
	c.helloworld = helloworld
	return nil
}

// 创建新的MCP服务器实例
// serverName: 服务器名称
// 返回值: MCP服务器实例和可能的错误
func (c *DBConfig) NewServer(serverName string) (*internal.MCPServer, error) {
	mcpServer := internal.NewMCPServer(serverName, Version)
    
	// 添加工具方法到服务器
	// mcpServer.AddTool()	
	
	// 添加资源到服务器
	// mcpServer.AddResource()
	
	return mcpServer, nil
}
```

**Note**: 
需要在config.go里面使用下划线导入以执行包的init函数
```go
import (
	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/gorm"
)
```
