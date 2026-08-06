# Higress


## 📋 Overview of This Release

This release includes **30** updates, covering various aspects such as feature enhancements, bug fixes, performance optimizations, and more.

### Update Distribution

- **New Features**: 13
- **Bug Fixes**: 7
- **Refactoring and Optimization**: 5
- **Documentation Updates**: 4
- **Testing Improvements**: 1

### ⭐ Key Highlights

This release includes **2** major updates, which are highly recommended for your attention:

- **feat: add rag mcp server** ([#2930](https://github.com/alibaba/higress/pull/2930)): By introducing the RAG MCP server, this update provides a new way for users to manage and retrieve knowledge, enhancing the functionality and practicality of the system.
- **refactor(mcp): use ECDS for golang filter configuration to avoid connection drain** ([#2931](https://github.com/alibaba/higress/pull/2931)): Using ECDS for filter configuration avoids instability caused by directly embedding golang filter configurations, improving the system's stability and maintainability, and reducing unnecessary service interruptions for users.

For more details, please refer to the important features section below.

---

## 🌟 Detailed Description of Important Features

Below is a detailed description of the key features and improvements in this release:

### 1. feat: add rag mcp server

**Related PR**: [#2930](https://github.com/alibaba/higress/pull/2930) | **Contributor**: [@2456868764](https://github.com/2456868764)

**Use Case**

In modern applications, knowledge management and retrieval have become increasingly important. Many systems require fast and accurate extraction and retrieval of information from large volumes of text data. RAG (Retrieval-Augmented Generation) technology combines retrieval and generation models to effectively enhance the efficiency and accuracy of knowledge management. This PR introduces a Model Context Protocol (MCP) server specifically for knowledge management and retrieval, meeting the needs of users for efficient information processing. The target user group includes enterprises and developers who need to handle large amounts of text data, especially in the fields of natural language processing (NLP) and machine learning.

**Feature Details**

This PR implements the RAG MCP server, adding multiple functional modules, including knowledge management, block management, search, and chat functions. The core features include:
1. **Knowledge Management**: Supports creating knowledge blocks from text.
2. **Block Management**: Provides functionalities for listing and deleting knowledge blocks.
3. **Search**: Supports keyword-based search.
4. **Chat Function**: Allows users to send chat messages and receive responses.
Technically, the server uses several external libraries, such as `github.com/dlclark/regexp2`, `github.com/milvus-io/milvus-sdk-go/v2`, and `github.com/pkoukk/tiktoken-go`, which provide regular expression handling, vector database management, and text encoding functionalities. Key code changes include adding an HTTP client, configuration files, and multiple processing functions to ensure the flexibility and configurability of the system.

**Usage Instructions**

To enable and configure the RAG MCP server, follow these steps:
1. Enable the MCP server in the `higress-config` configuration file and set the corresponding path and configuration items.
2. Configure the basic parameters of the RAG system, such as splitter type, chunk size, and overlap.
3. Configure the LLM (Large Language Model) provider and its API key, model name, etc.
4. Configure the embedding model provider and its API key, model name, etc.
5. Configure the vector database provider and its connection information.
Example configuration:
```yaml
rag:
  splitter:
    type: "recursive"
    chunk_size: 500
    chunk_overlap: 50
  top_k: 5
  threshold: 0.5
llm:
  provider: "openai"
  api_key: "your-llm-api-key"
  model: "gpt-3.5-turbo"
embedding:
  provider: "openai"
  api_key: "your-embedding-api-key"
  model: "text-embedding-ada-002"
vectordb:
  provider: "milvus"
  host: "localhost"
  port: 19530
  collection: "test_collection"
```
Notes:
- Ensure all configuration items are correct, especially API keys and model names.
- In production environments, it is recommended to adjust parameters such as timeout appropriately to adapt to different network conditions.

**Feature Value**

The RAG MCP server provides a complete solution for knowledge management and retrieval, enhancing the intelligence and automation of the system. Specific benefits include:
1. **Improved Efficiency**: Through integrated knowledge management and retrieval functions, users can quickly process and retrieve large volumes of text data, saving time and resources.
2. **Enhanced Accuracy**: Combining RAG technology, the system can more accurately extract and retrieve information, reducing error rates.
3. **Flexible Configuration**: Provides rich configuration options, allowing users to flexibly adjust according to actual needs, meeting the requirements of different scenarios.
4. **High Scalability**: Supports multiple providers and models, making it easy for users to choose suitable components and technology stacks based on business needs.
5. **Stability Improvement**: Through detailed configuration validation and error handling mechanisms, the stability and robustness of the system are ensured.

---

### 2. refactor(mcp): use ECDS for golang filter configuration to avoid connection drain

**Related PR**: [#2931](https://github.com/alibaba/higress/pull/2931) | **Contributor**: [@johnlanni](https://github.com/johnlanni)

**Use Case**

In the current implementation, Golang filter configurations are directly embedded in the HTTP_FILTER patch, which can lead to connection drain when configurations change. The main reason is the inconsistent sorting of Go maps in the `map[string]any` field, and the listener configuration changes triggered by HTTP_FILTER updates. This issue affects the stability and user experience of the system. The target user group is developers and operations personnel using Higress for service mesh management.

**Feature Details**

This PR splits the configuration into two parts: HTTP_FILTER only contains filter references with `config_discovery`, while EXTENSION_CONFIG contains the actual Golang filter configuration. This way, configuration changes do not directly cause connection drain. The specific implementation includes updating the `constructMcpSessionStruct` and `constructMcpServerStruct` methods to return formats compatible with EXTENSION_CONFIG and updating unit tests to match the new configuration structure. The core innovation lies in using the ECDS mechanism to separate configurations, making configuration changes smoother.

**Usage Instructions**

Enabling and configuring this feature does not require any additional operations as it is automatically handled in the background. A typical use case is when configuring Golang filters in Higress; the system will automatically split them into HTTP_FILTER and EXTENSION_CONFIG. Users only need to configure Golang filters as usual. Note that when upgrading to the new version, ensure all related configuration files are updated and thoroughly tested in the production environment to ensure that configuration changes do not introduce other issues.

**Feature Value**

By separating configurations and using ECDS, this feature eliminates the connection drain problem during configuration changes, significantly improving the system's stability and user experience. Additionally, this design makes configurations easier to manage and maintain, reducing potential issues caused by configuration changes. For large-scale service mesh deployments, this improvement is particularly important as it reduces service interruptions caused by configuration changes, thereby enhancing the overall reliability and availability of the system.

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#2926](https://github.com/alibaba/higress/pull/2926) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds support for multimodal, function calls, and thinking in vertex-ai, involving the introduction of a regular expression library and improvements to the processing logic. \
  **Feature Value**: By adding new features, vertex-ai can better support application needs in complex scenarios, such as multimodal data processing and more flexible function call methods, enhancing the system's flexibility and practicality.

- **Related PR**: [#2917](https://github.com/alibaba/higress/pull/2917) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR adds support for Fireworks AI, expanding the functionality of the AI agent plugin, including the addition of necessary configuration files and test code. \
  **Feature Value**: Adding support for Fireworks AI allows users to leverage the AI features provided by the platform, broadening the range of AI services that applications can integrate with, and enhancing the user experience.

- **Related PR**: [#2907](https://github.com/alibaba/higress/pull/2907) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR upgrades wasm-go to support outputSchema, involving dependency updates for jsonrpc-converter and oidc plugins. \
  **Feature Value**: By supporting outputSchema, the functionality and flexibility of the wasm-go plugin are enhanced, making it easier for users to handle and define output data structures.

- **Related PR**: [#2897](https://github.com/alibaba/higress/pull/2897) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds multimodal support and thinking functionality to the ai-proxy bedrock, achieved by extending the relevant code in bedrock.go. \
  **Feature Value**: The added multimodal and thinking support enriches the ai-proxy's feature set, enabling users to utilize more advanced AI technologies for complex scenarios, enhancing the system's flexibility and practicality.

- **Related PR**: [#2891](https://github.com/alibaba/higress/pull/2891) \
  **Contributor**: @rinfx \
  **Change Log**: This PR adds the ability to configure specific detection services for different consumers in the AI content security plugin, allowing users to customize request and response check rules according to their needs. \
  **Feature Value**: By supporting independent detection services for different consumers, this feature enhances the system's flexibility and security, enabling users to control the content review process more precisely, thus meeting diverse security policy requirements.

- **Related PR**: [#2883](https://github.com/alibaba/higress/pull/2883) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR adds support for Meituan Longcat, including integration with the Longcat platform and related unit tests. \
  **Feature Value**: Adding support for Meituan Longcat expands the plugin's functionality, allowing users to leverage more AI service providers' technologies, enhancing the flexibility and diversity of the application.

- **Related PR**: [#2867](https://github.com/alibaba/higress/pull/2867) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR adds support for Gzip configuration and updates the default settings. By adding gzip options in the Helm configuration file, users can customize compression parameters to optimize response performance. \
  **Feature Value**: Adding support for Gzip configuration allows users to adjust the compression level of HTTP responses according to their needs, helping to reduce the amount of transmitted data, speed up page loading, and improve the user experience.

- **Related PR**: [#2844](https://github.com/alibaba/higress/pull/2844) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR enhances the consistent hashing algorithm for load balancing by supporting useSourceIp, modifying the relevant Go code files, and adding an example configuration file. \
  **Feature Value**: The newly added useSourceIp option allows users to perform consistent hash load balancing based on source IP addresses, which helps to improve the stability and reliability of services under specific network conditions.

- **Related PR**: [#2843](https://github.com/alibaba/higress/pull/2843) \
  **Contributor**: @erasernoob \
  **Change Log**: This PR adds NVIDIA Triton server support to the AI agent plugin, including related configuration instructions and code implementation. \
  **Feature Value**: Adding support for the Triton server expands the AI agent plugin's feature set, allowing users to leverage high-performance machine learning inference services.

- **Related PR**: [#2806](https://github.com/alibaba/higress/pull/2806) \
  **Contributor**: @C-zhaozhou \
  **Change Log**: This PR makes ai-security-guard compatible with the MultiModalGuard interface, adding support for multimodal APIs and updating the relevant documentation. \
  **Feature Value**: By supporting multimodal APIs, the functionality of ai-security-guard is enhanced, enabling it to handle more complex content security scenarios, improving the user experience and security.

- **Related PR**: [#2727](https://github.com/alibaba/higress/pull/2727) \
  **Contributor**: @Aias00 \
  **Change Log**: This PR adds end-to-end testing support for OpenAI, including test cases for non-streaming and streaming requests. \
  **Feature Value**: The added end-to-end testing for OpenAI ensures the system remains stable and accurate when handling different types of requests, improving the user experience.

- **Related PR**: [#2593](https://github.com/alibaba/higress/pull/2593) \
  **Contributor**: @Xscaperrr \
  **Change Log**: Adds the WorkloadSelector field to limit the scope of EnvoyFilter, ensuring that it does not affect other components in the same namespace in an open-source istio environment. \
  **Feature Value**: By limiting EnvoyFilter to only apply to the Higress Gateway, this feature prevents interference with other istio gateways/sidecars in the environment, enhancing the security and isolation of the configuration.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#2938](https://github.com/alibaba/higress/pull/2938) \
  **Contributor**: @wydream \
  **Change Log**: This PR fixes the issue where prompt attack detection fails due to the lack of AttackLevel field support in MultiModalGuard mode, ensuring that all levels of attacks are correctly identified. \
  **Feature Value**: By adding support for the AttackLevel field, the system's security is improved, preventing high-risk-level prompt attacks from going undetected, ensuring user experience and security.

- **Related PR**: [#2904](https://github.com/alibaba/higress/pull/2904) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR fixes the issue where the original Authorization header might be overwritten when processing HTTP requests. By unconditionally saving and checking for non-empty before writing to the context, it ensures the accuracy and security of authentication information. \
  **Feature Value**: This fix improves the system's security and stability, preventing potential authentication failures or security vulnerabilities due to lost authentication information, enhancing user experience and trust.

- **Related PR**: [#2899](https://github.com/alibaba/higress/pull/2899) \
  **Contributor**: @Jing-ze \
  **Change Log**: This PR optimizes the MCP server, including pre-parsing the host pattern to reduce runtime overhead and removing the unused DomainList field. It also fixes the SSE message format issue, particularly the handling of extra newline characters. \
  **Feature Value**: By improving pattern matching efficiency and memory usage, as well as correcting errors in SSE messages, the user experience and service stability are enhanced, ensuring the correctness and integrity of data transmission.

- **Related PR**: [#2892](https://github.com/alibaba/higress/pull/2892) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR corrects the JSON unmarshalling error when Claude API returns content in array format and removes redundant code structures, improving code quality and maintainability. \
  **Feature Value**: This resolves the message parsing failure due to incorrect data types, enhancing the system's stability and user experience. For users using array as the content format, this fix ensures a smooth message processing flow.

- **Related PR**: [#2882](https://github.com/alibaba/higress/pull/2882) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR addresses the SSE event chunking issue in Claude's streaming response conversion logic, improving protocol auto-conversion and tool invocation state tracking. \
  **Feature Value**: It enhances the bidirectional conversion reliability between Claude and OpenAI-compatible providers, avoiding connection blocking, and enhancing the user experience.

- **Related PR**: [#2865](https://github.com/alibaba/higress/pull/2865) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: This PR solves the issue where SSE connections would be blocked when SSE events were split into multiple chunks. By adding a caching mechanism in the proxy mcp server scenario, it ensures the continuity of data stream processing. \
  **Feature Value**: This fix resolves the potential issue of SSE connection interruption, enhancing the system's stability and user experience. Users will no longer encounter incomplete data reception due to network conditions or server response methods.

- **Related PR**: [#2859](https://github.com/alibaba/higress/pull/2859) \
  **Contributor**: @lcfang \
  **Change Log**: This PR solves the issue of route configuration failure when the registered service instance ports are inconsistent by adding a vport element in the mcpbridge. The main changes include updating the CRD definition, protobuf files, and related generated code. \
  **Feature Value**: This feature ensures that even if the backend instance ports change, the service route configuration remains valid, thereby improving the system's stability and compatibility, providing a more reliable service experience for users.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#2933](https://github.com/alibaba/higress/pull/2933) \
  **Contributor**: @rinfx \
  **Change Log**: This PR removes duplicate think tags in bedrock and vertex, reducing redundant code and improving code readability and maintainability. \
  **Feature Value**: By removing unnecessary duplicate code, the overall quality and development efficiency of the project are improved, making the code structure clearer and easier to maintain and extend.

- **Related PR**: [#2927](https://github.com/alibaba/higress/pull/2927) \
  **Contributor**: @rinfx \
  **Change Log**: This PR modifies the API name extraction logic in the ai-statistics plugin, adjusting the check condition from a fixed length of 5 to at least 3 parts to enhance flexibility and compatibility. \
  **Feature Value**: By relaxing the restriction on API string splitting, the system's support for different format API strings is enhanced, improving the system's adaptability and stability.

- **Related PR**: [#2922](https://github.com/alibaba/higress/pull/2922) \
  **Contributor**: @daixijun \
  **Change Log**: This PR upgrades the Higress SDK package reference in the project from `github.com/alibaba/higress` to `github.com/alibaba/higress/v2` to be compatible with the latest version. \
  **Feature Value**: By updating the package name, the project can introduce and use the latest features and improvements of Higress, enhancing development efficiency and code quality.

- **Related PR**: [#2890](https://github.com/alibaba/higress/pull/2890) \
  **Contributor**: @johnlanni \
  **Change Log**: This PR refactors the `matchDomain` function, introduces the HostMatcher struct and matching types, replaces regular expressions with simple string operations to improve performance, and implements port stripping logic. \
  **Feature Value**: By optimizing the host matching logic, the system performance and code maintainability are improved, making the handling of host headers with port numbers more accurate and efficient, enhancing the user experience.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2915](https://github.com/alibaba/higress/pull/2915) \
  **Contributor**: @a6d9a6m \
  **Change Log**: This PR fixes a broken link in README_JP.md and adds missing parts in README.md, making the multilingual documentation more consistent. \
  **Feature Value**: This improves the accuracy and consistency of the documentation, helping users find relevant information more easily, enhancing the user experience.

- **Related PR**: [#2912](https://github.com/alibaba/higress/pull/2912) \
  **Contributor**: @hanxiantao \
  **Change Log**: This PR optimizes the English and Chinese documentation for the hmac-auth-apisix plugin, adding more detailed configuration explanations, and improving the clarity of the documentation. \
  **Feature Value**: By providing more detailed documentation, it helps developers better understand and use the hmac-auth-apisix plugin, improving the user experience.

- **Related PR**: [#2880](https://github.com/alibaba/higress/pull/2880) \
  **Contributor**: @a6d9a6m \
  **Change Log**: This PR fixes grammatical errors in README.md, README_JP.md, and README_ZH.md files, ensuring the correctness and consistency of the documentation. \
  **Feature Value**: By correcting language errors in the documentation, the quality and readability of the documentation are improved, helping users better understand project information.

- **Related PR**: [#2873](https://github.com/alibaba/higress/pull/2873) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR adds methods to obtain Higress runtime logs and configurations in the non-crash-safe vulnerability issue template, helping to better investigate problems. \
  **Feature Value**: By providing more detailed log and configuration information, users can more easily diagnose and resolve issues, improving the efficiency and accuracy of problem handling.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#2928](https://github.com/alibaba/higress/pull/2928) \
  **Contributor**: @rinfx \
  **Change Log**: This PR updates the test code for the ai-security-guard component, adding new test cases and adjusting some existing test logic. \
  **Feature Value**: By improving the test coverage and accuracy of ai-security-guard, the stability and reliability of the entire project are enhanced, helping developers better understand and maintain related features.

---

## 📊 Release Statistics

- 🚀 New Features: 13
- 🐛 Bug Fixes: 7
- ♻️ Refactoring and Optimization: 5
- 📚 Documentation Updates: 4
- 🧪 Testing Improvements: 1

**Total**: 30 changes (including 2 major updates)

Thank you to all contributors for your hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **4** updates, covering aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Content Distribution

- **New Features**: 1 item
- **Bug Fixes**: 2 items
- **Documentation Updates**: 1 item

### ⭐ Key Focus

This release contains **1** significant update, which is recommended for special attention:

- **feat: Support using a known service in OpenAI LLM provider** ([#589](https://github.com/higress-group/higress-console/pull/589)): This feature allows users to utilize existing service resources within the OpenAI LLM provider, thereby enhancing the flexibility and usability of the system, offering more options to users.

For more details, please refer to the "Important Features in Detail" section below.

---

## 🌟 Important Features in Detail

Here are detailed explanations of the important features and improvements in this release:

### 1. feat: Support using a known service in OpenAI LLM provider

**Related PR**: [#589](https://github.com/higress-group/higress-console/pull/589) | **Contributor**: [@CH3CHO](https://github.com/CH3CHO)

**Usage Background**

In many application scenarios, developers may wish to use their own custom OpenAI service instance instead of the default one. This could be due to specific security requirements, performance optimizations, or infrastructure constraints. This PR meets these needs by introducing support for known services. The target user group includes enterprise-level users and technical experts who require highly customized configurations. This feature addresses the issue of users not being able to flexibly choose and configure OpenAI services, improving the adaptability and user experience of the system.

**Feature Details**

This PR mainly implements the following:
1. Allows users to specify a custom service when configuring the OpenAI LLM provider.
2. Modifies the `OpenaiLlmProviderHandler` class, adding the `buildServiceSource` and `buildUpstreamService` methods to handle the logic for custom services.
3. Adds a delete method with an `internal` parameter to the `WasmPluginInstanceService` interface, supporting finer-grained control.
4. Updates the frontend internationalization resource files, adding prompts related to custom services. The key technical point lies in extending the existing architecture so that the system can recognize and use user-provided custom services while maintaining backward compatibility.

**Usage Instructions**

Enabling and configuring this feature is straightforward. First, when creating or updating an LLM provider, select the "Custom OpenAI Service" option and enter the corresponding service host and service path. Then, the system will automatically use these custom configurations to connect to the OpenAI service. Typical use cases include internally deployed OpenAI service instances within enterprises or environments requiring specific security policies. It's important to ensure that the entered URL is valid and that the service host and service path are correct. Best practice involves thorough testing to ensure that the custom configuration works as expected.

**Feature Value**

This new feature significantly enhances the flexibility and configurability of the system, allowing users to choose the most suitable OpenAI service based on their needs. For enterprise-level users who require high levels of customization, this flexibility is particularly crucial. Additionally, by supporting custom services, the system can better integrate into existing infrastructures, improving overall stability and performance. This is of great significance for maintaining and scaling large application systems. Overall, this feature not only enhances the user experience but also brings higher scalability and reliability to the system.

---

## 📝 Full Changelog

### 🐛 Bug Fixes

- **Related PR**: [#591](https://github.com/higress-group/higress-console/pull/591) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR fixes the issue where mandatory fields were not properly validated when enabling route rewrite configuration, ensuring that both `host` and `newPath.path` must provide valid values to avoid configuration errors. \
  **Feature Value**: By correcting the validation logic for route rewrites, it prevents potential errors caused by incomplete configurations, enhancing the system's stability and user experience.

- **Related PR**: [#590](https://github.com/higress-group/higress-console/pull/590) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed an error in the Route.customLabels handling logic, ensuring that built-in labels are correctly excluded during updates. \
  **Feature Value**: Resolved the conflict between custom labels and built-in labels, ensuring flexibility and accuracy for users when updating route settings.

### 📚 Documentation

- **Related PR**: [#595](https://github.com/higress-group/higress-console/pull/595) \
  **Contributor**: @CH3CHO \
  **Change Log**: Removed irrelevant descriptions from README.md and added a code formatting guide, making the documentation more focused on the project itself. \
  **Feature Value**: By updating README.md, users can more clearly understand the project structure and code formatting requirements, helping new contributors get up to speed quickly.

---

## 📊 Release Statistics

- 🚀 New Features: 1 item
- 🐛 Bug Fixes: 2 items
- 📚 Documentation Updates: 1 item

**Total**: 4 changes (including 1 significant update)

Thank you to all contributors for their hard work! 🎉

