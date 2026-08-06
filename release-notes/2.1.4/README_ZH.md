# Higress Core

## 📌feature
### 支持Google Cloud Vertex AI服务
+ 相关pr：[https://github.com/alibaba/higress/pull/2119](https://github.com/alibaba/higress/pull/2119)
+ 贡献者：[HecarimV](https://github.com/HecarimV)
+ 改变记录：新增对Google Cloud Vertex AI的支持，允许通过OpenAI协议代理Vertex服务。
+ 功能价值：该功能扩展了AI代理的兼容性，使用户能够利用Vertex AI提供的模型和能力。

### 新增 HackMD MCP Server
+ 相关pr：[https://github.com/alibaba/higress/pull/2260](https://github.com/alibaba/higress/pull/2260)
+ 贡献者：[Whitea029](https://github.com/Whitea029)
+ 改变记录：新增 HackMD MCP 服务器功能，支持用户通过 MCP 协议与 HackMD 平台交互，包括用户数据管理、笔记操作和团队协作功能。
+ 功能价值：该 PR 增加了对 HackMD 的支持，扩展了 MCP 服务器的功能，增强了用户的协作能力。

### 新增君润人力社保工具MCP Server
+ 相关pr：[https://github.com/alibaba/higress/pull/2303](https://github.com/alibaba/higress/pull/2303)
+ 贡献者：[hourmoneys](https://github.com/hourmoneys)
+ 改变记录：君润人力提交的社保工具MCP服务器的mcp to rest配置，详细描述了其功能、使用方法和配置方式，包括多个API接口的说明和示例。
+ 功能价值：为开发者提供了清晰的社保计算工具使用指南，有助于提升工具的可集成性和易用性。

### 添加 Claude 图片理解和 Tools 调用能力
+ 相关pr：[https://github.com/alibaba/higress/pull/2385](https://github.com/alibaba/higress/pull/2385)
+ 贡献者：[daixijun](https://github.com/daixijun)
+ 改变记录：为AI代理添加了Claude图片理解和工具调用功能，支持流式输出和tokens统计，兼容OpenAI接口规范，并扩展了models接口支持。
+ 功能价值：该PR增强了AI代理的功能，使其能够处理图片输入和调用工具，提升了与Claude的兼容性和用户体验。

### 新增Gemini模型支持
+ 相关pr：[https://github.com/alibaba/higress/pull/2380](https://github.com/alibaba/higress/pull/2380)
+ 贡献者：[daixijun](https://github.com/daixijun)
+ 改变记录：新增了对Gemini模型的支持，包括模型列表接口、生图接口和对话文生图能力，扩展了AI代理的功能范围。
+ 功能价值：新增了Gemini模型的完整支持，提升了AI代理的多模型兼容性和图像生成能力。

### 新增Amazon Bedrock图像生成支持
+ 相关pr：[https://github.com/alibaba/higress/pull/2212](https://github.com/alibaba/higress/pull/2212)
+ 贡献者：[daixijun](https://github.com/daixijun)
+ 改变记录：新增对Amazon Bedrock图像生成的支持，扩展了AI代理的功能，允许通过Bedrock API进行文本到图像的生成。
+ 功能价值：为用户提供了一种新的AI图像生成方式，增强了系统的功能和灵活性。

### 新增模型映射正则表达式支持
+ 相关pr：[https://github.com/alibaba/higress/pull/2358](https://github.com/alibaba/higress/pull/2358)
+ 贡献者：[daixijun](https://github.com/daixijun)
+ 改变记录：新增了对模型映射的正则表达式支持，允许更灵活地进行模型名称替换，解决了特定场景下的模型调用问题。
+ 功能价值：该PR增强了AI代理插件的功能，使模型映射更加灵活和强大，提高了系统的可配置性和适用性。

### 集群限流规则全局阈值配置
+ 相关pr：[https://github.com/alibaba/higress/pull/2262](https://github.com/alibaba/higress/pull/2262)
+ 贡献者：[hanxiantao](https://github.com/hanxiantao)
+ 改变记录：新增了对集群限流规则的全局阈值配置支持，提升了限流策略的灵活性和可配置性。
+ 功能价值：该PR为集群限流插件增加了全局限流阈值配置功能，允许对整个自定义规则组设置统一的限流阈值，增强了限流策略的灵活性和适用性。

### 新增OpenAI文件和批次接口支持
+ 相关pr：[https://github.com/alibaba/higress/pull/2355](https://github.com/alibaba/higress/pull/2355)
+ 贡献者：[daixijun](https://github.com/daixijun)
+ 改变记录：为AI代理模块添加了对OpenAI和Qwen的/v1/files与/v1/batches接口的支持，扩展了AI服务的兼容性。
+ 功能价值：新增文件和批次接口支持，提升了AI代理对多种服务的兼容能力。

### 新增OpenAI兼容接口映射能力
+ 相关pr：[https://github.com/alibaba/higress/pull/2341](https://github.com/alibaba/higress/pull/2341)
+ 贡献者：[daixijun](https://github.com/daixijun)
+ 改变记录：新增对OpenAI兼容的图片生成、图片编辑和音频处理接口的支持，扩展了AI代理的功能，使其能够适配更多模型。
+ 功能价值：该PR为AI代理增加了对OpenAI兼容接口的映射能力，提升了系统灵活性和扩展性。

### 新增访问日志记录请求插件
+ 相关pr：[https://github.com/alibaba/higress/pull/2265](https://github.com/alibaba/higress/pull/2265)
+ 贡献者：[forgottener](https://github.com/forgottener)
+ 改变记录：新增功能：支持在Higress访问日志中记录请求头、请求体、响应头和响应体信息，提升日志可追溯性。
+ 功能价值：该PR增强了Higress的日志功能，使开发者能够更全面地监控和调试HTTP通信过程。

### 新增dify ai-proxy e2e测试
+ 相关pr：[https://github.com/alibaba/higress/pull/2319](https://github.com/alibaba/higress/pull/2319)
+ 贡献者：[VinciWu557](https://github.com/VinciWu557)
+ 改变记录：新增 dify ai-proxy 插件 e2e 测试，支持对 dify 模型的完整端到端测试，确保其功能正确性和稳定性。
+ 功能价值：为 dify ai-proxy 插件添加了完整的 e2e 测试，提升了插件的可靠性和可维护性。

### 前端灰度发布唯一标识配置
+ 相关pr：[https://github.com/alibaba/higress/pull/2371](https://github.com/alibaba/higress/pull/2371)
+ 贡献者：[heimanba](https://github.com/heimanba)
+ 改变记录：新增uniqueGrayTag配置项检测功能，支持根据用户自定义的uniqueGrayTag设置唯一标识cookie，提升灰度发布灵活性和可配置性。
+ 功能价值：该PR增强了前端灰度配置能力，允许用户自定义唯一标识，优化了灰度流量控制机制，提升了系统的可扩展性和用户体验。

### 新增Doubao图像生成接口支持
+ 相关pr：[https://github.com/alibaba/higress/pull/2331](https://github.com/alibaba/higress/pull/2331)
+ 贡献者：[daixijun](https://github.com/daixijun)
+ 改变记录：新增对Doubao图像生成接口的支持，扩展了AI代理的功能，使其能够处理图像生成请求。
+ 功能价值：该PR为AI代理添加了对Doubao图像生成功能的支持，提升了系统的能力和灵活性。

### WasmPlugin E2E测试跳过构建Higress控制器镜像
+ 相关pr：[https://github.com/alibaba/higress/pull/2264](https://github.com/alibaba/higress/pull/2264)
+ 贡献者：[cr7258](https://github.com/cr7258)
+ 改变记录：新增了在运行WasmPlugin E2E测试时跳过构建Higress控制器开发镜像的功能，提升测试效率。
+ 功能价值：该PR优化了WasmPlugin测试流程，允许用户选择性地跳过不必要的镜像构建步骤，提高测试效率。

### MCP Server API认证支持
+ 相关pr：[https://github.com/alibaba/higress/pull/2241](https://github.com/alibaba/higress/pull/2241)
+ 贡献者：[johnlanni](https://github.com/johnlanni)
+ 改变记录：该PR为Higress MCP Server插件引入了全面的API认证功能，支持通过OAS3的安全方案实现HTTP Basic、HTTP Bearer和API Key认证，增强了与后端REST API的安全集成能力。
+ 功能价值：该PR为MCP Server增加了对多种API认证方式的支持，提升了系统安全性和灵活性，对社区在构建安全微服务架构方面有显著帮助。

### GitHub Action同步CRD文件
+ 相关pr：[https://github.com/alibaba/higress/pull/2268](https://github.com/alibaba/higress/pull/2268)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：该PR新增了一个GitHub Action，用于在main分支上自动将CRD定义文件从api文件夹复制到helm文件夹，并创建一个PR。
+ 功能价值：实现了自动化同步CRD文件的功能，提高了开发流程的效率和一致性。

### ai-search插件日志信息增强
+ 相关pr：[https://github.com/alibaba/higress/pull/2323](https://github.com/alibaba/higress/pull/2323)
+ 贡献者：[johnlanni](https://github.com/johnlanni)
+ 改变记录：为ai-search插件添加了详细的日志信息，包括请求URL、集群名称和搜索重写模型，有助于调试和监控。
+ 功能价值：增加了更详细的日志信息，便于开发人员排查问题并优化性能。

### 更新Helm文件夹中的CRD文件
+ 相关pr：[https://github.com/alibaba/higress/pull/2392](https://github.com/alibaba/higress/pull/2392)
+ 贡献者：[github-actions[bot]](https://github.com/apps/github-actions)
+ 改变记录：更新了Helm文件夹中的CRD文件，增加了对MCP服务器的配置支持和元数据字段，提升了资源定义的灵活性和扩展性。
+ 功能价值：改进了Kubernetes资源定义，为MCP服务器配置提供了更全面的支持。

### Wasm ABI添加上游操作支持
+ 相关pr：[https://github.com/alibaba/higress/pull/2387](https://github.com/alibaba/higress/pull/2387)
+ 贡献者：[johnlanni](https://github.com/johnlanni)
+ 改变记录：该PR添加了与上游操作相关的Wasm ABI，为未来在Wasm插件中实现细粒度负载均衡策略（如基于GPU的LLM场景）做准备。
+ 功能价值：为Wasm插件支持更复杂的负载均衡策略奠定了基础，提升了系统灵活性和扩展性。

### key-auth插件日志级别修改
+ 相关pr：[https://github.com/alibaba/higress/pull/2275](https://github.com/alibaba/higress/pull/2275)
+ 贡献者：[lexburner](https://github.com/lexburner)
+ 改变记录：将key-auth插件中的日志级别从WARN修改为DEBUG，以减少不必要的警告信息，提高日志的可读性和准确性。
+ 功能价值：修复了key-auth插件中不必要的警告日志，优化了日志输出，提升了系统日志的清晰度。



## 📌bugfix
### WasmPlugin生成逻辑修复
+ 相关pr：[https://github.com/alibaba/higress/pull/2237](https://github.com/alibaba/higress/pull/2237)
+ 贡献者：[Erica177](https://github.com/Erica177)
+ 改变记录：修复了WasmPlugin生成逻辑中未设置fail strategy的问题，新增了FAIL_OPEN策略以提高系统稳定性。
+ 功能价值：为WasmPlugin添加了默认的fail strategy，避免因插件故障导致系统异常。

### 修复OpenAI自定义路径透传问题
+ 相关pr：[https://github.com/alibaba/higress/pull/2364](https://github.com/alibaba/higress/pull/2364)
+ 贡献者：[daixijun](https://github.com/daixijun)
+ 改变记录：修复了配置 openaiCustomUrl 后，对不支持的 API 路径透传时出现错误的问题，新增了对多个 OpenAI API 路径的支持。
+ 功能价值：该 PR 修正了代理服务在自定义路径配置下的逻辑问题，提高了兼容性和稳定性。

### 修复Nacos MCP工具配置处理逻辑
+ 相关pr：[https://github.com/alibaba/higress/pull/2394](https://github.com/alibaba/higress/pull/2394)
+ 贡献者：[Erica177](https://github.com/Erica177)
+ 改变记录：修复了Nacos MCP工具配置处理逻辑，并添加了单元测试，确保配置更新和监听机制的稳定性与正确性。
+ 功能价值：改进了MCP服务的配置处理逻辑，提高了系统的稳定性和可维护性。

### 修复SSE响应中混合换行符处理
+ 相关pr：[https://github.com/alibaba/higress/pull/2344](https://github.com/alibaba/higress/pull/2344)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：修复了SSE响应中混合换行符的处理问题，改进了SSE数据解析逻辑，确保支持不同换行符组合的正确处理。
+ 功能价值：该PR解决了SSE响应中换行符处理不兼容的问题，提升了系统对SSE数据的兼容性和稳定性。

### 修复proxy-wasm-cpp-sdk依赖问题
+ 相关pr：[https://github.com/alibaba/higress/pull/2281](https://github.com/alibaba/higress/pull/2281)
+ 贡献者：[johnlanni](https://github.com/johnlanni)
+ 改变记录：修复了 proxy-wasm-cpp-sdk 依赖的 emsdk 配置问题，解决了处理大请求体时内存分配失败的问题。
+ 功能价值：修复了影响请求处理的严重 Bug，提升了系统稳定性。

### 修复Bedrock请求中模型名称URL编码问题
+ 相关pr：[https://github.com/alibaba/higress/pull/2321](https://github.com/alibaba/higress/pull/2321)
+ 贡献者：[HecarimV](https://github.com/HecarimV)
+ 改变记录：修复Bedrock请求中模型名称的URL编码问题，避免特殊字符导致的请求失败，并移除了冗余的编码函数。
+ 功能价值：解决了模型名称在请求中因特殊字符导致的问题，提升系统稳定性。

### 修复未配置向量提供程序时的错误
+ 相关pr：[https://github.com/alibaba/higress/pull/2351](https://github.com/alibaba/higress/pull/2351)
+ 贡献者：[mirror58229](https://github.com/mirror58229)
+ 改变记录：修复了在未配置向量提供程序时，'EnableSemanticCachefalse' 被错误设置的问题，避免了在 'handleResponse' 中出现错误日志。
+ 功能价值：该PR修复了一个可能导致错误日志的Bug，提升了系统的稳定性和用户体验。

### 修复Nacos 3 MCP服务器重写配置错误
+ 相关pr：[https://github.com/alibaba/higress/pull/2211](https://github.com/alibaba/higress/pull/2211)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：修复了Nacos 3 MCP服务器生成的重写配置错误问题，确保流量路由正确。
+ 功能价值：修正了MCP服务器的重写配置，避免因配置错误导致的服务不可用问题。

### 修复ai-search插件Content-Length请求头问题
+ 相关pr：[https://github.com/alibaba/higress/pull/2363](https://github.com/alibaba/higress/pull/2363)
+ 贡献者：[johnlanni](https://github.com/johnlanni)
+ 改变记录：修复了ai-search插件中未正确移除Content-Length请求头的问题，确保请求头处理逻辑的完整性。
+ 功能价值：修复了ai-search插件中Content-Length请求头未被移除的问题，提升了插件的稳定性和兼容性。

### 修复Gemini代理请求中的Authorization头问题
+ 相关pr：[https://github.com/alibaba/higress/pull/2220](https://github.com/alibaba/higress/pull/2220)
+ 贡献者：[hanxiantao](https://github.com/hanxiantao)
+ 改变记录：修复了AI代理Gemini时错误携带Authorization请求头的问题，确保代理请求符合Gemini API的要求。
+ 功能价值：移除了Gemini代理请求中的Authorization头，解决了API调用失败的问题。

### 修复ToolArgs结构体类型定义问题
+ 相关pr：[https://github.com/alibaba/higress/pull/2231](https://github.com/alibaba/higress/pull/2231)
+ 贡献者：[Erica177](https://github.com/Erica177)
+ 改变记录：修复了issue #2222，将ToolArgs结构体中的Items字段从[]interface{}改为interface{}，以适配特定的使用场景。
+ 功能价值：修复了一个类型定义问题，提高了代码的灵活性和兼容性。

## 📌refactor

### MCP服务器配置生成逻辑重构
+ 相关pr：[https://github.com/alibaba/higress/pull/2207](https://github.com/alibaba/higress/pull/2207)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：重构了mcpServer.matchList配置生成逻辑，支持从Nacos 3.x发现mcp-sse类型的MCP服务器，并修复了DestinationRules的ServiceKey问题。
+ 功能价值：改进了MCP服务器的配置管理，增强了对Nacos 3.x的支持，并解决了多MCP服务器的路由问题。

### MCP服务器自动发现逻辑重构
+ 相关pr：[https://github.com/alibaba/higress/pull/2382](https://github.com/alibaba/higress/pull/2382)
+ 贡献者：[Erica177](https://github.com/Erica177)
+ 改变记录：重构了 MCP 服务器的自动发现逻辑，并修复了一些问题，提高了代码的可维护性和扩展性。
+ 功能价值：通过重构和优化 MCP 服务器的自动发现逻辑，提升了系统的稳定性和可扩展性，同时修复了一些潜在的问题。

## 📌doc
### 优化README.md翻译流程
+ 相关pr：[https://github.com/alibaba/higress/pull/2208](https://github.com/alibaba/higress/pull/2208)
+ 贡献者：[littlejiancc](https://github.com/littlejiancc)
+ 改变记录：优化了README.md的翻译流程，支持流式传输并避免重复PR，提升了多语言文档的维护效率。
+ 功能价值：改进了自动化翻译流程，确保文档一致性并减少人工干预。

### 自动化翻译工作流
+ 相关pr：[https://github.com/alibaba/higress/pull/2228](https://github.com/alibaba/higress/pull/2228)
+ 贡献者：[MAVRICK-1](https://github.com/MAVRICK-1)
+ 改变记录：该PR添加了一个GitHub Actions工作流，用于自动翻译非英文的issue、PR和讨论内容，提高Higress的国际化和可访问性。
+ 功能价值：通过自动化翻译提升Higress对国际用户和贡献者的友好度，增强项目全球化能力。

# Higress Console

## 📌feature
### 支持配置多个自定义OpenAI LLM提供者端点
+ 相关pr：[https://github.com/higress-group/higress-console/pull/517](https://github.com/higress-group/higress-console/pull/517)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：该PR为自定义OpenAI LLM提供者支持配置多个端点，增强了系统的灵活性和可扩展性。通过重构LLM提供者端点管理逻辑，实现了对IP+端口格式的URL的支持，并确保所有URL具有相同的协议和路径。
+ 功能价值：该PR使系统能够支持多个自定义OpenAI服务端点，提升了系统的灵活性和可靠性，适用于需要多实例或负载均衡的场景。

### 自定义图片URL模式迁移与Wasm插件服务配置类引入
+ 相关pr：[https://github.com/higress-group/higress-console/pull/504](https://github.com/higress-group/higress-console/pull/504)
+ 贡献者：[Thomas-Eliot](https://github.com/Thomas-Eliot)
+ 改变记录：将自定义图片URL模式从SDK模块迁移到控制台模块，并引入Wasm插件服务配置类，以支持更灵活的Wasm插件管理。
+ 功能价值：该PR重构了配置管理逻辑，提升了系统对Wasm插件的可配置性和扩展性，为后续功能增强打下基础。

### 新增配置参数dependControllerApi
+ 相关pr：[https://github.com/higress-group/higress-console/pull/506](https://github.com/higress-group/higress-console/pull/506)
+ 贡献者：[Thomas-Eliot](https://github.com/Thomas-Eliot)
+ 改变记录：新增配置参数dependControllerApi，支持在不使用注册中心时解耦对Higress Controller的依赖，提升架构灵活性和可配置性。
+ 功能价值：该PR通过引入新配置项，使系统在特定场景下可以绕过注册中心直接与K8s API交互，增强了系统的灵活性和适应性。

### 更新Nacos3服务源表单以支持Nacos 3.0.1+
+ 相关pr：[https://github.com/higress-group/higress-console/pull/521](https://github.com/higress-group/higress-console/pull/521)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：该PR更新了nacos3服务源的表单，以支持nacos 3.0.1+版本，并修复了删除服务源后创建新源时显示错误的问题。
+ 功能价值：该PR优化了服务源配置界面，提升了对nacos 3.0.1+版本的支持，同时改善用户体验。

### 改进K8s能力初始化逻辑
+ 相关pr：[https://github.com/higress-group/higress-console/pull/513](https://github.com/higress-group/higress-console/pull/513)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：改进了K8s能力初始化逻辑，增加了重试机制和失败后默认支持Ingress V1的处理，提升了系统稳定性和容错性。
+ 功能价值：修复了K8s能力检测不稳定的问题，确保控制台正常运行，提升用户体验。

### 支持JDK 8
+ 相关pr：[https://github.com/higress-group/higress-console/pull/497](https://github.com/higress-group/higress-console/pull/497)
+ 贡献者：[Thomas-Eliot](https://github.com/Thomas-Eliot)
+ 改变记录：修复了代码中使用Java 11特性导致的兼容性问题，使其支持JDK 8。主要修改了代码中使用String.repeat()方法和List.of()等Java 11特性的部分。
+ 功能价值：该PR解决了项目对JDK 8的兼容性问题，使项目可以在JDK 8环境中正常运行。

### 在证书编辑表单中添加安全提示信息
+ 相关pr：[https://github.com/higress-group/higress-console/pull/512](https://github.com/higress-group/higress-console/pull/512)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：在证书编辑表单中添加了一条安全提示信息，明确告知用户当前证书和私钥数据不会显示，并指导用户直接输入新数据。
+ 功能价值：为用户提供更清晰的操作指引，提升数据安全性意识，避免误操作。

### 更新OpenAI提供者类型的显示名称
+ 相关pr：[https://github.com/higress-group/higress-console/pull/510](https://github.com/higress-group/higress-console/pull/510)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：更新了OpenAI提供者类型的显示名称，使其更明确地表明其兼容性，提升用户对服务的识别度。
+ 功能价值：修改了OpenAI提供者的显示名称，使用户能更清楚地区分服务类型，提升使用体验。



## 📌bugfix
### 修复AI路由中无法启用路径大小写忽略匹配的Bug
+ 相关pr：[https://github.com/higress-group/higress-console/pull/508](https://github.com/higress-group/higress-console/pull/508)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：修复了AI路由中无法启用路径大小写忽略匹配的Bug，通过修改路径谓词的处理逻辑并新增规范化函数确保功能正确性。
+ 功能价值：修复了AI路由配置中路径大小写匹配的问题，提升了路由规则的灵活性和用户体验。

### 修复higress-config更新功能中的多个问题
+ 相关pr：[https://github.com/higress-group/higress-console/pull/509](https://github.com/higress-group/higress-console/pull/509)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：修复了higress-config更新功能中的多个问题，包括将HTTP方法从POST改为PUT、添加成功提示信息以及修复方法名拼写错误。
+ 功能价值：修复了配置更新的API调用方式和提示逻辑，提升了用户体验和系统稳定性。

### 修复前端页面中的文本显示错误
+ 相关pr：[https://github.com/higress-group/higress-console/pull/503](https://github.com/higress-group/higress-console/pull/503)
+ 贡献者：[CH3CHO](https://github.com/CH3CHO)
+ 改变记录：修复了前端页面中一个文本显示错误的问题，将原本不正确的文本内容更正为准确的描述。
+ 功能价值：修正了界面中文本内容，提升了用户对功能的理解和使用体验。

## 📌refactor
### 优化分页工具逻辑
+ 相关pr：[https://github.com/higress-group/higress-console/pull/499](https://github.com/higress-group/higress-console/pull/499)
+ 贡献者：[Thomas-Eliot](https://github.com/Thomas-Eliot)
+ 改变记录：优化了分页工具的逻辑，通过引入更高效的集合处理方式和简化代码结构，提升了分页功能的性能和可维护性。
+ 功能价值：优化了分页工具的实现方式，提高了数据处理效率和代码可读性，对系统性能有积极影响。
