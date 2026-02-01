# 在Clawdbot中使用Agent Session Monitor

## 背景

该skill专为Clawdbot设计，用于实时观测对话过程和token开销。当用户询问当前会话的token消耗、对话历史、成本统计时，你可以生成Web界面链接供用户查看。

## 前提条件

1. **后台监控程序已运行**

```bash
# 在服务器上运行（持续监控Higress访问日志）
python3 /path/to/agent-session-monitor/main.py \
  --log-path /var/log/higress/access.log \
  --output-dir /path/to/sessions
```

2. **Web服务器已启动**

```bash
# 启动Web界面服务
python3 /path/to/agent-session-monitor/scripts/webserver.py \
  --data-dir /path/to/sessions \
  --port 8888 \
  --host 0.0.0.0
```

## 使用场景

### 场景1：用户询问token消耗

**用户：** "我这次对话用了多少token？"

**你的回复：**

```markdown
你的当前会话信息：

📊 **Session ID**: agent:main:discord:channel:1465367993012981988

🔗 **查看详情**: http://your-server:8888/session?id=agent:main:discord:channel:1465367993012981988

点击链接可以看到：
✅ 完整对话历史（每轮messages）
✅ Token消耗明细（input/output/reasoning）
✅ 工具调用记录
✅ 实时成本统计

如果需要导出数据，可以访问：
📥 http://your-server:8888/api/session?id=agent:main:discord:channel:1465367993012981988
```

### 场景2：用户询问总体统计

**用户：** "今天我们的对话总共用了多少token？"

**你的回复：**

```markdown
📊 **总体统计**

🔗 查看所有会话: http://your-server:8888

页面功能：
- 📋 所有会话列表（按时间/成本排序）
- 📈 按模型统计token开销
- 💰 总成本计算
- 🔄 每30秒自动刷新

如果需要导出报表：
📥 http://your-server:8888/api/stats
```

### 场景3：用户询问对话历史

**用户：** "我们刚才聊了什么？"

**你的回复：**

```markdown
你可以查看完整对话历史：

🔗 http://your-server:8888/session?id=agent:main:discord:channel:1465367993012981988

每轮对话都包含：
- 💬 完整messages（system/user/assistant/tool）
- ❓ 用户问题
- ✅ AI回答
- 🧠 思考过程（reasoning，如果有）
- 🛠️ 工具调用记录
```

## 获取当前Session ID

在Clawdbot中，你可以从runtime或context中获取当前session的ID：

```python
# 示例（伪代码）
session_id = runtime.session_key  # 或从context中获取
# 例如: "agent:main:discord:channel:1465367993012981988"
```

然后生成URL：

```python
base_url = "http://your-server:8888"
session_url = f"{base_url}/session?id={session_id}"
```

## API端点

如果用户需要程序化访问数据：

| 端点 | 说明 | 示例 |
|------|------|------|
| `/api/sessions` | 所有session列表 | `http://your-server:8888/api/sessions` |
| `/api/session?id=<id>` | 指定session详情 | `http://your-server:8888/api/session?id=sess_123` |
| `/api/stats` | 总体统计（按模型、按日期） | `http://your-server:8888/api/stats` |

## 注意事项

1. **URL替换**：将 `http://your-server:8888` 替换为实际的Web服务器地址
2. **Session ID编码**：如果session ID包含特殊字符，需要URL编码
3. **隐私保护**：确保Web服务器只在可信网络中访问
4. **实时性**：数据每30秒刷新一次，可能有延迟

## 高级用法

### 直接返回JSON数据

对于喜欢编程的用户，可以提供API链接：

```bash
# 获取session数据
curl http://your-server:8888/api/session?id=<session-id> | jq .

# 获取统计数据
curl http://your-server:8888/api/stats | jq '.by_model'
```

### CLI导出报表

如果用户需要离线分析：

```bash
# 导出FinOps报表
python3 scripts/cli.py export finops-report.json --data-dir /path/to/sessions

# 导出CSV格式
python3 scripts/cli.py export finops-report --format csv --data-dir /path/to/sessions
```

## 故障排查

### 问题：Web界面无法访问

检查：
1. Web服务器是否已启动
2. 端口是否正确
3. 防火墙是否允许访问

### 问题：Session数据为空

检查：
1. 后台监控程序是否运行
2. 日志路径是否正确
3. ai_log字段是否包含session_id

### 问题：数据不实时

- 数据每30秒刷新一次
- 也可以手动刷新页面
- 后台监控程序需要持续运行
