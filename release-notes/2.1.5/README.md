# Higress


## 📋 Overview of This Release

This release includes **41** updates covering feature enhancements, bug fixes, performance optimizations, and more.

### Update Distribution

- **New Features**: 20 items
- **Bug Fixes**: 15 items
- **Refactoring & Optimization**: 2 items
- **Documentation Updates**: 4 items

### ⭐ Highlights

This release includes **9** significant updates, which are recommended for special attention:

- **fix: The mcp to rest capability of the mcp server supports returning status without returning a body from the backend, and instead responds via sse** ([#2445](https://github.com/alibaba/higress/pull/2445)): Improves the stability and compatibility of the MCP server when handling specific REST requests, avoiding exceptions caused by backend responses without bodies, and ensuring consistent user experience.
- **feat(mcp/sse): support passthrough the query parameter in sse server to the rest api server** ([#2460](https://github.com/alibaba/higress/pull/2460)): Enhances the flexibility of SSE functionality by enabling query parameters to be correctly passed, improving system compatibility and user experience.
- **fix too much logs when nacos is not available** ([#2469](https://github.com/alibaba/higress/pull/2469)): Enhances system stability, avoiding program crashes due to excessive logging, reducing unnecessary log output, and improving log readability and system performance.
- **feat: support for wanxiang image/video generation in ai-proxy & ai-statistics** ([#2378](https://github.com/alibaba/higress/pull/2378)): Supports asynchronous image/video generation, enhancing AI service capabilities; configuration optimizations prevent log statistics errors, improving system compatibility and stability.
- **feat: add DB MCP Server execute, list tables, describe table tools** ([#2506](https://github.com/alibaba/higress/pull/2506)): Users can more conveniently execute SQL statements, list table names, and describe table structures, improving database management and debugging efficiency.
- **fix(ai-proxy): fix bedrock Sigv4 mismatch** ([#2402](https://github.com/alibaba/higress/pull/2402)): Ensures the AWS SigV4 signature mechanism functions correctly, preventing API call failures due to modelId decoding errors, and improving system stability and security.
- **feat: add mcp-router plugin** ([#2409](https://github.com/alibaba/higress/pull/2409)): Provides a unified gateway aggregation capability, allowing multiple MCP backend services to be accessed through a single entry point, simplifying client configuration and enhancing service integration and expansion flexibility.
- **feat(ai-proxy): add support for OpenAI Fine-Tuning API** ([#2424](https://github.com/alibaba/higress/pull/2424)): Users can now utilize OpenAI's fine-tuning API functionality, enabling more flexible model customization and improved model performance on specific tasks.
- **feat: add default route support for wanx image&video synthesis** ([#2431](https://github.com/alibaba/higress/pull/2431)): Enhances routing functionality, allowing users to more efficiently call Wanx image and video synthesis interfaces, improving system flexibility and usability.

For detailed information, please see the Important Features section below.

---

## 🌟 Detailed Description of Important Features

Below is a detailed explanation of the important features and improvements in this release:

### 1. fix: The mcp to rest capability of the mcp server supports returning status without returning a body from the backend, and instead responds via sse

**Related PR**: [#2445](https://github.com/alibaba/higress/pull/2445) | **Contributor**: [johnlanni](https://github.com/johnlanni)

**Use Background**

In microservices and cloud-native architectures, the MCP (Mesh Configuration Protocol) server is responsible for delivering configuration information to downstream services via REST interfaces. In some cases, the backend might only need to return HTTP status codes without a response body. However, the previous implementation required a response body, leading to resource waste or incompatibility with certain downstream clients. Additionally, the traditional HTTP response model has limitations in real-time and streaming data transmission, limiting the MCP server’s performance in high-concurrency, low-latency scenarios. This fix addresses the need for a more flexible response mechanism for REST interfaces. Target users include developers and operations engineers integrating control planes based on MCP.

**Feature Details**

This PR refactors the `makeHttpResponse` function in the MCP utility library, enabling HTTP responses to return only status codes without including a response body, and integrates SSE (Server-Sent Events) mechanisms to enable streaming responses. This improvement makes the MCP server more flexible in handling client requests, especially in scenarios involving status feedback or event notifications. Technically, the forced requirement for response bodies in responses is removed, while the `sendDirectly` parameter is removed from callback functions, indicating that the response logic has been unified to be event-driven. Code changes affect multiple MCP utility modules, including updates to dependency versions in go.mod and go.sum to ensure compatibility and stability.

**Usage**

There is no additional configuration required to enable this feature; the MCP server defaults to supporting returning status codes without response bodies and responding via SSE. Developers implementing custom MCP tools should ensure that callback functions follow the new signature format, i.e., no longer including the `sendDirectly` parameter. Typical use cases include status check interfaces, asynchronous task notifications, and event stream pushing. Best practices recommend standardizing the returned HTTP status codes and combining them with SSE client event monitoring mechanisms to enhance response performance and user experience.

**Feature Value**

This fix significantly improves the flexibility and performance of the MCP server in handling REST requests. By supporting no-body responses and the SSE mechanism, it simplifies the status feedback process, reduces unnecessary network overhead, and improves system throughput. At the same time, this improvement enhances the compatibility of the MCP server with streaming clients, making it more suitable for real-time control, asynchronous notifications, and event-driven architecture scenarios. Furthermore, the unified response mechanism lowers the complexity of implementing and maintaining MCP tools for developers, enhancing the overall ecosystem's scalability and stability.

---

### 2. feat(mcp/sse): support passthrough the query parameter in sse server to the rest api server 

**Related PR**: [#2460](https://github.com/alibaba/higress/pull/2460) | **Contributor**: [erasernoob](https://github.com/erasernoob)

**Use Background**

In real-time message pushing scenarios based on SSE (Server-Sent Events), the frontend typically passes context information via query parameters, such as user identifiers or session status. However, in the original design, these parameters could not be effectively passed to the backend REST API server, resulting in the loss of context information and affecting backend processing logic. This feature’s introduction addresses this issue, enabling query parameters in SSE connections to be correctly forwarded to the message processing interface, thereby achieving a more complete request context transfer. Target users include developers using Higress for SSE proxying and frontend and backend services that need to carry session state via query parameters.

**Feature Details**

This PR primarily implements the pass-through functionality of query parameters during SSE connection establishment. Specifically, in `filter.go`, query parameters are extracted from the original request URL and appended to the constructed `messageEndpoint` address. In `sse.go`, the `net/url` package is used in a safer way to concatenate URLs and parameters, avoiding formatting errors from manual concatenation. Compared to the previous hard-coded approach, this improvement enhances support for and reliability of parameter handling. Key technical points include using `url.Parse` and `Query()` methods to parse and concatenate URL parameters, and preserving original query parameters when constructing `messageEndpoint`. This feature enhances the capabilities of the SSE proxy, seamlessly integrating with the original logic and improving system extensibility.

**Usage**

This feature is enabled by default with no additional configuration required. When using SSE proxying, simply include query parameters in the URL when the frontend initiates an SSE connection (e.g., `/sse?userId=123`), and these parameters will automatically be passed through to the corresponding REST API message processing interface. Typical use cases include user identity identification, session control, or dynamic routing based on query parameters. It is recommended to ensure the legitimacy and security of parameters when using them to avoid injection attacks. Best practices involve combining authentication mechanisms and validating the validity of key parameters on the backend.

**Feature Value**

This feature significantly improves the flexibility and practicality of the SSE proxy, enabling backend services to process based on a complete request context. It enhances the usability of SSE connections and supports richer business scenarios, such as personalized message pushing and multi-tenant session management. At the system level, it improves the general-purpose nature and compatibility of the proxy service, lowering the coupling between the frontend and gateway. At the ecosystem level, this improvement makes Higress more competitive in supporting real-time communication scenarios (such as chat, notifications, data dashboards), providing a solid foundation for building richer microservice integration solutions.

---

### 3. fix too much logs when nacos is not available

**Related PR**: [#2469](https://github.com/alibaba/higress/pull/2469) | **Contributor**: [luoxiner](https://github.com/luoxiner)

**Use Background**

In the MCP service discovery module, when the registration center Nacos is unavailable, the system attempts to continuously pull configuration information to maintain service discovery capabilities. At this time, if Nacos cannot respond normally, it triggers logging logic. Due to missing log parameters in the original code, not only are key error messages missing from the logs, but runtime panic may also be triggered. Additionally, frequent log output increases system I/O pressure, affecting overall observability and performance. Target users are primarily system operations engineers and developers using the MCP architecture and integrating Nacos as a service discovery mechanism.

**Feature Details**

This PR mainly fixes two issues: first, the format string and parameter count in the log recording call in client.go do not match—the `err` parameter is missing on line 144, and an extra %v is present on line 149 without a corresponding parameter, easily leading to runtime panic; second, the log output logic is optimized so that when an error or empty result occurs, it no longer continues retrying but chooses to exit the loop, avoiding log storms. Additionally, watcher.go adds default parameters for log rolling configuration, including the maximum size of a single log file (64MB) and the maximum number of backups (3), improving log management controllability. The fix is achieved by modifying the log call method and adjusting the log level control strategy, being compatible with existing functions and requiring no additional configuration.

**Usage**

The fixes in this PR take effect automatically without manual activation or configuration. Typical use scenarios include the startup of MCP services, switching during Nacos service anomalies, or service discovery processes during network interruptions. Users only need to deploy the MCP service normally and integrate it with Nacos. When Nacos is unavailable, the system will significantly reduce unnecessary log output and avoid program crashes caused by log formatting errors. Best practices involve observing log volume changes using log monitoring systems to ensure the system has sufficient log fault tolerance. Note: If custom log paths or log levels are defined, ensure compatibility with the logrolling configuration.

**Feature Value**

This fix enhances system robustness and reliability in log processing, effectively preventing log storms and panic issues when Nacos is abnormal, thereby enhancing the stability of the service discovery mechanism. By introducing log rolling configuration parameters, it also enhances control over log storage space and lifecycle, reducing operational costs. Additionally, this fix helps improve the reliability of the MCP architecture in complex network environments, further strengthening the robustness of service registration and discovery modules, which has a positive impact on the stability of the entire ecosystem.

---

### 4. feat: support for wanxiang image/video generation in ai-proxy & ai-statistics

**Related PR**: [#2378](https://github.com/alibaba/higress/pull/2378) | **Contributor**: [mirror58229](https://github.com/mirror58229)

**Use Background**

This PR addresses incomplete support for proxying and logging statistics for asynchronous generation tasks when using the WanXiang (WanXiang) AIGC service. The text-to-image or video generation provided by WanXiang has a relatively long processing latency, thus requiring the ability to submit and query task status asynchronously. Furthermore, WanXiang's API protocol differs from the OpenAI standard, leading to the failure to extract fields like model and token in logs, affecting monitoring and statistics capabilities. Target users are primarily AI platform administrators and developers using the Alibaba Cloud WanXiang service, who require stable and scalable AIGC invocation and observability capabilities.

**Feature Details**

This PR adds support for identifying and routing WanXiang's asynchronous AIGC interfaces in the ai-proxy, including two API paths: `/api/v1/services/aigc` for task submission and `/api/v1/tasks` for querying task status. At the same time, in the ai-statistics module, a new `disable_openai_usage` configuration item is added to disable the OpenAI-compatible format log field extraction logic to avoid errors caused by WanXiang's non-standard response structure. The code implements this through new routing matching rules and configuration logic, ensuring that existing OpenAI-compatible service configurations are not affected, while enhancing compatibility and observability for WanXiang services.

**Usage**

When configuring ai-proxy, map the WanXiang service's API paths to `/api/v1/services/aigc` (generation request) and `/api/v1/tasks` (status query). In the ai-statistics plugin configuration, if using non-OpenAI protocols (such as WanXiang), set `disable_openai_usage: true` to avoid log parsing errors. A typical usage workflow includes: submitting text-to-image requests, asynchronously polling task status, and recording task completion. Best practices include properly configuring log fields, monitoring task latency, and ensuring asynchronous path configurations are accurate to avoid path matching errors affecting services.

**Feature Value**

This feature enhances the platform's support for asynchronous AIGC generation tasks, strengthens compatibility with non-standard protocol services (like WanXiang), avoids log statistics errors, and improves observability. At the same time, through structured configuration support, it simplifies platform management and monitoring, improving service stability and maintainability. For the AI platform ecosystem, the added WanXiang interface support helps expand AIGC application scenarios and enhances the platform’s unified access capability and observability level for image and video generation services.

---

### 5. feat: add DB MCP Server execute, list tables, describe table tools

**Related PR**: [#2506](https://github.com/alibaba/higress/pull/2506) | **Contributor**: [hongzhouzi](https://github.com/hongzhouzi)

**Use Background**

With the widespread application of MCP (Model Control Protocol) services in database connection and interaction scenarios, users have higher demands for immediate database control capabilities. The original implementation only supported read-only SQL queries, lacking the ability to execute change statements or obtain table structure information, leading to functional limitations in complex scenarios. For example, developers need to view database table structures during debugging or deployment phases, and operations personnel need to execute change SQLs in batches—these scenarios cannot be supported by existing functions. Target users mainly include database developers, AI application integrators, and system operations personnel who need to implement more flexible and comprehensive database operations through MCP services.

**Feature Details**

This PR adds three core tools: execute (execute SQL statements, such as INSERT/UPDATE/DELETE), list tables (list all table names), and describe table (obtain field structure for a specified table). Technically, database type constants (such as MYSQL, POSTGRES, etc.) are introduced in db.go to improve maintainability; in tools.go, handler functions for the four tools are implemented separately, such as HandleExecuteTool for handling write operations, and HandleListTablesTool and HandleDescribeTableTool call underlying GORM capabilities to obtain table information and structure. In server.go, the newly added tools are registered to the MCP Server through the AddTool method, and description information formats are unified to improve consistency. Community feedback pointed out potential SQL injection risks and repetitive error handling issues, suggesting optimization in future versions.

**Usage**

To enable these functions, correctly set database connection information (DSN) and type (e.g., mysql, postgres) in the MCP Server configuration file. Users can call the added tools via the MCP client:
- execute: pass in SQL statement parameters to perform INSERT/UPDATE/DELETE operations;
- list tables: no parameters required, directly call to return all table names;
- describe table: pass in the table name parameter to obtain the table's fields, types, constraints, etc.
Typical usage scenarios include: automatically executing initialization SQL in deployment scripts, viewing table structures through a UI interface, and checking database status during the debugging phase. When using, attention should be paid to permission control to avoid unauthorized SQL execution; parameterized queries are also recommended to prevent SQL injection.

**Feature Value**

This feature enhancement significantly improves the MCP Server's database interaction capabilities, enabling it to meet more complex application requirements. The execute tool supports write operations,弥补ing the limitations of the original read-only queries; the list tables and describe table tools enhance awareness of the database structure, providing foundational support for automated operations and visualization interfaces. From a system perspective, these tools improve the convenience of database debugging and management, reduce manual intervention, and enhance overall stability. Additionally, defining database types as constants improves code readability and maintainability. Although there are still issues like SQL injection and repetitive error handling, a foundation has been laid for future optimization, making this an important step forward in the evolution of the MCP Server ecosystem.

---

### 6. fix(ai-proxy): fix bedrock Sigv4 mismatch

**Related PR**: [#2402](https://github.com/alibaba/higress/pull/2402) | **Contributor**: [HecarimV](https://github.com/HecarimV)

**Use Background**

AWS Bedrock is Amazon's fully managed foundational model service, allowing users to access multiple models through a unified interface. When using this service, requests must be authenticated through the Sigv4 signing mechanism. However, the previous implementation's URL path encoding did not conform to AWS IAM documentation regarding Sigv4 encoding specifications, leading to signature verification failures, manifested as 403 Forbidden or other authentication errors. This issue affected users of AI Proxy proxying AWS Bedrock services, especially in scenarios requiring path parameters. Target users mainly include developers and system architects using AI Proxy as the AWS Bedrock service frontend, aiming to unify the API proxy layer and achieve protocol compatibility.

**Feature Details**

This PR primarily fixes the encoding issue of the path portion (Canonical URI) in the Sigv4 signature. According to AWS IAM documentation specifications, the path portion should preserve slashes `/` and encode other special characters in RFC 3986-compatible format, with encoded characters in uppercase. The code introduces a new `encodeSigV4Path` function to process the path by segments and uses `url.PathEscape` to implement correct encoding. Additionally, the decoding logic for `modelId` was fixed to avoid potential data contamination risks. This change ensures the correctness of the signature, solving the authentication mismatch issue raised in #2396. Compared to the original implementation, it enhances compatibility with AWS services and reduces invalid requests caused by signature errors.

**Usage**

To use this feature, set the type to `bedrock` in the AI Proxy configuration and provide `awsAccessKey`, `awsSecretKey`, and `awsRegion` parameters. For example:

```yaml
provider:
  type: bedrock
  awsAccessKey: "YOUR_AWS_ACCESS_KEY_ID"
  awsSecretKey: "YOUR_AWS_SECRET_ACCESS_KEY"
  awsRegion: "YOUR_AWS_REGION"
```

Typical usage scenarios include proxying AWS Bedrock services via the OpenAI protocol to achieve unified API interface access. Request examples can use standard OpenAI formats to invoke Bedrock models. When using, ensure AWS credentials have permissions to access target models and correctly configure region information. Best practices include regularly rotating keys, using IAM roles to manage access permissions, and enhancing security with VPC and other mechanisms.

**Feature Value**

This fix significantly improves the AI Proxy's integration stability with AWS Bedrock services, solving common Sigv4 signature mismatch issues, allowing users to more reliably use a unified proxy layer to access AWS model services. It also enhances system security and compatibility, reducing the risk of service unavailability caused by signature errors. For scenarios requiring unified management and protocol compatibility of multiple model backend, this improvement holds significant value, enhancing the practicality and reliability of AI Proxy in enterprise-level AI service gateway scenarios.

---

### 7. feat: add mcp-router plugin

**Related PR**: [#2409](https://github.com/alibaba/higress/pull/2409) | **Contributor**: [johnlanni](https://github.com/johnlanni)

**Use Background**

In the current MCP architecture, an MCP Server typically corresponds to only one backend service instance, and this one-to-one relationship limits the ability to create a unified MCP endpoint. For AI proxies that need to integrate multiple backend tools, this limitation causes clients to manage multiple MCP endpoints, increasing complexity and maintenance costs. Therefore, a dynamic routing mechanism is needed to allow an MCP gateway to route tool calls to different backend services based on request content. The mcp-router plugin was designed to solve this problem, enabling clients to call different services via a single entry point.

**Feature Details**

The mcp-router plugin parses the tool name in the tools/call request to determine whether it needs to be routed to a specific backend MCP server. If the tool name has a prefix (e.g., server-name/tool-name), the plugin will route the request to the corresponding backend server based on the configured routing rules. The plugin is written in Wasm-Go and implements dynamic routing by modifying request headers and bodies. Technically, it leverages the method name parsing in the JSON-RPC protocol and the secondary processing mechanism of the gateway routing engine to dynamically modify the target domain name and path of the request, achieving seamless integration of multiple backend services.

**Usage**

To use the mcp-router plugin, enable the plugin in the Higress route configuration and configure the server routing rules. Configuration requires providing the name, domain, and path of each backend MCP server. For example: define a list of servers in higress-plugins.yaml including the server name, domain, and path. When the client initiates a tools/call request, simply add the server prefix to the tool name (e.g., `server-name/tool-name`), and the plugin will automatically route the request to the specified backend server. Best practices include ensuring that the name field matches the server configuration and properly configuring domain and path to avoid routing errors.

**Feature Value**

The mcp-router plugin provides users with a unified entry point for calling MCP tools, simplifying client management of multiple backend services. Through dynamic routing, users do not need to maintain multiple endpoints, improving the system's scalability and ease of use. In addition, the plugin enhances the capabilities of Higress as an MCP gateway, making it more flexible to support tool combinations and complex service architectures, providing stronger backend support for AI proxies. The plugin's design also offers good maintainability, facilitating future extensions of more routing strategies and configuration management enhancements.

---

### 8. feat(ai-proxy): add support for OpenAI Fine-Tuning API

**Related PR**: [#2424](https://github.com/alibaba/higress/pull/2424) | **Contributor**: [wydream](https://github.com/wydream)

**Use Background**

OpenAI's Fine-Tuning API allows users to fine-tune foundational models based on specific datasets to generate customized models with vertical domain capabilities. As demand for customized large models grows, AI proxy platforms need to provide end-to-end support for fine-tuning tasks, including task creation, status monitoring, event log retrieval, checkpoint management, and more. This feature is primarily aimed at AI engineers and data scientists who need to optimize and customize models, helping them efficiently manage the fine-tuning process within a unified proxy layer rather than directly interfacing with the underlying API. Target users include enterprise AI application developers, model training teams, and MLOps operations personnel.

**Feature Details**

This PR adds complete routing mapping and default capability configuration for the OpenAI Fine-Tuning API in the ai-proxy. Specific implementations include: adding routing recognition logic for multiple Fine-Tuning-related paths in main.go, such as creating tasks, retrieving events, canceling tasks, pausing tasks, and managing checkpoint permissions; updating the default capability mapping table in provider/openai.go to map new API name constants to actual paths; defining multiple new ApiName constants and path constants in provider/provider.go to ensure semantic consistency; and introducing multiple regular expressions in util/http.go for path matching and parameter extraction, such as extracting fine-tuning task IDs, checkpoint IDs, and permission IDs. Code changes achieve precise recognition of complex path structures through structured mapping and regex matching mechanisms. Although community feedback points out that some constant names have spelling or semantic inconsistencies, the overall implementation meets basic functional requirements.

**Usage**

This feature is integrated by default in the OpenAI proxy module with no additional activation required. Users can access the following Fine-Tuning API paths through the proxy layer: create a fine-tuning job (POST /v1/fine_tuning/jobs), list jobs (GET /v1/fine_tuning/jobs), get job details (GET /v1/fine_tuning/jobs/{job_id}), retrieve event logs (GET /v1/fine_tuning/jobs/{job_id}/events), get checkpoints (GET /v1/fine_tuning/jobs/{job_id}/checkpoints), and cancel a job (POST /v1/fine_tuning/jobs/{job_id}/cancel). Typical usage scenarios include internal enterprise model fine-tuning service management, training log monitoring, checkpoint permission configuration, and task pause/resume. When using, ensure the correct job ID is passed, pay attention to strict format requirements in regex path matches, and consider access control policies when using permission management interfaces.

**Feature Value**

This feature enhances the AI proxy platform's model training support capabilities, allowing users to manage the full lifecycle of fine-tuning tasks within a unified proxy layer. By integrating the Fine-Tuning API, users can achieve task automation orchestration, status monitoring, and log analysis without directly interfacing with the underlying OpenAI service, improving development and operations efficiency. At the same time, support for checkpoint permission management and task control operations (such as pausing, resuming, and canceling) enhances security control over sensitive model training processes. For the platform ecosystem, this feature expands the AI proxy's application boundaries in model customization scenarios, helping to build a more complete AI engineering toolchain and enhancing platform competitiveness and user stickiness.

---

### 9. feat: add default route support for wanx image&video synthesis

**Related PR**: [#2431](https://github.com/alibaba/higress/pull/2431) | **Contributor**: [mirror58229](https://github.com/mirror58229)

**Use Background**

As AI platforms continue to evolve, they gradually support multiple generative AI capabilities, including text generation, image generation, speech synthesis, and video generation. model-mapper and model-router are key plugins in the AI proxy gateway, used to make routing decisions based on model parameters in requests. Before this PR, these two plugins did not support image synthesis (image-synthesis) and video synthesis (video-synthesis) interfaces, resulting in a lack of routing mapping capabilities when processing such requests, preventing them from being correctly directed to backend services. Therefore, the introduction of this feature addresses the lack of support for new AI generation capabilities. Target users are primarily AI service gateway operations personnel and developers needing to integrate image/video generation capabilities.

**Feature Details**

This PR mainly extends the support of the model-mapper and model-router plugins for WanX image synthesis and video synthesis interfaces. Specifically, in the default values of the enableOnPathSuffix configuration item in both plugins, new path suffixes /image-synthesis and /video-synthesis were added to ensure that requests to these interfaces can be correctly identified and undergo model parameter parsing and routing decisions. This implementation is completed by modifying the default configuration arrays in the plugin source code and configuration descriptions in the documentation, ensuring consistency of functionality and up-to-date documentation. Compared to existing functionality, this extension does not introduce new configuration fields but expands based on the existing enableOnPathSuffix mechanism, maintaining configuration logic consistency and maintainability.

**Usage**

No additional configuration changes are required to enable this feature; the system defaults to including requests with paths /image-synthesis and /video-synthesis in the model mapping and routing process. Users can continue to use existing modelMapping configuration rules to define specific model mapping strategies for image and video generation interfaces. Typical usage scenarios include: when the AI gateway needs to proxy image/video generation APIs from platforms like WanX, model-mapper automatically identifies request model parameters and completes the mapping, then model-router routes the request to the corresponding backend service. Best practices recommend users to regularly check and customize the enableOnPathSuffix configuration according to actual needs, ensuring interface security and flexibility.

**Feature Value**

This feature significantly enhances the AI gateway's support for multimedia generation capabilities, making the platform more compatible and extensible when integrating new AI services like image synthesis and video synthesis. Through automated model parameter identification and routing distribution, it reduces the configuration complexity required to integrate new models, improving overall development and operations efficiency. At the same time, this feature further improves the applicable scenarios of plugins, allowing them to cover a wider range of service types in the AI proxy gateway. Additionally, by updating documentation and default configurations, it enhances system usability and maintainability, laying a solid foundation for future AI functionality expansion.

---

## 📝 Complete Change Log

### 🚀 New Features (Features)

- **Related PR**: [#2536](https://github.com/alibaba/higress/pull/2536)
  **Contributor**: johnlanni
  **Change Log**: This PR primarily completed the version upgrade from 2.1.5-rc.1 to 2.1.5, involving updates to Makefile, VERSION, and Helm Chart-related files, marking the official release of version 2.1.5.
  **Feature Value**: This version release provides the latest stable version for user deployment and use, ensuring users can access the latest enhancements and issue fixes, improving product experience and stability.

- **Related PR**: [#2533](https://github.com/alibaba/higress/pull/2533)
  **Contributor**: johnlanni
  **Change Log**: Added support for subPath field configuration in ai-proxy, improving path handling flexibility, and synchronizing updates to Chinese and English documentation descriptions, enhancing feature usability.
  **Feature Value**: Users can optimize request path handling logic by configuring subPath prefixes, enhancing the AI proxy plugin's adaptability to complex routing scenarios.

- **Related PR**: [#2531](https://github.com/alibaba/higress/pull/2531)
  **Contributor**: rinfx
  **Change Log**: Added three load balancing strategies for LLM services: least load, Redis-based global least request, and prompt prefix matching strategy, implemented through WASM plugins.
  **Feature Value**: Provides more intelligent load balancing options for LLM services, improving system resource utilization and response efficiency, while supporting KV Cache reuse, optimizing inference performance.

- **Related PR**: [#2516](https://github.com/alibaba/higress/pull/2516)
  **Contributor**: HecarimV
  **Change Log**: This PR adds system message handling functionality for Bedrock API requests in the AI Proxy component, including adding a System field to the request payload structure and updating the request building logic to support conditional inclusion of system messages.
  **Feature Value**: This feature enables users to send system messages through the Bedrock API, providing more flexible control over conversation context and model behavior, improving the accuracy and practicality of AI interactions.

- **Related PR**: [#2509](https://github.com/alibaba/higress/pull/2509)
  **Contributor**: daixijun
  **Change Log**: Added handling for the OpenAI responses interface Body and added support for responses interfaces from the Doubao large model.
  **Feature Value**: Enhances AI proxy functionality, supports responses processing for Doubao models, and improves system extensibility and model adaptability.

- **Related PR**: [#2488](https://github.com/alibaba/higress/pull/2488)
  **Contributor**: rinfx
  **Change Log**: Added `trace_span_key` and `as_seperate_log_field` configuration items, used to differentiate log and Span attribute keys and control whether log fields are recorded separately.
  **Feature Value**: Enhances log and trace configuration flexibility, allowing users to more clearly manage and query log and distributed tracing data, enhancing system observability.

- **Related PR**: [#2485](https://github.com/alibaba/higress/pull/2485)
  **Contributor**: johnlanni
  **Change Log**: Added the `errorResponseTemplate` function to the mcp server plugin to customize error response templates when backend HTTP status codes are greater than 300.
  **Feature Value**: Improves user flexibility and experience in error handling through support for custom error response templates, enhancing system configurability and adaptability.

- **Related PR**: [#2450](https://github.com/alibaba/higress/pull/2450)
  **Contributor**: kenneth-bro
  **Change Log**: Added the Investoday MCP Server module market function, including real-time market data and constituent stock data for industries and concepts, covering key market indicators.
  **Feature Value**: Provides real-time and comprehensive industry and concept market data support for smart investment research and market hotspot tracking, enhancing user insight into market dynamics.

- **Related PR**: [#2446](https://github.com/alibaba/higress/pull/2446)
  **Contributor**: johnlanni
  **Change Log**: Updated the version number to v2.1.5-rc.1 and modified the application version information in the Helm Chart, starting to support the new release version.
  **Feature Value**: Provides users with access to the trial version, helping them promptly obtain the latest features and improvements, enhancing the user experience.

- **Related PR**: [#2404](https://github.com/alibaba/higress/pull/2404)
  **Contributor**: 007gzs
  **Change Log**: Added support for the reasoning_content field and returning multiple index groups, improving the flexibility and compatibility of AI data masking functions in streaming mode.
  **Feature Value**: Users can more efficiently handle multiple response data groups and are compatible with OpenAI's n parameter feature, enhancing system extensibility and ease of use.

- **Related PR**: [#2391](https://github.com/alibaba/higress/pull/2391)
  **Contributor**: daixijun
  **Change Log**: Adjusted the AI proxy's streaming response structure to output null when the usage, logprobs, and finish_reason fields are empty, maintaining consistency with the OpenAI official interface.
  **Feature Value**: Enhances system compatibility and consistency, allowing users to receive a unified response format when using different AI models, reducing the complexity of subsequent processing logic.

- **Related PR**: [#2389](https://github.com/alibaba/higress/pull/2389)
  **Contributor**: NorthernBob
  **Change Log**: Added support for one-click Kubernetes deployment to the plugin server, and configured default download URLs for plugins, involving the addition and adjustment of Helm Chart templates and service configurations.
  **Feature Value**: Simplifies the deployment process for Higress plugin servers in Kubernetes environments, enhancing user deployment efficiency and experience.

- **Related PR**: [#2343](https://github.com/alibaba/higress/pull/2343)
  **Contributor**: hourmoneys
  **Change Log**: Added an AI-based bidding information tool MCP service, including English and Chinese user documentation, configuration files, and providing a function to query bid information lists based on keywords.
  **Feature Value**: Helps enterprises quickly obtain accurate bid information, improving bidding efficiency and win rates, and optimizing bid information querying experiences.

- **Related PR**: [#1925](https://github.com/alibaba/higress/pull/1925)
  **Contributor**: kai2321
  **Change Log**: Implemented the AI-IMAGE-READER plugin, integrated with OCR services, supports the Qwen-VL model service from Alibaba Cloud, and added Chinese and English documentation and related configurations.
  **Feature Value**: Users can call OCR services through this plugin to extract text content from images, enhancing the AI gateway's ability to process image content and expanding platform functionality.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#2524](https://github.com/alibaba/higress/pull/2524)
  **Contributor**: daixijun
  **Change Log**: Fixed an issue where the `stream_options` parameter caused errors in non-OpenAI streaming interfaces by limiting its effectiveness to the openai/v1/chatcompletions interface.
  **Feature Value**: Prevents interface errors caused by incorrectly adding the `stream_options` parameter, improving system stability and accuracy in parameter handling.

- **Related PR**: [#2514](https://github.com/alibaba/higress/pull/2514)
  **Contributor**: daixijun
  **Change Log**: Commented out the default value of tracing.skywalking in values.yaml to avoid helm upgrade automatically adding skywalking configuration when configuring other tracing types, causing errors.
  **Feature Value**: Fixes errors caused by automatic injection of invalid tracing configurations during helm upgrades, enhancing configuration flexibility and user experience.

- **Related PR**: [#2497](https://github.com/alibaba/higress/pull/2497)
  **Contributor**: johnlanni
  **Change Log**: Fixed an issue where decoding behavior was incorrect when the configured URL path contained URL-encoded parts during request construction and sending.
  **Feature Value**: Solves request exception issues caused by URL path decoding errors, improving system stability and compatibility, ensuring correct processing of user requests.

- **Related PR**: [#2480](https://github.com/alibaba/higress/pull/2480)
  **Contributor**: HecarimV
  **Change Log**: Fixed a null pointer exception issue caused by uninitialized AdditionalModelRequestFields during AWS Bedrock request construction and improved the formatting of related documentation tables.
  **Feature Value**: Enhances the stability of AI proxy calls to AWS Bedrock services, ensuring user requests can correctly carry additional parameters and avoid runtime crashes.

- **Related PR**: [#2475](https://github.com/alibaba/higress/pull/2475)
  **Contributor**: daixijun
  **Change Log**: Fixed an issue where incorrect customPath transmission caused 404 errors when the openai provider was configured with openaiCustomUrl as a single interface.
  **Feature Value**: Enhances the openai provider's compatibility with different interface paths (e.g., Qwen's /compatible-mode/v1), ensuring requests are processed correctly.

- **Related PR**: [#2443](https://github.com/alibaba/higress/pull/2443)
  **Contributor**: Colstuwjx
  **Change Log**: Fixed the missing annotations configuration in the controller service account, allowing users to bind AWS IAM roles via annotations.
  **Feature Value**: Enhances the flexibility of the controller SA, allowing users to authenticate resource identities through AWS IAM, improving system integration capabilities.

- **Related PR**: [#2441](https://github.com/alibaba/higress/pull/2441)
  **Contributor**: wydream
  **Change Log**: Unified naming conventions for API name constants, fixed ApiName mapping issues, and corrected multiple typos to ensure correct API path matching.
  **Feature Value**: Enhances API calling accuracy and stability, avoiding 404 errors or feature failures due to path typos, improving user experience.

- **Related PR**: [#2440](https://github.com/alibaba/higress/pull/2440)
  **Contributor**: johnlanni
  **Change Log**: Fixed an issue where rds caching was ineffective when enabling consistent hashing in Istio, and fixed the envoy wasm abi acquisition interface.
  **Feature Value**: Enhances Istio's stability and functional availability in consistent hashing scenarios and improves envoy wasm functional compatibility.

- **Related PR**: [#2423](https://github.com/alibaba/higress/pull/2423)
  **Contributor**: johnlanni
  **Change Log**: Fixed an issue where configuring an MCP server with SSE forwarding might cause the controller to crash, optimizing code robustness.
  **Feature Value**: Solves potential controller crash issues, improves system stability, ensuring continuous availability of user services.

- **Related PR**: [#2408](https://github.com/alibaba/higress/pull/2408)
  **Contributor**: daixijun
  **Change Log**: Fixed missing finishReason in the Gemini provider, converting STOP to lowercase and maintaining consistency with the OpenAI API, while fixing missing finishReason content in streaming responses.
  **Feature Value**: Enhances the compatibility and stability of AI proxies, ensuring users can correctly obtain finishReason information when using the Gemini API, avoiding potential errors and experience issues.

- **Related PR**: [#2405](https://github.com/alibaba/higress/pull/2405)
  **Contributor**: Erica177
  **Change Log**: Fixed spelling errors in the `McpStreamableProtocol` constant in multiple files, ensuring the correctness of protocol support mapping, upstream type mapping, and route rewriting logic.
  **Feature Value**: Ensures protocol support consistency and correctness after spelling corrections, avoiding protocol recognition failures or mapping anomalies due to spelling issues, improving system stability.

- **Related PR**: [#2398](https://github.com/alibaba/higress/pull/2398)
  **Contributor**: Erica177
  **Change Log**: Fixed spelling errors in the McpStreambleProtocol constant and replaced hardcoded namespace values in the processServerConfig function with constants.
  **Feature Value**: Enhances code logic correctness and maintainability, avoiding potential runtime issues caused by spelling errors or hardcoded values.

### ♻️ Refactoring & Optimization

- **Related PR**: [#2458](https://github.com/alibaba/higress/pull/2458)
  **Contributor**: johnlanni
  **Change Log**: Switched the MCP server's dependency from higress's wasm-go repository to an independent wasm-go repository, involving path adjustments and dependency updates across multiple modules.
  **Feature Value**: Enhances code maintainability and independence, reduces inter-module coupling, and provides users with more stable and efficient WASM functionality support.

- **Related PR**: [#2403](https://github.com/alibaba/higress/pull/2403)
  **Contributor**: johnlanni
  **Change Log**: Unified line-ending markers in the MCP session filter to improve code consistency and maintainability.
  **Feature Value**: Reduces confusion by unifying line-ending markers, improving code readability and maintainability, with no direct impact on users.

### 📚 Documentation Updates

- **Related PR**: [#2503](https://github.com/alibaba/higress/pull/2503)
  **Contributor**: CH3CHO
  **Change Log**: Corrected the spelling of configuration attribute names in the ai-proxy plugin README documentation, changing `vertexGeminiSafetySetting` to `geminiSafetySetting`.
  **Feature Value**: Improves documentation accuracy and standardization, avoiding configuration issues caused by incorrect configuration item names, enhancing user experience.

- **Related PR**: [#2433](https://github.com/alibaba/higress/pull/2433)
  **Contributor**: johnlanni
  **Change Log**: Added release notes for Higress version 2.1.4, including support for Google Cloud Vertex AI services and other new features, and supplemented related Chinese documentation content.
  **Feature Value**: Provides users with clear version update information, helping them quickly understand the new features, improvements, and usage changes in version 2.1.4, enhancing user experience and upgrade efficiency.

- **Related PR**: [#2418](https://github.com/alibaba/higress/pull/2418)
  **Contributor**: xuruidong
  **Change Log**: Fixed broken links in the mcp-servers README_zh.md and updated references to GJSON Template syntax.
  **Feature Value**: Improves documentation accuracy and readability, ensuring users can correctly access related tools and documentation resources, enhancing user experience.

- **Related PR**: [#2327](https://github.com/alibaba/higress/pull/2327)
  **Contributor**: hourmoneys
  **Change Log**: Added documentation for mcp-server, including tool function descriptions and configuration file updates. Specifically, it involves descriptions and configurations for tools related to querying殘保金年份 and基数, and social security calculations.
  **Feature Value**: Provides users with clear mcp-server functionality descriptions and configuration guidance, improving usability and integration efficiency, helping developers quickly understand and use related tools.

---

## 📊 Release Statistics

- 🚀 New Features: 20 items
- 🐛 Bug Fixes: 15 items
- ♻️ Refactoring & Optimization: 2 items
- 📚 Documentation Updates: 4 items

**Total**: 41 changes (including 9 significant updates)

Thanks to all contributors! 🎉\n
# Higress Console


## 📋 Release Overview

This release contains **8** updates, covering areas such as feature enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 3 items
- **Bug Fixes**: 3 items
- **Documentation Updates**: 1 item
- **Testing Improvements**: 1 item

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#540](https://github.com/higress-group/higress-console/pull/540)
  **Contributor**: CH3CHO
  **Change Log**: Added support for the Vertex LLM provider type, including implementation of authentication mechanisms and configuration interfaces, expanding AI service integration capabilities.
  **Feature Value**: Users can now integrate advanced models like Google Gemini through the Vertex AI platform, enhancing the scalability and multi-cloud support capabilities of AI services.

- **Related PR**: [#530](https://github.com/higress-group/higress-console/pull/530)
  **Contributor**: Thomas-Eliot
  **Change Log**: This PR mainly implemented console management functionality for the MCP Server. By adding and modifying multiple module codes, the server's configuration management and operational capabilities were enhanced.
  **Feature Value**: Provides users with visual management capabilities for the MCP Server, improving configuration convenience and service management efficiency, positively impacting system scalability.

- **Related PR**: [#529](https://github.com/higress-group/higress-console/pull/529)
  **Contributor**: CH3CHO
  **Change Log**: Added multi-model mapping rule configuration capability for AI routing upstreams, supporting advanced configuration editing via pop-up windows, enhancing the flexibility of routing policies.
  **Feature Value**: Users can define mapping rules for different models, improving the diversity and accuracy of AI routing configurations, enhancing service adaptability and user experience.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#537](https://github.com/higress-group/higress-console/pull/537)
  **Contributor**: CH3CHO
  **Change Log**: Fixed compatibility issues in the `URL.parse` function by replacing it with `new URL()` to support more browser versions.
  **Feature Value**: Enhances the application's compatibility and stability across different browser environments, ensuring features operate correctly.

- **Related PR**: [#528](https://github.com/higress-group/higress-console/pull/528)
  **Contributor**: cr7258
  **Change Log**: Changed the default value for the PVC access mode from `rwxSupported: true` to `false` to align with the more commonly used `ReadWriteOnce` mode and avoid unnecessary configurations.
  **Feature Value**: Optimizes default configurations, reducing resource waste and potential configuration errors, improving deployment stability and合理性, while allowing users requiring multi-replica access to manually enable `ReadWriteMany`.

- **Related PR**: [#525](https://github.com/higress-group/higress-console/pull/525)
  **Contributor**: NorthernBob
  **Change Log**: Corrected the field name in the configuration from "UrlPattern" to "urlPattern", resolving naming consistency issues.
  **Feature Value**: Improves configuration readability and maintainability through consistent naming conventions, avoiding potential errors caused by inconsistent naming.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#538](https://github.com/higress-group/higress-console/pull/538)
  **Contributor**: zhangjingcn
  **Change Log**: Updated the MCP Server plugin documentation, correcting the description of the errorResponseTemplate trigger conditions and fixing escape issues in GJSON paths.
  **Feature Value**: Helps users correctly configure error response templates, avoiding template misfires due to incorrect status code evaluation, improving configuration accuracy and usability.

### 🧪 Testing Improvements (Testing)

- **Related PR**: [#526](https://github.com/higress-group/higress-console/pull/526)
  **Contributor**: CH3CHO
  **Change Log**: Added a unit test case to check whether the Wasm plugin image is up to date, by comparing the manifest of the currently used image tag with the latest tag.
  **Feature Value**: Ensures Wasm plugin images remain up to date, avoiding potential security and functionality issues, improving system stability and security.

---

## 📊 Release Statistics

- 🚀 New Features: 3 items
- 🐛 Bug Fixes: 3 items
- 📚 Documentation Updates: 1 item
- 🧪 Testing Improvements: 1 item

**Total**: 8 changes

Thank you to all contributors for your hard work! 🎉\n
