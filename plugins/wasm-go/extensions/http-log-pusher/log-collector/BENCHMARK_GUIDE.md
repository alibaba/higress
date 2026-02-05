# Log Collector 性能测试指南

## 📋 概述

本目录包含两个性能测试脚本，用于验证 log-collector 服务的 batch 写入和 query 查询性能。

## 🛠️ 测试环境准备

### 1. 启动 MySQL 数据库

```bash
# 使用 Docker 启动 MySQL
docker run -d \
  --name mysql-test \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=higress_poc \
  -p 3306:3306 \
  mysql:8.0

# 等待 MySQL 启动完成
sleep 10

# 创建表结构
mysql -h127.0.0.1 -uroot -proot higress_poc < schema.sql
```

### 2. 启动 log-collector 服务

```bash
# 设置环境变量
export MYSQL_DSN="root:root@tcp(127.0.0.1:3306)/higress_poc?charset=utf8mb4&parseTime=True"

# 启动服务
go run main.go
```

### 3. 验证服务状态

```bash
curl http://localhost:8080/health
# 预期输出: ok
```

## 🚀 测试脚本使用

### Batch 性能测试

测试 `/ingest` 接口的写入性能和 batch 逻辑。

```bash
# 赋予执行权限
chmod +x benchmark_batch.sh

# 运行测试（使用默认URL）
./benchmark_batch.sh

# 指定自定义URL
COLLECTOR_URL=http://localhost:8080 ./benchmark_batch.sh
```

**测试场景包括：**

1. **批次大小测试** - 测试不同批次大小的性能（1, 10, 25, 50, 100, 200条）
2. **并发写入测试** - 测试并发级别（1, 5, 10, 20线程）
3. **吞吐量压测** - 持续30秒的高负载测试
4. **边界条件测试** - 空数据、超长字段、快速/慢速发送
5. **状态码分布测试** - 不同HTTP状态码的日志写入

### Query 性能测试

测试 `/query` 接口的查询性能和不同查询条件的效率。

```bash
# 赋予执行权限
chmod +x benchmark_query.sh

# 安装 jq（如果未安装）
# macOS: brew install jq
# Ubuntu: sudo apt-get install jq

# 运行测试
./benchmark_query.sh

# 指定自定义URL
COLLECTOR_URL=http://localhost:8080 ./benchmark_query.sh
```

**测试场景包括：**

1. **全表扫描** - 无条件查询，不同页面大小
2. **索引字段查询** - trace_id, start_time, response_code, authority
3. **非索引字段查询** - path, method 等
4. **分页性能** - 不同页面大小和页码
5. **排序性能** - 按不同字段排序
6. **并发查询** - 多线程并发查询（1, 5, 10, 20线程）
7. **复杂查询** - 多条件组合查询
8. **压力测试** - 持续30秒的高并发查询
9. **边界条件** - 无效参数、不存在数据、特殊字符

## 📊 测试报告

测试报告将保存在 `./benchmark_reports/` 目录：

```
benchmark_reports/
├── batch_benchmark_20260204_143020.txt   # Batch 测试报告
└── query_benchmark_20260204_143521.txt   # Query 测试报告
```

## 🔍 如何分析结果

### Batch 测试关键指标

从 log-collector 日志中查看：

```bash
# 查看 Batch 相关日志
docker logs -f <log-collector-container> | grep "\[Batch\]"
```

**关注指标：**

1. **触发方式统计**
   - `Trigger flush by count` - 条数触发（达到50条）
   - `Trigger flush by timer` - 定时触发（每1秒）

2. **Flush 性能**
   - 批次大小：实际写入的日志条数
   - 总耗时：从开始到完成的时间
   - 平均耗时：每条日志的平均处理时间

3. **TPS（Transactions Per Second）**
   - 单线程 TPS：反映基础性能
   - 并发 TPS：反映扩展能力

**示例日志：**
```
[Batch] Starting background flush goroutine, interval=1s, threshold=50 logs
[Batch] Trigger flush by count: buffer=50/50
[Batch] Start flushing 50 logs to MySQL
[Batch] ✓ SUCCESS flushed 50 logs to MySQL (duration=45ms, avg=0.9ms/log)
```

### Query 测试关键指标

从 log-collector 日志中查看：

```bash
# 查看 Query 相关日志
docker logs -f <log-collector-container> | grep "\[Query\]"
```

**关注指标：**

1. **查询阶段耗时分解**
   - COUNT 耗时：统计总记录数
   - SELECT 耗时：执行主查询
   - Scan 耗时：解析结果集
   - 总耗时：完整请求时间

2. **查询条件分析**
   - 使用的过滤条件
   - 分页参数
   - 排序字段

