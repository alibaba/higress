# Higress Console


## 📋 本次发布概览

本次发布包含 **6** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 4项
- **Bug修复**: 2项

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#666](https://github.com/higress-group/higress-console/pull/666) \
  **Contributor**: @johnlanni \
  **Change Log**: 新增插件镜像仓库和命名空间配置项，支持通过环境变量HIGRESS_ADMIN_WASM_PLUGIN_IMAGE_REGISTRY/NAMESPACE动态指定内置插件镜像地址，无需修改plugins.properties；同时在Helm Chart中集成对应values参数与部署模板渲染逻辑。 \
  **Feature Value**: 用户可在不同网络环境（如私有云、离线环境）灵活配置WASM插件镜像源，提升部署灵活性与安全性；降低运维成本，避免硬编码配置带来的维护困难和升级风险。

- **Related PR**: [#665](https://github.com/higress-group/higress-console/pull/665) \
  **Contributor**: @johnlanni \
  **Change Log**: 新增Zhipu AI的Code Plan模式支持和Claude的API版本配置能力，通过扩展ZhipuAILlmProviderHandler和ClaudeLlmProviderHandler实现自定义域名、代码生成优化开关及API版本参数，提升大模型调用灵活性与场景适配性。 \
  **Feature Value**: 用户可基于不同AI厂商特性启用代码专项生成模式（如Zhipu Code Plan）并精确控制Claude API版本，显著提升代码生成质量与兼容性，降低集成门槛，增强AI网关在多模型协同开发场景中的实用性。

- **Related PR**: [#661](https://github.com/higress-group/higress-console/pull/661) \
  **Contributor**: @johnlanni \
  **Change Log**: 为AI统计插件引入轻量模式配置，新增USE_DEFAULT_ATTRIBUTES常量，并在AiRouteServiceImpl中启用use_default_response_attributes: true，减少响应属性采集开销，避免内存缓冲问题。 \
  **Feature Value**: 提升生产环境稳定性与性能，降低AI路由统计的资源消耗；用户无需手动配置复杂属性，系统自动采用默认精简属性集，简化运维并增强高并发场景下的可靠性。

- **Related PR**: [#657](https://github.com/higress-group/higress-console/pull/657) \
  **Contributor**: @liangziccc \
  **Change Log**: PR在路由管理页移除了原有输入框搜索，新增路由名称、域名、路由条件、目标服务、请求授权五个字段的多选下拉筛选，并完成中英文国际化适配，支持多维度组合筛选（各字段内OR、字段间AND），提升数据过滤精准度。 \
  **Feature Value**: 用户可通过直观下拉选择快速定位特定路由，避免手动输入错误；中英文切换支持国际化使用场景；多条件联合筛选显著提升运维人员在海量路由配置中的查询效率和操作体验。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#662](https://github.com/higress-group/higress-console/pull/662) \
  **Contributor**: @johnlanni \
  **Change Log**: 修复了mcp-server插件OCI镜像路径未同步迁移的问题，将原路径mcp-server/all-in-one更新为plugins/mcp-server，适配新插件目录结构，确保插件加载和部署正常。 \
  **Feature Value**: 避免因镜像路径错误导致插件无法拉取或启动失败，保障Higress网关中mcp-server插件的稳定运行与无缝升级，提升用户在插件化场景下的部署可靠性。

- **Related PR**: [#654](https://github.com/higress-group/higress-console/pull/654) \
  **Contributor**: @fgksking \
  **Change Log**: 升级了springdoc依赖的swagger-ui版本，通过在pom.xml中引入更高版本的webjars-lo依赖和更新相关版本属性，修复了Swagger UI中请求体schema显示为空的问题。 \
  **Feature Value**: 用户在使用Higress控制台的API文档功能时，能正确查看和交互请求体结构，提升API调试体验与开发效率，避免因文档展示异常导致的接口调用误解。

---

## 📊 发布统计

- 🚀 新功能: 4项
- 🐛 Bug修复: 2项

**总计**: 6项更改

感谢所有贡献者的辛勤付出！🎉


