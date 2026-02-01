---
name: higress-clawdbot-integration
description: "Deploy and configure Higress AI Gateway for Clawdbot/OpenClaw integration. Use when: (1) User wants to deploy Higress AI Gateway, (2) User wants to configure Clawdbot/OpenClaw to use Higress as a model provider, (3) User mentions 'higress', 'ai gateway', 'model gateway', 'AI网关', (4) User wants to set up model routing or auto-routing, (5) User needs to manage LLM provider API keys, (6) User wants to track token usage and conversation history."
---

# Higress AI Gateway Integration

Deploy and configure Higress AI Gateway for Clawdbot/OpenClaw integration with one-click deployment, model provider configuration, auto-routing, and session monitoring.

## Prerequisites

- Docker installed and running
- Internet access to download the setup script
- LLM provider API keys (at least one)

## Workflow

### Step 1: Download Setup Script

Download the official get-ai-gateway.sh script:

```bash
curl -fsSL https://raw.githubusercontent.com/higress-group/higress-standalone/main/all-in-one/get-ai-gateway.sh -o get-ai-gateway.sh
chmod +x get-ai-gateway.sh
```

### Step 2: Gather Configuration

Ask the user for:

1. **LLM Provider API Keys** (at least one required):
   - Dashscope (Qwen): `--dashscope-key`
   - DeepSeek: `--deepseek-key`
   - OpenAI: `--openai-key`
   - OpenRouter: `--openrouter-key`
   - Claude: `--claude-key`
   - See CLI Parameters Reference for complete list

2. **Port Configuration** (optional):
   - HTTP port: `--http-port` (default: 8080)
   - HTTPS port: `--https-port` (default: 8443)
   - Console port: `--console-port` (default: 8001)

3. **Auto-routing** (optional):
   - Enable: `--auto-routing`
   - Default model: `--auto-routing-default-model`

### Step 3: Run Setup Script

Run the script in non-interactive mode with gathered parameters:

```bash
./get-ai-gateway.sh start --non-interactive \
  --dashscope-key sk-xxx \
  --openai-key sk-xxx \
  --auto-routing \
  --auto-routing-default-model qwen-turbo
```

### Step 4: Verify Deployment

After script completion:

1. Check container is running:
   ```bash
   docker ps --filter "name=higress-ai-gateway"
   ```

2. Test the gateway endpoint:
   ```bash
   curl http://localhost:8080/v1/models
   ```

3. Access the console (optional):
   ```
   http://localhost:8001
   ```

### Step 5: Configure Clawdbot/OpenClaw (if applicable)

If the user wants to use Higress with Clawdbot/OpenClaw:

```bash
# For Clawdbot
clawdbot models auth login --provider higress

# For OpenClaw
openclaw models auth login --provider higress
```

This configures Clawdbot/OpenClaw to use Higress AI Gateway as a model provider.

### Step 6: Manage API Keys (optional)

After deployment, manage API keys without redeploying:

```bash
# View configured API keys
./get-ai-gateway.sh config list

# Add or update an API key (hot-reload, no restart needed)
./get-ai-gateway.sh config add --provider <provider> --key <api-key>

# Remove an API key (hot-reload, no restart needed)
./get-ai-gateway.sh config remove --provider <provider>
```

**Note:** Changes take effect immediately via hot-reload. No container restart required.

## CLI Parameters Reference

### Basic Options
| Parameter | Description | Default |
|-----------|-------------|---------|
| `--non-interactive` | Run without prompts | - |
| `--http-port` | Gateway HTTP port | 8080 |
| `--https-port` | Gateway HTTPS port | 8443 |
| `--console-port` | Console port | 8001 |
| `--container-name` | Container name | higress-ai-gateway |
| `--data-folder` | Data folder path | ./higress |

### Auto-Routing Options
| Parameter | Description |
|-----------|-------------|
| `--auto-routing` | Enable auto-routing feature |
| `--auto-routing-default-model` | Default model when no rule matches |

### LLM Provider API Keys
| Parameter | Provider |
|-----------|----------|
| `--dashscope-key` | Aliyun Dashscope (Qwen) |
| `--deepseek-key` | DeepSeek |
| `--moonshot-key` | Moonshot (Kimi) |
| `--zhipuai-key` | Zhipu AI |
| `--openai-key` | OpenAI |
| `--openrouter-key` | OpenRouter |
| `--claude-key` | Claude |
| `--gemini-key` | Google Gemini |
| `--groq-key` | Groq |
| `--doubao-key` | Doubao |
| `--baichuan-key` | Baichuan AI |
| `--yi-key` | 01.AI (Yi) |
| `--stepfun-key` | Stepfun |
| `--minimax-key` | Minimax |
| `--cohere-key` | Cohere |
| `--mistral-key` | Mistral AI |
| `--github-key` | Github Models |
| `--fireworks-key` | Fireworks AI |
| `--togetherai-key` | Together AI |
| `--grok-key` | Grok |

