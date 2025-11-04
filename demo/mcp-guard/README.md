# mcp-guard Demo (MCP 能力身份管理：允许/不允许)

本 Demo 以最小自包含 Go 模块演示“按主体（Subject）与能力集（Capabilities）授权”的判定逻辑，覆盖：
- 允许访问：主体与路由能力集有交集，且（若声明具体能力）在交集内
- 不允许访问：无主体、无交集、或请求的能力不在授权集合

## 运行测试

```bash
cd demo/mcp-guard
go test ./... -v
```

可看到如下用例通过：
- TestCheckAccess_NoSubject（拒绝）
- TestCheckAccess_AllowedIntersection（允许）
- TestCheckAccess_DenyWrongCap（拒绝）
- TestCheckAccess_AllowedWhenReqCapEmpty（允许）

## 判定模型（精简版）

- 输入：
  - Headers：`X-Subject`（或 `Authorization: Bearer <subject>`）、`X-MCP-Capability`（可选）
- 配置：
  - route 允许：`AllowedCapabilities`
  - 主体授权：`SubjectPolicy`（subject → capabilities[]）
- 逻辑：
  - `eff = intersection(AllowedCapabilities, SubjectPolicy[subject])`
  - 若 `subject == ""` → 拒绝
  - 若 `len(eff) == 0` → 拒绝
  - 若 `X-MCP-Capability` 非空且不在 `eff` → 拒绝
  - 否则允许

该逻辑与 `higress/plugins/wasm-go/extensions/mcp-guard` 插件实现一致（插件在数据面中将作为授权守卫，位于 MCP Server 之前）。

