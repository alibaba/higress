# Log Collector API 接口文档

## 概述

Log Collector 是一个用于收集、存储和查询 HTTP 请求日志的服务。它提供了多种查询接口，支持传统查询和批量聚合查询，满足不同业务场景的需求。

## 基础信息

- **服务地址**: `http://localhost:8080`
- **Content-Type**: `application/json`
- **认证方式**: 无（测试环境）

## API 端点列表

### 1. 健康检查
```
GET /health
```

**响应示例**:
```json
ok
```

---

### 2. 日志收集
```
POST /ingest
```

**请求体**:
```json
{
    "start_time": "2026-02-09T10:30:00Z",
    "authority": "api-service.default.svc.cluster.local",
    "trace_id": "trace-0000000000000001",
    "method": "GET",
    "path": "/api/users",
    "protocol": "HTTP/1.1",
    "request_id": "req-abc123",
    "user_agent": "Mozilla/5.0",
    "x_forwarded_for": "192.168.1.1",
    "response_code": 200,
    "response_flags": "-",
    "response_code_details": "via_upstream",
    "bytes_received": 1024,
    "bytes_sent": 2048,
    "duration": 150,
    "upstream_cluster": "outbound|80||api-service.default.svc.cluster.local",
    "upstream_host": "10.0.0.1:8080",
    "upstream_service_time": "145",
    "upstream_transport_failure_reason": "",
    "upstream_local_address": "127.0.0.1:0",
    "downstream_local_address": "127.0.0.1:8080",
    "downstream_remote_address": "192.168.1.1:0",
    "route_name": "api-route",
    "requested_server_name": "api-service.default.svc.cluster.local",
    "istio_policy_status": "-",
    "ai_log": "{\"model\":\"qwen-turbo\",\"input_tokens\":100,\"output_tokens\":200}",
    "instance_id": "gw-instance-001",
    "api": "user-api",
    "model": "qwen-turbo",
    "consumer": "user-001",
    "route": "user-route",
    "service": "api-service.default.svc.cluster.local",
    "mcp_server": "mcp-server-1",
    "mcp_tool": "calculator",
    "input_tokens": 100,
    "output_tokens": 200,
    "total_tokens": 300
}
```

**响应**:
- Status: 200 OK

---

### 3. 传统日志查询
```
GET /query
```

#### 查询参数

| 参数 | 类型 | 必需 | 描述 | 示例 |
|------|------|------|------|------|
| start_time | string | 否 | 开始时间 | 2026-02-09 10:00:00 |
| start | string | 否 | 开始时间(兼容) | 2026-02-09 10:00:00 |
| end | string | 否 | 结束时间 | 2026-02-09 11:00:00 |
| authority | string | 否 | 服务域名 | api-service.default.svc.cluster.local |
| service | string | 否 | 服务名 | api-service.default.svc.cluster.local |
| method | string | 否 | HTTP方法 | GET |
| path | string | 否 | 请求路径 | /api/users |
| response_code | string | 否 | 响应状态码 | 200 |
| status | string | 否 | 状态码(兼容) | 200 |
| trace_id | string | 否 | Trace ID | trace-0000000000000001 |
| instance_id | string | 否 | 实例ID | gw-instance-001 |
| api | string | 否 | API名称 | user-api |
| model | string | 否 | 模型名称 | qwen-turbo |
| consumer | string | 否 | 消费者 | user-001 |
| route | string | 否 | 路由名称 | user-route |
| mcp_server | string | 否 | MCP服务器 | mcp-server-1 |
| mcp_tool | string | 否 | MCP工具 | calculator |
| page | integer | 否 | 页码(默认1) | 1 |
| page_size | integer | 否 | 每页大小(默认10,最大100) | 50 |
| sort_by | string | 否 | 排序字段 | start_time |
| sort_order | string | 否 | 排序方向(ASC/DESC) | DESC |

#### 响应格式
```json
{
    "total": 150,
    "logs": [
        {
            "start_time": "2026-02-09T10:30:00Z",
            "authority": "api-service.default.svc.cluster.local",
            "trace_id": "trace-0000000000000001",
            "method": "GET",
            "path": "/api/users",
            "response_code": 200,
            "duration": 150,
            "bytes_received": 1024,
            "bytes_sent": 2048,
            "model": "qwen-turbo",
            "consumer": "user-001",
            "input_tokens": 100,
            "output_tokens": 200,
            "total_tokens": 300
        }
    ],
    "status": "success"
}
```

---

### 4. 批量KPI查询
```
POST /batch/kpi
```

#### 请求体
```json
[
    {
        "timeRange": {
            "start": "2026-02-09 00:00:00",
            "end": "2026-02-10 00:00:00"
        },
        "bizType": "MODEL_API",
        "filters": {
            "model": "qwen-turbo",
            "consumer": "user-001"
        }
    },
    {
        "timeRange": {
            "start": "2026-02-09 00:00:00",
            "end": "2026-02-10 00:00:00"
        },
        "bizType": "MCP_SERVER",
        "filters": {
            "mcp_server": "mcp-server-1"
        }
    }
]
```

#### bizType 说明

**MODEL_API** 返回字段:
- `pv`: 页面访问量
- `uv`: 独立访客数
- `input_tokens`: 输入token总数
- `output_tokens`: 输出token总数
- `total_tokens`: 总token数
- `fallback_count`: Fallback请求数

