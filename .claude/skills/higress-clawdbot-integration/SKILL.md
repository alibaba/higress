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
   
   **Top Commonly Used Providers:**
   - Aliyun Dashscope (Qwen): `--dashscope-key`
   - DeepSeek: `--deepseek-key`
   - Moonshot (Kimi): `--moonshot-key`
   - Zhipu AI: `--zhipuai-key`
   - Minimax: `--minimax-key`
   - Azure OpenAI: `--azure-key`
   - AWS Bedrock: `--bedrock-key`
   - Google Vertex AI: `--vertex-key`
   - OpenAI: `--openai-key`
   - OpenRouter: `--openrouter-key`
   - Grok: `--grok-key`
   
   See CLI Parameters Reference for complete list with model pattern options.

2. **Port Configuration** (optional):
   - HTTP port: `--http-port` (default: 8080)
   - HTTPS port: `--https-port` (default: 8443)
   - Console port: `--console-port` (default: 8001)

3. **Auto-routing** (optional):
   - Enable: `--auto-routing`
   - Default model: `--auto-routing-default-model`

### Step 3: Detect Optimal Image Repository

Before running the deployment script, automatically detect the current timezone and select the geographically closest image repository for faster downloads:

```bash
# Detect timezone and select optimal image repository
TZ=$(timedatectl show --property=Timezone --value 2>/dev/null || cat /etc/timezone 2>/dev/null || echo "UTC")

case "$TZ" in
  Asia/Shanghai|Asia/Hong_Kong|Asia/Taipei|Asia/Chongqing|Asia/Urumqi|Asia/Harbin)
    # China and nearby regions
    IMAGE_REPO="higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one"
    ;;
  Asia/Singapore|Asia/Jakarta|Asia/Bangkok|Asia/Kuala_Lumpur|Asia/Manila|Asia/Ho_Chi_Minh)
    # Southeast Asia
    IMAGE_REPO="higress-registry.ap-southeast-7.cr.aliyuncs.com/higress/all-in-one"
    ;;
  America/*|US/*|Canada/*)
    # North America
    IMAGE_REPO="higress-registry.us-west-1.cr.aliyuncs.com/higress/all-in-one"
    ;;
  *)
    # Default to Hangzhou for other regions
    IMAGE_REPO="higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one"
    ;;
esac

echo "Auto-selected image repository based on timezone ($TZ): $IMAGE_REPO"
```

**Available Image Repositories:**
- **Hangzhou/China**: `higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one` (default)
  - Optimal for: China (Asia/Shanghai, Asia/Hong_Kong, etc.)
- **Southeast Asia**: `higress-registry.ap-southeast-7.cr.aliyuncs.com/higress/all-in-one`
  - Optimal for: Singapore, Indonesia, Thailand, Malaysia, Philippines, Vietnam
- **North America**: `higress-registry.us-west-1.cr.aliyuncs.com/higress/all-in-one`
  - Optimal for: United States, Canada, Mexico

### Step 4: Run Setup Script

Run the script in non-interactive mode with gathered parameters and auto-selected image repository:

```bash
IMAGE_REPO="$IMAGE_REPO" ./get-ai-gateway.sh start --non-interactive \
  --dashscope-key sk-xxx \
  --openai-key sk-xxx \
  --auto-routing \
  --auto-routing-default-model qwen-turbo
```

**Note:** The `IMAGE_REPO` environment variable is automatically set based on the detected timezone. This ensures optimal download speeds without user intervention.

### Step 5: Verify Deployment

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

### Step 6: Configure Clawdbot/OpenClaw Plugin

If the user wants to use Higress with Clawdbot/OpenClaw, install the appropriate plugin:

#### Automatic Installation

Detect runtime and install the correct plugin version:

```bash
# Detect which runtime is installed
if command -v clawdbot &> /dev/null; then
  RUNTIME="clawdbot"
  RUNTIME_DIR="$HOME/.clawdbot"
  PLUGIN_SRC="scripts/plugin-clawdbot"
elif command -v openclaw &> /dev/null; then
  RUNTIME="openclaw"
  RUNTIME_DIR="$HOME/.openclaw"
  PLUGIN_SRC="scripts/plugin"
else
  echo "Error: Neither clawdbot nor openclaw is installed"
  exit 1
fi

# Install the plugin
PLUGIN_DEST="$RUNTIME_DIR/extensions/higress-ai-gateway"
echo "Installing Higress AI Gateway plugin for $RUNTIME..."
mkdir -p "$(dirname "$PLUGIN_DEST")"
[ -d "$PLUGIN_DEST" ] && rm -rf "$PLUGIN_DEST"
cp -r "$PLUGIN_SRC" "$PLUGIN_DEST"
echo "✓ Plugin installed at: $PLUGIN_DEST"

# Configure provider
echo
echo "Configuring provider..."
$RUNTIME models auth login --provider higress
```

The plugin will guide you through an interactive setup for:
1. Gateway URL (default: `http://localhost:8080`)
2. Console URL (default: `http://localhost:8001`)
3. API Key (optional for local deployments)
4. Model list (auto-detected or manually specified)
5. Auto-routing default model (if using `higress/auto`)

### Step 7: Manage API Keys (optional)

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
| `--auto-routing` | Enable auto-routing feature | - |
| `--auto-routing-default-model` | Default model when no rule matches | - |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `IMAGE_REPO` | Docker image repository URL (auto-selected based on timezone) | `higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one` |

