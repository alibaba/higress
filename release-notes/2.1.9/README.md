# Higress


## 📋 Overview of This Release

This release includes **44** updates, covering feature enhancements, bug fixes, performance optimizations, and more.

### Update Content Distribution

- **New Features**: 23 items
- **Bug Fixes**: 13 items
- **Refactoring and Optimization**: 3 items
- **Documentation Updates**: 1 item
- **Testing Improvements**: 4 items

### ⭐ Key Highlights

This release contains **3** important updates, which are recommended for your attention:

- **feat(mcp-server): add server-level default authentication and MCP proxy server support** ([#3096](https://github.com/alibaba/higress/pull/3096)): By introducing server-level default authentication, the system's security and flexibility have been enhanced, allowing users to manage tool and service security policies more conveniently, thereby improving overall service security and user experience.
- **feat: add higress api mcp server** ([#2923](https://github.com/alibaba/higress/pull/2923)): By integrating the higress api mcp server, users can more easily manage and operate Higress resources such as routing, service origins, and AI routing, enhancing the system's manageability and flexibility.
- **feat: implement `hgctl agent` & `mcp add` subcommand** ([#3051](https://github.com/alibaba/higress/pull/3051)): By introducing new subcommands, the convenience of user operations and the scalability of the system have been improved, allowing users to manage and configure MCP services more flexibly.

For more details, please refer to the Important Features section below.

---

## 🌟 Detailed Description of Important Features

Here is a detailed description of the key features and improvements in this release:

### 1. feat(mcp-server): add server-level default authentication and MCP proxy server support

**Related PR**: [#3096](https://github.com/alibaba/higress/pull/3096) | **Contributor**: [@johnlanni](https://github.com/johnlanni)

**Usage Background**

In modern microservice architectures, authentication and authorization are key factors in ensuring system security. Existing authentication mechanisms may be too dispersed and difficult to manage, leading to complex and error-prone configurations. Additionally, as the system scales, a single MCP server may not meet performance requirements, necessitating the introduction of a proxy server to distribute the load. This feature aims to address these issues by providing a unified, configurable authentication mechanism and supporting request forwarding through an MCP proxy server. The target user group includes developers and operators who need to enhance system security and scalability.

**Feature Details**

1. **Server-Level Default Authentication**: New configuration options `defaultDownstreamSecurity` and `defaultUpstreamSecurity` have been added to set default authentication for client-to-gateway and gateway-to-backend, respectively. These options can be configured at the global level, with tool-level settings overriding global settings. This design makes authentication configuration more flexible and manageable.
2. **MCP Proxy Server Type**: A new server type, `mcp-proxy`, has been introduced, allowing MCP requests from clients to be forwarded to backend MCP servers. The `mcpServerURL` field can specify the address of the backend MCP server, and the `timeout` field can control the request timeout. Additionally, full authentication mechanisms, including client-to-gateway and gateway-to-backend authentication, are supported.
3. **Authentication Code Refactoring**: The code related to authentication has been refactored to improve maintainability and extensibility. The version of the dependency library has been updated to ensure compatibility with the latest version.

**Usage Instructions**

1. **Enable and Configure**: First, enable `defaultDownstreamSecurity` and `defaultUpstreamSecurity` in the configuration file and set the corresponding authentication parameters. For the MCP proxy server, specify the `mcpServerURL` and `timeout` fields.
2. **Typical Use Cases**: Suitable for microservice architectures that require centralized management and unified distribution of authentication policies. For example, in a multi-tenant environment, default authentication configurations can be used to manage the authentication policies of all tenants uniformly.
3. **Notes**: Ensure version compatibility of all related components to avoid issues caused by version mismatches. It is also recommended to conduct thorough testing in a production environment before deployment to verify the effectiveness and performance of the configuration.

**Feature Value**

1. **Enhanced Security**: By providing a unified authentication configuration, the risk of configuration errors is reduced, enhancing the overall system security. In multi-tenant environments, it better isolates access permissions for different tenants.
2. **Increased Flexibility**: A multi-level priority configuration mechanism is provided, making authentication policies more flexible and manageable. Tool-level configurations can override global settings to adapt to different business needs.
3. **Improved Scalability**: By introducing an MCP proxy server, the load on a single MCP server can be effectively distributed, improving the overall performance and stability of the system. In large-scale distributed systems, this design significantly enhances system scalability.

---

### 2. feat: add higress api mcp server

**Related PR**: [#2923](https://github.com/alibaba/higress/pull/2923) | **Contributor**: [@Tsukilc](https://github.com/Tsukilc)

**Usage Background**

This feature addresses the convenience and flexibility of managing Higress resources. In the past, users may have needed to use multiple APIs or tools to manage different resources, such as routes, service origins, and plugins. Now, with the Higress API MCP Server, users can manage these resources in a unified interface, including new AI routes, AI providers, and MCP servers. This not only simplifies the operational process but also enhances the security and maintainability of the system. The target user group mainly consists of Higress operators and developers who need to manage various Higress resources efficiently and securely.

**Feature Details**

This PR mainly implements the following features:
1. **New Higress API MCP Server**: Provides a unified API interface to manage Higress resources such as routes, service origins, AI routes, AI providers, MCP servers, and plugins.
2. **Updated Authentication Mechanism**: Changed from the previous username and password authentication to HTTP Basic Authentication, enhancing security.
3. **New Tool Registration**: Registered new AI route, AI provider, and MCP server management tools, enabling the system to support these new features.
4. **Code Optimization**: Removed unnecessary type conversions, improving performance and code clarity.
5. **Documentation Update**: Updated the README documentation to reflect the new features. Technically, by introducing new structs and utility functions, the existing MCP Server functionality was extended, ensuring compatibility with existing features.

**Usage Instructions**

Enabling and configuring the Higress API MCP Server is simple:
1. Configure the URL address of the Higress Console.
2. Choose an appropriate authentication method (e.g., HTTP Basic Authentication) and provide the corresponding credentials.
3. Use the provided API tools for resource management. For example, use `list-ai-routes` to list all AI routes, and use `add-ai-route` to add a new AI route, etc.
4. For MCP server management, you can use `list-mcp-servers` to list all MCP servers and use `add-or-update-mcp-server` to add or update an MCP server, etc.
5. **Notes**: Ensure that all fields in the configuration files are correctly filled, especially those involving weight sum validation. It is recommended to use the latest client tools (e.g., Cherry Studio) to provide credentials, enhancing security.

**Feature Value**

This feature brings significant benefits to users:
1. **Improved Management Efficiency**: Through a unified API interface, users can more efficiently manage and configure various Higress resources, reducing operational complexity.
2. **Enhanced Security**: The new authentication mechanism (e.g., HTTP Basic Authentication) enhances system security, preventing unauthorized access.
3. **Extended Functionality**: The addition of AI route, AI provider, and MCP server management tools enables Higress to better support modern application needs.
4. **Code Optimization**: Removing unnecessary type conversions and optimizing query parameter concatenation logic improves system performance and code quality. Overall, this feature not only enhances user experience but also strengthens the system's stability and security, making it significant in the Higress ecosystem.

---

### 3. feat: implement `hgctl agent` & `mcp add` subcommand 

**Related PR**: [#3051](https://github.com/alibaba/higress/pull/3051) | **Contributor**: [@erasernoob](https://github.com/erasernoob)

**Usage Background**

This PR addresses the inconvenience users face when managing and configuring MCP services. In the current Higress CLI tool, there is no direct function to add MCP services, requiring users to manually configure complex API calls. Additionally, there is a lack of a unified CLI tool to initialize and manage the environment. The new feature aims to simplify these operations, allowing users to quickly add and manage MCP services via simple command-line instructions and providing an interactive agent window to set necessary environment variables. The target user group includes Higress developers, operators, and any users who need to manage MCP services.

**Feature Details**

This update implements two new subcommands: `hgctl agent` and `mcp add`. The `hgctl agent` command starts an interactive agent window to guide users through environment setup. The `mcp add` command allows users to directly add MCP services, supporting two types of services: direct proxy and OpenAPI-based MCP services. For direct proxy services, users can publish services by specifying URLs and other parameters; for OpenAPI-based services, users can upload OpenAPI specification files and configure them. The core technical points involve integrating with the Higress Console API to achieve automatic service registration and management. Code changes primarily focus on the newly added `agent` package and related modules, such as `base.go`, `core.go`, and `mcp.go`.

**Usage Instructions**

To enable `hgctl agent`, simply run the `hgctl agent` command. The system will prompt the user to set necessary environment variables upon first use. To use the `mcp add` command to add an MCP service, users can choose one of the following two methods based on their needs:
1. Add a direct proxy service:
   ```bash
   hgctl mcp add mcp-deepwiki -t http https://mcp.deepwiki.com --user admin --password 123 --url http://localhost:8080
   ```
2. Add an OpenAPI-based service:
   ```bash
   hgctl mcp add openapi-server -t openapi --spec openapi.yaml --user admin --password 123 --url http://localhost:8080
   ```
**Notes**: Ensure that Go 1.24 or higher is installed, and the environment variables are correctly configured. Best practices include using a custom logging library to record errors and debug information in production environments.

**Feature Value**

The new feature significantly enhances the usability and flexibility of the Higress CLI tool. By introducing `hgctl agent`, users can easily initialize and manage the environment without manually configuring complex environment variables. The `mcp add` command further simplifies the process of adding MCP services, supporting multiple types of MCP services, and improving development and operational efficiency. Additionally, integration with the Higress Console API ensures the consistency and reliability of services. These improvements not only enhance user experience but also strengthen the overall performance and stability of the system, making it significant in the Higress ecosystem.

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#3126](https://github.com/alibaba/higress/pull/3126) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR updates the Envoy dependency, allowing the WASM plugin to configure Redis client buffering behavior via URL query parameters, including setting the maximum buffer size and flush timeout. \
  **Feature Value**: This feature allows users to more flexibly control buffer parameters related to Redis calls, thereby optimizing performance and meeting specific application needs.

- **Related PR**: [#3123](https://github.com/alibaba/higress/pull/3123) \
  **Contributor**: @johnlanni \
  **Change Log**: Upgraded the proxy version to v2.2.0, updated the Go toolchain to 1.23.7, and added architecture-specific build targets for golang-filter, while fixing dependency issues related to MCP server, OpenAI, and Milvus SDK support. \
  **Feature Value**: This update enhances system compatibility and performance, making it easier for developers to deploy applications on different architectures and improving the stability of integration with other services like OpenAI and Milvus.

- **Related PR**: [#3108](https://github.com/alibaba/higress/pull/3108) \
  **Contributor**: @wydream \
  **Change Log**: Added video-related API paths and processing capabilities, including API name constants for video series, default function entries, and regular expression path handling, and updated the OpenAI provider to support these new endpoints. \
  **Feature Value**: This feature expands the AI proxy plugin's capability to handle and parse more types of media content requests, especially video-related operations, thus enhancing user flexibility and efficiency in multimedia content management.

- **Related PR**: [#3071](https://github.com/alibaba/higress/pull/3071) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds a new feature example named `inject_encoded_data_to_filter_chain_on_header`, allowing the addition of a response body to a request even if there is no response body. By calling a specific Wasm function and processing requests and responses according to specified rules, it ensures the correct injection of data. \
  **Feature Value**: This feature extends the application's service capabilities, allowing developers to more flexibly control HTTP response content, especially in scenarios where dynamic generation or modification of the response body is required, greatly enhancing service flexibility and user experience.

- **Related PR**: [#3067](https://github.com/alibaba/higress/pull/3067) \
  **Contributor**: @wydream \
  **Change Log**: This PR adds vLLM as a new AI provider, supporting various OpenAI-compatible APIs, including chat completion and text completion. \
  **Feature Value**: The addition of vLLM support greatly expands Higress' capabilities in proxying AI services, allowing users to more flexibly use different types of AI models and services.

- **Related PR**: [#3060](https://github.com/alibaba/higress/pull/3060) \
  **Contributor**: @erasernoob \
  **Change Log**: This PR enhances the `hgctl mcp` and `hgctl agent` commands, enabling them to automatically fetch Higress Console credentials from installation configuration files and Kubernetes secrets. \
  **Feature Value**: Simplifies the process of handling authentication information when using Higress, improving operational convenience and user experience.

- **Related PR**: [#3043](https://github.com/alibaba/higress/pull/3043) \
  **Contributor**: @2456868764 \
  **Change Log**: Fixed the default port error for Milvus and added Python example code in the README.md to help users better understand and use the feature. \
  **Feature Value**: By correcting the configuration error and providing Python example code, the system's stability and usability are improved, making it easier for users to get started and integrate.

- **Related PR**: [#3040](https://github.com/alibaba/higress/pull/3040) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: This PR adds ApiNameAnthropicMessages to the Claude feature, supporting the configuration of the anthropic provider without using protocol=original and directly forwarding /v1/messages requests to anthropic. \
  **Feature Value**: Enhances the flexibility and compatibility of different AI service providers, making it easier for users to interact with the Claude API, thus improving the diversity and user experience of the application.

- **Related PR**: [#3038](https://github.com/alibaba/higress/pull/3038) \
  **Contributor**: @Libres-coder \
  **Change Log**: Added the `list-plugin-instances` tool, allowing the AI proxy to query plugin instances within a specified range via the MCP protocol, and updated the bilingual documentation. \
  **Feature Value**: This feature enhances the management capability of plugin instances, allowing users to more flexibly query plugin information at different levels, improving system maintainability and user experience.

- **Related PR**: [#3032](https://github.com/alibaba/higress/pull/3032) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR enables Qwen compatibility mode by default and adds missing API endpoints, including AsyncAIGC, AsyncTask, and V1Rerank, enhancing the AI proxy features. \
  **Feature Value**: By enabling compatibility mode by default and expanding API coverage, this update provides users with a more complete out-of-the-box experience and more comprehensive feature support, enhancing system usability and flexibility.

- **Related PR**: [#3029](https://github.com/alibaba/higress/pull/3029) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: This PR adds support for v1/responses in the groq provider by updating the code in the groq.go file. \
  **Feature Value**: The new responses feature allows users to better manage and process response data, improving system flexibility and usability, providing developers with more customization space.

- **Related PR**: [#3024](https://github.com/alibaba/higress/pull/3024) \
  **Contributor**: @rinfx \
  **Change Log**: Added malicious URL and model hallucination detection, fixed the issue of incorrect response when the response contains empty content, and adjusted specific consumer configurations. \
  **Feature Value**: Enhances the system's ability to identify malicious behavior, improves user experience and security, and optimizes the handling logic for multiple event return scenarios.

- **Related PR**: [#3008](https://github.com/alibaba/higress/pull/3008) \
  **Contributor**: @hellocn9 \
  **Change Log**: Added support for custom parameter names to configure MCP SSE stateful sessions via the `higress.io/mcp-sse-stateful-param-name` annotation. \
  **Feature Value**: Allows users to customize the parameter names for MCP SSE stateful sessions, increasing the flexibility and configurability of the application to meet more scenario needs.

- **Related PR**: [#3006](https://github.com/alibaba/higress/pull/3006) \
  **Contributor**: @SaladDay \
  **Change Log**: This PR introduces Secret reference support for the Redis configuration of the MCP Server, allowing users to securely store passwords without exposing them in the ConfigMap. \
  **Feature Value**: By allowing the use of Kubernetes Secrets to store sensitive information, the system's security is improved, avoiding the risks associated with storing passwords in plain text.

- **Related PR**: [#2992](https://github.com/alibaba/higress/pull/2992) \
  **Contributor**: @rinfx \
  **Change Log**: This PR records the consumer's name during the authentication and authorization process, even if the consumer is not authorized, for better log observation. \
  **Feature Value**: By recording the names of unauthorized consumers, the system's auditability and troubleshooting capabilities are enhanced, allowing administrators to have a more comprehensive understanding of access requests.

- **Related PR**: [#2978](https://github.com/alibaba/higress/pull/2978) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds the X-Mse-Consumer header to the request to record the consumer's name, regardless of whether the consumer is authorized, once the consumer's identity is determined during the authentication process. \
  **Feature Value**: This feature enhances the traceability of the system, allowing each request to carry consumer information, which is helpful for subsequent audits, log analysis, and problem troubleshooting.

- **Related PR**: [#2968](https://github.com/alibaba/higress/pull/2968) \
  **Contributor**: @2456868764 \
  **Change Log**: Implemented vector database mapping, including a field mapping system and index configuration management, supporting various index types such as HNSW, IVF, and SCANN. \
  **Feature Value**: Enhances the flexibility and compatibility of the system, allowing users to customize field mappings and index configurations, thus better adapting to the needs of different database architectures.

- **Related PR**: [#2943](https://github.com/alibaba/higress/pull/2943) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: This PR adds support for custom system prompts, allowing users to use custom system prompts when generating release notes. The implementation is done by modifying the GitHub Actions workflow configuration. \
  **Feature Value**: This feature allows users to add personalized system prompt information when generating release note documents, improving the flexibility and user experience of the document, making the release notes more aligned with the actual project situation.

- **Related PR**: [#2942](https://github.com/alibaba/higress/pull/2942) \
  **Contributor**: @2456868764 \
  **Change Log**: Fixed the handling logic when the LLM provider is empty, optimized the document structure and content, and updated the README to better describe the functions and configurations of the MCP server. \
  **Feature Value**: Enhances the robustness of the system when the LLM provider is empty, improves the user experience, and allows users to better understand the tools and configuration requirements provided by the MCP server.

- **Related PR**: [#2916](https://github.com/alibaba/higress/pull/2916) \
  **Contributor**: @imp2002 \
  **Change Log**: Implemented Nginx migration to the MCP server and provided 7 MCP tools to automate the migration of Nginx configurations and Lua plugins to Higress. \
  **Feature Value**: This feature greatly simplifies the migration from Nginx to Higress, improves migration efficiency, and reduces the complexity of user operations.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#3120](https://github.com/alibaba/higress/pull/3120) \
  **Contributor**: @lexburner \
  **Change Log**: Adjusted the log level in the ai-proxy plugin by lowering the log level of specific warning messages, reducing unnecessary redundant warning outputs. \
  **Feature Value**: Reducing redundant warning messages improves the readability and maintainability of logs, helping users focus on important log information, thereby enhancing the overall user experience.

- **Related PR**: [#3118](https://github.com/alibaba/higress/pull/3118) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixes the issue where port-level TLS and load balancing settings would unconditionally overwrite existing ingress annotation configurations. By adding null checks and improving policy merge logic, it ensures that existing configurations are not mistakenly overwritten. \
  **Feature Value**: This fix avoids unnecessary loss or replacement of configurations, improving system stability and reliability, ensuring that user-defined ingress annotations take effect correctly, and enhancing the user experience.

- **Related PR**: [#3095](https://github.com/alibaba/higress/pull/3095) \
  **Contributor**: @rinfx \
  **Change Log**: Fixed the issue of usage information being lost during the claude2openai conversion process and added an index field to the bedrock streaming tool response to improve the accuracy and completeness of data processing. \
  **Feature Value**: This fix ensures that users can obtain complete usage information when using claude2openai conversion and enhances the tracking capability of bedrock streaming responses by adding the index field, improving the user experience and system maintainability.

- **Related PR**: [#3084](https://github.com/alibaba/higress/pull/3084) \
  **Contributor**: @rinfx \
  **Change Log**: This PR fixes the issue where the Claude to OpenAI request does not include include_usage: true when using streaming. \
  **Feature Value**: This fix ensures that users can correctly obtain usage statistics in streaming mode, improving the completeness and user experience of the service.

- **Related PR**: [#3074](https://github.com/alibaba/higress/pull/3074) \
  **Contributor**: @Jing-ze \
  **Change Log**: Added a check for Content-Encoding in the log-request-response plugin to prevent garbled log content due to compressed request/response bodies. \
  **Feature Value**: By improving the log recording mechanism, it ensures that the response body information in the access logs is readable and accurate, enhancing the user experience and system debugging efficiency.

- **Related PR**: [#3069](https://github.com/alibaba/higress/pull/3069) \
  **Contributor**: @Libres-coder \
  **Change Log**: This PR fixes a bug in the CI test framework by adding the go mod tidy command in the prebuild.sh script, ensuring that the go.mod file in the root directory is also updated. \
  **Feature Value**: Solves the issue of CI test failures due to the go.mod file not being correctly updated, ensuring that all PRs triggering e2e tests for wasm plugins pass the CI verification.

- **Related PR**: [#3010](https://github.com/alibaba/higress/pull/3010) \
  **Contributor**: @rinfx \
  **Change Log**: Fixed the parsing failure issue caused by the EventStream response being split, and adjusted the maxtoken conversion logic to ensure data integrity. \
  **Feature Value**: Fixes the EventStream parsing error, improving system stability and reliability, ensuring that users receive accurate data.

- **Related PR**: [#2997](https://github.com/alibaba/higress/pull/2997) \
  **Contributor**: @hanxiantao \
  **Change Log**: Optimized the rate-limiting logic for clusters, AI tokens, and WASM plugins by cumulatively counting request counts and token usage, solving the issue of resetting counters when changing rate limit values. \
  **Feature Value**: Ensures that even when adjusting rate limit thresholds, the existing request count or token usage is not reset, providing a more accurate and reliable rate-limiting mechanism.

- **Related PR**: [#2988](https://github.com/alibaba/higress/pull/2988) \
  **Contributor**: @johnlanni \
  **Change Log**: Corrected the issue in jsonrpc-converter by using the original JSON instead of the incorrect JSON string formatting method for data processing. \
  **Feature Value**: Solves the data processing issue caused by JSON formatting errors, improving system stability and reliability, ensuring that users receive accurate data responses.

- **Related PR**: [#2973](https://github.com/alibaba/higress/pull/2973) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR solves the compatibility issue caused by setting `match_rule_domain` to an empty string by always using a wildcard when generating `mcp-session` configurations, avoiding incompatibility with Higress 2.1.8. \
  **Feature Value**: Fixes the issue caused by setting `match_rule_domain` to an empty string, improving system stability and compatibility, allowing users to use the MCP server without encountering errors due to version differences.

- **Related PR**: [#2952](https://github.com/alibaba/higress/pull/2952) \
  **Contributor**: @Erica177 \
  **Change Log**: Corrected the json tag of the Id field in the ToolSecurity struct from type to id to ensure correct mapping during data serialization. \
  **Feature Value**: This fix resolves the data parsing issue caused by incorrect json tags, improving system stability and data accuracy, enhancing the user experience.

- **Related PR**: [#2948](https://github.com/alibaba/higress/pull/2948) \
  **Contributor**: @johnlanni \
  **Change Log**: Corrected the Azure service URL type detection logic, added support for the Azure OpenAI Response API, and improved streaming event parsing. \
  **Feature Value**: Improves the stability and compatibility of Azure OpenAI integration, ensuring that custom paths and response APIs are correctly handled, enhancing the user experience.

- **Related PR**: [#2941](https://github.com/alibaba/higress/pull/2941) \
  **Contributor**: @rinfx \
  **Change Log**: This PR fixes compatibility issues with old configurations by adjusting the definition of data structures in the `main.go` file to support older versions. \
  **Feature Value**: Improves backward compatibility, ensuring that existing users do not encounter functional anomalies due to changes in configuration format when upgrading to the new version.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#3119](https://github.com/alibaba/higress/pull/3119) \
  **Contributor**: @johnlanni \
  **Change Log**: Replaced reqChan and deltaReqChan in Connection with channels.Unbounded to prevent deadlock issues caused by HTTP2 flow control. \
  **Feature Value**: By avoiding deadlocks caused by HTTP2 flow control, it ensures smooth handling of client requests and responses, improving system stability and performance.

- **Related PR**: [#3113](https://github.com/alibaba/higress/pull/3113) \
  **Contributor**: @johnlanni \
  **Change Log**: Implemented recursive hash calculation and caching for Protobuf messages using the xxHash algorithm, with special handling for google.protobuf.Any type and map fields. \
  **Feature Value**: Optimizes LDS performance by reducing redundant serialization operations in filter chain matching and listener processing, thereby improving overall system efficiency.

- **Related PR**: [#2945](https://github.com/alibaba/higress/pull/2945) \
  **Contributor**: @rinfx \
  **Change Log**: This PR optimizes the pod selection logic in the ai-load-balancer by updating the Lua script for global minimum request count, reducing unnecessary code lines and improving performance. \
  **Feature Value**: The optimized load balancing strategy more efficiently distributes requests, reducing latency and resource waste, enhancing user experience and service stability.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2965](https://github.com/alibaba/higress/pull/2965) \
  **Contributor**: @CH3CHO \
  **Change Log**: Updated the description of azureServiceUrl in the ai-proxy README to ensure the documentation accurately reflects the actual purpose of the configuration item. \
  **Feature Value**: By improving the description of the azureServiceUrl field, it helps users better understand its role and configuration method, enhancing the readability and practicality of the documentation.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#3110](https://github.com/alibaba/higress/pull/3110) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR adds the CODECOV_TOKEN environment variable in the CI workflow to ensure the Codecov upload step is correctly authenticated. \
  **Feature Value**: By adding CODECOV_TOKEN, it improves the accuracy and security of code coverage reports, helping project maintainers better monitor and improve test coverage.

- **Related PR**: [#3097](https://github.com/alibaba/higress/pull/3097) \
  **Contributor**: @johnlanni \
  **Change Log**: Added unit test code for the mcp-server plugin to ensure the stability and reliability of its core functionalities. \
  **Feature Value**: By adding unit tests, the quality of the mcp-server plugin is improved, enhancing system robustness and reducing the likelihood of potential errors.

- **Related PR**: [#2998](https://github.com/alibaba/higress/pull/2998) \
  **Contributor**: @Patrisam \
  **Change Log**: This PR implements end-to-end test cases for Cloudflare, adding content to go-wasm-ai-proxy.go and go-wasm-ai-proxy.yaml. \
  **Feature Value**: By adding Cloudflare end-to-end test cases, it improves the reliability and stability of the system, helping developers better understand and validate the integrated system's performance.

- **Related PR**: [#2980](https://github.com/alibaba/higress/pull/2980) \
  **Contributor**: @Jing-ze \
  **Change Log**: Added coverage gating functionality to the WASM Go plugin unit test workflow, including detailed coverage information display and a 80% coverage threshold setting. \
  **Feature Value**: By increasing the coverage requirements in the CI process, it ensures the quality and stability of the WASM Go plugin, helping developers promptly identify potential issues.

---

## 📊 Release Statistics

- 🚀 New Features: 23 items
- 🐛 Bug Fixes: 13 items
- ♻️ Refactoring and Optimization: 3 items
- 📚 Documentation Updates: 1 item
- 🧪 Testing Improvements: 4 items

**Total**: 44 changes (including 3 key updates)

Thank you to all contributors for their hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **18** updates, covering various aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 7
- **Bug Fixes**: 10
- **Documentation Updates**: 1

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#621](https://github.com/higress-group/higress-console/pull/621) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: This PR optimizes some interaction capabilities of the MCP Server, including header rewriting in direct routing scenarios, support for selecting transport types, and special character handling in DB to MCP Server scenarios. \
  **Feature Value**: By improving the interaction methods and processing capabilities of the MCP Server, the system's flexibility and compatibility are enhanced, making it more convenient for users to configure and use backend services, thereby improving the user experience.

- **Related PR**: [#612](https://github.com/higress-group/higress-console/pull/612) \
  **Contributor**: @zhwaaaaaa \
  **Change Log**: This PR resolves the issue of Grafana pages not working properly due to the reverse proxy server sending a `transfer-encoding: chunked` header by adding support for ignoring hop-to-hop headers. \
  **Feature Value**: This feature ensures that even in complex network environments (such as when using a reverse proxy), the Grafana monitoring dashboard can display information correctly, enhancing user experience and system compatibility.

- **Related PR**: [#608](https://github.com/higress-group/higress-console/pull/608) \
  **Contributor**: @Libres-coder \
  **Change Log**: This PR adds plugin display support to the AI routing management page, allowing users to view enabled plugins and expand the AI routing row for more information. \
  **Feature Value**: This enhancement improves the AI routing management function, enabling users to more intuitively understand and manage the status of plugins in their AI routing configurations, thereby enhancing the user experience.

- **Related PR**: [#604](https://github.com/higress-group/higress-console/pull/604) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR enhances path rewriting flexibility by adding support for using regular expressions with the `higress.io/rewrite-target` annotation. \
  **Feature Value**: This increase in path rewriting flexibility allows users to modify request paths using more complex rules, enhancing the system's customizability and user experience.

- **Related PR**: [#603](https://github.com/higress-group/higress-console/pull/603) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR adds a fixed service port 80 display function to the frontend page for static service sources. It defines a constant and updates the form component to show this port number. \
  **Feature Value**: This provides users with a more intuitive view to identify and confirm the standard HTTP port (80) used by static service sources, simplifying the configuration process and reducing potential misunderstandings.

- **Related PR**: [#602](https://github.com/higress-group/higress-console/pull/602) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR adds a search function when selecting upstream services for AI routing. By introducing a search mechanism in the frontend component, it improves the efficiency of users finding the required services. \
  **Feature Value**: The new search function significantly enhances the user experience, especially when the service list is long, helping users quickly locate the target service and simplifying the configuration process.

- **Related PR**: [#566](https://github.com/higress-group/higress-console/pull/566) \
  **Contributor**: @OuterCyrex \
  **Change Log**: This PR adds support for custom Qwen services, including enabling internet search and uploading file IDs. The main changes involve the backend SDK and frontend pages to support these new features. \
  **Feature Value**: By adding support for custom Qwen services, users can now more flexibly configure their AI services, particularly for applications requiring specific functionalities like internet search or file handling, greatly enhancing the user experience and service customizability.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#620](https://github.com/higress-group/higress-console/pull/620) \
  **Contributor**: @CH3CHO \
  **Change Log**: Corrected a spelling error in the `sortWasmPluginMatchRules` logic to ensure correct sorting of match rules. \
  **Feature Value**: Fixing the spelling error improves code accuracy and readability, avoiding potential logical errors due to misspellings, thereby enhancing system stability and user experience.

- **Related PR**: [#619](https://github.com/higress-group/higress-console/pull/619) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR removes version information from the data JSON when converting from AiRoute to ConfigMap, as these details are already stored in the ConfigMap metadata. \
  **Feature Value**: Removing redundant version information helps reduce data duplication and ensures that the information stored in the ConfigMap is more concise and clear, improving maintainability and consistency.

- **Related PR**: [#618](https://github.com/higress-group/higress-console/pull/618) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR refactors the API authentication logic in the SystemController to eliminate known security vulnerabilities. It introduces a new `AllowAnonymous` annotation and updates the relevant controllers to ensure system security. \
  **Feature Value**: This fix eliminates security vulnerabilities in the system, enhancing overall security and protecting users from potential attack threats, thus increasing user trust and experience.

- **Related PR**: [#617](https://github.com/higress-group/higress-console/pull/617) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed warnings about missing unique key attributes in list items in the frontend console, issues with image loading violating the content security policy, and incorrect type for the Consumer.name field. \
  **Feature Value**: Addressing these frontend errors encountered by users enhances the user experience and system stability.

- **Related PR**: [#614](https://github.com/higress-group/higress-console/pull/614) \
  **Contributor**: @lc0138 \
  **Change Log**: This PR corrects an error in the ServiceSource type field and adds dictionary value validation logic to ensure the field's accuracy. \
  **Feature Value**: Fixing the error in the service source type field enhances system stability and data consistency, preventing issues caused by type mismatches.

- **Related PR**: [#613](https://github.com/higress-group/higress-console/pull/613) \
  **Contributor**: @lc0138 \
  **Change Log**: This PR addresses CSP and other security risks in the frontend documentation by adding specific meta tags to enhance web page security. \
  **Feature Value**: Enhancing the security of the web application reduces potential security threats, improving the security of user usage.

- **Related PR**: [#611](https://github.com/higress-group/higress-console/pull/611) \
  **Contributor**: @qshuai \
  **Change Log**: Corrected a spelling error in the API method annotations of the LlmProvidersController class, changing 'Add a new route' to the correct description. \
  **Feature Value**: Although a small fix, it ensures the accuracy of the API documentation, helping developers better understand and use the API interfaces.

- **Related PR**: [#609](https://github.com/higress-group/higress-console/pull/609) \
  **Contributor**: @CH3CHO \
  **Change Log**: Corrected the type error of the name field in the Consumer interface, changing it from a boolean to a string. \
  **Feature Value**: This fix ensures the correct data type for the Consumer.name field, avoiding runtime errors due to type mismatches, thus enhancing system stability and data accuracy.

- **Related PR**: [#605](https://github.com/higress-group/higress-console/pull/605) \
  **Contributor**: @SaladDay \
  **Change Log**: Corrected the AI routing name validation rules to support periods and only allow lowercase letters, and updated the error message to accurately describe the new validation rules. \
  **Feature Value**: This fix addresses the inconsistency between the UI prompt and the actual validation logic, improving the consistency and accuracy of the user experience, ensuring that users receive the correct feedback when configuring AI routes.

- **Related PR**: [#552](https://github.com/higress-group/higress-console/pull/552) \
  **Contributor**: @lcfang \
  **Change Log**: To address compatibility issues caused by inconsistent backend service ports, a new vport attribute was added. When the service instance port in the registry changes, the default or specified virtual port can be configured through vport to maintain the validity of the routing configuration. \
  **Feature Value**: This PR enhances system stability and reliability, ensuring that routing configurations do not fail when service ports change, thereby improving the user experience and system availability.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#610](https://github.com/higress-group/higress-console/pull/610) \
  **Contributor**: @heimanba \
  **Change Log**: This PR adjusts the configuration field descriptions in the frontend canary documentation, including making fields like rewrite non-mandatory and updating the associated description for the name field in rules. It also updates the Chinese and English README and spec.yaml files. \
  **Feature Value**: By increasing the flexibility of configuration options and improving compatibility, users can more easily configure according to their actual needs. Additionally, the consistency and accuracy of the documentation are improved, helping to reduce the difficulty of use and enhance the user experience.

---

## 📊 Release Statistics

- 🚀 New Features: 7
- 🐛 Bug Fixes: 10
- 📚 Documentation Updates: 1

**Total**: 18 Changes

Thank you to all contributors for your hard work! 🎉

