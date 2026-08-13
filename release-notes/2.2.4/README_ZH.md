# Higress Console


## 📋 本次发布概览

本次发布包含 **18** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 7项
- **Bug修复**: 9项
- **文档更新**: 2项

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#621](https://github.com/higress-group/higress-console/pull/621) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: 优化MCP Server交互能力：支持DNS后端自动重写Host头；增强直接路由场景的Transport选择与完整路径配置；改进DB到MCP Server场景的DSN特殊字符（如@）解析逻辑。 \
  **Feature Value**: 提升MCP Server接入灵活性与兼容性，使用户能更便捷地配置不同后端类型，避免路径歧义和认证字符解析失败问题，降低运维复杂度并增强系统稳定性。

- **Related PR**: [#608](https://github.com/higress-group/higress-console/pull/608) \
  **Contributor**: @Libres-coder \
  **Change Log**: 为AI路由管理页面新增插件展示功能，通过扩展AI路由行显示已启用插件，并在配置页添加'Enabled'标签，同时在前端新增i18n国际化支持及插件查询逻辑适配。 \
  **Feature Value**: 用户可在AI路由管理界面直观查看和管理已启用插件，与常规路由功能对齐，提升AI路由配置透明度和操作一致性，降低学习与运维成本。

- **Related PR**: [#604](https://github.com/higress-group/higress-console/pull/604) \
  **Contributor**: @CH3CHO \
  **Change Log**: 新增基于正则表达式的路径重写功能，通过higress.io/rewrite-target注解支持复杂路径匹配与替换，扩展了KubernetesModelConverter中重写配置解析逻辑，并在前后端国际化资源中新增正则重写类型标识。 \
  **Feature Value**: 使用户能灵活定义正则规则进行精细化路径重写，提升路由匹配能力与转发灵活性，适用于多版本API管理、动态路径映射等场景，显著增强网关的适配性和运维效率。

- **Related PR**: [#603](https://github.com/higress-group/higress-console/pull/603) \
  **Contributor**: @CH3CHO \
  **Change Log**: 在静态服务源表单组件中新增常量STATIC_SERVICE_PORT = 80，并在UI中显示该固定端口，使用户明确知晓静态服务默认使用80端口，提升配置透明度和一致性。 \
  **Feature Value**: 用户在配置静态服务源时能直观看到预设的80端口，避免因端口误解导致的服务部署失败，降低使用门槛，提升配置准确性和运维效率。

- **Related PR**: [#602](https://github.com/higress-group/higress-console/pull/602) \
  **Contributor**: @CH3CHO \
  **Change Log**: 在AI路由的上游服务选择组件中新增搜索功能，通过前端输入过滤服务列表，提升服务选择效率。修改RouteForm组件，增加搜索输入框及过滤逻辑，仅需少量代码变更即可实现交互增强。 \
  **Feature Value**: 用户在配置AI路由时可快速筛选目标上游服务，避免在大量服务中手动滚动查找，显著提升配置效率和用户体验，尤其适用于微服务数量庞大的生产环境。

- **Related PR**: [#566](https://github.com/higress-group/higress-console/pull/566) \
  **Contributor**: @OuterCyrex \
  **Change Log**: 新增通义千问（Qwen）大模型服务支持，包括自定义服务地址、互联网搜索启用、文件ID上传等功能，新增QwenLlmProviderHandler实现，并在前后端完成配置项和国际化适配。 \
  **Feature Value**: 用户可通过界面便捷配置私有或第三方Qwen服务，提升AI能力扩展性；支持互联网搜索与文件ID上传，增强实际业务场景下的LLM调用灵活性和实用性。

- **Related PR**: [#552](https://github.com/higress-group/higress-console/pull/552) \
  **Contributor**: @lcfang \
  **Change Log**: 新增vport属性支持服务虚拟端口配置，扩展V1RegistryConfig和ServiceSource模型，引入VPort类，并在Kubernetes模型转换逻辑中集成vport字段映射，解决注册中心实例端口不一致导致的路由失效问题。 \
  **Feature Value**: 用户可在配置Eureka/Nacos等注册中心时指定服务虚拟端口，确保后端实例端口动态变化时路由规则持续生效，提升服务发现与流量调度的稳定性与兼容性，降低运维配置复杂度。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#620](https://github.com/higress-group/higress-console/pull/620) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了sortWasmPluginMatchRules逻辑中的拼写错误，修正了匹配规则排序时因变量名或逻辑误写导致的潜在排序异常，确保WASM插件匹配规则按预期优先级正确排序。 \
  **Feature Value**: 避免因拼写错误引发的规则排序错乱，保障WASM插件在网关中按用户定义的优先级准确生效，提升流量路由和策略执行的可靠性与一致性。

- **Related PR**: [#619](https://github.com/higress-group/higress-console/pull/619) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了AiRoute转换为ConfigMap时重复保存版本信息的问题，从data JSON中移除version字段，仅保留ConfigMap metadata中的版本信息，避免数据冗余和潜在不一致。 \
  **Feature Value**: 提升了配置一致性与可靠性，防止因版本信息重复导致的解析错误或部署异常，使用户在使用AiRoute功能时获得更稳定的Kubernetes资源管理体验。

- **Related PR**: [#618](https://github.com/higress-group/higress-console/pull/618) \
  **Contributor**: @CH3CHO \
  **Change Log**: 重构SystemController中的API鉴权逻辑，引入AllowAnonymous注解机制，统一处理免认证接口，移除硬编码的路径白名单，增强鉴权规则的可维护性和安全性。 \
  **Feature Value**: 修复了系统控制器中潜在的安全漏洞，防止未授权访问敏感API，提升平台整体安全性；用户无需额外操作即可获得更安全的服务访问保障，降低被恶意利用的风险。

- **Related PR**: [#617](https://github.com/higress-group/higress-console/pull/617) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了前端控制台的三个关键错误：列表渲染缺少唯一key导致的警告、Content Security Policy阻止图片加载、以及Consumer.name字段类型定义错误（由boolean改为string）。 \
  **Feature Value**: 提升了应用稳定性和用户体验，避免控制台报错干扰开发调试，确保头像和列表正确渲染，同时修正数据类型以防止运行时类型错误，增强前端代码健壮性。

- **Related PR**: [#614](https://github.com/higress-group/higress-console/pull/614) \
  **Contributor**: @lc0138 \
  **Change Log**: 修复了ServiceSource类中服务来源type字段的类型定义错误，增加了对字典值的校验逻辑，确保仅允许合法的注册中心类型被设置，避免运行时类型不匹配引发的异常。 \
  **Feature Value**: 提升了服务来源配置的健壮性和安全性，防止因非法type值导致的服务注册失败或系统异常，保障用户在配置多类型服务源时的稳定性与可靠性。

- **Related PR**: [#613](https://github.com/higress-group/higress-console/pull/613) \
  **Contributor**: @lc0138 \
  **Change Log**: 修复前端Content Security Policy（CSP）配置缺陷，通过在Document组件中正确注入meta标签和安全头，防止XSS等客户端脚本注入攻击，提升整体Web应用安全性。 \
  **Feature Value**: 显著降低前端遭受跨站脚本（XSS）等安全攻击的风险，增强用户数据与会话的安全性，保障生产环境合规性，提升平台可信度与用户信任感。

- **Related PR**: [#612](https://github.com/higress-group/higress-console/pull/612) \
  **Contributor**: @zhwaaaaaa \
  **Change Log**: 在DashboardServiceImpl中添加对hop-to-hop头部（如Transfer-Encoding）的忽略逻辑，依据RFC 2616规范，防止反向代理转发chunked编码头导致Grafana页面异常，修复了因HTTP头部透传引发的前端渲染失败问题。 \
  **Feature Value**: 解决了Grafana监控页面无法正常加载的问题，提升了控制台与后端服务集成的稳定性；用户无需手动配置代理过滤规则，开箱即用获得兼容RFC标准的代理行为，改善运维体验和系统可观测性。

- **Related PR**: [#609](https://github.com/higress-group/higress-console/pull/609) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修正了Consumer接口中name字段的类型定义，将其从boolean更正为string，修复了类型不匹配导致的潜在运行时错误和TypeScript编译问题。 \
  **Feature Value**: 确保Consumer.name字段正确表示字符串类型的用户名，提升类型安全性与API一致性，避免前端逻辑误判及数据渲染异常，增强系统稳定性和开发者体验。

- **Related PR**: [#605](https://github.com/higress-group/higress-console/pull/605) \
  **Contributor**: @SaladDay \
  **Change Log**: 修正AI路由名称的正则验证规则，支持点号并限制为小写字母；同步更新中英文错误提示文本，确保界面提示与实际校验逻辑一致，修复了用户输入合法名称时被错误拒绝的问题。 \
  **Feature Value**: 提升AI路由配置体验，使用户可正确使用含点号的路由名称（如api.v1），避免因验证不一致导致的表单提交失败，增强系统可靠性与用户信任度。

### 📚 文档更新 (Documentation)

- **Related PR**: [#611](https://github.com/higress-group/higress-console/pull/611) \
  **Contributor**: @qshuai \
  **Change Log**: 修正了LlmProvidersController中@PostMapping接口的Swagger API文档摘要描述，将错误的'Add a new route'更新为更准确的描述，提升API文档准确性与可读性。 \
  **Feature Value**: 使API文档与实际功能一致，帮助开发者快速理解接口用途，减少误用风险，提升控制台AI服务模块的开发体验和集成效率。

- **Related PR**: [#610](https://github.com/higress-group/higress-console/pull/610) \
  **Contributor**: @heimanba \
  **Change Log**: 修改 frontend-gray 插件文档中 rewrite、backendVersion、enabled 字段为非必填，更新 rules.name 关联路径为 grayDeployments[].name，并同步中英文 README 和 spec.yaml 的字段描述与术语，提升配置说明准确性与一致性。 \
  **Feature Value**: 使灰度配置更灵活兼容不同部署场景，降低用户配置门槛；统一文档术语和关联逻辑，减少因文档歧义导致的配置错误，提升开发者理解和使用插件的效率与可靠性。

---

## 📊 发布统计

- 🚀 新功能: 7项
- 🐛 Bug修复: 9项
- 📚 文档更新: 2项

**总计**: 18项更改

感谢所有贡献者的辛勤付出！🎉


