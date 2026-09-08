# A2A protocol-aware route

Update the host names in `quickstart.yaml`, then apply it to attach the A2A
protocol plugin only to the example route:

```shell
kubectl apply -f samples/a2a/quickstart.yaml
```

Clients send A2A 1.0 JSON-RPC requests with `Content-Type: application/json`
and `A2A-Version: 1.0`. The plugin removes untrusted
`x-higress-a2a-*` request headers and publishes bounded canonical metadata for
later authentication, authorization, rate-limit, and observability plugins.

Agent Card responses on `/.well-known/agent-card.json` and the legacy
`/.well-known/agent.json` path are bounded and validated. Only the JSON-RPC 1.0
interface supported by this route is advertised, and its URL is rewritten to
the explicitly configured public `agent.externalBaseURL`. Signed Cards use
`preserve` mode only when their endpoint already matches that trusted URL.

The gateway observes task state returned by the upstream Agent; it does not
store or own task state.

## Discovery through existing service sources

The controller on this branch accepts `higress.io/a2a-config` on an Ingress.
Its value is the JSON configuration of `a2a-protocol`. The controller generates
an Istio WasmPlugin scoped to that Ingress and updates/removes it when the
annotation changes. No additional CRD or registry process is required.

```yaml
metadata:
  annotations:
    higress.io/a2a-plugin-url: file:///a2a/fixed.wasm
    higress.io/a2a-config: >-
      {"protocolVersion":"1.0","legacy03":{"enabled":true},"agent":{"id":"weather","externalBaseURL":"https://weather.example.com/"}}
```

`a2a-plugin-url` selects an operator-built artifact. The default is the configured
`a2a-protocol:1.0.0-alpha` OCI path; this work does not publish that image. Do not
also attach a manual A2A WasmPlugin to the same Ingress.

- **Kubernetes Agent:** use an ordinary Ingress `backend.service` referencing
  the Agent's Service. Kubernetes watches and the gateway's normal endpoint
  discovery follow Pod changes. The Agent must already implement A2A.
- **External Agent:** configure a DNS or static registry in the existing
  `McpBridge/default`, and reference it with `higress.io/destination`. Use an
  Ingress `backend.resource` (McpBridge), rather than `backend.service`, for
  this source. Configure `backend-protocol: HTTPS` for HTTPS origins or
  `HTTP2` for cleartext HTTP/2 origins.
- **Client discovery:** the client resolves the Agent Card through the public
  gateway hostname. The plugin rewrites the Card to that configured hostname;
  clients do not need Kubernetes credentials or internal endpoint addresses.

This is explicit route publication, not a search/catalog API that automatically
publishes every Service. Agent identity, protocol configuration and public
hostname are declared by the operator. Task state stays with the Agent: replicas
can enable the endpoint affinity described below.

## Endpoint affinity for stateful Agents

Add `affinity` to the Ingress's `higress.io/a2a-config`:

```json
{
  "agent": {"id": "weather", "externalBaseURL": "https://weather.example.com/"},
  "legacy03": {"enabled": true},
  "affinity": {
    "enabled": true,
    "ttlSeconds": 3600,
    "redis": {"serviceFQDN": "a2a-redis.dns", "servicePort": 6379, "timeout": 1000}
  }
}
```

Register the Redis service as an existing McpBridge DNS source, for example
`name: a2a-redis`, `type: dns`, `domain: a2a-redis.a2a-demo.svc.cluster.local`,
`port: 6379`. The Redis cluster must be discoverable by the gateway; merely
creating an otherwise unreferenced Kubernetes Service does not publish it to
Higress. Redis username and password are optional. Timeout is in milliseconds;
TTL is in seconds, defaults to 3600, and must be 1–86400.

