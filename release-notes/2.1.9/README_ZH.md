# Higress


## 📋 本次发布概览

本次发布包含 **44** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 23项
- **Bug修复**: 14项
- **重构优化**: 2项
- **文档更新**: 1项
- **测试改进**: 4项

### ⭐ 重点关注

本次发布包含 **3** 项重要更新，建议重点关注：

- **feat(mcp-server): add server-level default authentication and MCP proxy server support** ([#3096](https://github.com/alibaba/higress/pull/3096)): 此特性增强了Higress对MCP流量的安全管理能力，允许用户通过统一接口进行认证设置，简化了安全策略的部署流程，提升了系统的安全性与灵活性。
- **feat: add higress api mcp server** ([#2923](https://github.com/alibaba/higress/pull/2923)): 通过添加higress-ops MCP Server，用户可以使用hgctl agent命令来管理Higress配置和排查问题，提升了运维效率和用户体验。
- **feat: implement `hgctl agent` & `mcp add` subcommand ** ([#3051](https://github.com/alibaba/higress/pull/3051)): 增强了Higress的运维能力，特别是通过Agent进行交互式管理和调试的能力，使得用户能够更便捷地配置和调试MCP流量治理，是Higress向AI原生运维迈进的重要一步。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat(mcp-server): add server-level default authentication and MCP proxy server support

**相关PR**: [#3096](https://github.com/alibaba/higress/pull/3096) | **贡献者**: [@johnlanni](https://github.com/johnlanni)

**使用背景**

随着AI原生API网关Higress的发展，用户对API安全性、灵活性以及易用性的要求越来越高。在实际应用中，MCP（Model Context Protocol）协议被广泛用于管理和调用AI模型。然而，现有的MCP服务器缺乏统一的安全认证机制，导致在不同场景（MCP Server直接代理，REST API转换MCP Server）下需要配置差异化的认证机制。本次更新解决了这些问题，目标用户群体包括但不限于开发者、运维人员和系统管理员，他们需要一个更加安全、灵活且易于管理的API网关。

**功能详述**

此次更新主要实现了两个核心功能：1. 在MCP服务器级别添加了默认认证机制，包括客户端到网关及网关到后端的认证；2. 引入了一种新的MCP代理服务器类型，可以将客户端的MCP请求代理到后端的MCP服务器，并支持超时配置和完整的认证支持。从技术实现上来看，主要通过更新依赖库版本（如wasm-go 和 proxy-wasm-go-sdk）来支持新功能，同时对现有代码进行了重构以适应新的认证和代理逻辑。

**使用方式**

启用此功能需在Higress配置文件中设置相应的参数。例如，要配置默认的下游安全认证，可以在`defaultDownstreamSecurity`字段中指定认证策略；类似地，上游认证则通过`defaultUpstreamSecurity`字段进行配置。若要使用MCP代理服务器，需定义一个新的`mcp-proxy`类型的服务器，并通过`mcpServerURL`指定后端MCP服务器地址。此外，还可以通过`timeout`字段控制请求超时时间。最佳实践建议尽可能利用优先级配置机制，确保工具级设置能覆盖服务器级别默认值，从而获得更细粒度的控制。

**功能价值**

该功能显著提高了Higress的安全性和灵活性，使得API管理变得更加高效。通过引入服务器级别的默认认证，减少了重复配置工作量，降低了因配置错误引发的安全风险。MCP代理服务器不仅简化了REST to MCP转换过程中的复杂性，还通过卸载状态保持任务至Higress侧，有效减轻了后端MCP服务器的压力。这些改进共同作用于提升整个生态系统的稳定性和用户体验，为Higress成为AI时代不可或缺的API网关奠定了坚实基础。

---

### 2. feat: add higress api mcp server

**相关PR**: [#2923](https://github.com/alibaba/higress/pull/2923) | **贡献者**: [@Tsukilc](https://github.com/Tsukilc)

**使用背景**

随着AI技术的发展，API网关需要更好地支持AI相关的功能。Higress作为一个AI原生的API网关，需要提供更强大的管理工具来统一管理LLM API、MCP API和Agent API等核心API资产。本次PR通过集成Higress API MCP Server，提供了对AI路由、AI提供商和MCP服务器的全面管理能力。这些新功能可以帮助用户更高效地配置和维护Higress的AI特性，满足现代应用的需求。目标用户群体包括Higress的运维人员和开发人员，尤其是那些在AI领域有深入需求的用户。

**功能详述**

该PR主要实现了以下功能：
1. **AI路由管理**：新增了`list-ai-routes`、`get-ai-route`、`add-ai-route`、`update-ai-route`和`delete-ai-route`等工具，允许用户管理AI路由。
2. **AI提供商管理**：新增了`list-ai-providers`、`get-ai-provider`、`add-ai-provider`、`update-ai-provider`和`delete-ai-provider`等工具，允许用户管理AI提供商。
3. **MCP服务器管理**：新增了`list-mcp-servers`、`get-mcp-server`、`add-or-update-mcp-server`、`delete-mcp-server`等工具，允许用户管理MCP服务器及其消费者。
4. **认证配置**：使用HTTP Basic Authentication进行鉴权，并在客户端请求头中携带`Authorization`头。
5. **代码变更**：移除了用户名和密码的硬编码，改为在运行时通过MCP Client提供，提高了安全性。同时，新增了`higress-ops`模块，用于hgctl agent命令对接，实现Agent方式管理Higress的配置。

**使用方式**

要启用和配置这个功能，请按照以下步骤操作：
1. **配置Higress API MCP Server**：在Higress配置文件中添加Higress API MCP Server的配置，指定Higress Console的URL地址。
2. **使用hgctl agent**：通过`hgctl agent`命令启动交互式Agent，可以使用自然语言的方式管理Higress。例如，使用`mcp add`子命令添加remote MCP Server到Higress的MCP管理目录中。
3. **管理AI路由**：使用`list-ai-routes`、`get-ai-route`、`add-ai-route`、`update-ai-route`和`delete-ai-route`等工具来管理AI路由。
4. **管理AI提供商**：使用`list-ai-providers`、`get-ai-provider`、`add-ai-provider`、`update-ai-provider`和`delete-ai-provider`等工具来管理AI提供商。
5. **管理MCP服务器**：使用`list-mcp-servers`、`get-mcp-server`、`add-or-update-mcp-server`、`delete-mcp-server`等工具来管理MCP服务器及其消费者。
注意事项：确保在使用这些工具时，正确配置鉴权信息，并在请求头中携带`Authorization`头。

**功能价值**

该功能为用户带来了以下具体好处：
1. **增强的管理能力**：用户可以通过新的MCP工具更方便地管理和调试Higress的AI路由、AI提供商和MCP服务器配置，提高了管理效率。
2. **更高的安全性**：通过在运行时通过MCP Client提供用户名和密码，而不是硬编码在配置文件中，提高了系统的安全性。
3. **更好的用户体验**：通过hgctl agent的交互式管理方式，用户可以使用自然语言的方式管理Higress，降低了学习成本和使用难度。
4. **系统性能和稳定性提升**：新的MCP工具提供了更多的管理和调试手段，有助于及时发现和解决问题，提高系统的稳定性和性能。
5. **生态重要性**：作为Higress从传统运维方式走向借助Agent运维方式迈出的第一步，该功能对于Higress生态的发展具有重要意义，为未来的更多创新打下了基础。

---

### 3. feat: implement `hgctl agent` & `mcp add` subcommand 

**相关PR**: [#3051](https://github.com/alibaba/higress/pull/3051) | **贡献者**: [@erasernoob](https://github.com/erasernoob)

**使用背景**

Higress 是一个AI原生的API网关，用于统一管理LLM API、MCP API和Agent API。随着Higress的发展，传统的命令行工具已经不能满足用户的需求，特别是在MCP服务的管理和调试方面。本次PR引入了类似Claude Code的交互式Agent，使得用户可以通过自然语言来管理Higress。同时，新增的`mcp add`子命令可以方便地将远程MCP服务添加到Higress的MCP管理目录中，实现MCP流量的治理。这些功能不仅简化了MCP服务的配置过程，还增强了系统的可维护性和易用性。

**功能详述**

本次PR主要实现了两个新的子命令：`hgctl agent` 和 `mcp add`。

- `hgctl agent`：这个命令允许用户通过自然语言与Higress进行交互。它会调用底层的`claude-code`代理，并在首次使用时提示用户设置必要的环境。`hgctl agent`提供了一个交互式的窗口，使用户能够以更直观的方式管理Higress。

- `mcp add`：这个命令允许用户通过简单的参数添加MCP服务。支持两种类型的MCP服务：直接代理类型和基于OpenAPI的类型。直接代理类型的MCP服务可以直接调用Higress Console API并发布到Higress MCP Server管理工具中。基于OpenAPI的MCP服务则通过解析OpenAPI规范来生成MCP配置。代码变更中，新增了多个文件和大量的代码，包括`agent.go`、`base.go`、`core.go`、`mcp.go`和`client.go`，这些文件共同实现了上述功能。

**使用方式**

要启用和配置这些新功能，用户需要更新到最新版本的`hgctl`工具。

1. **启用`hgctl agent`**：
   - 运行`hgctl agent`命令，首次使用时会提示用户设置必要的环境，如安装`claude-code`代理。
   - 通过自然语言与Higress进行交互，例如查询或修改配置。

2. **使用`mcp add`添加MCP服务**：
   - 添加直接代理类型的MCP服务：
     ```bash
     hgctl mcp add mcp-deepwiki -t http https://mcp.deepwiki.com --user admin --password 123 --url http://localhost:8080
     ```
   - 添加基于OpenAPI的MCP服务：
     ```bash
     hgctl mcp add openapi-server -t openapi --spec openapi.yaml --user admin --password 123 --url http://localhost:8080
     ```

注意事项：确保在运行这些命令之前，系统已经正确配置了Higress和相关依赖。

**功能价值**

这些新功能为用户带来了显著的好处，包括：

- **提升用户体验**：通过自然语言交互，降低了用户的学习曲线，使得Higress的管理更加直观和友好。
- **简化配置过程**：`mcp add`命令极大地简化了MCP服务的添加和配置过程，减少了手动操作的复杂性和出错率。
- **增强系统稳定性**：通过统一的MCP服务管理，可以更好地监控和维护MCP流量，提高系统的稳定性和可靠性。
- **扩展生态系统**：这些新功能使得Higress能够更好地支持不同的MCP服务类型，增强了其在AI时代的竞争力和生态影响力。

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#3126](https://github.com/alibaba/higress/pull/3126) \
  **Contributor**: @johnlanni \
  **Change Log**: 更新了Envoy依赖，支持通过WASM设置Redis调用相关的参数，如buffer_flush_timeout和max_buffer_size_before_flush。 \
  **Feature Value**: 此功能增强了WASM插件的灵活性，允许用户通过URL查询参数自定义Redis客户端缓冲行为，提升了配置管理的便利性和效率。

- **Related PR**: [#3123](https://github.com/alibaba/higress/pull/3123) \
  **Contributor**: @johnlanni \
  **Change Log**: 升级了Higress代理版本至v2.2.0，更新Go工具链及多个依赖包版本，并为golang-filter添加了特定架构的构建目标，修复了与MCP服务器、OpenAI和Milvus SDK相关的依赖问题。 \
  **Feature Value**: 提升了Higress的整体性能与稳定性，支持更多架构类型，同时增强了对最新技术栈的支持能力。对于用户而言，这意味着更广泛的兼容性、更好的安全性和更丰富的功能扩展可能性。

- **Related PR**: [#3108](https://github.com/alibaba/higress/pull/3108) \
  **Contributor**: @wydream \
  **Change Log**: 新增了与视频相关的API路径和能力，包括常量、默认能力和正则表达式路径处理，使代理能够正确解析多个视频相关端点，并更新OpenAI提供者以优化对这些新端点的支持。 \
  **Feature Value**: 通过增加视频相关的API支持，增强了Higress管理AI服务的能力，特别是对于需要处理视频内容的应用场景，这将让用户能够更方便地集成和使用涉及视频的高级功能。

- **Related PR**: [#3071](https://github.com/alibaba/higress/pull/3071) \
  **Contributor**: @rinfx \
  **Change Log**: PR添加了`inject_encoded_data_to_filter_chain_on_header`函数的使用示例，展示了如何在无响应body情况下为请求添加body数据。通过修改README.md、go.mod等文件实现。 \
  **Feature Value**: 此功能允许用户在没有响应body的情况下向请求添加body数据，增强了API网关处理请求的能力和灵活性，特别是在需要动态生成或修改响应内容时提供了更多的可能性。

- **Related PR**: [#3067](https://github.com/alibaba/higress/pull/3067) \
  **Contributor**: @wydream \
  **Change Log**: 此PR为Higress的ai-proxy插件新增了对vLLM作为AI提供商的支持，实现了与OpenAI兼容的多个API接口，包括聊天和文本补全、模型列表展示等功能。 \
  **Feature Value**: 通过引入vLLM作为新的AI服务提供者，用户现在可以直接通过Higress代理访问vLLM提供的各种AI能力，如生成文本等，这极大地丰富了Higress在AI应用场景中的可用性，简化了集成流程。

- **Related PR**: [#3060](https://github.com/alibaba/higress/pull/3060) \
  **Contributor**: @erasernoob \
  **Change Log**: 此PR通过增强`hgctl mcp`和`hgctl agent`命令实现了从安装配置文件及Kubernetes secrets中自动获取Higress Console凭据的功能，优化了用户的使用体验。 \
  **Feature Value**: 该功能减少了用户手动输入凭据的步骤，提升了操作便捷性和安全性，特别是在Higress通过hgctl安装的情况下，对用户来说是一个重要的便利性改进。

- **Related PR**: [#3043](https://github.com/alibaba/higress/pull/3043) \
  **Contributor**: @2456868764 \
  **Change Log**: 此PR修复了Milvus默认端口错误的问题，并在README.md中添加了Python示例代码。通过修改配置文件中的match_rule_domain字段解决了端口问题，同时提供了使用指导。 \
  **Feature Value**: 修复了可能导致服务无法正确运行的端口配置问题，增强了文档的实用性，为用户提供了一个具体的Python示例来帮助理解和快速上手使用插件功能。

- **Related PR**: [#3040](https://github.com/alibaba/higress/pull/3040) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: 此PR新增了Anthropic的ApiNameAnthropicMessages功能，并支持在没有设置protocol=original的情况下配置anthropic提供商，让/v1/messages请求直接转发给anthropic，而/v1/chat/completion则会将OpenAI格式的消息体转换为Claude兼容的格式。 \
  **Feature Value**: 通过增加对Anthropic消息API的支持，提升了Higress管理不同类型AI服务的能力。用户现在可以更灵活地使用Anthropic提供的服务，特别是在需要与Claude进行交互时变得更加方便，增强了平台的多样性和灵活性。

- **Related PR**: [#3038](https://github.com/alibaba/higress/pull/3038) \
  **Contributor**: @Libres-coder \
  **Change Log**: 新增了`list-plugin-instances`工具，允许AI代理通过MCP协议查询特定作用域下的插件实例。该PR向MCP Server添加了两个新函数，并在文档中更新了相关说明。 \
  **Feature Value**: 此功能使用户能够更方便地管理Higress中的插件配置，增强了系统的可管理性和透明度，特别是在需要对特定范围内的插件状态进行检查或调整时。

- **Related PR**: [#3032](https://github.com/alibaba/higress/pull/3032) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR默认启用了Qwen兼容模式，并添加了缺失的API端点，包括AsyncAIGC、AsyncTask和V1Rerank，以提供更完整的API覆盖。 \
  **Feature Value**: 通过默认启用兼容模式并填补API接口缺口，提升了用户的开箱即用体验，同时增强了系统对于Qwen AI服务的支持力度，使开发者能够更加便捷地接入和使用Qwen相关功能。

- **Related PR**: [#3029](https://github.com/alibaba/higress/pull/3029) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: 在groq提供者中添加了对v1/responses的支持，具体改动包括引入新的响应处理逻辑。 \
  **Feature Value**: 新增的功能支持使得用户能够通过Higress更好地管理和利用groq插件提供的服务响应，提升了API管理的灵活性和功能完整性。

- **Related PR**: [#3024](https://github.com/alibaba/higress/pull/3024) \
  **Contributor**: @rinfx \
  **Change Log**: 新增恶意URL和模型幻觉检测功能，确保AI生成内容的安全性；同时调整了消费者级别的特定配置，以更好地适应不同场景下的需求。 \
  **Feature Value**: 通过增加对恶意URL及模型幻觉的检测，提升了Higress平台处理AI生成内容时的安全性和准确性，有助于保护用户免受潜在威胁。此外，调整后的消费者级配置增强了系统的灵活性与适应性。

- **Related PR**: [#3008](https://github.com/alibaba/higress/pull/3008) \
  **Contributor**: @hellocn9 \
  **Change Log**: 本次PR为MCP SSE stateful sessions增加了自定义参数名支持。通过在ingress配置中新增`higress.io/mcp-sse-stateful-param-name`注解，用户可以指定自己的参数名称。 \
  **Feature Value**: 此功能允许用户根据自身需求灵活设置MCP SSE stateful会话的参数名称，提高了配置灵活性和用户体验。这使得Higress能更好地适应多样化的应用场景。

- **Related PR**: [#3006](https://github.com/alibaba/higress/pull/3006) \
  **Contributor**: @SaladDay \
  **Change Log**: 此PR为MCP Server的Redis配置添加了Secret引用支持，允许通过Kubernetes Secret来存储Redis密码，增强了安全性，并修改了相关文档和测试代码。 \
  **Feature Value**: 通过使用Kubernetes Secret存储Redis密码而不是明文写入ConfigMap，提高了系统的安全性。用户可以更加安全地管理敏感信息，降低了密码泄露的风险。

- **Related PR**: [#2992](https://github.com/alibaba/higress/pull/2992) \
  **Contributor**: @rinfx \
  **Change Log**: 该PR修改了key_auth插件中的认证逻辑，在日志中记录消费者名称，即使未被授权访问。通过在认证鉴权过程中增加对消费者识别的日志记录，增强了系统的可观察性。 \
  **Feature Value**: 这项功能提高了系统监控和故障排查的效率，允许运营人员更清晰地了解请求来源，即便请求未被授权也能追踪到具体的消费者，从而更好地进行问题诊断和安全审计。

- **Related PR**: [#2978](https://github.com/alibaba/higress/pull/2978) \
  **Contributor**: @rinfx \
  **Change Log**: 在确定消费者名称后，无论认证是否通过，都将消费者名称添加到请求头中，以供后续处理使用。 \
  **Feature Value**: 此功能增强了对消费者行为的追踪能力，有助于更好地理解API调用情况及消费者活动模式，从而为用户提供更加个性化的服务体验。

- **Related PR**: [#2968](https://github.com/alibaba/higress/pull/2968) \
  **Contributor**: @2456868764 \
  **Change Log**: 增加了向量数据库映射功能，引入了字段映射系统与索引配置管理机制，支持多种索引类型，如HNSW、IVF、SCANN等，以提高系统的灵活性和适应性。 \
  **Feature Value**: 通过提供灵活的字段映射及丰富的索引配置选项，增强了对不同向量数据库的支持能力，简化了开发者集成多样化存储方案的过程，提升了用户体验。

- **Related PR**: [#2943](https://github.com/alibaba/higress/pull/2943) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: 此PR增加了在生成发布说明时支持自定义系统提示的功能，通过修改GitHub Actions工作流文件实现。 \
  **Feature Value**: 允许用户在生成发布说明时添加个性化系统提示，提高了发布说明的灵活性和实用性，更好地满足不同项目的需求。

- **Related PR**: [#2942](https://github.com/alibaba/higress/pull/2942) \
  **Contributor**: @2456868764 \
  **Change Log**: 修复了LLM提供者为空时的处理逻辑，并优化了文档结构与内容，增加了对MCP工具的详细介绍。 \
  **Feature Value**: 提高了系统在LLM配置缺失时的健壮性，增强了用户对MCP工具的理解和使用体验，使用户能够更清晰地了解不同工具的功能及其配置需求。

- **Related PR**: [#2916](https://github.com/alibaba/higress/pull/2916) \
  **Contributor**: @imp2002 \
  **Change Log**: 实现了Nginx迁移MCP服务器，并提供了7种MCP工具来自动化从Nginx配置/Lua插件迁移到Higress的过程。包括了配置转换等重要功能。 \
  **Feature Value**: 该功能极大简化了用户从Nginx向Higress迁移的工作量，通过提供一套完整的工具集使得迁移过程更加平滑和高效，有助于用户更快速地采用Higress作为其API网关解决方案。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#3120](https://github.com/alibaba/higress/pull/3120) \
  **Contributor**: @lexburner \
  **Change Log**: 调整了ai-proxy组件中的日志级别，具体修改位于wasm-go/extensions/ai-proxy/provider/qwen.go文件中，减少了不必要的警告信息输出。 \
  **Feature Value**: 通过降低特定部分的日志级别，减少了系统运行时产生的冗余警告信息，有助于提高开发和运维人员查看日志的效率，使他们能够更专注于真正的错误或重要信息。

- **Related PR**: [#3119](https://github.com/alibaba/higress/pull/3119) \
  **Contributor**: @johnlanni \
  **Change Log**: 更新istio依赖，并将Connection中的reqChan和deltaReqChan替换为channels.Unbounded，以防止HTTP2流控导致的死锁问题。 \
  **Feature Value**: 通过解决HTTP2流控引起的死锁问题，确保了客户端请求和ACK请求可以正常无阻塞地处理，提升了系统的稳定性和响应速度。

- **Related PR**: [#3118](https://github.com/alibaba/higress/pull/3118) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR修复了端口级别策略无条件覆盖已存在的由Ingress注解转换而来的配置的问题。通过在设置policy.Tls和policy.LoadBalancer前添加nil检查来避免覆写现有配置。 \
  **Feature Value**: 解决了因DestinationRule中的TLS及负载均衡器设置导致的配置意外覆盖问题，确保了用户自定义的Ingress注解配置能够被正确保留并应用，增强了系统的稳定性和可靠性。

- **Related PR**: [#3095](https://github.com/alibaba/higress/pull/3095) \
  **Contributor**: @rinfx \
  **Change Log**: 修复了claude2openai转换过程中usage信息丢失的问题，并在bedrock流式工具响应中添加了index字段，以确保数据完整性和准确性。 \
  **Feature Value**: 该修复提升了系统在处理API转换时的数据完整性，确保用户能够准确获取到所有必要的使用信息，特别是在涉及到流式响应的情况下，通过引入index字段增强了响应管理的灵活性。

- **Related PR**: [#3084](https://github.com/alibaba/higress/pull/3084) \
  **Contributor**: @rinfx \
  **Change Log**: 修复了Claude请求转换为OpenAI请求时未正确包含include_usage: true的问题，确保在流式响应模式下能够正常传递使用情况信息。 \
  **Feature Value**: 该修复使得用户可以在使用流式API处理时获得更准确的资源使用反馈，提升了系统对资源消耗监控的准确性。

- **Related PR**: [#3074](https://github.com/alibaba/higress/pull/3074) \
  **Contributor**: @Jing-ze \
  **Change Log**: 此PR在log-request-response插件中添加了对Content-Encoding的检查，从而避免记录压缩后的请求/响应体导致日志出现乱码的情况。 \
  **Feature Value**: 通过改进日志记录机制来防止输出难以阅读的日志条目，提高了系统运维人员排查问题时的工作效率和准确性。

- **Related PR**: [#3069](https://github.com/alibaba/higress/pull/3069) \
  **Contributor**: @Libres-coder \
  **Change Log**: 本PR修复了CI测试框架中的一个问题，即由于`go.mod`文件未被正确更新而导致的e2e测试失败。通过在`prebuild.sh`脚本中添加`go mod tidy`命令来确保根目录下的`go.mod`也得到更新。 \
  **Feature Value**: 该修复解决了所有触发wasm插件端到端测试的PR都可能遇到的CI测试失败问题，保证了构建和测试流程的稳定性，提升了开发者的体验。

- **Related PR**: [#3010](https://github.com/alibaba/higress/pull/3010) \
  **Contributor**: @rinfx \
  **Change Log**: 修复了bedrock返回响应时因拆包问题导致的解析失败情况，并调整了maxtoken转换逻辑，确保了事件流处理的准确性和完整性。 \
  **Feature Value**: 解决了用户在使用bedrock服务时遇到的数据解析错误问题，提升了系统的稳定性和用户体验。通过优化边界条件处理，保证了数据传输的一致性。

- **Related PR**: [#2997](https://github.com/alibaba/higress/pull/2997) \
  **Contributor**: @hanxiantao \
  **Change Log**: 优化了集群限流和AI Token限流的逻辑，调整为累加方式统计请求次数和token使用量，避免在修改限流值时重置计数。 \
  **Feature Value**: 通过改进限流机制，确保即使在更改限流阈值后，系统依然能够准确地追踪和限制请求流量，从而提高了系统的稳定性和可靠性。

- **Related PR**: [#2988](https://github.com/alibaba/higress/pull/2988) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR修复了jsonrpc-converter中JSON格式化错误的问题，改为使用原始JSON数据，避免了由于字符串格式化导致的数据解析问题。 \
  **Feature Value**: 通过修正JSON处理方式，确保了数据传输的准确性和一致性，提升了系统的稳定性和可靠性，减少了因数据格式错误引发的潜在问题。

- **Related PR**: [#2973](https://github.com/alibaba/higress/pull/2973) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了当`match_rule_domain`设置为空字符串时Higress 2.1.8版本不支持的问题，通过使用通配符匹配所有域来消除兼容性风险。 \
  **Feature Value**: 此修复确保了MCP服务器配置的生成与旧版本向后兼容，避免因配置错误导致的服务中断或行为异常，提升了系统的稳定性和用户体验。

- **Related PR**: [#2952](https://github.com/alibaba/higress/pull/2952) \
  **Contributor**: @Erica177 \
  **Change Log**: 修正了 ToolSecurity 结构体中字段 Id 的 JSON 标签，从 type 更改为 id，以确保序列化正确。 \
  **Feature Value**: 此修复确保了 ToolSecurity 结构体在数据传输时的正确性，避免因字段标签错误导致的数据解析问题，提升了系统的稳定性和用户体验。

- **Related PR**: [#2948](https://github.com/alibaba/higress/pull/2948) \
  **Contributor**: @johnlanni \
  **Change Log**: 修复了Azure OpenAI Response API的处理问题和服务URL类型检测逻辑，包括添加对自定义完整路径的支持和改进流式事件解析。 \
  **Feature Value**: 增强了Azure OpenAI服务的支持，提高了API响应处理的准确性与效率，使得用户能够更稳定地使用Azure OpenAI相关功能。

- **Related PR**: [#2941](https://github.com/alibaba/higress/pull/2941) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR修复了ai-security-guard插件与旧配置不兼容的问题，通过调整`main.go`文件中的相关代码来确保向后兼容性。 \
  **Feature Value**: 解决了因配置更新导致的兼容性问题，使得使用旧版配置的用户能够无缝过渡到新版本，提升了用户体验和系统的稳定性。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#3113](https://github.com/alibaba/higress/pull/3113) \
  **Contributor**: @johnlanni \
  **Change Log**: 该PR实现了Protobuf消息的哈希缓存功能，通过xxHash算法进行递归哈希计算，并对google.protobuf.Any类型和具有确定性排序的map字段进行了特殊处理，优化了LDS性能。 \
  **Feature Value**: 此改动显著提升了Envoy在处理大量配置更新时的效率，减少了因重复序列化导致的性能开销，特别是在频繁变更或大规模部署环境中，能够加速配置传播速度，提高系统响应能力。

- **Related PR**: [#2945](https://github.com/alibaba/higress/pull/2945) \
  **Contributor**: @rinfx \
  **Change Log**: 优化了ai-load-balancer中全局最小请求数选pod的Lua脚本逻辑，通过调整健康检查机制和负载均衡策略提高了请求分发效率。 \
  **Feature Value**: 此次改动提升了AI负载均衡器对于请求处理的公平性和效率，减少了因单一节点过载导致的服务响应时间延长问题，对提高整体系统的稳定性和用户体验有正面影响。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2965](https://github.com/alibaba/higress/pull/2965) \
  **Contributor**: @CH3CHO \
  **Change Log**: 更新了ai-proxy README文件中azureServiceUrl的描述，增加了关于该参数使用的详细信息，以帮助用户更好地理解和配置。 \
  **Feature Value**: 通过提供更详细的azureServiceUrl参数说明，此更改有助于改善用户体验，使用户能够更容易地根据文档进行正确的配置设置，从而避免潜在的使用错误。

### 🧪 测试改进 (Testing)

- **Related PR**: [#3110](https://github.com/alibaba/higress/pull/3110) \
  **Contributor**: @Jing-ze \
  **Change Log**: 本PR在GitHub Actions工作流中增加了CODECOV_TOKEN环境变量配置，以确保Codecov能够正确地进行身份验证并上传代码覆盖率数据。 \
  **Feature Value**: 通过添加CODECOV_TOKEN环境变量，提升了CI/CD流程中的安全性与可靠性，保证了代码覆盖率报告的准确性和完整性，有助于维护项目质量。

- **Related PR**: [#3097](https://github.com/alibaba/higress/pull/3097) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR为mcp-server添加了单元测试，总共增加了2766行代码，主要集中在main_test.go文件中，增强了mcp-server的测试覆盖率。 \
  **Feature Value**: 通过增加单元测试，提高了mcp-server模块的稳定性与可靠性，确保新功能或修复不会引入新的问题。对于用户而言，这提升了Higress整体的质量保证和使用体验。

- **Related PR**: [#2998](https://github.com/alibaba/higress/pull/2998) \
  **Contributor**: @Patrisam \
  **Change Log**: 本PR实现了针对Cloudflare的端到端测试案例，增强了Higress项目的测试覆盖范围。通过在go-wasm-ai-proxy.go和go-wasm-ai-proxy.yaml中添加新的测试逻辑及配置，提高了系统集成度。 \
  **Feature Value**: 新增加的Cloudflare e2e测试案例有助于确保Higress与Cloudflare服务之间的兼容性和稳定性，对于使用或计划使用Cloudflare作为其网络基础设施一部分的用户来说，这将极大提升他们对Higress可靠性的信心。

- **Related PR**: [#2980](https://github.com/alibaba/higress/pull/2980) \
  **Contributor**: @Jing-ze \
  **Change Log**: 增强了WASM插件单元测试的CI工作流，添加了覆盖率显示功能并设置了80%的覆盖率阈值。 \
  **Feature Value**: 提高了测试流程的质量和透明度，确保WASM插件满足一定的代码覆盖率标准，有助于发现潜在问题，提高代码可靠性。

---

## 📊 发布统计

- 🚀 新功能: 23项
- 🐛 Bug修复: 14项
- ♻️ 重构优化: 2项
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
  **Change Log**: 该PR优化了MCP Server的交互能力，包括重写header host、修改交互方式以支持选择transport，并改进了DSN字符处理逻辑，支持特殊字符@。 \
  **Feature Value**: 通过这些改进，用户可以更灵活地配置和使用MCP Server，特别是在直接路由场景下能更好地处理DNS地址和服务路径，提高了系统的灵活性和易用性。

- **Related PR**: [#612](https://github.com/higress-group/higress-console/pull/612) \
  **Contributor**: @zhwaaaaaa \
  **Change Log**: 在DashboardServiceImpl中添加了对hop-to-hop头部的忽略处理，防止如Transfer-Encoding: chunked这样的头部被误传递。 \
  **Feature Value**: 通过正确处理hop-to-hop头部，确保了Grafana页面能正常工作于使用反向代理服务器的环境中，提升了系统的兼容性和用户体验。

- **Related PR**: [#608](https://github.com/higress-group/higress-console/pull/608) \
  **Contributor**: @Libres-coder \
  **Change Log**: 此PR为AI路由管理页面添加了插件显示支持，允许用户展开AI路由行以查看已启用的插件，并在配置页面中看到“Enabled”标签。 \
  **Feature Value**: 增强了AI路由管理功能，使用户能够更直观地管理与AI相关的插件状态，提升了用户体验和操作便捷性。

- **Related PR**: [#604](https://github.com/higress-group/higress-console/pull/604) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR引入了使用`higress.io/rewrite-target`注解进行路径重写的功能，支持正则表达式，增强了路径配置的灵活性。 \
  **Feature Value**: 通过增加基于正则表达式的路径重写能力，用户能够更灵活地控制和转换请求路径，提升了Higress网关的路由处理能力，满足更多场景下的需求。

- **Related PR**: [#603](https://github.com/higress-group/higress-console/pull/603) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR为静态服务源显示固定的服务端口80，通过在前端组件中硬编码该值实现。 \
  **Feature Value**: 用户可以更直观地看到并理解特定于静态服务源的默认端口号，增强了UI的清晰度和用户体验。

- **Related PR**: [#602](https://github.com/higress-group/higress-console/pull/602) \
  **Contributor**: @CH3CHO \
  **Change Log**: 该PR通过在前端页面中添加搜索功能，支持用户在选择AI路由的上游服务时进行搜索，提升了用户体验。 \
  **Feature Value**: 此功能使用户能够更快速准确地找到所需的上游服务，简化了配置过程，提高了操作效率。

- **Related PR**: [#566](https://github.com/higress-group/higress-console/pull/566) \
  **Contributor**: @OuterCyrex \
  **Change Log**: 该PR新增了对自定义Qwen服务的支持，包括启用互联网搜索、上传文件ID等功能。主要改动集中在后端SDK和前端UI部分。 \
  **Feature Value**: 通过支持自定义Qwen服务，用户能够更灵活地配置AI服务，比如使用特定的互联网搜索功能或指定文件ID，从而满足更多个性化需求。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#620](https://github.com/higress-group/higress-console/pull/620) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修正了sortWasmPluginMatchRules逻辑中的拼写错误，确保匹配规则正确排序。 \
  **Feature Value**: 修复此拼写错误提高了代码的可靠性和可读性，确保Wasm插件匹配规则能够按预期工作，减少了潜在的运行时错误。

- **Related PR**: [#619](https://github.com/higress-group/higress-console/pull/619) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR从AiRoute转换到ConfigMap的过程中移除了数据JSON中的版本信息，因为这些信息已经保存在ConfigMap的元数据中。 \
  **Feature Value**: 通过移除冗余的数据，提高了配置的一致性和简洁性，减少了潜在的数据冲突和不一致问题。

- **Related PR**: [#618](https://github.com/higress-group/higress-console/pull/618) \
  **Contributor**: @CH3CHO \
  **Change Log**: 该PR重构了SystemController中的API认证逻辑，以消除存在的安全漏洞。通过新增AllowAnonymous注解并调整ApiStandardizationAspect类，确保系统更安全。 \
  **Feature Value**: 此修复增强了系统的安全性，防止未经授权的访问和潜在的安全威胁，提升了用户体验与信任度。

- **Related PR**: [#617](https://github.com/higress-group/higress-console/pull/617) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了前端控制台错误，包括列表元素key属性缺失、CSP策略限制导致的图片加载失败以及Consumer.name字段类型错误等问题。 \
  **Feature Value**: 解决了用户在使用过程中遇到的多个前端问题，提升了用户体验，确保了应用的稳定性和安全性。

- **Related PR**: [#614](https://github.com/higress-group/higress-console/pull/614) \
  **Contributor**: @lc0138 \
  **Change Log**: 该PR修正了ServiceSource类中type字段的错误类型，并添加了字典值校验以确保数据准确性。 \
  **Feature Value**: 通过修复服务来源类型的错误，提高了系统的数据一致性和可靠性，减少了因类型不匹配导致的潜在问题。

- **Related PR**: [#613](https://github.com/higress-group/higress-console/pull/613) \
  **Contributor**: @lc0138 \
  **Change Log**: 此PR通过修改前端document.tsx文件，新增了15行代码来修复CSP等安全风险问题，确保了网站的安全性。 \
  **Feature Value**: 修复了与内容安全策略相关的安全风险，提升了应用程序的安全水平，保护用户免受潜在的安全威胁。

- **Related PR**: [#611](https://github.com/higress-group/higress-console/pull/611) \
  **Contributor**: @qshuai \
  **Change Log**: 修正了LlmProvidersController.java文件中关于添加新路由API的描述错误，将标题从'Add a new route'更正为'Ad'。 \
  **Feature Value**: 此修复解决了API文档中的误导性信息问题，确保开发者能够准确理解API的功能，提升开发体验和减少潜在的误用。

- **Related PR**: [#609](https://github.com/higress-group/higress-console/pull/609) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了Consumer.name字段类型错误的问题，将该字段的类型从boolean更正为string。 \
  **Feature Value**: 此修复确保了Consumer.name字段的数据一致性与准确性，避免因类型错误导致的数据处理问题，提升了系统的稳定性和用户体验。

- **Related PR**: [#605](https://github.com/higress-group/higress-console/pull/605) \
  **Contributor**: @SaladDay \
  **Change Log**: 修正了AI路由名称验证规则，使其支持点号，并统一大小写限制与界面提示。此外，更新了多语言环境下的错误提示信息。 \
  **Feature Value**: 解决了用户在设置AI路由名称时遇到的不一致问题，提升了用户体验及系统的易用性，确保了信息的一致性和准确性。

- **Related PR**: [#552](https://github.com/higress-group/higress-console/pull/552) \
  **Contributor**: @lcfang \
  **Change Log**: 该PR为解决服务实例端口不一致导致的兼容性问题，新增了vport属性，并在注册中心配置时提供了选择性配置虚拟端口的功能。 \
  **Feature Value**: 通过引入vport属性，用户可以更灵活地处理后端实例端口变化的情况，避免因端口变动而导致的服务路由失效问题，提升了系统的稳定性和灵活性。

### 📚 文档更新 (Documentation)

- **Related PR**: [#610](https://github.com/higress-group/higress-console/pull/610) \
  **Contributor**: @heimanba \
  **Change Log**: 此PR更新了前端灰度插件的文档配置，包括修改必填字段说明、更新关联规则以及同步中英文README和spec.yaml文件中的内容。 \
  **Feature Value**: 通过调整文档配置要求与描述，增强了配置的灵活性和兼容性，便于用户理解和使用前端灰度插件功能。

---

## 📊 发布统计

- 🚀 新功能: 7项
- 🐛 Bug修复: 10项
- 📚 文档更新: 1项

**总计**: 18项更改

感谢所有贡献者的辛勤付出！🎉


