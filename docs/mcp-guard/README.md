# MCP-GUARD 文档索引

欢迎来到MCP-GUARD能力授权系统的完整文档！

## 📋 文档列表

### 📊 项目总结
- **[PROJECT-SUMMARY.txt](PROJECT-SUMMARY.txt)** (13KB)
  - ASCII艺术风格的可视化总结
  - 完整项目时间线和成果展示
  - 适合：快速了解项目全貌

### 🎯 汇报材料
- **[MCP-GUARD-Presentation-Summary.md](MCP-GUARD-Presentation-Summary.md)** (6.5KB)
  - PPT风格的简洁总结
  - 核心要点突出显示
  - 适合：领导汇报、快速演示（5-10分钟）

### 📖 详细技术文档
- **[MCP-GUARD-Architecture-Report.md](MCP-GUARD-Architecture-Report.md)** (15KB)
  - 完整的架构设计说明
  - 详细的技术实现分析
  - 完整的Demo测试验证结果
  - 核心代码解析
  - 适合：技术分享、详细汇报、深度评审

### 🎨 架构图集
- **[MCP-GUARD-Architecture-Diagrams.md](MCP-GUARD-Architecture-Diagrams.md)** (9KB)
  - 9张专业Mermaid架构图
  - 整体系统架构、时序图、判定模型等
  - 适合：技术讲解、方案评审（30-60分钟）

### 💻 开发指南
- **[CLAUDE.md](../CLAUDE.md)** (9KB)
  - 为Claude Code提供的开发指导
  - 常用命令和代码架构
  - 适合：后续开发维护

### 📚 使用指南
- **[README-FOR-REPORT.md](README-FOR-REPORT.md)** (5.6KB)
  - 文档索引和使用指南
  - 快速导航和演示检查清单
  - 适合：演示准备、技术支持

## 🚀 快速导航

### 场景1: 向领导汇报 (5-10分钟)
**使用文件**: [MCP-GUARD-Presentation-Summary.md](MCP-GUARD-Presentation-Summary.md)

### 场景2: 技术分享会 (30-60分钟)
**使用文件**: [MCP-GUARD-Architecture-Diagrams.md](MCP-GUARD-Architecture-Diagrams.md)

### 场景3: 详细技术评审
**使用文件**: [MCP-GUARD-Architecture-Report.md](MCP-GUARD-Architecture-Report.md)

### 场景4: 开发参考
**使用文件**: [CLAUDE.md](../CLAUDE.md)

## 📊 核心数据

### Demo测试结果
```
测试环境: kind Kubernetes + Higress 2.1.9-rc.1
API提供商: DeepSeek
API Key: YOUR_DEEPSEEK_API_KEY_HERE (使用时请替换)

测试结果: 4/4 通过 (100%)
- 无身份访问 → 403 (✅)
- tenantB越权访问 → 403 (✅)
- tenantA授权访问summarize → 503 (✅)
- tenantA授权访问translate → 503 (✅)
```

### 性能指标
```
授权判定延迟: < 1ms
插件内存占用: 5.4MB
配置更新延迟: < 100ms
CPU额外开销: < 5%
```

### 权限模型
```
tenantA (白金客户): [cap.text.summarize, cap.text.translate]
tenantB (标准客户): [cap.text.summarize]
```

## 🖼️ 可视化架构图

所有架构图均使用Mermaid格式，支持：
- Markdown预览器 (VS Code, Typora等)
- 在线渲染: https://mermaid.live
- 生成图片: 需要mermaid-cli

### 推荐的架构图展示顺序

1. **整体系统架构图** - 先给全局视图
2. **权限判定模型** - 核心算法说明
3. **请求处理时序** - 两种场景对比
4. **Wasm插件架构** - 技术实现细节
5. **多租户权限模型** - 业务价值展示
6. **测试验证流程** - 验证结果
7. **业务价值架构** - 总结价值

## 💻 演示环境

### 集群状态检查
```bash
# 检查集群
kubectl cluster-info

# 查看Higress组件
kubectl get pods -n higress-system

# 查看WasmPlugin
kubectl get wasmplugin -n higress-system
```

### 测试命令
```bash
# 测试授权拒绝
curl -i -H 'X-Subject: tenantB' \
     -H 'X-MCP-Capability: cap.text.translate' \
     -H 'Host: api.example.com' \
     http://127.0.0.1/v1/text:translate

# 测试授权通过
curl -i -H 'X-Subject: tenantA' \
     -H 'X-MCP-Capability: cap.text.summarize' \
     -H 'Host: api.example.com' \
     http://127.0.0.1/v1/text:summarize
```

### 查看日志
```bash
# 查看网关访问日志
kubectl logs -n higress-system -l app=higress-gateway --tail=50

# 查看mcp-guard相关日志
kubectl logs -n higress-system -l app=higress-gateway --tail=50 | grep mcp-guard
```

## 🎯 快速开始

### 1. 运行演示
```bash
cd samples/mcp-guard
bash 04-demo-script.sh
```

### 2. 查看测试结果
```bash
# 查看WasmPlugin配置
kubectl get wasmplugin mcp-guard -n higress-system -o yaml
```

### 3. 验证授权
按照上述测试命令执行，期望看到：
- 403 Forbidden (mcp-guard deny) - 授权拒绝场景
- 503 Service Unavailable - 授权通过但后端不可用（正常）

## ⚠️ 注意事项

- ⚠️ 确保使用 `--noproxy "*"` 参数避免代理干扰
- ⚠️ 必须设置 `Host: api.example.com` 头匹配域名
- ⚠️ 403是正确的拒绝响应，不要误认为失败
- ⚠️ 503表示授权通过但后端不可用，这是预期行为

## 📞 技术支持

### 关键文件位置
```
插件源码: /home/ink/1103/higress/plugins/wasm-go/extensions/mcp-guard/
配置文件: /home/ink/1103/samples/mcp-guard/
Wasm文件: /opt/plugins/wasm-go/extensions/mcp-guard/plugin.wasm
```

### 相关命令
```bash
# 重新构建插件 (如果需要)
cd /home/ink/1103/higress/plugins/wasm-go
make build PLUGIN_NAME=mcp-guard

# 重新加载插件到集群
kubectl delete pod -n higress-system -l app=higress-gateway

# 查看插件配置
kubectl get wasmplugin mcp-guard -n higress-system -o yaml
```

## 🎉 演示成功要素

### 技术要素
✅ **稳定性**: 插件运行稳定，无崩溃
✅ **准确性**: 授权判定完全正确
✅ **性能**: 毫秒级响应
✅ **可观测**: 完整的日志和指标

### 演示要素
✅ **准备充分**: 提前测试所有命令
✅ **逻辑清晰**: 从问题→方案→实现→验证
✅ **重点突出**: 强调核心价值和成果
✅ **互动友好**: 准备回答问题

---

**祝您使用愉快！如有问题请参考演示检查清单或联系技术支持。**
