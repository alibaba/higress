#!/bin/bash

# matchRules 验证脚本
# 验证不同 WasmPlugin 配置下的 matchRules 是否正确生效
# 测试场景：
# 1. 匹配特定 Ingress
# 2. 匹配特定 host
# 3. 匹配特定 path
# 4. 匹配特定 method
# 5. 组合匹配条件
# 6. 多个 matchRules

set -e

# 配置
GATEWAY_URL="${GATEWAY_URL:-http://localhost}"
GATEWAY_PORT="${GATEWAY_PORT:-80}"
REPORT_DIR="./benchmark_reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$REPORT_DIR/matchrules_test_${TIMESTAMP}.txt"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 创建报告目录
mkdir -p "$REPORT_DIR"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$REPORT_FILE"
}

log_success() {
    echo -e "${GREEN}[✓ PASS]${NC} $1" | tee -a "$REPORT_FILE"
}

log_fail() {
    echo -e "${RED}[✗ FAIL]${NC} $1" | tee -a "$REPORT_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$REPORT_FILE"
}

# 分隔线
print_separator() {
    echo "========================================" | tee -a "$REPORT_FILE"
}

# 发送测试请求
send_request() {
    local host="$1"
    local path="$2"
    local method="${3:-GET}"
    local extra_headers="$4"
    
    local url="${GATEWAY_URL}:${GATEWAY_PORT}${path}"
    
    # 构建 curl 命令
    local cmd="curl -s -w '\\n%{http_code}' -X $method"
    cmd="$cmd -H 'Host: $host'"
    
    if [ -n "$extra_headers" ]; then
        cmd="$cmd $extra_headers"
    fi
    
    cmd="$cmd '$url'"
    
    # 执行请求
    local response=$(eval $cmd)
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n-1)
    
    echo "$http_code"
}

# 检查日志是否被采集
check_log_collected() {
    local trace_id="$1"
    local collector_url="${COLLECTOR_URL:-http://localhost:8080}"
    
    # 等待日志被 flush
    sleep 2
    
    # 查询 collector
    local result=$(curl -s "${collector_url}/query?trace_id=${trace_id}" | jq -r '.total // 0')
    echo "$result"
}

# 测试场景1: 匹配特定 Ingress
test_ingress_match() {
    print_separator
    log_info "测试场景1: Ingress 匹配"
    print_separator
    
    cat <<'EOF' | tee -a "$REPORT_FILE"
WasmPlugin 配置示例:
```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: http-log-pusher
  namespace: higress-system
spec:
  matchRules:
  - ingress:
    - my-test-ingress
  pluginConfig:
    collector_service_name: "log-collector.higress-system.svc.cluster.local"
    collector_host: "log-collector.higress-system.svc.cluster.local"
    collector_port: 8080
    collector_path: "/ingest"
```
EOF
    
    log_info "1.1 发送请求到匹配的 Ingress"
    local trace_id="match-ingress-$(date +%s)"
    send_request "my-test-ingress.example.com" "/api/test" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "Ingress 匹配成功: 日志已采集 (trace_id=$trace_id, count=$count)"
    else
        log_fail "Ingress 匹配失败: 日志未采集 (trace_id=$trace_id)"
    fi
    
    log_info "1.2 发送请求到不匹配的 Ingress"
    local trace_id="no-match-ingress-$(date +%s)"
    send_request "other-ingress.example.com" "/api/test" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -eq 0 ]; then
        log_success "Ingress 不匹配验证成功: 日志未采集 (trace_id=$trace_id)"
    else
        log_fail "Ingress 不匹配验证失败: 日志被错误采集 (trace_id=$trace_id, count=$count)"
    fi
}

# 测试场景2: Host 匹配
test_host_match() {
    print_separator
    log_info "测试场景2: Host 匹配"
    print_separator
    
    cat <<'EOF' | tee -a "$REPORT_FILE"
WasmPlugin 配置示例:
```yaml
spec:
  matchRules:
  - ingress:
    - my-test-ingress
    config:
      hosts:
      - "api.example.com"
      - "*.test.com"
  pluginConfig:
    # ...
```
EOF
    
    log_info "2.1 精确 host 匹配"
    local trace_id="match-host-exact-$(date +%s)"
    send_request "api.example.com" "/api/test" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "精确 host 匹配成功 (api.example.com)"
    else
        log_fail "精确 host 匹配失败 (api.example.com)"
    fi
    
    log_info "2.2 通配符 host 匹配"
    local trace_id="match-host-wildcard-$(date +%s)"
    send_request "app1.test.com" "/api/test" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "通配符 host 匹配成功 (*.test.com)"
    else
        log_fail "通配符 host 匹配失败 (*.test.com)"
    fi
    
    log_info "2.3 不匹配的 host"
    local trace_id="no-match-host-$(date +%s)"
    send_request "other.example.org" "/api/test" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -eq 0 ]; then
        log_success "不匹配 host 验证成功"
    else
        log_fail "不匹配 host 验证失败: 日志被错误采集"
    fi
}

