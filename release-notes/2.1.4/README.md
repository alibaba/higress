# Higress Core

## 📌feature
### Support for Google Cloud Vertex AI service
+ Related PR: [https://github.com/alibaba/higress/pull/2119](https://github.com/alibaba/higress/pull/2119)
+ Contributor: [HecarimV](https://github.com/HecarimV)
+ Change Log: Added support for Google Cloud Vertex AI, allowing proxying of Vertex services through the OpenAI protocol.
+ Feature Value: This feature extends the compatibility of AI proxies, enabling users to leverage models and capabilities provided by Vertex AI.

### New HackMD MCP Server
+ Related PR: [https://github.com/alibaba/higress/pull/2260](https://github.com/alibaba/higress/pull/2260)
+ Contributor: [Whitea029](https://github.com/Whitea029)
+ Change Log: Added a new HackMD MCP server feature, supporting user interaction with the HackMD platform via the MCP protocol, including user data management, note operations, and team collaboration features.
+ Feature Value: This PR adds support for HackMD, extending the functionality of the MCP server and enhancing user collaboration capabilities.

### New Junrun Human Resources Social Security Tool MCP Server
+ Related PR: [https://github.com/alibaba/higress/pull/2303](https://github.com/alibaba/higress/pull/2303)
+ Contributor: [hourmoneys](https://github.com/hourmoneys)
+ Change Log: Submitted MCP to REST configuration for the social security tool MCP server by Junrun Human Resources, detailing its functions, usage, and configuration, including descriptions and examples of multiple API interfaces.
+ Feature Value: Provides developers with a clear guide to using the social security calculation tool, enhancing the tool's integrability and ease of use.

### Add Claude Image Understanding and Tools Invocation Capabilities
+ Related PR: [https://github.com/alibaba/higress/pull/2385](https://github.com/alibaba/higress/pull/2385)
+ Contributor: [daixijun](https://github.com/daixijun)
+ Change Log: Added Claude image understanding and tool invocation capabilities to the AI proxy, supporting streaming output and token statistics, compatible with the OpenAI interface specification, and extending the models interface support.
+ Feature Value: This PR enhances the AI proxy's functionality, enabling it to handle image input and tool invocation, improving compatibility with Claude and user experience.

### New Gemini Model Support
+ Related PR: [https://github.com/alibaba/higress/pull/2380](https://github.com/alibaba/higress/pull/2380)
+ Contributor: [daixijun](https://github.com/daixijun)
+ Change Log: Added support for the Gemini model, including model list interface, image generation interface, and text-to-image conversation capabilities, extending the AI proxy's functional scope.
+ Feature Value: Full support for the Gemini model, enhancing the AI proxy's multi-model compatibility and image generation capabilities.

### New Amazon Bedrock Image Generation Support
+ Related PR: [https://github.com/alibaba/higress/pull/2212](https://github.com/alibaba/higress/pull/2212)
+ Contributor: [daixijun](https://github.com/daixijun)
+ Change Log: Added support for Amazon Bedrock image generation, extending the AI proxy's functionality and allowing text-to-image generation via the Bedrock API.
+ Feature Value: Provides users with a new AI image generation method, enhancing system functionality and flexibility.

### New Model Mapping Regular Expression Support
+ Related PR: [https://github.com/alibaba/higress/pull/2358](https://github.com/alibaba/higress/pull/2358)
+ Contributor: [daixijun](https://github.com/daixijun)
+ Change Log: Added support for regular expressions in model mapping, allowing more flexible model name replacements and solving specific model invocation issues.
+ Feature Value: This PR enhances the AI proxy plugin's functionality, making model mapping more flexible and powerful, improving system configurability and applicability.

### Global Threshold Configuration for Cluster Rate Limiting Rules
+ Related PR: [https://github.com/alibaba/higress/pull/2262](https://github.com/alibaba/higress/pull/2262)
+ Contributor: [hanxiantao](https://github.com/hanxiantao)
+ Change Log: Added support for global threshold configuration of cluster rate limiting rules, enhancing the flexibility and configurability of rate limiting strategies.
+ Feature Value: This PR adds global rate limiting threshold configuration to the cluster rate limiting plugin, allowing unified rate limiting thresholds for the entire custom rule set, enhancing the flexibility and applicability of rate limiting strategies.

### New OpenAI Files and Batches Interface Support
+ Related PR: [https://github.com/alibaba/higress/pull/2355](https://github.com/alibaba/higress/pull/2355)
+ Contributor: [daixijun](https://github.com/daixijun)
+ Change Log: Added support for OpenAI and Qwen /v1/files and /v1/batches interfaces to the AI proxy module, extending AI service compatibility.
+ Feature Value: Added file and batch interface support, enhancing the AI proxy's compatibility with multiple services.

### New OpenAI Compatible Interface Mapping Capability
+ Related PR: [https://github.com/alibaba/higress/pull/2341](https://github.com/alibaba/higress/pull/2341)
+ Contributor: [daixijun](https://github.com/daixijun)
+ Change Log: Added support for OpenAI-compatible image generation, image editing, and audio processing interfaces, extending the AI proxy's functionality and making it compatible with more models.
+ Feature Value: This PR adds OpenAI-compatible interface mapping capability to the AI proxy, enhancing system flexibility and expandability.

### New Access Log Request Plugin
+ Related PR: [https://github.com/alibaba/higress/pull/2265](https://github.com/alibaba/higress/pull/2265)
+ Contributor: [forgottener](https://github.com/forgottener)
+ Change Log: Added the ability to record request headers, request bodies, response headers, and response bodies in Higress access logs, enhancing log traceability.
+ Feature Value: This PR enhances Higress's logging functionality, allowing developers to more comprehensively monitor and debug HTTP communication processes.

### New dify ai-proxy e2e Testing
+ Related PR: [https://github.com/alibaba/higress/pull/2319](https://github.com/alibaba/higress/pull/2319)
+ Contributor: [VinciWu557](https://github.com/VinciWu557)
+ Change Log: Added dify ai-proxy plugin e2e testing, supporting full end-to-end testing of dify models to ensure their functionality and stability.
+ Feature Value: Adds complete e2e testing to the dify ai-proxy plugin, enhancing its reliability and maintainability.

### Frontend Gray Release Unique Identifier Configuration
+ Related PR: [https://github.com/alibaba/higress/pull/2371](https://github.com/alibaba/higress/pull/2371)
+ Contributor: [heimanba](https://github.com/heimanba)
+ Change Log: Added uniqueGrayTag configuration item detection, supporting the setting of unique identifier cookies based on user-defined uniqueGrayTag, enhancing gray release flexibility and configurability.
+ Feature Value: This PR enhances frontend gray release configuration, allowing users to define unique identifiers, optimizing gray traffic control mechanisms, and enhancing system scalability and user experience.

### New Doubao Image Generation Interface Support
+ Related PR: [https://github.com/alibaba/higress/pull/2331](https://github.com/alibaba/higress/pull/2331)
+ Contributor: [daixijun](https://github.com/daixijun)
+ Change Log: Added support for the Doubao image generation interface, extending the AI proxy's functionality to handle image generation requests.
+ Feature Value: This PR adds support for Doubao image generation to the AI proxy, enhancing system capabilities and flexibility.

### WasmPlugin E2E Testing Skip Building Higress Controller Image
+ Related PR: [https://github.com/alibaba/higress/pull/2264](https://github.com/alibaba/higress/pull/2264)
+ Contributor: [cr7258](https://github.com/cr7258)
+ Change Log: Added the ability to skip building the Higress controller development image during WasmPlugin E2E testing, enhancing testing efficiency.
+ Feature Value: This PR optimizes the WasmPlugin testing process, allowing users to selectively skip unnecessary image building steps, improving testing efficiency.

### MCP Server API Authentication Support
+ Related PR: [https://github.com/alibaba/higress/pull/2241](https://github.com/alibaba/higress/pull/2241)
+ Contributor: [johnlanni](https://github.com/johnlanni)
+ Change Log: This PR introduces comprehensive API authentication for the Higress MCP Server plugin, supporting HTTP Basic, HTTP Bearer, and API Key authentication via OAS3 security schemes, enhancing secure integration with backend REST APIs.
+ Feature Value: This PR adds support for multiple API authentication methods to the MCP Server, enhancing system security and flexibility, and significantly helping the community in building secure microservice architectures.

### GitHub Action for Synchronizing CRD Files
+ Related PR: [https://github.com/alibaba/higress/pull/2268](https://github.com/alibaba/higress/pull/2268)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: This PR adds a GitHub Action to automatically copy CRD definition files from the api folder to the helm folder on the main branch and create a PR.
+ Feature Value: Implements automated synchronization of CRD files, improving the efficiency and consistency of the development process.

### Enhanced Logging for ai-search Plugin
+ Related PR: [https://github.com/alibaba/higress/pull/2323](https://github.com/alibaba/higress/pull/2323)
+ Contributor: [johnlanni](https://github.com/johnlanni)
+ Change Log: Added detailed logging information to the ai-search plugin, including request URL, cluster name, and search rewrite model, aiding in debugging and monitoring.
+ Feature Value: Added more detailed log information, making it easier for developers to diagnose issues and optimize performance.

### Update CRD Files in Helm Folder
+ Related PR: [https://github.com/alibaba/higress/pull/2392](https://github.com/alibaba/higress/pull/2392)
+ Contributor: [github-actions[bot]](https://github.com/apps/github-actions)
+ Change Log: Updated the CRD files in the Helm folder, adding configuration support and metadata fields for MCP servers, enhancing the flexibility and extensibility of resource definitions.
+ Feature Value: Improved Kubernetes resource definitions, providing more comprehensive support for MCP server configurations.

### Add Upstream Operation Support to Wasm ABI
+ Related PR: [https://github.com/alibaba/higress/pull/2387](https://github.com/alibaba/higress/pull/2387)
+ Contributor: [johnlanni](https://github.com/johnlanni)
+ Change Log: This PR adds Wasm ABI related to upstream operations, preparing for future implementation of fine-grained load balancing strategies (e.g., GPU-based LLM scenarios) in Wasm plugins.
+ Feature Value: Lays the foundation for Wasm plugins to support more complex load balancing strategies, enhancing system flexibility and extensibility.

### Modify Log Level for key-auth Plugin
+ Related PR: [https://github.com/alibaba/higress/pull/2275](https://github.com/alibaba/higress/pull/2275)
+ Contributor: [lexburner](https://github.com/lexburner)
+ Change Log: Changed the log level in the key-auth plugin from WARN to DEBUG to reduce unnecessary warning messages and improve log readability and accuracy.
+ Feature Value: Fixed unnecessary warning logs in the key-auth plugin, optimizing log output and enhancing the clarity of system logs.

## 📌bugfix
### Fix WasmPlugin Generation Logic
+ Related PR: [https://github.com/alibaba/higress/pull/2237](https://github.com/alibaba/higress/pull/2237)
+ Contributor: [Erica177](https://github.com/Erica177)
+ Change Log: Fixed the issue of not setting the fail strategy in the WasmPlugin generation logic and added the FAIL_OPEN strategy to improve system stability.
+ Feature Value: Added a default fail strategy to WasmPlugin to prevent system anomalies due to plugin failures.

### Fix OpenAI Custom Path Pass-Through Issue
+ Related PR: [https://github.com/alibaba/higress/pull/2364](https://github.com/alibaba/higress/pull/2364)
+ Contributor: [daixijun](https://github.com/daixijun)
+ Change Log: Fixed the issue where an error occurred when passing unsupported API paths after configuring openaiCustomUrl, and added support for multiple OpenAI API paths.
+ Feature Value: This PR corrects the proxy service logic under custom path configuration, improving compatibility and stability.

### Fix Nacos MCP Tool Configuration Handling Logic
+ Related PR: [https://github.com/alibaba/higress/pull/2394](https://github.com/alibaba/higress/pull/2394)
+ Contributor: [Erica177](https://github.com/Erica177)
+ Change Log: Fixed the Nacos MCP tool configuration handling logic and added unit tests to ensure the stability and correctness of the configuration update and listening mechanism.
+ Feature Value: Improved the configuration handling logic of the MCP service, enhancing system stability and maintainability.

### Fix Mixed Line Break Handling in SSE Responses
+ Related PR: [https://github.com/alibaba/higress/pull/2344](https://github.com/alibaba/higress/pull/2344)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Fixed the issue of mixed line break handling in SSE responses, improving the SSE data parsing logic to ensure correct handling of different line break combinations.
+ Feature Value: This PR resolves the issue of incompatible line break handling in SSE responses, enhancing the system's compatibility and stability with SSE data.

### Fix proxy-wasm-cpp-sdk Dependency Issue
+ Related PR: [https://github.com/alibaba/higress/pull/2281](https://github.com/alibaba/higress/pull/2281)
+ Contributor: [johnlanni](https://github.com/johnlanni)
+ Change Log: Fixed the emsdk configuration issue in the proxy-wasm-cpp-sdk dependency, addressing the memory allocation failure when handling large request bodies.
+ Feature Value: Fixed a critical bug affecting request processing, enhancing system stability.

### Fix URL Encoding Issue for Model Names in Bedrock Requests
+ Related PR: [https://github.com/alibaba/higress/pull/2321](https://github.com/alibaba/higress/pull/2321)
+ Contributor: [HecarimV](https://github.com/HecarimV)
+ Change Log: Fixed the URL encoding issue for model names in Bedrock requests, preventing request failures due to special characters and removing redundant encoding functions.
+ Feature Value: Resolved the issue of request failures due to special characters in model names, enhancing system stability.

### Fix Error When Vector Provider is Not Configured
+ Related PR: [https://github.com/alibaba/higress/pull/2351](https://github.com/alibaba/higress/pull/2351)
+ Contributor: [mirror58229](https://github.com/mirror58229)
+ Change Log: Fixed the issue where 'EnableSemanticCachefalse' was incorrectly set when the vector provider was not configured, preventing errors in the 'handleResponse' function.
+ Feature Value: This PR fixed a bug that could cause error logs, enhancing system stability and user experience.

### Fix Nacos 3 MCP Server Rewrite Configuration Error
+ Related PR: [https://github.com/alibaba/higress/pull/2211](https://github.com/alibaba/higress/pull/2211)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Fixed the rewrite configuration error generated by the Nacos 3 MCP server, ensuring correct traffic routing.
+ Feature Value: Corrected the rewrite configuration of the MCP server to avoid service unavailability due to configuration errors.

### Fix Content-Length Request Header Issue in ai-search Plugin
+ Related PR: [https://github.com/alibaba/higress/pull/2363](https://github.com/alibaba/higress/pull/2363)
+ Contributor: [johnlanni](https://github.com/johnlanni)
+ Change Log: Fixed the issue where the Content-Length request header was not correctly removed in the ai-search plugin, ensuring the integrity of request header processing logic.
+ Feature Value: Fixed the issue of the Content-Length request header not being removed in the ai-search plugin, enhancing the plugin's stability and compatibility.

### Fix Authorization Header Issue in Gemini Proxy Requests
+ Related PR: [https://github.com/alibaba/higress/pull/2220](https://github.com/alibaba/higress/pull/2220)
+ Contributor: [hanxiantao](https://github.com/hanxiantao)
+ Change Log: Fixed the issue where the Authorization request header was incorrectly included in Gemini proxy requests, ensuring that the proxy requests meet Gemini API requirements.
+ Feature Value: Removed the Authorization header from Gemini proxy requests, resolving API call failures.

### Fix ToolArgs Struct Type Definition Issue
+ Related PR: [https://github.com/alibaba/higress/pull/2231](https://github.com/alibaba/higress/pull/2231)
+ Contributor: [Erica177](https://github.com/Erica177)
+ Change Log: Fixed issue #2222 by changing the Items field in the ToolArgs struct from []interface{} to interface{}, to accommodate specific use cases.
+ Feature Value: Fixed a type definition issue, enhancing code flexibility and compatibility.

## 📌refactor

### Refactor MCP Server Configuration Generation Logic
+ Related PR: [https://github.com/alibaba/higress/pull/2207](https://github.com/alibaba/higress/pull/2207)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Refactored the mcpServer.matchList configuration generation logic to support discovering mcp-sse type MCP servers from Nacos 3.x and fixed the ServiceKey issue in DestinationRules.
+ Feature Value: Improved MCP server configuration management, enhanced support for Nacos 3.x, and resolved routing issues for multiple MCP servers.

### Refactor MCP Server Auto-Discovery Logic
+ Related PR: [https://github.com/alibaba/higress/pull/2382](https://github.com/alibaba/higress/pull/2382)
+ Contributor: [Erica177](https://github.com/Erica177)
+ Change Log: Refactored the auto-discovery logic for MCP servers and fixed some issues, improving code maintainability and extensibility.
+ Feature Value: Enhanced the stability and extensibility of the system by refactoring and optimizing the auto-discovery logic for MCP servers, while also fixing some potential issues.

## 📌doc
### Optimize README.md Translation Process
+ Related PR: [https://github.com/alibaba/higress/pull/2208](https://github.com/alibaba/higress/pull/2208)
+ Contributor: [littlejiancc](https://github.com/littlejiancc)
+ Change Log: Optimized the translation process for README.md, supporting streaming transmission and avoiding duplicate PRs, enhancing the maintenance efficiency of multilingual documentation.
+ Feature Value: Improved the automated translation process to ensure document consistency and reduce manual intervention.

### Automated Translation Workflow
+ Related PR: [https://github.com/alibaba/higress/pull/2228](https://github.com/alibaba/higress/pull/2228)
+ Contributor: [MAVRICK-1](https://github.com/MAVRICK-1)
+ Change Log: This PR adds a GitHub Actions workflow for automatically translating non-English issues, PRs, and discussion content, enhancing the internationalization and accessibility of Higress.
+ Feature Value: Enhances the friendliness of Higress to international users and contributors through automated translation, strengthening the project's global reach.

# Higress Console

## 📌feature
### Support for Configuring Multiple Custom OpenAI LLM Provider Endpoints
+ Related PR: [https://github.com/higress-group/higress-console/pull/517](https://github.com/higress-group/higress-console/pull/517)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: This PR supports configuring multiple endpoints for custom OpenAI LLM providers, enhancing system flexibility and scalability. The LLM provider endpoint management logic was refactored to support IP+port format URLs and ensure all URLs have the same protocol and path.
+ Feature Value: This PR enables the system to support multiple custom OpenAI service endpoints, enhancing flexibility and reliability, suitable for multi-instance or load-balancing scenarios.

### Migration of Custom Image URL Patterns and Introduction of Wasm Plugin Service Configuration Class
+ Related PR: [https://github.com/higress-group/higress-console/pull/504](https://github.com/higress-group/higress-console/pull/504)
+ Contributor: [Thomas-Eliot](https://github.com/Thomas-Eliot)
+ Change Log: Migrated custom image URL patterns from the SDK module to the console module and introduced a Wasm plugin service configuration class to support more flexible Wasm plugin management.
+ Feature Value: This PR refactors the configuration management logic, enhancing the system's configurability and extensibility for Wasm plugins and laying the groundwork for future enhancements.

### New Configuration Parameter dependControllerApi
+ Related PR: [https://github.com/higress-group/higress-console/pull/506](https://github.com/higress-group/higress-console/pull/506)
+ Contributor: [Thomas-Eliot](https://github.com/Thomas-Eliot)
+ Change Log: Added a new configuration parameter dependControllerApi, supporting decoupling from the Higress Controller when not using a registry, enhancing architectural flexibility and configurability.
+ Feature Value: This PR introduces a new configuration option, allowing the system to bypass the registry and directly interact with the K8s API in specific scenarios, enhancing system flexibility and adaptability.

### Update Nacos3 Service Source Form to Support Nacos 3.0.1+
+ Related PR: [https://github.com/higress-group/higress-console/pull/521](https://github.com/higress-group/higress-console/pull/521)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Updated the Nacos3 service source form to support Nacos 3.0.1+ and fixed the issue where an error was displayed when creating a new source after deleting one.
+ Feature Value: This PR optimizes the service source configuration interface, enhancing support for Nacos 3.0.1+ and improving the user experience.

### Improve K8s Capability Initialization Logic
+ Related PR: [https://github.com/higress-group/higress-console/pull/513](https://github.com/higress-group/higress-console/pull/513)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Improved the K8s capability initialization logic by adding a retry mechanism and default support for Ingress V1 in case of failure, enhancing system stability and fault tolerance.
+ Feature Value: Fixed the unstable K8s capability detection issue, ensuring the console runs normally and improving the user experience.

### Support JDK 8
+ Related PR: [https://github.com/higress-group/higress-console/pull/497](https://github.com/higress-group/higress-console/pull/497)
+ Contributor: [Thomas-Eliot](https://github.com/Thomas-Eliot)
+ Change Log: Fixed compatibility issues caused by using Java 11 features, making the project compatible with JDK 8. Mainly modified the code using String.repeat() and List.of() methods.
+ Feature Value: This PR resolves the project's JDK 8 compatibility issues, allowing the project to run in a JDK 8 environment.

### Add Security Tips in Certificate Edit Form
+ Related PR: [https://github.com/higress-group/higress-console/pull/512](https://github.com/higress-group/higress-console/pull/512)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Added a security tip in the certificate edit form, clearly informing users that the current certificate and private key data will not be displayed and guiding them to directly enter new data.
+ Feature Value: Provides clearer operational guidance to users, enhancing data security awareness and preventing misoperations.

### Update Display Name for OpenAI Provider Type
+ Related PR: [https://github.com/higress-group/higress-console/pull/510](https://github.com/higress-group/higress-console/pull/510)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Updated the display name for the OpenAI provider type to more clearly indicate its compatibility, enhancing user recognition of the service.
+ Feature Value: Modified the display name of the OpenAI provider, making it easier for users to distinguish between service types and improving the user experience.

## 📌bugfix
### Fix Bug Where Case-Insensitive Path Matching Could Not Be Enabled in AI Routing
+ Related PR: [https://github.com/higress-group/higress-console/pull/508](https://github.com/higress-group/higress-console/pull/508)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Fixed the bug where case-insensitive path matching could not be enabled in AI routing by modifying the path predicate handling logic and adding a normalization function to ensure correct functionality.
+ Feature Value: Fixed the issue of case-insensitive path matching in AI routing configuration, enhancing the flexibility of routing rules and user experience.

### Fix Multiple Issues in higress-config Update Functionality
+ Related PR: [https://github.com/higress-group/higress-console/pull/509](https://github.com/higress-group/higress-console/pull/509)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Fixed multiple issues in the higress-config update functionality, including changing the HTTP method from POST to PUT, adding success prompt messages, and correcting method name spelling errors.
+ Feature Value: Fixed the API call method and prompt logic in the configuration update, enhancing the user experience and system stability.

### Fix Text Display Error in Frontend Pages
+ Related PR: [https://github.com/higress-group/higress-console/pull/503](https://github.com/higress-group/higress-console/pull/503)
+ Contributor: [CH3CHO](https://github.com/CH3CHO)
+ Change Log: Fixed a text display error in the frontend pages, correcting the incorrect text content to an accurate description.
+ Feature Value: Corrected the text content in the interface, enhancing the user's understanding and experience of the feature.

## 📌refactor
### Optimize Pagination Tool Logic
+ Related PR: [https://github.com/higress-group/higress-console/pull/499](https://github.com/higress-group/higress-console/pull/499)
+ Contributor: [Thomas-Eliot](https://github.com/Thomas-Eliot)
+ Change Log: Optimized the pagination tool logic by introducing more efficient collection processing and simplifying the code structure, enhancing the performance and maintainability of the pagination function.
+ Feature Value: Improved the implementation of the pagination tool, increasing data processing efficiency and code readability, positively impacting system performance.
