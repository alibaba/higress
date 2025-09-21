# Higress


## 📋 Overview of This Release

This release includes **29** updates, covering various aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Content Distribution

- **New Features**: 13 items
- **Bug Fixes**: 7 items
- **Refactoring and Optimization**: 5 items
- **Documentation Updates**: 3 items
- **Testing Improvements**: 1 item

### ⭐ Key Focus

This release includes **3** important updates, which are recommended to be closely monitored:

- **feat(gzip): add gzip configuration support and update default settings** ([#2867](https://github.com/alibaba/higress/pull/2867)): By enabling the Gzip feature and adjusting its default parameters, users can more flexibly control resource compression, thereby optimizing website performance and user experience.
- **feat: add rag mcp server** ([#2930](https://github.com/alibaba/higress/pull/2930)): This PR introduces a new MCP server for users, which helps them manage and retrieve knowledge more effectively, enhancing the functionality and practicality of the system.
- **refactor(mcp): use ECDS for golang filter configuration to avoid connection drain** ([#2931](https://github.com/alibaba/higress/pull/2931)): This improvement enhances the MCP server configuration generation logic by using ECDS services to dynamically discover and load filter configurations, thereby improving system stability and user experience.

For more details, please refer to the key features section below.

---

## 🌟 Detailed Description of Key Features

Here is a detailed explanation of the important features and improvements in this release:

### 1. feat(gzip): add gzip configuration support and update default settings

**Related PR**: [#2867](https://github.com/alibaba/higress/pull/2867) | **Contributor**: [@Aias00](https://github.com/Aias00)

**Usage Background**

In modern web applications, data transmission efficiency is a critical factor. Uncompressed HTTP responses can lead to high bandwidth consumption and long loading times, affecting the user experience. This feature is primarily aimed at developers and operations personnel who need to optimize network performance. It addresses the issue of transmission delays caused by large response data, especially in mobile or low-bandwidth network environments. Additionally, for microservice architectures in cloud-native environments, Gzip compression helps reduce network overhead during cross-service calls, improving overall system response speed.

**Feature Details**

This PR adds Gzip compression configuration support to the Higress gateway and updates the default settings. The specific implementation includes: 1. Adding Gzip-related configuration items in the `helm/core/values.yaml` file, such as whether to enable, minimum content length, etc.; 2. Modifying the default values in the `pkg/ingress/kube/configmap/gzip.go` file to enable Gzip compression by default; 3. Updating relevant test cases to verify the correctness of the new feature. The key technical points lie in reasonably configuring Gzip parameters (e.g., compression level, window size) to balance the relationship between compression ratio and CPU usage, ensuring optimal compression effects without significantly increasing server load.

**Usage Instructions**

To enable and configure Gzip compression, users can customize its behavior by modifying the corresponding fields in the `values.yaml` file. For example, set `gzip.enable: true` to activate the compression feature and adjust `minContentLength` to specify the minimum byte length of content to be compressed. A typical use case is to enable Gzip when deploying new microservices or optimizing the network performance of existing services. Note that while Gzip can effectively reduce the amount of transmitted data, excessive compression may negatively impact server performance, so it is recommended to adjust the compression strategy based on actual circumstances.

**Feature Value**

This feature brings several important advantages to users: 1. Significantly reduces the size of HTTP response data, speeding up page loading and improving the user experience; 2. Reduces network traffic costs, especially suitable for mobile devices or limited bandwidth scenarios; 3. Improves overall system performance, particularly in handling a large number of small file requests. By providing flexible configuration options, administrators can adjust the compression intensity according to actual needs to achieve the best performance. This is crucial for building efficient and responsive web services.

---

### 2. feat: add rag mcp server

**Related PR**: [#2930](https://github.com/alibaba/higress/pull/2930) | **Contributor**: [@2456868764](https://github.com/2456868764)

**Usage Background**

With the development of large language models (LLMs), knowledge management and retrieval have become increasingly important. Traditional knowledge management systems often struggle with complex text data and lack integration with LLMs. The Higress RAG MCP server addresses this issue by providing a unified interface for managing and retrieving knowledge, integrating multiple LLM providers (such as OpenAI, DashScope, etc.). The target user group includes enterprises, developers, and researchers who need to efficiently manage and retrieve knowledge.

**Feature Details**

The PR implements the RAG MCP server, adding multiple new files and dependencies. The main features include knowledge management (creating, deleting, and listing knowledge blocks), search, and chat functionalities. Key technical points include using a recursive chunker to split text, supporting multiple embedding models (such as OpenAI and DashScope), and integrating with vector databases (such as Milvus). The code changes include adding multiple Go module dependencies, such as `github.com/dlclark/regexp2`, `github.com/milvus-io/milvus-sdk-go/v2`, and `github.com/pkoukk/tiktoken-go`, which are used for regular expression processing, vector database operations, and text encoding, respectively. Additionally, new HTTP clients and configuration structures were added to ensure the flexibility and configurability of the system.

**Usage Instructions**

To enable and configure the RAG MCP server, first, enable the MCP server in the `higress-config` ConfigMap and set the corresponding path and matching rules. Then, configure the basic parameters of the RAG system, such as the chunker type, block size, and overlap, as well as the relevant information for LLM and embedding models. Typical use cases include creating knowledge blocks from text, searching for related knowledge, and interacting with LLMs through chat. For example, you can create knowledge blocks by sending a POST request to `/mcp-servers/rag/create-chunks-from-text` or perform a search via `/mcp-servers/rag/search`. Be sure to check the required fields in the configuration file, such as API keys and model names, to avoid runtime errors.

**Feature Value**

The RAG MCP server provides powerful knowledge management and retrieval capabilities, significantly enhancing the intelligence of the system. By integrating multiple LLMs and embedding models, users can choose the most suitable tools based on their specific needs. Additionally, flexible configuration options allow the system to adapt to different network environments and data scales. This feature not only improves system performance and stability but also enhances the user experience, making knowledge management and retrieval more efficient and convenient. For the ecosystem, the RAG MCP server adds new core functionalities to the Higress platform, further solidifying its leading position in the field of intelligent applications.

---

### 3. refactor(mcp): use ECDS for golang filter configuration to avoid connection drain

**Related PR**: [#2931](https://github.com/alibaba/higress/pull/2931) | **Contributor**: [@johnlanni](https://github.com/johnlanni)

**Usage Background**

In the current implementation, the golang filter configuration is directly embedded into the HTTP_FILTER patch. This leads to connection drain issues when the configuration changes, primarily due to the inconsistent order of Go maps and the listener configuration changes triggered by HTTP_FILTER updates. To address this, the PR introduces the Extension Configuration Discovery Service (ECDS) to separate the HTTP_FILTER from the actual golang filter configuration. The target user group mainly includes network service operations personnel who require high availability and low latency.

**Feature Details**

This refactoring separates the configuration into two parts: the HTTP_FILTER contains only filters with config_discovery references, while the EXTENSION_CONFIG contains the actual golang filter configuration. Specifically, the `constructMcpSessionStruct` and `constructMcpServerStruct` methods were updated to return a format compatible with EXTENSION_CONFIG. Additionally, unit tests were updated to match the new configuration structure. This technical improvement uses ECDS to decouple filter configuration from HTTP_FILTER, thereby avoiding connection drain issues caused by configuration changes.

**Usage Instructions**

To enable this feature, users do not need to make any additional configurations, as this is an internal implementation change. A typical use case is when users need to frequently update golang filter configurations, and the system can automatically handle configuration updates without affecting existing connections. However, users should ensure that their environment supports ECDS and that the related Envoy proxy version is compatible. Best practices recommend regularly checking logs and monitoring data to verify the stability of connections during configuration updates.

**Feature Value**

This refactoring significantly improves the system's stability and reliability by eliminating connection drain issues during configuration changes, making the system more robust. For applications that rely on low latency and high availability, this is particularly important. Additionally, using ECDS enhances the maintainability and scalability of the code, laying a good foundation for future feature development. Overall, this change not only solves existing problems but also brings long-term benefits to the entire ecosystem.

---

## 📝 Full Changelog

### 🚀 New Features

- **Related PR**: [#2926](https://github.com/alibaba/higress/pull/2926) \
  **Contributor**: @rinfx \
  **Change Log**: Introduced support for multimodal, function calling, and thinking, expanding the capabilities of vertex-ai. Mainly modified the vertex-related code, enhancing the processing capabilities. \
  **Feature Value**: The added support for multimodal, function calling, and thinking greatly enriches the application scenarios of vertex-ai, allowing users to more flexibly utilize AI for complex task processing, improving the user experience.

- **Related PR**: [#2917](https://github.com/alibaba/higress/pull/2917) \
  **Contributor**: @Aias00 \
  **Change Log**: Added support for Fireworks AI, including integrating its services in the proxy plugin and related test cases. \
  **Feature Value**: By supporting Fireworks AI, users can leverage the capabilities of this AI service, enhancing the diversity and flexibility of the system to meet the needs of more scenarios.

- **Related PR**: [#2907](https://github.com/alibaba/higress/pull/2907) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR adds support for outputSchema by upgrading the wasm-go library and updating the related dependency versions. \
  **Feature Value**: The addition of outputSchema support enhances the plugin's ability to handle data output formats, allowing users to more flexibly define and process output content.

- **Related PR**: [#2897](https://github.com/alibaba/higress/pull/2897) \
  **Contributor**: @rinfx \
  **Change Log**: This PR introduces multimodal support and thinking functionality in the ai-proxy bedrock by updating the logic in the bedrock.go file. \
  **Feature Value**: The added multimodal and thinking functionalities expand the capabilities of ai-proxy bedrock, allowing users to utilize richer interaction modes and thinking processes, enhancing the system's flexibility and practicality.

- **Related PR**: [#2891](https://github.com/alibaba/higress/pull/2891) \
  **Contributor**: @rinfx \
  **Change Log**: This PR implements support for configuring specific detection services for different consumers in the AI content security plugin, achieved by adding configuration items such as requestCheckService and responseCheckService for flexible service customization. \
  **Feature Value**: This update allows users to specify independent content security check policies for different services or clients according to their needs, thereby increasing the system's flexibility and security, better meeting diverse business scenarios.

- **Related PR**: [#2883](https://github.com/alibaba/higress/pull/2883) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR adds support for Meituan Longcat, including implementing the corresponding request handling logic and test cases. \
  **Feature Value**: Provides users with more choices of AI service providers, enhancing the applicability and flexibility of the plugin to meet the needs of different scenarios.

- **Related PR**: [#2844](https://github.com/alibaba/higress/pull/2844) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR enhances the consistent hashing-based load balancing algorithm by adding useSourceIp support. It uses the source IP address instead of the information in the request header for more precise load distribution. \
  **Feature Value**: This feature improves the load balancing strategy, allowing services to make more reasonable and consistent routing decisions based on the client's real IP address, thereby enhancing system stability and performance.

- **Related PR**: [#2843](https://github.com/alibaba/higress/pull/2843) \
  **Contributor**: @erasernoob \
  **Change Log**: Added support for NVIDIA Triton Server, including configuration information and related code implementation. \
  **Feature Value**: This feature allows users to interact with the NVIDIA Triton Inference Server through the OpenAI protocol proxy, expanding the compatibility and flexibility of AI services.

- **Related PR**: [#2806](https://github.com/alibaba/higress/pull/2806) \
  **Contributor**: @C-zhaozhou \
  **Change Log**: This PR updates the ai-security-guard plugin to be compatible with the MultiModalGuard interface, thus supporting more types of content security checks. \
  **Feature Value**: Enhances the plugin's multifunctionality, allowing users to utilize multimodal APIs for more comprehensive content security checks, improving the system's flexibility and practicality.

- **Related PR**: [#2727](https://github.com/alibaba/higress/pull/2727) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR adds end-to-end testing for the OpenAI service, including test cases for non-streaming and streaming requests. \
  **Feature Value**: Adds validation means for OpenAI integration, ensuring the correctness and stability of both non-streaming and streaming API requests, improving system reliability and user experience.

- **Related PR**: [#2593](https://github.com/alibaba/higress/pull/2593) \
  **Contributor**: @Xscaperrr \
  **Change Log**: This PR adds the WorkloadSelector field in the EnvoyFilter to restrict the scope of the filter, ensuring it only affects the Higress Gateway and avoids conflicts with Istio components in the same namespace. \
  **Feature Value**: This improvement enhances the precise control of configurations, preventing unwanted impacts from spreading to other services or gateways, improving system stability and security.

### 🐛 Bug Fixes

- **Related PR**: [#2938](https://github.com/alibaba/higress/pull/2938) \
  **Contributor**: @wydream \
  **Change Log**: This PR fixes the issue where prompt attack detection does not work in MultiModalGuard mode due to the lack of AttackLevel field support, resolving the problem by updating the data structure and documentation. \
  **Feature Value**: After the fix, the system can correctly identify and prevent different levels of prompt attacks, enhancing the security and reliability of MultiModalGuard, ensuring users can use the related features more safely.

- **Related PR**: [#2904](https://github.com/alibaba/higress/pull/2904) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixes the issue where the original Authorization header might be overwritten during the initialization of the HTTP request context. It ensures that the Authorization header is unconditionally retrieved from the request headers and stored in the context, preventing the loss of authentication information. \
  **Feature Value**: This solution addresses the security issues and authentication failure risks caused by the accidental overwriting of the original authorization header, improving the system's stability and security, ensuring secure access to user data.

- **Related PR**: [#2899](https://github.com/alibaba/higress/pull/2899) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR optimizes MCP server host pattern matching, reducing runtime parsing overhead, and fixes the SSE message formatting issue by removing unused fields to improve memory usage efficiency. \
  **Feature Value**: By optimizing host pattern matching and resolving the newline character issue in SSE messages, this improvement enhances system performance and message accuracy, improving the user experience.

- **Related PR**: [#2892](https://github.com/alibaba/higress/pull/2892) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixes the JSON decoding error caused by the Claude tool returning the content field in array format and removes redundant code structures to improve code quality. \
  **Feature Value**: This change resolves the data parsing issue in specific scenarios, improving the system's stability and developer debugging efficiency, and simplifies maintenance by reducing redundant code.

- **Related PR**: [#2882](https://github.com/alibaba/higress/pull/2882) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixes the conversion error in the Claude protocol in streaming scenarios and improves tool call state tracking and connection blocking issues. \
  **Feature Value**: This improvement enhances the reliability of Claude to OpenAI bidirectional conversion, solving key issues in streaming response handling and improving the user experience.

- **Related PR**: [#2865](https://github.com/alibaba/higress/pull/2865) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: This PR fixes the issue where SSE connections would be blocked when SSE events are split into multiple chunks, by adding a caching mechanism to handle multi-chunk events. \
  **Feature Value**: This fix ensures that even if SSE responses are split into multiple parts, data transmission will not be interrupted, thereby improving system stability and user experience.

- **Related PR**: [#2859](https://github.com/alibaba/higress/pull/2859) \
  **Contributor**: @lcfang \
  **Change Log**: This PR adds a vport element in the mcpbridge to solidify the port configuration in the serviceEntry, ensuring that routing rules do not fail when the backend service instance port changes. It also adjusts the related data structures and logic. \
  **Feature Value**: This solution addresses the compatibility issues caused by inconsistent backend service instance ports, improving system stability and consistency, reducing service interruptions due to port changes, and enhancing the user experience.

### ♻️ Refactoring and Optimization

- **Related PR**: [#2933](https://github.com/alibaba/higress/pull/2933) \
  **Contributor**: @rinfx \
  **Change Log**: This PR removes duplicate think tag definitions in the bedrock and vertex modules, reducing redundant code and improving code maintainability. \
  **Feature Value**: By eliminating redundant code, this PR enhances the tidiness and maintainability of the project, indirectly providing a more stable and reliable system environment for users.

- **Related PR**: [#2927](https://github.com/alibaba/higress/pull/2927) \
  **Contributor**: @rinfx \
  **Change Log**: This PR modifies the condition judgment in the `getAPIName` function, changing the fixed-length check to a more flexible approach that requires at least three parts. \
  **Feature Value**: By relaxing the restrictions on API name extraction, this improvement enhances the system's compatibility and robustness with different formats of API strings, reducing errors due to incomplete matches.

- **Related PR**: [#2922](https://github.com/alibaba/higress/pull/2922) \
  **Contributor**: @daixijun \
  **Change Log**: This PR upgrades the package name referenced by the Higress SDK from github.com/alibaba/higress to version 2, ensuring the project can reference the latest SDK version. \
  **Feature Value**: By updating the package name to the latest version, this PR resolves dependency inclusion issues caused by version mismatches, enhancing the project's compatibility and maintainability.

- **Related PR**: [#2890](https://github.com/alibaba/higress/pull/2890) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR introduces the HostMatcher struct, using a dedicated matching type instead of regex-based matching, and implements port stripping logic to correctly handle host headers with port numbers. \
  **Feature Value**: By improving the host matching logic, this enhancement enhances code maintainability and execution efficiency, making the system more accurate and efficient in handling complex host headers, indirectly enhancing system stability and response speed.

### 📚 Documentation Updates

- **Related PR**: [#2912](https://github.com/alibaba/higress/pull/2912) \
  **Contributor**: @hanxiantao \
  **Change Log**: This PR optimizes the Chinese and English documentation for the hmac-auth-apisix plugin, adding configuration instructions for route name and domain matching, and restructuring the document to improve readability. \
  **Feature Value**: By providing detailed instructions on the usage and configuration rules of the hmac-auth-apisix plugin, this update helps developers better understand and use the authentication mechanism, thereby enhancing the security and user experience of the API gateway.

- **Related PR**: [#2880](https://github.com/alibaba/higress/pull/2880) \
  **Contributor**: @a6d9a6m \
  **Change Log**: This PR fixes grammatical errors in the README.md and its Japanese and Chinese versions, ensuring the accuracy and consistency of the documentation. \
  **Feature Value**: By correcting errors in the documentation, this update improves users' understanding of the project, enhancing the user experience and maintaining the professional image of the project.

- **Related PR**: [#2873](https://github.com/alibaba/higress/pull/2873) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR adds instructions for obtaining logs and configurations in the issue template, helping users provide more comprehensive information for troubleshooting. \
  **Feature Value**: By improving the issue template, this update guides users to provide more complete log and configuration information, helping to improve the efficiency of issue localization and enabling maintainers to resolve issues more quickly.

### 🧪 Testing Improvements

- **Related PR**: [#2928](https://github.com/alibaba/higress/pull/2928) \
  **Contributor**: @rinfx \
  **Change Log**: This PR updates the unit test cases for the ai-security-guard plugin, adding new test scenarios and optimizing existing test logic. \
  **Feature Value**: By enhancing and updating test cases, this improvement increases the stability and reliability of the ai-security-guard feature, ensuring that new features or fixes do not affect existing security functions.

---

## 📊 Release Statistics

- 🚀 New Features: 13 items
- 🐛 Bug Fixes: 7 items
- ♻️ Refactoring and Optimization: 5 items
- 📚 Documentation Updates: 3 items
- 🧪 Testing Improvements: 1 item

**Total**: 29 changes (including 3 key updates)

Thank you to all contributors for their hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **4** updates, covering multiple aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Content Distribution

- **New Features**: 1 item
- **Bug Fixes**: 2 items
- **Documentation Updates**: 1 item

### ⭐ Key Highlights

This release contains **1** significant update, which is recommended for your attention:

- **feat: Support using a known service in OpenAI LLM provider** ([#589](https://github.com/higress-group/higress-console/pull/589)): The new support allows users to more conveniently integrate and use OpenAI's LLM services, enhancing the system's flexibility and usability, providing users with more options.

For more details, please see the Important Features section below.

---

## 🌟 Detailed Important Features

Below are the detailed explanations of the important features and improvements in this release:

### 1. feat: Support using a known service in OpenAI LLM provider

**Related PR**: [#589](https://github.com/higress-group/higress-console/pull/589) | **Contributor**: [@CH3CHO](https://github.com/CH3CHO)

**Usage Background**

In the current system, when configuring the OpenAI LLM provider, users can only use the default service address. However, in practical applications, users may need to use their own proxy server or other known services to interact with OpenAI. This demand may be due to considerations such as performance optimization, enhanced security, or specific business needs. For example, enterprises may wish to route all external API calls through an internal proxy server to ensure data security and compliance. Additionally, some users may have already deployed mirrors or proxies of OpenAI services in their infrastructure, thus requiring a flexible way to specify these services. The target user group mainly includes developers, system administrators, and enterprises that need highly customized OpenAI services.

**Feature Details**

This update primarily implements the following features:
1. **Support for custom service configuration**: Added `buildServiceSource` and `buildUpstreamService` methods, allowing users to specify custom OpenAI services through configuration. If the user provides a custom service configuration, it will directly use that configuration without creating a new service source.
2. **Enhanced Wasm plugin management**: Added a method to delete plugin instances in the `WasmPluginInstanceService` interface, supporting the passing of an `internal` parameter, making the management of internal resources more flexible.
3. **Internationalization resource check**: Added relevant keys in the frontend internationalization resource check to support the correct display of newly added features in different language environments. The core technological innovation lies in providing a flexible and unified way to handle custom service configurations and improving system maintainability by adding support for internal resource management.

**Usage Instructions**

The specific steps to enable and configure this feature are as follows:
1. Find the relevant settings section for the OpenAI LLM provider in the configuration file.
2. Add or modify the `openaiCustomServiceHost` and `openaiCustomServicePath` fields to specify the hostname and path of the custom service, respectively. For example:
   ```json
   {
     "provider": "OpenAI",
     "openaiCustomServiceHost": "api.openai.internal",
     "openaiCustomServicePath": "/v1"
   }
   ```
3. If further control over internal resource management is needed, pass the `internal` parameter when deleting Wasm plugin instances, such as `wasmPluginInstanceService.delete(WasmPluginInstanceScope.ROUTE, routeName, BuiltInPluginName.MODEL_MAPPER, true);`.
4. The system will automatically connect to the specified service address based on the provided custom service configuration, thus achieving a more flexible integration approach. Precautions include ensuring the validity and reachability of the custom service address and properly configuring related security policies.

**Feature Value**

This new feature brings significant benefits to users:
1. **Increased Flexibility**: Users can choose different OpenAI service configuration methods according to their needs, whether using external services or internal proxies.
2. **Performance Optimization**: By using internal proxies or local mirror services, network latency can be reduced, and response speed improved.
3. **Enhanced Security**: For enterprises with strict security requirements, accessing OpenAI services through the internal network helps ensure the security of data transmission.
4. **Better User Experience**: The frontend interface provides corresponding multilingual support, ensuring that users can smoothly configure and use this feature in different language environments. Overall, this feature not only meets the diverse needs of users but also further enhances the reliability and user satisfaction of the entire system.

---

## 📝 Full Changelog

### 🐛 Bug Fixes

- **Related PR**: [#591](https://github.com/higress-group/higress-console/pull/591) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed the condition check logic for route rewrite configuration, ensuring that both host and newPath.path must be provided with valid values when enabled, resolving errors caused by missing required fields. \
  **Feature Value**: By fixing the validation logic issue in the route rewrite configuration, the stability and user experience of the system were improved, avoiding functional anomalies due to incomplete configuration.

- **Related PR**: [#590](https://github.com/higress-group/higress-console/pull/590) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed an error in the Route.customLabels processing logic, excluding built-in labels to ensure they can be correctly removed during the update process. \
  **Feature Value**: This fix resolved the issue of confusion between custom labels and built-in labels, enhancing the accuracy and user experience of the system, especially when updating route configurations.

### 📚 Documentation

- **Related PR**: [#595](https://github.com/higress-group/higress-console/pull/595) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR removed irrelevant descriptions from README.md and added code formatting guidelines, totaling 72 lines of changes. \
  **Feature Value**: By cleaning up unnecessary information and adding formatting guidelines, the quality and readability of the documentation were improved, helping developers better understand and follow the project’s contribution rules.

---

## 📊 Release Statistics

- 🚀 New Features: 1 item
- 🐛 Bug Fixes: 2 items
- 📚 Documentation Updates: 1 item

**Total**: 4 changes (including 1 significant update)

Thank you to all contributors for their hard work! 🎉

