# Reproduce the A2A demo

Use a Linux amd64 host with Go 1.26+, Python 3.11+, Docker (or the configured
Podman provider), kind, kubectl, Helm and OpenSSL. The host must reach the kind
node's container IP. The scripts create only the `higress-a2a-demo` cluster and
two `higress-a2a-demo-remote-*` containers. They refuse to reuse existing names.
All test artifacts stay in an operator-selected directory; do not commit TLS keys.

## Build this revision

Check out the PR's exact commit as `$A2A_ARTIFACT_ROOT/higress`. Initialize the
five Go dependency submodules at their recorded gitlinks and expose them using
the repository's normal `external` layout (`api`, `client-go`, `istio`, `pkg`
and `go-control-plane`). Do not substitute another branch's dependency checkout.
The plugin pins a publicly fetchable wasm-go revision; no SDK replacement is needed.

```sh
export A2A_ARTIFACT_ROOT="$(pwd)/a2a-artifacts"
mkdir -p "$A2A_ARTIFACT_ROOT/evidence"
# Place the exact Higress checkout at "$A2A_ARTIFACT_ROOT/higress" first.
cd "$A2A_ARTIFACT_ROOT/higress"
git submodule update --init istio/api istio/client-go istio/istio istio/pkg envoy/go-control-plane
mkdir -p external
ln -s ../istio/api external/api
ln -s ../istio/client-go external/client-go
ln -s ../istio/istio external/istio
ln -s ../istio/pkg external/pkg
ln -s ../envoy/go-control-plane external/go-control-plane
go test ./pkg/ingress/config ./pkg/ingress/kube/ingress ./pkg/ingress/kube/ingressv1
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../higress-controller-affinity-final ./cmd/higress
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../demo-agent-affinity-v2 samples/a2a/demo/main.go
(cd plugins/wasm-go/pkg/a2a && go test ./...)
(cd plugins/wasm-go/extensions/a2a-protocol && go test ./... && \
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o "$A2A_ARTIFACT_ROOT/affinity-final.wasm" .)
(cd hgctl && go test ./pkg/services ./pkg/agent)
python3.11 -m venv "$A2A_ARTIFACT_ROOT/venv"
"$A2A_ARTIFACT_ROOT/venv/bin/pip" install a2a-sdk==0.3.22 httpx[http2]
```

The replay uses gateway/pilot v2.2.4 with the controller binary above mounted
into `higress-core`. Prepare `images.tar` as a containerd/OCI image archive with
the v2.2.4 images selected by `helm template higress helm/core --set global.tag=v2.2.4`.
Prepare `redis.tar` with the Redis-compatible fixture image
`docker.io/tairmodule/tairhash:latest`, and cache `docker.io/kindest/node:v1.34.0`
in the host runtime (the external Agent containers replace its entrypoint;
Agent Pods replace the gateway image entrypoint). The recorded Redis manifest is
`sha256:2f5f98a479ee03e04f964a518fbbff586234c29f83f8e96cd7b006815863469c`;
preserve that image under the fixture tag for an exact replay. `ctr -n k8s.io
images export` can export images from a dedicated image cache. The archive is
imported only into the new test node, and the fixture uses `imagePullPolicy: Never`.

## Run

```sh
cd "$A2A_ARTIFACT_ROOT"
# Set KIND_EXPERIMENTAL_PROVIDER=podman only when using Podman.
python3 higress/samples/a2a/runtime/recreate.py "$PWD"
export KUBECONFIG="$PWD/kubeconfig"
node_ip=$(docker inspect higress-a2a-demo-control-plane --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
python311="$PWD/venv/bin/python"
"$python311" higress/samples/a2a/runtime/affinity_verify.py "$node_ip" fixed
"$python311" higress/samples/a2a/runtime/sdk_client.py "$node_ip"
go run higress/samples/a2a/runtime/verify/main.go -node "$node_ip" -variant fixed
python3 higress/samples/a2a/runtime/redis_atomic.py "$PWD"
# Faults stop an external Agent and erase Redis bindings. Run them last.
"$python311" higress/samples/a2a/runtime/affinity_faults.py "$node_ip"
python3 higress/samples/a2a/runtime/collect.py "$PWD"
python3 higress/samples/a2a/runtime/cleanup.py "$PWD"
```

Wait for configuration convergence before testing. The two individually
addressable gateway replicas use NodePorts 30081/30082; the shared route uses
30443. TLS is self-signed only for this test. On cgroup-v1 hosts, `recreate.py`
creates the missing kubelet cgroup directory inside the new node during startup.

The protocol verifier makes 34 assertions, including HTTP/2 trailers. The
affinity matrix tests server-generated task/context aliases, two gateways,
concurrent context messages, cancellation, SSE and resubscription for both
native and external sources. The fault suite checks Pod removal under traffic,
external endpoint removal, TTL expiry, streaming Redis failure and Redis state
loss/recovery. `collect.py` checks actual xDS filter order, strict mode, disabled
retries and binary hashes. It excludes TLS secrets and full Envoy dumps.

For a red trailers baseline, build the original plugin from the baseline
revision linked in the PR using its original wasm-go dependency, render with
`--variant baseline`, and run the Go verifier with `-variant baseline`.
That mode asserts that the documented defects reproduce: PASS means a
successful bug reproduction. An affinity-disabled multi-replica route can
be checked with `affinity_verify.py NODE_IP baseline` to count lost task lookups.