## Managing API Keys

After deployment, use the `config` subcommand to manage LLM provider API keys:

```bash
# List all configured API keys
./get-ai-gateway.sh config list

# Add or update an API key
./get-ai-gateway.sh config add --provider deepseek --key sk-xxx

# Remove an API key
./get-ai-gateway.sh config remove --provider deepseek
```

**Important:** API key changes take effect immediately via hot-reload. No container restart is required.

**Supported providers:**
- `dashscope` (or `qwen`) - Aliyun Dashscope (Qwen)
- `deepseek` - DeepSeek
- `moonshot` (or `kimi`) - Moonshot (Kimi)
- `zhipuai` (or `zhipu`) - Zhipu AI
- `openai` - OpenAI
- `openrouter` - OpenRouter
- `claude` - Claude
- `gemini` - Google Gemini
- `groq` - Groq
- `doubao` - Doubao
- `baichuan` - Baichuan AI
- `yi` - 01.AI (Yi)
- `stepfun` - Stepfun
- `minimax` - Minimax
- `cohere` - Cohere
- `mistral` - Mistral AI
- `github` - Github Models
- `fireworks` - Fireworks AI
- `togetherai` (or `together`) - Together AI
- `grok` - Grok

## Managing Routing Rules

After deployment, use the `route` subcommand to manage auto-routing rules:

```bash
# Add a routing rule
./get-ai-gateway.sh route add --model claude-opus-4.5 --trigger "深入思考|deep thinking"

# List all rules
./get-ai-gateway.sh route list

# Remove a rule
./get-ai-gateway.sh route remove --rule-id 0
```

See [higress-auto-router](../higress-auto-router/SKILL.md) for detailed documentation.

## Access Logs

After deployment, gateway access logs are available at:
```
$DATA_FOLDER/logs/access.log
```

These logs can be used with the **agent-session-monitor** skill for token tracking and conversation analysis.

## Related Skills

### higress-auto-router
Configure automatic model routing using CLI commands. Example:
```bash
./get-ai-gateway.sh route add --model claude-opus-4.5 --trigger "深入思考|deep thinking"
```

See: [higress-auto-router](../higress-auto-router/SKILL.md)

### agent-session-monitor
Monitor and track token usage across sessions. Example:
- View session statistics in web UI
- Export FinOps reports
- Parse logs from `$DATA_FOLDER/logs/access.log`

See: [agent-session-monitor](../agent-session-monitor/SKILL.md)

## Examples

### Example 1: Basic Deployment with Dashscope

**User:** 帮我部署一个Higress AI网关，使用阿里云的通义千问

**Steps:**
1. Download script
2. Get Dashscope API key from user
3. Run:
   ```bash
   ./get-ai-gateway.sh start --non-interactive \
     --dashscope-key sk-xxx
   ```

**Response:**
```
✅ Higress AI Gateway 部署完成！

网关地址: http://localhost:8080/v1/chat/completions
控制台: http://localhost:8001
日志目录: ./higress/logs

已配置的模型提供商:
- Aliyun Dashscope (Qwen)

测试命令:
curl 'http://localhost:8080/v1/chat/completions' \
  -H 'Content-Type: application/json' \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Hello!"}]}'
```

### Example 2: Deployment with Auto-Routing

**User:** 部署Higress网关，启用自动路由，默认使用qwen-turbo

**Steps:**
1. Download script
2. Get API keys
3. Run:
   ```bash
   ./get-ai-gateway.sh start --non-interactive \
     --dashscope-key sk-xxx \
     --auto-routing \
     --auto-routing-default-model qwen-turbo
   ```

**Response:**
```
✅ Higress AI Gateway 部署完成！

网关地址: http://localhost:8080/v1/chat/completions
控制台: http://localhost:8001

🔀 自动路由已启用:
- 默认模型: qwen-turbo
- 使用 model="higress/auto" 触发自动路由

添加路由规则:
./get-ai-gateway.sh route add --model claude-opus-4.5 --trigger "深入思考|deep thinking"
```

