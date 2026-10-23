# MCP Server
[English](./README_en.md) | 简体中文

## 概述

MCP Server 是一个基于 Envoy 的 Golang Filter 插件，提供了统一的 MCP (Model Context Protocol) 服务接口。它支持多种后端服务的集成，包括：

- 数据库服务：通过 GORM 支持多种数据库的访问和管理
- 配置中心：支持 Nacos 配置中心的集成
- 可扩展性：支持自定义服务器实现，方便集成其他服务

> **注意**：MCP Server 需要 Higress 2.1.0 或更高版本才能使用。

## MCP Server 开发指南

```go
// 在init函数中注册你的服务器
// 参数1: 服务器名称
// 参数2: 配置结构体实例
func init() {
	common.GlobalRegistry.RegisterServer("demo", &DemoConfig{})
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
func (c *DBConfig) NewServer(serverName string) (*common.MCPServer, error) {
	mcpServer := common.NewMCPServer(serverName, Version)
    
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
