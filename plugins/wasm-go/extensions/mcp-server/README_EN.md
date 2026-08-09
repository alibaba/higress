# mcp-server

[中文](./README.md)

## Overview

`mcp-server` hosts MCP tool servers in the gateway. The current binary includes Quark Search and Amap tools, and it can also expose configured REST tools or proxy an upstream MCP server. MCP server plugins require Higress 2.1.0 or later.

## MCP 2026 Tools baseline

The plugin supports the stateless HTTP Tools baseline from MCP `2026-07-28`:

- `server/discover`, `tools/list`, and `tools/call`;
- validation of `MCP-Protocol-Version`, `Mcp-Method`, `Mcp-Name`, and per-request `_meta`;
- REST servers and both `modern` and `legacy` upstream strategies for `mcp-proxy`;
- same-origin enforcement, batch rejection, and structured JSON-RPC errors;
- compatibility with the existing legacy `initialize`, `tools/list`, and `tools/call` flow.

A modern request must carry both the transport headers and request metadata:

```http
MCP-Protocol-Version: 2026-07-28
Mcp-Method: tools/call
Mcp-Name: get_weather
Content-Type: application/json
Accept: application/json, text/event-stream
```

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "_meta": {
      "io.modelcontextprotocol/protocolVersion": "2026-07-28",
      "io.modelcontextprotocol/clientInfo": {"name": "example", "version": "1.0.0"},
      "io.modelcontextprotocol/clientCapabilities": {}
    },
    "name": "get_weather",
    "arguments": {"location": "Hangzhou"}
  }
}
```

For `mcp-proxy`, `protocolStrategy` describes the upstream protocol. `modern` forwards one stateless 2026 request; `legacy` performs an isolated legacy handshake for each modern request and translates the result. The default is `legacy`. Do not place runtime session IDs in configuration or test evidence.

## Configuration

The compiled-in Quark and Amap servers normally need no additional
`defaultConfig`; their tools and authentication are defined by their server
implementations. To customize the service version returned by MCP `initialize`,
set `server.version`; it defaults to `1.0.0`. See the
[MCP server development guide](../../mcp-servers/README.md).

### REST server example

```yaml
server:
  name: weather
  type: rest
tools:
  - name: get_weather
    description: Query weather
    args:
      - name: location
        type: string
        required: true
    responseTemplate:
      body: "weather for {{.args.location}}"
```

### Custom server version example

```yaml
# Customize serverInfo.version in the initialize response.
server:
  name: quark-search
  version: 2.5.0
```

### MCP proxy example

```yaml
server:
  name: upstream-tools
  type: mcp-proxy
  transport: http
  protocolStrategy: modern # or legacy
  mcpServerURL: https://mcp.example.com/mcp
```

## WasmPlugin resource

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: mcp-server
  namespace: higress-system
spec:
  selector:
    matchLabels:
      higress: higress-system-higress-gateway
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/mcp-server:<version>
```

## Reproducible verification

Official examples are pinned to `modelcontextprotocol/modelcontextprotocol@f817239f4d6b1efff2c4dfc2f7af85c985d73076`. The official clients are locked to Go SDK `v1.7.0` and TypeScript client `2.0.0`; tests never fetch a moving `latest` version.

```bash
# Unit, official-example, and compatibility coverage
go test -count=1 ./...

# Go 1.25+ and Node.js 20+; exercises discover/list/call on all three modes
./testdata/interop/run.sh

# Explicit e2e WASM build, independent of VERSION/-alpha scanning
make -C ../../../.. build-mcp-server-wasmplugin
```

The complete kind/Envoy e2e requires Docker, kind, kubectl, and Helm:

```bash
PLUGIN_TYPE=GO PLUGIN_NAME=mcp-server TEST_SHORTNAME=WasmPluginsMCP20260728 make higress-wasmplugin-test
```

Local TestHost verification does not replace this real data-plane step; CI runs it. `plugin.wasm` is a build artifact and must not be committed.

## Related documentation

- [MCP quick start](https://higress.cn/en/ai/mcp-quick-start/)
- [MCP server development guide](../../mcp-servers/README.md)
- [Wasm plugin marketplace](https://higress.cn/en/plugin/)
