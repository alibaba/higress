# Higress


## 📋 Overview of This Release

This release includes **33** updates, covering various aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 14 items
- **Bug Fixes**: 5 items
- **Refactoring and Optimization**: 8 items
- **Documentation Updates**: 5 items
- **Testing Improvements**: 1 item

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#2679](https://github.com/alibaba/higress/pull/2679)
  **Contributor**: erasernoob
  **Change Log**: This PR adds support for external FQDNs by introducing a new configuration option in the mirror annotation and updating the relevant test cases to ensure the correctness of the new feature.
  **Feature Value**: Users can now use external services as mirror targets, enhancing the system's flexibility and scalability, allowing for a wider range of service integration scenarios.

- **Related PR**: [#2667](https://github.com/alibaba/higress/pull/2667)
  **Contributor**: hanxiantao
  **Change Log**: This PR adds the functionality to set global route rate limit thresholds for the AI Token rate limiting plugin and improves the base logic and log prompts for the `cluster-key-rate-limit` and `ai-token-ratelimit` plugins.
  **Feature Value**: The new feature allows users to more flexibly control traffic by setting global rate limit thresholds to prevent overload, improving system stability and availability, and optimizing the user experience.

- **Related PR**: [#2652](https://github.com/alibaba/higress/pull/2652)
  **Contributor**: OxalisCu
  **Change Log**: This PR adds support for first-byte timeout for LLM streaming requests in the AI proxy plugin by introducing the `firstByteTimeout` parameter in the configuration.
  **Feature Value**: This feature enhances the system's ability to handle long non-responsive LLM services, improving user experience and system stability.

- **Related PR**: [#2650](https://github.com/alibaba/higress/pull/2650)
  **Contributor**: zhangjingcn
  **Change Log**: This PR implements the functionality to fetch ErrorResponseTemplate configurations from the Nacos MCP registry by modifying the `mcp_model.go` and `watcher.go` files.
  **Feature Value**: This feature makes it easier for users using the MCP registry to obtain error response templates, thereby enhancing the system's flexibility and maintainability.

- **Related PR**: [#2649](https://github.com/alibaba/higress/pull/2649)
  **Contributor**: CH3CHO
  **Change Log**: This PR introduces support for different formats of Azure OpenAI URLs, including three new URL configuration methods, and ensures that the `api-version` parameter is always required.
  **Feature Value**: This enhancement increases the system's flexibility and compatibility, making it easier for users to configure connections with Azure OpenAI services, supporting more use cases.

- **Related PR**: [#2648](https://github.com/alibaba/higress/pull/2648)
  **Contributor**: daixijun
  **Change Log**: This PR adds support for the `/v1/messages` interface of the Anthropic provider in the qwen Provider by introducing new dependencies and modifying related code logic in the `qwen.go` file.
  **Feature Value**: The new feature expands the capabilities of the qwen Provider, allowing users to process messages using the Anthropic API, enhancing the system's flexibility and applicability.

- **Related PR**: [#2639](https://github.com/alibaba/higress/pull/2639)
  **Contributor**: johnlanni
  **Change Log**: This PR optimizes the request processing flow by disabling route redirection in specific plugins. The main changes involve multiple WASM plugin-related files to ensure that plugins that do not need re-matching do not trigger additional routing logic.
  **Feature Value**: For users using these plugins, this feature can improve API gateway performance, reduce unnecessary resource consumption, and speed up response times, thereby enhancing overall system efficiency.

- **Related PR**: [#2585](https://github.com/alibaba/higress/pull/2585)
  **Contributor**: akolotov
  **Change Log**: This PR adds a configuration file for the Blockscout MCP server, including detailed YAML configuration and corresponding README documentation.
  **Feature Value**: By integrating the Blockscout MCP server, users can more easily monitor and analyze EVM-compatible blockchain networks, enhancing system observability and user experience.

- **Related PR**: [#2551](https://github.com/alibaba/higress/pull/2551)
  **Contributor**: daixijun
  **Change Log**: This PR adds support for the Anthropic and Gemini APIs, including the `anthropic/v1/messages`, `anthropic/v1/complete`, and `gemini/v1beta/generatecontent` interfaces.
  **Feature Value**: By supporting more AI service provider APIs, users can more flexibly choose and integrate different AI functionalities, thereby expanding application capabilities and enhancing user experience.

- **Related PR**: [#2542](https://github.com/alibaba/higress/pull/2542)
  **Contributor**: daixijun
  **Change Log**: This PR adds a token usage statistics feature for the images, audio, and responses interfaces, and defines `UnifySSEChunk` and `GetTokenUsage` as public utility functions to reduce code duplication.
  **Feature Value**: This new feature allows users to better monitor and manage API token usage, especially for multimedia file processing-related interfaces, enhancing system observability and cost control capabilities.

- **Related PR**: [#2537](https://github.com/alibaba/higress/pull/2537)
  **Contributor**: wydream
  **Change Log**: This PR adds text re-ranking support for the Qwen model, introducing a new API endpoint `qwen/v1/rerank` and making corresponding updates in the `provider.go` and `qwen.go` files.
  **Feature Value**: By introducing the Qwen text re-ranking feature, users can now use this new feature for more efficient information retrieval and sorting, enhancing the application's data processing capabilities.

- **Related PR**: [#2535](https://github.com/alibaba/higress/pull/2535)
  **Contributor**: wydream
  **Change Log**: This PR introduces the `basePath` and `basePathHandling` options, allowing flexible handling of request paths. By setting the `removePrefix` or `prepend` mode, the behavior of the base path can be controlled to adapt to different routing needs.
  **Feature Value**: This new feature enables the API gateway to better handle URLs with prefixes, simplifying the backend service's path handling logic and enhancing the flexibility of service configuration and user experience.

- **Related PR**: [#2517](https://github.com/alibaba/higress/pull/2517)
  **Contributor**: cr7258
  **Change Log**: This PR re-implements the Higress API MCP server using golang-filter, adding features such as route management, service discovery, and plugin resource management.
  **Feature Value**: The new feature provides a more flexible way to manage Higress resources, allowing users to more easily manage routes and service origins, enhancing system maintainability and scalability.

- **Related PR**: [#2499](https://github.com/alibaba/higress/pull/2499)
  **Contributor**: heimanba
  **Change Log**: This PR adds the `useManifestAsEntry` configuration option, updates the `GrayConfig` struct, and modifies the related processing logic and documentation. The main changes include adding the `UseManifestAsEntry` field in `GrayConfig`, updating the HTML response handling logic, and updating the README.
  **Feature Value**: This feature allows users to control whether the homepage request is cached through the `useManifestAsEntry` configuration, enhancing the flexibility during gray releases and ensuring that the homepage request is not cached as expected in specific scenarios, thereby ensuring the effectiveness of the gray release strategy.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#2687](https://github.com/alibaba/higress/pull/2687)
  **Contributor**: Thomas-Eliot
  **Change Log**: This PR fixes an SQL error encountered when using the `mcp client` tool's `describeTable` function by introducing the `strings` library.
  **Feature Value**: This fix ensures that users can correctly obtain table structure information, enhancing the stability and reliability of the tool during the migration from Postgres to the MCP Server.

- **Related PR**: [#2662](https://github.com/alibaba/higress/pull/2662)
  **Contributor**: johnlanni
  **Change Log**: This PR fixes a memory leak issue in `proxy-wasm-cpp-host` and a 404 error caused by port mapping mismatches when PPV2 is enabled, by updating the relevant configurations and optimizing the lookup logic.
  **Feature Value**: This resolution addresses resource wastage and service interruption risks due to memory leaks, enhancing system stability and performance; it also ensures correct routing in complex network environments, improving the user experience.

- **Related PR**: [#2656](https://github.com/alibaba/higress/pull/2656)
  **Contributor**: co63oc
  **Change Log**: This PR corrects spelling errors in multiple files, including variable names, function names, interface method names, and documentation, improving code readability and consistency.
  **Feature Value**: By fixing these spelling errors, the code quality and user experience are improved, reducing potential logical errors and runtime exceptions, ensuring system stability and reliability.

- **Related PR**: [#2623](https://github.com/alibaba/higress/pull/2623)
  **Contributor**: Guo-Chenxu
  **Change Log**: This PR fixes issues that may occur when handling special character translations, ensuring that version release notes do not cause JSON format errors due to special characters.
  **Feature Value**: By resolving special character translation issues, the system stability and user experience are improved, ensuring the accuracy and readability of the version release notes.

- **Related PR**: [#2507](https://github.com/alibaba/higress/pull/2507)
  **Contributor**: hongzhouzi
  **Change Log**: This PR corrects an error that occurs when compiling `golang-filter.so` on arm64 architecture machines, ensuring the correct installation of the corresponding architecture toolchain.
  **Feature Value**: This solution addresses compilation issues on specific architectures, ensuring cross-platform compatibility, and enhancing user experience and development efficiency.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#2688](https://github.com/alibaba/higress/pull/2688)
  **Contributor**: johnlanni
  **Change Log**: This PR updates the OSS upload tool in the CI/CD workflow and upgrades the version number from v2.1.5 to v2.1.6, with minor adjustments to the Makefile.
  **Feature Value**: By updating the tools and version numbers in the CI/CD process, the project's maintainability and consistency are improved, ensuring that users can access the latest stable versions.

- **Related PR**: [#2673](https://github.com/alibaba/higress/pull/2673)
  **Contributor**: johnlanni
  **Change Log**: This PR improves the `findEndpointUrl` function to handle multiple SSE messages, supporting continued processing of subsequent messages even when non-`endpoint` events are encountered.
  **Feature Value**: This enhancement increases the robustness and flexibility of the MCP endpoint parser, ensuring that `endpoint` information can be correctly parsed even after receiving other types of messages, enhancing system stability and user experience.

- **Related PR**: [#2661](https://github.com/alibaba/higress/pull/2661)
  **Contributor**: johnlanni
  **Change Log**: This PR adjusts the DNS service domain name validation regular expression, relaxing the domain format restrictions.
  **Feature Value**: By relaxing the domain validation rules, the system's flexibility and compatibility are increased, allowing a wider variety of domain names to be accepted, providing users with a broader range of use cases.

- **Related PR**: [#2615](https://github.com/alibaba/higress/pull/2615)
  **Contributor**: johnlanni
  **Change Log**: This PR removes some unnecessary variables and configurations in the build process of wasm-go-related plugins, simplifying the Dockerfile, Makefile, and related extension files.
  **Feature Value**: By cleaning up redundant code, the project remains tidy, making maintenance easier and reducing potential error sources. For users, this helps to improve system stability and maintainability.

- **Related PR**: [#2600](https://github.com/alibaba/higress/pull/2600)
  **Contributor**: johnlanni
  **Change Log**: This PR updates the Go version in the wasm-go build image to 1.24.4 and removes some comments and outdated information from the DockerfileBuilder.
  **Feature Value**: By upgrading the Go version, the security and stability of the build process are improved, ensuring developers can use the latest features and technical improvements.

- **Related PR**: [#2598](https://github.com/alibaba/higress/pull/2598)
  **Contributor**: johnlanni
  **Change Log**: This PR updates the Go version in the wasm-go builder image to 1.24.4 and removes descriptions of support for specific architectures from the Dockerfile.
  **Feature Value**: By upgrading the Go version and simplifying the Dockerfile content, the consistency and stability of the build environment are enhanced, indirectly promoting the performance and compatibility of WASM plugins developed using this builder.

- **Related PR**: [#2564](https://github.com/alibaba/higress/pull/2564)
  **Contributor**: rinfx
  **Change Log**: This PR optimizes the location of the request count logic to ensure correct handling even in exceptional cases; it also improves the Redis Lua script logic, including fixing string comparison issues and configuration parameter judgment errors.
  **Feature Value**: By moving the minimum request count -1 logic to `streamdone` and fixing type conversion and logical errors in the Lua script, the system's stability and accuracy are improved, reducing potential runtime errors.

- **Related PR**: [#2532](https://github.com/alibaba/higress/pull/2532)
  **Contributor**: erasernoob
  **Change Log**: This PR migrates the WASM Go plugins to the new SDK and Go 1.24, updating the CI/CD configuration files and other related code to ensure compatibility and correct builds.
  **Feature Value**: By upgrading the Go version and SDK, the performance and stability of the plugins are improved, laying the foundation for future feature development and reducing potential compile-time and runtime errors.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2675](https://github.com/alibaba/higress/pull/2675)
  **Contributor**: Aias00
  **Change Log**: This PR fixes four invalid links in the documentation files, ensuring that users can access the correct external resources.
  **Feature Value**: By fixing broken links, the accuracy and usability of the documentation are improved, allowing developers to more smoothly obtain the necessary information, thereby enhancing the user experience.

- **Related PR**: [#2668](https://github.com/alibaba/higress/pull/2668)
  **Contributor**: Aias00
  **Change Log**: This PR updates the README file for the Rust Wasm plugin development framework, providing detailed development guidelines, including environment requirements, build steps, and testing methods.
  **Feature Value**: This update greatly improves the project's maintainability and ease of use, enabling new developers to quickly get started and understand how to develop Wasm plugins using Rust.

- **Related PR**: [#2647](https://github.com/alibaba/higress/pull/2647)
  **Contributor**: Guo-Chenxu
  **Change Log**: This PR adds a new contributors list and a complete changelog, and adds a forced line break function in markdown to improve the organization and readability of the documentation.
  **Feature Value**: By adding a new contributors list and a complete changelog, the transparency of the project and recognition of community members are enhanced; the documentation format is also optimized, making the information more clearly and easily readable.

- **Related PR**: [#2635](https://github.com/alibaba/higress/pull/2635)
  **Contributor**: github-actions[bot]
  **Change Log**: This PR adds release notes in both Chinese and English for version 2.1.5, summarizing 41 updates, covering new features, bug fixes, and performance optimizations.
  **Feature Value**: By providing detailed release notes, users can understand the specific content and improvements of the version update, enhancing the user experience and system transparency.

- **Related PR**: [#2586](https://github.com/alibaba/higress/pull/2586)
  **Contributor**: erasernoob
  **Change Log**: This PR updates the README files related to the wasm-go plugins, removing TinyGo-related configurations and upgrading the Go version requirement from 1.18 to 1.24 to support wasm build features.
  **Feature Value**: By updating the documentation, users can obtain the latest compilation environment requirements, avoiding compilation failures due to outdated tool versions, thereby improving development efficiency and experience.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#2596](https://github.com/alibaba/higress/pull/2596)
  **Contributor**: Guo-Chenxu
  **Change Log**: This PR introduces a new GitHub Action workflow to automatically generate and submit release notes when a new version is released. By setting the necessary secrets, the security and reliability of the automation process are ensured.
  **Feature Value**: This feature greatly simplifies the version release process, reducing the manual work of writing and submitting release notes, improving the team's efficiency, and ensuring the consistency and accuracy of the documentation.

---

## 📊 Release Statistics

- 🚀 New Features: 14 items
- 🐛 Bug Fixes: 5 items
- ♻️ Refactoring and Optimization: 8 items
- 📚 Documentation Updates: 5 items
- 🧪 Testing Improvements: 1 item

**Total**: 33 changes

Thank you to all the contributors for their hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **12** updates, covering feature enhancements, bug fixes, and performance optimizations.

### Update Content Distribution

- **New Features**: 6 items
- **Bug Fixes**: 5 items
- **Refactoring and Optimization**: 1 item

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#562](https://github.com/higress-group/higress-console/pull/562)
  **Contributor**: CH3CHO
  **Change Log**: This PR implements the functionality to configure multiple routes within a single route or AI route by modifying the backend services and frontend components.
  **Feature Value**: It allows users to configure multiple routes under one route or AI route, simplifying route management in complex scenarios, improving flexibility and user experience.

- **Related PR**: [#560](https://github.com/higress-group/higress-console/pull/560)
  **Contributor**: Erica177
  **Change Log**: Added JSON Schema for multiple plugins, including AI agent, caching, data masking, history, and intent recognition, defining detailed configuration properties.
  **Feature Value**: By introducing JSON Schema, it enhances the plugin configuration management capability, allowing users to intuitively understand and set plugin parameters, improving the accuracy and ease of use of configurations.

- **Related PR**: [#555](https://github.com/higress-group/higress-console/pull/555)
  **Contributor**: hongzhouzi
  **Change Log**: Added execution, list display, table, and tool configuration features for DB MCP Server, ensuring that the configurations displayed in the console are consistent with those in higress-gateway.
  **Feature Value**: Users can now view and manage detailed configuration information of the DB MCP Server through the console, enhancing system maintainability and consistency.

- **Related PR**: [#550](https://github.com/higress-group/higress-console/pull/550)
  **Contributor**: CH3CHO
  **Change Log**: This PR updated the AI routing configuration logic to ensure correct synchronization of routing configurations when updating specific types of LLM providers.
  **Feature Value**: With this feature update, users can more accurately manage their AI service routing configurations, especially after changing LLM provider types, ensuring consistency and availability of service names.

- **Related PR**: [#547](https://github.com/higress-group/higress-console/pull/547)
  **Contributor**: CH3CHO
  **Change Log**: Enhanced user control over configuration changes by introducing undo/redo functionality on the system configuration page, mainly modifying the CodeEditor component and related page logic.
  **Feature Value**: The new undo/redo feature allows users to easily roll back or restore configuration changes, improving user experience and reducing the risk of errors.

- **Related PR**: [#543](https://github.com/higress-group/higress-console/pull/543)
  **Contributor**: erasernoob
  **Change Log**: This PR upgraded the plugin version from 1.0.0 to 2.0.0, involving updates to the `plugins.properties` file.
  **Feature Value**: Upgrading the plugin version to 2.0.0 enables users to enjoy performance improvements and enhanced features, improving overall system stability and user experience.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#559](https://github.com/higress-group/higress-console/pull/559)
  **Contributor**: KarlManong
  **Change Log**: This PR corrected the line endings format in project files, ensuring all non-binary and cmd files end with LF, improving code consistency and cross-platform compatibility.
  **Feature Value**: Standardizing line endings to LF helps avoid issues caused by different operating systems, enhancing project maintainability and user experience.

- **Related PR**: [#554](https://github.com/higress-group/higress-console/pull/554)
  **Contributor**: CH3CHO
  **Change Log**: Fixed two UI issues in the LLM provider management module: 1. Missing scheme for Google Vertex service endpoint; 2. Ensured form state is reset after canceling the addition of a new provider.
  **Feature Value**: Fixing these UI issues improves the user experience when managing LLM providers, reducing operational errors due to interface problems and enhancing system usability and stability.

- **Related PR**: [#549](https://github.com/higress-group/higress-console/pull/549)
  **Contributor**: CH3CHO
  **Change Log**: This PR ensures that the latest plugin configuration is always loaded when opening the configuration edit drawer, achieved by updating a few lines of code in specific files.
  **Feature Value**: It fixed the issue of potentially displaying outdated plugin configurations, ensuring that users see the latest settings each time they view or edit, improving user experience consistency and accuracy.

- **Related PR**: [#548](https://github.com/higress-group/higress-console/pull/548)
  **Contributor**: CH3CHO
  **Change Log**: This PR fixed the issue of leading and trailing spaces in the Wasm image URL before submission, ensuring the URL format is correct by removing extra spaces.
  **Feature Value**: This fix improves system robustness and user input fault tolerance, ensuring successful submission even if users accidentally add spaces to the URL.

- **Related PR**: [#544](https://github.com/higress-group/higress-console/pull/544)
  **Contributor**: CH3CHO
  **Change Log**: Fixed the incorrect error message displayed when enabling authentication but not selecting a consumer, by updating translation files and adjusting component code to ensure accurate error messages.
  **Feature Value**: It improves the accuracy of information in the user interface, avoiding confusion caused by misleading error messages, enhancing user experience.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#551](https://github.com/higress-group/higress-console/pull/551)
  **Contributor**: JayLi52
  **Change Log**: Removed the disabled state for host and port fields in database configuration, changed the default API gateway URL from https to http, and updated the API gateway URL display logic in the MCP details page.
  **Feature Value**: These changes improve system flexibility, allowing users to customize database connection information and ensure consistency and compatibility by simplifying the URL protocol.

---

## 📊 Release Statistics

- 🚀 New Features: 6 items
- 🐛 Bug Fixes: 5 items
- ♻️ Refactoring and Optimization: 1 item

**Total**: 12 changes

Thank you to all contributors for your hard work! 🎉

