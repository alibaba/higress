# nginx-migration-mcp

Nginx到Higress迁移工具的MCP服务器实现。

## 使用方法

启动服务：
```bash
go run main.go config.go
```

配置MCP客户端，添加以下配置：
```json
{
  "mcpServers": {
    "nginx-migration": {
      "command": "go",
      "args": ["run", "main.go", "config.go"],
      "cwd": "/path/to/nginx-migration-mcp"
    }
  }
}
```

重启MCP客户端后即可使用。

## 功能

- `parse_nginx_config` - 解析nginx配置
- `convert_to_higress` - 转换为Higress HTTPRoute
- `analyze_lua_plugin` - 分析Lua插件兼容性

## 配置

通过环境变量自定义：
```bash
HIGRESS_GATEWAY_NAME=my-gateway go run main.go config.go
```

## HTTP模式

如需HTTP API访问：
```bash
go run http-server.go config.go
curl http://localhost:8080/health
```
