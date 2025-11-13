# Higress


## 📋 本次发布概览

本次发布包含 **44** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 23项
- **Bug修复**: 13项
- **重构优化**: 3项
- **文档更新**: 1项
- **测试改进**: 4项

### ⭐ 重点关注

本次发布包含 **3** 项重要更新，建议重点关注：

- **feat(mcp-server): add server-level default authentication and MCP proxy server support** ([#3096](https://github.com/alibaba/higress/pull/3096)): 通过引入服务器级默认认证，增强了系统的安全性与灵活性，使得用户可以更方便地管理工具和服务间的安全策略，提升了整体服务的安全性和用户体验。
- **feat: add higress api mcp server** ([#2923](https://github.com/alibaba/higress/pull/2923)): 通过集成higress api mcp server，用户能够更方便地管理和操作Higress的路由、服务来源、AI路由等资源，提升了系统的可管理性和灵活性。
- **feat: implement `hgctl agent` & `mcp add` subcommand ** ([#3051](https://github.com/alibaba/higress/pull/3051)): 通过引入新的子命令，提升了用户的操作便捷性和系统的可扩展性，使得用户能够更灵活地管理和配置MCP服务。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat(mcp-server): add server-level default authentication and MCP proxy server support

**相关PR**: [#3096](https://github.com/alibaba/higress/pull/3096) | **贡献者**: [@johnlanni](https://github.com/johnlanni)

**使用背景**

在现代微服务架构中，认证和授权是确保系统安全的关键因素。现有的认证机制可能过于分散且难以管理，导致配置复杂且容易出错。此外，随着系统规模的扩大，单一的MCP服务器可能无法满足性能需求，需要引入代理服务器来分担负载。此功能旨在解决这些问题，提供一个统一的、可配置的认证机制，并支持通过MCP代理服务器进行请求转发。目标用户群体包括需要增强系统安全性和可扩展性的开发者和运维人员。

**功能详述**

1. **服务器级别的默认认证**：新增了`defaultDownstreamSecurity`和`defaultUpstreamSecurity`配置项，分别用于设置客户端到网关和网关到后端的默认认证。这些配置项可以在全局级别设置，工具级别的设置会覆盖全局设置。这种设计使得认证配置更加灵活和易于管理。
2. **MCP代理服务器类型**：引入了新的服务器类型`mcp-proxy`，允许将客户端的MCP请求转发到后端的MCP服务器。通过`mcpServerURL`字段可以指定后端MCP服务器的地址，并通过`timeout`字段控制请求超时时间。此外，还支持完整的认证机制，包括客户端到网关和网关到后端的认证。
3. **认证代码重构**：对认证相关的代码进行了重构，提高了代码的可维护性和扩展性。更新了依赖库的版本，确保了与最新版本的兼容性。

**使用方式**

1. **启用和配置**：首先，需要在配置文件中启用`defaultDownstreamSecurity`和`defaultUpstreamSecurity`，并设置相应的认证参数。对于MCP代理服务器，需要指定`mcpServerURL`和`timeout`字段。
2. **典型使用场景**：适用于需要集中管理和统一分发认证策略的微服务架构。例如，在一个多租户环境中，可以通过默认认证配置来统一管理所有租户的认证策略。
3. **注意事项**：确保所有相关组件的版本兼容性，避免因版本不一致导致的问题。同时，建议在生产环境部署前进行充分的测试，以验证配置的有效性和性能。

**功能价值**

1. **增强安全性**：通过统一的认证配置，减少了配置错误的风险，提高了整个系统的安全性。特别是在多租户环境中，能够更好地隔离不同租户的访问权限。
2. **提高灵活性**：提供了多层次的优先级配置机制，使得认证策略更加灵活和易于管理。工具级别的配置可以覆盖全局设置，适应不同的业务需求。
3. **提升可扩展性**：通过引入MCP代理服务器，可以有效地分担单个MCP服务器的负载，提高系统的整体性能和稳定性。特别是在大规模分布式系统中，这种设计能够显著提升系统的可扩展性。

---

### 2. feat: add higress api mcp server

**相关PR**: [#2923](https://github.com/alibaba/higress/pull/2923) | **贡献者**: [@Tsukilc](https://github.com/Tsukilc)

**使用背景**

此功能解决了用户在管理Higress资源时的便捷性和灵活性问题。过去，用户可能需要通过多个API或工具来管理不同的资源，如路由、服务来源和插件等。现在，通过Higress API MCP Server，用户可以在一个统一的接口中管理这些资源，包括新的AI路由、AI提供商和MCP服务器。这不仅简化了操作流程，还增强了系统的安全性和可维护性。目标用户群体主要是Higress的运维人员和开发者，他们需要高效且安全地管理Higress的各类资源。

**功能详述**

此次PR主要实现了以下功能：
1. **新增Higress API MCP Server**：提供了统一的API接口来管理Higress的路由、服务来源、AI路由、AI提供商、MCP服务器和插件等资源。
2. **鉴权机制更新**：从之前的用户名密码认证改为HTTP Basic Authentication，提高了安全性。
3. **新增工具注册**：注册了新的AI路由、AI提供商和MCP服务器管理工具，使系统能够支持这些新功能。
4. **代码优化**：移除了不必要的类型转换，提升了性能和代码清晰度。
5. **文档更新**：更新了README文档以反映新增功能。技术实现方面，通过引入新的结构体和工具函数，扩展了现有的MCP Server功能，并确保与现有功能的兼容性。

**使用方式**

启用和配置Higress API MCP Server非常简单：
1. 配置Higress Console的URL地址。
2. 选择合适的鉴权方式（如HTTP Basic Authentication）并提供相应的凭据。
3. 使用提供的API工具进行资源管理。例如，使用`list-ai-routes`列出所有AI路由，使用`add-ai-route`添加新的AI路由等。
4. 对于MCP服务器管理，可以使用`list-mcp-servers`列出所有MCP服务器，使用`add-or-update-mcp-server`添加或更新MCP服务器等。
5. 注意事项：确保所有配置文件中的字段正确填写，特别是涉及权重总和的校验。建议使用最新的客户端工具（如Cherry Studio）来提供凭据，提高安全性。

**功能价值**

这一功能为用户带来了显著的好处：
1. **提升管理效率**：通过统一的API接口，用户可以更高效地管理和配置各类Higress资源，减少了操作复杂度。
2. **增强安全性**：新的鉴权机制（如HTTP Basic Authentication）提高了系统的安全性，防止未授权访问。
3. **扩展功能**：新增的AI路由、AI提供商和MCP服务器管理工具，使得Higress能够更好地支持现代应用的需求。
4. **代码优化**：移除不必要的类型转换和优化查询参数拼接逻辑，提升了系统性能和代码质量。总的来说，这一功能不仅提升了用户体验，还增强了系统的稳定性和安全性，在Higress生态中具有重要意义。

---

### 3. feat: implement `hgctl agent` & `mcp add` subcommand 

**相关PR**: [#3051](https://github.com/alibaba/higress/pull/3051) | **贡献者**: [@erasernoob](https://github.com/erasernoob)

**使用背景**

此PR解决了用户在管理和配置MCP服务时遇到的不便。在现有的Higress CLI工具中，缺乏直接添加MCP服务的功能，用户需要手动配置复杂的API调用。此外，缺乏一个统一的代理CLI工具来初始化和管理环境。新功能旨在简化这些操作，使用户能够通过简单的命令行指令快速添加和管理MCP服务，并提供一个交互式的代理窗口来设置必要的环境变量。目标用户群体包括Higress的开发者、运维人员以及任何需要管理MCP服务的用户。

**功能详述**

此次更新实现了两个新的子命令：`hgctl agent`和`mcp add`。`hgctl agent`命令用于启动一个交互式的代理窗口，引导用户完成环境设置。`mcp add`命令则允许用户直接添加MCP服务，支持两种类型的服务：直接代理型和基于OpenAPI的MCP服务。对于直接代理型服务，用户可以通过指定URL和其他参数来发布服务；对于基于OpenAPI的服务，用户可以上传OpenAPI规范文件并进行配置。核心技术要点在于通过与Higress Console API集成，实现服务的自动注册和管理。代码变更主要集中在新增的`agent`包及其相关模块，如`base.go`、`core.go`和`mcp.go`等。

**使用方式**

启用`hgctl agent`只需运行`hgctl agent`命令，系统将在首次使用时提示用户设置必要的环境变量。要使用`mcp add`命令添加MCP服务，用户可以根据需求选择以下两种方式之一：
1. 添加直接代理型服务：
   ```bash
   hgctl mcp add mcp-deepwiki -t http https://mcp.deepwiki.com --user admin --password 123 --url http://localhost:8080
   ```
2. 添加基于OpenAPI的服务：
   ```bash
   hgctl mcp add openapi-server -t openapi --spec openapi.yaml --user admin --password 123 --url http://localhost:8080
   ```
注意事项：确保已安装Go 1.24或更高版本，且环境变量正确配置。最佳实践是在生产环境中使用自定义的日志库记录错误和调试信息。

**功能价值**

新功能显著提升了Higress CLI工具的易用性和灵活性。通过引入`hgctl agent`，用户可以轻松地初始化和管理环境，而无需手动配置复杂的环境变量。`mcp add`命令进一步简化了MCP服务的添加过程，支持多种类型的MCP服务，提高了开发和运维效率。此外，通过与Higress Console API集成，确保了服务的一致性和可靠性。这些改进不仅提升了用户体验，还增强了系统的整体性能和稳定性，在Higress生态系统中具有重要意义。

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#3126](https://github.com/alibaba/higress/pull/3126) \
  **Contributor**: @johnlanni \
  **Change Log**: 该PR更新了Envoy依赖，使得WASM插件能够通过URL查询参数配置Redis客户端缓冲行为，包括设置最大缓冲大小和刷新超时时间。 \
  **Feature Value**: 此功能允许用户更灵活地控制Redis调用相关的缓冲区参数，从而优化性能并满足特定的应用需求。

- **Related PR**: [#3123](https://github.com/alibaba/higress/pull/3123) \
  **Contributor**: @johnlanni \
  **Change Log**: 升级了代理版本至v2.2.0，更新Go工具链到1.23.7，并且为golang-filter增加了特定架构的构建目标，同时修复了与MCP服务器、OpenAI及Milvus SDK支持相关的依赖问题。 \
  **Feature Value**: 通过此次更新，增强了系统的兼容性和性能，使开发者能够更轻松地在不同架构上部署应用，同时也提升了与其他服务（如OpenAI和Milvus）集成的稳定性。

- **Related PR**: [#3108](https://github.com/alibaba/higress/pull/3108) \
  **Contributor**: @wydream \
  **Change Log**: 新增了与视频相关的API路径和处理能力，包括视频系列的API名称常量、默认功能条目及正则表达式路径处理，并更新了OpenAI提供者以支持这些新的端点。 \
  **Feature Value**: 此功能扩展了AI代理插件的能力范围，使其能够处理和解析更多类型的媒体内容请求，特别是视频相关的操作，从而增强了用户在多媒体内容管理上的灵活性和效率。

- **Related PR**: [#3071](https://github.com/alibaba/higress/pull/3071) \
  **Contributor**: @rinfx \
  **Change Log**: 该PR添加了一个名为`inject_encoded_data_to_filter_chain_on_header`的新功能示例，允许在无响应body的情况下为请求添加响应body。通过调用特定的Wasm函数，并按照指定规则处理请求和响应，确保了数据的正确注入。 \
  **Feature Value**: 此功能扩展了应用的服务能力，使得开发者能够更灵活地控制HTTP响应内容，特别是在需要动态生成或修改响应体的情况下，极大提升了服务的灵活性与用户体验。

- **Related PR**: [#3067](https://github.com/alibaba/higress/pull/3067) \
  **Contributor**: @wydream \
  **Change Log**: 该PR通过添加vLLM作为新的AI提供商，实现了对多种OpenAI兼容API的支持，包括聊天补全、文本补全等功能。 \
  **Feature Value**: 新增vLLM支持极大扩展了Higress在代理AI服务方面的能力，使用户能够更灵活地使用不同类型的AI模型和服务。

- **Related PR**: [#3060](https://github.com/alibaba/higress/pull/3060) \
  **Contributor**: @erasernoob \
  **Change Log**: 此PR增强了`hgctl mcp`和`hgctl agent`命令，使其能够自动从安装配置文件及Kubernetes secrets中获取Higress Console的凭证信息。 \
  **Feature Value**: 简化了用户在使用Higress时处理认证信息的过程，提高了操作便捷性和用户体验。

- **Related PR**: [#3043](https://github.com/alibaba/higress/pull/3043) \
  **Contributor**: @2456868764 \
  **Change Log**: 修复了Milvus默认端口错误，并在README.md中添加了Python示例代码，帮助用户更好地理解和使用该功能。 \
  **Feature Value**: 通过修正配置错误并提供Python示例代码，提高了系统的稳定性和可用性，方便用户快速上手和集成。

- **Related PR**: [#3040](https://github.com/alibaba/higress/pull/3040) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: 此PR通过添加ApiNameAnthropicMessages到Claude功能中，支持在不使用协议=original的情况下配置anthropic提供商，并直接转发/v1/messages请求至anthropic。 \
  **Feature Value**: 增强了用户对不同AI服务提供商的灵活性和兼容性，让用户能够更方便地与Claude API进行交互，从而提升了应用的多样性和用户体验。

- **Related PR**: [#3038](https://github.com/alibaba/higress/pull/3038) \
  **Contributor**: @Libres-coder \
  **Change Log**: 新增了`list-plugin-instances`工具，使AI代理能够通过MCP协议查询指定范围内的插件实例，并更新了双语文档。 \
  **Feature Value**: 此功能增强了对插件实例的管理能力，允许用户更灵活地查询不同层级下的插件信息，提升了系统的可维护性和用户体验。

- **Related PR**: [#3032](https://github.com/alibaba/higress/pull/3032) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR默认启用Qwen兼容模式，并添加了缺失的API端点，包括AsyncAIGC、AsyncTask和V1Rerank，增强了AI代理功能。 \
  **Feature Value**: 通过默认启用兼容模式并扩展API覆盖范围，该更新为用户提供了更完善的开箱即用体验及更全面的功能支持，增强了系统的易用性和灵活性。

- **Related PR**: [#3029](https://github.com/alibaba/higress/pull/3029) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: 此PR在groq提供程序中添加了对v1/responses的支持，通过更新groq.go文件中的代码实现新功能。 \
  **Feature Value**: 新增的responses功能允许用户更好地管理和处理响应数据，提高了系统的灵活性和可用性，为开发者提供了更多定制空间。

- **Related PR**: [#3024](https://github.com/alibaba/higress/pull/3024) \
  **Contributor**: @rinfx \
  **Change Log**: 增加了恶意URL和模型幻觉检测，解决了response包含空内容时的错误响应问题，并调整了特定消费者配置。 \
  **Feature Value**: 增强了系统对恶意行为的识别能力，提升了用户体验和安全性，同时优化了对于多event返回场景下的处理逻辑。

- **Related PR**: [#3008](https://github.com/alibaba/higress/pull/3008) \
  **Contributor**: @hellocn9 \
  **Change Log**: 新增支持自定义参数名以配置MCP SSE stateful会话，通过添加`higress.io/mcp-sse-stateful-param-name`注解实现。 \
  **Feature Value**: 允许用户自定义MCP SSE stateful会话的参数名，提高了应用的灵活性和可配置性，满足了更多场景下的需求。

- **Related PR**: [#3006](https://github.com/alibaba/higress/pull/3006) \
  **Contributor**: @SaladDay \
  **Change Log**: 此PR为MCP Server的Redis配置引入了Secret引用支持，使用户能够安全地存储密码而无需将其暴露在ConfigMap中。 \
  **Feature Value**: 通过允许使用Kubernetes Secret来存储敏感信息，提高了系统的安全性，避免了密码明文存储带来的风险。

- **Related PR**: [#2992](https://github.com/alibaba/higress/pull/2992) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR在认证鉴权过程中即使消费者未被授权也会记录下其名称，便于日志观察。 \
  **Feature Value**: 通过记录未授权消费者的名称，增强了系统的可审计性和故障排查能力，使管理员能够更全面地了解访问请求。

- **Related PR**: [#2978](https://github.com/alibaba/higress/pull/2978) \
  **Contributor**: @rinfx \
  **Change Log**: 该PR在认证过程中，无论是否通过，只要确定了消费者身份，就会向请求头中添加X-Mse-Consumer字段记录消费者名称。 \
  **Feature Value**: 此功能增强了系统的可追踪性，使得每个请求都能携带消费者信息，有助于后续的审计、日志分析和问题排查。

- **Related PR**: [#2968](https://github.com/alibaba/higress/pull/2968) \
  **Contributor**: @2456868764 \
  **Change Log**: 实现了向量数据库映射功能，包括字段映射系统和索引配置管理，支持多种索引类型如HNSW、IVF、SCANN等。 \
  **Feature Value**: 增强了系统的灵活性和兼容性，允许用户自定义字段映射与索引配置，从而更好地适应不同数据库架构的需求。

- **Related PR**: [#2943](https://github.com/alibaba/higress/pull/2943) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: 该PR增加了对自定义系统提示的支持，使得在生成发布说明时可以使用用户自定义的系统提示。具体实现通过修改GitHub Actions工作流配置。 \
  **Feature Value**: 此功能允许用户在生成发布说明文档时添加个性化的系统提示信息，提高了文档的灵活性和用户体验，使发布说明更加贴合项目实际情况。

- **Related PR**: [#2942](https://github.com/alibaba/higress/pull/2942) \
  **Contributor**: @2456868764 \
  **Change Log**: 修复了LLM提供者为空时的处理逻辑，优化了文档结构和内容，更新了README以更好地描述MCP工具的功能及其配置。 \
  **Feature Value**: 增强了系统对于空LLM提供者的健壮性，改善了用户体验，使用户能够更清晰地理解MCP服务器所提供的工具及其配置要求。

- **Related PR**: [#2916](https://github.com/alibaba/higress/pull/2916) \
  **Contributor**: @imp2002 \
  **Change Log**: 实现了Nginx迁移MCP服务器，并提供了7种MCP工具来自动化Nginx配置和Lua插件迁移到Higress的过程。 \
  **Feature Value**: 该功能极大地简化了从Nginx到Higress的迁移工作，提高了迁移效率，降低了用户的操作复杂度。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#3120](https://github.com/alibaba/higress/pull/3120) \
  **Contributor**: @lexburner \
  **Change Log**: 调整了ai-proxy插件中的日志级别，通过将特定警告信息的日志级别从Warn下调，减少了不必要的冗余警告输出。 \
  **Feature Value**: 减少冗余警告信息可以提高日志的可读性与维护性，帮助用户更专注于重要的日志信息，从而提升整体用户体验。

- **Related PR**: [#3118](https://github.com/alibaba/higress/pull/3118) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR修复了端口级别的TLS和负载均衡设置会无条件覆盖现有ingress注解配置的问题。通过添加空检查并改进策略合并逻辑，确保了已有配置不会被误覆盖。 \
  **Feature Value**: 该修复避免了不必要的配置丢失或替换，提高了系统的稳定性和可靠性，确保用户自定义的ingress注解能够正确生效，增强了用户体验。

- **Related PR**: [#3095](https://github.com/alibaba/higress/pull/3095) \
  **Contributor**: @rinfx \
  **Change Log**: 修复了claude2openai转换过程中usage信息丢失的问题，并在bedrock流式工具响应中添加了index字段，以提高数据处理的准确性和完整性。 \
  **Feature Value**: 该修复确保了用户在使用claude2openai转换时能够获得完整的usage信息，同时通过添加index字段增强了对bedrock流式响应的追踪能力，提升了用户体验和系统的可维护性。

- **Related PR**: [#3084](https://github.com/alibaba/higress/pull/3084) \
  **Contributor**: @rinfx \
  **Change Log**: PR修复了当使用流式传输时，Claude转换为OpenAI请求过程中不包含include_usage: true的问题。 \
  **Feature Value**: 该修复确保了在流式传输模式下，用户能够正确获得使用统计信息，提升了服务的完整性和用户体验。

- **Related PR**: [#3074](https://github.com/alibaba/higress/pull/3074) \
  **Contributor**: @Jing-ze \
  **Change Log**: 在log-request-response插件中增加了对Content-Encoding的检查，以避免记录被压缩的请求/响应体导致日志内容乱码。 \
  **Feature Value**: 通过改进日志记录机制，确保了访问日志中的响应体信息可读性和准确性，提升了用户体验和系统调试效率。

- **Related PR**: [#3069](https://github.com/alibaba/higress/pull/3069) \
  **Contributor**: @Libres-coder \
  **Change Log**: 此PR通过在prebuild.sh脚本中添加go mod tidy命令修复了CI测试框架中的一个bug，确保根目录下的go.mod文件也得到更新。 \
  **Feature Value**: 解决了因go.mod文件未正确更新导致的CI测试失败问题，保证了所有触发wasm插件e2e测试的PR能顺利通过CI验证。

- **Related PR**: [#3010](https://github.com/alibaba/higress/pull/3010) \
  **Contributor**: @rinfx \
  **Change Log**: 解决了bedrock EventStream响应拆包导致的解析失败问题，调整了maxtoken转换逻辑，确保数据完整性。 \
  **Feature Value**: 修复了EventStream解析错误的问题，提高了系统的稳定性和可靠性，确保用户获得准确的数据。

- **Related PR**: [#2997](https://github.com/alibaba/higress/pull/2997) \
  **Contributor**: @hanxiantao \
  **Change Log**: 优化了集群、AI Token和WASM插件的限流逻辑，通过累加方式统计请求次数和token使用量，解决了修改限流值时重置计数的问题。 \
  **Feature Value**: 确保了即使在调整限流阈值的情况下，也不会导致已有的请求计数或token使用量被重置，从而提供了更准确可靠的限流机制。

- **Related PR**: [#2988](https://github.com/alibaba/higress/pull/2988) \
  **Contributor**: @johnlanni \
  **Change Log**: 修正了jsonrpc-converter中的问题，使用原始JSON而不是不正确的JSON字符串格式化方法进行数据处理。 \
  **Feature Value**: 解决了因JSON格式化错误导致的数据处理问题，提高了系统的稳定性和可靠性，确保用户能够获得准确的数据响应。

- **Related PR**: [#2973](https://github.com/alibaba/higress/pull/2973) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR通过确保在生成`mcp-session`配置时总是使用通配符来解决设置`match_rule_domain`为空字符串导致的兼容性问题，避免了MCP服务器与Higress 2.1.8版本不兼容的风险。 \
  **Feature Value**: 修复了因设置`match_rule_domain`为空字符串引起的问题，提高了系统的稳定性和兼容性，使用户能够顺利使用MCP服务器而不会遇到由于版本差异引起的错误。

- **Related PR**: [#2952](https://github.com/alibaba/higress/pull/2952) \
  **Contributor**: @Erica177 \
  **Change Log**: 修正了ToolSecurity结构体中Id字段的json标签，从type更改为id，以确保数据序列化时正确的映射。 \
  **Feature Value**: 此修复解决了因Json标签错误导致的数据解析问题，提高了系统的稳定性和数据准确性，增强了用户体验。

- **Related PR**: [#2948](https://github.com/alibaba/higress/pull/2948) \
  **Contributor**: @johnlanni \
  **Change Log**: 修正了Azure服务URL类型检测逻辑，增加了对Azure OpenAI Response API的支持，并改进了流式事件解析。 \
  **Feature Value**: 提高了Azure OpenAI集成的稳定性和兼容性，确保了自定义路径和响应API能够被正确处理，提升了用户体验。

- **Related PR**: [#2941](https://github.com/alibaba/higress/pull/2941) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR修复了与旧配置兼容性的问题，通过在`main.go`文件中调整了数据结构的定义方式来支持旧版本配置。 \
  **Feature Value**: 提升了系统的向后兼容能力，确保现有用户在升级到新版本时不会遇到由于配置格式变化引起的功能异常。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#3119](https://github.com/alibaba/higress/pull/3119) \
  **Contributor**: @johnlanni \
  **Change Log**: 将Connection中的reqChan和deltaReqChan替换为channels.Unbounded，以防止HTTP2流控导致的死锁问题。 \
  **Feature Value**: 通过避免HTTP2流控引起的死锁，确保客户端请求和响应的顺畅处理，提升了系统的稳定性和性能。

- **Related PR**: [#3113](https://github.com/alibaba/higress/pull/3113) \
  **Contributor**: @johnlanni \
  **Change Log**: 实现了Protobuf消息的递归哈希计算及缓存，采用xxHash算法，并为google.protobuf.Any类型和映射字段提供了特殊处理。 \
  **Feature Value**: 通过减少在过滤链匹配和监听器处理中的重复序列化操作来优化LDS性能，从而提高整体系统效率。

- **Related PR**: [#2945](https://github.com/alibaba/higress/pull/2945) \
  **Contributor**: @rinfx \
  **Change Log**: 该PR通过更新全局最小请求数的Lua脚本来优化了ai-load-balancer中选择pod的逻辑，减少了不必要的代码行并提高了性能。 \
  **Feature Value**: 优化后的负载均衡策略更加高效地分配请求，减少了延迟和资源浪费，提升了用户体验和服务稳定性。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2965](https://github.com/alibaba/higress/pull/2965) \
  **Contributor**: @CH3CHO \
  **Change Log**: 更新了ai-proxy README文件中azureServiceUrl的描述，确保文档内容准确无误地反映了配置项的实际用途。 \
  **Feature Value**: 通过改进对azureServiceUrl字段的说明，帮助用户更好地理解其作用和配置方法，提升文档的可读性和实用性。

### 🧪 测试改进 (Testing)

- **Related PR**: [#3110](https://github.com/alibaba/higress/pull/3110) \
  **Contributor**: @Jing-ze \
  **Change Log**: 该PR在CI工作流中添加了CODECOV_TOKEN环境变量，用于确保Codecov上传步骤能够正确认证。 \
  **Feature Value**: 通过增加CODECOV_TOKEN，提高了代码覆盖率报告的准确性和安全性，有助于项目维护者更好地监控和改进测试覆盖率。

- **Related PR**: [#3097](https://github.com/alibaba/higress/pull/3097) \
  **Contributor**: @johnlanni \
  **Change Log**: 为mcp-server插件新增了单元测试代码，确保其核心功能的稳定性和可靠性。 \
  **Feature Value**: 通过增加单元测试提高了mcp-server插件的质量，增强了系统的健壮性，减少了潜在错误的发生几率。

- **Related PR**: [#2998](https://github.com/alibaba/higress/pull/2998) \
  **Contributor**: @Patrisam \
  **Change Log**: 此PR实现了Cloudflare的端到端测试用例，增加了go-wasm-ai-proxy.go和go-wasm-ai-proxy.yaml两个文件中的内容。 \
  **Feature Value**: 通过增加Cloudflare端到端测试用例，提高了系统的可靠性和稳定性，有助于开发者更好地理解和验证系统集成后的表现。

- **Related PR**: [#2980](https://github.com/alibaba/higress/pull/2980) \
  **Contributor**: @Jing-ze \
  **Change Log**: 在WASM Go插件单元测试工作流中添加了覆盖率门控功能，包括详细的覆盖率信息显示和80%的覆盖率阈值设定。 \
  **Feature Value**: 通过提高CI流程中的覆盖率要求，确保WASM Go插件的质量和稳定性，有助于开发者及时发现潜在问题。

---

## 📊 发布统计

- 🚀 新功能: 23项
- 🐛 Bug修复: 13项
- ♻️ 重构优化: 3项
- 📚 文档更新: 1项
- 🧪 测试改进: 4项

**总计**: 44项更改（包含3项重要更新）

感谢所有贡献者的辛勤付出！🎉


# Higress Console


## 📋 本次发布概览

本次发布包含 **18** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 7项
- **Bug修复**: 10项
- **文档更新**: 1项

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#621](https://github.com/higress-group/higress-console/pull/621) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: 此PR优化了MCP Server的部分交互能力，包括直接路由场景下的header重写、支持选择transport类型以及DB to MCP Server场景下的特殊字符处理。 \
  **Feature Value**: 通过改进MCP Server的交互方式和增强其处理能力，提高了系统的灵活性与兼容性，使得用户能够更方便地配置和使用后端服务，从而提升了用户体验。

- **Related PR**: [#612](https://github.com/higress-group/higress-console/pull/612) \
  **Contributor**: @zhwaaaaaa \
  **Change Log**: 此PR通过添加对hop-to-hop头部的忽略，解决了因反向代理服务器发送transfer-encoding: chunked头而导致Grafana页面无法正常工作的问题。 \
  **Feature Value**: 该功能确保了即使在复杂的网络环境中（如使用反向代理），Grafana监控面板也能正确显示信息，提升了用户体验和系统的兼容性。

- **Related PR**: [#608](https://github.com/higress-group/higress-console/pull/608) \
  **Contributor**: @Libres-coder \
  **Change Log**: 该PR为AI路由管理页面添加了插件显示支持，用户可以查看已启用的插件并扩展AI路由行以获取更多信息。 \
  **Feature Value**: 增强了AI路由管理功能，使得用户能够更直观地理解和管理其AI路由配置中的插件状态，提升了用户体验。

- **Related PR**: [#604](https://github.com/higress-group/higress-console/pull/604) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR通过添加`higress.io/rewrite-target`注解支持使用正则表达式进行路径重写，增强了路径重写的灵活性。 \
  **Feature Value**: 增加了路径重写的灵活性，允许用户利用更复杂的规则来修改请求路径，提高了系统的可定制性和用户体验。

- **Related PR**: [#603](https://github.com/higress-group/higress-console/pull/603) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR在前端页面中添加了固定的服务端口80显示功能，适用于静态服务源。通过定义常量并更新表单组件以展示该端口号。 \
  **Feature Value**: 为用户提供了一个更直观的视图来识别和确认静态服务源所使用的标准HTTP端口（80），简化了配置过程并减少了潜在误解。

- **Related PR**: [#602](https://github.com/higress-group/higress-console/pull/602) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR为AI路由选择上游服务时添加了搜索功能，通过在前端组件中引入搜索机制，提高了用户查找所需服务的效率。 \
  **Feature Value**: 新增的搜索功能显著提升了用户体验，尤其是在服务列表较长的情况下，能够帮助用户快速定位到目标服务，简化了配置流程。

- **Related PR**: [#566](https://github.com/higress-group/higress-console/pull/566) \
  **Contributor**: @OuterCyrex \
  **Change Log**: 本PR新增了对自定义Qwen服务的支持，包括启用互联网搜索、上传文件ID等功能。主要变更涉及后端SDK和前端页面以支持这些新功能。 \
  **Feature Value**: 通过增加对自定义Qwen服务的支持，用户现在可以更灵活地配置其AI服务，特别是对于需要特定功能如互联网搜索或文件处理的应用场景，极大地提升了用户体验和服务的可定制性。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#620](https://github.com/higress-group/higress-console/pull/620) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修正了sortWasmPluginMatchRules逻辑中的拼写错误，确保匹配规则排序正确无误。 \
  **Feature Value**: 通过修复拼写错误提高了代码准确性和可读性，避免了因拼写问题导致的潜在逻辑错误，从而提升了系统的稳定性和用户体验。

- **Related PR**: [#619](https://github.com/higress-group/higress-console/pull/619) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR移除了从AiRoute转换到ConfigMap时数据JSON中的版本信息，因为这些信息已经在ConfigMap的元数据中保存。 \
  **Feature Value**: 移除冗余的版本信息有助于减少数据重复，并确保ConfigMap中存储的信息更加简洁明了，提高维护性和一致性。

- **Related PR**: [#618](https://github.com/higress-group/higress-console/pull/618) \
  **Contributor**: @CH3CHO \
  **Change Log**: PR重构了SystemController中的API认证逻辑，以消除已知的安全漏洞。通过引入新的AllowAnonymous注解并更新相关控制器，确保了系统的安全性。 \
  **Feature Value**: 此次修复消除了系统中存在的安全漏洞，提升了整体的安全性，保护用户免受潜在攻击威胁，增强了用户的信任度与使用体验。

- **Related PR**: [#617](https://github.com/higress-group/higress-console/pull/617) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了前端控制台中关于列表项缺少唯一key属性的警告、图片加载违反内容安全策略的问题以及Consumer.name字段类型错误。 \
  **Feature Value**: 解决了用户在使用过程中遇到的前端错误，提升了用户体验和系统的稳定性。

- **Related PR**: [#614](https://github.com/higress-group/higress-console/pull/614) \
  **Contributor**: @lc0138 \
  **Change Log**: 此PR修正了ServiceSource类型字段中的错误，并添加了字典值校验逻辑，确保该字段的准确性。 \
  **Feature Value**: 修复服务来源类型字段的错误提升了系统的稳定性和数据的一致性，防止因类型不匹配导致的问题。

- **Related PR**: [#613](https://github.com/higress-group/higress-console/pull/613) \
  **Contributor**: @lc0138 \
  **Change Log**: 此PR针对前端文档中的CSP等安全风险进行了修复，通过增加特定的meta标签加强了网页的安全性。 \
  **Feature Value**: 增强了Web应用的安全防护能力，减少了潜在的安全威胁，提升了用户的使用安全性。

- **Related PR**: [#611](https://github.com/higress-group/higress-console/pull/611) \
  **Contributor**: @qshuai \
  **Change Log**: 修正了LlmProvidersController类中API方法注解的拼写错误，将'Add a new route'更正为正确的描述。 \
  **Feature Value**: 虽然只是一项小修复，但确保了API文档的准确性，有助于开发者更好地理解和使用API接口。

- **Related PR**: [#609](https://github.com/higress-group/higress-console/pull/609) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修正了Consumer接口中name字段的类型错误，从布尔值更改为字符串。 \
  **Feature Value**: 此修复确保了Consumer.name字段的数据类型正确无误，避免因类型不匹配导致的应用程序运行时错误，提升了系统的稳定性和数据准确性。

- **Related PR**: [#605](https://github.com/higress-group/higress-console/pull/605) \
  **Contributor**: @SaladDay \
  **Change Log**: 修正了AI路由名称验证规则，使其支持点号并仅允许小写字母，同时更新了错误提示信息以准确描述新的验证规则。 \
  **Feature Value**: 此修复解决了界面提示与实际验证逻辑不一致的问题，提高了用户体验的一致性和准确性，确保用户在配置AI路由时能够获得正确的反馈。

- **Related PR**: [#552](https://github.com/higress-group/higress-console/pull/552) \
  **Contributor**: @lcfang \
  **Change Log**: 为解决服务后端端口不一致导致的兼容性问题，新增vport属性。当注册中心中的服务实例端口发生变化时，通过配置vport默认端口或指定虚拟端口来保持路由配置的有效性。 \
  **Feature Value**: 该PR增强系统的稳定性和可靠性，确保在服务端口变化时路由配置不会失效，从而提升用户体验和系统可用性。

### 📚 文档更新 (Documentation)

- **Related PR**: [#610](https://github.com/higress-group/higress-console/pull/610) \
  **Contributor**: @heimanba \
  **Change Log**: 该PR调整了前端灰度文档中的配置字段说明，包括将rewrite等字段改为非必填，并更新了rules中name字段的关联说明，同步更新了中英文README和spec.yaml文件。 \
  **Feature Value**: 通过增加配置项的灵活性并提高兼容性，使得用户能够更方便地根据实际需求进行配置。同时，文档的一致性和准确性得到提升，有助于降低用户的使用难度，提高用户体验。

---

## 📊 发布统计

- 🚀 新功能: 7项
- 🐛 Bug修复: 10项
- 📚 文档更新: 1项

**总计**: 18项更改

感谢所有贡献者的辛勤付出！🎉


