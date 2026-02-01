# 日志轮转支持

Agent Session Monitor 现在完全支持日志轮转（log rotation），能够正确处理被 logrotate 轮转的日志文件。

## 工作原理

### 日志轮转场景

典型的 logrotate 配置：

```
/var/log/proxy/access.log {
    su 1337 1337
    rotate 5
    create 644 1337 1337
    nocompress
    notifempty
    minsize 100M
    postrotate
        ps aux|grep "envoy -c"|grep -v "grep"|awk '{print $2}'|xargs -i kill -SIGUSR1 {}
    endscript
}
```

轮转过程：
```
access.log       -> access.log.1
access.log.1     -> access.log.2
access.log.2     -> access.log.3
...
access.log.5     -> 删除
（创建新的 access.log）
```

### 增量解析策略

Agent Session Monitor 使用以下策略实现增量解析：

1. **文件唯一标识**：使用 `inode` 而不是文件名
   - Linux 中 `mv` 不会改变文件的 inode
   - 即使文件被重命名（access.log -> access.log.1），仍然可以追踪

2. **状态持久化**：记录每个文件的读取offset
   ```json
   {
     "4057726": 1631,   // inode: offset
     "4057730": 815,
     "4057731": 489
   }
   ```

3. **Session数据持久化**：每次启动时加载已有的session数据
   - 累积多次运行的统计
   - 避免重复计算

4. **轮转文件扫描**：从旧到新按顺序解析
   ```
   access.log.5 (最旧)
   access.log.4
   access.log.3
   access.log.2
   access.log.1
   access.log (最新)
   ```

## 使用方法

### 基本用法

```bash
python3 main.py \
    --log-path /var/log/proxy/access.log \
    --output-dir ./sessions \
    --max-rotate 5
```

参数说明：
- `--log-path`: 当前日志文件路径
- `--output-dir`: Session数据和状态文件存储目录
- `--max-rotate`: 最大轮转文件数量（默认5）

### 状态文件

默认状态文件：`<output-dir>/.state.json`

自定义状态文件：
```bash
python3 main.py \
    --log-path /var/log/proxy/access.log \
    --output-dir ./sessions \
    --state-file /var/lib/monitor/state.json
```

状态文件记录了每个inode的读取offset，避免重复解析。

## 定时任务配置

### 使用 cron

每分钟运行一次（增量解析）：

```bash
* * * * * /usr/bin/python3 /path/to/main.py \
    --log-path /var/log/proxy/access.log \
    --output-dir /var/lib/sessions \
    --max-rotate 5 >> /var/log/monitor.log 2>&1
```

### 使用 systemd timer

创建服务文件 `/etc/systemd/system/session-monitor.service`：

```ini
[Unit]
Description=Agent Session Monitor
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/bin/python3 /path/to/main.py \
    --log-path /var/log/proxy/access.log \
    --output-dir /var/lib/sessions \
    --max-rotate 5

[Install]
WantedBy=multi-user.target
```

创建 timer `/etc/systemd/system/session-monitor.timer`：

```ini
[Unit]
Description=Run session monitor every minute

[Timer]
OnBootSec=1min
OnUnitActiveSec=1min

[Install]
WantedBy=timers.target
```

启用：
```bash
systemctl daemon-reload
systemctl enable session-monitor.timer
systemctl start session-monitor.timer
```

## 测试

运行测试脚本验证日志轮转功能：

```bash
cd example
bash test_rotation.sh
```

测试场景：
1. 首次解析：处理10条记录
2. 轮转后解析：只处理新增的5条（总计15条）
3. 再次轮转：只处理新增的3条（总计18条）

验证要点：
- ✅ 不重复解析已处理的日志
- ✅ 正确追踪轮转的文件
- ✅ Session数据累积正确
- ✅ 状态文件记录正确

## 工作流程示例

### 第一次运行（t1）

```
日志文件：
  access.log (100 lines)

执行：
  python3 main.py --log-path access.log

结果：
  - 处理100行
  - session_001: 100条消息
  - state.json: {"12345": 10000}  # inode:offset
```

### logrotate 轮转（t2）

