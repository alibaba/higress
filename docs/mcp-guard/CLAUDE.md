# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Higress?

Higress is a cloud-native AI Native API Gateway based on Istio and Envoy. It provides both traditional API gateway capabilities and advanced AI features including:

- **AI Gateway**: Connect to all LLM model providers with unified protocol, AI observability, multi-model load balancing, token rate limiting, and streaming (SSE) support
- **MCP Server Hosting**: Host MCP (Model Context Protocol) servers through Wasm plugins, enabling AI agents to call various tools and services
- **Kubernetes Ingress Controller**: Feature-rich ingress controller compatible with K8s nginx ingress annotations
- **Microservice Gateway**: Service discovery from Nacos, ZooKeeper, Consul, Eureka, etc.
- **Security Gateway**: WAF protection and various authentication strategies (key-auth, hmac-auth, jwt-auth, basic-auth, oidc)

## High-Level Architecture

### Control Plane (`pkg/bootstrap/`, `pkg/ingress/`)
The Higress Controller converts Kubernetes/Gateway API resources and Higress custom resources into Istio resources, then uses xDS (ADS) to push configurations to Envoy:

- **Ingress Config Controller** (`pkg/ingress/config/ingress_config.go`): Central converter that transforms Ingress/Gateway API, WasmPlugin, McpBridge, Http2Rpc, and ConfigMap resources into Istio Gateway/VirtualService/DestinationRule/ServiceEntry/EnvoyFilter/WasmPlugin resources
- **McpBridge Controller** (`registry/reconcile/reconcile.go`): Integrates external service registries (Nacos/Consul/Eureka/ZK/DNS) and watches services to generate ServiceEntry resources
- **Configmap Manager** (`pkg/ingress/kube/configmap/`): Manages global configuration from `higress-config` ConfigMap, including MCP Server definitions
- **Cert Server** (`pkg/cert/server.go`): Automatic certificate management via ACME/HTTP-01 challenge with Let's Encrypt

### Data Plane (Envoy + Pilot Agent)
- **Envoy Proxy**: Handles all traffic routing, load balancing, and HTTP filter chain execution
- **Wasm Plugin System**: Plugins written in Go/Rust/JS/AssemblyScript extend gateway functionality
  - Key plugins: `ai-proxy` (AI model routing/protocol adaptation), `waf`, `oidc`, `jwt-auth`, `mcp-guard` (capability authorization)
  - Location: `plugins/wasm-go/extensions/`, `plugins/wasm-rust/`, etc.

### Configuration Flow
```
Kubernetes/Gateway API → Higress CRDs → Istio Resources → xDS/ADS → Envoy
                                          ↓
ConfigMap (mcp-guard) → EnvoyFilter (ECDS) → Dynamic MCP/Guard Filters
                                          ↓
External Registries → McpBridge → ServiceEntry → Dynamic Service Discovery
```

## Common Development Commands

### Building
```bash
# Build Higress binary
make build

# Build for Linux (required for Docker images)
make build-linux

# Build specific architectures
make $(OUT_LINUX)/higress  # amd64
make $(ARM64_OUT_LINUX)/higress  # arm64

# Build Docker images
make docker-build          # Build and push to registry
make docker-build-amd64    # AMD64 only
make docker-buildx-push    # Multi-arch buildx

# Build with build container (recommended)
BUILD_WITH_CONTAINER=1 make build

# Build hgctl CLI tool
make build-hgctl
make build-hgctl-multiarch  # All platforms
```

### Testing
```bash
# Run unit tests with coverage
make go.test.coverage

# Run conformance tests (requires Kubernetes cluster)
make higress-conformance-test-prepare  # Setup test environment
make higress-conformance-test          # Run tests
make higress-conformance-test-clean    # Cleanup

# Wasm plugin tests
make higress-wasmplugin-test-prepare-skip-docker-build
make higress-wasmplugin-test-skip-docker-build

# Run single test
go test -v ./pkg/ingress/config/ -run TestIngressConfig

# E2E tests with specific test
make run-higress-e2e-test EXECUTE-tests=TestIngressStatic

# Run in debug mode
go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true
```

### Development Setup
```bash
# Initial setup (required before first build)
make prebuild && go mod tidy

# Create kind cluster for local testing
make create-cluster KIND_NODE_TAG=v1.25.3

# Load images into kind cluster
make kube-load-image

# Install Higress in dev mode
make install-dev

# Install with Wasm plugin support
make install-dev-wasmplugin

# Clean build artifacts
make clean-higress  # Clean Higress builds
make clean-gateway  # Clean Envoy/Istio builds
make clean          # Clean everything
```

### Wasm Plugin Development
```bash
# Build all Wasm plugins
make build-wasmplugins

# Build Golang filters
TARGET_ARCH=amd64 ./tools/hack/build-golang-filters.sh
```

