# Higress Ops MCP Server

Higress Ops MCP Server provides MCP tools for debugging and monitoring Istio and Envoy components, helping operations teams with troubleshooting and performance analysis.

## Features

### Istiod Debug Interfaces

#### Configuration
- `get-istiod-config-dump`: Get complete Istiod configuration snapshot including all xDS configs
- `get-istiod-configz`: Get Istiod configuration status and error information

#### Service Discovery
- `get-istiod-endpointz`: Get all service endpoints discovered by Istiod
- `get-istiod-clusters`: Get all clusters discovered by Istiod
- `get-istiod-registryz`: Get Istiod service registry information

#### Status Monitoring
- `get-istiod-syncz`: Get synchronization status between Istiod and Envoy proxies
- `get-istiod-proxy-status`: Get status of all proxies connected to Istiod
- `get-istiod-metrics`: Get Prometheus metrics from Istiod

#### System Information
- `get-istiod-version`: Get Istiod version information
- `get-istiod-debug-vars`: Get Istiod debug variables

### Envoy Debug Interfaces

#### Configuration
- `get-envoy-config-dump`: Get complete Envoy configuration snapshot with resource filtering and sensitive data masking
- `get-envoy-listeners`: Get all Envoy listener information
- `get-envoy-routes`: Get Envoy route configuration
- `get-envoy-clusters`: Get all Envoy cluster information and health status

#### Runtime
- `get-envoy-stats`: Get Envoy statistics with filtering and multiple output formats
- `get-envoy-runtime`: Get Envoy runtime configuration
- `get-envoy-memory`: Get Envoy memory usage

#### Status Check
- `get-envoy-server-info`: Get Envoy server basic information
- `get-envoy-ready`: Check if Envoy is ready
- `get-envoy-hot-restart-version`: Get Envoy hot restart version

#### Security
- `get-envoy-certs`: Get Envoy certificate information

## Configuration Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `istiodURL` | string | Yes | URL address of Istiod debug interface |
| `envoyAdminURL` | string | Yes | URL address of Envoy Admin interface |
| `namespace` | string | Optional | Kubernetes namespace, defaults to istio-system |
| `description` | string | Optional | Server description |

## Configuration Example

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
      match_list: # MCP Server session persistence routing rules
        - match_rule_domain: "*"
          match_rule_path: /higress-ops
          match_rule_type: "prefix"
      servers:
        - name: higress-ops-mcp-server # MCP Server name
          path: /higress-ops # Access path, must match match_list configuration
          type: higress-ops # Type consistent with RegisterServer
          config:
            istiodURL: http://istiod.istio-system.svc.cluster.local:15014
            envoyAdminURL: http://higress-gateway.higress-system.svc.cluster.local:15000
            namespace: istio-system
```

## Use Cases

### 1. Troubleshooting
- Use `get-istiod-syncz` to check configuration sync status
- Use `get-envoy-clusters` to check cluster health status
- Use `get-envoy-listeners` to check listener configuration

### 2. Performance Analysis
- Use `get-istiod-metrics` to get Istiod performance metrics
- Use `get-envoy-stats` to get Envoy statistics
- Use `get-envoy-memory` to monitor memory usage

### 3. Configuration Validation
- Use `get-istiod-config-dump` to validate Istiod configuration
- Use `get-envoy-config-dump` to validate Envoy configuration
- Use `get-envoy-routes` to check route configuration

### 4. Security Audit
- Use `get-envoy-certs` to check certificate status
- Use `get-istiod-debug-vars` to view debug variables

## Tool Parameter Examples

### Istiod Tool Examples

```bash
# Get specific proxy status
get-istiod-proxy-status --proxy="gateway-proxy.istio-system"

# Get configuration dump
get-istiod-config-dump

# Get sync status
get-istiod-syncz
```

### Envoy Tool Examples

```bash
# Get config dump, filter listeners
get-envoy-config-dump --resource="listeners"

# Get cluster info in JSON format
get-envoy-clusters --format="json"

# Get stats containing "cluster", JSON format
get-envoy-stats --filter="cluster.*" --format="json"

# Get specific route table info
get-envoy-routes --name="80" --format="json"
```

## FAQ

### Q: How to get detailed information for a specific cluster?
A: Use `get-envoy-clusters` tool, then use `get-envoy-config-dump --resource="clusters"` for detailed configuration.

### Q: How to monitor configuration sync status?
A: Use `get-istiod-syncz` for overall sync status, use `get-istiod-proxy-status` for specific proxy status.

### Q: How to troubleshoot routing issues?
A: Use `get-envoy-routes` to view route configuration, use `get-envoy-config-dump --resource="routes"` for detailed route information.

### Q: What output formats are supported?
A: Most tools support text and json formats, statistics also support prometheus format.
