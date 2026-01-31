# 稳妥的Nginx到Higress迁移操作指南

**目标**: 通过本地仿真环境测试，生成可人工review的完整操作步骤，最大化降低正式环境迁移风险

---

## 整体流程

```
┌─────────────────────────────────────────────────────────────────┐
│ Phase 1: 本地仿真环境准备                                      │
│ ├─ 使用Kind创建仿真集群                                        │
│ ├─ 复制正式环境配置到仿真环境                                  │
│ └─ 部署Nginx和Higress并行运行                                 │
├─────────────────────────────────────────────────────────────────┤
│ Phase 2: 自动化测试验证                                        │
│ ├─ 生成测试脚本                                                │
│ ├─ 对比Nginx和Higress行为                                      │
│ └─ 验证所有Ingress在Higress上可用                              │
├─────────────────────────────────────────────────────────────────┤
│ Phase 3: 生成操作步骤                                          │
│ ├─ 自动生成操作清单（YAML变更、WasmPlugin、命令）              │
│ ├─ 标注每一步的风险等级                                        │
│ ├─ 生成回滚计划                                                │
│ └─ 导出为人可读的文档（Markdown）                              │
├─────────────────────────────────────────────────────────────────┤
│ Phase 4: 人工Review                                            │
│ ├─ DevOps/Platform Engineer review步骤                         │
│ ├─ 验证每一步的正确性                                          │
│ ├─ 确认WasmPlugin配置                                          │
│ └─ 签字确认                                                    │
├─────────────────────────────────────────────────────────────────┤
│ Phase 5: 灰度执行                                              │
│ ├─ 按照approved步骤执行                                        │
│ ├─ 实时监控和日志收集                                          │
│ ├─ 遇到问题则立即回滚                                          │
│ └─ 执行验证后sign off                                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: 本地仿真环境准备

### 1.1 快速搭建仿真集群

```bash
# 创建仿真集群（镜像正式环境的规模）
kind create cluster --name nginx-to-higress-simulation \
  --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 8080
    protocol: TCP
  - containerPort: 443
    hostPort: 8443
    protocol: TCP
EOF

# 验证集群就绪
kubectl cluster-info
kubectl get nodes
```

### 1.2 导出正式环境配置

```bash
# 在正式环境执行，导出所有Ingress资源
kubectl get ingress -A -o yaml > ingress-backup.yaml
kubectl get configmap -n ingress-nginx ingress-nginx-controller -o yaml > nginx-configmap.yaml
kubectl get secret -A --field-selector type=kubernetes.io/tls -o yaml > tls-secrets.yaml

# 导出Service和Endpoints（用于后端验证）
kubectl get svc -A -o yaml > services-backup.yaml
```

### 1.3 导入配置到仿真环境

```bash
# 在仿真集群中恢复配置（可能需要调整namespace等）
kubectl apply -f tls-secrets.yaml
kubectl apply -f services-backup.yaml

# 部署Nginx（仅用于对标）
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install nginx ingress-nginx/ingress-nginx \
  -n ingress-nginx --create-namespace

# 在仿真环境应用Ingress资源
kubectl apply -f ingress-backup.yaml
```

### 1.4 部署Higress（并行）

```bash
# 安装Higress（与Nginx共存）
helm repo add higress https://higress.io/helm-charts
helm repo update

helm install higress higress/higress \
  -n higress-system --create-namespace \
  --set global.ingressClass=higress \
  --set global.enableStatus=false \
  --set higress-core.gateway.replicas=1
```

---

## Phase 2: 自动化测试验证

### 2.1 生成测试脚本

```bash
#!/bin/bash
# test-migration-compatibility.sh

