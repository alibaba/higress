# Agent Session Monitor

实时Agent对话观测程序，专为Clawdbot设计，用于监控Higress访问日志，追踪多轮对话的token开销。

## 特点

- 🔍 **完整对话追踪**：记录每轮的messages、question、answer、reasoning、tool_calls
- 💰 **Token开销统计**：区分input/output/reasoning/cached token，实时计算成本
- 🌐 **Web可视化界面**：浏览器访问，总览+下钻查看session详情
- 🔗 **实时URL生成**：Clawdbot可根据当前会话ID生成观测链接

## Quick Start

### 1. 运行Demo

```bash
cd example
bash demo.sh
```

### 2. 启动Web界面

```bash
# 解析日志
python3 main.py --log-path /var/log/higress/access.log --output-dir ./sessions

# 启动Web服务器
python3 scripts/webserver.py --data-dir ./sessions --port 8888

# 浏览器访问
open http://localhost:8888
```

### 3. 在Clawdbot中使用

当用户询问"我这次对话用了多少token"时，你可以：

```
你的当前会话统计：
- Session ID: agent:main:discord:channel:1465367993012981988
- 查看详情: http://localhost:8888/session?id=agent:main:discord:channel:1465367993012981988

点击链接可以看到：
✅ 完整对话历史
✅ 每轮token消耗明细
✅ 工具调用记录
✅ 成本统计
```

## 文件说明

- `main.py`: 后台监控程序，解析Higress访问日志
- `scripts/webserver.py`: Web服务器，提供浏览器访问界面
- `scripts/cli.py`: 命令行工具，支持查询和导出报表
- `example/`: 演示示例和测试数据

## 依赖

- Python 3.8+
- 可选：`watchdog`（用于实时文件监控）

## License

MIT
