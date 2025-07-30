# Higress


## 📋 Overview of This Release

This release includes **31** updates, covering multiple aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 14 items
- **Bug Fixes**: 5 items
- **Refactoring and Optimization**: 6 items
- **Documentation Updates**: 5 items
- **Testing Improvements**: 1 item

### ⭐ Key Highlights

This release contains **2** major updates, which are recommended for special attention:

- **feat: Add Higress API MCP server** ([#2517](https://github.com/alibaba/higress/pull/2517)): This new feature provides users with a new way to manage and configure Higress resources, enhancing the system's flexibility and scalability, making it easier for users to manage routes and services.
- **Migrate WASM Go Plugins to New SDK and Go 1.24** ([#2532](https://github.com/alibaba/higress/pull/2532)): By migrating to the new SDK and Go version, this update improves code quality and compatibility, reduces potential compilation errors and runtime issues, and enhances system stability and performance.

For more details, please refer to the important features section below.

---

## 🌟 Detailed Description of Important Features

Below are the detailed descriptions of the key features and improvements in this release:

### 1. feat: Add Higress API MCP server

**Related PR**: [#2517](https://github.com/alibaba/higress/pull/2517) | **Contributor**: [cr7258](https://github.com/cr7258)

**Usage Background**

In a microservices architecture, effective route and service management is crucial. Higress, as a high-performance API gateway, requires a powerful tool to manage its internal resources. The existing management methods may lack flexibility or be difficult to scale. To address this issue, PR #2517 introduces a new Higress API MCP Server, which centralizes the management of routes, service origins, and plugins by calling the Higress Console API. This not only improves the system's operability but also enhances the user experience, particularly for operations teams that need to frequently adjust and optimize API gateway configurations.

**Feature Details**

This feature re-implements the Higress Ops MCP Server using golang-filter and adds the Higress API MCP Server. The main technical implementation includes:
1. Writing the `HigressClient` in Go, supporting basic HTTP operations (GET, POST, PUT, DELETE).
2. Implementing multiple API tools, such as route management (list-routes, get-route, add-route, update-route), service origin management (list-service-sources, get-service-source, add-service-source, update-service-source), and plugin management (get-plugin, delete-plugin, update-request-block-plugin).
3. Providing detailed configuration parameter descriptions, including the Higress Console URL, username, and password.
4. Code changes involve multiple files, including README documentation, configuration files, client implementation, and tool registration, totaling 1546 lines of code changes.

**Usage Instructions**

To enable and configure the Higress API MCP Server, follow these steps:
1. Add the relevant MCP Server configuration to the Higress Gateway configuration file, such as setting the `higressURL`, `username`, and `password` parameters.
2. When building the Higress Gateway image, ensure that the `golang-filter.so` plugin is included, using the `make build-gateway-local` command.
3. After starting the Higress Gateway, access the Higress API MCP Server through the configured path, e.g., `/higress-api`.
4. Use the provided API tools to manage routes, service origins, and plugins. For example, use `list-routes` to get all route information and `add-route` to add a new route.
**Notes**:
- Ensure that the Higress Console URL, username, and password are correct.
- In a production environment, it is recommended to use environment variables or encrypted storage to protect sensitive information.
- Follow best practices and regularly check and update configurations to maintain system security and stability.

**Feature Value**

The Higress API MCP Server brings the following significant advantages:
1. **Enhanced Management Efficiency**: Through a unified API interface, users can easily manage and configure various Higress resources, reducing the time and complexity of manual operations.
2. **Increased System Flexibility**: Supports dynamic adjustments to routes and service origins, allowing the system to quickly respond to changes in business needs.
3. **Improved Security**: Through strict parameter validation and error handling mechanisms, the system's stability and security are ensured.
4. **Simplified Operations**: Provides detailed logging and error handling, making it easier for operators to quickly locate and resolve issues.
5. **Promotes Ecosystem Development**: As part of the Higress ecosystem, this feature will further drive community development and improvement, providing users with more convenient tools and solutions.

---

### 2. Migrate WASM Go Plugins to New SDK and Go 1.24

**Related PR**: [#2532](https://github.com/alibaba/higress/pull/2532) | **Contributor**: [erasernoob](https://github.com/erasernoob)

**Usage Background**

This PR addresses the issues encountered when using the old Go SDK and lower versions of the Go language. With the release of Go 1.24, the new version offers better performance, security, and stability. Additionally, the new WASM Go SDK brings more features and improvements. The target user group is developers who use Higress for WebAssembly plugin development, who need a more modern and efficient development environment. Furthermore, for maintainers, a unified dependency management and build process can reduce maintenance costs.

**Feature Details**

Specifically, this update migrates all WASM Go plugins from the old SDK to the new SDK and upgrades the Go version to 1.24. The main technical points include:
1. Updating the Dockerfile and GitHub Actions workflows to support the new Go version and build parameters.
2. Modifying the go.mod file to update dependencies and remove unnecessary ones.
3. Fixing log type mismatch issues caused by changes in the logging package.
4. Optimizing the build script by removing unnecessary main functions and ensuring that the init() function complies with the proxy-wasm-go-sdk specification.
5. Adding resource cleanup and error handling logic to improve code robustness.

**Usage Instructions**

To enable and configure this feature, users need to update the relevant files in their projects:
1. Update the Dockerfile to use new build parameters (e.g., GOOS=wasip1 GOARCH=wasm).
2. Update the go.mod file to ensure the Go version is 1.24 and remove any unnecessary dependencies.
3. Update log calls in the project to use the new logger instance.
4. Check all callback functions and method parameters to ensure they match the new log types and other parameter orders.
**Typical Use Cases**:
- Developing new WASM Go plugins
- Upgrading existing plugins to take advantage of the new version's performance and features
**Notes**:
- Ensure all dependencies are updated, and there are no missing old dependency hashes.
- Check all log calls and callback functions to ensure type and parameter order consistency.

**Feature Value**

This update brings the following benefits to users:
1. **Performance Improvement**: Go 1.24 brings significant performance improvements, especially in concurrency and memory management, making plugins run faster and more efficiently.
2. **Enhanced Stability**: The new SDK and Go version provide better error handling and resource management, reducing the risk of runtime crashes.
3. **Strengthened Security**: The new versions of Go and the SDK fix multiple security vulnerabilities, enhancing the security of the plugins.
4. **Ease of Use**: A unified dependency management and build process simplify the development and deployment process, reducing maintenance costs.
5. **Ecosystem Compatibility**: By adopting the latest SDK and toolchain, plugins can better integrate with other modern WebAssembly ecosystem components, enhancing the interoperability of the entire ecosystem.

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#2679](https://github.com/alibaba/higress/pull/2679)
  **Contributor**: erasernoob
  **Change Log**: This PR implements support for external FQDN in image annotations and adds corresponding test cases, mainly modifying mirror.go and mirror_test.go files.
  **Feature Value**: This feature allows users to configure external services via image annotations, enhancing the system's flexibility and scalability, and making it easier to integrate with external services.

- **Related PR**: [#2667](https://github.com/alibaba/higress/pull/2667)
  **Contributor**: hanxiantao
  **Change Log**: Added support for setting a global rate limit threshold for the AI Token rate-limiting plugin and unified the base logic of cluster-key-rate-limit and ai-token-ratelimit plugins.
  **Feature Value**: Users can more flexibly control API request traffic by setting a global rate limit to protect backend services from overloading. The improvement in configuration consistency also reduces potential configuration errors.

- **Related PR**: [#2652](https://github.com/alibaba/higress/pull/2652)
  **Contributor**: OxalisCu
  **Change Log**: This PR adds support for first-byte timeout for LLM streaming requests in the ai-proxy plugin, implemented by introducing the strconv package and modifying the ProviderConfig struct in provider.go.
  **Feature Value**: The new first-byte timeout feature allows users to set a timeout for LLM streaming requests, enabling appropriate actions if the response is not received within the specified time, improving the system's flexibility and reliability.

- **Related PR**: [#2650](https://github.com/alibaba/higress/pull/2650)
  **Contributor**: zhangjingcn
  **Change Log**: This PR implements the functionality to fetch ErrorResponseTemplate configuration from the Nacos MCP registry, by modifying mcp_model.go and watcher.go files to support the new feature requirements.
  **Feature Value**: The added ability to fetch ErrorResponseTemplate from the Nacos MCP registry enhances the system's flexibility and configurability, allowing users to customize error response templates as needed.

- **Related PR**: [#2649](https://github.com/alibaba/higress/pull/2649)
  **Contributor**: CH3CHO
  **Change Log**: This PR adds support for three different URL configuration formats for Azure OpenAI, strengthens model mapping, and ensures the `api-version` parameter is always present.
  **Feature Value**: By supporting multiple URL formats, users can more flexibly configure Azure OpenAI services, enhancing the system's compatibility and ease of use, and improving the user experience.

- **Related PR**: [#2648](https://github.com/alibaba/higress/pull/2648)
  **Contributor**: daixijun
  **Change Log**: This PR adds support for the /v1/messages interface of Anthropic in the qwen Provider, implemented by modifying the related ai-proxy files.
  **Feature Value**: The new interface support expands the qwen feature set, allowing users to leverage Anthropic services, enhancing the system's flexibility and applicability.

- **Related PR**: [#2639](https://github.com/alibaba/higress/pull/2639)
  **Contributor**: johnlanni
  **Change Log**: This PR disables the rerouting feature in specified plugins by setting ctx.DisableReroute to uniformly control, ensuring that plugins that do not require re-matching routes avoid unnecessary processing.
  **Feature Value**: This enhancement improves the functional flexibility and performance of specific plugins, preventing them from forcibly re-matching routes after modifying request headers, thus improving processing efficiency and response speed.

- **Related PR**: [#2585](https://github.com/alibaba/higress/pull/2585)
  **Contributor**: akolotov
  **Change Log**: This PR adds the configuration file for the Blockscout MCP server, including detailed YAML configuration and README documentation, to support users in deploying and using the service.
  **Feature Value**: By providing support for the Blockscout MCP server, the system enhances data analysis capabilities for EVM-compatible blockchains, improving the user experience and system functionality.

- **Related PR**: [#2551](https://github.com/alibaba/higress/pull/2551)
  **Contributor**: daixijun
  **Change Log**: This PR adds support for Anthropic and Gemini APIs, specifically including interfaces such as anthropic/v1/messages, anthropic/v1/complete, and gemini/v1beta/generatecontent.
  **Feature Value**: By introducing new API support for Anthropic and Gemini, users can leverage more AI service capabilities, enhancing the system's functional diversity and flexibility, and providing users with richer application scenarios.

- **Related PR**: [#2542](https://github.com/alibaba/higress/pull/2542)
  **Contributor**: daixijun
  **Change Log**: This PR adds the functionality to track token usage for the images, audio, and responses interfaces, and defines related utility functions as public to reduce code duplication.
  **Feature Value**: This update allows users to better monitor and manage their API token usage, helping to improve resource utilization and cost control, especially for services that frequently call these interfaces.

- **Related PR**: [#2537](https://github.com/alibaba/higress/pull/2537)
  **Contributor**: wydream
  **Change Log**: This PR adds support for text reordering functionality for the Qwen model, implemented by adding a new API path in the ai-proxy.
  **Feature Value**: The addition of text reordering capability for the Qwen model enables users to perform more precise data processing and content management.

- **Related PR**: [#2535](https://github.com/alibaba/higress/pull/2535)
  **Contributor**: wydream
  **Change Log**: This PR introduces the `basePath` and `basePathHandling` options, supporting flexible handling of request paths. It allows setting `removePrefix` or `prepend` to determine how to use `basePath`.
  **Feature Value**: This new feature allows users to adjust the way request paths are handled according to their needs, making the API gateway better suited for backend services, enhancing system flexibility and usability.

- **Related PR**: [#2499](https://github.com/alibaba/higress/pull/2499)
  **Contributor**: heimanba
  **Change Log**: This PR adds support for the useManifestAsEntry configuration, updates the GrayConfig struct and related processing logic, and modifies the documentation to reflect these changes.
  **Feature Value**: By introducing the useManifestAsEntry configuration, users can more flexibly control the caching strategy for homepage requests, enhancing system flexibility and user experience.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#2687](https://github.com/alibaba/higress/pull/2687)
  **Contributor**: Thomas-Eliot
  **Change Log**: Fixed an SQL error that occurred when using the mcp client tool to describeTable, ensuring that table structures are correctly described during data migration from Postgres to the MCP Server.
  **Feature Value**: This fix resolves a critical issue in the data migration process, improving system stability and reliability, and allowing users to complete database operations smoothly without interruptions.

- **Related PR**: [#2662](https://github.com/alibaba/higress/pull/2662)
  **Contributor**: johnlanni
  **Change Log**: Fixed a memory leak issue in Envoy's proxy-wasm and a 404 error due to port mapping mismatch when ppv2 is enabled.
  **Feature Value**: This fix addresses issues caused by memory leaks and port mapping errors, improving system stability and user experience.

- **Related PR**: [#2656](https://github.com/alibaba/higress/pull/2656)
  **Contributor**: co63oc
  **Change Log**: This PR corrects multiple spelling errors, including variable names, function names, interface method names, and plugin names in the documentation, improving code readability and consistency.
  **Feature Value**: By correcting spelling errors, the correctness of program logic and the accuracy of documentation are ensured, enhancing user experience and developer maintenance efficiency.

- **Related PR**: [#2623](https://github.com/alibaba/higress/pull/2623)
  **Contributor**: Guo-Chenxu
  **Change Log**: Fixed a translation issue caused by special characters by adjusting the way JSON data is generated to avoid potential format errors.
  **Feature Value**: This fix ensures that the system can correctly generate and parse JSON when handling data containing special characters, improving system stability and reliability.

- **Related PR**: [#2507](https://github.com/alibaba/higress/pull/2507)
  **Contributor**: hongzhouzi
  **Change Log**: Corrected an error that occurred when compiling golang-filter.so on arm64 architecture machines, ensuring the correct installation of the corresponding architecture toolchain.
  **Feature Value**: This fix resolves issues encountered by arm64 users during compilation, improving cross-platform compatibility and user experience.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#2673](https://github.com/alibaba/higress/pull/2673)
  **Contributor**: johnlanni
  **Change Log**: Improved the `findEndpointUrl` function to handle multiple SSE messages, not just the first one. By increasing tolerance for other types of messages, the function's robustness and flexibility are enhanced.
  **Feature Value**: This improvement enhances system compatibility, allowing the correct parsing of the required endpoint URL even when non-'endpoint' initial messages are encountered, thereby improving user experience and system stability.

- **Related PR**: [#2661](https://github.com/alibaba/higress/pull/2661)
  **Contributor**: johnlanni
  **Change Log**: This PR relaxes the regular expression for DNS service domain validation, allowing more flexible domain formats. Implemented by modifying the domainRegex variable definition in watcher.go.
  **Feature Value**: By relaxing domain validation rules, the system can support more types of valid domains, increasing system compatibility and flexibility, and providing a better user experience.

- **Related PR**: [#2615](https://github.com/alibaba/higress/pull/2615)
  **Contributor**: johnlanni
  **Change Log**: This PR removes the unused EXTRA_TAGS variable from Dockerfiles, Makefiles, and multiple extension configuration files related to wasm-go plugins, simplifying the build process.
  **Feature Value**: By cleaning up unnecessary configuration items, the project structure becomes more concise and clear, reducing potential maintenance costs and improving development efficiency.

- **Related PR**: [#2598](https://github.com/alibaba/higress/pull/2598)
  **Contributor**: johnlanni
  **Change Log**: This PR updates the Go version in the wasm-go builder image to 1.24.4, removes a large amount of old code, and simplifies the Dockerfile.
  **Feature Value**: By upgrading the Go version and streamlining the Dockerfile, the efficiency and security of the WASM plugin build process are improved, making maintenance more convenient.

- **Related PR**: [#2564](https://github.com/alibaba/higress/pull/2564)
  **Contributor**: rinfx
  **Change Log**: This PR optimizes the location of the minimum request count logic and the Redis Lua script, ensuring the accuracy of request counting and configuration judgment, and improving system stability and performance.
  **Feature Value**: By improving the request counting logic and fixing potential type conversion errors, the accuracy and reliability of the load balancing strategy are enhanced, improving the user experience.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2675](https://github.com/alibaba/higress/pull/2675)
  **Contributor**: Aias00
  **Change Log**: This PR fixes several dead links in the project documentation, ensuring users can access the correct resources.
  **Feature Value**: By correcting broken links in the documentation, the user experience is improved, ensuring they can smoothly obtain the required information, enhancing the reliability and usability of the documentation.

- **Related PR**: [#2668](https://github.com/alibaba/higress/pull/2668)
  **Contributor**: Aias00
  **Change Log**: This PR significantly improves the README for Rust plugins, adding detailed development guidelines, including environment requirements, build steps, and testing methods.
  **Feature Value**: By providing comprehensive development documentation, new developers can quickly understand and get started with Rust Wasm plugin development, improving the project's maintainability and usability.

- **Related PR**: [#2647](https://github.com/alibaba/higress/pull/2647)
  **Contributor**: Guo-Chenxu
  **Change Log**: This PR adds the New Contributors and full changelog sections, and improves the markdown format to support forced line breaks.
  **Feature Value**: By adding contributor lists and a full changelog, the readability and richness of the documentation are enhanced, helping users better understand project update dynamics.

- **Related PR**: [#2635](https://github.com/alibaba/higress/pull/2635)
  **Contributor**: github-actions[bot]
  **Change Log**: This PR adds detailed release notes for Higress 2.1.5, including new features, bug fixes, and performance optimizations.
  **Feature Value**: By providing detailed release notes, users can better understand the improvements and changes in the latest version, helping them to use and maintain the system more effectively.

- **Related PR**: [#2586](https://github.com/alibaba/higress/pull/2586)
  **Contributor**: erasernoob
  **Change Log**: Updated the README documentation for wasm-go, removed TinyGo-related configurations, and updated the Go version requirement to 1.24. Also adjusted the environment variable settings in the Dockerfile.
  **Feature Value**: By updating the documentation and dependency information, developers can build the project according to the latest requirements, avoiding issues caused by using outdated or incompatible toolchains, and improving the development experience and efficiency.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#2596](https://github.com/alibaba/higress/pull/2596)
  **Contributor**: Guo-Chenxu
  **Change Log**: Added a GitHub Actions workflow to automatically generate and submit release notes as a PR whenever a new version is released. This is achieved by setting necessary secrets.
  **Feature Value**: This feature automates the generation and updating of project documentation, reducing manual operations for maintainers, improving work efficiency, and ensuring that the change log for each release is timely and accurately communicated to users.

---

## 📊 Release Statistics

- 🚀 New Features: 14 items
- 🐛 Bug Fixes: 5 items
- ♻️ Refactoring and Optimization: 6 items
- 📚 Documentation Updates: 5 items
- 🧪 Testing Improvements: 1 item

**Total**: 31 changes (including 2 major updates)

Thank you to all contributors for your hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **12** updates, covering enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 6
- **Bug Fixes**: 5
- **Refactoring and Optimization**: 1

---

## 📝 Complete Changelog

### 🚀 New Features (Features)

- **Related PR**: [#562](https://github.com/higress-group/higress-console/pull/562)
  **Contributor**: CH3CHO
  **Change Log**: This PR implements the feature of configuring multiple routes within a single route or AI route by modifying the backend SDK service and frontend components to support this new feature.
  **Feature Value**: Users can now configure multiple routing rules in a single route definition, enhancing the system's flexibility and scalability, making route management more convenient and efficient.

- **Related PR**: [#560](https://github.com/higress-group/higress-console/pull/560)
  **Contributor**: Erica177
  **Change Log**: This PR adds JSON Schema for multiple plugins including AI proxy, AI cache, AI data masking, AI history, and AI intent recognition, to enhance the standardization and ease of use of configurations.
  **Feature Value**: By introducing JSON Schema, users can more intuitively understand and configure plugin parameters, improving development efficiency and reducing configuration errors, thereby enhancing user experience.

- **Related PR**: [#555](https://github.com/higress-group/higress-console/pull/555)
  **Contributor**: hongzhouzi
  **Change Log**: This PR adds functionality for executing, listing tables, and describing tables in DB MCP Server, and synchronizes the console with the configuration in higress-gateway.
  **Feature Value**: The new features allow users to view the configuration of related tools in DB MCP Server through the console, improving system maintainability and user experience.

- **Related PR**: [#550](https://github.com/higress-group/higress-console/pull/550)
  **Contributor**: CH3CHO
  **Change Log**: This PR implements the feature of updating AI route configurations after updating an LLM provider with a specific type, ensuring compatibility and correctness after service name changes.
  **Feature Value**: By automatically adjusting AI route settings in response to LLM provider updates, the system's flexibility and maintenance efficiency are improved, reducing the need for manual intervention.

- **Related PR**: [#547](https://github.com/higress-group/higress-console/pull/547)
  **Contributor**: CH3CHO
  **Change Log**: Implemented undo/redo functionality on the system configuration page by introducing forwardRef and useImperativeHandle to manage the state of the CodeEditor component.
  **Feature Value**: Users can now undo or redo changes on the system configuration page, enhancing the flexibility of configuration editing and user experience.

- **Related PR**: [#543](https://github.com/higress-group/higress-console/pull/543)
  **Contributor**: erasernoob
  **Change Log**: This PR upgrades the plugin version from 1.0.0 to 2.0.0, involving updating the plugin addresses in the configuration files to point to the new version.
  **Feature Value**: By upgrading the plugin version, users can take advantage of new features, performance improvements, and potential bug fixes, thereby enhancing the overall functionality and stability of the application.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#559](https://github.com/higress-group/higress-console/pull/559)
  **Contributor**: KarlManong
  **Change Log**: This PR fixes the issue of inconsistent line endings in project files, ensuring that all files except binary and cmd files end with LF, avoiding potential issues caused by different line endings.
  **Feature Value**: By unifying the line ending format, code consistency and portability are improved, reducing various issues caused by line ending differences across operating systems, and enhancing user experience and development efficiency.

- **Related PR**: [#554](https://github.com/higress-group/higress-console/pull/554)
  **Contributor**: CH3CHO
  **Change Log**: Fixed two UI issues in the LLM provider management module: added the missing scheme in the Google Vertex service endpoint and ensured form state reset after canceling a new provider operation.
  **Feature Value**: Resolved issues encountered by users when configuring and managing LLM providers, enhancing user experience and system usability.

- **Related PR**: [#549](https://github.com/higress-group/higress-console/pull/549)
  **Contributor**: CH3CHO
  **Change Log**: This PR fixes the issue where the latest plugin configuration was not loaded when opening the configuration edit drawer, ensuring users can make modifications based on the latest configuration information.
  **Feature Value**: Resolved the issue of out-of-sync information when editing plugin configurations, improving user experience and the accuracy of configuration management.

- **Related PR**: [#548](https://github.com/higress-group/higress-console/pull/548)
  **Contributor**: CH3CHO
  **Change Log**: Fixed the issue of leading and trailing spaces in the Wasm image URL before submission, implemented by modifying the relevant code in index.tsx.
  **Feature Value**: Resolved potential errors or failures caused by leading and trailing spaces in URLs, improving system stability and user experience.

- **Related PR**: [#544](https://github.com/higress-group/higress-console/pull/544)
  **Contributor**: CH3CHO
  **Change Log**: Fixed the issue of incorrect error messages when enabling authentication but not selecting a consumer, implemented by updating the text in translation files and removing redundant code.
  **Feature Value**: Improved system accuracy and user experience, ensuring users receive correct feedback under specific configurations, avoiding misguidance.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#551](https://github.com/higress-group/higress-console/pull/551)
  **Contributor**: JayLi52
  **Change Log**: Removed the disabled state of host and port fields in database configuration, changed the default API gateway URL to http, and updated the logic for displaying the API gateway URL on the MCP page.
  **Feature Value**: Users can now edit the host and port fields in the database configuration, and improve the consistency and availability of the API gateway URL by using the new default protocol.

---

## 📊 Release Statistics

- 🚀 New Features: 6
- 🐛 Bug Fixes: 5
- ♻️ Refactoring and Optimization: 1

**Total**: 12 changes

Thanks to all contributors for their hard work! 🎉