### Working with MCP Guard (Recent Feature)
The `mcp-guard` plugin provides capability-based authorization for MCP servers. Key files:
- Plugin implementation: `plugins/wasm-go/extensions/mcp-guard/`
- Demo configuration: `samples/mcp-guard/higress-config.yaml`
- Usage: Configure in `higress-config` ConfigMap to specify `requestedCapabilityHeader`, `subjectPolicy`, and `rules`

Example validation:
```bash
# Allowed request
curl -i -X POST \
  -H 'X-Subject: tenantA' \
  -H 'X-MCP-Capability: cap.image.moderate' \
  http://gateway/v1/images:moderate

# Denied request (returns 403)
curl -i -X POST \
  -H 'X-Subject: tenantB' \
  -H 'X-MCP-Capability: cap.image.moderate' \
  http://gateway/v1/images:moderate
```

## Key Code Locations

### Core Controller Logic
- **Server Bootstrap**: `pkg/bootstrap/server.go:160` (initialization), `:312` (xDS generator registration)
- **Ingress Config**: `pkg/ingress/config/ingress_config.go:180` (init), `:220` (event handlers), `:640` (Istio conversion), `:700` (EnvoyFilter caching)
- **SSE Stateful Session**: `pkg/ingress/config/ingress_config.go:1940` (MCP SSE session management)
- **MCP Server ECDS**: `pkg/ingress/kube/configmap/mcp_server.go:320` (dynamic MCP filter config)

### External Registry Integration
- **McpBridge Reconciler**: `registry/reconcile/reconcile.go:60` (init), `:212` (reconciliation logic)

### Wasm Plugins
- **ai-proxy** (Protocol adaptation/streaming): `plugins/wasm-go/extensions/ai-proxy/main.go:91` (request), `:220` (response), `:360` (streaming)
- **mcp-guard** (Capability authorization): `plugins/wasm-go/extensions/mcp-guard/`
- Go plugin template: `plugins/wasm-go/extensions/`

### xDS Resources
- **Resource Generators**: `pkg/ingress/mcp/generator.go:160` (mcp-specific xDS generation)

### Testing
- **E2E Tests**: `test/e2e/e2e_test.go` (main test runner)
- **Gateway Tests**: `test/gateway/`

## Key Directories

- **`pkg/`**: Core controller code (bootstrap, cert, ingress, config, kube)
- **`plugins/`**: Wasm plugins (wasm-go, wasm-rust, wasm-cpp, wasm-assemblyscript, golang-filter)
- **`registry/`**: External registry integration (Nacos, Consul, Eureka, ZK, etc.)
- **`cmd/`**: Main entry points (higress, hgctl)
- **`test/`**: E2E and conformance tests
- **`api/`**: Custom Resource Definitions and API generation
- **`helm/`**: Kubernetes Helm charts
- **`samples/`**: Demo configurations and examples (including mcp-guard)
- **`tools/`**: Build and development tools

## Recent Feature Additions

### MCP Guard (Capability Authorization)
Added mcp-guard Wasm plugin for fine-grained capability-based access control for MCP servers. Allows defining which tenants/capabilities can access specific MCP tools/routes. See `samples/mcp-guard/` for examples.

### AI Gateway Enhancements
- Protocol adaptation between different LLM providers (Claude ↔ OpenAI formats)
- Streaming SSE event handling and transformation
- Token-based rate limiting and caching

## Important Build Notes

- **Go Version**: Uses build container by default (`BUILD_WITH_CONTAINER=1`)
- **Dependencies**: Submodules required (`make submodule` or `make prebuild`)
- **Registry**: Default image registry is `higress-registry.cn-hangzhou.cr.aliyuncs.com/higress`
- **Architecture**: Supports amd64 and arm64 (multi-arch builds available)
- **GOPROXY**: Set to `https://proxy.golang.org,direct` for faster builds

## Development Workflow

1. **Initial Setup**: `make prebuild && go mod tidy`
2. **Create Test Cluster**: `make create-cluster`
3. **Build**: `BUILD_WITH_CONTAINER=1 make docker-build`
4. **Load & Install**: `make kube-load-image && make install-dev-wasmplugin`
5. **Run Tests**: `make higress-wasmplugin-test-skip-docker-build`
6. **Clean Up**: `make higress-conformance-test-clean`

## Key Resources

- **Official Docs**: https://higress.cn/en/docs/latest/
- **Developer Guide**: https://higress.cn/en/docs/latest/dev/architecture/
- **MCP QuickStart**: https://higress.cn/en/ai/mcp-quick-start/
- **Demo Console**: http://demo.higress.io/
- **MCP Server Platform**: https://mcp.higress.ai/

## Contributing

- **Branch**: All contributions target `main` branch
- **Commit Messages**: Use conventional format (docs:, feature:, bugfix:, refactor:, test:)
- **PR Description**: Follow `.github/PULL_REQUEST_TEMPLATE.md`
- **Testing**: All PRs should include appropriate tests
- **Security Issues**: Report privately to higress@googlegroups.com
- **Code Style**: Follow Go best practices (golangci-lint checks in CI)
