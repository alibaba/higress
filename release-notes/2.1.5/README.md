# Higress


## 📋 Overview of This Release

This release includes **41** updates, covering various aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Content Distribution

- **New Features**: 19
- **Bug Fixes**: 14
- **Refactoring Optimizations**: 2
- **Documentation Updates**: 6

### ⭐ Key Focus

This release includes **2** significant updates, which are recommended for your attention:

- **feat: add DB MCP Server execute, list tables, describe table tools** ([#2506](https://github.com/alibaba/higress/pull/2506)): By adding these tools, users can more conveniently manage and operate databases, enhancing the system's flexibility and usability, making database operations more intuitive and efficient.
- **feat: advanced load balance policies for LLM service through wasm plugin** ([#2531](https://github.com/alibaba/higress/pull/2531)): By introducing advanced load balancing strategies, the performance and resource utilization of LLM services have been improved, allowing users to choose the most suitable strategy to optimize their services based on their needs.

For more details, please refer to the key features section below.

---

## 🌟 Detailed Description of Key Features

Here is a detailed description of the important features and improvements in this release:

### 1. feat: add DB MCP Server execute, list tables, describe table tools

**Related PR**: [#2506](https://github.com/alibaba/higress/pull/2506) | **Contributor**: [hongzhouzi](https://github.com/hongzhouzi)

**Usage Background**

In many application development scenarios, developers need to frequently interact with databases, such as executing SQL statements and viewing table structures. While the existing MCP server supports basic database query functions, it lacks more advanced operation tools. This update adds three tools: `execute` (execute SQL), `list tables` (list tables), and `describe table` (describe table), aiming to meet higher user demands for database management. The target user groups include, but are not limited to, database administrators, backend developers, and application developers who need to frequently interact with databases.

**Feature Details**

Specifically, by modifying the `db.go` file, new database type constants were introduced, and the new tools were registered in the `server.go` file. The newly added tools implement the functionality of executing arbitrary SQL statements, listing all table names, and obtaining detailed information about specific tables. The core technical points lie in using the GORM framework to handle different types of database connections and providing customized SQL query logic for each type of database. Additionally, the code changes also involved optimizing the error handling mechanism, such as unifying the error handling function `handleSQLError`, improving the maintainability of the code. These improvements not only enriched the MCP server's feature set but also enhanced its applicability in various database environments.

**Usage Instructions**

Enabling these new features is straightforward; just ensure that your MCP server configuration includes the correct database DSN and type. For the `execute` tool, users can send requests containing the `sql` parameter to perform INSERT, UPDATE, or DELETE operations; the `list tables` tool requires no additional parameters and can be called directly to return all table names in the current database; the `describe table` tool requires a `table` parameter to specify the table name to view. Typical use cases include, but are not limited to, periodically checking the consistency of database table structures, generating automated scripts, and verifying data before and after migration. It is important to note that when using the `execute` tool, caution should be exercised to avoid executing commands that may compromise data integrity.

**Feature Value**

This feature significantly expands the application scope of the MCP server in database management, enabling users to complete daily tasks more efficiently. It not only simplifies complex manual operations and reduces the likelihood of errors but also provides a solid foundation for building automated O&M processes. Especially for projects that need to work across multiple database platforms, this unified and flexible interface design is undoubtedly a boon. Additionally, by improving error handling logic and adding security measures (such as preventing SQL injection), this PR further ensures the stability and security of the system.

---

### 2. feat: advanced load balance policies for LLM service through wasm plugin

**Related PR**: [#2531](https://github.com/alibaba/higress/pull/2531) | **Contributor**: [rinfx](https://github.com/rinfx)

**Usage Background**

With the widespread application of large language models (LLMs), the demand for high performance and high availability is growing. Traditional load balancing strategies may not meet these requirements, especially when handling a large number of concurrent requests. The new load balancing strategies aim to address these issues by providing smarter request distribution. The target user group includes enterprises and developers who require high-performance and high-availability LLM services.

**Feature Details**

This PR implements three new load balancing strategies: 1. Minimum Load Strategy, implemented using WASM, suitable for [gateway-api-inference-extension](https://github.com/kubernetes-sigs/gateway-api-inference-extension/blob/main/README.md); 2. Global Least Request Strategy based on Redis, which tracks and manages the number of requests for each host via Redis, ensuring that requests are allocated to the host with the least current load; 3. Prompt Prefix Matching Strategy, which selects backend nodes based on prompt prefixes, and if no match is found, uses the Global Least Request Strategy. These strategies are implemented using WASM plugins, providing high scalability and flexibility.

**Usage Instructions**

To enable these load balancing strategies, you need to specify the corresponding strategy type and configuration parameters in the Higress gateway configuration. For example, to enable the Global Least Request Strategy based on Redis, set `lb_policy` to `global_least_request` in the configuration file and provide the FQDN, port, username, and password of the Redis service. For the Prompt Prefix Matching Strategy, set `lb_policy` to `prefix_cache` and make the corresponding configuration. Best practice is to choose the appropriate strategy based on the actual application scenario and regularly monitor and adjust the configuration to optimize performance.

**Feature Value**

These new load balancing strategies bring significant performance improvements to LLM services. The Minimum Load Strategy ensures that requests are allocated to the host with the least current load, thereby improving response speed and resource utilization. The Global Least Request Strategy based on Redis further optimizes resource allocation by tracking the number of requests for each host in real time. The Prompt Prefix Matching Strategy improves processing efficiency by caching and reusing KV Cache. These features not only enhance system performance and stability but also improve user experience, especially in high-concurrency scenarios.

---

## 📝 Complete Changelog

### 🚀 New Features (Features)

- **Related PR**: [#2533](https://github.com/alibaba/higress/pull/2533)
  **Contributor**: johnlanni
  **Change Log**: Added support for the subPath field, allowing users to configure rules for removing request path prefixes, and updated the Chinese and English documentation to include usage instructions for the new feature.
  **Feature Value**: By introducing the subPath configuration option, the flexibility and customizability of the AI proxy plugin have been enhanced, enabling developers to more finely control the request path processing logic and improve the user experience.

- **Related PR**: [#2514](https://github.com/alibaba/higress/pull/2514)
  **Contributor**: daixijun
  **Change Log**: This PR commented out the default tracing.skywalking configuration in values.yaml, resolving the issue where skywalking configurations were automatically added when users chose other tracing types.
  **Feature Value**: By removing unnecessary skywalking configurations, conflicts with user-defined tracing settings are avoided, enhancing the system's flexibility and user experience.

- **Related PR**: [#2509](https://github.com/alibaba/higress/pull/2509)
  **Contributor**: daixijun
  **Change Log**: This PR implemented handling of the OpenAI responses interface Body and added support for the Volcano Ark large model responses interface, achieved by extending the logic in the provider/doubao.go file.
  **Feature Value**: The new feature enables the system to support more types of AI response processing, particularly for users using the Volcano Ark large model, significantly enhancing the system's compatibility and flexibility.

- **Related PR**: [#2488](https://github.com/alibaba/higress/pull/2488)
  **Contributor**: rinfx
  **Change Log**: Added `trace_span_key` and `as_separate_log_field` configuration options, allowing the keys for logging and span attribute recording to be different and enabling log content to exist as separate fields.
  **Feature Value**: By providing more flexible logging and tracing data recording methods, the system's monitoring capabilities have been enhanced, helping developers better understand and optimize application performance.

- **Related PR**: [#2485](https://github.com/alibaba/higress/pull/2485)
  **Contributor**: johnlanni
  **Change Log**: This PR introduced the errorResponseTemplate feature, allowing the mcp server plugin to customize response content when the backend HTTP status code is greater than 300.
  **Feature Value**: This feature allows users to customize error response templates based on actual conditions, enhancing the system's flexibility and user experience, especially by providing friendlier feedback in handling exceptions.

- **Related PR**: [#2460](https://github.com/alibaba/higress/pull/2460)
  **Contributor**: erasernoob
  **Change Log**: This PR modified the message endpoint sending logic in the mcp-session plugin's SSE server, allowing it to pass query parameters to the REST API server and URL-encode the sessionID.
  **Feature Value**: By supporting the SSE server to pass query parameters to the REST API server, the system's flexibility and functional integration capabilities have been enhanced, making it easier for users to customize service requests.

- **Related PR**: [#2450](https://github.com/alibaba/higress/pull/2450)
  **Contributor**: kenneth-bro
  **Change Log**: Added a sector market MCP Server, integrating the latest real-time market data and constituent stock information for industry and concept sectors.
  **Feature Value**: Provides users with detailed market data analysis tools, helping investors track the performance of industry and concept sectors in real time and make more informed investment decisions.

- **Related PR**: [#2440](https://github.com/alibaba/higress/pull/2440)
  **Contributor**: johnlanni
  **Change Log**: This PR fixed two issues in istio and envoy and added a new wasm API to support injecting encoding filter chains during the encodeHeader phase.
  **Feature Value**: By addressing consistency hashing-related issues and providing a new API, this update enhances the system's stability and flexibility, allowing users to more finely control the request processing process.

- **Related PR**: [#2431](https://github.com/alibaba/higress/pull/2431)
  **Contributor**: mirror58229
  **Change Log**: This PR added default route support for WANX image and video synthesis and updated the relevant README files to reflect these changes.
  **Feature Value**: By introducing default route support, users can more flexibly handle WANX image and video synthesis requests, enhancing the system's availability and user experience.

- **Related PR**: [#2424](https://github.com/alibaba/higress/pull/2424)
  **Contributor**: wydream
  **Change Log**: This PR added support for the OpenAI Fine-Tuning API in the ai-proxy plugin, including path routing, capability configuration, and related constant definitions.
  **Feature Value**: By introducing support for the Fine-Tuning API, users can now leverage this service for more advanced model fine-tuning tasks, enhancing the system's flexibility and functionality.

- **Related PR**: [#2409](https://github.com/alibaba/higress/pull/2409)
  **Contributor**: johnlanni
  **Change Log**: Added a Wasm-Go plugin named mcp-router, supporting dynamic routing for MCP tool requests, including the creation of Dockerfile, Makefile, and related documentation.
  **Feature Value**: This plugin allows aggregating different tools from multiple backend MCP servers through a single gateway endpoint, simplifying multi-service integration and management, and enhancing the system's flexibility and scalability.

- **Related PR**: [#2404](https://github.com/alibaba/higress/pull/2404)
  **Contributor**: 007gzs
  **Change Log**: This PR added `reasoning_content` support for the AI data masking feature and supported returning multiple `index` groups in the request, enhancing the flexibility and diversity of AI responses.
  **Feature Value**: By adding support for `reasoning_content` and allowing multiple `index` groups to be returned, users can more flexibly handle AI response data, enhancing the system's adaptability and user experience in complex scenarios.

- **Related PR**: [#2391](https://github.com/alibaba/higress/pull/2391)
  **Contributor**: daixijun
  **Change Log**: Adjusted the AI proxy's streaming response structure to output null when the usage, logprobs, and finish_reason fields are empty, maintaining consistency with the OpenAI interface.
  **Feature Value**: By maintaining consistency with the OpenAI interface, the system's compatibility and user experience have been improved, making it easier for developers to integrate and use APIs.

- **Related PR**: [#2389](https://github.com/alibaba/higress/pull/2389)
  **Contributor**: NorthernBob
  **Change Log**: This PR implemented one-click Kubernetes deployment support for the plugin server and configured the default download URL for the plugin. Changes included adding and modifying multiple Helm template files to support the plugin server.
  **Feature Value**: By supporting one-click Kubernetes deployment and presetting the plugin download URL, the process of deploying and using plugins in K8s environments has been simplified, enhancing ease of use and efficiency.

- **Related PR**: [#2378](https://github.com/alibaba/higress/pull/2378)
  **Contributor**: mirror58229
  **Change Log**: This PR added support paths for WANXIANG image/video generation in the ai-proxy and added a new configuration item in ai-statistics to avoid OpenAI-related errors.
  **Feature Value**: Provides users with new image and video generation features while ensuring system stability and compatibility through the new configuration item, enhancing the user experience.

- **Related PR**: [#2343](https://github.com/alibaba/higress/pull/2343)
  **Contributor**: hourmoneys
  **Change Log**: This PR introduced an MCP service for AI-based bidding information, including detailed Chinese and English README files and configuration descriptions.
  **Feature Value**: The new feature allows users to query bid lists by keyword, enhancing the ability of enterprises to acquire projects and customers, providing more comprehensive and accurate information support.

- **Related PR**: [#1925](https://github.com/alibaba/higress/pull/1925)
  **Contributor**: kai2321
  **Change Log**: This PR implemented the AI-image-reader plugin, parsing image content by interfacing with OCR services (such as Alibaba Cloud Lingji). Added related Go code and Chinese and English documentation.
  **Feature Value**: This feature enables users to automatically read and process text information in images using AI technology, enhancing the system's intelligence level and user experience.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#2524](https://github.com/alibaba/higress/pull/2524)
  **Contributor**: daixijun
  **Change Log**: This PR fixed the issue of the `stream_options` parameter being misused on non-openai/v1/chatcompletions interfaces, limiting the parameter to only take effect on the specified interface to avoid errors.
  **Feature Value**: Ensured the correctness of API calls, preventing errors caused by misadded parameters, enhancing the system's stability and user experience.

- **Related PR**: [#2516](https://github.com/alibaba/higress/pull/2516)
  **Contributor**: HecarimV
  **Change Log**: This PR fixed the lack of system prompt support in the AI Proxy component by adding system message handling capability to Bedrock API requests. Specifically, it added a System field to the request body structure and updated the request construction logic to conditionally include system messages.
  **Feature Value**: Enhanced the AI proxy's support for Bedrock services, allowing users to include system-level instructions or information when sending requests, which helps in more precisely controlling the style and direction of generated content, enhancing user experience and application flexibility.

- **Related PR**: [#2497](https://github.com/alibaba/higress/pull/2497)
  **Contributor**: johnlanni
  **Change Log**: This PR fixed the issue of incorrect decoding behavior when the configured URL path contains URL-encoded parts, achieved by modifying the lib-side code.
  **Feature Value**: This fix ensures that requests with URL-encoded parts in the path are correctly decoded, enhancing the system's stability and user experience.

- **Related PR**: [#2480](https://github.com/alibaba/higress/pull/2480)
  **Contributor**: HecarimV
  **Change Log**: This PR fixed the issue of AWS Bedrock supporting additional request fields, ensuring that the AdditionalModelRequestFields field is properly initialized, avoiding potential null pointer exceptions.
  **Feature Value**: By adding support for additional model request fields, users can more flexibly configure AWS Bedrock services, enhancing the customizability and stability of API calls.

- **Related PR**: [#2475](https://github.com/alibaba/higress/pull/2475)
  **Contributor**: daixijun
  **Change Log**: Fixed the 404 issue caused by incorrect customPath transmission when openaiCustomUrl is configured for a single interface and the path prefix is not /v1. Adjusted the request handling logic to ensure compatibility.
  **Feature Value**: This fix resolved the 404 errors encountered by users under specific conditions, enhancing the stability and user experience when using custom OpenAI service paths.

- **Related PR**: [#2469](https://github.com/alibaba/higress/pull/2469)
  **Contributor**: luoxiner
  **Change Log**: Fixed the issue of excessive logging during MCP server discovery when Nacos is unavailable, reducing unnecessary log output by fixing the erroneous log recording call.
  **Feature Value**: Reduced the amount of logs generated when the Nacos service is unreachable, avoiding storage pressure and performance issues due to rapidly growing log files, enhancing the system's stability and user experience.

- **Related PR**: [#2445](https://github.com/alibaba/higress/pull/2445)
  **Contributor**: johnlanni
  **Change Log**: Fixed the issue of the mcp server not returning a body when returning a status, changed to respond via sse; and refactored makeHttpResponse.
  **Feature Value**: Resolved potential errors due to missing response bodies, enhancing the system's stability and user experience, ensuring correct communication between the backend and frontend.

- **Related PR**: [#2443](https://github.com/alibaba/higress/pull/2443)
  **Contributor**: Colstuwjx
  **Change Log**: This PR fixed an issue by adding a missing annotation in the controller service account, allowing users to set annotations for the controller service account.
  **Feature Value**: This change allows users to more flexibly configure service accounts, such as binding AWS IAM roles to the service account via annotations, enabling authentication for AWS resources.

- **Related PR**: [#2441](https://github.com/alibaba/higress/pull/2441)
  **Contributor**: wydream
  **Change Log**: This PR standardized the naming conventions for API name constants and corrected the API name mapping error in the getApiName function, ensuring that API requests are correctly matched.
  **Feature Value**: By correcting API name spelling and format inconsistencies, the system's stability and reliability have been enhanced, avoiding functional failures or 404 errors due to path mismatches.

- **Related PR**: [#2423](https://github.com/alibaba/higress/pull/2423)
  **Contributor**: johnlanni
  **Change Log**: This PR fixed a potential controller crash issue when configuring the MCP server for SSE forwarding, by modifying the relevant logic in the ingress_config.go file to prevent abnormal situations.
  **Feature Value**: Fixed the potential controller crash issue, enhancing the system's stability and reliability, ensuring that users do not encounter service interruptions when using the SSE forwarding feature.

- **Related PR**: [#2408](https://github.com/alibaba/higress/pull/2408)
  **Contributor**: daixijun
  **Change Log**: Adjusted the Gemini API's finishReason to lowercase and fixed the missing finishReason content in the streaming response, ensuring consistency and completeness with the OpenAI API.
  **Feature Value**: This fix enhances API compatibility and stability, ensuring that users receive consistent and complete response results when using the Gemini provider, enhancing the user experience.

- **Related PR**: [#2405](https://github.com/alibaba/higress/pull/2405)
  **Contributor**: Erica177
  **Change Log**: Corrected the spelling error of `McpStreambleProtocol`, ensuring the protocol support logic, type mapping, and route rewrite rules are correct.
  **Feature Value**: Fixed the protocol recognition and mapping issues caused by constant name spelling errors, enhancing the system's stability and reliability.

- **Related PR**: [#2402](https://github.com/alibaba/higress/pull/2402)
  **Contributor**: HecarimV
  **Change Log**: Fixed the Bedrock Sigv4 signature mismatch issue in the AI proxy and improved the modelId decoding logic to avoid potential data pollution risks.
  **Feature Value**: This fix enhances system stability, preventing service call failures due to incorrect model IDs, and improves the user experience and system reliability.

- **Related PR**: [#2398](https://github.com/alibaba/higress/pull/2398)
  **Contributor**: Erica177
  **Change Log**: Corrected the spelling error in the `McpStreambleProtocol` constant, changing 'mcp-streamble' to 'mcp-streamable', and adjusted related references to ensure the consistency and correctness of the protocol name.
  **Feature Value**: Fixed potential protocol matching failures or configuration parsing issues due to spelling errors, enhancing the system's stability and reliability, and avoiding service anomalies caused by such simple errors.

### ♻️ Refactoring Optimizations (Refactoring)

- **Related PR**: [#2458](https://github.com/alibaba/higress/pull/2458)
  **Contributor**: johnlanni
  **Change Log**: This PR updated the mcp server's dependency on the wasm-go repository to the latest version, adjusting the dependency path in the go.mod file to ensure the project uses the latest codebase.
  **Feature Value**: By depending on the latest wasm-go repository, the project can utilize the latest features and performance optimizations, enhancing the system's stability and compatibility.

- **Related PR**: [#2403](https://github.com/alibaba/higress/pull/2403)
  **Contributor**: johnlanni
  **Change Log**: This PR standardized the newline character markers in the MCP session filter, achieving consistency by modifying two lines of code in the sse.go file.
  **Feature Value**: Standardizing newline character markers reduces confusion caused by inconsistent formatting, enhancing code readability and maintainability, making it easier for developers to understand and use the related features.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2536](https://github.com/alibaba/higress/pull/2536)
  **Contributor**: johnlanni
  **Change Log**: This PR primarily updated the version number and version information in relevant configuration files to prepare for the 2.1.5 release.
  **Feature Value**: By updating the version number, the latest software status is reflected, allowing users to clearly understand the current software version and its stability.

- **Related PR**: [#2503](https://github.com/alibaba/higress/pull/2503)
  **Contributor**: CH3CHO
  **Change Log**: Corrected the spelling error of the configuration item name in the ai-proxy plugin README, changing `vertexGeminiSafetySetting` to `geminiSafetySetting`.
  **Feature Value**: Ensures the documentation is accurate, preventing users from being unable to set up correctly due to configuration item name errors, enhancing the user experience and document readability.

- **Related PR**: [#2446](https://github.com/alibaba/higress/pull/2446)
  **Contributor**: johnlanni
  **Change Log**: Updated the version number to 2.1.5-rc.1 and synchronized the version information in relevant files, including Makefile, VERSION file, and Helm charts.
  **Feature Value**: This PR primarily updated the project's version information, ensuring that all related configuration files and documents reflect the latest version number, providing accurate version tracking information for users.

- **Related PR**: [#2433](https://github.com/alibaba/higress/pull/2433)
  **Contributor**: johnlanni
  **Change Log**: This PR added the English and Chinese release notes for version 2.1.4 and updated the license configuration file to exclude the release-notes directory.
  **Feature Value**: By providing detailed release notes, users can better understand the new features and fixed issues in the new version, making it easier to adopt and use the software's new features.

- **Related PR**: [#2418](https://github.com/alibaba/higress/pull/2418)
  **Contributor**: xuruidong
  **Change Log**: Fixed a broken link issue in the mcp-servers README_zh.md file, ensuring the correctness and availability of the document links.
  **Feature Value**: By correcting the broken links in the documentation, the user experience when reading and using the documentation is enhanced, avoiding information retrieval barriers due to invalid links.

- **Related PR**: [#2327](https://github.com/alibaba/higress/pull/2327)
  **Contributor**: hourmoneys
  **Change Log**: This PR primarily updated the mcp-server-related documentation, including content adjustments in README_ZH.md and mcp-server.yaml configuration files.
  **Feature Value**: By updating the documentation, users can more clearly understand and use the mcp-shebao-tools, providing detailed explanations and configuration examples, enhancing the user experience.

---

## 📊 Release Statistics

- 🚀 New Features: 19
- 🐛 Bug Fixes: 14
- ♻️ Refactoring Optimizations: 2
- 📚 Documentation Updates: 6

**Total**: 41 changes (including 2 significant updates)

Thank you to all contributors for their hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **8** updates, covering multiple aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 5 items
- **Bug Fixes**: 2 items
- **Testing Improvements**: 1 item

### ⭐ Key Focus

This release contains **1** major update, which is recommended for special attention:

- **Feature/issue 514 mcp server manage** ([#530](https://github.com/higress-group/higress-console/pull/530)): The newly added mcp server console management feature allows users to more conveniently manage and configure the mcp server through the interface, enhancing user experience and operational efficiency.

For more details, please refer to the important features section below.

---

## 🌟 Detailed Description of Important Features

Below are detailed descriptions of the key features and improvements in this release:

### 1. Feature/issue 514 mcp server manage

**Related PR**: [#530](https://github.com/higress-group/higress-console/pull/530) | **Contributor**: [Thomas-Eliot](https://github.com/Thomas-Eliot)

**Usage Background**

In modern microservice architectures, the mcp server serves as a critical component responsible for managing and facilitating communication between services. However, existing management systems lack centralized and visual management capabilities for the mcp server, leading to manual configuration and management by operations personnel, which is inefficient and prone to errors. To address this issue, a new mcp server console management feature has been added, allowing users to easily create, update, delete, and query mcp server instances through a graphical interface. This feature is primarily aimed at system administrators and operations personnel to improve their work efficiency and reduce errors.

**Feature Details**

This change mainly implements the following functionalities:
1. **Create mcp server**: Users can create new mcp server instances by filling in the necessary parameters through the console interface.
2. **Update mcp server**: Users can modify the configuration information of existing mcp servers and save it via the console interface.
3. **Delete mcp server**: Users can select and delete mcp server instances through the console interface.
4. **Query mcp server**: Users can query all mcp server instances and their detailed information.

Technically, this was achieved by building RESTful API interfaces using the Spring Boot framework and generating API documentation with Swagger. A new `McpServerController` class was added to handle HTTP requests related to the mcp server. Additionally, the Dockerfile was modified to include the copying and permission settings for mcp-related tools. Furthermore, adjustments were made to the SDK configuration files to support the new features.

**Usage Instructions**

To enable and configure this feature, follow these steps:
1. **Start the application**: Ensure that the Higress Console application is correctly deployed and running.
2. **Access the console**: Access the Higress Console URL through a web browser to enter the console interface.
3. **Create mcp server**: In the console, select the "mcp server" tab, click the "Create" button, fill in the necessary parameters (such as name, type, etc.), and then click the "Save" button.
4. **Update mcp server**: Find the instance you need to update in the mcp server list, click the "Edit" button, modify the relevant information, and then click the "Save" button.
5. **Delete mcp server**: Find the instance you need to delete in the mcp server list, click the "Delete" button, and confirm the deletion operation.
6. **Query mcp server**: View all instances and their detailed information in the mcp server list.
**Note**: Before performing any operations, ensure that data is backed up to prevent data loss due to accidental operations.

**Feature Value**

By adding the mcp server console management feature, users can more conveniently manage and configure mcp server instances, significantly improving the system's usability and maintainability. Specifically, this feature brings the following benefits:
1. **Improved Efficiency**: Users no longer need to manually write configuration files or execute complex command-line operations; they can manage mcp servers through a simple graphical interface.
2. **Reduced Error Rate**: The visual operation interface reduces the likelihood of errors caused by manual configuration.
3. **Enhanced User Experience**: An intuitive operation interface allows users to quickly get started, reducing the learning curve.
4. **Increased System Stability**: Unified console management ensures consistency and standardization in configurations, reducing system instability caused by inconsistent configurations.

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#540](https://github.com/higress-group/higress-console/pull/540)
  **Contributor**: CH3CHO
  **Change Log**: This PR adds a new LLM provider type: vertex, by extending the `LlmProviderType` enum class and adding a new `VertexLlmProviderHandler` class.
  **Feature Value**: Adding support for vertex as an LLM provider will allow users to utilize the services provided by vertex, enriching the system's functionality and meeting the needs of more scenarios.

- **Related PR**: [#538](https://github.com/higress-group/higress-console/pull/538)
  **Contributor**: zhangjingcn
  **Change Log**: This PR introduces errorResponseTemplate support for the mcp-server plugin, allowing users to customize error response templates and correcting the documentation regarding error response trigger conditions and GJSON path escaping.
  **Feature Value**: By providing the ability to customize error responses, this feature enhances user experience and flexibility, enabling developers to adjust error message display based on actual needs, thus better controlling the application's behavior.

- **Related PR**: [#529](https://github.com/higress-group/higress-console/pull/529)
  **Contributor**: CH3CHO
  **Change Log**: This PR adds the functionality to configure multiple model mapping rules for AI routing upstreams, implemented through an added pop-up dialog for advanced configuration editing.
  **Feature Value**: Users can more flexibly manage model mappings for AI services, improving configuration efficiency and flexibility, and meeting the needs of diverse scenarios.

- **Related PR**: [#528](https://github.com/higress-group/higress-console/pull/528)
  **Contributor**: cr7258
  **Change Log**: Changes the default PVC access mode from ReadWriteMany to ReadWriteOnce, which is more suitable for most default settings.
  **Feature Value**: This change reduces unnecessary complexity and improves resource utilization efficiency, while providing flexibility for users who need multiple replicas.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#537](https://github.com/higress-group/higress-console/pull/537)
  **Contributor**: CH3CHO
  **Change Log**: Replaces `URL.parse` with `new URL()` to resolve compatibility issues in older browser versions.
  **Feature Value**: Enhances the application's compatibility across different browser versions, ensuring a wider range of users can use the related features normally.

- **Related PR**: [#525](https://github.com/higress-group/higress-console/pull/525)
  **Contributor**: NorthernBob
  **Change Log**: This PR corrects a spelling error in the configuration file, changing 'UrlPattern' to 'urlPattern', ensuring consistent variable naming.
  **Feature Value**: Correcting the spelling error ensures the correctness and consistency of the configuration file, avoiding service configuration issues due to case sensitivity, thereby improving system stability and user experience.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#526](https://github.com/higress-group/higress-console/pull/526)
  **Contributor**: CH3CHO
  **Change Log**: This PR adds a unit test case to check if the Wasm plugin image is the latest version. It compares the currently used image tag with the latest image tag manifest.
  **Feature Value**: This feature ensures that the Wasm plugin always uses the latest image, improving system stability and security, and avoiding security vulnerabilities or other issues caused by using outdated images.

---

## 📊 Release Statistics

- 🚀 New Features: 5 items
- 🐛 Bug Fixes: 2 items
- 🧪 Testing Improvements: 1 item

**Total**: 8 changes (including 1 major update)

Thank you to all contributors for their hard work! 🎉

