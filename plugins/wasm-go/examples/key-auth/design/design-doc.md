# key-auth 路由级三列表允许配置设计文档（allow / allow_groups / allow_credentials）

- 日期：2026-09-29
- 状态：已评审通过（Go + C++ 双端同步实现）
- 涉及代码：
  - Go：`plugins/wasm-go/examples/key-auth/`
  - C++：`plugins/wasm-cpp/extensions/key_auth/`、`plugins/wasm-cpp/common/route_rule_matcher.h` 及其余 4 个使用方插件（jwt_auth、basic_auth、hmac_auth、oauth）的机械适配

## 1. 背景与目标

key-auth 插件目前仅在路由/域名级支持 `allow`（consumer 名称列表）一种授权粒度。本次新增两个路由级允许列表，实现更细粒度的访问控制：

1. `allow`：允许的消费者名称列表（现有，保持不变）。
2. `allow_groups`：允许的消费者组名列表（新增）。
3. `allow_credentials`：允许的 API Key（凭证值）列表（新增）——请求携带的 API Key 命中列表即放行，对应 consumer 由全局凭证映射自动确定，适合"把某个 token 直接授权给某条路由"而无需关心 consumer 名称的场景。

### 匹配语义（OR）

认证通过的 consumer 满足任一条件即放行：

1. `consumer.Name ∈ allow`
2. `consumer.Group ∈ allow_groups`（无组的 consumer 不可能命中）
3. 当前请求携带的凭证 ∈ `allow_credentials`

命中任一列表后，`X-Mse-Consumer` 与 `X-Mse-Consumer-Group`（有组时）照常注入。

### 不生效的场景（保持现状）

- `global_auth=true` 且当前路由/域名未配置该插件：所有认证通过的 consumer 放行。
- `global_auth` 未设置且没有任何路由/域名配置该插件：同上。

允许列表只在规则级起作用。

## 2. 配置说明

### 路由/域名级配置

```yaml
_rules_:
- _match_route_:
  - route-a
  allow:              # 现有：consumer 名称列表
  - consumer1
  allow_groups:       # 新增：consumer 组名列表
  - team-a
  allow_credentials:  # 新增：API Key（凭证值）列表
  - token3
```

### 校验规则

- 路由级三个字段（`allow` / `allow_groups` / `allow_credentials`）至少配置一个，否则配置加载失败（Go 与 C++ 一致）。
- 已配置的字段不能是空数组。此为 Go 侧校验，违反则加载失败；C++ 侧空数组解析为空集、永不命中（差异见 4.4）。
- 不校验列表值是否存在于全局 `consumers` / 组名 / 凭证中（与 `allow` 现状一致，避免与全局配置的下发顺序耦合）。不存在的值只是永不命中。
- 元素解析保持与现有 `allow` 相同的宽松风格（Go 侧 gjson `String()`）。

## 3. Go 侧改动（plugins/wasm-go/examples/key-auth/main.go）

### 配置结构

`KeyAuthConfig` 新增：

```go
allowGroups      []string `yaml:"allow_groups"`
allowCredentials []string `yaml:"allow_credentials"`
```

与现有 `allow` 字段一致，不加 @Title 等注解（`allow` 现状即无注解）。

### 路由级解析（parseOverrideRuleConfig）

从"`allow` 必填"改为"三字段至少一个"；逐字段解析追加。

### 匹配逻辑（onHttpRequestHeaders）

- `noAllow` 定义改为"三个列表全空"，即该路由未配置本插件的授权规则。
- 现有三段重复的 `contains(config.allow, consumer.Name)` 检查合并为一处：

```go
if !noAllow && !consumerAllowed(config, consumer, tokens[0]) {
    return deniedUnauthorizedConsumer()
}
```

```go
func consumerAllowed(config KeyAuthConfig, consumer Consumer, credential string) bool {
    if contains(config.allow, consumer.Name) {
        return true
    }
    if consumer.Group != "" && contains(config.allowGroups, consumer.Group) {
        return true
    }
    return contains(config.allowCredentials, credential)
}
```

重构前后行为等价，四种组合逐一核对：

| 场景 | 现状 | 重构后 |
| --- | --- | --- |
| 全局生效 + 路由未配置 | 放行 | `!noAllow == false`，不检查，放行 |
| 全局生效 + 路由已配置 | 检查 allow | 检查三列表 OR |
| 非全局 + 路由已配置 | 检查 allow | 检查三列表 OR |
| 非全局 + 路由未配置 | 开头已放行 | 开头已放行（不变） |

继续使用线性 `contains`，与现有风格一致（允许列表通常很短，不引入 map 索引）。`X-Mse-Consumer` 注入位置不动。

## 4. C++ 侧改动

### 4.1 共享基础设施（plugins/wasm-cpp/common/route_rule_matcher.h）

`allow` 的路由级解析位于共享的 `RouteRuleMatcher::parseAuthRuleConfig`，jwt_auth、basic_auth、hmac_auth、oauth 与 key_auth 共用。关键存量语义：**规则未配置 `allow` 时 `allow_set` 是空集 → `checkAllowSet` 拒绝所有人**。本设计把"未配置的字段"视为空集，"至少一个列表命中才放行"的 OR 语义自动涵盖该存量行为，无需 `has_allow` 标志或 optional 包装。

改动点：

1. `AuthRuleConfig` 新增两个普通集合成员（与 `allow_set` 同模式）：
   ```cpp
   std::unordered_set<std::string> allow_groups_set;
   std::unordered_set<std::string> allow_credentials_set;
   ```