**Auto-Selection Logic:**
- Asia/Shanghai and China timezones → Hangzhou mirror
- Southeast Asia timezones → Singapore mirror
- America/* timezones → North America mirror
- Other timezones → Hangzhou mirror (default)

### LLM Provider API Keys

**Top Providers:**

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

**Additional Providers:**
`--doubao-key`, `--baichuan-key`, `--yi-key`, `--stepfun-key`, `--minimax-key`, `--cohere-key`, `--mistral-key`, `--github-key`, `--fireworks-key`, `--togetherai-key`, `--grok-key`, `--azure-key`, `--bedrock-key`, `--vertex-key`

## Managing Configuration

### API Keys

```bash
# List all configured API keys
./get-ai-gateway.sh config list

# Add or update an API key (hot-reload)
./get-ai-gateway.sh config add --provider deepseek --key sk-xxx

# Remove an API key (hot-reload)
./get-ai-gateway.sh config remove --provider deepseek
```

**Supported provider aliases:**
`dashscope`/`qwen`, `moonshot`/`kimi`, `zhipuai`/`zhipu`, `togetherai`/`together`

### Routing Rules

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

Gateway access logs are available at:
```
$DATA_FOLDER/logs/access.log
```

These logs can be used with the **agent-session-monitor** skill for token tracking and conversation analysis.

## Related Skills

- **higress-auto-router**: Configure automatic model routing using CLI commands  
  See: [higress-auto-router](../higress-auto-router/SKILL.md)

- **agent-session-monitor**: Monitor and track token usage across sessions  
  See: [agent-session-monitor](../agent-session-monitor/SKILL.md)

## Examples

### Example 1: Basic Deployment with Dashscope

**User:** 帮我部署一个Higress AI网关，使用阿里云的通义千问

**Steps:**
1. Download script
2. Get Dashscope API key from user
3. Auto-detect timezone and select image repository
4. Run:
   ```bash
   # Auto-detect timezone
   TZ=$(timedatectl show --property=Timezone --value 2>/dev/null || echo "Asia/Shanghai")
   
   # Select repository (Asia/Shanghai detected, using Hangzhou mirror)
   IMAGE_REPO="higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one"
   
   IMAGE_REPO="$IMAGE_REPO" ./get-ai-gateway.sh start --non-interactive \
     --dashscope-key sk-xxx
   ```

**Response:**
```
检测到时区: Asia/Shanghai
自动选择镜像: higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one

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

### Example 2: Full Integration with Clawdbot

**User:** 完整配置Higress和Clawdbot的集成

**Steps:**
1. Auto-detect timezone and select optimal image repository
2. Deploy Higress AI Gateway
3. Install and configure Clawdbot plugin
4. Enable auto-routing
5. Set up session monitoring

**Response:**
```
检测到时区: Asia/Shanghai
自动选择镜像: higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one (杭州镜像)

✅ Higress AI Gateway 集成完成！

1. 网关已部署:
   - HTTP: http://localhost:8080
   - Console: http://localhost:8001
   - 镜像: 杭州镜像 (基于时区自动选择)

2. Clawdbot 插件配置:
   Plugin installed at: /root/.clawdbot/extensions/higress-ai-gateway
   Run: clawdbot models auth login --provider higress

3. 自动路由:
   已启用，使用 model="higress/auto"

4. 会话监控:
   日志路径: ./higress/logs/access.log

需要我帮你配置自动路由规则吗？
```

### Example 3: Manage API Keys

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

✅ API key updated successfully!

Provider: DeepSeek
Key: sk-xx***yy56

Configuration has been hot-reloaded (no restart needed).
```

### Example 4: North America Deployment

**User:** 帮我部署Higress AI网关

**Context:** User's timezone is America/Los_Angeles

**Steps:**
1. Download script
2. Get API keys from user
3. Auto-detect timezone (America/Los_Angeles detected)
4. Auto-select North America mirror
5. Run deployment:
   ```bash
   # Auto-detect timezone
   TZ=$(timedatectl show --property=Timezone --value)  # Returns: America/Los_Angeles
   
   # Auto-select North America mirror
   IMAGE_REPO="higress-registry.us-west-1.cr.aliyuncs.com/higress/all-in-one"
   
   IMAGE_REPO="$IMAGE_REPO" ./get-ai-gateway.sh start --non-interactive \
     --openai-key sk-xxx \
     --openrouter-key sk-xxx
   ```

**Response:**
```
检测到时区: America/Los_Angeles
自动选择镜像: higress-registry.us-west-1.cr.aliyuncs.com/higress/all-in-one (北美镜像)

✅ Higress AI Gateway 部署完成！

网关地址: http://localhost:8080/v1/chat/completions
控制台: http://localhost:8001
日志目录: ./higress/logs
使用镜像: 北美镜像 (基于时区自动选择，优化下载速度)

已配置的模型提供商:
- OpenAI
- OpenRouter
```

## Troubleshooting

For detailed troubleshooting guides, see [TROUBLESHOOTING.md](references/TROUBLESHOOTING.md).

Common issues:
- **Container fails to start**: Check Docker status, port availability, and container logs
- **"too many open files" error**: Increase `fs.inotify.max_user_instances` to 8192
- **Gateway not responding**: Verify container status and port mapping
- **Plugin not recognized**: Check installation path and restart runtime
- **Auto-routing not working**: Verify model list and routing rules
- **Timezone detection fails**: Manually set `IMAGE_REPO` environment variable
