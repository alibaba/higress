# A2A runtime verification — 2026-09-06

Tested implementation:
[`ed78c0252e07e98dde15d8af54a38b947d510844`](https://github.com/higress-group/higress/commit/ed78c0252e07e98dde15d8af54a38b947d510844),
based on Higress main `637ba40c`. Subsequent changes add this evidence and the
trailers replay helper, and format two test files; production source is unchanged.
Public SDK:
[`35813546bdbe882693a4a37c8e1e736e34c78210`](https://github.com/higress-group/wasm-go/commit/35813546bdbe882693a4a37c8e1e736e34c78210),
pinned as `v1.1.3-0.20260906072435-35813546bdbe` without a local replacement.

## Environment and inputs

Linux amd64, Go 1.26.5, Python 3.11, `a2a-sdk==0.3.22`, kind v0.32.0,
Kubernetes v1.34.0. A fresh isolated kind cluster ran two Higress v2.2.4 gateway
replicas, the built controller, three in-memory Agent Pods, Redis, and two
standalone Agent containers outside Kubernetes. The static McpBridge source
registered the external containers directly. The native route used a Service
whose Pod endpoints were delivered to Envoy; requests did not route through
the Service ClusterIP.

[REPLAY.md](REPLAY.md) gives build/run commands.
[recreate.py](recreate.py) and [render.py](render.py) define the manifests,
Helm values and mounts. [images.json](evidence/images.json) records running
image identities. The Redis archive's hash is in the artifact manifest;
its manifest digest is recorded in REPLAY.md.

The controller used the exact main-branch dependency gitlinks:

| Module | Commit |
| --- | --- |
| istio/istio | `65133dd61c6c597fb301c53beaa777ca5c9e4ba2` |
| istio/api | `efc0fe428cccecc0285dbf51e2023f8e0422597c` |
| istio/client-go | `09ed8dc4e7484d303c931fa42df7f7c728296923` |
| istio/pkg | `15c662994e6515201dc106a2ec3de85ed50e8078` |
| envoy/go-control-plane | `4f11f65d9bc54bb8b18f1cc7c25460d3f8afef75` |

## Results

| Verification | Result / evidence |
| --- | --- |
| Controller config and both Ingress controllers | [PASS](evidence/controller-unit.log) |
| Protocol helpers and plugin | [PASS](evidence/protocol-unit.log), [PASS](evidence/plugin-unit.log) |
| hgctl publication helpers | [PASS](evidence/hgctl-unit.log) |
| Full wasm-go SDK tests on its current main base | [PASS](evidence/sdk-unit.log) |
| Native and remote sources, 34 task lookups per source across two gateways, ten concurrent context messages, alias conflicts, route isolation, unknown IDs, cancellation and stream resubscription | [PASS](evidence/affinity-runtime.jsonl) |
| Official Python SDK Card/send/get/cancel/SSE through both sources | [PASS](evidence/sdk-client.jsonl) |
| 34 protocol/Card/HTTP2 trailers checks with affinity enabled | [PASS](evidence/protocol-runtime.jsonl) |
| Pod removal during traffic, external endpoint stop/removal, TTL expiry, Redis failure mid-stream and Redis binding loss/recovery | [PASS, five cases](evidence/affinity-faults.jsonl) |
| Atomic Redis alias writes, conflict without partial writes, 16 concurrent writers | [PASS](evidence/affinity-redis-atomic.json) |
| Removal of the isolated cluster and both fixture containers | [PASS](evidence/affinity-cleanup.json) |

The first SSE event arrived in approximately 4.4 ms in the cross-gateway
matrix; the other gateway resolved the returned task while the stream was
still open. During Pod removal, three responses came from the original Agent
and 27 returned unavailable; none successfully fell back to another Agent.
These are fixture observations, not production latency guarantees.

The actual xDS checks verify `strict: true`, the native affinity filter after
A2A WASM and before the router, and no retry policy that can bypass affinity:
[gateway 1](evidence/higress-gateway-65559f7fd9-tfcxd-routing.json),
[gateway 2](evidence/higress-gateway-65559f7fd9-w4ss7-routing.json).

## Red/fixed SDK regression comparison

The original plugin at
[`31dd75add1ae5b879e433988fd6b06b3702e145b`](https://github.com/higress-group/higress/commit/31dd75add1ae5b879e433988fd6b06b3702e145b)
with wasm-go `v1.0.10-0.20260115123534-84ef43c39dc9` reproduces skipped Card
rewriting, skipped unary metadata processing and skipped request validation
when HTTP/2 messages terminate in trailers. Each defect is repeated three
times per source. Baseline-mode PASS means the expected defect was observed.

After the fault suite, [trailers_replay.py](trailers_replay.py) selected one
endpoint per source and disabled affinity for this comparison. It sent the
same 34 requests with only the plugin artifact changed:
[baseline reproduction](evidence/trailers-baseline.jsonl),
[fixed behavior](evidence/trailers-fixed.jsonl).
All assertions passed in both modes. The fixed run uses the same WASM artifact
as the earlier multi-replica affinity suite.

## Artifact identity

[affinity-artifacts.sha256](evidence/affinity-artifacts.sha256) identifies the
actual WASM, controller, fixture binary and Redis archive.
[baseline-artifact.sha256](evidence/baseline-artifact.sha256) identifies the
pre-fix WASM. [evidence.sha256](evidence/evidence.sha256) hashes the published
result files. Raw pod logs and TLS material are not included.

WASM SHA-256:
`b1d7ce9b20299a0e6fd19d84098fd127a7a03231674c37c852c05dbb54f2e50c`.
Controller SHA-256:
`281286fdfe438d2c14dd00b57c0237b9e8b2609b1bef6238a6e509902a1c567a`.

This is runtime verification of the described functionality, including A2A
0.3 SDK interoperability and selected 1.0 protocol/Card paths. It is not full
A2A 1.0 conformance certification. Affinity stores endpoint bindings, not
Agent task state; address reuse, durable Agent state and application
idempotency still need deployment-level decisions. No plugin OCI image is
published by this change.
