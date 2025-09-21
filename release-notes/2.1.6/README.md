# Higress


## 📋 Overview of This Release

This release includes **31** updates, covering enhancements, bug fixes, performance optimizations, and more.

### Update Distribution

- **New Features**: 13 items
- **Bug Fixes**: 5 items
- **Refactoring and Optimization**: 7 items
- **Documentation Updates**: 5 items
- **Testing Improvements**: 1 item

### ⭐ Key Highlights

This release contains **2** major updates, which are highly recommended to focus on:

- **feat: Add Higress API MCP server** ([#2517](https://github.com/alibaba/higress/pull/2517)): The newly added Higress API MCP server functionality enhances AI Agent's management capabilities over Higress resources, supporting the creation, deletion, modification, and querying of routes and services through MCP, thereby improving the system's flexibility and maintainability.
- **Migrate WASM Go Plugins to New SDK and Go 1.24** ([#2532](https://github.com/alibaba/higress/pull/2532)): The underlying compilation dependency for developing Wasm Go plugins has been switched from TinyGo to native Go 1.24, improving plugin compatibility and performance, ensuring alignment with the latest technology stack, and providing users with more stable and efficient plugin support.

For more details, please refer to the detailed description of key features below.

---

## 🌟 Detailed Description of Key Features

Below are the detailed descriptions of the important features and improvements in this release:

### 1. feat: Add Higress API MCP server

**Related PR**: [#2517](https://github.com/alibaba/higress/pull/2517) | **Contributor**: [@cr7258](https://github.com/cr7258)

**Usage Background**

In modern microservice architectures, the API gateway, as the entry point, requires flexible and powerful configuration management capabilities. Higress, as a high-performance API gateway, provides rich features for managing routes, service origins, and plugins. However, the existing configuration management methods may not be flexible enough to meet complex operational needs. To address this issue, PR #2517 introduces the Higress API MCP Server, providing a new way to manage configurations through the Higress Console API. This feature is primarily aimed at operations personnel and developers who need advanced and dynamic management of Higress.

**Feature Details**

This change implements the Higress API MCP Server, re-implementing an MCP server using golang-filter that can call the Higress Console API to manage routes, service origins, and plugins. The specific implementation includes:
1. Added the HigressClient class to handle interactions with the Higress Console API.
2. Implemented various management tools such as route management (list-routes, get-route, add-route, update-route), service origin management (list-service-sources, get-service-source, add-service-source, update-service-source), and plugin management (get-plugin, delete-plugin, update-request-block-plugin).
3. Modified relevant configuration files and README documentation, providing detailed configuration examples and usage instructions.
4. Code changes involve multiple files, including `config.go`, `client.go`, `server.go`, etc., ensuring the completeness and extensibility of the feature.

**Usage Instructions**

To enable and configure the Higress API MCP Server, follow these steps:
1. Add the MCP Server configuration in the Higress ConfigMap, specifying the URL, username, and password of the Higress Console.
2. When starting the Higress Gateway, ensure that `mcpServer.enable` is set to `true`.
3. Use the provided tool commands (e.g., `list-routes`, `add-route`) to manage routes, service origins, and plugins.
4. Configuration example:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: higress-config
  namespace: higress-system
data:
  higress: |-
    mcpServer:
      sse_path_suffix: /sse
      enable: true
      servers:
        - name: higress-api-mcp-server
          path: /higress-api
          type: higress-api
          config:
            higressURL: http://higress-console.higress-system.svc.cluster.local
            username: admin
            password: <password>
```
Notes:
- Ensure that the Higress Console URL, username, and password are correct.
- It is recommended to use environment variables or encrypted storage for the password to enhance security.

**Feature Value**

The Higress API MCP Server brings the following specific benefits to users:
1. **Improved Operational Efficiency**: Through a unified MCP interface, users can more conveniently manage and configure Higress resources via AI Agent, reducing the complexity and error rate of manual operations.
2. **Enhanced System Flexibility**: Support for dynamic management and updating of routes, service origins, and plugins makes the system more flexible and able to quickly respond to changes in business requirements.
3. **Increased System Stability**: Automated configuration management reduces the possibility of human errors, thereby enhancing the stability and reliability of the system.
4. **Easy Integration**: The design of the Higress API MCP Server makes it easy to integrate with other AI agents and tools, facilitating the construction of a complete automated operations system.

---

### 2. Migrate WASM Go Plugins to New SDK and Go 1.24

**Related PR**: [#2532](https://github.com/alibaba/higress/pull/2532) | **Contributor**: [@erasernoob](https://github.com/erasernoob)

**Usage Background**

With the development of the Go language, new versions provide many performance optimizations and security improvements. This PR aims to migrate WASM Go plugins from the old SDK to the new SDK and upgrade the Go version to 1.24. This not only resolves some known issues in the old version but also paves the way for future feature expansion and performance optimization. The target user group includes developers and operations personnel using Higress for microservice management and traffic control.

**Feature Details**

This PR mainly implements the following features: 1) Updated the workflow files for building and testing plugins to support the new Go version; 2) Modified the Dockerfile and Makefile, removing support for TinyGo and switching to the standard Go compiler for generating WASM files; 3) Updated the go.mod file, referencing new package paths and versions; 4) Adjusted the import path of the logging library, unifying the use of the new logging library. These changes allow the plugins to better utilize the new features of Go 1.24, such as improved garbage collection and more efficient compiler optimizations. Additionally, removing support for TinyGo simplifies the build process and reduces potential compatibility issues.

**Usage Instructions**

To enable and configure this feature, first ensure that your development environment has Go 1.24 installed. Then, you can specify the new build parameters by modifying the project's Makefile and Dockerfile. For example, set `GO_VERSION ?= 1.24.4` in the Makefile and use `ARG BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.4-oras1.0.0` in the Dockerfile. A typical use case is when you need to deploy new WASM plugins in Higress. Best practices include regularly updating dependencies to the latest versions and ensuring that all related code is adapted to the new version.

**Feature Value**

This refactoring brings multiple benefits to users: 1) Improved plugin runtime efficiency and stability, thanks to the new features and optimizations in Go 1.24; 2) Simplified build process, reducing dependency on third-party tools (such as TinyGo) and lowering maintenance costs; 3) Unified code style and dependency management, improving the readability and maintainability of the project; 4) Enhanced system security by adopting the latest Go version to fix known security vulnerabilities. These improvements make the Higress ecosystem more robust, providing a more powerful and reliable microservice management platform for users.

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#2679](https://github.com/alibaba/higress/pull/2679) \
  **Contributor**: @erasernoob \
  **Change Log**: This PR adds support for external service FQDN in image annotations and includes corresponding test cases to ensure the correctness and stability of the new feature. \
  **Feature Value**: Allows users to specify external FQDN as the image target, enhancing the system's flexibility and applicability, and facilitating the integration of more external resources.

- **Related PR**: [#2667](https://github.com/alibaba/higress/pull/2667) \
  **Contributor**: @hanxiantao \
  **Change Log**: This PR adds support for setting a global route rate limit threshold for the AI Token rate-limiting plugin, while optimizing the underlying logic related to the cluster-key-rate-limit plugin and improving log messages. \
  **Feature Value**: By adding support for global rate limit thresholds, users can more flexibly manage traffic, avoiding the impact of a single route's excessive traffic on the entire system's stability.

- **Related PR**: [#2652](https://github.com/alibaba/higress/pull/2652) \
  **Contributor**: @OxalisCu \
  **Change Log**: This PR adds support for the first-byte timeout for LLM streaming requests in the ai-proxy plugin by modifying the provider.go file. \
  **Feature Value**: This feature allows users to set a first-byte timeout for LLM streaming requests, improving system stability and user experience.

- **Related PR**: [#2650](https://github.com/alibaba/higress/pull/2650) \
  **Contributor**: @zhangjingcn \
  **Change Log**: This PR implements the functionality to fetch ErrorResponseTemplate configuration from the Nacos MCP registry by modifying the mcp_model.go and watcher.go files to support new metadata handling. \
  **Feature Value**: This feature enhances the integration with the Nacos MCP registry, allowing the use of custom response templates in case of errors, thus improving error handling flexibility and user experience.

- **Related PR**: [#2649](https://github.com/alibaba/higress/pull/2649) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR adds support for three different URL formats for Azure OpenAI and ensures that the `api-version` parameter is always required. The changes involve modifying and adding code in several Go files, including request header and path parsing. \
  **Feature Value**: This enhancement improves the integration capability of the plugin with Azure OpenAI services, allowing users to deploy their models with more diverse URL configurations, thereby enhancing system flexibility and compatibility.

- **Related PR**: [#2648](https://github.com/alibaba/higress/pull/2648) \
  **Contributor**: @daixijun \
  **Change Log**: This PR implements support for the qwen Provider for the anthropic /v1/messages interface by adding the relevant code logic in the qwen.go file. \
  **Feature Value**: Adds support for the Anthropic message interface, enabling users to proxy more artificial intelligence services through Qwen, thus expanding the system's application scope and functionality.

- **Related PR**: [#2585](https://github.com/alibaba/higress/pull/2585) \
  **Contributor**: @akolotov \
  **Change Log**: This PR provides a configuration file for the Blockscout MCP server, including detailed README documentation and YAML format configuration settings. \
  **Feature Value**: By integrating the Blockscout MCP server, users can more conveniently inspect and analyze EVM-compatible blockchains, enhancing the system's functionality and user experience.

- **Related PR**: [#2551](https://github.com/alibaba/higress/pull/2551) \
  **Contributor**: @daixijun \
  **Change Log**: This PR adds support for the Anthropic and Gemini APIs in the AI proxy plugin, expanding the system's ability to handle AI requests from different sources. \
  **Feature Value**: By introducing new API support, users can more flexibly choose different AI service providers, enhancing the system's diversity and availability.

- **Related PR**: [#2542](https://github.com/alibaba/higress/pull/2542) \
  **Contributor**: @daixijun \
  **Change Log**: This PR adds token usage statistics for images, audio, and responses interfaces and defines related utility functions as public to reduce code duplication. \
  **Feature Value**: By supporting token usage statistics for more interfaces, users can more comprehensively understand and manage resource consumption, thereby optimizing cost control.

- **Related PR**: [#2537](https://github.com/alibaba/higress/pull/2537) \
  **Contributor**: @wydream \
  **Change Log**: This PR adds support for the Qwen model's text reordering feature in the AI proxy plugin by introducing a new API path. \
  **Feature Value**: The new Qwen text reordering feature expands the platform's text processing capabilities, allowing users to leverage more advanced models for content optimization and sorting.

- **Related PR**: [#2535](https://github.com/alibaba/higress/pull/2535) \
  **Contributor**: @wydream \
  **Change Log**: This PR introduces the `basePath` and `basePathHandling` options for flexible handling of request paths. Users can decide how to use `basePath` by setting `removePrefix` or `prepend`. \
  **Feature Value**: The new options allow users to more flexibly manage the path mapping between the API gateway and backend services, enhancing the system's adaptability and flexibility.

- **Related PR**: [#2499](https://github.com/alibaba/higress/pull/2499) \
  **Contributor**: @heimanba \
  **Change Log**: This PR introduces the `UseManifestAsEntry` field in the GrayConfig structure, updates the relevant functions to support this configuration, and modifies the README documentation and HTML response handling logic. \
  **Feature Value**: The new `useManifestAsEntry` configuration option allows users to more flexibly control whether to use caching for homepage requests, thereby enhancing the system's flexibility and user experience.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#2687](https://github.com/alibaba/higress/pull/2687) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: Fixed an SQL error that occurred when the mcp client used the describeTable tool, ensuring the correctness of the Postgres table description function. \
  **Feature Value**: This fix improves the system's stability and reliability, ensuring that users can accurately obtain table information when interacting with the mcp-server and Postgres database, enhancing the user experience.

- **Related PR**: [#2662](https://github.com/alibaba/higress/pull/2662) \
  **Contributor**: @johnlanni \
  **Change Log**: Resolved two issues in Envoy: fixed a memory leak in proxy-wasm-cpp-host and a 404 error caused by incorrect port mapping when ppv2 was enabled. \
  **Feature Value**: By fixing the memory leak and port mapping issues, the system's stability and reliability are improved, reducing resource waste and ensuring correct routing configuration.

- **Related PR**: [#2656](https://github.com/alibaba/higress/pull/2656) \
  **Contributor**: @co63oc \
  **Change Log**: This PR corrects spelling errors in multiple files, including constant names, function names, and plugin names, ensuring code consistency and readability. \
  **Feature Value**: By fixing these spelling errors, the code quality is improved, avoiding potential logical errors or compilation failures due to inconsistent naming, and enhancing the system's stability and user experience.

- **Related PR**: [#2623](https://github.com/alibaba/higress/pull/2623) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: Fixed an issue caused by special characters during translation by adjusting the method of generating and processing JSON data to avoid potential JSON structure corruption. \
  **Feature Value**: This fix ensures that content containing special characters can be correctly processed and displayed, thereby improving the system's stability and user experience.

- **Related PR**: [#2507](https://github.com/alibaba/higress/pull/2507) \
  **Contributor**: @hongzhouzi \
  **Change Log**: Corrected an error when compiling golang-filter.so on arm64 architecture due to the installation of x86 toolchains, by ensuring the installation of tools matching the target architecture. \
  **Feature Value**: This fix resolves the compilation issue on specific hardware architectures (arm64), allowing the project to be successfully built on a wider range of processors, increasing software compatibility and user base.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#2673](https://github.com/alibaba/higress/pull/2673) \
  **Contributor**: @johnlanni \
  **Change Log**: Improved the `findEndpointUrl` function to handle multiple SSE messages, not just the first one. This involved optimizing the code logic and adding new unit tests. \
  **Feature Value**: This enhancement strengthens the MCP endpoint parser's functionality, making it more robust and better compatible with different message formats sent by backend services, improving the system's stability and user experience.

- **Related PR**: [#2661](https://github.com/alibaba/higress/pull/2661) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR relaxes the DNS service domain validation rules by modifying the regular expression to allow more flexible domain formats. \
  **Feature Value**: Relaxing domain validation helps improve the system's flexibility and compatibility, allowing users to use more diverse domain configurations, thereby enhancing the user experience.

- **Related PR**: [#2639](https://github.com/alibaba/higress/pull/2639) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR optimizes the request processing flow by disabling rerouting in specific plugins. Specifically, it sets `ctx.DisableReroute` in official plugins that do not require route re-matching. \
  **Feature Value**: This optimization improves the performance of the plugins, reducing unnecessary route redirections and enhancing the overall efficiency and response speed of the application, providing a smoother experience for users.

- **Related PR**: [#2615](https://github.com/alibaba/higress/pull/2615) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR removes the EXTRA_TAGS variable from the Dockerfile and Makefile of the wasm-go plugin and updates the relevant configuration files, simplifying the build process. \
  **Feature Value**: By cleaning up unused configuration items, this change makes the project structure more concise and clear, helping to reduce potential maintenance costs while maintaining the stability of existing features.

- **Related PR**: [#2598](https://github.com/alibaba/higress/pull/2598) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR updates the Go version in the WASM builder image to 1.24.4 and simplifies the contents of the DockerfileBuilder file. \
  **Feature Value**: By upgrading the Go version and cleaning up unnecessary code, the performance and security of the build environment are improved, allowing users to take advantage of the latest Go language features and security fixes.

- **Related PR**: [#2564](https://github.com/alibaba/higress/pull/2564) \
  **Contributor**: @rinfx \
  **Change Log**: Optimized the location of the minimum request count logic, moving it to streamdone, and improved the counting comparison logic in the Redis Lua script. \
  **Feature Value**: This improvement enhances the system's stability and accuracy under abnormal conditions, ensuring the correct implementation of request counting and load balancing strategies, and improving the user experience.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2675](https://github.com/alibaba/higress/pull/2675) \
  **Contributor**: @Aias00 \
  **Change Log**: Fixed some broken links in the project documentation, ensuring that users can access the correct links, improving the usability and accuracy of the documentation. \
  **Feature Value**: By fixing the broken links in the documentation, users can more easily find and use the relevant resources, enhancing the user experience and overall quality of the documentation.

- **Related PR**: [#2668](https://github.com/alibaba/higress/pull/2668) \
  **Contributor**: @Aias00 \
  **Change Log**: Improved the README documentation for the Rust plugin development framework, adding a detailed development guide, including environment requirements, build steps, and testing methods. \
  **Feature Value**: This improvement enhances the maintainability and usability of the project, allowing new developers to get started quickly and better understand and use the Rust Wasm plugin development framework.

- **Related PR**: [#2647](https://github.com/alibaba/higress/pull/2647) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: This PR adds the New Contributors and full changelog sections and introduces markdown forced line breaks to improve the readability and completeness of the documentation. \
  **Feature Value**: By adding the contributor list and full changelog, as well as improving the Markdown format, the project documentation becomes clearer and more readable, making it easier for users to understand the latest updates and contributors' contributions.

- **Related PR**: [#2635](https://github.com/alibaba/higress/pull/2635) \
  **Contributor**: @github-actions[bot] \
  **Change Log**: This PR adds detailed release notes for Higress version 2.1.5, including new features, bug fixes, and performance optimizations. \
  **Feature Value**: By providing detailed release information, users can better understand the new features and improvements of Higress, enabling them to use the software more effectively.

- **Related PR**: [#2586](https://github.com/alibaba/higress/pull/2586) \
  **Contributor**: @erasernoob \
  **Change Log**: Updated the README file for the wasm-go plugin, removed TinyGo-related configurations, adjusted the Go version requirement to 1.24 or higher to support WASM build features, and cleaned up unused code paths. \
  **Feature Value**: By updating the documentation and environment configuration requirements, developers can correctly set up their development environment to compile the wasm-go plugin, avoiding issues caused by incompatible language versions or dependencies.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#2596](https://github.com/alibaba/higress/pull/2596) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: This PR adds a new GitHub Actions workflow file to automatically generate and submit a PR for release notes during the release process. The process is based on the higress-report-agent. \
  **Feature Value**: This feature greatly simplifies the documentation maintenance work during the release process, improves the team's efficiency, and ensures that each version release has detailed change records for users to reference.

---

## 📊 Release Statistics

- 🚀 New Features: 13 items
- 🐛 Bug Fixes: 5 items
- ♻️ Refactoring and Optimization: 7 items
- 📚 Documentation Updates: 5 items
- 🧪 Testing Improvements: 1 item

**Total**: 31 changes (including 2 major updates)

Thank you to all the contributors for their hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **12** updates, covering various aspects such as feature enhancements, bug fixes, and performance optimizations.

### Distribution of Updates

- **New Features**: 6
- **Bug Fixes**: 5
- **Refactoring and Optimization**: 1

---

## 📝 Complete Changelog

### 🚀 New Features (Features)

- **Related PR**: [#562](https://github.com/higress-group/higress-console/pull/562) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR implements the functionality to configure multiple routes in a single route or AI route, modifies the relevant backend and frontend code, and enhances the Kubernetes model converter. \
  **Feature Value**: It supports adding multiple sub-routes in a single route configuration, providing users with more flexible route management capabilities, enhancing the system's configuration flexibility and user experience.

- **Related PR**: [#560](https://github.com/higress-group/higress-console/pull/560) \
  **Contributor**: @Erica177 \
  **Change Log**: This PR adds JSON Schema for multiple plugins, including AI agent, AI cache, etc., defining the structure and properties of plugin configurations, which helps improve the standardization and readability of configurations. \
  **Feature Value**: By introducing JSON Schema, users can more clearly understand each plugin's configuration items and their functions, simplifying the configuration process and reducing the risk of misconfiguration, thus improving the user experience.

- **Related PR**: [#555](https://github.com/higress-group/higress-console/pull/555) \
  **Contributor**: @hongzhouzi \
  **Change Log**: Added execution, list display, and table description tool configuration features for the DB MCP Server, ensuring consistency between console settings and those in higress-gateway. \
  **Feature Value**: Users can now view and manage the configuration information of the DB MCP Server tools through the console, enhancing the system's visual management and consistency.

- **Related PR**: [#550](https://github.com/higress-group/higress-console/pull/550) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR updates the logic for AI route configuration after updating specific types of LLM providers, ensuring that the routes are correctly synchronized when the upstream service name changes. \
  **Feature Value**: By automatically updating AI route configurations to adapt to service name changes after certain LLM provider type modifications, it enhances the system's flexibility and stability, reducing the need for manual adjustments.

- **Related PR**: [#547](https://github.com/higress-group/higress-console/pull/547) \
  **Contributor**: @CH3CHO \
  **Change Log**: Added undo/redo functionality in the system configuration page by introducing forwardRef and useImperativeHandle to support new APIs for the code editor component. \
  **Feature Value**: The newly added undo/redo feature improves the operational flexibility of users during system configuration, reducing inconvenience caused by errors and enhancing the user experience.

- **Related PR**: [#543](https://github.com/higress-group/higress-console/pull/543) \
  **Contributor**: @erasernoob \
  **Change Log**: This PR upgrades the plugin version from 1.0.0 to 2.0.0, involving updates to related entries in the plugins.properties file. \
  **Feature Value**: By upgrading the plugin version, it enhances the system's functionality and compatibility, allowing users to benefit from performance optimizations and additional features in the new version.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#559](https://github.com/higress-group/higress-console/pull/559) \
  **Contributor**: @KarlManong \
  **Change Log**: This PR corrects the line endings of all files in the project except binary and cmd files, unifying them to LF format, avoiding issues caused by inconsistent newline characters. \
  **Feature Value**: By unifying the line endings to LF, it improves the consistency and compatibility of the code, reducing problems caused by newline character differences, especially in cross-platform development environments.

- **Related PR**: [#554](https://github.com/higress-group/higress-console/pull/554) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed UI issues in the LLM provider management module, including the missing scheme for Google Vertex service endpoint and the form state not being reset after canceling the add provider operation. \
  **Feature Value**: By fixing these issues, it improves the user experience when managing and configuring LLM providers, ensuring the consistency and accuracy of the interface and its functions.

- **Related PR**: [#549](https://github.com/higress-group/higress-console/pull/549) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR ensures that the latest plugin configuration is always loaded when opening the configuration drawer, achieved by modifying the data retrieval logic in useEffect. \
  **Feature Value**: It resolves potential issues where user operations were based on outdated configurations due to untimely updates, improving the user experience and the system's response accuracy.

- **Related PR**: [#548](https://github.com/higress-group/higress-console/pull/548) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR corrects the issue of not trimming leading and trailing whitespace characters from the Wasm image URL before submission, ensuring the URL's validity. \
  **Feature Value**: By removing extra spaces from the Wasm image URL, it improves data accuracy, preventing loading failures due to formatting issues, thereby enhancing the user experience.

- **Related PR**: [#544](https://github.com/higress-group/higress-console/pull/544) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed the issue of incorrect error messages displayed when enabling authentication but not selecting a consumer, by updating the translation files and adjusting the code logic to ensure proper error prompts. \
  **Feature Value**: This fix improves the system's usability and user experience, ensuring that users receive accurate feedback when configuring services, avoiding confusion caused by misleading error messages.

### ♻️ Refactoring (Refactoring)

- **Related PR**: [#551](https://github.com/higress-group/higress-console/pull/551) \
  **Contributor**: @JayLi52 \
  **Change Log**: Removed the disabled state for host and port fields in database configurations, changed the default API gateway URL from https to http, and updated the API gateway URL display logic in the MCP detail page. \
  **Feature Value**: These changes enhance the system's flexibility and user-friendliness, allowing users to customize more configuration options and ensuring that the UI is consistent with backend behavior, improving the user experience.

---

## 📊 Release Statistics

- 🚀 New Features: 6
- 🐛 Bug Fixes: 5
- ♻️ Refactoring: 1

**Total**: 12 changes

Thank you to all contributors for your hard work! 🎉

