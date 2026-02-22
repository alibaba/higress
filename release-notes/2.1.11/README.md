# Higress Console


## 📋 Overview of This Release

This release includes **6** updates, covering feature enhancements, bug fixes, and performance optimizations.

### Distribution of Updates

- **New Features**: 4  
- **Bug Fixes**: 2  

---

## 📝 Full Change Log

### 🚀 New Features

- **Related PR**: [#666](https://github.com/higress-group/higress-console/pull/666) \
  **Contributor**: @johnlanni \
  **Change Log**: Added configuration options for the plugin image registry and namespace. Supports dynamically specifying built-in WASM plugin image addresses via environment variables `HIGRESS_ADMIN_WASM_PLUGIN_IMAGE_REGISTRY`/`NAMESPACE`, eliminating the need to modify `plugins.properties`. Corresponding Helm Chart `values` parameters and deployment template rendering logic have also been integrated. \
  **Feature Value**: Enables users to flexibly configure WASM plugin image sources across diverse network environments (e.g., private cloud, air-gapped environments), improving deployment flexibility and security; reduces operational overhead and mitigates maintenance difficulties and upgrade risks associated with hard-coded configurations.

- **Related PR**: [#665](https://github.com/higress-group/higress-console/pull/665) \
  **Contributor**: @johnlanni \
  **Change Log**: Added support for Zhipu AI’s Code Plan mode and Claude’s API version configuration. Achieved by extending `ZhipuAILlmProviderHandler` and `ClaudeLlmProviderHandler` to support custom domains, code-generation optimization toggles, and API version parameters—enhancing LLM invocation flexibility and scenario adaptability. \
  **Feature Value**: Allows users to enable model-specific code generation modes (e.g., Zhipu Code Plan) based on AI vendor characteristics and precisely control Claude API versions, significantly improving code generation quality and compatibility, lowering integration barriers, and strengthening the practicality of the AI Gateway in multi-model collaborative development scenarios.

- **Related PR**: [#661](https://github.com/higress-group/higress-console/pull/661) \
  **Contributor**: @johnlanni \
  **Change Log**: Introduced a lightweight mode configuration for the AI statistics plugin. Added the `USE_DEFAULT_ATTRIBUTES` constant and enabled `use_default_response_attributes: true` in `AiRouteServiceImpl`, reducing response attribute collection overhead and preventing memory buffer issues. \
  **Feature Value**: Improves production environment stability and performance while lowering resource consumption of AI route statistics; eliminates the need for manual configuration of complex attributes—the system automatically adopts a default, streamlined attribute set—simplifying operations and enhancing reliability under high-concurrency workloads.

- **Related PR**: [#657](https://github.com/higress-group/higress-console/pull/657) \
  **Contributor**: @liangziccc \
  **Change Log**: Removed the original text-input search from the Route Management page and introduced multi-select dropdown filters for five fields: Route Name, Domain, Route Conditions, Destination Service, and Request Authorization. Completed Chinese–English internationalization support and implemented multi-dimensional composite filtering (OR within each field, AND across fields), significantly improving data filtering precision. \
  **Feature Value**: Enables users to quickly locate specific routes via intuitive dropdown selection, avoiding input errors; bilingual support accommodates international usage scenarios; multi-condition combined filtering substantially boosts query efficiency and operational experience for SREs managing large-scale route configurations.

### 🐛 Bug Fixes

- **Related PR**: [#662](https://github.com/higress-group/higress-console/pull/662) \
  **Contributor**: @johnlanni \
  **Change Log**: Fixed an issue where the OCI image path for the `mcp-server` plugin was not migrated synchronously—updated the original path `mcp-server/all-in-one` to `plugins/mcp-server` to align with the new plugin directory structure, ensuring correct plugin loading and deployment. \
  **Feature Value**: Prevents plugin pull or startup failures caused by incorrect image paths, guaranteeing stable operation and seamless upgrades of the `mcp-server` plugin within the Higress Gateway, thereby enhancing deployment reliability in plugin-driven use cases.

- **Related PR**: [#654](https://github.com/higress-group/higress-console/pull/654) \
  **Contributor**: @fgksking \
  **Change Log**: Upgraded the `swagger-ui` version dependency of `springdoc` by introducing a newer version of the `webjars-lo` dependency in `pom.xml` and updating related version properties, resolving an issue where request body schemas appeared empty in Swagger UI. \
  **Feature Value**: Ensures users can correctly view and interact with request body structures when using the API documentation functionality in the Higress Console, improving API debugging experience and development efficiency—and preventing interface misinterpretations caused by documentation display anomalies.

---

## 📊 Release Statistics

- 🚀 New Features: 4  
- 🐛 Bug Fixes: 2  

**Total**: 6 changes  

Thank you to all contributors for your hard work! 🎉