For Agents, the controller discovers the Service's actual endpoints and
Higress connects to Pod IPs. The plugin picks a healthy endpoint for a new
message. Before exposing a returned taskId/contextId, it atomically binds
both aliases to that endpoint in Redis. Keys are separated by Agent, route,
upstream cluster and ID type. Subsequent task/context operations, including
queries, cancellation and stream subscriptions, resolve that binding across
gateway replicas. Concurrent messages in one context remain on its original
Agent process. Conflicting aliases are rejected.

The annotation adapter also generates a route-scoped **strict** Envoy
`stateful_session` filter after the WASM filter, and disables route retries
before Istio translation. This native filter is mandatory: the existing WASM
host-override API permits fallback and is insufficient for this guarantee.
Do not enable affinity with a standalone manual WasmPlugin without installing
the corresponding native filter. Client-provided endpoint and retry headers
cannot select a target or enable retries.

A provided ID without a binding, an expired binding, an unavailable endpoint,
or a Redis failure returns HTTP 503 instead of selecting another Agent. A
connection already being established during endpoint removal may first wait
for the configured upstream connection timeout. A running SSE stream reports
an error event and terminates if binding persistence fails. Streaming retains
only bounded pending event data while the Redis operation completes; it does
not wait for the entire conversation. HTTP/2 trailers wait for preceding data.

Affinity is disabled by default and requires `mode: enforce`. IDs must be
shorter than 256 UTF-8 bytes because diagnostic metadata is bounded; truncated
IDs are never used as routing keys. Operations that supply no task/context
binding, other than starting a message, are rejected in affinity mode; this
implementation does not aggregate global task listings across Agent replicas.
Task IDs created outside this gateway have no binding and are also rejected.

Bindings contain endpoint addresses, not copies of Agent state. Agent restart,
address reuse, Redis persistence and application idempotency remain deployment
concerns: routing affinity does not restore an in-memory task or guarantee
exactly-once execution. A new request may have reached an Agent before a later
Redis write fails, even though the client receives an error. Use durable Agent
state and application idempotency where that recovery is required. Binding
lookup is not authorization; existing authentication and task ownership checks
are still required.

## Verified demo

`demo/main.go` is a deterministic A2A fixture with an in-memory task store.
It implements Card discovery, send, context continuation, get, cancel,
resubscription and delayed SSE events without
requiring an LLM API key. Its HTTP/2 server needs Go 1.24 or newer.

`runtime/affinity_verify.py` verifies three native Agent Pods, two external
Agent processes and two individually addressed gateway replicas.
`runtime/affinity_faults.py` removes endpoints and injects TTL/Redis failures.
`runtime/render.py` renders both a native Service route and an external static
source. `runtime/verify/main.go` tests both routes with HTTP/2, including request
and response trailers. `runtime/sdk_client.py` exercises the official
`a2a-sdk==0.3.22` client. This verifies 0.3 SDK interoperability and the plugin's
1.0 Card handling; it is not full A2A 1.0 conformance certification.

The plugin pins the public wasm-go SDK revision containing the trailers lifecycle
fix and streaming-trailer callback. No sibling checkout is needed to build it.

See `runtime/REPLAY.md` for reproducible build commands and
verification artifacts. Demo TLS uses a self-signed certificate and test
clients deliberately disable certificate verification; production clients
must use trusted certificates.

## AgentTeams findings

AgentTeams revision `eeaab64391ccaec9118e84977f538aefd40720d6` uses Matrix/Tuwunel
for team communication and Kubernetes Worker/Manager/Team resources for
lifecycle management. Its Nacos integration distributes Skills and Worker
packages; it is not an A2A endpoint registry. Consequently no AgentTeams/Nacos
A2A registry adapter is introduced. A Worker exposing an actual A2A HTTP
endpoint can use either of the service-source paths above.

Sources: [architecture](https://github.com/agentscope-ai/AgentTeams/blob/eeaab64391ccaec9118e84977f538aefd40720d6/docs/zh-cn/design/architecture.md),
[resource management](https://github.com/agentscope-ai/AgentTeams/blob/eeaab64391ccaec9118e84977f538aefd40720d6/docs/zh-cn/usage/resource-management.md).
