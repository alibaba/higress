# Nginx to Higress Migration Skill

一站式 Nginx Ingress 到 Higress 网关迁移解决方案，包含完整的配置兼容性验证、智能迁移工具链和 Agent 驱动的功能补齐。

## 概述

本 Skill 基于真实生产环境实践，提供：
- 🔍 **配置分析与兼容性评估**：自动扫描 Nginx Ingress 配置，识别迁移风险
- 🧪 **Kind 集群仿真**：本地快速验证配置兼容性，确保迁移安全
- 🚀 **灰度迁移方案**：分阶段迁移策略，最小化业务风险
- 🤖 **Agent 驱动的功能补齐**：自动开发 WASM 插件，补齐 Higress 不支持的 Nginx 功能

## 核心价值

✅ **配置完全兼容** - 60+ Ingress 资源直接迁移，零修改
✅ **迁移风险最小** - 灰度策略 + 本地仿真验证
✅ **功能无缺口** - Agent 自动开发 WASM 插件补齐不支持的功能
✅ **运维效率提升** - 配置管理集中化、自动化程度更高

## 快速开始

### 1. 评估现有 Nginx 配置
```bash
# 分析 Ingress 资源
kubectl get ingress -A --export=true

# 检查 ConfigMap 中的自定义配置
kubectl get configmap -n ingress-nginx nginx-configuration -o yaml
```

### 2. 使用 Kind 本地仿真验证
```bash
# 创建本地 Kind 集群
kind create cluster --config kind-config.yaml

# 部署 Higress
helm install higress higress/higress \
  --namespace higress-system \
  --create-namespace

# 迁移现有 Ingress 资源（直接应用，无需修改）
kubectl apply -f migrated-ingress.yaml
```

### 3. 灰度迁移到生产
- **阶段 1**：Higress 与 Nginx 并存，部分流量探测（1 周）
- **阶段 2**：逐步增加 Higress 流量占比（10% → 50% → 100%，3 天）
- **阶段 3**：确保无异常后，完全关闭 Nginx Ingress

### 4. Agent 开发 WASM 插件补齐功能
遇到不支持的 Nginx 功能时，调用 Agent：
```
"我需要基于客户端 IP 进行动态路由转发，
内网 IP 转发到内部服务，外网 IP 转发到外部服务"
```
Agent 会自动：
- 设计 Higress WASM 插件方案
- 生成类型安全的 Go 代码
- 编译、验证、部署到集群

## 文件结构

```
nginx-to-higress-migration/
├── README.md                    # 本文件，快速入门指南
├── PRACTICE.md                  # 完整迁移实践记录
├── SKILL.md                     # 详细的技术方案和工具集
├── scripts/                     # 迁移工具脚本
│   ├── analyze-ingress.sh      # 分析现有 Ingress 配置
│   ├── generate-kind-config.sh # 生成 Kind 集群配置
│   └── validate-migration.sh   # 验证迁移结果
└── references/                  # 参考文档和示例
    ├── ingress-examples/       # Ingress 示例
    ├── wasm-plugins/           # WASM 插件示例
    └── migration-checklist.md  # 迁移检查清单
```

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

### 场景 3：多集群协同迁移
多个数据中心的 Nginx Ingress 集群需要统一升级

**特点**：
- 地域分散的多个集群
- 配置差异较大
- 需要跨地域灾备
- **迁移复杂度**：高，需要定制化方案

## 实践案例

### 完整迁移记录：[PRACTICE.md](./PRACTICE.md)

**规模**：60+ Ingress 资源，3 节点高可用集群

**流程**：
1. 📋 **现状分析**：60+ Ingress 资源、30+ Service 后端、自定义配置
2. 🧪 **Kind 仿真**：本地验证 100% 配置兼容性
3. 🚀 **灰度迁移**：3 阶段灰度，完全验证后下线 Nginx
4. 🤖 **功能补齐**：Agent 自动开发 WASM 插件，解决不支持的功能

**关键成果**：
- ✅ 所有配置零改动迁移成功
- ✅ 未支持的功能通过 WASM 插件智能补齐
- ✅ 迁移过程业务无感知、零故障

## 常见问题

### Q: 迁移需要改动现有 Ingress 配置吗？
**A**: 不需要。标准的 Ingress 资源和注解 100% 兼容，可直接迁移。

### Q: Nginx ConfigMap 中的自定义配置怎么处理？
**A**: 通过 Agent 自动开发 WASM 插件补齐功能，代码自动生成和部署。

### Q: 迁移过程中出现问题如何回滚？
**A**: 采用灰度策略，保留原 Nginx 集群，可随时切回。推荐保留至少 1 周。

### Q: WASM 插件的性能如何？
**A**: WASM 插件编译后直接运行，性能优秀，相比 Lua 脚本更高效且更安全。

## 最佳实践

1. **前期评估** - 用脚本分析现有配置，识别迁移风险和不兼容项
2. **本地仿真** - Kind 集群快速验证，确保配置完全兼容
3. **灰度部署** - 分阶段灰度，监控关键指标
4. **插件优先** - 复杂功能让 Agent 自动设计，提升可维护性
5. **持续观测** - 接入网关监控，设置告警，确保平稳运行

## 技术栈

- **源网关**：Nginx Ingress Controller
- **目标网关**：Higress（基于 Envoy）
- **本地仿真**：Kind（Kubernetes in Docker）
- **插件开发**：Go 1.24+ + WebAssembly (WASM)
- **API**：proxy-wasm-go-sdk

## 相关资源

- [Higress 官方文档](https://higress.io/)
- [Nginx Ingress Controller](https://kubernetes.github.io/ingress-nginx/)
- [WASM 插件开发指南](./SKILL.md)
- [完整实践记录](./PRACTICE.md)

## 支持和反馈

- 有问题？查看 [SKILL.md](./SKILL.md) 的常见问题部分
- 想分享经验？欢迎提交 PR 或 Issue
- 需要帮助？使用 Agent 自动分析和生成解决方案

---

**通过自动化、智能化的迁移流程，让 Nginx Ingress 升级变成简单、安全、高效的过程。**