NGINX_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
HIGRESS_IP=$(kubectl get svc -n higress-system higress-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# 如果IP未分配，使用port-forward
if [ -z "$NGINX_IP" ]; then
  kubectl port-forward -n ingress-nginx svc/ingress-nginx-controller 8080:80 &
  NGINX_IP="127.0.0.1:8080"
fi

if [ -z "$HIGRESS_IP" ]; then
  kubectl port-forward -n higress-system svc/higress-gateway 8081:80 &
  HIGRESS_IP="127.0.0.1:8081"
fi

# 生成测试报告
TEST_REPORT="migration-test-report-$(date +%Y%m%d-%H%M%S).md"

{
  echo "# Nginx vs Higress 兼容性测试报告"
  echo "生成时间: $(date)"
  echo ""
  echo "## 测试环境"
  echo "- Nginx Gateway: $NGINX_IP"
  echo "- Higress Gateway: $HIGRESS_IP"
  echo ""
  echo "## 测试结果"
  echo ""
  
  # 遍历所有Ingress资源进行测试
  kubectl get ingress -A -o json | jq -r '.items[] | 
    [.metadata.namespace, .metadata.name, .spec.rules[0].host] | 
    @csv' | while IFS=',' read -r NS NAME HOST; do
    
    NS=$(echo $NS | tr -d '"')
    NAME=$(echo $NAME | tr -d '"')
    HOST=$(echo $HOST | tr -d '"')
    
    echo "### $NS/$NAME (Host: $HOST)"
    echo ""
    echo "| 测试项 | Nginx | Higress | 结果 |"
    echo "|--------|-------|---------|------|"
    
    # 测试基本连接
    NGINX_RESULT=$(curl -s -w "%{http_code}" -H "Host: $HOST" http://$NGINX_IP/ -o /dev/null 2>/dev/null)
    HIGRESS_RESULT=$(curl -s -w "%{http_code}" -H "Host: $HOST" http://$HIGRESS_IP/ -o /dev/null 2>/dev/null)
    
    if [ "$NGINX_RESULT" = "$HIGRESS_RESULT" ]; then
      STATUS="✅ 一致"
    else
      STATUS="⚠️ 不一致"
    fi
    
    echo "| HTTP Status | $NGINX_RESULT | $HIGRESS_RESULT | $STATUS |"
    echo ""
    
  done
  
  echo "## 总结"
  echo "- 测试时间: $(date)"
  echo "- 下一步: 检查上述结果，确保所有测试都显示 ✅ 一致"
  
} | tee $TEST_REPORT

echo "测试报告已生成: $TEST_REPORT"
```

### 2.2 执行测试

```bash
chmod +x test-migration-compatibility.sh
./test-migration-compatibility.sh
```

---

## Phase 3: 生成操作步骤

### 3.1 自动生成操作清单脚本

```bash
#!/bin/bash
# generate-migration-plan.sh

PLAN_FILE="migration-plan-$(date +%Y%m%d-%H%M%S).md"

{
  echo "# Nginx → Higress 迁移操作计划"
  echo ""
  echo "**生成时间**: $(date)"
  echo "**审批状态**: ⏳ 待Review"
  echo ""
  echo "---"
  echo ""
  echo "## 操作步骤"
  echo ""
  
  STEP=1
  
  # 步骤1: 创建WasmPlugin（如果有snippet）
  SNIPPET_COUNT=$(kubectl get ingress -A -o yaml | grep -c "snippet")
  if [ "$SNIPPET_COUNT" -gt 0 ]; then
    echo "### 步骤 $STEP: 部署WasmPlugin替代Snippet功能"
    echo ""
    echo "**风险等级**: 🟢 低 (无影响现有流量)"
    echo ""
    echo "**操作内容**:"
    echo "```bash"
    echo "# 应用WasmPlugin配置"
    echo "kubectl apply -f wasmplugin-*.yaml -n higress-system"
    echo "```"
    echo ""
    echo "**验证方式**:"
    echo "```bash"
    echo "kubectl get wasmplugin -n higress-system"
    echo "```"
    echo ""
    echo "**回滚方式**:"
    echo "```bash"
    echo "kubectl delete wasmplugin -n higress-system --all"
    echo "```"
    echo ""
    STEP=$((STEP+1))
  fi
  
  # 步骤2: 创建Higress版本的Ingress
  echo "### 步骤 $STEP: 创建Higress版本的Ingress资源"
  echo ""
  echo "**风险等级**: 🟢 低 (新增资源，不影响现有流量)"
  echo ""
  echo "**操作内容**:"
  echo "```bash"
  # 生成Higress版本的Ingress
  kubectl get ingress -A -o json | jq '
    .items[] | 
    .metadata.annotations["kubernetes.io/ingress.class"] = null |
    .spec.ingressClassName = "higress" |
    .metadata.name += "-higress"
  ' > higress-ingress-generated.yaml
  echo "kubectl apply -f higress-ingress-generated.yaml"
  echo "```"
  echo ""
  echo "**验证方式**:"
  echo "```bash"
  echo "kubectl get ingress -A | grep higress"
  echo "```"
  echo ""
  STEP=$((STEP+1))
  
  # 步骤3: 灰度流量切换
  echo "### 步骤 $STEP: 灰度切换流量（10% → 25% → 50% → 100%）"
  echo ""
  echo "**风险等级**: 🟡 中 (需要监控，可快速回滚)"
  echo ""
  echo "**操作内容**:"
  echo ""
  echo "#### 阶段1: 10%流量"
  echo "```bash"
  echo "# 修改DNS或负载均衡，10%流量指向Higress"
  echo "# 命令取决于你的基础设施，例如："
  echo "kubectl patch svc higress-gateway -n higress-system -p '{\"spec\":{\"externalTrafficPolicy\":\"Local\"}}'"
  echo "```"
  echo ""
  echo "#### 阶段2: 观察（建议15-30分钟）"
  echo "```bash"
  echo "# 监控关键指标"
  echo "watch kubectl top pod -n higress-system"
  echo "kubectl logs -f -n higress-system -l app=higress-gateway"
  echo "```"
  echo ""
  echo "#### 阶段3: 逐步增加流量"
  echo "25% → 50% → 100%，每个阶段观察15-30分钟"
  echo ""
  STEP=$((STEP+1))
  
  # 步骤4: 删除Nginx（可选，谨慎操作）
  echo "### 步骤 $STEP: 清理Nginx Ingress Controller（仅在100%流量切换后）"
  echo ""
  echo "**风险等级**: 🔴 高 (不可逆，必须100%确认后)"
  echo ""
  echo "**前置条件**:"
  echo "- ✅ Higress已承载100%流量，运行稳定"
  echo "- ✅ 所有关键指标正常"
  echo "- ✅ 所有alert都已处理"
  echo ""
  echo "**操作内容**:"
  echo "```bash"
  echo "# 备份Nginx配置（防止意外）"
  echo "kubectl get all -n ingress-nginx -o yaml > nginx-backup-final.yaml"
  echo ""
  echo "# 删除Nginx"
  echo "helm uninstall nginx -n ingress-nginx"
  echo "kubectl delete namespace ingress-nginx"
  echo "```"
  echo ""
  STEP=$((STEP+1))
  
  echo "---"
  echo ""
  echo "## 风险评估"
  echo ""
  echo "| 步骤 | 风险等级 | 影响范围 | 回滚时间 |"
  echo "|------|---------|---------|---------|"
  echo "| 1. 部署WasmPlugin | 🟢 低 | 仅Higress | <1分钟 |"
  echo "| 2. 创建Higress Ingress | 🟢 低 | 新增资源 | <1分钟 |"
  echo "| 3. 灰度切换 | 🟡 中 | 部分用户 | <5分钟 |"
  echo "| 4. 删除Nginx | 🔴 高 | 全体用户 | 5-15分钟 |"
  echo ""
  echo "**总体风险**: 🟡 中 (通过灰度和monitoring可控)"
  echo ""
  
  echo "---"
  echo ""
  echo "## 回滚计划"
  echo ""
  echo "### 快速回滚（<5分钟）"
  echo "```bash"
  echo "# 将流量切回Nginx"
  echo "# 操作取决于你的基础设施（DNS/LB/Proxy）"
  echo "```"
  echo ""
  echo "### 完全回滚（如果需要）"
  echo "```bash"
  echo "# 删除Higress相关资源"
  echo "kubectl delete wasmplugin -n higress-system --all"
  echo "kubectl delete ingress -A -l created-by=migration-plan"
  echo "```"
  echo ""
  
  echo "---"
  echo ""
  echo "## 监控指标"
  echo ""
  echo "迁移过程中关键监控指标:"
  echo ""
  echo "### 必监控"
  echo "- 错误率 (HTTP 5xx) - 应保持 < 0.1%"
  echo "- P99延迟 - 应无明显增加"
  echo "- Pod CPU/Memory - 应在正常范围"
  echo ""
  echo "### 告警规则建议"
  echo "```yaml"
  echo "- alert: HighgressErrorRateHigh"
  echo "  expr: rate(higress_request_total{status=~\"5..\"}[5m]) > 0.001"
  echo ""
  echo "- alert: HighgressLatencyHigh"
  echo "  expr: histogram_quantile(0.99, higress_request_duration) > 1000ms"
  echo ""
  echo "- alert: HighgressPodMemoryHigh"
  echo "  expr: container_memory_usage_bytes{pod=~\"higress-.*\"} > 1Gi"
  echo "```"
  echo ""
  
  echo "---"
  echo ""
  echo "## 审批信息"
  echo ""
  echo "**审批人**: _________________"
  echo "**审批时间**: _________________"
  echo "**备注**: _________________"
  echo ""
  
} | tee $PLAN_FILE

echo ""
echo "✅ 迁移计划已生成: $PLAN_FILE"
echo "📋 请将此文件发给DevOps/Platform Engineer进行Review"
```

### 3.2 生成操作计划

```bash
chmod +x generate-migration-plan.sh
./generate-migration-plan.sh
```

**输出示例** (`migration-plan-YYYYMMDD-HHMMSS.md`):
```
# Nginx → Higress 迁移操作计划

生成时间: 2026-01-31 08:46:00
审批状态: ⏳ 待Review

## 操作步骤

### 步骤 1: 部署WasmPlugin替代Snippet功能
风险等级: 🟢 低 (无影响现有流量)
...

### 步骤 2: 创建Higress版本的Ingress资源
风险等级: 🟢 低 (新增资源，不影响现有流量)
...

### 步骤 3: 灰度切换流量（10% → 25% → 50% → 100%）
风险等级: 🟡 中 (需要监控，可快速回滚)
...

### 步骤 4: 清理Nginx Ingress Controller
风险等级: 🔴 高 (不可逆)
...

## 风险评估

| 步骤 | 风险等级 | 影响范围 | 回滚时间 |
|...
```

---

## Phase 4: 人工Review

### 4.1 Review检查清单

**DevOps/Platform Engineer应检查**:

```markdown
- [ ] **操作步骤完整性**
  - [ ] 每个步骤都有明确的kubectl命令
  - [ ] 每个步骤都有验证方式
  - [ ] 每个步骤都有回滚计划
  - [ ] 灰度阶段的流量分配合理

- [ ] **风险评估准确性**
  - [ ] 风险等级标注正确
  - [ ] 影响范围评估准确
  - [ ] 回滚时间估计合理

- [ ] **WasmPlugin配置正确**
  - [ ] Snippet迁移的plugin配置完整
  - [ ] 正则表达式验证正确
  - [ ] Config参数合理

- [ ] **监控和告警**
  - [ ] 关键告警规则已配置
  - [ ] 监控Dashboard已准备
  - [ ] 告警接收人明确

- [ ] **沟通和文档**
  - [ ] 已通知相关团队
  - [ ] 用户沟通计划清楚
  - [ ] 事后回顾时间已安排
```

### 4.2 Review模板

```markdown
# 迁移操作计划 Review

**计划ID**: migration-plan-20260131-084600.md
**Review者**: _______________
**Review时间**: _______________

## Review结果

### 1. 操作步骤检查
- [ ] ✅ 通过 / [ ] ⚠️ 需要修改 / [ ] ❌ 不通过

**意见**:
```
[输入意见]
```

### 2. 风险评估检查
- [ ] ✅ 通过 / [ ] ⚠️ 需要修改 / [ ] ❌ 不通过

**意见**:
```
[输入意见]
```

### 3. 监控告警检查
- [ ] ✅ 通过 / [ ] ⚠️ 需要修改 / [ ] ❌ 不通过

**意见**:
```
[输入意见]
```

## 总体评分

- 🟢 **通过** - 可以执行
- 🟡 **有条件通过** - 修改后可执行
- 🔴 **不通过** - 需要重新评估

**最终评分**: [ ] 🟢 / [ ] 🟡 / [ ] 🔴

**签名**: _______________ **日期**: _______________
```

---

## Phase 5: 灰度执行

### 5.1 执行检查清单

```bash
#!/bin/bash
# pre-migration-checklist.sh

echo "🔍 迁移前最终检查..."
echo ""

# 1. 确认仿真环境通过测试
echo "1️⃣ 仿真环境测试结果检查"
if [ -f migration-test-report-*.md ]; then
  echo "   ✅ 测试报告已生成"
else
  echo "   ❌ 未找到测试报告，请先运行 test-migration-compatibility.sh"
  exit 1
fi

# 2. 确认Review已完成
echo "2️⃣ Review状态检查"
read -p "   📋 Review是否已完成并批准? (y/n): " REVIEW_STATUS
if [ "$REVIEW_STATUS" != "y" ]; then
  echo "   ❌ 请先完成Review"
  exit 1
fi

# 3. 确认回滚计划就绪
echo "3️⃣ 回滚计划检查"
read -p "   📋 回滚计划是否已充分演练? (y/n): " ROLLBACK_STATUS
if [ "$ROLLBACK_STATUS" != "y" ]; then
  echo "   ❌ 请先演练回滚计划"
  exit 1
fi

# 4. 确认监控和告警已配置
echo "4️⃣ 监控告警检查"
read -p "   📋 监控告警是否已配置? (y/n): " MONITORING_STATUS
if [ "$MONITORING_STATUS" != "y" ]; then
  echo "   ❌ 请先配置监控告警"
  exit 1
fi

# 5. 备份所有配置
echo "5️⃣ 备份配置"
BACKUP_DIR="pre-migration-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p $BACKUP_DIR
kubectl get all -A -o yaml > $BACKUP_DIR/all-resources.yaml
echo "   ✅ 配置已备份到 $BACKUP_DIR"

echo ""
echo "✅ 所有检查通过，可以开始迁移！"
echo ""
echo "下一步: 按照approved的迁移计划执行"
```

### 5.2 执行和监控

```bash
# 运行前置检查
./pre-migration-checklist.sh

# 开始执行（按照approved的migration-plan-*.md操作）
# 每个步骤后都要观察监控5-10分钟

# 监控关键指标
watch kubectl top pod -n higress-system
kubectl logs -f -n higress-system -l app=higress-gateway
```

---

## 总结：为什么这个方案更稳妥

| 环节 | 传统方式 | 稳妥方式 | 优势 |
|------|---------|---------|------|
| **测试** | 无或仓促 | 完整的自动化测试 | 提前发现问题 |
| **计划** | 口头或模糊 | 自动生成的详细步骤 | 清晰可追踪 |
| **审核** | 可能缺失 | 强制的人工Review | 多人把关 |
| **执行** | 凭经验 | 按照review通过的步骤 | 降低遗漏风险 |
| **回滚** | 可能混乱 | 每步都有回滚计划 | 快速恢复 |
| **追溯** | 困难 | 完整的文档记录 | 事后回顾 |

**结果**: ✅ 从无序 → 有序、从抓瞎 → 有据、从高风险 → 可控

