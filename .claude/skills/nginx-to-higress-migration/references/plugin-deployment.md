# WASM Plugin Build and Deployment

## Plugin Project Structure

```
my-plugin/
├── main.go          # Plugin entry point
├── go.mod           # Go module
├── go.sum           # Dependencies
├── Dockerfile       # OCI image build
└── wasmplugin.yaml  # K8s deployment manifest
```

## Build Process

### 1. Initialize Project

```bash
mkdir my-plugin && cd my-plugin
go mod init my-plugin

# Set proxy (China)
go env -w GOPROXY=https://proxy.golang.com.cn,direct

# Get dependencies
go get github.com/higress-group/proxy-wasm-go-sdk@go-1.24
go get github.com/higress-group/wasm-go@main
go get github.com/tidwall/gjson
```

### 2. Write Plugin Code

See the higress-wasm-go-plugin skill for detailed API reference. Basic template:

```go
package main

import (
    "github.com/higress-group/wasm-go/pkg/wrapper"
    "github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
    "github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
    "github.com/tidwall/gjson"
)

func main() {}

func init() {
    wrapper.SetCtx(
        "my-plugin",
        wrapper.ParseConfig(parseConfig),
        wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
    )
}

type MyConfig struct {
    // Config fields
}

func parseConfig(json gjson.Result, config *MyConfig) error {
    // Parse YAML config (converted to JSON)
    return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config MyConfig) types.Action {
    // Process request
    return types.HeaderContinue
}
```

### 3. Compile to WASM

```bash
go mod tidy
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./
```

### 4. Create Dockerfile

```dockerfile
FROM scratch
COPY main.wasm /plugin.wasm
```

### 5. Build and Push Image

```bash
# User must provide registry
REGISTRY=your-registry.com/higress-plugins

# Build
docker build -t ${REGISTRY}/my-plugin:v1 .

# Push
docker push ${REGISTRY}/my-plugin:v1
```

## Deployment

### WasmPlugin CRD

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: my-plugin
  namespace: higress-system
spec:
  # OCI image URL
  url: oci://your-registry.com/higress-plugins/my-plugin:v1
  
  # Plugin phase (when to execute)
  # UNSPECIFIED_PHASE | AUTHN | AUTHZ | STATS
  phase: UNSPECIFIED_PHASE
  
  # Priority (higher = earlier execution)
  priority: 100
  
  # Plugin configuration
  defaultConfig:
    key: value
  
  # Optional: specific routes/domains
  matchRules:
  - domain:
    - "*.example.com"
    config:
      key: domain-specific-value
  - ingress:
    - default/my-ingress
    config:
      key: ingress-specific-value
```

### Apply to Cluster

```bash
kubectl apply -f wasmplugin.yaml
```

### Verify Deployment

```bash
# Check plugin status
kubectl get wasmplugin -n higress-system

# Check gateway logs
kubectl logs -n higress-system -l app=higress-gateway | grep -i plugin

# Test endpoint
curl -v http://<gateway-ip>/test-path
```

## Troubleshooting

### Plugin Not Loading

```bash
# Check image accessibility
kubectl run test --rm -it --image=your-registry.com/higress-plugins/my-plugin:v1 -- ls

# Check gateway events
kubectl describe pod -n higress-system -l app=higress-gateway
```

### Plugin Errors

```bash
# Enable debug logging
kubectl set env deployment/higress-gateway -n higress-system LOG_LEVEL=debug

# View plugin logs
kubectl logs -n higress-system -l app=higress-gateway -f
```

### Image Pull Issues

```bash
# Create image pull secret if needed
kubectl create secret docker-registry regcred \
  --docker-server=your-registry.com \
  --docker-username=user \
  --docker-password=pass \
  -n higress-system

# Reference in WasmPlugin
spec:
  imagePullSecrets:
  - name: regcred
```

## Plugin Configuration via Console

If using Higress Console:

1. Navigate to **Plugins** → **Custom Plugins**
2. Click **Add Plugin**
3. Enter OCI URL: `oci://your-registry.com/higress-plugins/my-plugin:v1`
4. Configure plugin settings
5. Apply to routes/domains as needed