### Example 3: Add Routing Rules

**User:** 我希望在解决困难问题时路由到claude-opus-4.5

**Steps:**
```bash
./get-ai-gateway.sh route add \
  --model claude-opus-4.5 \
  --trigger "深入思考|deep thinking"
```

**Response:**
```
✅ 路由规则添加成功！

触发方式：以 "深入思考" 或 "deep thinking" 开头
目标模型：claude-opus-4.5

使用示例：
- 深入思考 这道算法题应该怎么解？
- deep thinking What's the best architecture for this system?
```

### Example 4: Manage API Keys

**User:** 帮我查看当前配置的API keys，并添加一个DeepSeek的key

**Steps:**
1. List current API keys:
   ```bash
   ./get-ai-gateway.sh config list
   ```

2. Add DeepSeek API key:
   ```bash
   ./get-ai-gateway.sh config add --provider deepseek --key sk-xxx
   ```

**Response:**
```
当前配置的API keys:

  Aliyun Dashscope (Qwen): sk-ab***ef12
  OpenAI:                  sk-cd***gh34

Adding API key for DeepSeek...
Updating AI Gateway configuration...

✅ API key updated successfully!

Provider: DeepSeek
Key: sk-xx***yy56

Configuration has been hot-reloaded (no restart needed).
```

### Example 5: Full Integration with Clawdbot/OpenClaw

**User:** 完整配置Higress和Clawdbot的集成

**Steps:**
1. Deploy Higress AI Gateway
2. Configure Clawdbot/OpenClaw provider
3. Enable auto-routing
4. Set up session monitoring

**Response:**
```
✅ Higress AI Gateway 集成完成！

1. 网关已部署:
   - HTTP: http://localhost:8080
   - Console: http://localhost:8001

2. Clawdbot/OpenClaw 配置:
   - Clawdbot: `clawdbot models auth login --provider higress`
   - OpenClaw: `openclaw models auth login --provider higress`

3. 自动路由:
   已启用，使用 model="higress/auto"

4. 会话监控:
   日志路径: ./higress/logs/access.log
   启动 Web UI: python3 agent-session-monitor/scripts/webserver.py --data-dir ./sessions

需要我帮你配置自动路由规则吗？
```

## Troubleshooting

### Container fails to start
- Check Docker is running: `docker info`
- Check port availability: `netstat -tlnp | grep 8080`
- View container logs: `docker logs higress-ai-gateway`

### Gateway not responding
- Check container status: `docker ps -a`
- Verify port mapping: `docker port higress-ai-gateway`
- Test locally: `curl http://localhost:8080/v1/models`

### API key errors
- Verify the API key is correct
- Check provider documentation for key format
- Some providers require additional configuration (e.g., Azure, Bedrock)

## Clawdbot/OpenClaw Plugin Integration

The Higress AI Gateway plugin enables Clawdbot and OpenClaw to use Higress as a model provider with full support for auto-routing and model management.

### Installation

The plugin is automatically available as part of this skill. To install it into your Clawdbot/OpenClaw environment:

```bash
# Detect runtime and set variables
if command -v clawdbot &> /dev/null; then
  RUNTIME="clawdbot"
  RUNTIME_DIR="$HOME/.clawdbot"
elif command -v openclaw &> /dev/null; then
  RUNTIME="openclaw"
  RUNTIME_DIR="$HOME/.openclaw"
else
  echo "Error: Neither clawdbot nor openclaw is installed"
  exit 1
fi

# Install the plugin
PLUGIN_SRC_DIR="scripts/plugin"
PLUGIN_DEST_DIR="$RUNTIME_DIR/extensions/higress-ai-gateway"

if [ ! -d "$PLUGIN_SRC_DIR" ]; then
  echo "Error: Plugin source not found at $PLUGIN_SRC_DIR"
  exit 1
fi

echo "Installing Higress AI Gateway plugin for $RUNTIME..."
mkdir -p "$(dirname "$PLUGIN_DEST_DIR")"
if [ -d "$PLUGIN_DEST_DIR" ]; then
  echo "Plugin already exists, updating..."
  rm -rf "$PLUGIN_DEST_DIR"
fi

cp -r "$PLUGIN_SRC_DIR" "$PLUGIN_DEST_DIR"
echo "✓ Higress AI Gateway plugin installed at: $PLUGIN_DEST_DIR"

echo
echo "To configure the provider, run:"
echo "  $RUNTIME models auth login --provider higress"
```

### Configuration