# 测试场景3: Path 匹配
test_path_match() {
    print_separator
    log_info "测试场景3: Path 匹配"
    print_separator
    
    cat <<'EOF' | tee -a "$REPORT_FILE"
WasmPlugin 配置示例:
```yaml
spec:
  matchRules:
  - ingress:
    - my-test-ingress
    config:
      paths:
      - "/api/v1/*"
      - "/admin"
  pluginConfig:
    # ...
```
EOF
    
    log_info "3.1 前缀匹配 (/api/v1/*)"
    local trace_id="match-path-prefix-$(date +%s)"
    send_request "api.example.com" "/api/v1/users" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "前缀 path 匹配成功 (/api/v1/users)"
    else
        log_fail "前缀 path 匹配失败 (/api/v1/users)"
    fi
    
    log_info "3.2 精确匹配 (/admin)"
    local trace_id="match-path-exact-$(date +%s)"
    send_request "api.example.com" "/admin" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "精确 path 匹配成功 (/admin)"
    else
        log_fail "精确 path 匹配失败 (/admin)"
    fi
    
    log_info "3.3 不匹配的 path"
    local trace_id="no-match-path-$(date +%s)"
    send_request "api.example.com" "/public/index.html" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -eq 0 ]; then
        log_success "不匹配 path 验证成功"
    else
        log_fail "不匹配 path 验证失败: 日志被错误采集"
    fi
}

# 测试场景4: Method 匹配
test_method_match() {
    print_separator
    log_info "测试场景4: Method 匹配"
    print_separator
    
    cat <<'EOF' | tee -a "$REPORT_FILE"
WasmPlugin 配置示例:
```yaml
spec:
  matchRules:
  - ingress:
    - my-test-ingress
    config:
      methods:
      - POST
      - PUT
  pluginConfig:
    # ...
```
EOF
    
    log_info "4.1 POST 方法匹配"
    local trace_id="match-method-post-$(date +%s)"
    send_request "api.example.com" "/api/users" "POST" "-H 'X-B3-TraceID: $trace_id' -d '{\"name\":\"test\"}'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "POST 方法匹配成功"
    else
        log_fail "POST 方法匹配失败"
    fi
    
    log_info "4.2 PUT 方法匹配"
    local trace_id="match-method-put-$(date +%s)"
    send_request "api.example.com" "/api/users/1" "PUT" "-H 'X-B3-TraceID: $trace_id' -d '{\"name\":\"updated\"}'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "PUT 方法匹配成功"
    else
        log_fail "PUT 方法匹配失败"
    fi
    
    log_info "4.3 GET 方法不匹配"
    local trace_id="no-match-method-get-$(date +%s)"
    send_request "api.example.com" "/api/users" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -eq 0 ]; then
        log_success "GET 方法不匹配验证成功"
    else
        log_fail "GET 方法不匹配验证失败: 日志被错误采集"
    fi
}

# 测试场景5: 组合条件匹配
test_combined_match() {
    print_separator
    log_info "测试场景5: 组合条件匹配"
    print_separator
    
    cat <<'EOF' | tee -a "$REPORT_FILE"
WasmPlugin 配置示例:
```yaml
spec:
  matchRules:
  - ingress:
    - my-test-ingress
    config:
      hosts:
      - "api.example.com"
      paths:
      - "/api/v1/*"
      methods:
      - POST
  pluginConfig:
    # ...
```
EOF
    
    log_info "5.1 所有条件匹配"
    local trace_id="match-combined-all-$(date +%s)"
    send_request "api.example.com" "/api/v1/users" "POST" "-H 'X-B3-TraceID: $trace_id' -d '{\"name\":\"test\"}'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "组合条件全匹配成功"
    else
        log_fail "组合条件全匹配失败"
    fi
    
    log_info "5.2 部分条件不匹配 (host 错误)"
    local trace_id="no-match-combined-host-$(date +%s)"
    send_request "other.example.com" "/api/v1/users" "POST" "-H 'X-B3-TraceID: $trace_id' -d '{\"name\":\"test\"}'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -eq 0 ]; then
        log_success "组合条件部分不匹配验证成功 (host)"
    else
        log_fail "组合条件部分不匹配验证失败 (host)"
    fi
    
    log_info "5.3 部分条件不匹配 (path 错误)"
    local trace_id="no-match-combined-path-$(date +%s)"
    send_request "api.example.com" "/public/index.html" "POST" "-H 'X-B3-TraceID: $trace_id' -d '{\"name\":\"test\"}'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -eq 0 ]; then
        log_success "组合条件部分不匹配验证成功 (path)"
    else
        log_fail "组合条件部分不匹配验证失败 (path)"
    fi
    
    log_info "5.4 部分条件不匹配 (method 错误)"
    local trace_id="no-match-combined-method-$(date +%s)"
    send_request "api.example.com" "/api/v1/users" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -eq 0 ]; then
        log_success "组合条件部分不匹配验证成功 (method)"
    else
        log_fail "组合条件部分不匹配验证失败 (method)"
    fi
}

