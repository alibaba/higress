# Higress Console


## 📋 Overview of This Release

This release includes **18** updates, covering feature enhancements, bug fixes, and performance optimizations.

### Distribution of Updates

- **New Features**: 7 items  
- **Bug Fixes**: 9 items  
- **Documentation Updates**: 2 items  

---

## 📝 Full Change Log

### 🚀 New Features (Features)

- **Related PR**: [#621](https://github.com/higress-group/higress-console/pull/621) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: Enhanced MCP Server interaction capabilities: added support for automatic Host header rewriting for DNS backends; improved transport selection and full-path configuration in direct routing scenarios; refined DSN special-character (e.g., `@`) parsing logic for DB-to-MCP Server scenarios. \
  **Feature Value**: Improves flexibility and compatibility when integrating with MCP Servers, enabling users to configure diverse backend types more easily, avoiding path ambiguity and authentication-character parsing failures—thus reducing operational complexity and enhancing system stability.

- **Related PR**: [#608](https://github.com/higress-group/higress-console/pull/608) \
  **Contributor**: @Libres-coder \
  **Change Log**: Added plugin display functionality to the AI Route Management page: extended AI route rows to show enabled plugins, added an "Enabled" label to the configuration page, and implemented frontend i18n support along with plugin-query logic adaptation. \
  **Feature Value**: Enables users to intuitively view and manage enabled plugins directly from the AI Route Management interface, aligning with conventional route functionality—improving transparency and operational consistency of AI route configurations and lowering learning and operational overhead.

- **Related PR**: [#604](https://github.com/higress-group/higress-console/pull/604) \
  **Contributor**: @CH3CHO \
  **Change Log**: Introduced regex-based path rewrite functionality, supporting complex path matching and replacement via the `higress.io/rewrite-target` annotation; extended the rewrite configuration parsing logic in `KubernetesModelConverter`; added regex rewrite type identifiers to frontend and backend i18n resources. \
  **Feature Value**: Empowers users to define fine-grained path rewrite rules using regular expressions, significantly enhancing route matching precision and forwarding flexibility—ideal for multi-version API management, dynamic path mapping, and similar use cases—thereby improving gateway adaptability and operational efficiency.

- **Related PR**: [#603](https://github.com/higress-group/higress-console/pull/603) \
  **Contributor**: @CH3CHO \
  **Change Log**: Added a constant `STATIC_SERVICE_PORT = 80` to the static service source form component and displayed this fixed port in the UI, making it explicit to users that static services default to port 80—improving configuration transparency and consistency. \
  **Feature Value**: Users can clearly see the preconfigured port 80 when configuring static service sources, preventing service deployment failures caused by port misinterpretation—reducing adoption barriers and improving configuration accuracy and operational efficiency.

- **Related PR**: [#602](https://github.com/higress-group/higress-console/pull/602) \
  **Contributor**: @CH3CHO \
  **Change Log**: Added search functionality to the upstream service selector component for AI routes, enabling frontend input-based filtering of the service list to accelerate service selection. Modified the `RouteForm` component to include a search input field and filtering logic—achieving enhanced interactivity with minimal code changes. \
  **Feature Value**: Allows users to rapidly filter target upstream services during AI route configuration—eliminating manual scrolling through large service lists—significantly boosting configuration efficiency and user experience, especially in production environments with extensive microservice inventories.

- **Related PR**: [#566](https://github.com/higress-group/higress-console/pull/566) \
  **Contributor**: @OuterCyrex \
  **Change Log**: Added support for Qwen (Tongyi Qwen) large language model (LLM) services, including custom service endpoints, internet search enablement, and file ID upload capabilities; introduced `QwenLlmProviderHandler`; completed configuration item and i18n adaptations on both frontend and backend. \
  **Feature Value**: Enables convenient web-based configuration of private or third-party Qwen services, expanding AI capability extensibility; internet search and file ID upload support enhance LLM invocation flexibility and practicality across real-world business scenarios.

- **Related PR**: [#552](https://github.com/higress-group/higress-console/pull/552) \
  **Contributor**: @lcfang \
  **Change Log**: Added `vport` attribute support for service virtual port configuration; extended `V1RegistryConfig` and `ServiceSource` models; introduced `VPort` class; integrated `vport` field mapping into Kubernetes model conversion logic—resolving routing failures caused by inconsistent instance ports in service registries. \
  **Feature Value**: Users can now specify virtual ports for services registered via Eureka/Nacos, ensuring routing rules remain effective despite dynamic backend instance port changes—enhancing stability and compatibility of service discovery and traffic scheduling while reducing operational configuration complexity.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#620](https://github.com/higress-group/higress-console/pull/620) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed a typographical error in the `sortWasmPluginMatchRules` logic, correcting variable names or logical errors that could cause unexpected sorting anomalies—ensuring WASM plugin match rules are prioritized and sorted as intended. \
  **Feature Value**: Prevents rule-ordering inconsistencies due to typos, guaranteeing that WASM plugins are applied at the correct priority within the gateway—improving reliability and consistency of traffic routing and policy enforcement.

- **Related PR**: [#619](https://github.com/higress-group/higress-console/pull/619) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed duplicate version information persistence when converting `AiRoute` to `ConfigMap`: removed the `version` field from the data JSON payload, retaining version metadata solely in the `ConfigMap` metadata—eliminating data redundancy and potential inconsistency. \
  **Feature Value**: Enhances configuration consistency and reliability, preventing parsing errors or deployment anomalies caused by duplicated version fields—delivering a more stable Kubernetes resource management experience for `AiRoute` users.

- **Related PR**: [#618](https://github.com/higress-group/higress-console/pull/618) \
  **Contributor**: @CH3CHO \
  **Change Log**: Refactored API authorization logic in `SystemController`, introducing an `@AllowAnonymous` annotation mechanism to uniformly handle unauthenticated endpoints and removing hardcoded path whitelists—improving maintainability and security of authorization policies. \
  **Feature Value**: Resolves potential security vulnerabilities in the system controller, preventing unauthorized access to sensitive APIs and enhancing overall platform security; users benefit from stronger access protection without additional configuration—reducing exposure to malicious exploitation.

- **Related PR**: [#617](https://github.com/higress-group/higress-console/pull/617) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed three critical frontend console issues: missing unique `key` props causing rendering warnings, Content Security Policy (CSP) blocking image loading, and incorrect type definition for `Consumer.name` (corrected from `boolean` to `string`). \
  **Feature Value**: Improves application stability and user experience—eliminating console errors that disrupt development debugging, ensuring proper avatar and list rendering, and fixing data types to prevent runtime type errors—strengthening frontend code robustness.

- **Related PR**: [#614](https://github.com/higress-group/higress-console/pull/614) \
  **Contributor**: @lc0138 \
  **Change Log**: Fixed a type-definition error for the `type` field in the `ServiceSource` class and added dictionary-value validation logic—ensuring only valid registry types are permitted and preventing runtime type mismatches. \
  **Feature Value**: Enhances robustness and security of service-source configuration, preventing service registration failures or system exceptions caused by invalid `type` values—ensuring stability and reliability when configuring heterogeneous service sources.

- **Related PR**: [#613](https://github.com/higress-group/higress-console/pull/613) \
  **Contributor**: @lc0138 \
  **Change Log**: Fixed a frontend Content Security Policy (CSP) configuration defect by correctly injecting meta tags and security headers in the `Document` component—preventing XSS and other client-side script injection attacks and improving overall web application security. \
  **Feature Value**: Significantly reduces the risk of cross-site scripting (XSS) and related security exploits on the frontend, strengthening user data and session security, ensuring production environment compliance, and elevating platform trustworthiness and user confidence.

- **Related PR**: [#612](https://github.com/higress-group/higress-console/pull/612) \
  **Contributor**: @zhwaaaaaa \
  **Change Log**: Added hop-to-hop header (e.g., `Transfer-Encoding`) ignore logic in `DashboardServiceImpl`, adhering to RFC 2616—preventing reverse proxy forwarding of chunked encoding headers from disrupting Grafana dashboard rendering and resolving frontend rendering failures caused by HTTP header passthrough. \
  **Feature Value**: Fixes inability to load Grafana monitoring pages, enhancing stability of console-backend integration; users gain RFC-compliant proxy behavior out-of-the-box—no manual proxy filtering rules required—improving operational experience and system observability.

- **Related PR**: [#609](https://github.com/higress-group/higress-console/pull/609) \
  **Contributor**: @CH3CHO \
  **Change Log**: Corrected the type definition for the `name` field in the `Consumer` interface—from `boolean` to `string`—resolving potential runtime errors and TypeScript compilation issues caused by type mismatch. \
  **Feature Value**: Ensures `Consumer.name` correctly represents a string-typed username, improving type safety and API consistency—preventing front-end logic misinterpretation and data rendering anomalies—and enhancing system stability and developer experience.

- **Related PR**: [#605](https://github.com/higress-group/higress-console/pull/605) \
  **Contributor**: @SaladDay \
  **Change Log**: Refined the regex validation rule for AI route names to permit periods (`.`) and restrict characters to lowercase letters; synchronized Chinese/English error messages to ensure UI prompts accurately reflect actual validation logic—fixing erroneous rejection of valid route names (e.g., `api.v1`). \
  **Feature Value**: Improves AI route configuration experience—enabling correct usage of period-containing names—while eliminating form submission failures due to inconsistent validation—enhancing system reliability and user trust.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#611](https://github.com/higress-group/higress-console/pull/611) \
  **Contributor**: @qshuai \
  **Change Log**: Corrected the Swagger API documentation summary description for the `@PostMapping` endpoint in `LlmProvidersController`, replacing the inaccurate "Add a new route" with a precise functional description—improving API documentation accuracy and readability. \
  **Feature Value**: Aligns API documentation with actual behavior, helping developers quickly grasp endpoint purpose—reducing misuse risk—and improving development experience and integration efficiency for the console’s AI service module.

- **Related PR**: [#610](https://github.com/higress-group/higress-console/pull/610) \
  **Contributor**: @heimanba \
  **Change Log**: Updated `frontend-gray` plugin documentation: marked `rewrite`, `backendVersion`, and `enabled` fields as optional; updated `rules.name` reference path to `grayDeployments[].name`; synchronized field descriptions and terminology across Chinese/English READMEs and `spec.yaml`—enhancing configuration guidance accuracy and consistency. \
  **Feature Value**: Increases flexibility of gray-scale configuration for diverse deployment scenarios—lowering user configuration barriers; standardized documentation terminology and linkage logic—reducing configuration errors stemming from ambiguous documentation—and improving developer understanding and reliable plugin usage.

---

## 📊 Release Statistics

- 🚀 New Features: 7 items  
- 🐛 Bug Fixes: 9 items  
- 📚 Documentation Updates: 2 items  

**Total**: 18 changes  

Thank you to all contributors for your hard work! 🎉

