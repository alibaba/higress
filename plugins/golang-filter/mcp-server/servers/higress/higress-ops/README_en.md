# Higress Ops MCP Server

Higress Ops MCP Server provides MCP tools for debugging and monitoring Istio and Envoy components, helping operations teams with troubleshooting and performance analysis.

## Features

### Istiod Debug Interfaces

#### Configuration
- `get-istiod-configz`: Get Istiod configuration status and error information

#### Service Discovery
- `get-istiod-endpointz`: Get all service endpoints discovered by Istiod
- `get-istiod-clusters`: Get all clusters discovered by Istiod
- `get-istiod-registryz`: Get Istiod service registry information

#### Status Monitoring
- `get-istiod-syncz`: Get synchronization status between Istiod and Envoy proxies
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
| `namespace` | string | Optional | Kubernetes namespace, defaults to `higress-system` |
| `istiodToken` | string | **Strongly Recommended** | Istiod authentication token (required for cross-pod access) |
| `description` | string | Optional | Server description, defaults to "Higress Ops MCP Server, which provides debug interfaces for Istio and Envoy components." |

### ⚠️ Important: Istiod Token Configuration

**The `istiodToken` must be configured when accessing Istiod interfaces across pods**, otherwise you will encounter 401 authentication errors.

#### Token Generation

Generate a long-lived Istiod authentication token with the following command:

```bash
kubectl create token higress-gateway -n higress-system --audience istio-ca --duration 87600h
```

**Parameter Description:**
- `higress-gateway`: ServiceAccount name (must match the ServiceAccount used by Higress Gateway Pod)
- `-n higress-system`: Namespace (must match the `namespace` configuration parameter)
- `--audience istio-ca`: Token audience, must be `istio-ca`
- `--duration 87600h`: Token validity period (87600 hours ≈ 10 years)

#### Configuration Status

- **Token Configured**: Logs will show "Istiod authentication token configured", Istiod interfaces can be accessed normally
- **Token Not Configured**: Logs will show warning "No istiodToken configured. Cross-pod Istiod API requests may fail with 401 errors.", cross-pod access will fail

## Configuration Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: higress-config
  namespace: higress-system
  resourceVersion: '107160'
data:
  higress: |
    mcpServer:
      sse_path_suffix: /sse  # SSE connection path suffix
      enable: true           # Enable MCP Server
      redis:
        address: redis-stack-server.higress-system.svc.cluster.local:6379  # Redis service address
        username: ""         # Redis username (optional)
        password: ""         # Redis password (optional)
        db: 0                # Redis database (optional)
      match_list:            # MCP Server session persistence routing rules
        - match_rule_domain: "*"
          match_rule_path: /higress-api
          match_rule_type: "prefix"
        - match_rule_domain: "*"
          match_rule_path: /higress-ops
          match_rule_type: "prefix"
        - match_rule_domain: "*"
          match_rule_path: /mysql
          match_rule_type: "prefix"
      servers:
        - name: higress-api-mcp-server     # MCP Server name
          path: /higress-api               # Access path, must match match_list configuration
          type: higress-api                # Type consistent with RegisterServer
          config:
            higressURL: http://higress-console.higress-system.svc.cluster.local:8080
            username: admin
            password: admin
        - name: higress-ops-mcp-server
          path: /higress-ops
          type: higress-ops
          config:
            istiodURL: http://higress-controller.higress-system.svc.cluster.local:15014   # istiod url
            istiodToken: "your_token"      # how to produce: kubectl create token higress-gateway -n higress-system --audience istio-ca --duration 87600h
            envoyAdminURL: http://127.0.0.1:15000 # envoy url, use 127.0.0.1 as it's in the same container as gateway
            namespace: higress-system
            description: "Higress Ops MCP Server for Istio and Envoy debugging"
  mesh: |-
    accessLogEncoding: TEXT
    accessLogFile: /dev/stdout
    accessLogFormat: '{"ai_log":"%FILTER_STATE(wasm.ai_log:PLAIN)%","authority":"%REQ(X-ENVOY-ORIGINAL-HOST?:AUTHORITY)%","bytes_received":"%BYTES_RECEIVED%","bytes_sent":"%BYTES_SENT%","downstream_local_address":"%DOWNSTREAM_LOCAL_ADDRESS%","downstream_remote_address":"%DOWNSTREAM_REMOTE_ADDRESS%","duration":"%DURATION%","istio_policy_status":"%DYNAMIC_METADATA(istio.mixer:status)%","method":"%REQ(:METHOD)%","path":"%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%","protocol":"%PROTOCOL%","request_id":"%REQ(X-REQUEST-ID)%","requested_server_name":"%REQUESTED_SERVER_NAME%","response_code":"%RESPONSE_CODE%","response_flags":"%RESPONSE_FLAGS%","route_name":"%ROUTE_NAME%","start_time":"%START_TIME%","trace_id":"%REQ(X-B3-TRACEID)%","upstream_cluster":"%UPSTREAM_CLUSTER%","upstream_host":"%UPSTREAM_HOST%","upstream_local_address":"%UPSTREAM_LOCAL_ADDRESS%","upstream_service_time":"%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%","upstream_transport_failure_reason":"%UPSTREAM_TRANSPORT_FAILURE_REASON%","user_agent":"%REQ(USER-AGENT)%","x_forwarded_for":"%REQ(X-FORWARDED-FOR)%","response_code_details":"%RESPONSE_CODE_DETAILS%"}'
    configSources:
    - address: xds://127.0.0.1:15051
    - address: k8s://
    defaultConfig:
      discoveryAddress: higress-controller.higress-system.svc:15012
      proxyStatsMatcher:
        inclusionRegexps:
        - .*
      tracing: {}
    dnsRefreshRate: 200s
    enableAutoMtls: false
    enablePrometheusMerge: true
    ingressControllerMode: "OFF"
    mseIngressGlobalConfig:
      enableH3: false
      enableProxyProtocol: false
    protocolDetectionTimeout: 100ms
    rootNamespace: higress-system
    trustDomain: cluster.local
  meshNetworks: 'networks: {}'
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
