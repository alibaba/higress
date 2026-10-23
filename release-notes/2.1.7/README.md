# Higress


## 📋 Overview of This Release

This release includes **42** updates, covering various aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 21 items
- **Bug Fixes**: 14 items
- **Refactoring and Optimization**: 4 items
- **Documentation Updates**: 2 items
- **Testing Improvements**: 1 item

### ⭐ Key Highlights

This release includes **3** significant updates, which are recommended for your attention:

- **feat: add MCP SSE stateful session load balancer support** ([#2818](https://github.com/alibaba/higress/pull/2818)): This feature enables MCP services based on the SSE protocol to better maintain persistent connections between the client and the server, enhancing user experience and application performance, especially in scenarios requiring long-lasting connections for data pushing.
- **feat: Support adding a proxy server in between when forwarding requests to upstream** ([#2710](https://github.com/alibaba/higress/pull/2710)): This feature allows users to use a proxy server when forwarding requests to upstream services, enhancing the system's flexibility and security, suitable for scenarios where communication through specific proxies is required.
- **feat(ai-proxy): add auto protocol compatibility for OpenAI and Claude APIs** ([#2810](https://github.com/alibaba/higress/pull/2810)): By automatically detecting and converting protocols, all AI providers can simultaneously support both the OpenAI protocol and the Claude protocol, allowing for seamless integration with Claude Code.

For more details, please refer to the Important Features section below.

---

## 🌟 Detailed Description of Important Features

Here are the detailed descriptions of important features and improvements in this release:

### 1. feat: add MCP SSE stateful session load balancer support

**Related PR**: [#2818](https://github.com/alibaba/higress/pull/2818) | **Contributor**: [@johnlanni](https://github.com/johnlanni)

**Usage Background**

As the demand for real-time communication grows, Server-Sent Events (SSE) have become a key technology for many applications. However, in distributed systems, ensuring that requests from the same client are always routed to the same backend service to maintain session state has been a challenge. Traditional load balancing strategies cannot meet this need. This feature addresses this issue by introducing MCP SSE stateful session load balancing support. By specifying the `mcp-sse` type in the `higress.io/load-balance` annotation, users can easily manage SSE connection state sessions. The target user group mainly consists of application developers and service providers who need to perform real-time data pushing in distributed environments.

**Feature Details**

This PR mainly implements the following features:
1. **Extend `load-balance` annotation**: In the `loadbalance.go` file, support for the `mcp-sse` value is added, and the `McpSseStateful` field is added to the `LoadBalanceConfig` struct.
2. **Simplified Configuration**: Users only need to set `mcp-sse` in the `higress.io/load-balance` annotation to enable this feature, with no additional configuration required.
3. **Backend Address Encoding**: When MCP SSE stateful session load balancing is enabled, the backend address will be Base64 encoded and embedded in the session ID of the SSE message. This ensures that the client can correctly identify and maintain the session. The core innovation lies in dynamically generating SSE session-related configurations through EnvoyFilter, thereby achieving stateful session management.

**Usage Instructions**

To use this feature, users need to follow these steps:
1. **Enable the Feature**: Add the `higress.io/load-balance: mcp-sse` annotation to the Ingress resource.
2. **Configuration Example**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: sse-ingress
  annotations:
    higress.io/load-balance: mcp-sse
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /mcp-servers/test/sse
        pathType: Prefix
        backend:
          service:
            name: sse-service
            port:
              number: 80
```
3. **Testing**: Access the SSE endpoint using the `curl` command and check if the returned messages contain the correct session ID.
**Notes**:
- Ensure that the backend service can handle Base64 encoded session IDs.
- Avoid frequent changes to the backend service deployment to prevent session consistency issues.

**Feature Value**

This feature brings the following specific benefits to users:
1. **Session Consistency**: Ensures that requests from the same client are always routed to the same backend service, maintaining session state consistency.
2. **Simplified Configuration**: Enables the feature with simple annotation configuration, reducing the complexity of user configuration.
3. **Enhanced User Experience**: For applications that rely on SSE, such as real-time notifications and stock market data, it provides a more stable and consistent service experience.
4. **Reduced Operations Costs**: Reduces errors and failures caused by inconsistent sessions, lowering the workload of the operations team.

---

### 2. feat: Support adding a proxy server in between when forwarding requests to upstream

**Related PR**: [#2710](https://github.com/alibaba/higress/pull/2710) | **Contributor**: [@CH3CHO](https://github.com/CH3CHO)

**Usage Background**

In modern microservice architectures, especially in complex network environments, directly forwarding requests from the client to the backend service may encounter various issues, such as network security and performance bottlenecks. Introducing an intermediate proxy server can effectively solve these problems, for example, by performing traffic control, load balancing, and SSL offloading through the proxy server. Additionally, in some cases, enterprises may need to use specific proxy servers to meet compliance and security requirements. The target user group for this feature mainly consists of enterprises and developers who need to optimize request forwarding paths in complex network environments.

**Feature Details**

This PR mainly implements the ability to configure one or more proxy servers in the McpBridge resource and allows specifying proxy servers for each registry. The specific implementation includes: 
1. Adding the `proxies` field in the `McpBridge` resource definition to configure the list of proxy servers, and adding the `proxyName` field in the `registries` item to associate the proxy server with the registry.
2. When creating or updating the `McpBridge` resource, the system automatically generates the corresponding EnvoyFilter resources, which define how to forward requests to the specified proxy server.
3. Additionally, EnvoyFilters are generated for each service bound to a proxy, ensuring they correctly point to the local listener on the corresponding proxy server. The entire technical implementation is based on Envoy's advanced routing capabilities, demonstrating the project's powerful functionality in handling complex network topologies.

**Usage Instructions**

To enable this feature, at least one proxy server must first be configured in the `McpBridge` resource. This can be done by adding new `ProxyConfig` objects to the `spec.proxies` array, each containing necessary information such as `name`, `serverAddress`, and `serverPort`. Next, for the registry entries that need to use a proxy server, simply reference the defined proxy name in their `proxyName` field. Once configured, the system will automatically handle all related EnvoyFilter generation work. It is worth noting that before actual deployment, the correctness of the configuration files should be carefully checked to avoid service unavailability due to misconfiguration.

**Feature Value**

The newly added proxy server support feature greatly enhances the system's network flexibility, allowing users to flexibly adjust request forwarding paths according to their needs. For example, by setting up different proxy servers, it is easy to achieve data transmission optimization across multiple regions; at the same time, with the additional security features provided by the proxy layer (such as SSL encryption), the overall system security is significantly improved. In addition, this feature also helps simplify operations management, especially in situations where frequent adjustments to the network architecture are needed. Through simple configuration changes, rapid responses to changes can be achieved without major modifications to the underlying infrastructure. In summary, this improvement not only expands the project's scope but also provides users with more powerful tools to tackle increasingly complex network challenges.

---

### 3. feat(ai-proxy): add auto protocol compatibility for OpenAI and Claude APIs

**Related PR**: [#2810](https://github.com/alibaba/higress/pull/2810) | **Contributor**: [@johnlanni](https://github.com/johnlanni)

**Usage Background**

In the AI proxy plugin, users may need to interact with multiple AI service providers (such as OpenAI and Anthropic Claude) simultaneously. These providers typically use different API protocols, leading to the need for manual configuration of protocol types when switching services, which increases complexity and the likelihood of errors. This feature solves this problem, allowing users to seamlessly use different providers' services without worrying about the differences in underlying protocols. The target user group consists of developers and enterprises who want to simplify the AI service integration process.

**Feature Details**

This PR implements the automatic protocol compatibility feature. The core technological innovation lies in automatically detecting the request path and intelligently converting the protocol based on the target provider's capabilities. Specifically, when the request path is `/v1/chat/completions`, it is recognized as the OpenAI protocol; when the request path is `/v1/messages`, it is recognized as the Claude protocol. If the target provider does not support the native Claude protocol, the plugin converts the request from Claude format to OpenAI format, and vice versa. In the `main.go` file, new logic for automatic protocol detection based on the request path is added, and path replacements are made as necessary. Additionally, a new `claude_to_openai.go` file is added to implement the specific conversion logic from Claude to OpenAI protocol.

**Usage Instructions**

Enabling this feature is very simple; users just need to send requests as usual, with no additional configuration required. For example, for OpenAI protocol requests, the URL is `http://your-domain/v1/chat/completions`, and for Claude protocol requests, the URL is `http://your-domain/v1/messages`. The plugin will automatically detect and handle protocol conversion. If the target provider does not support the Claude protocol, the plugin will convert it to OpenAI format. Example configuration is as follows:

```yaml
provider:
  type: claude  # Provider natively supporting the Claude protocol
  apiTokens:
    - 'YOUR_CLAUDE_API_TOKEN'
  version: '2023-06-01'
```

**Notes**: Ensure that the API token and version number are correctly configured so that the plugin can correctly identify and process the requests.

**Feature Value**

This feature significantly improves the usability and flexibility of the AI proxy plugin, reducing the user's configuration burden. Through automatic protocol detection and intelligent conversion, users can more easily switch between different AI service providers without worrying about protocol compatibility issues. This not only improves development efficiency but also enhances the stability and reliability of the system. Additionally, the feature supports streaming responses, further expanding its application scenarios, especially in cases requiring real-time interaction. In summary, this feature provides users with a more efficient and convenient way to integrate and manage multiple AI service providers.

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#2847](https://github.com/alibaba/higress/pull/2847) \
  **Contributor**: @Erica177 \
  **Change Log**: This PR adds a security mode for Nacos MCP, involving modifications to `mcp_model.go` and `watcher.go` files, including the addition and adjustment of configuration options. \
  **Feature Value**: By adding security mode support, the security of Nacos MCP services is enhanced, allowing users to manage their microservice configurations in a more secure environment.

- **Related PR**: [#2842](https://github.com/alibaba/higress/pull/2842) \
  **Contributor**: @hanxiantao \
  **Change Log**: Added detailed Chinese and English documentation for the hmac-auth-apisix plugin and added corresponding test cases to ensure the stability and reliability of the newly added features. \
  **Feature Value**: By adding documentation and tests, the availability and stability of the hmac-auth-apisix plugin are improved, helping users better understand and use the HMAC authentication mechanism, enhancing API security.

- **Related PR**: [#2823](https://github.com/alibaba/higress/pull/2823) \
  **Contributor**: @johnlanni \
  **Change Log**: Added OpenRouter as an AI service provider, supporting access to various AI models through a unified API. Core implementations include support for chat completions and text completions. \
  **Feature Value**: By introducing OpenRouter, users can more flexibly choose different AI models and interact with them, simplifying the complexity of cross-platform AI service usage and enhancing the user experience.

- **Related PR**: [#2815](https://github.com/alibaba/higress/pull/2815) \
  **Contributor**: @hanxiantao \
  **Change Log**: This PR adds the hmac-auth-apisix plugin, implementing API request authentication functionality. It verifies the integrity and authenticity of requests by generating signatures using the HMAC algorithm. \
  **Feature Value**: The newly added hmac-auth-apisix plugin enhances system security, ensuring that only authenticated clients can access protected resources, improving the user experience and system protection capabilities.

- **Related PR**: [#2808](https://github.com/alibaba/higress/pull/2808) \
  **Contributor**: @daixijun \
  **Change Log**: Added support for the Anthropic API and the OpenAI v1/models interface, expanding the compatibility and functional scope of DeepSeek. \
  **Feature Value**: The introduction of new support allows users to leverage more artificial intelligence service options, enhancing the system's flexibility and practicality.

- **Related PR**: [#2805](https://github.com/alibaba/higress/pull/2805) \
  **Contributor**: @johnlanni \
  **Change Log**: Added a JSON-RPC protocol conversion plugin, capable of extracting request and response information from MCP protocol to headers, facilitating further observation, rate limiting, and authentication processing. \
  **Feature Value**: This feature allows users to utilize JSON-RPC for higher-level policy control in A2A protocols, such as authentication and traffic management, thereby enhancing the system's flexibility and security.

- **Related PR**: [#2788](https://github.com/alibaba/higress/pull/2788) \
  **Contributor**: @zat366 \
  **Change Log**: This PR updates the dependency `github.com/higress-group/wasm-go` in mcp-server to support MCP plugin responses with images. This is achieved by updating the `go.mod` and `go.sum` files. \
  **Feature Value**: The new feature allows MCP plugins to handle and respond to image data, enhancing the system's multimedia processing capabilities and providing users with richer content display options.

- **Related PR**: [#2769](https://github.com/alibaba/higress/pull/2769) \
  **Contributor**: @github-actions[bot] \
  **Change Log**: This PR updates the CRD files in the `helm` folder, adding new attribute definitions for `proxies`. \
  **Feature Value**: By updating the CRD files to add new attributes, the Kubernetes resource definitions become more enriched and complete, enhancing the system's configuration flexibility and extensibility.

- **Related PR**: [#2761](https://github.com/alibaba/higress/pull/2761) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR introduces two new deduplication strategies: `SPLIT_AND_RETAIN_FIRST` and `SPLIT_AND_RETAIN_LAST`, used to retain the first and last elements of comma-separated header values, respectively. \
  **Feature Value**: The new strategies provide users with more granular control options, allowing them to choose to retain specific position data during deduplication operations, thus better meeting diverse needs.

- **Related PR**: [#2739](https://github.com/alibaba/higress/pull/2739) \
  **Contributor**: @WeixinX \
  **Change Log**: Added a new plugin configuration field `reroute`, allowing users to control whether to disable route reselection. This feature is implemented by modifying the main configuration file and adding relevant test cases. \
  **Feature Value**: This feature provides users with a way to finely control the routing behavior during request processing, enhancing the system's flexibility and configurability, and meeting the needs of specific scenarios.

- **Related PR**: [#2730](https://github.com/alibaba/higress/pull/2730) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds tool usage support for the Bedrock service by modifying the structures and logic in `bedrock.go` and other files, enabling the system to handle tool-related requests. \
  **Feature Value**: The new feature allows users to effectively utilize tool invocation capabilities in the Bedrock environment, enhancing the system's flexibility and functionality, and better meeting the needs of applications that require external tool integration.

- **Related PR**: [#2729](https://github.com/alibaba/higress/pull/2729) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds a length limit for each value in the AI statistics plugin, automatically truncating when the length exceeds the set limit. This helps reduce memory usage when processing large files such as base64-encoded images and videos. \
  **Feature Value**: By limiting and truncating overly long data values, this feature can effectively prevent memory overflow issues caused by logging large media files, thereby improving system stability and performance.

- **Related PR**: [#2713](https://github.com/alibaba/higress/pull/2713) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR adds Grok provider support for the AI proxy, including the addition of Grok Go files and updates to related documentation. \
  **Feature Value**: By integrating Grok as a new AI provider, users can now leverage Grok's AI capabilities to process requests, increasing the system's flexibility and functional diversity.

- **Related PR**: [#2712](https://github.com/alibaba/higress/pull/2712) \
  **Contributor**: @SCMRCORE \
  **Change Log**: Added support for the Gemini model thinking function, specifically adapting to the 2.5 Flash, 2.5 Pro, and 2.5 Flash-Lite models. \
  **Feature Value**: This enhancement improves the functionality of the AI proxy plugin, allowing users to utilize specific Gemini models for more complex thinking tasks, enhancing the user experience and application scope.

- **Related PR**: [#2704](https://github.com/alibaba/higress/pull/2704) \
  **Contributor**: @hanxiantao \
  **Change Log**: This PR implements the functionality of Rust WASM plugin support for Redis database configuration options and improves the `demo-wasm` to retrieve Redis configuration from the Wasm plugin configuration. \
  **Feature Value**: This feature allows developers to more flexibly configure and integrate Redis databases when using Rust WASM plugins, improving development efficiency and the configurability of applications.

- **Related PR**: [#2698](https://github.com/alibaba/higress/pull/2698) \
  **Contributor**: @erasernoob \
  **Change Log**: Implemented support for multimodal data in the Gemini model, adding the ability to handle images and text. This is achieved by introducing new dependencies and modifying existing code logic. \
  **Feature Value**: This enhancement strengthens the functionality of the AI proxy plugin, allowing it to support more complex multimodal data processing, providing users with a richer and more flexible AI service experience.

- **Related PR**: [#2696](https://github.com/alibaba/higress/pull/2696) \
  **Contributor**: @rinfx \
  **Change Log**: This PR introduces streaming response support when the content security plugin is enabled, adjusting the detection frequency via the `bufferLimit` parameter to improve the flexibility and efficiency of content detection. \
  **Feature Value**: The new streaming response feature allows users to more efficiently handle content security detection, reducing latency and improving the user experience, especially in scenarios requiring real-time feedback.

- **Related PR**: [#2671](https://github.com/alibaba/higress/pull/2671) \
  **Contributor**: @Aias00 \
  **Change Log**: Implemented path suffix and content type filtering functionality to address performance and resource management issues in the ai-statistics plugin. By introducing the SkipProcessing mechanism, it avoids indiscriminate processing of all requests and reduces unnecessary response body caching. \
  **Feature Value**: This enhancement improves the selective processing capability of the AI statistics plugin, enhancing system performance and optimizing resource usage efficiency. It is particularly beneficial for scenarios with a large number of complex API requests, significantly improving the user experience.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#2816](https://github.com/alibaba/higress/pull/2816) \
  **Contributor**: @Asnowww \
  **Change Log**: This PR corrects a spelling error in the `scanners-user-agents.data` file, changing 'scannr' to 'scanner'. \
  **Feature Value**: Correcting spelling errors in the documentation improves the accuracy and readability of the document, helping users better understand and use the related features.

- **Related PR**: [#2799](https://github.com/alibaba/higress/pull/2799) \
  **Contributor**: @erasernoob \
  **Change Log**: This PR fixes the wasm-go-build plugin build command to ensure that all files in the directory are included during compilation, solving the compilation failure issue caused by missing dependencies. \
  **Feature Value**: By fixing the build command, this PR prevents compilation errors due to missing files, enhancing the stability and reliability of the build process and providing a better development experience for developers.

- **Related PR**: [#2787](https://github.com/alibaba/higress/pull/2787) \
  **Contributor**: @co63oc \
  **Change Log**: This PR fixes a spelling error in the `RegisteTickFunc` function, ensuring the correctness of the timer task registration. By correcting the function name, it avoids potential functional failures. \
  **Feature Value**: This fix corrects the issue where timer tasks could not be registered correctly due to a spelling error, enhancing the system's stability and reliability and ensuring that applications dependent on timer task execution run as expected.

- **Related PR**: [#2786](https://github.com/alibaba/higress/pull/2786) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR removes the `accept-encoding` header when the mcp-session filter handles SSE transport requests, solving the issue of incorrect handling of compressed response body data. \
  **Feature Value**: This fix ensures that the MCP server can work correctly when using SSE transport upstream, avoiding data parsing errors due to compression and enhancing the system's stability and reliability.

- **Related PR**: [#2782](https://github.com/alibaba/higress/pull/2782) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR fixes the issue of the Azure URL configuration component being unexpectedly changed, ensuring the correctness and consistency of the URL components by defining a new enum type `azureServiceUrlType`. \
  **Feature Value**: This fix ensures that users can maintain their original Azure service URL configuration when using the AI proxy, avoiding service call failures or inconsistencies due to incorrect changes.

- **Related PR**: [#2757](https://github.com/alibaba/higress/pull/2757) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR fixes the issue with the mcp server building Envoy filter unit tests, ensuring the correctness and stability of the test cases. \
  **Feature Value**: By fixing the errors in the unit tests, this PR enhances the reliability and maintainability of the code, helping developers better perform subsequent development and debugging work.

- **Related PR**: [#2755](https://github.com/alibaba/higress/pull/2755) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR fixes the issue where adding duplicate IPs in the ip-restriction configuration would throw an error, by ignoring the error for existing IPs and displaying the specific error details from iptree. \
  **Feature Value**: Allowing duplicate entries in the IP restriction list improves configuration flexibility and user experience while ensuring that other types of errors are still handled effectively.

- **Related PR**: [#2754](https://github.com/alibaba/higress/pull/2754) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR corrects the stopping and buffering issues when decoding data in golang-filter, ensuring a more stable data processing flow. \
  **Feature Value**: This fix resolves errors in the data decoding process, enhancing the system's reliability and user experience, and preventing potential data loss or processing anomalies.

- **Related PR**: [#2743](https://github.com/alibaba/higress/pull/2743) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR fixes the error when setting `ip_source_type` to `origin-source`, ensuring that the IP restriction feature can be correctly configured based on the source type. \
  **Feature Value**: This fix resolves the issue of incorrect IP source type settings under specific conditions, enhancing the system's stability and security, and allowing users to more reliably use the IP restriction feature.

- **Related PR**: [#2723](https://github.com/alibaba/higress/pull/2723) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR corrects the functional anomaly in the C++ Wasm plugin due to using an incorrect attribute name in the `_match_service_` rule, restoring the rule by modifying it to the correct attribute name. \
  **Feature Value**: This fix resolves the service routing issue caused by an incorrect matching rule, enhancing the system's stability and accuracy, and ensuring that users can correctly access the desired services.

- **Related PR**: [#2706](https://github.com/alibaba/higress/pull/2706) \
  **Contributor**: @WeixinX \
  **Change Log**: This PR fixes the issue where the transformer performs an add operation when the key does not exist, and adds test cases for mapping operations, ensuring correct transformations from headers/query to body and from body to headers/query. \
  **Feature Value**: This fix enhances the system's stability and reliability, preventing erroneous data operations, and boosting user confidence in the data processing logic, thereby improving the user experience.

- **Related PR**: [#2663](https://github.com/alibaba/higress/pull/2663) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR fixes the error in the bedrock model name escaping logic, removes unnecessary URL encoding in the request body, and ensures that the response matches expectations. \
  **Feature Value**: By correcting the name escaping logic issue, this fix enhances the system's stability and compatibility, ensuring that users do not encounter issues due to mismatched escaping during use.

- **Related PR**: [#2653](https://github.com/alibaba/higress/pull/2653) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixes the issue where the AI route fallback function fails when using Bedrock. It ensures that the path can be correctly obtained even when headers are nil, avoiding null pointer exceptions. \
  **Feature Value**: This fix resolves the issue of request rejection due to signature verification failure under specific conditions, enhancing the system's stability and reliability, and ensuring that users can smoothly access the service.

- **Related PR**: [#2628](https://github.com/alibaba/higress/pull/2628) \
  **Contributor**: @co63oc \
  **Change Log**: This PR corrects spelling errors in multiple files, involving 5 files and 36 lines of code, ensuring the accuracy of the documentation and comments. \
  **Feature Value**: Correcting spelling errors enhances the professionalism of the codebase, allowing developers to more accurately understand the content when reading the documentation, thereby reducing errors caused by misunderstandings.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#2777](https://github.com/alibaba/higress/pull/2777) \
  **Contributor**: @StarryVae \
  **Change Log**: Updated the ai-prompt-decorator plugin to the new encapsulated API, improving the initialization configuration and the way the request header handling method is called. \
  **Feature Value**: This refactoring enhances the consistency and maintainability of the code, making it easier for developers to integrate and use the ai-prompt-decorator feature.

- **Related PR**: [#2773](https://github.com/alibaba/higress/pull/2773) \
  **Contributor**: @CH3CHO \
  **Change Log**: Refactored the path-to-API-name mapping logic in ai-proxy, introducing regular expressions to simplify the mapping process, and added test cases to verify the correctness of the functionality. \
  **Feature Value**: By optimizing the path mapping logic structure, this refactoring enhances the maintainability and extensibility of the code, making it easier to support more paths, indirectly improving the system's flexibility and user experience.

- **Related PR**: [#2740](https://github.com/alibaba/higress/pull/2740) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR downgrades some log levels from `warn` to `info` in the `ai-statistics` component to more accurately reflect the actual importance of these log messages. \
  **Feature Value**: By adjusting the log levels, this change makes the log records more in line with actual needs, helping to reduce false alarms when users view the logs and improve the user experience.

- **Related PR**: [#2711](https://github.com/alibaba/higress/pull/2711) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR deprecates the use of slashes as separators in the mcp server and tool, adopting a format that better conforms to function naming conventions. This includes updating some of the library's dependency versions and making adjustments to the relevant files. \
  **Feature Value**: By adhering to standard function naming conventions, this change enhances the consistency and readability of the code, helping to reduce future maintenance costs and minimizing potential errors due to non-compliant naming.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2770](https://github.com/alibaba/higress/pull/2770) \
  **Contributor**: @co63oc \
  **Change Log**: Corrected spelling errors in multiple files, including test files, README, and variable names and configuration item names in Go code. \
  **Feature Value**: This improves the accuracy and readability of the documentation, ensuring the consistency and user experience of the code. For users of the plugin, these changes help avoid confusion or configuration issues caused by spelling errors.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#2809](https://github.com/alibaba/higress/pull/2809) \
  **Contributor**: @Jing-ze \
  **Change Log**: Added unit tests for multiple Wasm extensions and introduced CI/CD workflows to automate these tests, ensuring code quality and stability. \
  **Feature Value**: This improves the reliability of Wasm plugins by adding comprehensive unit tests and automated CI/CD processes, helping developers quickly identify and fix issues, thereby enhancing the user experience.

---

## 📊 Release Statistics

- 🚀 New Features: 21 items
- 🐛 Bug Fixes: 14 items
- ♻️ Refactoring and Optimization: 4 items
- 📚 Documentation Updates: 2 items
- 🧪 Testing Improvements: 1 item

**Total**: 42 changes (including 3 significant updates)

Thank you to all the contributors for their hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **12** updates, covering multiple aspects such as feature enhancements, bug fixes, and performance optimizations.

### Distribution of Updates

- **New Features**: 5 items
- **Bug Fixes**: 5 items
- **Refactoring and Optimization**: 2 items

---

## 📝 Complete Changelog

### 🚀 New Features (Features)

- **Related PR**: [#585](https://github.com/higress-group/higress-console/pull/585) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR adds a new AI service provider and updates the list of available models, including updating translation files to support the newly added provider. \
  **Feature Value**: By introducing more AI service providers and updating the model list, users now have access to a wider range of service options, enhancing the system's flexibility and usability.

- **Related PR**: [#582](https://github.com/higress-group/higress-console/pull/582) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: Added support for the ai-load-balancer plugin, enabling visual configuration in higress-console and defining its priority within the system. \
  **Feature Value**: By providing white-screen configuration options, it greatly improves the efficiency and flexibility of managing AI load balancers, lowering the barrier to use.

- **Related PR**: [#579](https://github.com/higress-group/higress-console/pull/579) \
  **Contributor**: @JayLi52 \
  **Change Log**: This update adds support for PostgreSQL and ClickHouse databases to the MCP server management function, while optimizing the MySQL database connection string format and fixing some database connection-related issues. \
  **Feature Value**: The addition of new database support expands the application scope of MCP, allowing users to choose the most suitable database type flexibly, improving the system's compatibility and user experience.

- **Related PR**: [#572](https://github.com/higress-group/higress-console/pull/572) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR adds the functionality to manage proxy servers, including new classes and service controllers, allowing users to configure and manage proxy servers. \
  **Feature Value**: With the added support, users can more flexibly manage and configure proxy servers, increasing the system's flexibility and availability.

- **Related PR**: [#565](https://github.com/higress-group/higress-console/pull/565) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR improves MCP server management tasks 6 and 7, including updating the README.md documentation, modifying the system service implementation code, and optimizing the ConfigMap handling logic. \
  **Feature Value**: By improving the MCP server management features, it enhances the system's stability and maintainability, simplifying the management of Higress configurations and improving the user experience.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#584](https://github.com/higress-group/higress-console/pull/584) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed an error that occurred when authentication was enabled but no allowed consumers were present, including incorrectly clearing the list of allowed consumers and displaying incorrect authentication status. \
  **Feature Value**: Ensures that the authentication feature works correctly even without allowed consumers, and the user interface accurately reflects the current authentication status.

- **Related PR**: [#581](https://github.com/higress-group/higress-console/pull/581) \
  **Contributor**: @hongzhouzi \
  **Change Log**: Fixed an NPE exception that occurred during the update of the openapi mcp server and corrected the PostgreSQL enumeration values to ensure consistency with constants in Higress. \
  **Feature Value**: Improves the system's stability and reliability by resolving the NPE issue, and the consistency of enumeration values improves the accuracy of configuration management, reducing potential error sources.

- **Related PR**: [#577](https://github.com/higress-group/higress-console/pull/577) \
  **Contributor**: @CH3CHO \
  **Change Log**: Synchronized the domain name regex validation patterns between the front-end and back-end, ensuring that long top-level domains like `test.internal` are accepted, involving minor code changes and the addition of test cases. \
  **Feature Value**: Resolves the issue where some valid domain names could not pass due to inconsistent domain validation rules between the front-end and back-end, enhancing the system's compatibility and user experience.

- **Related PR**: [#574](https://github.com/higress-group/higress-console/pull/574) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed a logical error when filtering V1alpha1WasmPlugin based on the internal flag, ensuring that non-internal instances are not mistakenly returned. \
  **Feature Value**: Improves system accuracy, ensuring that users get the correct list of plugin instances, avoiding data inconsistency issues caused by logical errors.

- **Related PR**: [#570](https://github.com/higress-group/higress-console/pull/570) \
  **Contributor**: @CH3CHO \
  **Change Log**: Corrected a spelling mistake that caused a 'Cannot read properties of undefined' error when editing an OpenAI type LLM provider. \
  **Feature Value**: By fixing this issue, it prevents runtime errors when configuring OpenAI service providers, improving the system's stability and user experience.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#573](https://github.com/higress-group/higress-console/pull/573) \
  **Contributor**: @CH3CHO \
  **Change Log**: Refactored the authentication module for MCP server integration, allowing regular routes and MCP servers to share the same authentication logic. Major changes include adding, removing, and modifying code in multiple files. \
  **Feature Value**: By refactoring the authentication module, it unifies the authentication logic, improving code maintainability and reusability, reducing redundant code, and contributing to the overall stability and performance of the system.

- **Related PR**: [#571](https://github.com/higress-group/higress-console/pull/571) \
  **Contributor**: @JayLi52 \
  **Change Log**: Optimized the performance of the EditToolDrawer, McpServerCommand, and MCPDetail components by updating the way the Monaco editor is imported and configuring on-demand loading. \
  **Feature Value**: Improves the application's loading speed and response efficiency, reduces unnecessary resource consumption, and enhances the user experience.

---

## 📊 Release Statistics

- 🚀 New Features: 5 items
- 🐛 Bug Fixes: 5 items
- ♻️ Refactoring and Optimization: 2 items

**Total**: 12 changes

Thanks to all contributors for their hard work! 🎉

