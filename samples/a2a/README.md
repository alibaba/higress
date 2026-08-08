# A2A protocol-aware route

Update the host names in `quickstart.yaml`, then apply it to attach the A2A
protocol plugin only to the example route:

```shell
kubectl apply -f samples/a2a/quickstart.yaml
```

Clients send A2A 1.0 JSON-RPC requests with `Content-Type:
application/a2a+json` and `A2A-Version: 1.0`. The plugin removes untrusted
`x-higress-a2a-*` request headers and publishes bounded canonical metadata for
later authentication, authorization, rate-limit, and observability plugins.

Agent Card responses on `/.well-known/agent-card.json` and the legacy
`/.well-known/agent.json` path are bounded and validated. JSON-RPC interface
URLs are rewritten to the configured public `agent.externalBaseURL`. Signed
Cards use `preserve` mode and are never rewritten without re-signing.

The gateway observes task state returned by the upstream Agent; it does not
store or own task state.
