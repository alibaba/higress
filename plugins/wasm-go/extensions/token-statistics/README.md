
创建WasmPlugin资源配置：

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: token-statistics
  namespace: higress-system
spec:
  url: file:///opt/plugins/token-statistics/main.wasm
  phase: UNSPECIFIED_PHASE
  priority: 200
  defaultConfig:
    dimensions:
      - name: "model"
        value_source: "response_body"
        value: "model"
        apply_to_metric: true
        apply_to_log: true
      - name: "consumer"
        value_source: "request_header"
        value: "x-mse-consumer"
        apply_to_metric: true
        apply_to_log: true
    exporters:
      - type: "prometheus"
        config:
          namespace: "ai"
          subsystem: "token"
      - type: "log"
        config:
          level: "info"
    enable_path_suffixes:
      - "/v1/chat/completions"
      - "/v1/completions"
    enable_content_types:
      - "application/json"
      - "text/event-stream"
  defaultConfigDisable: false
```

### 4.6.3 功能验证

#### 验证指标输出

部署后可以通过Prometheus查询相关指标：

```promql
# 查询每分钟Token消耗量
rate(ai_token_input_tokens_total[1m])

# 按模型统计Token消耗
sum by (model) (ai_token_input_tokens_total)

# 查询平均响应Token数
avg(ai_token_output_tokens_total)
```

#### 验证日志输出

检查Higress网关日志中是否包含Token统计信息：

```json
{
  "timestamp": "2025-12-17T10:00:00Z",
  "level": "INFO",
  "message": "Token usage statistics",
  "model": "gpt-3.5-turbo",
  "input_tokens": 15,
  "output_tokens": 42,
  "total_tokens": 57,
  "consumer": "test-user"
}
```

#### 验证HTTP导出

如果配置了HTTP导出器，可以检查目标接收端是否收到统计数据。