3. **结果统计**
   - 匹配总数（total）
   - 返回条数（returned）
   - 平均扫描速度（avg/row）

**示例日志：**
```
[Query] Request received: status=200&page_size=50&sort_by=duration&sort_order=DESC
[Query] Filters applied: [status=200]
[Query] COUNT result: total=1250 (duration=12ms)
[Query] Pagination: page=1, page_size=50, offset=0
[Query] Sorting: sort_by=duration, sort_order=DESC
[Query] SELECT executed (duration=23ms)
[Query] Rows scanned: count=50 (duration=8ms, avg=160µs/row)
[Query] ✓ SUCCESS: returned=50/1250 logs (total_duration=45ms, count=12ms, query=23ms, scan=8ms)
```

## 📈 性能基准参考

### Batch 写入性能

| 场景 | 预期 TPS | 说明 |
|------|----------|------|
| 单线程批次写入 | 500-1000 | 基础性能 |
| 5线程并发 | 2000-3000 | 良好扩展 |
| 10线程并发 | 3000-5000 | 接近数据库瓶颈 |
| 持续压测 | 2000-4000 | 稳定吞吐量 |

### Query 查询性能

| 场景 | 预期响应时间 | 说明 |
|------|--------------|------|
| 索引字段精确查询 | < 20ms | 使用索引 |
| 时间范围查询 | < 50ms | 索引扫描 |
| 模糊查询（path LIKE） | 50-200ms | 全表扫描 |
| 复杂多条件查询 | 50-150ms | 取决于索引 |
| 并发查询（10线程） | QPS > 100 | 数据库连接池 |

## 🎯 优化建议

### 1. Batch 优化

**如果 flush 延迟高：**
- 减小 `flushSize`（当前50）提高实时性
- 减小定时器间隔（当前1秒）

**如果吞吐量不足：**
- 增大 `flushSize` 提高批量效率
- 增加数据库连接池大小
- 考虑异步写入或消息队列

### 2. Query 优化

**如果查询慢：**
- 添加索引（特别是 start_time, trace_id, response_code, authority）
- 限制查询时间范围
- 使用分页避免大结果集

**建议索引：**
```sql
CREATE INDEX idx_start_time ON access_logs(start_time);
CREATE INDEX idx_trace_id ON access_logs(trace_id);
CREATE INDEX idx_response_code ON access_logs(response_code);
CREATE INDEX idx_authority ON access_logs(authority);
CREATE INDEX idx_method ON access_logs(method);
CREATE INDEX idx_composite ON access_logs(start_time, response_code, method);
```

## 🧪 其他测试场景

### 3. matchRules 匹配验证测试

验证 WasmPlugin 的 matchRules 配置是否正确生效。

**重要说明：**
- ✅ **wrapper 已处理 matchRules 过滤逻辑**，插件代码无需关心
- ✅ 插件只会在匹配的请求上被调用
- ❌ matchRules 必须与 pluginConfig 同级，不可嵌套在 config 内部
- ❌ 必须包含 ingress 名称 + 至少一个匹配条件（host/path/method）

```bash
# 使用专门的验证脚本
cd /Users/terry/work/higress/plugins/wasm-go/extensions/http-log-pusher
chmod +x verify_matchrules.sh

# 配置环境变量
export GATEWAY_URL="http://your-gateway-ip"
export GATEWAY_PORT="80"
export COLLECTOR_URL="http://log-collector-ip:8080"

# 运行测试
./verify_matchrules.sh
```

**测试流程：**

1. **准备 WasmPlugin 配置**
   ```yaml
   apiVersion: extensions.higress.io/v1alpha1
   kind: WasmPlugin
   metadata:
     name: http-log-pusher
     namespace: higress-system
   spec:
     matchRules:  # 与 pluginConfig 同级
     - ingress:
       - my-test-ingress
       config:
         hosts:
         - "api.example.com"
         paths:
         - "/api/v1/*"
         methods:
         - POST
     pluginConfig:  # 与 matchRules 同级
       collector_service_name: "log-collector.higress-system.svc.cluster.local"
       collector_host: "log-collector.higress-system.svc.cluster.local"
       collector_port: 8080
       collector_path: "/ingest"
   ```

2. **发送测试请求**
   ```bash
   # 应该匹配的请求
   curl -X POST http://gateway-ip/api/v1/users \
     -H "Host: api.example.com" \
     -H "X-B3-TraceID: test-trace-001" \
     -d '{"name":"test"}'
   
   # 不应该匹配的请求（host 不匹配）
   curl -X POST http://gateway-ip/api/v1/users \
     -H "Host: other.example.com" \
     -H "X-B3-TraceID: test-trace-002" \
     -d '{"name":"test"}'
   ```

