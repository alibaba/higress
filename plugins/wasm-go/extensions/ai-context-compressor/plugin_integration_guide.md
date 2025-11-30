# AI Context Compressor 插件集成指南

## 1. 插件调用机制

在Higress中，WASM插件的调用是通过配置来实现的。要让ai-proxy插件调用ai-context-compressor插件的压缩能力，需要进行以下配置：

### 1.1 插件链配置
Higress的WASM插件系统支持链式调用，可以通过以下方式实现插件间的协作：

### 1.2 请求处理流程
```
客户端请求 → ai-context-compressor → ai-proxy → 后端LLM服务
```

## 2. 配置示例

要让ai-proxy在处理请求时调用ai-context-compressor的压缩能力，需要在Kubernetes中创建相应的配置：

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-context-compressor
  namespace: higress-system
spec:
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-context-compressor:1.0.0
  phase: AUTHN  # 在认证阶段执行
  priority: -1  # 优先级，数值越小越先执行
  config:
    method: "token_based"
    rate: 0.5
    model: "gpt-4"
    minTokens: 100
---
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-proxy
  namespace: higress-system
spec:
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-proxy:1.0.0
  phase: AUTHN
  priority: 0  # 在ai-context-compressor之后执行
  # ai-proxy的其他配置...
```

## 3. 调用时序

### 3.1 请求处理顺序
1. **ai-context-compressor插件**首先处理请求体，对上下文进行压缩
2. **ai-proxy插件**接着处理已经被压缩的请求体，转发给后端LLM服务
3. 响应按相反顺序处理：ai-proxy → ai-context-compressor → 客户端

### 3.2 数据传递
两个插件通过HTTP请求体进行数据传递：
- ai-context-compressor修改请求体后，ai-proxy会处理修改后的请求体
- 插件间通过`proxywasm.ReplaceHttpRequestBody` API传递数据

## 4. 实际部署配置

### 4.1 完整配置示例
```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-context-compressor
  namespace: higress-system
spec:
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-context-compressor:1.0.0
  phase: AUTHN
  priority: -1
  matchRules:
  - ingress:
    - ai-gateway  # 只在特定ingress中生效
  config:
    method: "token_based"
    rate: 0.5
    model: "gpt-4"
    minTokens: 100
---
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-proxy
  namespace: higress-system
spec:
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-proxy:1.0.0
  phase: AUTHN
  priority: 0
  matchRules:
  - ingress:
    - ai-gateway
  config:
    # ai-proxy配置
    provider:
      type: "openai"
      # 其他配置...
```

### 4.2 基于路由的配置
也可以通过路由规则来指定插件应用范围：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ai-gateway
  namespace: higress-system
  annotations:
    higress.io/wasm-plugins: "ai-context-compressor,ai-proxy"  # 指定插件链
spec:
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /v1/chat/completions
        pathType: Prefix
        backend:
          service:
            name: llm-service
            port:
              number: 80
```

## 5. 验证调用

### 5.1 日志验证
可以通过查看插件日志来验证调用顺序：

```bash
# 查看ai-context-compressor插件日志
kubectl logs -n higress-system <higress-gateway-pod> -c higress-gateway | grep "ai-context-compressor"

# 查看ai-proxy插件日志
kubectl logs -n higress-system <higress-gateway-pod> -c higress-gateway | grep "ai-proxy"
```

### 5.2 监控指标
在插件中添加监控指标来跟踪处理过程：

```go
// 在ai-context-compressor中添加指标
proxywasm.LogInfof("Context compressed, original tokens: %d, compressed tokens: %d", originalTokens, compressedTokens)
```

## 6. 注意事项

### 6.1 性能考虑
- 插件链中的每个插件都会增加处理延迟
- 需要合理设置priority确保执行顺序
- 对于高并发场景，需要考虑插件的性能影响

### 6.2 错误处理
- 如果ai-context-compressor处理失败，不会影响ai-proxy的正常执行
- 插件间通过HTTP状态码和日志进行错误传递

### 6.3 配置管理
- 不同的路由可以配置不同的插件链
- 可以通过matchRules精确控制插件的应用范围

通过以上配置，ai-proxy在处理请求时会自动调用ai-context-compressor的压缩能力，实现对上下文的智能压缩，从而降低Token使用量并提高响应性能。