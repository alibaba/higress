# nginx-migration-mcp

Nginx到Higress迁移工具的MCP实现。

## 使用方法

启动服务：
```bash
cd PATH/TO/higress/nginx-migration
go build -o nginx-migration-mcp
```

配置MCP客户端，添加以下配置：
```json
{
  "mcpServers": {
    "nginx-lua-converter": {
      "command": "PATH/TO/nginx-migration/nginx-migration-mcp",
      "args": [],
      "cwd": "PATH/TO/nginx-migration"
    }
  }
}
```
重启MCP客户端后即可使用。

## 功能

- `parse_nginx_config` - 解析nginx配置
- `convert_to_higress` - 转换为Higress HTTPRoute
- `analyze_lua_plugin` - 分析Lua插件兼容性
- `convert_lua_to_Wasm` - 将lua脚本转化为wasm插件


