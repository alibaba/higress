---
name: higress-openclaw-integration
description: "Deploy and configure Higress AI Gateway for OpenClaw integration. Use when: (1) User wants to deploy Higress AI Gateway, (2) User wants to configure OpenClaw to use Higress as a model provider, (3) User mentions 'higress', 'ai gateway', 'model gateway', 'AI网关', (4) User wants to set up model routing or auto-routing, (5) User needs to manage LLM provider API keys."
---

# Higress AI Gateway Integration

Deploy Higress AI Gateway and configure OpenClaw to use it as a unified model provider.

## Quick Start

### Step 1: Collect Information from User

**Ask the user for the following information upfront:**

1. **Which LLM provider(s) to use?** (at least one required)

   | Provider | Parameter | Notes |
   |----------|-----------|-------|
   | 阿里云通义千问 (Dashscope) | `--dashscope-key` | Models: qwen-* |
   | DeepSeek | `--deepseek-key` | Models: deepseek-* |
   | Moonshot (Kimi) | `--moonshot-key` | Models: moonshot-*, kimi-* |
   | 智谱 AI (Zhipu) | `--zhipuai-key` | Models: glm-* |
   | OpenAI | `--openai-key` | Models: gpt-*, o1-*, o3-* |
   | Claude | `--claude-key` | Models: claude-* |
   | Claude Code | `--claude-code-key` | **⚠️ 需运行 `claude setup-token` 获取 OAuth Token** |
   | Google Gemini | `--gemini-key` | Models: gemini-* |
   | OpenRouter | `--openrouter-key` | Supports all models (catch-all) |
   | Grok | `--grok-key` | Models: grok-* |
   | Groq | `--groq-key` | Fast inference |
   | Doubao (豆包) | `--doubao-key` | Models: doubao-* |
   | Minimax | `--minimax-key` | Models: abab-* |
   | Mistral | `--mistral-key` | Models: mistral-* |
   | Baichuan (百川) | `--baichuan-key` | Models: Baichuan* |
   | 01.AI (Yi) | `--yi-key` | Models: yi-* |
   | Stepfun (阶跃星辰) | `--stepfun-key` | Models: step-* |
   | Cohere | `--cohere-key` | Models: command* |
   | Fireworks AI | `--fireworks-key` | - |
   | Together AI | `--togetherai-key` | - |
   | GitHub Models | `--github-key` | - |

   **Cloud Providers (require additional config):**
   - Azure OpenAI: `--azure-key` (需要 service URL)
   - AWS Bedrock: `--bedrock-key` (需要 region 和 access key)
   - Google Vertex AI: `--vertex-key` (需要 project ID 和 region)

2. **Enable auto-routing?** (recommended)
   - If yes: `--auto-routing --auto-routing-default-model <model-name>`
   - Auto-routing allows using `model="higress/auto"` to automatically route requests based on message content

3. **Custom ports?** (optional, defaults: HTTP=8080, HTTPS=8443, Console=8001)

### Step 2: Deploy Gateway

```bash
# Download script (if not exists)
curl -fsSL https://raw.githubusercontent.com/higress-group/higress-standalone/main/all-in-one/get-ai-gateway.sh -o get-ai-gateway.sh
chmod +x get-ai-gateway.sh

# Deploy with user's configuration
./get-ai-gateway.sh start --non-interactive \
  --<provider>-key <api-key> \
  [--auto-routing --auto-routing-default-model <model>]
```

**Example:**
```bash
./get-ai-gateway.sh start --non-interactive \
  --zhipuai-key sk-xxx \
  --auto-routing \
  --auto-routing-default-model glm-4
```

### Step 3: Install OpenClaw Plugin

Install the Higress provider plugin for OpenClaw:

```bash
# Copy plugin files (PLUGIN_SRC is relative to skill directory: scripts/plugin)
PLUGIN_SRC="scripts/plugin"
PLUGIN_DEST="$HOME/.openclaw/extensions/higress-ai-gateway"

mkdir -p "$PLUGIN_DEST"
cp -r "$PLUGIN_SRC"/* "$PLUGIN_DEST/"

# Configure provider (interactive setup)
openclaw models auth login --provider higress
```

The `openclaw models auth login` command will prompt for:
1. Gateway URL (default: `http://localhost:8080`)
2. Console URL (default: `http://localhost:8001`)
3. API Key (optional for local deployments)
4. Model list (auto-detected or manually specified)
5. Auto-routing default model (if using `higress/auto`)

After configuration, Higress models are available in OpenClaw with `higress/` prefix (e.g., `higress/glm-4`, `higress/auto`).

## Post-Deployment Management

### Add/Update API Keys (Hot-reload)

```bash
./get-ai-gateway.sh config add --provider <provider> --key <api-key>
./get-ai-gateway.sh config list
./get-ai-gateway.sh config remove --provider <provider>
```

Provider aliases: `dashscope`/`qwen`, `moonshot`/`kimi`, `zhipuai`/`zhipu`

### Add Routing Rules (for auto-routing)

```bash
# Add rule: route to specific model when message starts with trigger
./get-ai-gateway.sh route add --model <model> --trigger "关键词1|关键词2"

# Examples
./get-ai-gateway.sh route add --model glm-4-flash --trigger "简单|快速"
./get-ai-gateway.sh route add --model claude-opus-4 --trigger "深入思考|复杂问题"
./get-ai-gateway.sh route add --model deepseek-coder --trigger "写代码|debug"

# List/remove rules
./get-ai-gateway.sh route list
./get-ai-gateway.sh route remove --rule-id 0
```

### Stop/Delete Gateway

```bash
./get-ai-gateway.sh stop
./get-ai-gateway.sh delete
```

## Endpoints

| Endpoint | URL |
|----------|-----|
| Chat Completions | http://localhost:8080/v1/chat/completions |
| Console | http://localhost:8001 |
| Logs | `./higress/logs/access.log` |

## Testing

```bash
# Test with specific model
curl 'http://localhost:8080/v1/chat/completions' \
  -H 'Content-Type: application/json' \
  -d '{"model": "<model-name>", "messages": [{"role": "user", "content": "Hello"}]}'

# Test auto-routing (if enabled)
curl 'http://localhost:8080/v1/chat/completions' \
  -H 'Content-Type: application/json' \
  -d '{"model": "higress/auto", "messages": [{"role": "user", "content": "简单 什么是AI?"}]}'
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Container fails to start | Check `docker logs higress-ai-gateway` |
| Port already in use | Use `--http-port`, `--console-port` to change ports |
| API key error | Run `./get-ai-gateway.sh config list` to verify keys |
| Auto-routing not working | Ensure `--auto-routing` was set during deployment |
| Slow image download | Script auto-selects nearest registry based on timezone |

## Important Notes

1. **Claude Code Mode**: Requires OAuth token from `claude setup-token` command, not a regular API key
2. **Auto-routing**: Must be enabled during initial deployment (`--auto-routing`); routing rules can be added later
3. **OpenClaw Integration**: After plugin installation and `openclaw models auth login --provider higress`, models are available with `higress/` prefix
4. **Hot-reload**: API key changes take effect immediately; no container restart needed
