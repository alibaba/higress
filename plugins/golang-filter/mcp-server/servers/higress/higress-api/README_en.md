# Higress API MCP Server

Higress API MCP Server provides MCP tools to manage Higress routes, service sources, AI routes, AI providers, MCP servers, plugins and other resources.

## Features

### Route Management
- `list-routes`: List routes
- `get-route`: Get route
- `add-route`: Add route
- `update-route`: Update route
- `delete-route`: Delete route

### AI Route Management
- `list-ai-routes`: List AI routes
- `get-ai-route`: Get AI route
- `add-ai-route`: Add AI route
- `update-ai-route`: Update AI route
- `delete-ai-route`: Delete AI route

### Service Source Management
- `list-service-sources`: List service sources
- `get-service-source`: Get service source
- `add-service-source`: Add service source
- `update-service-source`: Update service source
- `delete-service-source`: Delete service source

### AI Provider Management
- `list-ai-providers`: List LLM providers
- `get-ai-provider`: Get LLM provider
- `add-ai-provider`: Add LLM provider
- `update-ai-provider`: Update LLM provider
- `delete-ai-provider`: Delete LLM provider

### MCP Server Management
- `list-mcp-servers`: List MCP servers
- `get-mcp-server`: Get MCP server details
- `add-or-update-mcp-server`: Add or update MCP server
- `delete-mcp-server`: Delete MCP server
- `list-mcp-server-consumers`: List MCP server allowed consumers
- `add-mcp-server-consumers`: Add MCP server allowed consumers
- `delete-mcp-server-consumers`: Delete MCP server allowed consumers
- `swagger-to-mcp-config`: Convert Swagger content to MCP configuration

### Plugin Management
- `list-plugin-instances`: List all plugin instances for a specific scope (supports global, domain, service, and route levels)
- `get-plugin`: Get plugin configuration
- `delete-plugin`: Delete plugin
- `update-request-block-plugin`: Update request block configuration

## Configuration Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `higressURL` | string | Required | Higress Console URL address |
| `username` | string | Required | Higress Console login username |
| `password` | string | Required | Higress Console login password |
| `description` | string | Optional | MCP Server description |

Configuration Example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    meta.helm.sh/release-name: higress
    meta.helm.sh/release-namespace: higress-system
  labels:
    app: higress-gateway
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: higress-gateway
    app.kubernetes.io/version: 2.1.4
    helm.sh/chart: higress-core-2.1.4
    higress: higress-system-higress-gateway
  name: higress-config
  namespace: higress-system
data:
  higress: |-
    mcpServer:
      sse_path_suffix: /sse # SSE connection path suffix
      enable: true # Enable MCP Server
      redis:
        address: redis-stack-server.higress-system.svc.cluster.local:6379 # Redis service address
        username: "" # Redis username (optional)
        password: "" # Redis password (optional, plaintext)
        passwordSecret: # Reference password from Secret (recommended, higher priority than password)
          name: redis-credentials # Secret name
          key: password # Key in Secret
          namespace: higress-system # Secret namespace (optional, defaults to higress-system)
        db: 0 # Redis database (optional)
      match_list: # MCP Server session persistence routing rules (when matching the following paths, it will be recognized as an MCP session and maintained through SSE)
        - match_rule_domain: "*"
          match_rule_path: /higress-api
          match_rule_type: "prefix"
      servers:
        - name: higress-api-mcp-server # MCP Server name
          path: /higress-api # Access path, needs to match the configuration in match_list
          type: higress-api # Type defined in RegisterServer function
          config:
            higressURL: http://higress-console.higress-system.svc.cluster.local:8080
            username: admin
            password: admin
```
