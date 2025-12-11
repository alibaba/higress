# Nginx Migration MCP 快速开始


### 1. 构建服务器

```bash
cd /path/to/higress/plugins/golang-filter/mcp-server/servers/higress/nginx-migration
make build
```

### 2. 配置 MCP 客户端

在 MCP 客户端配置文件中添加（以 Cursor 为例）：

**位置**: `~/.cursor/config/mcp_settings.json` 或 Cursor 设置中的 MCP 配置

```json
{
  "mcpServers": {
    "nginx-migration": {
      "command": "/path/to/nginx-migration/nginx-migration-mcp",
      "args": []
    }
  }
}
```

## 可用工具

### Nginx 配置转换

 `parse_nginx_config` | 解析和分析 Nginx 配置文件 |
 `convert_to_higress` | 转换为 Higress HTTPRoute 和 Service |

### Lua 插件迁移


 `convert_lua_to_wasm`        | 一键转换 Lua 脚本为 WASM 插件 |
 `analyze_lua_plugin`         | 分析 Lua 插件兼容性 |
 `generate_conversion_hints`  | 生成 API 映射和转换提示 |
 `validate_wasm_code`         | 验证 Go WASM 代码 |
 `generate_deployment_config` | 生成部署配置包 |

## 使用示例

### 示例 1：转换 Nginx 配置

```
我有一个 Nginx 配置，帮我转换为 Higress HTTPRoute
```

HOST LLM 会自动调用 `convert_to_higress` 工具完成转换。

### 示例 2：快速转换 Lua 插件

```
将这个 Lua 限流插件转换为 Higress WASM 插件：
[粘贴 Lua 代码]
```

HOST LLM 会调用 `convert_lua_to_wasm` 工具自动转换。

### 示例 3：使用工具链精细转换

```
分析这个 Lua 插件的兼容性：
[粘贴 Lua 代码]
```

然后按照工具链流程：
1. LLM 调用 `analyze_lua_plugin` 分析
2. LLM 调用 `generate_conversion_hints` 获取转换提示
3. LLM 基于提示生成 Go WASM 代码
4. LLM 调用 `validate_wasm_code` 验证代码
5. LLM 调用 `generate_deployment_config` 生成部署配置

## 调试

启用调试日志：

```bash
DEBUG=true ./nginx-migration-mcp
```

查看工具列表：

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./nginx-migration-mcp
```

