# MCP Guard 集成与验证（控制面 + 数据面）

本样例说明如何通过 `higress-config` 配置在数据面注入 `mcp-guard` 授权守卫，并验证允许/不允许：

## 前提
- 已安装 Higress（控制面 + 数据面）在 `higress-system` 名称空间
- 路由存在如下两个 API：
  - `POST /v1/images:moderate`
  - `POST /v1/text:summarize`

## 步骤

1) 应用配置（启用 mcp-guard）

```bash
kubectl apply -f samples/mcp-guard/higress-config.yaml
```

2) 配置说明
- `requestedCapabilityHeader: X-MCP-Capability`
- `subjectPolicy`：
  - tenantA → cap.image.moderate, cap.text.summarize
  - tenantB → cap.text.summarize
- `rules`：
  - 路径前缀 `/v1/images:moderate` → 允许能力 cap.image.moderate
  - 路径前缀 `/v1/text:summarize` → 允许能力 cap.text.summarize

3) Demo 验证

- 允许：tenantA 访问 images:moderate（声明能力 cap.image.moderate）

```bash
curl -i -X POST \
  -H 'Host: api.example.com' \
  -H 'X-Subject: tenantA' \
  -H 'X-MCP-Capability: cap.image.moderate' \
  http://<gateway-ip>/v1/images:moderate
```

- 拒绝：tenantB 访问 images:moderate（只被授权 text.summarize）

```bash
curl -i -X POST \
  -H 'Host: api.example.com' \
  -H 'X-Subject: tenantB' \
  -H 'X-MCP-Capability: cap.image.moderate' \
  http://<gateway-ip>/v1/images:moderate
```

预期返回：HTTP/403，body 类似 `mcp-guard deny: reason=requested-cap-not-allowed`。

> 注：在生产中建议使用 JWT/OIDC 做身份，mcp-guard 可从 `jwt_authn` 注入的 dynamic metadata 获取主体，而不是直接读 `X-Subject`。

## 清理

```bash
kubectl -n higress-system delete configmap higress-config
```