# 测试场景6: 多个 matchRules
test_multiple_rules() {
    print_separator
    log_info "测试场景6: 多个 matchRules (OR 逻辑)"
    print_separator
    
    cat <<'EOF' | tee -a "$REPORT_FILE"
WasmPlugin 配置示例:
```yaml
spec:
  matchRules:
  # 规则1: API 路径的 POST 请求
  - ingress:
    - my-test-ingress
    config:
      paths:
      - "/api/*"
      methods:
      - POST
  # 规则2: Admin 路径的所有请求
  - ingress:
    - my-test-ingress
    config:
      paths:
      - "/admin/*"
  pluginConfig:
    # ...
```
EOF
    
    log_info "6.1 匹配第一个规则"
    local trace_id="match-rule1-$(date +%s)"
    send_request "api.example.com" "/api/users" "POST" "-H 'X-B3-TraceID: $trace_id' -d '{\"name\":\"test\"}'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "第一个规则匹配成功"
    else
        log_fail "第一个规则匹配失败"
    fi
    
    log_info "6.2 匹配第二个规则"
    local trace_id="match-rule2-$(date +%s)"
    send_request "api.example.com" "/admin/dashboard" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -gt 0 ]; then
        log_success "第二个规则匹配成功"
    else
        log_fail "第二个规则匹配失败"
    fi
    
    log_info "6.3 两个规则都不匹配"
    local trace_id="no-match-rules-$(date +%s)"
    send_request "api.example.com" "/public/index.html" "GET" "-H 'X-B3-TraceID: $trace_id'"
    local count=$(check_log_collected "$trace_id")
    if [ "$count" -eq 0 ]; then
        log_success "多规则不匹配验证成功"
    else
        log_fail "多规则不匹配验证失败: 日志被错误采集"
    fi
}

# 生成配置示例
generate_config_examples() {
    print_separator
    log_info "WasmPlugin matchRules 配置示例汇总"
    print_separator
    
    cat <<'EOF' | tee -a "$REPORT_FILE"

## 基础配置（必填 ingress + 至少一个匹配条件）

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: http-log-pusher
  namespace: higress-system
spec:
  # matchRules 必须与 pluginConfig 同级
  matchRules:
  - ingress:
    - my-ingress-name    # 必填: Ingress 名称
    config:              # 至少包含一个匹配条件
      hosts:
      - "*.example.com"
      # 或
      paths:
      - "/api/*"
      # 或
      methods:
      - GET
      - POST
  
  # pluginConfig 与 matchRules 同级
  pluginConfig:
    collector_service_name: "log-collector.higress-system.svc.cluster.local"
    collector_host: "log-collector.higress-system.svc.cluster.local"
    collector_port: 8080
    collector_path: "/ingest"
```

## 常见错误配置

❌ 错误1: matchRules 嵌套在 config 内部
```yaml
spec:
  matchRules:
  - ingress:
    - my-ingress
    config:
      matchRules:         # ❌ 错误位置
        hosts:
        - "*.example.com"
```

❌ 错误2: 仅指定 ingress，没有匹配条件
```yaml
spec:
  matchRules:
  - ingress:
    - my-ingress         # ❌ 缺少 config 和匹配条件
  pluginConfig:
    # ...
```

✅ 正确配置
```yaml
spec:
  matchRules:            # ✅ 与 pluginConfig 同级
  - ingress:
    - my-ingress
    config:              # ✅ 包含匹配条件
      hosts:
      - "*.example.com"
  pluginConfig:          # ✅ 与 matchRules 同级
    # ...
```

EOF
}

# 主函数
main() {
    print_separator
    log_info "matchRules 验证测试开始"
    log_info "时间: $(date)"
    log_info "Gateway: ${GATEWAY_URL}:${GATEWAY_PORT}"
    log_info "Collector: ${COLLECTOR_URL:-http://localhost:8080}"
    log_info "报告文件: ${REPORT_FILE}"
    print_separator
    echo ""
    
    log_warning "注意: 此脚本需要以下前提条件："
    log_warning "1. Higress Gateway 已部署并运行"
    log_warning "2. WasmPlugin 资源已正确配置"
    log_warning "3. log-collector 服务已启动"
    log_warning "4. 测试用的 Ingress 资源已创建"
    echo ""
    
    # 生成配置示例
    generate_config_examples
    echo ""
    
    log_info "开始执行测试用例..."
    echo ""
    
    # 提示：实际测试需要对应的 WasmPlugin 配置
    log_warning "请确保已部署对应的 WasmPlugin 配置，否则测试结果无效"
    log_warning "可以使用 kubectl apply -f 部署以下配置，并在每个测试场景前切换配置"
    echo ""
    
    # 运行测试（需要手动切换配置）
    # test_ingress_match
    # test_host_match
    # test_path_match
    # test_method_match
    # test_combined_match
    # test_multiple_rules
    
    log_info "测试框架准备完成"
    log_info "请根据实际环境调整配置并运行各测试场景"
    
    print_separator
    log_info "测试脚本执行完成"
    log_info "详细报告: ${REPORT_FILE}"
    print_separator
}

# 执行主函数
main "$@"
