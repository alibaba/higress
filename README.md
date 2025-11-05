# Higress

<div align="center">
    <img src="https://img.alicdn.com/imgextra/i2/O1CN01NwxLDd20nxfGBjxmZ_!!6000000006895-2-tps-960-290.png" alt="Higress" width="240" height="72.5">
  <br>
  AI Gateway
</div>

Higress is a cloud-native AI Native API Gateway based on Istio and Envoy.

[**Official Site**](https://higress.ai/en/) &nbsp; |
&nbsp; [**Docs**](https://higress.cn/en/docs/latest/overview/what-is-higress/) &nbsp; |
&nbsp; [**Developer Guide**](https://higress.cn/en/docs/latest/dev/architecture/) &nbsp; |
&nbsp; [**MCP Server QuickStart**](https://higress.cn/en/ai/mcp-quick-start/)

---

## MCP-GUARD AI Capability Authorization

This repository contains the **MCP-GUARD** capability authorization system - a multi-tenant permission management solution based on Higress and Wasm plugins.

### 📚 Documentation

- **[Project Summary](docs/mcp-guard/PROJECT-SUMMARY.txt)** - Project overview and achievements
- **[Architecture Report](docs/mcp-guard/MCP-GUARD-Architecture-Report.md)** - Detailed technical report
- **[Architecture Diagrams](docs/mcp-guard/MCP-GUARD-Architecture-Diagrams.md)** - 9 professional architecture diagrams
- **[Presentation Summary](docs/mcp-guard/MCP-GUARD-Presentation-Summary.md)** - Executive briefing summary
- **[Development Guide](docs/mcp-guard/CLAUDE.md)** - Development guide for Claude Code

### 🎯 Key Features

✅ **Multi-tenant Governance** - Capability-based differentiated authorization  
✅ **Millisecond Response** - Data plane local authorization  
✅ **Zero Breaking Changes** - ai-proxy unified protocol adaptation  
✅ **Production Ready** - Wasm sandbox isolation, hot updates without interruption  

### 🚀 Quick Start

```bash
# Run demo
cd samples/mcp-guard
bash 04-demo-script.sh

# Test authorization (deny)
curl -i -H 'X-Subject: tenantB' \
     -H 'X-MCP-Capability: cap.text.translate' \
     http://127.0.0.1/v1/text:translate

# Test authorization (allow)
curl -i -H 'X-Subject: tenantA' \
     -H 'X-MCP-Capability: cap.text.summarize' \
     http://127.0.0.1/v1/text:summarize
```

### 📊 Test Results

- **Pass Rate**: 100% (4/4 test cases)
- **Authorization Delay**: < 1ms
- **Plugin Size**: 5.4MB (mcp-guard.wasm)
- **Test Environment**: kind Kubernetes + Higress 2.1.9-rc.1

### 💡 Permission Model

```
tenantA (Premium): [cap.text.summarize, cap.text.translate]
tenantB (Standard): [cap.text.summarize]
```

---

For more information, please refer to the [documentation](docs/mcp-guard/).
