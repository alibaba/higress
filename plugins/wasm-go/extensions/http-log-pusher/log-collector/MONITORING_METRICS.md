# Higress网关监控指标说明

## 概述
本文档详细说明了Higress网关日志收集器支持的各项监控指标及其对应的curl查询方法。

## 基础配置
```bash
# 日志收集器地址
COLLECTOR_URL="http://localhost:8080"

# 时间范围格式
TIME_RANGE_START="2026-02-11T00:00:00"
TIME_RANGE_END="2026-02-11T01:00:00"
```

## 核心监控指标

### 1. 网关流量监控

#### 1.1 入关流量 (Inbound Traffic)
**指标说明**: 监控进入网关的请求数量和速率

```bash
curl -X POST "$COLLECTOR_URL/batch/chart" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "qps_total_simple",
    "bizType": "MODEL_API"
  }'
```

**返回字段**:
- `total_qps`: 总请求速率 (requests/second)
- `stream_qps`: 流式请求速率 (包含"stream"路径的请求)
- `request_qps`: 普通请求速率 (非流式请求)

#### 1.2 出关流量 (Outbound Traffic)
**指标说明**: 监控从网关发出的流量大小

```bash
# 需要扩展支持，基于bytes_sent字段计算
curl -X POST "$COLLECTOR_URL/batch/chart" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "traffic_outbound",
    "bizType": "MODEL_API"
  }'
```

### 2. 性能指标

#### 2.1 请求响应时间分布 (RT Distribution)
**指标说明**: 监控请求的响应时间分布情况

```bash
curl -X POST "$COLLECTOR_URL/batch/chart" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "rt_distribution",
    "bizType": "MODEL_API"
  }'
```

**返回字段**:
- `avg_rt`: 平均响应时间 (毫秒)
- `p99_rt`: 99%分位响应时间
- `p95_rt`: 95%分位响应时间  
- `p90_rt`: 90%分位响应时间
- `p50_rt`: 50%分位响应时间

#### 2.2 成功率 (Success Rate)
**指标说明**: 监控请求的成功率百分比

```bash
curl -X POST "$COLLECTOR_URL/batch/chart" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "success_rate",
    "bizType": "MODEL_API"
  }'
```

**计算逻辑**: `成功请求数(响应码<400) / 总请求数 × 100%`

### 3. 请求分布统计

#### 3.1 状态码分布 (Status Code Distribution)
**指标说明**: 统计不同HTTP状态码的请求分布

```bash
curl -X POST "$COLLECTOR_URL/batch/table" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "status_code_distribution",
    "bizType": "MODEL_API"
  }'
```

**典型状态码**:
- 2xx: 成功响应
- 4xx: 客户端错误
- 5xx: 服务器错误
- 429: 限流响应

#### 3.2 请求方法分布 (Method Distribution)
**指标说明**: 统计不同HTTP方法的请求分布

```bash
curl -X POST "$COLLECTOR_URL/batch/table" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "method_distribution",
    "bizType": "MODEL_API"
  }'
```

**常见方法**: GET, POST, PUT, DELETE等

#### 3.3 API接口分布 (API Distribution)
**指标说明**: 统计不同API接口的调用情况

```bash
curl -X POST "$COLLECTOR_URL/batch/table" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "api_distribution",
    "bizType": "MODEL_API"
  }'
```

### 4. AI相关指标

#### 4.1 Token消耗速率 (Token Rate)
**指标说明**: 监控AI模型的Token消耗情况

```bash
curl -X POST "$COLLECTOR_URL/batch/chart" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "token_rate",
    "bizType": "MODEL_API"
  }'
```

**返回字段**:
- `input_token_rate`: 输入token消耗速率
- `output_token_rate`: 输出token消耗速率
- `total_token_rate`: 总token消耗速率

#### 4.2 缓存命中率 (Cache Hit Rate)
**指标说明**: 监控缓存的命中情况

```bash
curl -X POST "$COLLECTOR_URL/batch/chart" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "cache_hit_rate",
    "bizType": "MODEL_API"
  }'
```

### 5. 限流监控

#### 5.1 限流请求数 (Rate Limit Count)
**指标说明**: 监控触发限流的请求数量

```bash
curl -X POST "$COLLECTOR_URL/batch/chart" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "interval": "60s",
    "scenario": "rate_limit",
    "bizType": "MODEL_API"
  }'
```

**监控重点**: 429状态码的请求速率

### 6. 基础统计指标

#### 6.1 PV/UV统计 (Page View / Unique Visitor)
**指标说明**: 网站访问量和独立访客统计

```bash
curl -X POST "$COLLECTOR_URL/batch/kpi" \
  -H "Content-Type: application/json" \
  -d '{
    "timeRange": {
      "start": "'$TIME_RANGE_START'",
      "end": "'$TIME_RANGE_END'"
    },
    "bizType": "MODEL_API"
  }'
```

**返回字段**:
- `pv`: 页面访问量 (Page View)
- `uv`: 独立访客数 (Unique Visitor) - 基于trace_id去重
- `bytes_received`: 接收字节数
- `bytes_sent`: 发送字节数
- `input_tokens`: 输入token总数
- `output_tokens`: 输出token总数
- `total_tokens`: 总token数

## 时间粒度支持

支持多种时间聚合粒度：
- `1s`: 1秒粒度
- `15s`: 15秒粒度  
- `30s`: 30秒粒度
- `60s`: 60秒粒度（默认）
- `300s`: 5分钟粒度
- `1800s`: 30分钟粒度
- `3600s`: 1小时粒度

## 业务类型分类

- `MODEL_API`: AI模型API请求
- `MCP_SERVER`: MCP服务器请求

## 返回数据格式

### 图表数据格式 (Chart)
```json
{
  "status": "success",
  "data": {
    "timestamps": [1739232000000, 1739232060000, ...],
    "values": {
      "metric_name": [10.5, 12.3, ...]
    }
  }
}
```

### 表格数据格式 (Table)
```json
{
  "status": "success", 
  "data": {
    "rows": [
      {"dimension": "value1", "count": 100},
      {"dimension": "value2", "count": 80}
    ]
  }
}
```

### KPI数据格式
```json
{
  "status": "success",
  "data": {
    "pv": 1000,
    "uv": 500,
    "bytes_received": 1024000,
    "bytes_sent": 2048000
  }
}
```

## 错误处理

常见错误响应：
```json
{
  "status": "error",
  "error": "具体错误信息"
}
```

## 使用建议

1. **生产环境**: 建议使用60s以上的聚合粒度
2. **实时监控**: 可使用15s粒度获取近实时数据
3. **历史分析**: 可使用小时级粒度进行趋势分析
4. **告警设置**: 基于成功率、RT等关键指标设置阈值告警