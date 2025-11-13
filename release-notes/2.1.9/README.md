# Higress


## 📋 Overview of This Release

This release includes **44** updates, covering various aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 23
- **Bug Fixes**: 14
- **Refactoring Optimizations**: 2
- **Documentation Updates**: 1
- **Testing Improvements**: 4

### ⭐ Key Highlights

This release contains **3** significant updates, which are recommended for special attention:

- **feat(mcp-server): add server-level default authentication and MCP proxy server support** ([#3096](https://github.com/alibaba/higress/pull/3096)): This feature enhances Higress's security management capabilities for MCP traffic, allowing users to set up authentication through a unified interface, simplifying the deployment process of security policies, and enhancing system security and flexibility.
- **feat: add higress api mcp server** ([#2923](https://github.com/alibaba/higress/pull/2923)): By adding the higress-ops MCP Server, users can use the `hgctl agent` command to manage Higress configurations and troubleshoot issues, improving operational efficiency and user experience.
- **feat: implement `hgctl agent` & `mcp add` subcommand** ([#3051](https://github.com/alibaba/higress/pull/3051)): This enhancement improves Higress's operational capabilities, especially through interactive management and debugging via the Agent, making it easier for users to configure and debug MCP traffic governance. It is a significant step towards AI-native operations for Higress.

For more details, please refer to the detailed descriptions of key features below.

---

## 🌟 Detailed Description of Key Features

Below are the detailed explanations of the important features and improvements in this release:

### 1. feat(mcp-server): add server-level default authentication and MCP proxy server support

**Related PR**: [#3096](https://github.com/alibaba/higress/pull/3096) | **Contributor**: [@johnlanni](https://github.com/johnlanni)

**Usage Background**

As the AI-native API gateway Higress evolves, users' requirements for API security, flexibility, and ease of use continue to increase. In practical applications, the MCP (Model Context Protocol) protocol is widely used for managing and invoking AI models. However, the existing MCP servers lack a unified security authentication mechanism, leading to the need for repeated configuration of authentication information in different scenarios. Additionally, for certain scenarios where REST APIs are converted to MCP Servers, an efficient proxy mode is required to handle requests. This update addresses these issues, targeting users including but not limited to developers, operations personnel, and system administrators who need a more secure, flexible, and manageable API gateway.

**Feature Details**

This update primarily implements two core features: 1. Adding default authentication at the MCP server level, including client-to-gateway and gateway-to-backend authentication; 2. Introducing a new type of MCP proxy server that can proxy MCP requests from clients to backend MCP servers, supporting timeout configuration and full authentication support. Technically, this is achieved by updating dependency library versions (such as wasm-go and proxy-wasm-go-sdk) to support the new features, while also refactoring existing code to accommodate the new authentication and proxy logic.

**Usage**

To enable this feature, you need to set the corresponding parameters in the Higress configuration file. For example, to configure default downstream security, specify the authentication policy in the `defaultDownstreamSecurity` field; similarly, upstream authentication is configured through the `defaultUpstreamSecurity` field. To use the MCP proxy server, define a new `mcp-proxy` type server and specify the backend MCP server address via the `mcpServerURL` field. Additionally, you can control the request timeout time using the `timeout` field. Best practices recommend utilizing the priority configuration mechanism to ensure that tool-level settings can override server-level defaults, thereby achieving finer-grained control.

**Feature Value**

This feature significantly enhances the security and flexibility of Higress, making API management more efficient. By introducing server-level default authentication, it reduces the workload of repetitive configurations and lowers the security risks associated with configuration errors. The MCP proxy server not only simplifies the complexity in the REST to MCP conversion process but also offloads state maintenance tasks to Higress, effectively reducing the load on backend MCP servers. These improvements collectively contribute to the stability and user experience of the entire ecosystem, laying a solid foundation for Higress to become an indispensable API gateway in the AI era.

---

### 2. feat: add higress api mcp server

**Related PR**: [#2923](https://github.com/alibaba/higress/pull/2923) | **Contributor**: [@Tsukilc](https://github.com/Tsukilc)

**Usage Background**

As AI technology advances, API gateways need to better support AI-related functionalities. Higress, as an AI-native API gateway, needs to provide more powerful management tools to unify the management of core API assets such as LLM APIs, MCP APIs, and Agent APIs. This PR integrates the Higress API MCP Server, providing comprehensive management capabilities for AI routing, AI providers, and MCP servers. These new features help users more efficiently configure and maintain Higress's AI features, meeting the needs of modern applications. The target user groups include Higress operators and developers, especially those with deep needs in the AI domain.

**Feature Details**

This PR mainly implements the following features:
1. **AI Routing Management**: Added tools such as `list-ai-routes`, `get-ai-route`, `add-ai-route`, `update-ai-route`, and `delete-ai-route` to allow users to manage AI routes.
2. **AI Provider Management**: Added tools such as `list-ai-providers`, `get-ai-provider`, `add-ai-provider`, `update-ai-provider`, and `delete-ai-provider` to allow users to manage AI providers.
3. **MCP Server Management**: Added tools such as `list-mcp-servers`, `get-mcp-server`, `add-or-update-mcp-server`, and `delete-mcp-server` to allow users to manage MCP servers and their consumers.
4. **Authentication Configuration**: Uses HTTP Basic Authentication for authorization, carrying the `Authorization` header in the client request.
5. **Code Changes**: Removed hard-coded usernames and passwords, instead providing them at runtime via the MCP Client, enhancing security. Additionally, added the `higress-ops` module for `hgctl agent` command integration, enabling Agent-based management of Higress configurations.

**Usage**

To enable and configure this feature, follow these steps:
1. **Configure Higress API MCP Server**: Add the Higress API MCP Server configuration in the Higress configuration file, specifying the URL of the Higress Console.
2. **Use `hgctl agent`**: Start the interactive Agent using the `hgctl agent` command, allowing you to manage Higress using natural language. For example, use the `mcp add` subcommand to add a remote MCP Server to the Higress MCP management directory.
3. **Manage AI Routes**: Use tools like `list-ai-routes`, `get-ai-route`, `add-ai-route`, `update-ai-route`, and `delete-ai-route` to manage AI routes.
4. **Manage AI Providers**: Use tools like `list-ai-providers`, `get-ai-provider`, `add-ai-provider`, `update-ai-provider`, and `delete-ai-provider` to manage AI providers.
5. **Manage MCP Servers**: Use tools like `list-mcp-servers`, `get-mcp-server`, `add-or-update-mcp-server`, and `delete-mcp-server` to manage MCP servers and their consumers.
**Note**: Ensure that you correctly configure the authentication information and carry the `Authorization` header in the request.

**Feature Value**

This feature brings the following specific benefits to users:
1. **Enhanced Management Capabilities**: Users can more easily manage and debug Higress's AI routing, AI provider, and MCP server configurations using the new MCP tools, improving management efficiency.
2. **Higher Security**: By providing usernames and passwords at runtime via the MCP Client rather than hard-coding them in the configuration file, the system's security is enhanced.
3. **Better User Experience**: The interactive management method via `hgctl agent` allows users to manage Higress using natural language, reducing the learning curve and difficulty of use.
4. **Improved System Performance and Stability**: The new MCP tools provide more management and debugging options, helping to promptly identify and resolve issues, thereby improving system stability and performance.
5. **Ecosystem Importance**: As the first step for Higress to transition from traditional operations to Agent-based operations, this feature is significant for the development of the Higress ecosystem, laying the groundwork for future innovations.

---

### 3. feat: implement `hgctl agent` & `mcp add` subcommand 

**Related PR**: [#3051](https://github.com/alibaba/higress/pull/3051) | **Contributor**: [@erasernoob](https://github.com/erasernoob)

**Usage Background**

Higress is an AI-native API gateway used to unify the management of LLM APIs, MCP APIs, and Agent APIs. As Higress evolves, traditional command-line tools no longer meet user needs, especially in the management and debugging of MCP services. This PR introduces an interactive Agent similar to Claude Code, allowing users to manage Higress using natural language. Additionally, the new `mcp add` subcommand makes it easy to add remote MCP services to Higress's MCP management directory, enabling MCP traffic governance. These features not only simplify the configuration process for MCP services but also enhance the system's maintainability and usability.

**Feature Details**

This PR mainly implements two new subcommands: `hgctl agent` and `mcp add`.

- `hgctl agent`: This command allows users to interact with Higress using natural language. It calls the underlying `claude-code` agent and prompts the user to set up the necessary environment upon first use. `hgctl agent` provides an interactive window, enabling users to manage Higress in a more intuitive manner.

- `mcp add`: This command allows users to add MCP services with simple parameters. It supports two types of MCP services: direct proxy type and OpenAPI-based type. Direct proxy type MCP services can directly call the Higress Console API and publish to the Higress MCP Server management tool. OpenAPI-based MCP services generate MCP configurations by parsing the OpenAPI specification. The code changes include the addition of multiple files and a significant amount of code, including `agent.go`, `base.go`, `core.go`, `mcp.go`, and `client.go`, which collectively implement the above features.

**Usage**

To enable and configure these new features, users need to update to the latest version of the `hgctl` tool.

1. **Enable `hgctl agent`**:
   - Run the `hgctl agent` command. On the first use, it will prompt the user to set up the necessary environment, such as installing the `claude-code` agent.
   - Interact with Higress using natural language, for example, to query or modify configurations.

2. **Add MCP Services Using `mcp add`**:
   - Add a direct proxy type MCP service:
     ```bash
     hgctl mcp add mcp-deepwiki -t http https://mcp.deepwiki.com --user admin --password 123 --url http://localhost:8080
     ```
   - Add an OpenAPI-based MCP service:
     ```bash
     hgctl mcp add openapi-server -t openapi --spec openapi.yaml --user admin --password 123 --url http://localhost:8080
     ```

**Note**: Ensure that the system has correctly configured Higress and related dependencies before running these commands.

**Feature Value**

These new features bring significant benefits to users, including:

- **Improved User Experience**: Through natural language interaction, the learning curve for users is reduced, making Higress management more intuitive and user-friendly.
- **Simplified Configuration Process**: The `mcp add` command greatly simplifies the process of adding and configuring MCP services, reducing the complexity and error rate of manual operations.
- **Enhanced System Stability**: With unified MCP service management, it is easier to monitor and maintain MCP traffic, improving the system's stability and reliability.
- **Expanded Ecosystem**: These new features enable Higress to better support different types of MCP services, enhancing its competitiveness and ecosystem influence in the AI era.

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#3126](https://github.com/alibaba/higress/pull/3126) \
  **Contributor**: @johnlanni \
  **Change Log**: Updated Envoy dependencies, supporting the setting of Redis call-related parameters via WASM, such as `buffer_flush_timeout` and `max_buffer_size_before_flush`. \
  **Feature Value**: This feature enhances the flexibility of the WASM plugin, allowing users to customize Redis client buffer behavior through URL query parameters, improving the convenience and efficiency of configuration management.

- **Related PR**: [#3123](https://github.com/alibaba/higress/pull/3123) \
  **Contributor**: @johnlanni \
  **Change Log**: Upgraded the Higress proxy version to v2.2.0, updated the Go toolchain and multiple dependency package versions, and added specific architecture build targets for golang-filter, fixing dependency issues related to MCP servers, OpenAI, and Milvus SDK. \
  **Feature Value**: This improvement enhances the overall performance and stability of Higress, supporting more architecture types and enhancing support for the latest technology stack. For users, this means broader compatibility, better security, and richer feature expansion possibilities.

- **Related PR**: [#3108](https://github.com/alibaba/higress/pull/3108) \
  **Contributor**: @wydream \
  **Change Log**: Added video-related API paths and capabilities, including constants, default capabilities, and regular expression path handling, enabling the proxy to correctly parse multiple video-related endpoints and updating the OpenAI provider to optimize support for these new endpoints. \
  **Feature Value**: By adding support for video-related APIs, this enhancement strengthens Higress's ability to manage AI services, particularly for applications that need to handle video content. This will make it easier for users to integrate and use advanced features involving video.

- **Related PR**: [#3071](https://github.com/alibaba/higress/pull/3071) \
  **Contributor**: @rinfx \
  **Change Log**: The PR added an example of using the `inject_encoded_data_to_filter_chain_on_header` function, demonstrating how to add body data to a request when there is no response body. This was achieved by modifying README.md, go.mod, and other files. \
  **Feature Value**: This feature allows users to add body data to a request even when there is no response body, enhancing the API gateway's ability to handle requests flexibly and dynamically, especially when generating or modifying response content.

- **Related PR**: [#3067](https://github.com/alibaba/higress/pull/3067) \
  **Contributor**: @wydream \
  **Change Log**: This PR added support for vLLM as an AI provider in the Higress ai-proxy plugin, implementing multiple API interfaces compatible with OpenAI, including chat and text completion, model list display, and other functions. \
  **Feature Value**: By introducing vLLM as a new AI service provider, users can now directly access various AI capabilities provided by vLLM through the Higress proxy, such as generating text. This greatly enriches the availability of Higress in AI application scenarios and simplifies the integration process.

- **Related PR**: [#3060](https://github.com/alibaba/higress/pull/3060) \
  **Contributor**: @erasernoob \
  **Change Log**: This PR enhanced the `hgctl mcp` and `hgctl agent` commands to automatically obtain Higress Console credentials from installation configuration files and Kubernetes secrets, optimizing the user experience. \
  **Feature Value**: This feature reduces the steps required for users to manually enter credentials, improving operational convenience and security, especially when Higress is installed via `hgctl`. It is a significant usability improvement for users.

- **Related PR**: [#3043](https://github.com/alibaba/higress/pull/3043) \
  **Contributor**: @2456868764 \
  **Change Log**: This PR fixed the issue of incorrect default port for Milvus and added Python example code to the README.md. The port issue was resolved by modifying the `match_rule_domain` field in the configuration file, and usage guidance was provided. \
  **Feature Value**: This fix resolves the port configuration issue that could lead to service failure, enhancing the practicality of the documentation. It provides a specific Python example to help users understand and quickly get started with the plugin functionality.

- **Related PR**: [#3040](https://github.com/alibaba/higress/pull/3040) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: This PR added the ApiNameAnthropicMessages feature for Anthropic and supported configuring the Anthropique provider without setting `protocol=original`, allowing `/v1/messages` requests to be directly forwarded to Anthropic, while `/v1/chat/completion` requests convert the OpenAI format message body to a Claude-compatible format. \
  **Feature Value**: By adding support for the Anthropic messages API, this feature enhances Higress's ability to manage different types of AI services. Users can now more flexibly use services provided by Anthropic, especially when interacting with Claude, increasing the platform's diversity and flexibility.

- **Related PR**: [#3038](https://github.com/alibaba/higress/pull/3038) \
  **Contributor**: @Libres-coder \
  **Change Log**: Added the `list-plugin-instances` tool, allowing AI proxies to query plugin instances within a specific scope using the MCP protocol. This PR added two new functions to the MCP Server and updated the relevant documentation. \
  **Feature Value**: This feature enables users to more conveniently manage plugin configurations in Higress, enhancing the system's manageability and transparency, especially when checking or adjusting the status of plugins within a specific scope.

- **Related PR**: [#3032](https://github.com/alibaba/higress/pull/3032) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR enabled Qwen compatibility mode by default and added missing API endpoints, including AsyncAIGC, AsyncTask, and V1Rerank, to provide more comprehensive API coverage. \
  **Feature Value**: By enabling compatibility mode by default and filling in API endpoint gaps, this feature enhances the out-of-the-box experience for users and strengthens Higress's support for Qwen AI services, making it easier for developers to integrate and use Qwen-related features.

- **Related PR**: [#3029](https://github.com/alibaba/higress/pull/3029) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: Added support for `v1/responses` in the groq provider, specifically by introducing new response handling logic. \
  **Feature Value**: This new feature supports better management and utilization of the services provided by the groq plugin, enhancing the flexibility and completeness of API management.

- **Related PR**: [#3024](https://github.com/alibaba/higress/pull/3024) \
  **Contributor**: @rinfx \
  **Change Log**: Added malicious URL and model hallucination detection to ensure the security of AI-generated content; also adjusted specific configurations at the consumer level to better adapt to different scenario needs. \
  **Feature Value**: By adding detection for malicious URLs and model hallucinations, this feature enhances the security and accuracy of Higress in handling AI-generated content, helping to protect users from potential threats. Additionally, the adjusted consumer-level configurations enhance the system's flexibility and adaptability.

- **Related PR**: [#3008](https://github.com/alibaba/higress/pull/3008) \
  **Contributor**: @hellocn9 \
  **Change Log**: This PR added support for custom parameter names for MCP SSE stateful sessions. By adding the `higress.io/mcp-sse-stateful-param-name` annotation in the ingress configuration, users can specify their own parameter names. \
  **Feature Value**: This feature allows users to flexibly set the parameter names for MCP SSE stateful sessions according to their needs, improving configuration flexibility and user experience. This makes Higress better suited for diverse application scenarios.

- **Related PR**: [#3006](https://github.com/alibaba/higress/pull/3006) \
  **Contributor**: @SaladDay \
  **Change Log**: This PR added Secret reference support for Redis configuration in the MCP Server, allowing Redis passwords to be stored in Kubernetes Secrets, enhancing security, and modified the relevant documentation and test code. \
  **Feature Value**: By storing Redis passwords in Kubernetes Secrets instead of writing them in plaintext in ConfigMaps, this improvement enhances system security. Users can more securely manage sensitive information, reducing the risk of password leaks.

- **Related PR**: [#2992](https://github.com/alibaba/higress/pull/2992) \
  **Contributor**: @rinfx \
  **Change Log**: This PR modified the authentication logic in the `key_auth` plugin, logging the consumer name in the logs even if the access is not authorized. By adding logging of consumer identification during the authentication and authorization process, it enhances the observability of the system. \
  **Feature Value**: This feature improves the efficiency of system monitoring and troubleshooting, allowing operations personnel to clearly understand the source of requests, even if they are not authorized, thus better diagnosing issues and conducting security audits.

- **Related PR**: [#2978](https://github.com/alibaba/higress/pull/2978) \
  **Contributor**: @rinfx \
  **Change Log**: After determining the consumer name, it adds the consumer name to the request header regardless of whether the authentication is successful, for subsequent processing. \
  **Feature Value**: This feature enhances the ability to track consumer behavior, helping to better understand API call patterns and consumer activity, thus providing a more personalized service experience for users.

- **Related PR**: [#2968](https://github.com/alibaba/higress/pull/2968) \
  **Contributor**: @2456868764 \
  **Change Log**: Added vector database mapping functionality, introducing a field mapping system and index configuration management mechanism, supporting various index types such as HNSW, IVF, SCANN, etc., to improve system flexibility and adaptability. \
  **Feature Value**: By providing flexible field mapping and rich index configuration options, this feature enhances support for different vector databases, simplifying the process for developers to integrate various storage solutions and improving the user experience.

- **Related PR**: [#2943](https://github.com/alibaba/higress/pull/2943) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: This PR added the ability to support custom system prompts when generating release notes, achieved by modifying the GitHub Actions workflow file. \
  **Feature Value**: This feature allows users to add personalized system prompts when generating release notes, enhancing the flexibility and practicality of the release notes and better meeting the needs of different projects.

- **Related PR**: [#2942](https://github.com/alibaba/higress/pull/2942) \
  **Contributor**: @2456868764 \
  **Change Log**: Fixed the handling logic when the LLM provider is empty and optimized the document structure and content, adding detailed introductions to MCP tools. \
  **Feature Value**: This improvement enhances the robustness of the system when LLM configuration is missing, enhancing the user's understanding and experience with MCP tools, making it clearer for users to understand the functions and configuration requirements of different tools.

- **Related PR**: [#2916](https://github.com/alibaba/higress/pull/2916) \
  **Contributor**: @imp2002 \
  **Change Log**: Implemented Nginx migration to MCP servers and provided 7 MCP tools to automate the migration process from Nginx configuration/Lua plugins to Higress, including important features such as configuration conversion. \
  **Feature Value**: This feature greatly simplifies the effort required for users to migrate from Nginx to Higress, providing a complete set of tools to make the migration process smoother and more efficient, helping users adopt Higress as their API gateway solution more quickly.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#3120](https://github.com/alibaba/higress/pull/3120) \
  **Contributor**: @lexburner \
  **Change Log**: Adjusted the log level in the ai-proxy component, specifically in the `wasm-go/extensions/ai-proxy/provider/qwen.go` file, reducing unnecessary warning messages. \
  **Feature Value**: By lowering the log level in specific parts, this change reduces redundant warning messages during system operation, improving the efficiency of developers and operations personnel in viewing logs, allowing them to focus more on actual errors or important information.

- **Related PR**: [#3119](https://github.com/alibaba/higress/pull/3119) \
  **Contributor**: @johnlanni \
  **Change Log**: Updated the Istio dependency and replaced `reqChan` and `deltaReqChan` in the Connection with `channels.Unbounded` to prevent deadlock issues caused by HTTP2 flow control. \
  **Feature Value**: By resolving the deadlock issue caused by HTTP2 flow control, this improvement ensures that client requests and ACK requests can be processed normally without blocking, enhancing the stability and response speed of the system.

- **Related PR**: [#3118](https://github.com/alibaba/higress/pull/3118) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixed the issue where port-level policies unconditionally overwrite existing configurations converted from Ingress annotations. By adding nil checks before setting `policy.Tls` and `policy.LoadBalancer`, it avoids overwriting existing configurations. \
  **Feature Value**: This fix resolves the unexpected configuration overwrite issue caused by TLS and load balancer settings in DestinationRule, ensuring that user-defined Ingress annotation configurations are correctly retained and applied, enhancing the stability and reliability of the system.

- **Related PR**: [#3095](https://github.com/alibaba/higress/pull/3095) \
  **Contributor**: @rinfx \
  **Change Log**: Fixed the issue of usage information being lost during the `claude2openai` conversion process and added the `index` field in the bedrock streaming tool response to ensure data integrity and accuracy. \
  **Feature Value**: This fix enhances the system's data integrity when handling API conversions, ensuring that users can accurately obtain all necessary usage information, especially in the case of streaming responses, by introducing the `index` field to enhance response management flexibility.

- **Related PR**: [#3084](https://github.com/alibaba/higress/pull/3084) \
  **Contributor**: @rinfx \
  **Change Log**: Fixed the issue where the `include_usage: true` flag was not correctly included when converting Claude requests to OpenAI requests, ensuring that usage information is properly passed in streaming response mode. \
  **Feature Value**: This fix allows users to receive more accurate resource usage feedback when using streaming APIs, enhancing the accuracy of resource consumption monitoring.

- **Related PR**: [#3074](https://github.com/alibaba/higress/pull/3074) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR added a check for `Content-Encoding` in the `log-request-response` plugin to avoid logging compressed request/response bodies, which can result in garbled log entries. \
  **Feature Value**: By improving the logging mechanism to prevent unreadable log entries, this change enhances the efficiency and accuracy of system operations personnel in troubleshooting issues.

- **Related PR**: [#3069](https://github.com/alibaba/higress/pull/3069) \
  **Contributor**: @Libres-coder \
  **Change Log**: This PR fixed an issue in the CI testing framework where e2e tests failed due to the `go.mod` file not being correctly updated. By adding the `go mod tidy` command in the `prebuild.sh` script, it ensures that the `go.mod` in the root directory is also updated. \
  **Feature Value**: This fix resolves the CI test failure issue that all PRs triggering end-to-end testing of the wasm plugin might encounter, ensuring the stability of the build and test process and improving the developer experience.

- **Related PR**: [#3010](https://github.com/alibaba/higress/pull/3010) \
  **Contributor**: @rinfx \
  **Change Log**: Fixed the issue of parsing failures in bedrock responses due to unpacking problems and adjusted the `maxtoken` conversion logic to ensure the accuracy and integrity of event stream processing. \
  **Feature Value**: This fix resolves the data parsing error issue encountered by users when using bedrock services, enhancing the stability and user experience of the system. By optimizing boundary condition handling, it ensures the consistency of data transmission.

- **Related PR**: [#2997](https://github.com/alibaba/higress/pull/2997) \
  **Contributor**: @hanxiantao \
  **Change Log**: Optimized the cluster rate limiting and AI token rate limiting logic, changing to cumulative counting of request counts and token usage, avoiding reset of counters when changing rate limit values. \
  **Feature Value**: By improving the rate limiting mechanism, this change ensures that the system can accurately track and limit request traffic even after changing the rate limit thresholds, thereby enhancing the stability and reliability of the system.

- **Related PR**: [#2988](https://github.com/alibaba/higress/pull/2988) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixed the JSON formatting error in the `jsonrpc-converter`, switching to using raw JSON data to avoid data parsing issues caused by string formatting. \
  **Feature Value**: By correcting the JSON handling method, this change ensures the accuracy and consistency of data transmission, enhancing the stability and reliability of the system and reducing potential issues caused by data format errors.

- **Related PR**: [#2973](https://github.com/alibaba/higress/pull/2973) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed the issue where the Higress 2.1.8 version did not support an empty `match_rule_domain` by using a wildcard to match all domains, eliminating compatibility risks. \
  **Feature Value**: This fix ensures that the generation of MCP server configurations is backward-compatible with older versions, avoiding service interruptions or behavioral anomalies due to configuration errors, enhancing the stability and user experience of the system.

- **Related PR**: [#2952](https://github.com/alibaba/higress/pull/2952) \
  **Contributor**: @Erica177 \
  **Change Log**: Corrected the JSON tag for the `Id` field in the `ToolSecurity` struct, changing it from `type` to `id`, to ensure correct serialization. \
  **Feature Value**: This fix ensures the correctness of the `ToolSecurity` struct during data transmission, avoiding data parsing issues caused by incorrect field tags, enhancing the stability and user experience of the system.

- **Related PR**: [#2948](https://github.com/alibaba/higress/pull/2948) \
  **Contributor**: @johnlanni \
  **Change Log**: Fixed the handling issue with the Azure OpenAI Response API and the service URL type detection logic, including adding support for custom full paths and improving streaming event parsing. \
  **Feature Value**: This enhancement improves support for Azure OpenAI services, enhancing the accuracy and efficiency of API response handling, allowing users to more stably use Azure OpenAI-related features.

- **Related PR**: [#2941](https://github.com/alibaba/higress/pull/2941) \
  **Contributor**: @rinfx \
  **Change Log**: This PR fixed the compatibility issue between the `ai-security-guard` plugin and old configurations, by adjusting the relevant code in the `main.go` file to ensure backward compatibility. \
  **Feature Value**: This fix resolves the compatibility issue caused by configuration updates, allowing users with old configurations to seamlessly transition to the new version, enhancing the user experience and stability of the system.

### ♻️ Refactoring Optimizations (Refactoring)

- **Related PR**: [#3113](https://github.com/alibaba/higress/pull/3113) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR implemented a hash cache for Protobuf messages, using the xxHash algorithm for recursive hash calculation and handling `google.protobuf.Any` types and deterministically sorted map fields specially, optimizing LDS performance. \
  **Feature Value**: This change significantly improves the efficiency of Envoy in handling large-scale configuration updates, reducing performance overhead due to repeated serialization. In environments with frequent changes or large-scale deployments, it accelerates the propagation of configurations and enhances system responsiveness.

- **Related PR**: [#2945](https://github.com/alibaba/higress/pull/2945) \
  **Contributor**: @rinfx \
  **Change Log**: Optimized the Lua script logic for selecting pods with the global minimum number of requests in `ai-load-balancer`, improving request distribution efficiency by adjusting the health check mechanism and load balancing strategy. \
  **Feature Value**: This change enhances the fairness and efficiency of the AI load balancer in handling requests, reducing the service response time extension caused by a single node being overloaded, positively impacting the overall system stability and user experience.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2965](https://github.com/alibaba/higress/pull/2965) \
  **Contributor**: @CH3CHO \
  **Change Log**: Updated the description of `azureServiceUrl` in the ai-proxy README file, adding detailed information about the use of this parameter to help users better understand and configure it. \
  **Feature Value**: By providing a more detailed description of the `azureServiceUrl` parameter, this change improves the user experience, making it easier for users to correctly configure settings according to the documentation, thus avoiding potential usage errors.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#3110](https://github.com/alibaba/higress/pull/3110) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR added the `CODECOV_TOKEN` environment variable configuration in the GitHub Actions workflow to ensure that Codecov can correctly authenticate and upload code coverage data. \
  **Feature Value**: By adding the `CODECOV_TOKEN` environment variable, this improvement enhances the security and reliability of the CI/CD process, ensuring the accuracy and completeness of code coverage reports, which helps in maintaining project quality.

- **Related PR**: [#3097](https://github.com/alibaba/higress/pull/3097) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR added unit tests for the mcp-server, adding a total of 2766 lines of code, primarily in the `main_test.go` file, enhancing the test coverage of the mcp-server. \
  **Feature Value**: By adding unit tests, this improvement enhances the stability and reliability of the mcp-server module, ensuring that new features or fixes do not introduce new issues. For users, this improves the overall quality assurance and user experience of Higress.

- **Related PR**: [#2998](https://github.com/alibaba/higress/pull/2998) \
  **Contributor**: @Patrisam \
  **Change Log**: This PR implemented end-to-end test cases for Cloudflare, enhancing the test coverage of the Higress project. By adding new test logic and configurations in `go-wasm-ai-proxy.go` and `go-wasm-ai-proxy.yaml`, it improved system integration. \
  **Feature Value**: The newly added Cloudflare e2e test cases help ensure the compatibility and stability between Higress and Cloudflare services, greatly enhancing the confidence of users who use or plan to use Cloudflare as part of their network infrastructure.

- **Related PR**: [#2980](https://github.com/alibaba/higress/pull/2980) \
  **Contributor**: @Jing-ze \
  **Change Log**: Enhanced the CI workflow for WASM plugin unit tests, adding coverage display functionality and setting an 80% coverage threshold. \
  **Feature Value**: This improvement enhances the quality and transparency of the testing process, ensuring that the WASM plugin meets a certain code coverage standard, which helps in identifying potential issues and improving code reliability.

---

## 📊 Release Statistics

- 🚀 New Features: 23
- 🐛 Bug Fixes: 14
- ♻️ Refactoring Optimizations: 2
- 📚 Documentation Updates: 1
- 🧪 Testing Improvements: 4

**Total**: 44 changes (including 3 key updates)

Thank you to all contributors for your hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **18** updates, covering enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 7 items
- **Bug Fixes**: 10 items
- **Documentation Updates**: 1 item

---

## 📝 Complete Changelog

### 🚀 New Features (Features)

- **Related PR**: [#621](https://github.com/higress-group/higress-console/pull/621) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: This PR enhances the interaction capabilities of the MCP Server, including rewriting the header host, modifying the interaction method to support transport selection, and improving DSN character handling logic to support the special character @. \
  **Feature Value**: These improvements allow users to configure and use the MCP Server more flexibly, especially in direct routing scenarios, where DNS addresses and service paths can be handled better, enhancing system flexibility and usability.

- **Related PR**: [#612](https://github.com/higress-group/higress-console/pull/612) \
  **Contributor**: @zhwaaaaaa \
  **Change Log**: Added handling to ignore hop-by-hop headers in DashboardServiceImpl, preventing headers like `Transfer-Encoding: chunked` from being mistakenly passed. \
  **Feature Value**: By correctly handling hop-by-hop headers, it ensures that the Grafana page works properly in environments with reverse proxy servers, improving system compatibility and user experience.

- **Related PR**: [#608](https://github.com/higress-group/higress-console/pull/608) \
  **Contributor**: @Libres-coder \
  **Change Log**: This PR adds plugin display support to the AI route management page, allowing users to expand AI route rows to view enabled plugins and see the "Enabled" label on the configuration page. \
  **Feature Value**: Enhances AI route management by enabling users to manage AI-related plugin states more intuitively, improving user experience and operational convenience.

- **Related PR**: [#604](https://github.com/higress-group/higress-console/pull/604) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR introduces the feature of using the `higress.io/rewrite-target` annotation for path rewriting, supporting regular expressions, and enhancing the flexibility of path configuration. \
  **Feature Value**: By adding the ability to rewrite paths based on regular expressions, users can control and transform request paths more flexibly, enhancing the routing processing capability of the Higress gateway and meeting the needs of more scenarios.

- **Related PR**: [#603](https://github.com/higress-group/higress-console/pull/603) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR displays a fixed service port 80 for static service sources, implemented by hardcoding this value in the frontend component. \
  **Feature Value**: Users can more intuitively see and understand the default port number specific to static service sources, enhancing the clarity and user experience of the UI.

- **Related PR**: [#602](https://github.com/higress-group/higress-console/pull/602) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR adds a search function to the frontend page, allowing users to search when selecting upstream services for AI routes, improving the user experience. \
  **Feature Value**: This feature enables users to find the required upstream services more quickly and accurately, simplifying the configuration process and improving operational efficiency.

- **Related PR**: [#566](https://github.com/higress-group/higress-console/pull/566) \
  **Contributor**: @OuterCyrex \
  **Change Log**: This PR adds support for custom Qwen services, including enabling internet search and uploading file IDs. The main changes are in the backend SDK and frontend UI. \
  **Feature Value**: By supporting custom Qwen services, users can configure AI services more flexibly, such as using specific internet search features or specifying file IDs, thus meeting more personalized needs.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#620](https://github.com/higress-group/higress-console/pull/620) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed a spelling error in the sortWasmPluginMatchRules logic, ensuring correct sorting of match rules. \
  **Feature Value**: Fixing this spelling error improves the reliability and readability of the code, ensuring that Wasm plugin match rules work as expected and reducing potential runtime errors.

- **Related PR**: [#619](https://github.com/higress-group/higress-console/pull/619) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR removes version information from the data JSON during the conversion from AiRoute to ConfigMap, as this information is already saved in the ConfigMap metadata. \
  **Feature Value**: By removing redundant data, it improves the consistency and simplicity of the configuration, reducing potential data conflicts and inconsistencies.

- **Related PR**: [#618](https://github.com/higress-group/higress-console/pull/618) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR refactors the API authentication logic in SystemController to eliminate existing security vulnerabilities. It adds the AllowAnonymous annotation and adjusts the ApiStandardizationAspect class to ensure a more secure system. \
  **Feature Value**: This fix enhances the security of the system, preventing unauthorized access and potential security threats, improving user experience and trust.

- **Related PR**: [#617](https://github.com/higress-group/higress-console/pull/617) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed frontend console errors, including missing key attributes for list elements, image loading failures due to CSP policy restrictions, and incorrect type for the Consumer.name field. \
  **Feature Value**: Resolved multiple frontend issues encountered by users, improving the user experience and ensuring the stability and security of the application.

- **Related PR**: [#614](https://github.com/higress-group/higress-console/pull/614) \
  **Contributor**: @lc0138 \
  **Change Log**: This PR corrects the type of the type field in the ServiceSource class and adds dictionary value validation to ensure data accuracy. \
  **Feature Value**: By fixing the service source type error, it improves the data consistency and reliability of the system, reducing potential issues caused by type mismatches.

- **Related PR**: [#613](https://github.com/higress-group/higress-console/pull/613) \
  **Contributor**: @lc0138 \
  **Change Log**: This PR fixes CSP and other security risk issues by adding 15 lines of code to the frontend document.tsx file. \
  **Feature Value**: It resolves security risks related to Content Security Policy, enhancing the security level of the application and protecting users from potential security threats.

- **Related PR**: [#611](https://github.com/higress-group/higress-console/pull/611) \
  **Contributor**: @qshuai \
  **Change Log**: Corrected a description error in the LlmProvidersController.java file regarding the new route API, changing the title from 'Add a new route' to 'Ad'. \
  **Feature Value**: This fix addresses misleading information in the API documentation, ensuring developers can accurately understand the API's functionality, improving the development experience and reducing potential misuse.

- **Related PR**: [#609](https://github.com/higress-group/higress-console/pull/609) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed the type error for the Consumer.name field, changing its type from boolean to string. \
  **Feature Value**: This fix ensures the data consistency and accuracy of the Consumer.name field, avoiding data handling issues caused by type errors and improving system stability and user experience.

- **Related PR**: [#605](https://github.com/higress-group/higress-console/pull/605) \
  **Contributor**: @SaladDay \
  **Change Log**: Corrected the AI route name validation rules to support dot characters and unified case restrictions and interface prompts. Additionally, updated error messages in a multilingual environment. \
  **Feature Value**: Resolves inconsistencies encountered by users when setting AI route names, improving the user experience and system usability, ensuring information consistency and accuracy.

- **Related PR**: [#552](https://github.com/higress-group/higress-console/pull/552) \
  **Contributor**: @lcfang \
  **Change Log**: This PR adds the vport attribute to address compatibility issues caused by inconsistent service instance ports and provides an optional configuration for virtual ports during registration center setup. \
  **Feature Value**: By introducing the vport attribute, users can handle backend instance port changes more flexibly, avoiding service routing failures due to port changes, enhancing system stability and flexibility.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#610](https://github.com/higress-group/higress-console/pull/610) \
  **Contributor**: @heimanba \
  **Change Log**: This PR updates the documentation configuration for the frontend gray-scale plugin, including modifying the description of required fields, updating associated rules, and synchronizing the content in both Chinese and English README and spec.yaml files. \
  **Feature Value**: By adjusting the documentation configuration requirements and descriptions, it enhances the flexibility and compatibility of the configuration, making it easier for users to understand and use the frontend gray-scale plugin features.

---

## 📊 Release Statistics

- 🚀 New Features: 7 items
- 🐛 Bug Fixes: 10 items
- 📚 Documentation Updates: 1 item

**Total**: 18 changes

Thank you to all contributors for their hard work! 🎉

