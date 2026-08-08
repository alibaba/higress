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
  externalBaseURL: https://agents.example.com/a2a
agentCard:
  path: /.well-known/agent-card.json
  rewrite: true
  signatureMode: preserve
  maxResponseBytes: 262144
jsonrpc:
  maxRequestBytes: 4194304
  maxSSEEventBytes: 262144
authorization:
  exposeInternalHeaders: true
```

GET responses from the canonical Agent Card path and the legacy
`/.well-known/agent.json` path are validated before forwarding. The plugin
accepts public HTTPS JSON-RPC endpoints, preserves unknown fields, and rewrites
only declared endpoint fields to `agent.externalBaseURL` (or the sanitized
external request origin). Signed Cards are passed through unchanged in
`preserve` mode.