After installation, configure Higress as a model provider:

```bash
# For Clawdbot
clawdbot models auth login --provider higress

# For OpenClaw
openclaw models auth login --provider higress
```

The plugin will guide you through an interactive setup:

1. **Gateway URL**: HTTP endpoint for Higress AI Gateway (default: `http://localhost:8080`)
2. **Console URL**: Higress Console endpoint for routing config (default: `http://localhost:8001`)
3. **API Key**: Optional API key for authentication (leave empty for local deployments)
4. **Model List**: Comma-separated model IDs (plugin auto-detects available models)
5. **Auto-routing**: If you include `higress/auto` in the model list, configure the default fallback model

### Plugin Features

#### 1. Auto-routing Support

The plugin provides first-class support for Higress auto-routing:

- Use `higress/auto` as the model ID to enable intelligent routing
- Configure default fallback model during setup
- Auto-routing rules are managed separately via the `higress-auto-router` skill

#### 2. Dynamic Model Discovery

During configuration, the plugin:
- Tests connectivity to the gateway
- Fetches available models from the Console API
- Pre-populates the model list with discovered models
- Allows customization of the model list

#### 3. Smart URL Normalization

The plugin automatically:
- Strips trailing slashes from URLs
- Appends `/v1` suffix if missing
- Validates URL format before saving

#### 4. Profile Management

Creates appropriate credential profiles:
- `higress:local` - for local deployments without API key
- `higress:default` - for remote deployments with API key

### Plugin Structure

```
scripts/plugin/
├── index.ts              # Plugin implementation (TypeScript)
├── package.json          # NPM package metadata
└── openclaw.plugin.json  # OpenClaw plugin manifest
```

**index.ts**: Main plugin code implementing the provider registration and authentication flow.

**package.json**: Declares the plugin as an OpenClaw extension with proper metadata.

**openclaw.plugin.json**: Plugin manifest describing supported providers and configuration schema.

### Integration with Skills

The plugin works seamlessly with related skills:

#### higress-auto-router
After plugin setup, use this skill to configure routing rules:
```bash
./get-ai-gateway.sh route add --model claude-opus-4.5 --trigger "深入思考"
```

See: [higress-auto-router](../higress-auto-router/SKILL.md)

#### agent-session-monitor
Track token usage and costs across sessions using gateway access logs:
```bash
python3 agent-session-monitor/scripts/webserver.py --log-path ./higress/logs/access.log
```

See: [agent-session-monitor](../agent-session-monitor/SKILL.md)

### Example: Full Setup Flow

```bash
# 1. Deploy Higress AI Gateway (via get-ai-gateway.sh)
./get-ai-gateway.sh start --non-interactive \
  --dashscope-key sk-xxx \
  --auto-routing

# 2. Detect and install plugin
if command -v clawdbot &> /dev/null; then
  RUNTIME="clawdbot"
  RUNTIME_DIR="$HOME/.clawdbot"
else
  RUNTIME="openclaw"
  RUNTIME_DIR="$HOME/.openclaw"
fi

mkdir -p "$RUNTIME_DIR/extensions"
cp -r scripts/plugin "$RUNTIME_DIR/extensions/higress-ai-gateway"

# 3. Configure provider
$RUNTIME models auth login --provider higress
# Follow interactive prompts to configure gateway URL, models, etc.

# 4. Test the integration
$RUNTIME chat --model higress/auto "Hello, test auto-routing!"

# 5. Configure routing rules (optional)
./get-ai-gateway.sh route add --model claude-opus-4.5 --trigger "深入思考"
```

### Troubleshooting

#### Plugin not recognized
- Verify plugin is installed at `~/.clawdbot/extensions/higress-ai-gateway` or `~/.openclaw/extensions/higress-ai-gateway`
- Check `package.json` contains correct `openclaw.extensions` field
- Restart Clawdbot/OpenClaw after installation

#### Gateway connection fails
- Ensure Higress AI Gateway container is running: `docker ps`
- Verify gateway URL is accessible: `curl http://localhost:8080/v1/models`
- Check firewall/network settings if using remote gateway

#### Models not available
- Run `clawdbot models list` or `openclaw models list` to verify provider is configured
- Check gateway logs: `docker logs higress-ai-gateway`
- Verify API keys are correctly configured in gateway

#### Auto-routing not working
- Confirm `higress/auto` is in your model list
- Check routing rules exist: `./get-ai-gateway.sh route list`
- Verify default model is configured
- Check gateway logs for routing decisions
