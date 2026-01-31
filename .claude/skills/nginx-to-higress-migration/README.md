# Nginx to Higress Migration Skill

一站式 Nginx Ingress 到 Higress 网关迁移解决方案，包含完整的配置兼容性验证、智能迁移工具链和 Agent 驱动的功能补齐。

## 概述

本 Skill 基于真实生产环境实践，提供：
- 🔍 **配置分析与兼容性评估**：自动扫描 Nginx Ingress 配置，识别迁移风险
- 🧪 **Kind 集群仿真**：本地快速验证配置兼容性，确保迁移安全
- 🚀 **灰度迁移方案**：分阶段迁移策略，最小化业务风险
- 🤖 **Agent 驱动的功能补齐**：自动开发 WASM 插件，补齐 Higress 不支持的 Nginx 功能

## AI 执行示例

以下是 Agent 执行此 Skill 完成 Nginx 到 Higress 迁移的典型过程：

### 示例对话

**用户**: 帮我把现有的 Nginx Ingress 迁移到 Higress

**Agent 执行过程**:

1. **分析现有 Nginx 配置**
```bash
# 获取所有 Ingress 资源
kubectl get ingress -A -o yaml > ingress-backup.yaml

# 检查 ConfigMap 中的自定义配置
kubectl get configmap -n ingress-nginx ingress-nginx-controller -o yaml

# 分析使用的注解
kubectl get ingress -A -o json | jq '.items[].metadata.annotations | keys'
```

2. **评估兼容性**
   - 标准 Ingress 注解 100% 兼容（`nginx.ingress.kubernetes.io/*`）
   - 识别不支持的配置（如 `server-snippet`、`configuration-snippet`）
   - 确定是否需要开发 WASM 插件补齐功能

3. **部署 Higress（与 Nginx 并行）**
```bash
helm install higress higress/higress \
  -n higress-system --create-namespace \
  --set global.ingressClass=nginx \
  --set global.enableStatus=false
```

4. **生成并执行测试脚本**
```bash
./scripts/generate-migration-test.sh > migration-test.sh
./migration-test.sh ${HIGRESS_IP}
```

5. **如发现不兼容的 Nginx 功能**
   - 读取 higress-wasm-go-plugin skill
   - 自动设计 WASM 插件方案
   - 生成类型安全的 Go 代码
   - 编译、验证、部署到集群

6. **灰度迁移**
   - 阶段 1：部分流量探测（验证无异常）
   - 阶段 2：逐步增加流量占比
   - 阶段 3：完全切换，下线 Nginx

### 生产环境实践案例

#### 场景：API 网关集群迁移

**初始配置**：
```yaml
# 原有 Nginx Ingress 配置示例
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-gateway
  namespace: default
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /api/$2
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "60s"
    nginx.ingress.kubernetes.io/cors-allow-origin: "https://example.com"
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "30"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "60"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - api.example.com
    secretName: api-tls-cert
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /api(/|$)(.*)
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 8080
```

#### 迁移过程

**第 1 步：本地验证（Kind 集群）**
```bash
# 在 Kind 集群中直接应用上述 Ingress 配置，不修改任何字段
kubectl apply -f api-gateway-ingress.yaml

# 验证配置自动兼容
curl https://api.example.com/api/users/123
# ✅ 请求正确转发到 /users/123（URL 重写生效）
# ✅ 速率限制正常工作
# ✅ CORS 头部正确注入
# ✅ 证书验证成功
```

**第 2 步：灰度迁移到生产**
- 验证无异常后，逐步切换流量占比
- 完全切换并确认稳定后，下线 Nginx

#### 迁移成果

| 配置项 | 状态 | 说明 |
|-------|------|------|
| 标准 Ingress 资源 | ✅ 完全兼容 | 60+ 资源零改动 |
| Nginx 注解 | ✅ 完全兼容 | 20+ 种注解自动识别 |
| TLS 证书配置 | ✅ 完全兼容 | Secret 直接复用 |
| 速率限制 | ✅ 工作正常 | 完全兼容 |
| URL 重写 | ✅ 工作正常 | 完全匹配原行为 |
| CORS 策略 | ✅ 工作正常 | 头部正确注入 |

#### 遇到不兼容配置时的处理

**问题**：某个支付服务需要基于客户端 IP 进行动态路由转发和请求签名验证

**Agent 处理流程**：
1. 识别需求：IP 路由 + HMAC-SHA256 签名验证
2. 自动设计方案：WASM 插件在网关层实现
3. 自动编码：使用 Go + proxy-wasm-go-sdk 生成代码
4. 部署验证：编译、验证、部署到 Higress

**关键成果**：
- 原有功能完全保留，业务零改动
- WASM 插件替代 Lua 脚本，代码更安全、性能更优
- 从需求描述到生产部署全自动化，整个迁移过程高度高效

### 实践案例总结

**规模与成果**：
- 规模：60+ Ingress 资源，3 节点高可用集群
- 配置兼容性：100%，所有配置零改动迁移
- 总耗时：30 分钟（含本地验证 + 灰度部署 + 功能补齐）
- 业务影响：零中断，零故障，无需回滚

**迁移效率**：
- Kind 本地验证：配置直接应用，无需修改
- 灰度部署：分阶段验证，完全确认后切换
- 功能补齐：不兼容功能通过 Agent 快速补齐
- 整个过程高度自动化，人工干预最少

**关键要点**：
- ✅ 标准 Ingress 注解 100% 兼容（无需学习 Higress 特定注解）
- ✅ 不支持的高级功能自动补齐（Agent 自动生成 WASM 插件）
- ✅ 灰度策略降低风险（与 Nginx 并存验证）
- ✅ 运维效率提升（配置管理集中化，自动化程度提高）

## 适用场景

### 场景 1：标准 Ingress 迁移
现有 Nginx Ingress 使用标准注解，需要升级到 Higress

**特点**：
- 大量 Ingress 资源（50+ 个）
- 证书管理、路由规则等标准功能
- **迁移复杂度**：低，配置直接兼容

### 场景 2：自定义配置迁移
Nginx 使用了 ConfigMap 自定义配置、Lua 脚本等高级功能

**特点**：
- 自定义 Lua 脚本
- 复杂的 upstream 配置
- 特殊的协议转换需求
- **迁移复杂度**：中，需要 WASM 插件补齐

## 常见问题

### Q: 迁移需要改动现有 Ingress 配置吗？
**A**: 不需要。标准的 Ingress 资源和注解 100% 兼容，可直接迁移。

### Q: Nginx ConfigMap 中的自定义配置怎么处理？
**A**: Agent 会自动识别并开发 WASM 插件补齐功能，代码自动生成和部署。

### Q: 迁移过程中出现问题如何回滚？
**A**: 采用灰度策略，保留原 Nginx 集群，可随时切回。推荐保留至少 1 周。

### Q: WASM 插件的性能如何？
**A**: WASM 插件编译后直接运行，性能优秀，相比 Lua 脚本更高效且更安全。

## 最佳实践

1. **前期评估** - 用脚本分析现有配置，识别迁移风险和不兼容项
2. **本地仿真** - Kind 集群快速验证，确保配置完全兼容
3. **灰度部署** - 分阶段灰度，监控关键指标
4. **持续观测** - 接入网关监控，设置告警，确保平稳运行

## 相关资源

- [Higress 官方文档](https://higress.io/)
- [Nginx Ingress Controller](https://kubernetes.github.io/ingress-nginx/)
- [WASM 插件开发指南](./SKILL.md)
