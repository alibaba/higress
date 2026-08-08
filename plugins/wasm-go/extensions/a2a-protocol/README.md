# A2A Protocol

The `a2a-protocol` plugin extracts bounded, trusted metadata from A2A 1.0
JSON-RPC requests and unary/SSE responses. It removes client-provided
`x-higress-a2a-*` metadata before publishing canonical values for later
Higress policy plugins. Optional compatibility recognizes A2A 0.3 method
aliases without translating payloads.

The upstream Agent remains authoritative for task state. The plugin never
copies Parts, artifacts, credentials, or callback URLs into headers or logs.

```yaml
protocolVersion: "1.0"
mode: enforce
legacy03:
  enabled: false
agent:
  id: weather-agent
jsonrpc:
  maxRequestBytes: 4194304
  maxSSEEventBytes: 262144
authorization:
  exposeInternalHeaders: true
```
