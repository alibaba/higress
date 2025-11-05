# Higress AI Capability Auth (MCP-GUARD)

AI能力授权系统 - 基于Higress和Wasm插件的多租户权限管理解决方案

## 项目结构

```
/home/ink/1103/
├── higress/                    # Higress 核心代码
│   ├── plugins/wasm-go/        # Wasm 插件开发
│   │   └── extensions/mcp-guard/  # MCP-GUARD 插件
├── samples/mcp-guard/          # 演示配置和脚本
├── docs/mcp-guard/             # 📚 完整文档
└── scripts/                    # 工具脚本
```

## 📚 文档导航

### 快速开始
- **[演示总结](docs/mcp-guard/PROJECT-SUMMARY.txt)** - 项目概述和成果展示
- **[汇报PPT](docs/mcp-guard/MCP-GUARD-Presentation-Summary.md)** - 领导汇报摘要
- **[使用指南](docs/mcp-guard/README-FOR-REPORT.md)** - 文档索引和使用说明

### 技术文档
- **[架构报告](docs/mcp-guard/MCP-GUARD-Architecture-Report.md)** - 详细技术报告
- **[架构图集](docs/mcp-guard/MCP-GUARD-Architecture-Diagrams.md)** - 9张专业架构图
- **[开发指南](docs/mcp-guard/CLAUDE.md)** - 为Claude Code提供的开发指导

### 演示配置
- **[演示脚本](samples/mcp-guard/04-demo-script.sh)** - 一键部署脚本
- **[插件配置](samples/mcp-guard/03-wasmplugins-deepseek.yaml)** - WasmPlugin配置
- **[授权配置](samples/mcp-guard/higress-config.yaml)** - 权限策略配置

## 🎯 核心特性

✅ **多租户治理** - 基于能力集的差异化授权
✅ **毫秒级判定** - 数据面本地权限判定
✅ **零改造接入** - ai-proxy统一协议适配
✅ **生产就绪** - Wasm沙箱隔离，热更新无中断

## 🚀 快速开始

### 1. 运行演示
```bash
cd samples/mcp-guard
bash 04-demo-script.sh
```

### 2. 测试授权
```bash
# 授权拒绝（tenantB 访问 translate）
curl -i -H 'X-Subject: tenantB' \
     -H 'X-MCP-Capability: cap.text.translate' \
     http://127.0.0.1/v1/text:translate

# 授权通过（tenantA 访问 summarize）
curl -i -H 'X-Subject: tenantA' \
     -H 'X-MCP-Capability: cap.text.summarize' \
     http://127.0.0.1/v1/text:summarize
```

## 📊 测试结果

- **测试通过率**: 100% (4/4)
- **授权延迟**: < 1ms
- **插件大小**: 5.4MB (mcp-guard.wasm)
- **测试环境**: kind Kubernetes + Higress 2.1.9-rc.1

## 💡 权限模型

```
tenantA (白金客户) → [cap.text.summarize, cap.text.translate]
tenantB (标准客户) → [cap.text.summarize]
```

## 🏗️ 技术栈

- **网关**: Higress (Istio + Envoy)
- **扩展**: Wasm插件 (Go)
- **控制面**: Kubernetes + Go
- **配置**: WasmPlugin CRD + xDS

## 📞 联系方式

- **项目仓库**: git@github.com:ink-hz/higress-ai-capability-auth.git
- **技术文档**: [docs/mcp-guard/](docs/mcp-guard/)
- **演示配置**: [samples/mcp-guard/](samples/mcp-guard/)

## 📄 许可证

本项目基于 Apache 2.0 许可证开源。
