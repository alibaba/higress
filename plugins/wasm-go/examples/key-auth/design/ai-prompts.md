# key-auth 路由级三列表允许配置 — AI 编码提示记录

- 日期：2026-09-29
- 工具：Claude Code（Claude Code CLI， brainstorming 技能驱动需求澄清与设计）

## 提示记录

1. **初始需求**
   > plugins\wasm-go\examples\key-auth\main.go 需要支持 允许请求本路由的消费者名称列表，消费者组列表，apikey 列表

2. **需求澄清问答**（多选题逐个确认）
   - "apikey 列表"的含义 → 确认为**凭证值白名单**（路由级配置 credential 值，命中即放行对应 consumer），而非 keys 来源字段白名单。
   - 新增字段命名 → 确认 `allow_groups` + `allow_credentials`（`allow` 保持不变，兼容存量）。
   - 三列表关系 → 确认 **OR：任一命中即放行**。
   - 改动范围 → 确认 **Go + C++ 双端同步**（当前分支此前刚把 consumer group 移植到 C++ 版 key_auth）。

3. **方案选择**
   > A：扩展共享 AuthRule 基础设施（推荐）／ B：key_auth 私有解析 ／ C：结构化新字段
   - 确认采用**方案 A**：`route_rule_matcher.h` 的 `AuthRuleConfig` 与 `allow` 并列新增集合，`checkAuthRule` 回调改传 `AuthAllowLists`，key_auth 实现 OR 语义，其余 4 个使用方插件机械适配、行为不变。

4. **设计评审**
   - 呈现 6 节设计（配置结构 / Go 改动 / C++ 改动 / 错误处理 / 测试 / 文档）。
   - 关键确认点：C++ 空集统一语义——"未配置字段视为空集"使 OR 语义自然涵盖存量"空集全拒"行为；两处行为修正（只配 `allow_groups` 不再解析失败；`allow: []`+`allow_groups` 组合从全拒改为组可命中）被确认为有意修正。
   - 结论：**确认，继续**。

## 设计要点备忘

- Go 侧将三段重复的 `contains(config.allow, ...)` 合并为 `consumerAllowed` 单一出口，`noAllow` 重定义为三列表全空，行为等价。
- C++ 侧 `getMatchAuthConfig` 第二返回值改传整个 `AuthRuleConfig` 引用（已确认仅 `checkAuthRule` 一个调用方）；`AuthAllowLists` 三个裸指针在 `checkAuthRule` 栈帧内构建，规避 vector 重排导致的指针失效。
- credentials-only 旧用法下 `allow`/`allow_groups` 不生效（无消费者身份），`allow_credentials` 生效，双端一致。