3. **验证日志采集**
   ```bash
   # 查询 trace-001（应该被采集）
   curl "http://log-collector:8080/query?trace_id=test-trace-001" | jq '.total'
   # 预期输出: 1
   
   # 查询 trace-002（不应该被采集）
   curl "http://log-collector:8080/query?trace_id=test-trace-002" | jq '.total'
   # 预期输出: 0
   ```

**测试场景：**

| 场景 | 配置 | 测试请求 | 预期结果 |
|------|------|----------|----------|
| Ingress 匹配 | `ingress: [my-test-ingress]` + `hosts: ["*"]` | Host: my-test-ingress.example.com | 采集 |
| Host 精确匹配 | `hosts: ["api.example.com"]` | Host: api.example.com | 采集 |
| Host 通配符 | `hosts: ["*.test.com"]` | Host: app1.test.com | 采集 |
| Path 前缀匹配 | `paths: ["/api/v1/*"]` | Path: /api/v1/users | 采集 |
| Path 精确匹配 | `paths: ["/admin"]` | Path: /admin | 采集 |
| Method 匹配 | `methods: [POST, PUT]` | Method: POST | 采集 |
| 组合条件 | host + path + method | 全部匹配时 | 采集 |
| 多规则 OR | 两个 matchRules | 任一匹配时 | 采集 |

**常见配置错误：**

❌ **错误1**: matchRules 嵌套位置错误
```yaml
spec:
  matchRules:
  - config:
      matchRules:  # ❌ 不能嵌套在这里
        hosts: ["*.example.com"]
```

❌ **错误2**: 仅指定 ingress，没有匹配条件
```yaml
spec:
  matchRules:
  - ingress: [my-ingress]  # ❌ 缺少 config 和匹配条件
```
错误信息: `invalid match rule has no match condition`

✅ **正确配置**:
```yaml
spec:
  matchRules:  # 与 pluginConfig 同级
  - ingress: [my-ingress]
    config:    # 包含至少一个匹配条件
      hosts: ["*.example.com"]
  pluginConfig:  # 与 matchRules 同级
    # ...
```

### 4. 数据一致性测试

验证数据不丢失：

```bash
# 1. 发送1000条已知日志
for i in {1..1000}; do
  curl -X POST http://localhost:8080/ingest \
    -H "Content-Type: application/json" \
    -d "{\"trace_id\":\"test-$i\", ...}"
done

# 2. 等待flush完成
sleep 5

# 3. 查询验证
curl "http://localhost:8080/query?page_size=1000" | jq '.total'
# 预期输出: 至少包含1000条
```

### 5. 故障恢复测试

测试数据库故障场景：

```bash
# 1. 发送日志
./benchmark_batch.sh &

# 2. 中途停止 MySQL
docker stop mysql-test

# 3. 观察 log-collector 日志（应看到失败日志）

# 4. 重启 MySQL
docker start mysql-test

# 5. 继续发送日志验证恢复
```

### 6. 内存泄漏测试

长时间运行监控：

```bash
# 监控 log-collector 内存使用
watch -n 5 'ps aux | grep main'

# 持续压测1小时
timeout 3600 ./benchmark_batch.sh
```

## 📝 注意事项

1. **测试前清空数据库**
   ```sql
   TRUNCATE TABLE access_logs;
   ```

2. **生产环境测试**
   - 在非高峰期进行
   - 使用只读副本测试查询性能
   - 逐步增加压力，避免影响业务

3. **网络延迟**
   - 脚本和服务在同一机器时延迟最小
   - 跨网络测试需考虑网络开销

4. **资源限制**
   - MySQL 配置影响性能（innodb_buffer_pool_size等）
   - log-collector 的连接池大小（MaxOpenConns=10）
   - 系统文件描述符限制

## 🔧 故障排查

**服务无法连接**
```bash
# 检查服务是否运行
curl http://localhost:8080/health

# 检查端口占用
lsof -i :8080
```

**MySQL 连接失败**
```bash
# 检查 MySQL 状态
mysql -h127.0.0.1 -uroot -proot -e "SELECT 1"

# 检查数据库和表
mysql -h127.0.0.1 -uroot -proot higress_poc -e "SHOW TABLES"
```

**脚本权限问题**
```bash
chmod +x benchmark_*.sh
```

**jq 未安装**
```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# CentOS/RHEL
sudo yum install jq
```

## 📚 参考资料

- [main.go](./main.go) - log-collector 源码
- [MySQL Performance Tuning](https://dev.mysql.com/doc/refman/8.0/en/optimization.html)
- [Go Database/SQL Tutorial](https://go.dev/doc/database/sql-prepared-statements)
