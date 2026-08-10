# MCP server Envoy runtime verification

This harness compiles the exact checked-out `mcp-server` source to WASM and loads it into the fixed Higress gateway image `v2.2.3`. Every protocol request passes through an Envoy listener and the real proxy-Wasm runtime. Observable Python standard-library backends provide deterministic REST, modern MCP, and legacy MCP behavior.

Run from a clean committed checkout:

```bash
./plugins/wasm-go/extensions/mcp-server/testdata/runtime-verification/run.sh
```

The command prints a sibling `/Users/.../higress-mcp-runtime.*` evidence directory outside the worktree and returns non-zero when any matrix case fails. Podman machine can bind-mount that shared macOS path but cannot bind-mount an arbitrary host `/tmp` path. It records source/plugin/image identities, Podman and external Compose versions, a sanitized Compose config, per-case results with complete sanitized backend event snapshots, sanitized gateway logs, checksums, and cleanup proof. `plugin.wasm` is deleted after the run and is never committed.

The standalone composed endpoint intentionally verifies `tools/list` and rejects `tools/call`: successful composed calls require the separate `mcp-router` routing layer. This is an architecture boundary, not a runtime defect in `mcp-server`.

## Known exact-head runtime failure

With production code based on `3c5819ed1e6eba48b7d4ad2c24f57ff4c66ae405`, the command is expected to report `9 PASS / 2 FAIL` and exit 1. Both failing proxy matrix cells correctly send an isolated `initialize` → `notifications/initialized` → `tools/list` or `tools/call` sequence to the configured `/legacy` endpoint, but then also forward the original downstream request to `/mcp`:

- modern downstream → legacy upstream: 8 backend events instead of 6; the two extra requests also carry the otherwise era-scoped `Mcp-Param-Future` header.
- each legacy downstream → default legacy upstream flow: 24 backend events instead of 18; every list/call exchange has an extra original `/mcp` request.

The two failures expose the same resume/fallthrough behavior after a completed legacy proxy callout. The strict assertions are intentionally retained as production-bug reproductions; this harness does not modify production code or accept duplicate RPCs as expected behavior.

Development-only dirty-tree iterations can use `RUNTIME_ALLOW_DIRTY=1`; such a run records `source_tree_clean: false` and must not be used as final exact-head evidence.