**MCP_SERVER** 返回字段:
- `pv`: 页面访问量
- `uv`: 独立访客数
- `bytes_received`: 网关入流量
- `bytes_sent`: 网关出流量

#### 响应格式
```json
{
    "status": "success",
    "data": {
        "query_0": {
            "status": "success",
            "data": {
                "pv": 1250,
                "uv": 45,
                "input_tokens": 125000,
                "output_tokens": 250000,
                "total_tokens": 375000,
                "fallback_count": 5
            }
        },
        "query_1": {
            "status": "success",
            "data": {
                "pv": 2100,
                "uv": 78,
                "bytes_received": 1024000,
                "bytes_sent": 2048000
            }
        }
    }
}
```

---

### 5. 批量图表查询
```
POST /batch/chart
```

#### 请求体
```json
[
    {
        "timeRange": {
            "start": "2026-02-09 00:00:00",
            "end": "2026-02-10 00:00:00"
        },
        "interval": "60s",
        "scenario": "success_rate",
        "bizType": "MODEL_API",
        "filters": {}
    },
    {
        "timeRange": {
            "start": "2026-02-09 00:00:00",
            "end": "2026-02-10 00:00:00"
        },
        "interval": "60s",
        "scenario": "qps_total_simple",
        "bizType": "MCP_SERVER",
        "filters": {}
    }
]
```

#### scenario 支持列表

**MODEL_API 业务类型**:
- `success_rate`: 请求成功率(%)
- `token_rate`: Token消耗速率(tokens/s)
- `rt_distribution`: 响应时间分布
- `cache_hit_rate`: 缓存命中率
- `rate_limit`: 限流请求数/s

**MCP_SERVER 业务类型**:
- `success_rate`: 请求成功率(%)
- `qps_total_simple`: QPS统计
- `rt_distribution`: 响应时间分布

#### 响应格式
```json
{
    "status": "success",
    "data": {
        "query_0": {
            "status": "success",
            "data": {
                "timestamps": [1745123400000, 1745123460000, 1745123520000],
                "values": {
                    "success_rate": [98.5, 99.2, 97.8]
                }
            }
        },
        "query_1": {
            "status": "success",
            "data": {
                "timestamps": [1745123400000, 1745123460000],
                "values": {
                    "total_qps": [150.5, 162.3],
                    "stream_qps": [45.2, 51.7],
                    "request_qps": [105.3, 110.6]
                }
            }
        }
    }
}
```

---

### 6. 批量表格查询
```
POST /batch/table
```

#### 请求体
```json
[
    {
        "timeRange": {
            "start": "2026-02-09 00:00:00",
            "end": "2026-02-10 00:00:00"
        },
        "tableType": "model_token_stats",
        "bizType": "MODEL_API",
        "filters": {}
    },
    {
        "timeRange": {
            "start": "2026-02-09 00:00:00",
            "end": "2026-02-10 00:00:00"
        },
        "tableType": "method_distribution",
        "bizType": "MCP_SERVER",
        "filters": {}
    }
]
```

#### tableType 支持列表

**MODEL_API 业务类型**:
- `model_token_stats`: 模型token使用统计
- `consumer_token_stats`: 消费者token使用统计
- `service_token_stats`: 服务token使用统计
- `error_requests`: 错误请求统计
- `rate_limited_consumers`: 限流消费者统计
- `risk_types`: 风险类型统计
- `risk_consumers`: 风险消费者统计

**MCP_SERVER 业务类型**:
- `method_distribution`: HTTP方法分布
- `status_code_distribution`: 状态码分布

#### 响应格式
```json
{
    "status": "success",
    "data": {
        "query_0": {
            "status": "success",
            "data": {
                "data": [
                    {
                        "model": "qwen-turbo",
                        "request_count": 1250,
                        "input_tokens": 125000,
                        "output_tokens": 250000,
                        "total_tokens": 375000
                    },
                    {
                        "model": "gpt-3.5-turbo",
                        "request_count": 890,
                        "input_tokens": 89000,
                        "output_tokens": 178000,
                        "total_tokens": 267000
                    }
                ]
            }
        },
        "query_1": {
            "status": "success",
            "data": {
                "data": [
                    {
                        "method": "GET",
                        "request_count": 2100,
                        "avg_duration": 145.5
                    },
                    {
                        "method": "POST",
                        "request_count": 1350,
                        "avg_duration": 220.3
                    }
                ]
            }
        }
    }
}
```

## 错误响应格式

```json
{
    "status": "error",
    "error": "具体的错误信息",
    "total": 0,
    "logs": []
}
```

## 性能建议

1. **索引优化**: 对经常查询的字段建立数据库索引
2. **分页查询**: 大量数据查询时使用合理的分页大小
3. **时间范围**: 尽量使用较小的时间范围以提高查询效率
4. **批量查询**: 对于多个相似查询，使用批量API减少网络开销
5. **缓存策略**: 对于频繁查询的统计结果考虑实施缓存

## 测试脚本

项目提供了以下测试脚本:

- `setup_test_data.sh`: 准备测试数据
- `enhanced_test_time_range.sh`: 时间范围查询测试
- `enhanced_benchmark_query.sh`: 性能基准测试

使用方法:
```bash
# 准备测试数据
./setup_test_data.sh

# 运行时间范围测试
./enhanced_test_time_range.sh

# 运行性能测试
./enhanced_benchmark_query.sh
```