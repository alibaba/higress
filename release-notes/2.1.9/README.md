# Higress


## 📋 Overview of This Release

This release includes **44** updates, covering enhancements, bug fixes, performance optimizations, and more.

### Distribution of Updates

- **New Features**: 23
- **Bug Fixes**: 14
- **Refactoring and Optimization**: 2
- **Documentation Updates**: 1
- **Testing Improvements**: 4

### ⭐ Key Highlights

This release contains **3** important updates, which are recommended for your attention:

- **feat(mcp-server): add server-level default authentication and MCP proxy server support** ([#3096](https://github.com/alibaba/higress/pull/3096)): This feature enhances the security and flexibility of Higress, allowing users to set unified security authentication rules for all tools and requests. It simplifies security policy management and improves the user experience.
- **feat: add higress api mcp server** ([#2923](https://github.com/alibaba/higress/pull/2923)): This feature enhances the management capabilities of Higress, allowing users to manage and configure Higress resources such as routes and service origins more flexibly via the MCP tool, improving the user experience and system operability.
- **feat: implement `hgctl agent` & `mcp add` subcommand** ([#3051](https://github.com/alibaba/higress/pull/3051)): The new subcommands greatly enhance the convenience and flexibility of Higress management, enabling users to interact with the Agent in natural language to manage Higress and simplify the process of adding MCP services, enhancing the user experience. This marks a step towards more advanced operational methods for Higress.

For detailed information, please refer to the significant feature descriptions below.

---

## 🌟 Detailed Description of Significant Features

### 1. feat(mcp-server): add server-level default authentication and MCP proxy server support

**Related PR**: [#3096](https://github.com/alibaba/higress/pull/3096) | **Contributor**: [@johnlanni](https://github.com/johnlanni)

**Usage Background**

With the widespread adoption of microservices architecture, the requirements for API gateway security and flexibility have increased. This PR addresses the need to set default authentication methods for all tools and requests in Higress, as well as the requirement for a middleware that can proxy MCP requests to the backend MCP server. This not only meets the user's need for simplified authentication configuration management but also provides a new MCP traffic handling model, especially suitable for enterprise application development teams looking to shift state management responsibilities from backend services to the edge (e.g., Higress).

**Feature Details**

This update mainly implements two new features: one is server-level default authentication (`defaultDownstreamSecurity` and `defaultUpstreamSecurity`), allowing administrators to set a unified authentication strategy for the entire system; the other is the addition of an MCP proxy server type (`mcp-proxy`), which can forward MCP requests sent by clients to Higress to a specified backend MCP server, supporting timeout control and end-to-end authentication. Technically, these new features are supported by updating the dependency library versions (`github.com/higress-group/wasm-go` and `github.com/higress-group/proxy-wasm-go-sdk`).

**Usage Instructions**

Before enabling this feature, ensure you have updated to the latest version of Higress. For default authentication settings, you can add the corresponding JSON configuration items in the global configuration file. For example, use the `defaultDownstreamSecurity` field to specify the authentication method between the client and the gateway. To utilize the MCP proxy function, specify its type as `mcp-proxy` when creating an MCP Server instance, and indicate the target MCP server address via the `mcpServerURL` attribute. Additionally, you can adjust the request timeout duration using the `timeout` parameter. It is recommended to refer to the official documentation for more detailed configuration guidelines.

**Feature Value**

This update greatly facilitates the implementation of unified and flexible identity verification strategies on the Higress platform, reducing the workload of repetitive configurations. It also opens up new avenues for optimizing MCP protocol handling based on edge computing models. For enterprises pursuing efficient operations and security, this means they can more easily achieve fine-grained access control, reduce potential risk exposure, and improve the overall system stability and response speed. More importantly, this design allows Higress to better adapt to various complex network environments, providing more diverse service discovery and governance solutions while maintaining high performance.

---

### 2. feat: add higress api mcp server

**Related PR**: [#2923](https://github.com/alibaba/higress/pull/2923) | **Contributor**: [@Tsukilc](https://github.com/Tsukilc)

**Usage Background**

As the Higress system continues to evolve, the demand for system management and debugging has increased. While the existing Higress Console Admin API provides basic management functions, it lacks the ability to manage AI routing, AI providers, and MCP servers. This update enhances Higress' management capabilities by integrating the higress-ops MCP Server, allowing users to manage and debug Higress configurations more flexibly. The target user groups include Higress system operators, developers, and users who need to manage Higress via the Agent.

**Feature Details**

This update mainly implements the following features: 
1. New AI route (AI Route) management functionality, supporting listing, getting, adding, updating, and deleting AI routes.
2. New AI provider (AI Provider) management functionality, supporting listing, getting, adding, updating, and deleting AI providers.
3. New MCP server (MCP Server) management functionality, supporting listing, getting, adding or updating, and deleting MCP servers and their consumers.
4. Refactored HigressClient to remove username and password parameters, instead using HTTP Basic Authentication for authorization.
5. Updated relevant documentation to ensure users can understand and use these new features. The core innovation lies in the introduction of new MCP tools to expand Higress' management capabilities, making them more flexible and powerful.

**Usage Instructions**

To enable and configure this feature, follow these steps:
1. Register a new MCP Server in the Higress configuration file, specifying its type as `higress-api`.
2. Configure the Higress Console URL and set the description information.
3. Interact with the Higress API MCP Server using the HGCTL command-line tool or other MCP clients. Typical usage scenarios include:
   1. Managing Higress configurations through HGCTL Agent in natural language.
   2. Managing AI routes, AI providers, and MCP servers via MCP clients.
   Note: 
   1. Ensure the Higress Console URL is correct.
   2. Use HTTP Basic Authentication for authorization.
   3. Avoid unnecessary type conversion operations in code to improve performance and code clarity.

**Feature Value**

This update brings the following specific benefits to users:
1. Improved manageability and debuggability of the Higress system, allowing users to manage and debug Higress configurations more conveniently via MCP tools.
2. Enhanced security and usability, using HTTP Basic Authentication for authorization, which improves system security.
3. Provides a unified API interface for other tools and systems in the ecosystem, promoting integration and development.
4. Through the newly added AI route and AI provider management functions, users can better leverage AI technology to optimize Higress routing strategies.
5. Through the MCP server management function, users can more flexibly manage and configure MCP servers, improving the flexibility and scalability of the system.

---

### 3. feat: implement `hgctl agent` & `mcp add` subcommand

**Related PR**: [#3051](https://github.com/alibaba/higress/pull/3051) | **Contributor**: [@erasernoob](https://github.com/erasernoob)

**Usage Background**

With the prevalence of microservices architecture, service mesh has become an important tool for managing complex service-to-service communication. Higress, as a high-performance service mesh control plane, needs to provide more flexible and user-friendly management tools. This PR adds two new features to the `hgctl` command-line tool: `hgctl agent` and `mcp add`. The former introduces an interactive agent similar to Claude Code, allowing users to manage Higress in natural language. The latter simplifies the process of adding remote MCP servers, enabling them to be directly published to the Higress MCP Server management tool. These improvements not only enhance the operability of Higress but also strengthen its competitiveness in the ecosystem. The target user groups are primarily Higress operators and developers.

**Feature Details**

This change implements two main features:
1. `hgctl agent`: This command launches an interactive window, internally calling the `claude-code` agent to guide users in setting up necessary environment variables. During the first use, it prompts users to install the required dependencies.
2. `mcp add`: This command allows users to directly add two types of MCP services—HTTP-based direct proxies and OpenAPI-based dynamically generated services. By parsing the provided parameters (such as URL, username, password, etc.), it automatically configures and registers the new MCP server with the Higress Console. Technically, it adds support for the `github.com/getkin/kin-openapi` library to handle OpenAPI specification files and completes the service registration process by sending API requests to Higress. Additionally, it updates some dependency versions to ensure compatibility with the latest Go toolchain.

**Usage Instructions**

Enabling and configuring these two new features is straightforward:
- For `hgctl agent`, simply run `hgctl agent` to start the interactive interface and complete the environment initialization as prompted.
- To add an MCP service using `mcp add`, enter the command in the following format:
  - Add an HTTP-type MCP service: `hgctl mcp add <name> -t http <url> --user <username> --password <password> --url <higress_console_url>`
  - Add an OpenAPI-type MCP service: `hgctl mcp add <name> -t openapi --spec <openapi_yaml_path> --user <username> --password <password> --url <higress_console_url>`
  Note: Ensure all dependencies are correctly installed and you have the necessary permissions to access the Higress Console.

**Feature Value**

This feature greatly enhances the usability and flexibility of Higress, making it easier for non-technical users to manage and extend their service mesh. In terms of system performance, by simplifying the configuration process, it reduces the likelihood of human errors, indirectly improving the system's stability and reliability. More importantly, in the highly competitive service mesh market, such innovative features help attract more users to choose Higress as their solution. Additionally, it lays the foundation for further integration of more advanced features in the future, such as more intelligent automation tools for operations.

---

## 📝 Complete Changelog

### 🚀 New Features (Features)

- **Related PR**: [#3126](https://github.com/alibaba/higress/pull/3126) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR updates the Envoy dependency, allowing the configuration of Redis client buffer behavior via WASM plugins. Specifically, it implements parsing of `buffer_flush_timeout` and `max_buffer_size_before_flush` from the parameter graph. \
  **Feature Value**: This feature enhances the flexibility of WASM plugins, allowing users to fine-tune Redis call-related parameters, thereby optimizing performance or meeting specific needs, improving the user experience.

- **Related PR**: [#3123](https://github.com/alibaba/higress/pull/3123) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR upgrades the proxy version to v2.2.0 and updates the Go toolchain and multiple dependency packages, adding golang-filter support for different architectures and fixing related dependencies. \
  **Feature Value**: By upgrading core components and fixing dependency issues, this update enhances system stability and compatibility. The added multi-architecture support expands the software's applicability, improving the user experience.

- **Related PR**: [#3108](https://github.com/alibaba/higress/pull/3108) \
  **Contributor**: @wydream \
  **Change Log**: This PR adds new API paths and capabilities related to video, including constant definitions, default function entries, and regex path handling. It also updates the OpenAI service provider to support the newly added video endpoints. \
  **Feature Value**: This update expands the system's multimedia processing capabilities, particularly for video content, providing developers with richer interface options to integrate complex video processing logic, thereby enhancing the user experience.

- **Related PR**: [#3071](https://github.com/alibaba/higress/pull/3071) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds the `inject_encoded_data_to_filter_chain_on_header` example, allowing the addition of a response body to the request when there is no response body. It is implemented via a Wasm plugin and provides detailed usage instructions. \
  **Feature Value**: This feature helps users more flexibly handle response data, especially in scenarios where dynamic response content needs to be added, significantly enhancing the functionality and flexibility of Higress.

- **Related PR**: [#3067](https://github.com/alibaba/higress/pull/3067) \
  **Contributor**: @wydream \
  **Change Log**: This PR adds support for vLLM as an AI provider in the ai-proxy plugin, implementing multiple OpenAI-compatible API interfaces, including Chat Completions, Text Completions, and Model Listing. \
  **Feature Value**: By introducing support for vLLM, this feature expands Higress' capability to handle AI requests, allowing users to more flexibly utilize different types of AI services and providing more options for developing AI-related applications with Higress.

- **Related PR**: [#3060](https://github.com/alibaba/higress/pull/3060) \
  **Contributor**: @erasernoob \
  **Change Log**: This PR enhances the `hgctl mcp` and `hgctl agent` commands, enabling them to automatically retrieve Higress Console credentials from installation configuration files and Kubernetes secrets, simplifying the user operation process. \
  **Feature Value**: By automatically retrieving credentials, this update enhances the user experience, reducing the need for manual account and password entry, making the management of Higress with `hgctl` more convenient and efficient.

- **Related PR**: [#3043](https://github.com/alibaba/higress/pull/3043) \
  **Contributor**: @2456868764 \
  **Change Log**: This PR fixes the incorrect default port for Milvus and adds Python example code to README.md. It adjusts some configurations to suit the scenario where the gateway only performs retrieval and not data entry. \
  **Feature Value**: This resolves the port issue encountered by users during use and provides additional Python code examples, enhancing the practicality and ease of use of the documentation, helping users better understand and use the project's features.

- **Related PR**: [#3040](https://github.com/alibaba/higress/pull/3040) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: This PR adds support for the Anthropic message API, allowing users to call Anthropic services directly via /v1/messages and supports converting OpenAI format request bodies to Claude-compatible formats. \
  **Feature Value**: By introducing support for the Anthropic message API, users can more flexibly configure and use different AI service providers. This not only enriches Higress' feature set but also enhances the operational convenience and interoperability of the platform.

- **Related PR**: [#3038](https://github.com/alibaba/higress/pull/3038) \
  **Contributor**: @Libres-coder \
  **Change Log**: This PR adds the `list-plugin-instances` tool to the MCP Server, allowing AI Agents to query plugin instances within a specific scope via the MCP protocol, and updates the Chinese and English documentation. \
  **Feature Value**: This feature allows users to more flexibly manage and monitor the use of plugins in Higress, enhancing the maintainability and transparency of the system, providing users with a new way to understand the status and configuration of their services.

- **Related PR**: [#3032](https://github.com/alibaba/higress/pull/3032) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR improves the Qwen AI provider configuration, including default enabling of compatibility mode and adding missing API endpoints, enhancing the user experience. \
  **Feature Value**: By default enabling compatibility mode and increasing API coverage, users can enjoy more comprehensive feature support and a better out-of-the-box experience.

- **Related PR**: [#3029](https://github.com/alibaba/higress/pull/3029) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: This PR adds support for v1/responses for the groq provider by modifying the relevant code in the groq.go file. \
  **Feature Value**: Adding support for v1/responses enhances the functionality of the groq provider, allowing users to more flexibly handle and respond to data, improving the system's flexibility and usability.

- **Related PR**: [#3024](https://github.com/alibaba/higress/pull/3024) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds detection for malicious URLs and model hallucinations and adjusts the configuration for specific consumers. It addresses issues present in multi-event responses and empty content submissions. \
  **Feature Value**: This enhances system security and stability by adding new detection mechanisms to effectively identify potential threats and prevent security vulnerabilities caused by erroneous responses. It also optimizes user-level configuration flexibility, improving the user experience.

- **Related PR**: [#3008](https://github.com/alibaba/higress/pull/3008) \
  **Contributor**: @hellocn9 \
  **Change Log**: This PR implements support for custom parameter names in MCP SSE stateful sessions. Users can specify their own parameter names by setting the `higress.io/mcp-sse-stateful-param-name` annotation, enhancing the system's flexibility and configurability. \
  **Feature Value**: This feature allows users to customize the parameter names for MCP SSE stateful sessions according to their needs, improving the system's flexibility and user experience, making Higress better suited for a variety of application scenarios.

- **Related PR**: [#3006](https://github.com/alibaba/higress/pull/3006) \
  **Contributor**: @SaladDay \
  **Change Log**: This PR adds Secret reference support for Redis configuration in the MCP Server, allowing the use of Kubernetes Secrets to store sensitive information such as passwords, thus enhancing security. It updates the code and documentation to smoothly transition from ConfigMap to Secret. \
  **Feature Value**: This feature enables users to handle sensitive data more securely, avoiding the potential risks associated with hardcoding passwords in ConfigMaps. This is an important improvement for users who prioritize security.

- **Related PR**: [#2992](https://github.com/alibaba/higress/pull/2992) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds the recording of unauthorized consumer names in the authentication and authorization process. By modifying the key_auth section in the wasm-cpp plugin, it ensures that even if a consumer is not authorized, their name will still be recorded. \
  **Feature Value**: This change improves the transparency and traceability of the system, making it easier for administrators to identify all users attempting to access the system, including those who are not authorized. This helps enhance security auditing and troubleshooting capabilities.

- **Related PR**: [#2978](https://github.com/alibaba/higress/pull/2978) \
  **Contributor**: @rinfx \
  **Change Log**: This PR implements the recording of consumer names in the key-auth plugin, regardless of whether the authentication is successful, as long as the consumer name can be determined. Specifically, it adds a new request header X-Mse-Consumer to store the consumer name in the main.go file. \
  **Feature Value**: This feature improves the tracking of consumer behavior, allowing the system to more accurately monitor and analyze each consumer's activities, even in cases where authentication fails. This enhances the system's security and auditability.

- **Related PR**: [#2968](https://github.com/alibaba/higress/pull/2968) \
  **Contributor**: @2456868764 \
  **Change Log**: This PR adds the core functionality of Vector Mapping, including the field mapping system and index configuration management. These features support flexible integration with different database schemas and defining various types of vector indices. \
  **Feature Value**: By providing field mapping and index configuration management capabilities, users can more flexibly connect Higress with different vector databases, enhancing the system's adaptability and scalability.

- **Related PR**: [#2943](https://github.com/alibaba/higress/pull/2943) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: This PR adds support for custom system prompts when generating release notes. By modifying the GitHub Actions workflow file, it reads the system prompts from the specified sections. \
  **Feature Value**: This feature allows users to include specific system prompt information when generating release notes, providing a more flexible and personalized document generation experience, helping to improve the quality and relevance of project documentation.

- **Related PR**: [#2942](https://github.com/alibaba/higress/pull/2942) \
  **Contributor**: @2456868764 \
  **Change Log**: This PR fixes the handling logic when the LLM provider is empty and optimizes the relevant documentation. It includes updating README.md to more clearly describe the MCP tools and their configuration methods, as well as adjusting the prompt template. \
  **Feature Value**: This enhancement improves the robustness and user experience of the system by allowing the LLM provider to be empty, avoiding potential errors, and providing more detailed and understandable documentation to help users better understand and use the MCP tools.

- **Related PR**: [#2916](https://github.com/alibaba/higress/pull/2916) \
  **Contributor**: @imp2002 \
  **Change Log**: This PR implements the Nginx migration to Higress MCP server and provides 7 automated tools to help convert Nginx configurations and Lua plugins. \
  **Feature Value**: This feature significantly simplifies the process of migrating from Nginx to Higress by providing automated migration tools, reducing the complexity of user operations and improving migration efficiency.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#3120](https://github.com/alibaba/higress/pull/3120) \
  **Contributor**: @lexburner \
  **Change Log**: This PR reduces unnecessary warning messages by adjusting the log levels in the ai-proxy plugin. Specifically, it modifies the log recording level in the qwen.go file, changing some warning-level logs to more appropriate levels. \
  **Feature Value**: This fix helps improve system log management by reducing redundant log information, allowing operations personnel to focus more on truly important log messages, thereby improving problem localization efficiency and user experience.

- **Related PR**: [#3119](https://github.com/alibaba/higress/pull/3119) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR resolves the deadlock issue caused by HTTP2 flow control by replacing the reqChan and deltaReqChan in the Connection with channels.Unbounded, ensuring that the Stream method is not blocked, the Put method does not block request reception, and normal reception of client ACK requests. \
  **Feature Value**: This fix improves system stability and response speed, avoiding bidirectional communication deadlocks caused by HTTP2 flow control, enhancing the user experience, especially in handling large response data volumes.

- **Related PR**: [#3118](https://github.com/alibaba/higress/pull/3118) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR prevents unconditional overwriting of existing TLS and LoadBalancer settings at the port level by adding nil checks and performing fine-grained merging of load balancing policies, ensuring that configurations derived from ingress annotations are not accidentally modified. \
  **Feature Value**: This fix addresses the issue where port-level policies in DestinationRule might overwrite configurations generated from ingress annotations, enhancing system stability and consistency, ensuring that user-defined network policies are correctly applied.

- **Related PR**: [#3095](https://github.com/alibaba/higress/pull/3095) \
  **Contributor**: @rinfx \
  **Change Log**: This PR fixes the issue of usage information being lost during the claude2openai conversion process and adds an index field to the Bedrock streaming tool response, ensuring data integrity and accuracy. \
  **Feature Value**: This ensures that critical usage information is not lost during the conversion from Claude to OpenAI and enhances the functionality of the Bedrock streaming tool response, allowing developers to better track and manage streaming output. This positively impacts system reliability and user experience.

- **Related PR**: [#3084](https://github.com/alibaba/higress/pull/3084) \
  **Contributor**: @rinfx \
  **Change Log**: This PR fixes the issue where the include_usage: true parameter was not correctly included in the conversion from Claude to OpenAI requests when stream processing is enabled, ensuring the completeness and consistency of API calls. \
  **Feature Value**: This fix ensures that users can accurately obtain all relevant information, including usage, when using stream processing, enhancing the data integrity of API responses, which is crucial for applications that rely on this information for subsequent processing or analysis.

- **Related PR**: [#3074](https://github.com/alibaba/higress/pull/3074) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR adds a check for Content-Encoding in the log-request-response plugin to avoid log chaos caused by compressed request/response bodies. \
  **Feature Value**: This fix allows users to obtain clearer and more readable log outputs, especially in scenarios where compression methods like Gzip are enabled, significantly improving debugging efficiency and user experience.

- **Related PR**: [#3069](https://github.com/alibaba/higress/pull/3069) \
  **Contributor**: @Libres-coder \
  **Change Log**: This PR fixes a bug in the CI testing framework by adding the `go mod tidy` command in the prebuild.sh script to update the go.mod file in the root directory, resolving e2e test failures due to unupdated go.mod. \
  **Feature Value**: This resolves the CI test failure issues encountered by all PRs triggering WASM plugin e2e tests, ensuring the stability and reliability of the continuous integration process, and improving the contributor experience.

- **Related PR**: [#3010](https://github.com/alibaba/higress/pull/3010) \
  **Contributor**: @rinfx \
  **Change Log**: This PR fixes the issue where Bedrock event streams sometimes fail to parse due to packet splitting problems and adjusts the maxtoken conversion logic to handle edge cases. \
  **Feature Value**: This resolves the issue where Bedrock event streams cannot be correctly parsed under certain conditions, ensuring the stability and reliability of the service, and enhancing the user experience.

- **Related PR**: [#2997](https://github.com/alibaba/higress/pull/2997) \
  **Contributor**: @hanxiantao \
  **Change Log**: This PR optimizes the rate-limiting logic for clusters, AI tokens, and WASM plugins, changing to an accumulative method to count request numbers and token usage, avoiding reset upon rate limit changes. \
  **Feature Value**: This optimization improves system stability and accuracy, reducing data reset issues caused by threshold adjustments and enhancing the user experience.

- **Related PR**: [#2988](https://github.com/alibaba/higress/pull/2988) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixes the issue in the jsonrpc-converter plugin where an incorrect JSON string format was used, instead using the raw JSON data. It modifies the relevant code in the main.go file to ensure accurate data processing. \
  **Feature Value**: This fix addresses the data processing error caused by an incorrect JSON string format, enhancing system stability and accuracy, reducing potential runtime errors, and providing a more reliable service environment for users.

- **Related PR**: [#2973](https://github.com/alibaba/higress/pull/2973) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR fixes the issue where Higress 2.1.8 did not support `match_rule_domain` being set to an empty string. It always uses wildcard matching to eliminate compatibility risks. \
  **Feature Value**: This improves system stability and compatibility, avoiding configuration errors due to unsupported empty strings, ensuring a seamless migration experience for users across different versions.

- **Related PR**: [#2952](https://github.com/alibaba/higress/pull/2952) \
  **Contributor**: @Erica177 \
  **Change Log**: This PR fixes the field name error in the ToolSecurity struct, changing `type` to `id` to ensure correct JSON serialization. \
  **Feature Value**: This change resolves the data parsing issue caused by the field name error, enhancing system stability and data accuracy, improving the user experience.

- **Related PR**: [#2948](https://github.com/alibaba/higress/pull/2948) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixes the Azure OpenAI response API handling and service URL type detection issues. It improves the logic for custom full path and enhances support for response API endpoints, and fixes edge cases in stream event parsing. \
  **Feature Value**: This improvement enhances compatibility with Azure OpenAI services, especially for applications using non-standard URL paths or relying on specific API endpoints. Users can now more reliably use Higress to proxy Azure OpenAI requests, reducing service disruptions due to configuration anomalies.

- **Related PR**: [#2941](https://github.com/alibaba/higress/pull/2941) \
  **Contributor**: @rinfx \
  **Change Log**: This PR fixes the compatibility issue between the ai-security-guard plugin and old configurations by adjusting the specific field mapping logic in the main.go file. \
  **Feature Value**: This resolves the incompatibility issue caused by configuration updates, ensuring a smooth transition to the new version without interrupting the user's existing security settings.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#3113](https://github.com/alibaba/higress/pull/3113) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR implements recursive hash calculation and caching for Protobuf messages, using the xxHash algorithm to improve performance. It specifically handles the google.protobuf.Any type and maps with deterministic ordering. \
  **Feature Value**: By reducing redundant serialization operations in LDS, this optimization speeds up filter chain matching and listener updates, enhancing the user experience.

- **Related PR**: [#2945](https://github.com/alibaba/higress/pull/2945) \
  **Contributor**: @rinfx \
  **Change Log**: This PR optimizes the global minimum request count pod selection logic in ai-load-balancer by updating the Lua script to improve the load balancing strategy. \
  **Feature Value**: This enhancement improves load balancing efficiency, distributing requests more evenly and reasonably, contributing to better overall system performance and response speed.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2965](https://github.com/alibaba/higress/pull/2965) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR updates the description of `azureServiceUrl` in the ai-proxy README file, adding more detailed configuration instructions to help developers better understand and use this parameter. \
  **Feature Value**: This improves the quality of the documentation, allowing users to more clearly understand how to set and use the `azureServiceUrl` parameter, thereby increasing the accuracy and efficiency of the configuration process.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#3110](https://github.com/alibaba/higress/pull/3110) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR adds the `CODECOV_TOKEN` environment variable configuration to the Codecov upload step in the GitHub Actions workflow, ensuring that Codecov can correctly authenticate. \
  **Feature Value**: By securely configuring the Codecov token in the CI/CD process, this feature enhances the accuracy and reliability of the code coverage report, helping developers better track and improve code quality.

- **Related PR**: [#3097](https://github.com/alibaba/higress/pull/3097) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR adds unit tests for the mcp-server module, including a large number of new test cases, ensuring code quality and stability. \
  **Feature Value**: This enhancement strengthens the reliability of the mcp-server module by introducing comprehensive test coverage, helping developers identify potential issues earlier, and improving the user experience and system stability.

- **Related PR**: [#2998](https://github.com/alibaba/higress/pull/2998) \
  **Contributor**: @Patrisam \
  **Change Log**: This PR implements end-to-end test cases for Cloudflare, adding test code in go-wasm-ai-proxy.go and go-wasm-ai-proxy.yaml to ensure the stability and reliability of Cloudflare-related features. \
  **Feature Value**: By adding end-to-end tests for Cloudflare features, this enhancement strengthens the system's ability to verify the functionality of Cloudflare integration, helping developers identify and resolve issues early, and enhancing the security and stability of the user experience.

- **Related PR**: [#2980](https://github.com/alibaba/higress/pull/2980) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR enhances the CI workflow for WASM plugin unit tests, adding coverage display and an 80% coverage threshold, failing CI if the threshold is not met. \
  **Feature Value**: This improves the code quality monitoring standard, ensuring that all WASM plugins reach at least 80% test coverage, helping to identify potential issues and enhance software reliability.

---

## 📊 Release Statistics

- 🚀 New Features: 23
- 🐛 Bug Fixes: 14
- ♻️ Refactoring and Optimization: 2
- 📚 Documentation Updates: 1
- 🧪 Testing Improvements: 4

**Total**: 44 changes (including 3 significant updates)

Thank you to all contributors for their hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **18** updates, covering enhancements, bug fixes, and performance optimizations.

### Update Content Distribution

- **New Features**: 7
- **Bug Fixes**: 10
- **Documentation Updates**: 1

---

## 📝 Complete Changelog

### 🚀 New Features (Features)

- **Related PR**: [#621](https://github.com/higress-group/higress-console/pull/621) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: This PR optimizes the interaction capabilities of the mcp server, including default rewriting of the header host, improving the interaction method to support transport selection and complete path replacement, and enhancing DSN character handling logic to support special characters @. \
  **Feature Value**: These updates enhance the system's flexibility and compatibility, making it easier for users to configure backend service addresses and improving support for special characters, thus enhancing the user experience.

- **Related PR**: [#612](https://github.com/higress-group/higress-console/pull/612) \
  **Contributor**: @zhwaaaaaa \
  **Change Log**: This PR resolves the issue where Grafana pages do not work properly due to chunked encoding sent by reverse proxy servers by adding ignore handling for hop-to-hop headers. \
  **Feature Value**: Ensures that Grafana pages can still display correctly when using a reverse proxy, improving system compatibility and user experience.

- **Related PR**: [#608](https://github.com/higress-group/higress-console/pull/608) \
  **Contributor**: @Libres-coder \
  **Change Log**: This PR adds plugin display support to the AI routing management page, allowing users to view enabled plugins and see the "Enabled" label on the configuration page. \
  **Feature Value**: Enhances the functionality of the AI routing page, making it easier for users to intuitively understand which plugins are enabled on each AI route, thereby improving the user experience and management efficiency.

- **Related PR**: [#604](https://github.com/higress-group/higress-console/pull/604) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR adds support for `higress.io/rewrite-target` annotations to enable path rewriting based on regular expressions, enhancing the flexibility of route configurations. \
  **Feature Value**: The new regular expression path rewriting feature allows users to more flexibly control request path transformation logic, improving the application's ability to handle complex URL patterns.

- **Related PR**: [#603](https://github.com/higress-group/higress-console/pull/603) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR adds a constant STATIC_SERVICE_PORT with a value of 80 in the frontend page and displays this fixed port in the service source component for static service sources. \
  **Feature Value**: By displaying the fixed service port number 80, users can more clearly understand the standard HTTP port used by static service sources, thereby improving the clarity and usability of the configuration.

- **Related PR**: [#602](https://github.com/higress-group/higress-console/pull/602) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR implements search functionality when selecting upstream services for AI routing, enhancing the user experience through frontend interface optimization. \
  **Feature Value**: The addition of search functionality allows users to quickly find and select the required upstream services, improving configuration efficiency and ease of use.

- **Related PR**: [#566](https://github.com/higress-group/higress-console/pull/566) \
  **Contributor**: @OuterCyrex \
  **Change Log**: Adds support for custom Qwen services, including enabling internet search and uploading file IDs. The corresponding support logic has been added to both the frontend and backend code. \
  **Feature Value**: This feature allows users to configure custom Qwen services, enhancing the system's flexibility and extensibility to meet more personalized needs.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#620](https://github.com/higress-group/higress-console/pull/620) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR fixes a spelling error in the sortWasmPluginMatchRules logic, ensuring the correctness and consistency of the code. \
  **Feature Value**: Corrects a text error in the sorting rule processing, improving the system's stability and reliability, and avoiding potential failures due to spelling issues.

- **Related PR**: [#619](https://github.com/higress-group/higress-console/pull/619) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR removes version information from the JSON data converted from AiRoute to ConfigMap, as this information is already stored in the ConfigMap metadata. \
  **Feature Value**: By eliminating redundant data, it improves the consistency and accuracy of configuration management, simplifying the complexity of user operations when handling ConfigMaps.

- **Related PR**: [#618](https://github.com/higress-group/higress-console/pull/618) \
  **Contributor**: @CH3CHO \
  **Change Log**: Refactors the API authentication logic in SystemController, introducing a new AllowAnonymous annotation to fix security vulnerabilities, ensuring that API calls to the system controller are more secure and reliable. \
  **Feature Value**: This update resolves security issues in SystemController, enhancing the system's security, preventing unauthorized access, and protecting user data security and privacy.

- **Related PR**: [#617](https://github.com/higress-group/higress-console/pull/617) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixes multiple errors in the frontend console, including missing unique key warnings for list items, image loading violations of content security policy, and incorrect field types for consumer names. \
  **Feature Value**: By addressing these frontend errors, it improves the user experience and application stability, reducing potential issues caused by errors.

- **Related PR**: [#614](https://github.com/higress-group/higress-console/pull/614) \
  **Contributor**: @lc0138 \
  **Change Log**: This PR corrects the type of the type field in the ServiceSource class and adds dictionary value validation, ensuring that the field only accepts predefined valid values. \
  **Feature Value**: Fixes the incorrect setting of the service source type field, improving system stability and data accuracy, and avoiding potential runtime errors due to type mismatches.

- **Related PR**: [#613](https://github.com/higress-group/higress-console/pull/613) \
  **Contributor**: @lc0138 \
  **Change Log**: This PR adds 15 lines of code to the frontend document.tsx file to fix CSP and other security risk issues. \
  **Feature Value**: This fix enhances application security, effectively preventing potential cross-site scripting attacks and other threats related to content security policies, improving the user experience and data protection.

- **Related PR**: [#611](https://github.com/higress-group/higress-console/pull/611) \
  **Contributor**: @qshuai \
  **Change Log**: Corrects a spelling error in the API documentation annotations of the LlmProvidersController class, ensuring the accuracy of the API documentation. \
  **Feature Value**: This PR fixes a small but important documentation issue, improving the quality of the API documentation and enabling developers to more accurately understand the API's functionality.

- **Related PR**: [#609](https://github.com/higress-group/higress-console/pull/609) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR fixes the incorrect type of the name field in the Consumer interface, changing it from boolean to string. \
  **Feature Value**: This fix ensures that the Consumer name is correctly stored and displayed, avoiding data processing errors due to type mismatches, and improving system stability and data accuracy.

- **Related PR**: [#605](https://github.com/higress-group/higress-console/pull/605) \
  **Contributor**: @SaladDay \
  **Change Log**: Corrects the validation rules for AI route names to support dots and unifies the interface prompts with the actual validation logic, while also adjusting the error message. \
  **Feature Value**: Resolves inconsistencies encountered by users when setting AI route names, improving system availability and the user experience.

- **Related PR**: [#552](https://github.com/higress-group/higress-console/pull/552) \
  **Contributor**: @lcfang \
  **Change Log**: Adds the vport attribute to resolve the issue of route configuration failure due to changes in the backend service port, ensuring compatibility by configuring the vport attribute during service registration. \
  **Feature Value**: Solves the problem of route configuration failure due to inconsistent service instance ports, improving system stability and the user experience.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#610](https://github.com/higress-group/higress-console/pull/610) \
  **Contributor**: @heimanba \
  **Change Log**: Updates the configuration field descriptions in the frontend canary plugin documentation, including changing the rewrite, backendVersion, and enabled fields to optional and correcting some text descriptions to ensure terminology consistency and accuracy. \
  **Feature Value**: By increasing configuration flexibility and compatibility and ensuring the accuracy and consistency of the documentation, this change makes it easier for users to understand and use the frontend canary plugin, reducing the learning curve.

---

## 📊 Release Statistics

- 🚀 New Features: 7
- 🐛 Bug Fixes: 10
- 📚 Documentation Updates: 1

**Total**: 18 changes

Thank you to all contributors for their hard work! 🎉

