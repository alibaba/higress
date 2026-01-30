---
name: nginx-to-higress-migration
description: "Migrate from ingress-nginx to Higress in Kubernetes environments. Use when (1) analyzing existing ingress-nginx setup (2) reading nginx Ingress resources and ConfigMaps (3) installing Higress via helm with proper ingressClass (4) identifying unsupported nginx annotations (5) generating WASM plugins for nginx snippets/advanced features (6) building and deploying custom plugins to image registry. Supports full migration workflow with compatibility analysis and plugin generation."
---

# Nginx to Higress Migration

Automate migration from ingress-nginx to Higress in Kubernetes environments.

## Prerequisites

- kubectl configured with cluster access
- helm 3.x installed
- Go 1.24+ (for WASM plugin compilation)
- Docker (for plugin image push)

## Migration Workflow

### Phase 1: Discovery

```bash
# Check for ingress-nginx installation
kubectl get pods -A | grep ingress-nginx
kubectl get ingressclass

# List all Ingress resources using nginx class
kubectl get ingress -A -o json | jq '.items[] | select(.spec.ingressClassName=="nginx" or .metadata.annotations["kubernetes.io/ingress.class"]=="nginx")'

# Get nginx ConfigMap
kubectl get configmap -n ingress-nginx ingress-nginx-controller -o yaml
```

### Phase 2: Compatibility Analysis

**Supported nginx annotations** (native Higress support):
- `nginx.ingress.kubernetes.io/rewrite-target` → `higress.io/rewrite-target`
- `nginx.ingress.kubernetes.io/ssl-redirect` → `higress.io/ssl-redirect`
- `nginx.ingress.kubernetes.io/cors-*` → `higress.io/cors-*`
- `nginx.ingress.kubernetes.io/proxy-*-timeout` → `higress.io/timeout`
- `nginx.ingress.kubernetes.io/canary*` → `higress.io/canary*`
- `nginx.ingress.kubernetes.io/whitelist-source-range` → `higress.io/whitelist-source-range`
- `nginx.ingress.kubernetes.io/auth-*` → `higress.io/auth-*`
- `nginx.ingress.kubernetes.io/upstream-hash-by` → `higress.io/upstream-hash-by`
- `nginx.ingress.kubernetes.io/load-balance` → `higress.io/load-balance`

See [references/annotation-mapping.md](references/annotation-mapping.md) for complete mapping.

**Unsupported annotations** (require WASM plugin):
- `nginx.ingress.kubernetes.io/server-snippet`
- `nginx.ingress.kubernetes.io/configuration-snippet`
- `nginx.ingress.kubernetes.io/lua-resty-waf*`
- Complex Lua logic in snippets

### Phase 3: Higress Installation (Parallel with nginx)

Higress natively supports `nginx.ingress.kubernetes.io/*` annotations. Install Higress **alongside** nginx for safe parallel testing.

```bash
# 1. Get current nginx ingressClass name
INGRESS_CLASS=$(kubectl get ingressclass -o jsonpath='{.items[?(@.spec.controller=="k8s.io/ingress-nginx")].metadata.name}')
echo "Current nginx ingressClass: $INGRESS_CLASS"

# 2. Add Higress repo
helm repo add higress https://higress.io/helm-charts
helm repo update

# 3. Install Higress using the SAME ingressClass (both controllers will watch)
helm install higress higress/higress \
  -n higress-system --create-namespace \
  --set global.ingressClass=${INGRESS_CLASS:-nginx} \
  --set higress-core.gateway.replicas=2
```

Key points:
- `global.ingressClass`: Use the **same** class as ingress-nginx
- Both nginx and Higress will watch the same Ingress resources
- Higress automatically recognizes `nginx.ingress.kubernetes.io/*` annotations
- Traffic still flows through nginx until you switch the entry point

### Phase 4: Generate and Run Test Script

After Higress is running, generate a test script covering all Ingress routes:

```bash
# Generate test script
./scripts/generate-migration-test.sh > migration-test.sh
chmod +x migration-test.sh

# Run tests against Higress gateway
HIGRESS_IP=$(kubectl get svc -n higress-system higress-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
./migration-test.sh ${HIGRESS_IP}
```

The test script will:
- Extract all hosts and paths from Ingress resources
- Test each route against Higress gateway
- Verify response codes and basic functionality
- Report any failures for investigation

### Phase 5: Traffic Cutover (User Action Required)

⚠️ **Only proceed after all tests pass!**

Choose your cutover method based on infrastructure:

**Option A: DNS Switch**
```bash
# Update DNS records to point to Higress gateway IP
# Example: example.com A record -> ${HIGRESS_IP}
```

**Option B: Layer 4 Proxy/Load Balancer Switch**
```bash
# Update upstream in your L4 proxy (e.g., F5, HAProxy, cloud LB)
# From: nginx-ingress-controller service IP
# To: higress-gateway service IP
```

**Option C: Kubernetes Service Switch** (if using external traffic via Service)
```bash
# Update your external-facing Service selector or endpoints
```

### Phase 5: WASM Plugin for Unsupported Features

When nginx snippets or Lua logic detected, generate WASM plugin.

#### Plugin Generation Flow

1. **Analyze snippet** - Extract nginx directives/Lua code
2. **Generate Go WASM code** - Use higress-wasm-go-plugin skill
3. **Build plugin**:
```bash
cd plugin-dir
go mod tidy
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./
```

4. **Push to registry** (user provides registry):
```bash
# Build OCI image
docker build -t <registry>/higress-plugin-<name>:v1 .
docker push <registry>/higress-plugin-<name>:v1
```

5. **Deploy plugin**:
```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: custom-plugin
  namespace: higress-system
spec:
  url: oci://<registry>/higress-plugin-<name>:v1
  phase: UNSPECIFIED_PHASE
  priority: 100
```

See [references/plugin-deployment.md](references/plugin-deployment.md) for detailed plugin deployment.

## Common Snippet Conversions

### Header Manipulation
```nginx
# nginx snippet
more_set_headers "X-Custom: value";
```
→ Use `headerControl` annotation or generate plugin with `proxywasm.AddHttpResponseHeader()`.

### Request Validation
```nginx
# nginx snippet
if ($request_uri ~* "pattern") { return 403; }
```
→ Generate WASM plugin with request header/path check.

### Rate Limiting with Custom Logic
```nginx
# nginx snippet with Lua
access_by_lua_block { ... }
```
→ Generate WASM plugin implementing the logic.

See [references/snippet-patterns.md](references/snippet-patterns.md) for common patterns.

## Validation

Before traffic switch, use the generated test script:

```bash
# Generate test script
./scripts/generate-migration-test.sh > migration-test.sh
chmod +x migration-test.sh

# Get Higress gateway IP
HIGRESS_IP=$(kubectl get svc -n higress-system higress-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Run all tests
./migration-test.sh ${HIGRESS_IP}
```

The test script will:
- Test every host/path combination from all Ingress resources
- Report pass/fail for each route
- Provide a summary and next steps

**Only proceed with traffic cutover after all tests pass!**

## Rollback

Since nginx keeps running during migration, rollback is simply switching traffic back:

```bash
# If traffic was switched via DNS:
# - Revert DNS records to nginx gateway IP

# If traffic was switched via L4 proxy:
# - Revert upstream to nginx service IP

# Nginx is still running, no action needed on k8s side
```

## Post-Migration Cleanup

**Only after traffic has been fully migrated and stable:**

```bash
# 1. Monitor Higress for a period (recommended: 24-48h)

# 2. Backup nginx resources
kubectl get all -n ingress-nginx -o yaml > ingress-nginx-backup.yaml

# 3. Scale down nginx (keep for emergency rollback)
kubectl scale deployment -n ingress-nginx ingress-nginx-controller --replicas=0

# 4. (Optional) After extended stable period, remove nginx
kubectl delete namespace ingress-nginx
```