```
日志文件：
  access.log.1 (100 lines, inode=12345)
  access.log (新创建, 0 lines, inode=12346)
```

### 第二次运行（t3）

```
新增日志：
  access.log.1 仍然是100行
  access.log 新增了50行

执行：
  python3 main.py --log-path access.log

结果：
  - access.log.1: 跳过（offset未变）
  - access.log: 处理50行
  - session_001: 150条消息（累积）
  - state.json: {"12345": 10000, "12346": 5000}
```

### 再次轮转（t4）

```
日志文件：
  access.log.2 (100 lines, inode=12345)
  access.log.1 (50 lines, inode=12346)
  access.log (新创建, 0 lines, inode=12347)
```

### 第三次运行（t5）

```
新增日志：
  access.log.2: 0行（已全部处理）
  access.log.1: 0行（已全部处理）
  access.log: 新增30行

执行：
  python3 main.py --log-path access.log

结果：
  - access.log.2: 跳过
  - access.log.1: 跳过
  - access.log: 处理30行
  - session_001: 180条消息（累积）
  - state.json: {"12345": 10000, "12346": 5000, "12347": 3000}
```

## 注意事项

### 1. 状态文件清理

状态文件会记录所有处理过的inode。对于已删除的旧日志文件（如access.log.6+），其inode不会自动清理。

定期清理状态文件：
```bash
# 保留最近1000个inode
python3 -c "
import json
with open('.state.json') as f:
    state = json.load(f)
# 保留最新的1000个
new_state = dict(list(state.items())[-1000:])
with open('.state.json', 'w') as f:
    json.dump(new_state, f)
"
```

### 2. 文件删除

如果轮转的文件被删除（如access.log.6被删除），不影响功能。程序会自动跳过不存在的文件。

### 3. 并发运行

避免并发运行多个实例处理同一组日志，可能导致：
- 状态文件冲突
- Session数据不一致

使用文件锁或 systemd 的 oneshot 类型避免并发。

### 4. 磁盘空间

Session数据文件会持续增长。定期归档或清理旧的session数据：

```bash
# 归档30天前的session
find /var/lib/sessions -name "*.json" -mtime +30 -exec gzip {} \;

# 删除90天前的归档
find /var/lib/sessions -name "*.json.gz" -mtime +90 -delete
```

## 性能

### 大文件处理

日志文件很大时（如几GB），首次解析可能需要较长时间。优化建议：

1. **首次解析离线进行**：
   ```bash
   # 只处理最新的1个轮转文件
   python3 main.py --log-path access.log --max-rotate 1
   ```

2. **分批处理**：
   ```bash
   # 先处理旧文件
   python3 main.py --log-path access.log.5
   python3 main.py --log-path access.log.4
   # ...最后处理最新文件
   python3 main.py --log-path access.log
   ```

### 内存使用

内存使用与活跃session数量成正比。对于大量session：

- 考虑定期归档inactive session
- 使用`--session-key`过滤只关注的session

## 故障排查

### 重复计数

症状：Token统计翻倍或多倍

原因：状态文件丢失或损坏

解决：
```bash
# 检查状态文件
cat .state.json

# 如果损坏，删除并重新从头解析
rm .state.json
python3 main.py --log-path access.log
```

### 遗漏数据

症状：某些日志未被处理

原因：
1. 日志文件权限问题
2. max-rotate设置过小

解决：
```bash
# 检查文件权限
ls -l /var/log/proxy/access.log*

# 增加max-rotate
python3 main.py --log-path access.log --max-rotate 10
```

### 状态文件过大

症状：.state.json 文件很大（>1MB）

原因：记录了太多历史inode

解决：定期清理旧的inode记录（见"注意事项"）

## 总结

Agent Session Monitor 的日志轮转支持特性：

✅ **自动追踪**：无需手动指定轮转文件
✅ **增量解析**：只处理新增内容，不重复
✅ **状态持久化**：跨运行保持解析进度
✅ **Session累积**：正确累积多次运行的统计
✅ **高性能**：inode-based追踪，避免全量扫描

适用于生产环境的持续监控场景！
