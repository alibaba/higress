# Higress Clawdbot Integration SKILL

Deploy and configure Higress AI Gateway for Clawdbot integration.

## Description

This skill helps users deploy Higress AI Gateway and integrate it with Clawdbot. It provides:

1. **One-click deployment**: Download and run the AI Gateway setup script
2. **Model provider configuration**: Configure LLM providers via CLI parameters
3. **Auto-routing setup**: Enable intelligent model routing based on message content
4. **Session monitoring**: Track token usage and conversation history

## When to Use

Use this skill when:
- User wants to deploy Higress AI Gateway
- User wants to configure Clawdbot to use Higress as a model provider
- User mentions "higress", "ai gateway", "model gateway", "统一网关"
- User wants to set up model routing or auto-routing

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
1. **LLM Provider API Keys**: At least one provider is required
   - Dashscope (Qwen): `--dashscope-key`
   - DeepSeek: `--deepseek-key`
   - OpenAI: `--openai-key`
   - OpenRouter: `--openrouter-key`
   - Claude: `--claude-key`
   - And more...

2. **Port Configuration** (optional):
   - HTTP port: `--http-port` (default: 8080)
   - HTTPS port: `--https-port` (default: 8443)
   - Console port: `--console-port` (default: 8001)

3. **Auto-routing** (optional):
   - Enable: `--auto-routing`
   - Default model: `--auto-routing-default-model`

### Step 3: Run Setup Script

Run the script in non-interactive mode with the gathered parameters:

```bash
./get-ai-gateway.sh start --non-interactive \
  --dashscope-key sk-xxx \
  --openai-key sk-xxx \
  --auto-routing \
  --auto-routing-default-model qwen-turbo
```

### Step 4: Verify Deployment

After the script completes:

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

### Step 5: Configure Clawdbot (if applicable)

If the user wants to use Higress with Clawdbot:

```bash
clawdbot models auth login --provider higress
```

This will configure Clawdbot to use Higress AI Gateway as a model provider.

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

## Access Logs

After deployment, gateway access logs are available at:
```
$DATA_FOLDER/logs/access.log
```

These logs can be used with the **agent-session-monitor** skill for token tracking and conversation analysis.

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

## Related Skills

This skill works with:

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

### Example 4: Full Integration with Clawdbot

**User:** 完整配置Higress和Clawdbot的集成

**Steps:**
1. Deploy Higress AI Gateway
2. Configure Clawdbot provider
3. Enable auto-routing
4. Set up session monitoring

**Response:**
```
✅ Higress AI Gateway 集成完成！

1. 网关已部署:
   - HTTP: http://localhost:8080
   - Console: http://localhost:8001

2. Clawdbot 配置:
   运行 `clawdbot models auth login --provider higress`

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
