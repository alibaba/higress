# 面向客户的“AI 能力授权”演示方案（两条路径）

本演示聚焦业务价值：同一套 AI API，根据“客户身份（租户/套餐）”授权不同能力集。白金客户可用“摘要 + 翻译”，标准客户仅可用“摘要”。未授权将返回友好的 403 提示。

你可按网络情况选择两条路径：

- A. 真实 AI（OpenAI/兼容厂商）
  - 使用 Higress 的 `ai-proxy` 将客户请求转发到实际 AI 厂商；无需改造调用方协议
  - 需要有效的 API Key（你已有线上账号即可）
- B. 本地 Mock AI（离线、可控）
  - 使用一个轻量 HTTP 服务模拟“摘要/翻译”两个能力，便于稳定演示

授权控制统一由 `mcp-guard` 插件完成。能力“注册”（catalog）通过示例 CRD/YAML 展示，转换后由 `higress-config` 下发到数据面。

---

## 演示目标（对业务的意义）
- 区分客户身份：白金/标准/来宾
- 授权差异化能力：
  - 白金：允许 “文本摘要（summarize）+ 翻译（translate）”
  - 标准：仅允许 “文本摘要（summarize）”
  - 未注册客户：全部拒绝
- 访问体验：
  - 调用摘要（/v1/text:summarize）均成功
  - 调用翻译（/v1/text:translate）标准客户被友好拒绝（403 + 文案）

---

## A. 真实 AI（OpenAI/兼容）

1) 前置
- 具备可用的 OpenAI API Key（或 OpenRouter 等兼容厂商 Key）
- 按照官方文档部署 Higress（Helm）与网关

2) 安装 ai-proxy（Wasm）
- 参考 `higress/plugins/wasm-go/README.md` 构建或使用官方镜像
- 为简化演示，路径映射约定如下：
  - `/v1/text:summarize` → 转发到 OpenAI `chat.completions`（系统 prompt 固定为“请给出简短摘要”）
  - `/v1/text:translate` → 转发到 OpenAI `chat.completions`（系统 prompt 固定为“请翻译为英文”）

3) 配置授权（mcp-guard）
- 应用示例配置：

```bash
kubectl apply -f samples/mcp-guard/higress-config.yaml
```

- 该文件中：
  - `subjectPolicy` 指定客户身份和其可用能力（capabilities）
  - `rules` 将每个 API 路径前缀映射为能力集（如 summarize / translate）

4) 演示
- 白金客户（tenantA）访问摘要：

```bash
curl -i -X POST \
  -H 'Host: api.example.com' \
  -H 'X-Subject: tenantA' \
  -H 'X-MCP-Capability: cap.text.summarize' \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"用一句话概述：Higress 是什么？"}' \
  http://<gateway-ip>/v1/text:summarize
```

- 标准客户（tenantB）访问翻译（预期 403）：

```bash
curl -i -X POST \
  -H 'Host: api.example.com' \
  -H 'X-Subject: tenantB' \
  -H 'X-MCP-Capability: cap.text.translate' \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"请将这句话翻译为英文：你好，世界"}' \
  http://<gateway-ip>/v1/text:translate
```

> 提示：演示场景中我们使用请求头 `X-Subject` 表示客户身份；生产建议用 JWT/OIDC 等方式携带身份，由 `jwt_authn` 解析并注入。

---

## B. 本地 Mock AI（离线）

1) 部署 Mock 服务
- 该服务提供两个接口：
  - `POST /v1/text:summarize`：返回“摘要：<前 30 字>...”
  - `POST /v1/text:translate`：返回“翻译：<原文> -> <EN 模拟>”
- 参考文件：`samples/ai-tools`（如需，我可以为你补齐 Deployment/Service 与镜像构建说明）

2) 配置路由
- 将网关路由 `/v1/text:*` 指向 Mock 服务的 Service
- 应用授权配置：

```bash
kubectl apply -f samples/mcp-guard/higress-config.yaml
```

3) 演示
- 与“真实 AI”路径一致，使用两套身份对摘要/翻译接口进行访问，观察 200 与 403 的对比

---

## 能力注册（示例 CRD）

以下 CRD 用于“对外展示 catalog”的目的，便于客户理解与选型；当前演示中它们尚未被控制器实时消费，但可通过脚本映射为 `higress-config`（如需我可以提供脚本）：

```yaml
# samples/mcp-guard/crds/capabilities.yaml
apiVersion: mcp.higress.io/v1alpha1
kind: McpCapability
metadata:
  name: cap.text.summarize
spec:
  type: tool
  version: v1
  description: "将文本浓缩为一句话"
---
apiVersion: mcp.higress.io/v1alpha1
kind: McpCapability
metadata:
  name: cap.text.translate
spec:
  type: tool
  version: v1
  description: "将中文翻译为英文"
---
apiVersion: mcp.higress.io/v1alpha1
kind: McpCapabilitySet
metadata:
  name: capset.standard
spec:
  capabilities:
    - cap.text.summarize
---
apiVersion: mcp.higress.io/v1alpha1
kind: McpCapabilitySet
metadata:
  name: capset.premium
spec:
  capabilities:
    - cap.text.summarize
    - cap.text.translate
---
apiVersion: mcp.higress.io/v1alpha1
kind: McpAccessPolicy
metadata:
  name: policy.demo
spec:
  subjectSelector:
    claims:
      tenant_id: [tenantA, tenantB]
  objectSelector:
    hosts: ["api.example.com"]
  # 这里用于人类可读展示（演示时说明“如何与产品套餐挂钩”）
  rules:
    - subject: tenantA
      allowedSets: [capset.premium]
    - subject: tenantB
      allowedSets: [capset.standard]
```

> 如需“CRD → higress-config”的自动转换，我可以提供一个小脚本或控制器在演示时运行。

---

## 为什么这对客户有价值？
- 套餐差异化：按客户身份（租户/套餐）定制可用 AI 能力，灵活升级与增购
- 安全合规：未授权能力默认拒绝，支持影子模式（只记录不拦截）平滑上线
- 零改造接入：ai-proxy 负责“协议适配 + 厂商差异屏蔽”，客户只需面向统一 API
- 可观测与审计：每次访问带上身份、能力、结果（允许/拒绝），可生成报表

---

## 快速复现清单
- 已完成：
  - `mcp-guard` 插件二进制（plugin.wasm）已构建：`higress/plugins/wasm-go/extensions/mcp-guard/plugin.wasm`
  - 授权配置样例：`samples/mcp-guard/higress-config.yaml`
  - 参数校验（不拉镜像）：`ONLY_PRINT_NODE_IMAGE=1 tools/hack/create-cluster.sh`
- 如需：我可按你提供的镜像源拉起 kind 集群，或接入你已有 K8s 集群，完成现场演示脚本（含 curl 文案）。

