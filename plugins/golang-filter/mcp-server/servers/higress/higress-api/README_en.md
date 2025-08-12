# Higress API MCP Server

Higress API MCP Server provides MCP tools to manage Higress routes, service sources, plugins and other resources.

## Features

### Route Management
- `list-routes`: List routes
- `get-route`: Get route
- `add-route`: Add route
- `update-route`: Update route

### Service Source Management
- `list-service-sources`: List service sources
- `get-service-source`: Get service source
- `add-service-source`: Add service source
- `update-service-source`: Update service source

### Plugin Management
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
        password: "" # Redis password (optional)
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
