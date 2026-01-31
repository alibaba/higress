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
   - 阶段 1：部分流量探测（1 周）
   - 阶段 2：逐步增加流量占比
   - 阶段 3：完全切换，下线 Nginx

### 实践案例总结

在真实生产环境验证中：
- **规模**：60+ Ingress 资源，3 节点高可用集群
- **配置兼容性**：100%，所有配置零改动迁移成功
- **迁移耗时**：2 小时，业务无感知
- **功能补齐**：不支持的 Nginx 功能通过 WASM 插件自动补齐

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