2. `parseAuthRuleConfig` 中与 `allow` 并列解析 `allow_groups` / `allow_credentials`（同样的 `JsonArrayIterate` + `JsonValueAs<std::string>` 类型校验风格）。
3. 新增小结构（避免回调参数膨胀，指针在 `checkAuthRule` 栈帧内构建、生命周期安全）：
   ```cpp
   struct AuthAllowLists {
     const std::unordered_set<std::string>* allow = nullptr;              // null = 无规则匹配，不检查
     const std::unordered_set<std::string>* allow_groups = nullptr;
     const std::unordered_set<std::string>* allow_credentials = nullptr;
   };
   ```
4. `getMatchAuthConfig` 第二个返回值从 `std::optional<std::reference_wrapper<std::unordered_set<std::string>>>` 改为 `std::optional<std::reference_wrapper<AuthRuleConfig>>`。已确认该函数仅被 `checkAuthRule` 调用。
5. `checkAuthRule` 构建栈上 `AuthAllowLists` 并传给回调：
   - 规则匹配且 `has_local_config == false`：三项分别指向规则的 `allow_set` / `allow_groups_set` / `allow_credentials_set`（未配置的为空集，OR 判断下永不命中）。
   - 规则匹配且 `has_local_config == true`：全 null（沿用现状——规则本地配置自管，不检查全局允许列表）。
   - 无规则匹配（`global_auth=true` 全放行路径）：全 null。
6. `checkAuthRule` 回调签名改为接收 `const PluginConfig&` 与 `const AuthAllowLists&`。

### 4.2 key_auth 插件（plugins/wasm-cpp/extensions/key_auth/）

- `PluginRootContext::checkPlugin` 签名改为接收 `const AuthAllowLists&`。
- 现有 `checkAllowSet`（allow_set 存在时：空集 → 403；不含 name → 403）替换为 OR 检查：name / group / credential 三条件任一命中放行，否则 `deniedUnauthorizedConsumer`。4 处调用点统一替换。
- 组名查询复用 `rule.name_to_group`（与 `addConsumerHeaders` 一致）。
- credentials-only 分支（无 `consumers`、顶层 `credentials` 的旧用法）走同一 OR 检查：该模式无消费者身份，name/group 以空串参与判断（列表含空串才可能命中，实际不会命中），`allow_credentials` 对当前凭证可命中。与 Go 版语义一致：该模式下 `allow` / `allow_groups` 不生效，`allow_credentials` 生效。

### 4.3 其余 4 个使用方插件（行为不变）

jwt_auth、basic_auth、hmac_auth、oauth：lambda 与 `checkPlugin` 签名机械适配（改用 `lists.allow`），内部逻辑不动。现状 `checkAllowSet` 对空集/未命中的 403 行为原样保留（空集仍全拒，与现状一致）。

### 4.4 行为修正说明（有意为之）

| 场景 | 改动前 | 改动后 |
| --- | --- | --- |
| 规则只配 `allow_groups`（不配 `allow`） | 解析失败（`parsePluginConfig` 要求 consumers/credentials，`has_allow` 缺失导致整份配置被拒） | 正常工作，按 OR 判断 |
| 规则配 `allow: []` + `allow_groups: [t]` | 空集 `allow_set` → 403 全拒 | `t` 组命中放行 |
| 规则未配任何允许字段 | 空集 `allow_set` → 403 全拒 | 三空集 → 403 全拒（不变） |
| 无规则匹配 + `global_auth=true` | `allow_set = nullopt` → 全放行 | lists 全 null → 全放行（不变） |

注：Go 侧"已配置字段不能是空数组"会拒绝加载，C++ 侧空数组解析为空集（不命中）。错误配置在两端都不会导致意外放行，安全方向一致。

## 5. 错误处理

- Go：`parseOverrideRuleConfig` 校验失败（三字段全缺、已配置字段为空数组）→ 返回 error → 插件配置加载失败。
- C++：解析失败 → `LOG_WARN` + return false → 插件配置加载失败。
- 拒绝响应不变：未携带 key → 401 `key-auth.no_key`；多个 key → 401 `key-auth.multi_key`；key 未注册 → 401 `key-auth.unauthorized`；认证通过但不在允许列表 → 403 `key-auth.unauthorized`。

## 6. 测试计划

### Go（本地 go test）

- 解析用例：只配 `allow_groups` / 只配 `allow_credentials` / 三者全配 / 全缺（报错）/ 已配置字段为空数组（报错）。
- 匹配用例：组命中放行、组不命中 403、credential 命中放行、credential 不命中 403、无组 consumer 不被 `allow_groups` 误放、`global_auth` true/false 两分支、`X-Mse-Consumer` / `X-Mse-Consumer-Group` 注入断言。
- 回归：现有 `main_test.go`、`main_extra_test.go` 全部保持通过。

### C++（测试机 bazel）

- 环境：192.168.0.194 / 192.168.0.189，沿用已验证的 bazel 配置（`--features=-module_maps` + distdir 等绕行参数）。
- key_auth 新语义用例：解析（新字段、只配 `allow_groups` 不再失败）、OR 匹配（组/凭证/名称命中与不命中）、credentials-only 分支 `allow_credentials`。
- 回归：jwt_auth、basic_auth、hmac_auth、oauth 现有测试编译并通过。

## 7. 文档更新

- `plugins/wasm-go/examples/key-auth/README.md` / `README_EN.md`：鉴权配置表新增 `allow_groups`、`allow_credentials` 两行 + OR 语义说明与配置示例。
- C++ 侧无独立 README，不涉及。

## 8. 范围外（明确不做）

- 路由级 `deny` 黑名单。
- 允许列表的 map 索引优化（列表短，线性查找足够）。
- 校验列表值必须存在于全局 consumers（避免下发顺序耦合，与 `allow` 现状一致）。
- 其他认证插件（jwt/basic/hmac/oauth）的 groups/credentials 语义扩展——仅保留扩展点。
