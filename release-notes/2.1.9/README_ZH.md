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

- **feat(mcp-server): add server-level default authentication and MCP proxy server support** ([#3096](https://github.com/alibaba/higress/pull/3096)): 此功能增强了Higress的安全性和灵活性，允许用户为所有工具及请求设置统一的安全认证规则，简化了安全策略管理，提升了用户体验。
- **feat: add higress api mcp server** ([#2923](https://github.com/alibaba/higress/pull/2923)): 该功能增强了Higress管理能力，允许用户通过MCP工具更灵活地管理和配置Higress资源如路由和服务来源等，提升了用户体验和系统的可操作性。
- **feat: implement `hgctl agent` & `mcp add` subcommand ** ([#3051](https://github.com/alibaba/higress/pull/3051)): 新增的子命令极大提升了Higress管理的便捷性和灵活性，使得用户可以通过自然语言与Agent交互来管理Higress，并且简化了MCP服务的添加过程，增强了用户体验。这标志着Higress向更先进的运维方式迈进了一步。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat(mcp-server): add server-level default authentication and MCP proxy server support

**相关PR**: [#3096](https://github.com/alibaba/higress/pull/3096) | **贡献者**: [@johnlanni](https://github.com/johnlanni)

**使用背景**

随着微服务架构的普及，对API网关的安全性和灵活性要求越来越高。此PR解决了在Higress中为所有工具和请求设置默认认证方式的需求，以及需要一个能够代理MCP请求到后端MCP服务器的中间件问题。这不仅满足了用户对于简化认证配置管理的需求，也提供了一种新的MCP流量处理模式，尤其适用于那些希望将状态管理职责从后端服务转移到边缘侧（如Higress）的企业级应用开发团队。

**功能详述**

本次更新主要实现了两大新特性：一是服务器级别默认认证(`defaultDownstreamSecurity`与`defaultUpstreamSecurity`)，允许管理员为整个系统设定统一的认证策略；二是增加了MCP代理服务器类型(`mcp-proxy`)，它能够将客户端发送给Higress的MCP请求转发至指定的后端MCP服务器上，同时支持超时控制及全链路认证。技术上，通过更新依赖库版本 (`github.com/higress-group/wasm-go` 和 `github.com/higress-group/proxy-wasm-go-sdk`) 来支撑这些新功能的实现。

**使用方式**

启用该功能前，请确保已更新到最新版的Higress。对于默认认证设置，可以在全局配置文件中添加相应的JSON配置项来定义。例如，使用`defaultDownstreamSecurity`字段来指定客户端到网关间的认证方法。若要利用MCP代理功能，则需在创建MCP Server实例时指定其类型为`mcp-proxy`，并通过`mcpServerURL`属性指明目标MCP服务器地址。此外，还可以通过`timeout`参数调整请求超时时长。建议参考官方文档获取更详细的配置指南。

**功能价值**

此次更新极大地方便了开发者们在Higress平台上实施统一且灵活的身份验证策略，减少了重复配置工作量，同时也开启了基于边缘计算模型优化MCP协议处理的新途径。对于追求高效运维与安全保障的企业而言，这意味着可以更加轻松地实现细粒度访问控制、降低潜在风险敞口，并且有助于提升整体系统的稳定性和响应速度。更重要的是，这样的设计使得Higress能够更好地适配各种复杂的网络环境，在保持高性能的同时提供更多样化的服务发现与治理解决方案。

---

### 2. feat: add higress api mcp server

**相关PR**: [#2923](https://github.com/alibaba/higress/pull/2923) | **贡献者**: [@Tsukilc](https://github.com/Tsukilc)

**使用背景**

随着Higress系统的不断发展，用户对系统管理和调试的需求也在增加。原有的Higress Console Admin API虽然提供了基本的管理功能，但缺乏对AI路由、AI提供商和MCP服务器的管理能力。此次更新通过集成higress-ops MCP Server，增强了Higress的管理功能，使得用户能够更加灵活地管理和调试Higress配置。目标用户群体包括Higress系统的运维人员、开发人员以及需要通过Agent方式管理Higress的用户。

**功能详述**

此次更新主要实现了以下功能：1. 新增了AI路由（AI Route）管理功能，支持列出、获取、添加、更新和删除AI路由。2. 新增了AI提供商（AI Provider）管理功能，支持列出、获取、添加、更新和删除AI提供商。3. 新增了MCP服务器（MCP Server）管理功能，支持列出、获取、添加或更新、删除MCP服务器及其消费者。4. 重构了HigressClient，移除了用户名和密码参数，改为使用HTTP Basic Authentication进行鉴权。5. 更新了相关文档，确保用户能够了解并使用这些新功能。核心技术创新在于通过引入新的MCP工具，扩展了Higress的管理能力，使其更加灵活和强大。

**使用方式**

启用和配置此功能的方法如下：1. 在Higress配置文件中注册新的MCP Server，指定其类型为`higress-api`。2. 配置Higress Console的URL地址，并设置描述信息。3. 使用HGCTL命令行工具或其他MCP客户端与Higress API MCP Server进行交互。典型的使用场景包括：1. 通过HGCTL Agent以自然语言的方式管理Higress配置。2. 通过MCP客户端管理AI路由、AI提供商和MCP服务器。注意：1. 确保Higress Console URL正确无误。2. 使用HTTP Basic Authentication进行鉴权。3. 在编写代码时避免不必要的类型转换操作，以提升性能和代码清晰度。

**功能价值**

此次更新为用户带来了以下具体好处：1. 提升了Higress系统的可管理性和可调试性，用户可以通过MCP工具更方便地管理和调试Higress配置。2. 增强了系统的安全性和易用性，通过HTTP Basic Authentication进行鉴权，提高了系统的安全性。3. 为生态中的其他工具和系统提供了统一的API接口，促进了生态系统的整合和发展。4. 通过新增的AI路由和AI提供商管理功能，用户可以更好地利用AI技术优化Higress的路由策略。5. 通过MCP服务器管理功能，用户可以更灵活地管理和配置MCP服务器，提高系统的灵活性和可扩展性。

---

### 3. feat: implement `hgctl agent` & `mcp add` subcommand 

**相关PR**: [#3051](https://github.com/alibaba/higress/pull/3051) | **贡献者**: [@erasernoob](https://github.com/erasernoob)

**使用背景**

随着微服务架构的普及，服务网格（Service Mesh）成为管理复杂服务间通信的重要工具。Higress作为一款高性能的服务网格控制平面，需要提供更灵活、易用的管理工具。此次PR为`hgctl`命令行工具增加了两个新功能：`hgctl agent`和`mcp add`，前者引入了类似Claude Code的交互式代理，允许用户以自然语言的方式管理Higress；后者则简化了添加远程MCP服务器的过程，使其能够直接发布到Higress MCP Server管理工具中。这些改进不仅提高了Higress的可操作性，还增强了其在生态中的竞争力。目标用户群体主要是Higress的运维人员及开发者。

**功能详述**

本次变更实现了两个主要功能：
1. `hgctl agent`：该命令启动一个交互窗口，内部调用`claude-code`代理，引导用户设置必要的环境变量。首次使用时，会提示用户安装所需的依赖项。
2. `mcp add`：此命令允许用户直接添加两种类型的MCP服务——基于HTTP的直接代理和基于OpenAPI的动态生成服务。通过解析用户提供的参数（如URL、用户名、密码等），它能够自动配置并注册新的MCP服务器到Higress Console。技术上，新增了对`github.com/getkin/kin-openapi`库的支持，用于处理OpenAPI规范文件，并通过向Higress发送API请求完成服务注册流程。
此外，还更新了部分依赖项版本，确保与最新Go工具链兼容。

**使用方式**

启用和配置这两个新特性非常简单：
- 对于`hgctl agent`，只需运行`hgctl agent`即可启动交互界面，根据提示完成环境初始化。
- 使用`mcp add`来添加MCP服务时，按照如下格式输入命令：
  - 添加HTTP类型MCP服务：`hgctl mcp add <name> -t http <url> --user <username> --password <password> --url <higress_console_url>`
  - 添加OpenAPI类型MCP服务：`hgctl mcp add <name> -t openapi --spec <openapi_yaml_path> --user <username> --password <password> --url <higress_console_url>`
注意：确保已正确安装所有依赖项，并且具备访问Higress Console的权限。

**功能价值**

这个功能极大地提升了Higress的易用性和灵活性，使得非技术人员也能方便地管理和扩展其服务网格。对于系统性能来说，通过简化配置过程减少了人为错误的可能性，从而间接改善了系统的稳定性和可靠性。更重要的是，在当前高度竞争的服务网格市场中，这样的创新功能有助于吸引更多的用户选择Higress作为他们的解决方案。同时，也为未来进一步集成更多高级功能奠定了基础，比如更加智能化的运维自动化工具等。

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#3126](https://github.com/alibaba/higress/pull/3126) \
  **Contributor**: @johnlanni \
  **Change Log**: 该PR更新了Envoy依赖，允许通过WASM插件配置Redis客户端缓冲行为。具体而言，实现了从参数图中解析buffer_flush_timeout和max_buffer_size_before_flush。 \
  **Feature Value**: 此功能增强了WASM插件的灵活性，使用户能够更细粒度地控制Redis调用相关的参数，从而优化性能或满足特定需求，提升了用户体验。

- **Related PR**: [#3123](https://github.com/alibaba/higress/pull/3123) \
  **Contributor**: @johnlanni \
  **Change Log**: 此次PR升级了proxy版本至v2.2.0，并更新Go工具链与多个依赖包，同时添加了golang-filter对不同架构的支持并修复其相关依赖。 \
  **Feature Value**: 通过升级核心组件和修复依赖问题，提升了系统的稳定性和兼容性。增加的多架构支持扩大了软件适用范围，增强了用户体验。

- **Related PR**: [#3108](https://github.com/alibaba/higress/pull/3108) \
  **Contributor**: @wydream \
  **Change Log**: 新增了与视频相关的API路径和能力，包括常量定义、默认功能条目及正则路径处理。同时更新了OpenAI服务提供商以支持新添加的视频端点。 \
  **Feature Value**: 此次更新扩展了系统的多媒体处理能力，特别是对于视频内容的支持，为开发者提供了更丰富的接口选项，便于集成复杂的视频处理逻辑，从而提升用户体验。

- **Related PR**: [#3071](https://github.com/alibaba/higress/pull/3071) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR添加了`inject_encoded_data_to_filter_chain_on_header`示例，允许在无响应body的情况下向请求添加响应body。通过Wasm插件实现，并提供了详细的使用说明。 \
  **Feature Value**: 该功能帮助用户更灵活地处理响应数据，尤其是在需要动态添加响应内容的场景下，极大提升了Higress的功能性和灵活性。

- **Related PR**: [#3067](https://github.com/alibaba/higress/pull/3067) \
  **Contributor**: @wydream \
  **Change Log**: 此PR在ai-proxy插件中新增了vLLM作为AI提供商的支持，实现了包括Chat Completions、Text Completions、Model Listing等在内的多个OpenAI兼容API接口。 \
  **Feature Value**: 通过引入对vLLM的支持，该功能扩展了Higress处理AI请求的能力，使得用户能够更灵活地利用不同类型的AI服务，并且为使用Higress进行AI相关应用开发提供了更多选择。

- **Related PR**: [#3060](https://github.com/alibaba/higress/pull/3060) \
  **Contributor**: @erasernoob \
  **Change Log**: 此PR增强了`hgctl mcp`和`hgctl agent`命令，使其能够自动从安装配置文件及Kubernetes secrets中获取Higress Console的凭证信息，简化了用户操作流程。 \
  **Feature Value**: 通过自动检索凭据提升了用户体验，减少了手动输入账号密码的需求，使得使用`hgctl`工具管理Higress变得更加便捷高效。

- **Related PR**: [#3043](https://github.com/alibaba/higress/pull/3043) \
  **Contributor**: @2456868764 \
  **Change Log**: 修复了Milvus默认端口的错误，并在README.md中添加了Python示例代码。调整了部分配置以适应网关只做检索不做数据录入的场景。 \
  **Feature Value**: 解决了用户在使用过程中遇到的端口问题，同时提供了额外的Python代码示例，增强了文档的实用性和易用性，有助于用户更好地理解和使用项目功能。

- **Related PR**: [#3040](https://github.com/alibaba/higress/pull/3040) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: 该PR新增了对Anthropic消息API的支持，使得用户可以通过/v1/messages直接调用Anthropic服务，并且支持将OpenAI格式的请求体转换为Claude兼容的格式。 \
  **Feature Value**: 通过引入Anthropic消息API的支持，用户可以更灵活地配置和使用不同的AI服务提供者。这不仅丰富了Higress的功能集，还提升了用户的操作便捷性和平台的互操作性。

- **Related PR**: [#3038](https://github.com/alibaba/higress/pull/3038) \
  **Contributor**: @Libres-coder \
  **Change Log**: 新增了`list-plugin-instances`工具至MCP Server，支持AI Agents通过MCP协议查询特定作用域下的插件实例，并更新了中英文文档。 \
  **Feature Value**: 此功能允许用户更灵活地管理和监控Higress中的插件使用情况，增强了系统的可维护性和透明度，为用户提供了一种新的方式来了解其服务的状态和配置。

- **Related PR**: [#3032](https://github.com/alibaba/higress/pull/3032) \
  **Contributor**: @johnlanni \
  **Change Log**: 该PR改进了Qwen AI提供商配置，包括默认启用兼容模式和添加缺失的API端点，提升了用户体验。 \
  **Feature Value**: 通过默认启用兼容模式并增加API覆盖率，用户可以享受到更完善的功能支持与更好的开箱即用体验。

- **Related PR**: [#3029](https://github.com/alibaba/higress/pull/3029) \
  **Contributor**: @victorserbu2709 \
  **Change Log**: 该PR为groq提供者添加了v1/responses的支持，通过修改groq.go文件中的相关代码实现了新的功能。 \
  **Feature Value**: 增加对v1/responses的支持增强了groq提供者的功能，使得用户能够更灵活地处理和响应数据，提升了系统的灵活性与可用性。

- **Related PR**: [#3024](https://github.com/alibaba/higress/pull/3024) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR增加了对恶意URL和模型幻觉的检测，并针对特定消费者调整了配置。解决了在多事件响应及空内容送检时存在的问题。 \
  **Feature Value**: 增强了系统的安全性与稳定性，通过新增检测机制有效识别潜在威胁并防止错误响应引发的安全漏洞；同时，优化了用户级别的配置灵活性，提升了用户体验。

- **Related PR**: [#3008](https://github.com/alibaba/higress/pull/3008) \
  **Contributor**: @hellocn9 \
  **Change Log**: 该PR实现了支持MCP SSE状态会话的自定义参数名。用户可以通过设置`higress.io/mcp-sse-stateful-param-name`注解来指定自己的参数名称，从而增强了系统的灵活性和可配置性。 \
  **Feature Value**: 此功能允许用户根据自身需求自定义MCP SSE状态会话的参数名称，提高了系统的灵活性与用户体验，使得Higress能够更好地适应多样化的应用场景。

- **Related PR**: [#3006](https://github.com/alibaba/higress/pull/3006) \
  **Contributor**: @SaladDay \
  **Change Log**: 此PR为MCP Server的Redis配置添加了Secret引用支持，允许使用Kubernetes Secret存储敏感信息如密码，从而提高安全性。通过更新代码和文档，实现了从ConfigMap到Secret的平滑过渡。 \
  **Feature Value**: 该功能让用户能够更加安全地处理敏感数据，避免了直接在ConfigMap中硬编码密码带来的潜在风险。这对于重视安全性的用户来说是一个重要的改进。

- **Related PR**: [#2992](https://github.com/alibaba/higress/pull/2992) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR在认证鉴权流程中增加了对未授权消费者的名称记录。通过修改wasm-cpp插件中的key_auth部分，即使消费者没有被授权访问，其名称也会被记录下来。 \
  **Feature Value**: 这项改动提高了系统的透明度和可追踪性，使得管理员能够更容易地通过日志识别所有尝试访问的用户，包括那些未获授权的用户。这有助于增强安全审计和故障排查能力。

- **Related PR**: [#2978](https://github.com/alibaba/higress/pull/2978) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR在key-auth插件中实现了无论认证是否通过，只要能确定消费者名称就记录下来的功能。具体是在main.go文件中添加了一个新的请求头X-Mse-Consumer来存储消费者名称。 \
  **Feature Value**: 该功能改进了对消费者行为的追踪能力，使得系统能够更准确地监控和分析每个消费者的活动，即使在认证失败的情况下也能获取到消费者信息，从而提高了系统的安全性和可审计性。

- **Related PR**: [#2968](https://github.com/alibaba/higress/pull/2968) \
  **Contributor**: @2456868764 \
  **Change Log**: 新增了Vector Mapping的核心功能，包括字段映射系统和索引配置管理。这些功能支持灵活地与不同数据库模式集成，并定义多种向量索引类型。 \
  **Feature Value**: 通过提供字段映射和索引配置管理能力，用户可以更灵活地将Higress与不同的矢量数据库进行对接，从而增强了系统的适应性和可扩展性。

- **Related PR**: [#2943](https://github.com/alibaba/higress/pull/2943) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: 此PR增加了在生成release notes时支持自定义系统提示的功能，通过修改GitHub Actions工作流文件，实现了在指定段落中读取系统提示。 \
  **Feature Value**: 该功能允许用户在生成发布说明时加入特定的系统提示信息，从而提供了更灵活、个性化的文档生成体验，有助于提高项目文档的质量和相关性。

- **Related PR**: [#2942](https://github.com/alibaba/higress/pull/2942) \
  **Contributor**: @2456868764 \
  **Change Log**: 修复了当LLM提供者为空时的处理逻辑，并优化了相关文档。包括更新README.md以更清晰地描述MCP工具及其配置方式，以及调整了prompt模板。 \
  **Feature Value**: 增强了系统的健壮性和用户体验，通过允许LLM提供者为空避免了潜在错误，并提供了更加详尽、易懂的文档说明，帮助用户更好地理解和使用MCP工具。

- **Related PR**: [#2916](https://github.com/alibaba/higress/pull/2916) \
  **Contributor**: @imp2002 \
  **Change Log**: 本PR实现了Nginx迁移至Higress的MCP服务端，并提供了7种自动化工具来帮助转换Nginx配置和Lua插件。 \
  **Feature Value**: 该功能显著简化了从Nginx迁移到Higress的过程，通过提供自动化的迁移工具降低了用户操作难度，提高了迁移效率。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#3120](https://github.com/alibaba/higress/pull/3120) \
  **Contributor**: @lexburner \
  **Change Log**: 通过调整ai-proxy插件中的日志级别，减少了不必要的警告信息输出。具体来说，在qwen.go文件中修改了日志记录级别，将一部分警告级别的日志调整为更恰当的级别。 \
  **Feature Value**: 此次修复有助于改善系统的日志管理，减少了冗余的日志信息，使得运维人员能够更加专注于真正重要的日志消息，从而提高了问题定位效率及用户体验。

- **Related PR**: [#3119](https://github.com/alibaba/higress/pull/3119) \
  **Contributor**: @johnlanni \
  **Change Log**: 该PR通过替换Connection中的reqChan和deltaReqChan为channels.Unbounded来解决HTTP2流控导致的死锁问题，确保了Stream方法不会被阻塞、Put方法不阻塞请求接收及正常接收客户端ACK请求。 \
  **Feature Value**: 此修复提高了系统的稳定性和响应速度，避免了因HTTP2流控引发的双向通信死锁问题，提升了用户体验，特别是在处理大数据量响应时的表现。

- **Related PR**: [#3118](https://github.com/alibaba/higress/pull/3118) \
  **Contributor**: @johnlanni \
  **Change Log**: 通过添加nil检查来避免端口级别的TLS和LoadBalancer设置无条件地覆盖现有配置，并对负载均衡策略进行了细粒度的合并，保证了从ingress注解转换而来的配置不会被意外修改。 \
  **Feature Value**: 修复了DestinationRule中端口级别策略可能会覆盖ingress注解所生成配置的问题，增强了系统的稳定性和一致性，确保用户定义的网络策略能够正确应用。

- **Related PR**: [#3095](https://github.com/alibaba/higress/pull/3095) \
  **Contributor**: @rinfx \
  **Change Log**: 修复了claude2openai转换过程中usage信息丢失的问题，并在bedrock流式工具响应中添加了index字段，确保了数据完整性和准确性。 \
  **Feature Value**: 保证了从Claude到OpenAI的数据转换不会丢失关键的usage信息，同时增强了Bedrock流式工具响应的功能，使开发者能够更好地追踪和管理流式输出。这对提高系统可靠性和用户体验有着积极影响。

- **Related PR**: [#3084](https://github.com/alibaba/higress/pull/3084) \
  **Contributor**: @rinfx \
  **Change Log**: 修复了当启用流式处理时，从Claude转换到OpenAI请求过程中未能正确包含include_usage: true参数的问题，确保了API调用的完整性和一致性。 \
  **Feature Value**: 此修复确保了在使用流式处理功能时，用户能够准确获得包括使用情况在内的所有相关信息，提升了API响应的数据完整性，对于依赖这些信息进行后续处理或分析的应用至关重要。

- **Related PR**: [#3074](https://github.com/alibaba/higress/pull/3074) \
  **Contributor**: @Jing-ze \
  **Change Log**: 此次PR在log-request-response插件中增加了对Content-Encoding的检查，以避免日志中出现压缩后的请求/响应体导致的日志混乱问题。 \
  **Feature Value**: 通过此修复，用户能够获得更清晰、易读的日志输出，特别是对于启用了Gzip等压缩方式的应用场景下，极大提升了调试效率和使用体验。

- **Related PR**: [#3069](https://github.com/alibaba/higress/pull/3069) \
  **Contributor**: @Libres-coder \
  **Change Log**: 修复了CI测试框架中的一个bug，通过在prebuild.sh脚本中添加`go mod tidy`命令来更新根目录下的go.mod文件，解决因go.mod未更新导致的e2e测试失败问题。 \
  **Feature Value**: 解决了所有触发wasm插件e2e测试的PR遇到的CI测试失败问题，确保了持续集成流程的稳定性与可靠性，提升了开发者的贡献体验。

- **Related PR**: [#3010](https://github.com/alibaba/higress/pull/3010) \
  **Contributor**: @rinfx \
  **Change Log**: 修正了bedrock事件流有时因拆包问题导致解析失败的问题，同时调整了maxtoken转换逻辑以适应边界情况。 \
  **Feature Value**: 解决了bedrock事件流在特定情况下无法正确解析的问题，保证了服务的稳定性和可靠性，提升了用户体验。

- **Related PR**: [#2997](https://github.com/alibaba/higress/pull/2997) \
  **Contributor**: @hanxiantao \
  **Change Log**: 优化了集群、AI Token和WASM插件的限流逻辑，调整为累加方式统计请求次数和token使用量，避免在修改限流值时重置计数。 \
  **Feature Value**: 通过优化限流逻辑，提升了系统的稳定性和准确性，减少了因阈值调整导致的数据重置问题，增强了用户体验。

- **Related PR**: [#2988](https://github.com/alibaba/higress/pull/2988) \
  **Contributor**: @johnlanni \
  **Change Log**: 该PR修复了jsonrpc-converter插件中使用错误的JSON字符串格式的问题，改为直接使用原始JSON数据。通过修改main.go文件中的相关代码，确保了数据处理的准确性。 \
  **Feature Value**: 修复了由于不正确的JSON字符串格式导致的数据处理错误问题，提升了系统的稳定性与准确性，减少了潜在的运行时错误，为用户提供了一个更加可靠的服务环境。

- **Related PR**: [#2973](https://github.com/alibaba/higress/pull/2973) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了当`match_rule_domain`被设置为空字符串时，Higress 2.1.8 不支持的问题。通过始终使用通配符匹配所有域来消除兼容性风险。 \
  **Feature Value**: 提高了系统的稳定性和兼容性，避免了因不支持空字符串导致的配置错误问题，确保了用户在不同版本间的无缝迁移体验。

- **Related PR**: [#2952](https://github.com/alibaba/higress/pull/2952) \
  **Contributor**: @Erica177 \
  **Change Log**: 修复了ToolSecurity结构体中的字段名错误，将type改为id，确保JSON序列化正确。 \
  **Feature Value**: 此更改解决了因字段名错误导致的数据解析问题，提升了系统稳定性和数据准确性，改善用户体验。

- **Related PR**: [#2948](https://github.com/alibaba/higress/pull/2948) \
  **Contributor**: @johnlanni \
  **Change Log**: 修正了Azure OpenAI响应API处理和service URL类型检测的问题。改进了自定义完整路径的逻辑，增强了对响应API端点的支持，并修复了流事件解析中的边缘情况。 \
  **Feature Value**: 此更改改善了与Azure OpenAI服务的兼容性，特别是对于那些使用非标准URL路径或依赖特定API端点的应用程序。用户现在可以更可靠地利用Higress来代理Azure OpenAI请求，减少了因配置异常导致的服务中断。

- **Related PR**: [#2941](https://github.com/alibaba/higress/pull/2941) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR修复了ai-security-guard插件与旧配置的兼容性问题，通过调整main.go文件中的特定字段映射逻辑来实现。 \
  **Feature Value**: 解决了因配置更新导致的不兼容问题，确保系统能够平稳过渡到新版本而不会中断用户现有的安全设置。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#3113](https://github.com/alibaba/higress/pull/3113) \
  **Contributor**: @johnlanni \
  **Change Log**: 实现了Protobuf消息的递归哈希计算与缓存功能，利用xxHash算法提升性能。特别处理了google.protobuf.Any类型和具有确定排序的地图字段。 \
  **Feature Value**: 通过减少LDS中的重复序列化操作来优化性能，从而加快过滤链匹配和监听器更新的速度，提升了用户体验。

- **Related PR**: [#2945](https://github.com/alibaba/higress/pull/2945) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR优化了ai-load-balancer的全局最小请求数选pod逻辑，通过更新Lua脚本来改进负载均衡策略。 \
  **Feature Value**: 提升了系统的负载均衡效率，使请求分配更加均匀合理，有助于提高系统整体性能和响应速度。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2965](https://github.com/alibaba/higress/pull/2965) \
  **Contributor**: @CH3CHO \
  **Change Log**: 更新了ai-proxy README文件中azureServiceUrl的描述，增加了更多的配置说明细节，以帮助开发者更好地理解和使用该参数。 \
  **Feature Value**: 改进了文档质量，使得用户能够更清晰地理解如何设置和使用azureServiceUrl参数，从而提高配置过程中的准确性和效率。

### 🧪 测试改进 (Testing)

- **Related PR**: [#3110](https://github.com/alibaba/higress/pull/3110) \
  **Contributor**: @Jing-ze \
  **Change Log**: 此PR为GitHub Actions工作流中的Codecov上传步骤添加了`CODECOV_TOKEN`环境变量配置，确保Codecov能够正确认证。 \
  **Feature Value**: 通过在CI/CD流程中安全地配置Codecov令牌，该功能增强了代码覆盖率报告的准确性和可靠性，从而帮助开发者更好地跟踪和改善代码质量。

- **Related PR**: [#3097](https://github.com/alibaba/higress/pull/3097) \
  **Contributor**: @johnlanni \
  **Change Log**: 本PR为mcp-server模块添加了单元测试，包括大量新的测试用例，确保了代码质量和稳定性。 \
  **Feature Value**: 增强了mcp-server模块的可靠性，通过引入全面的测试覆盖，帮助开发者更早地发现潜在问题，提升了用户体验和系统稳定性。

- **Related PR**: [#2998](https://github.com/alibaba/higress/pull/2998) \
  **Contributor**: @Patrisam \
  **Change Log**: 该PR实现了Cloudflare的端到端测试用例，增加了go-wasm-ai-proxy.go和go-wasm-ai-proxy.yaml文件中的测试代码，以确保Cloudflare相关功能的稳定性和可靠性。 \
  **Feature Value**: 通过增加针对Cloudflare特性的端到端测试，增强了系统对于Cloudflare集成部分的功能验证能力，帮助开发者及早发现并解决问题，提升了用户使用体验的安全性与稳定性。

- **Related PR**: [#2980](https://github.com/alibaba/higress/pull/2980) \
  **Contributor**: @Jing-ze \
  **Change Log**: 增强了WASM插件单元测试的CI工作流，添加了覆盖率显示功能和80%覆盖率阈值门限，低于该阈值将导致CI失败。 \
  **Feature Value**: 提高了代码质量监控标准，确保所有WASM插件达到至少80%的测试覆盖率，有助于发现潜在问题并提升软件可靠性。

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
  **Change Log**: 本次PR优化了mcp server的交互能力，包括默认重写header host、改进交互方式支持选择transport并完整替换path，以及增强dsn字符处理逻辑以支持特殊字符@。 \
  **Feature Value**: 这些更新增强了系统的灵活性和兼容性，使得用户能够更方便地配置后端服务地址，并且提高了对特殊字符的支持，提升了用户体验。

- **Related PR**: [#612](https://github.com/higress-group/higress-console/pull/612) \
  **Contributor**: @zhwaaaaaa \
  **Change Log**: 此PR通过添加对hop-to-hop头部的忽略处理，解决了由于反向代理服务器发送chunked编码导致Grafana页面无法正常工作的问题。 \
  **Feature Value**: 确保了Grafana页面在使用反向代理时仍能正常显示，提升了系统的兼容性和用户体验。

- **Related PR**: [#608](https://github.com/higress-group/higress-console/pull/608) \
  **Contributor**: @Libres-coder \
  **Change Log**: 此PR为AI路由管理页面增加了插件显示支持，用户可以查看启用的插件，并且在配置页面中看到“启用”标签。 \
  **Feature Value**: 增强了AI路由页面的功能性，使得用户能够直观地了解每个AI路由上已经启用的插件，提升了用户体验和管理效率。

- **Related PR**: [#604](https://github.com/higress-group/higress-console/pull/604) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR通过添加`higress.io/rewrite-target`注解支持了基于正则表达式的路径重写功能，增强了路由配置的灵活性。 \
  **Feature Value**: 新增的正则路径重写功能使得用户能够更灵活地控制请求路径转换逻辑，提升了应用在处理复杂URL模式时的能力。

- **Related PR**: [#603](https://github.com/higress-group/higress-console/pull/603) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR在前端页面中添加了一个常量STATIC_SERVICE_PORT，值为80，并在服务源组件中显示该固定端口，用于静态服务源。 \
  **Feature Value**: 通过显示固定的服务端口号80，用户可以更明确地知道静态服务源使用的标准HTTP端口，从而提高配置的清晰度和易用性。

- **Related PR**: [#602](https://github.com/higress-group/higress-console/pull/602) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR实现了在选择AI路由的上游服务时支持搜索功能，通过前端界面优化提升了用户体验。 \
  **Feature Value**: 新增了搜索功能，用户可以更快速地找到并选择所需的上游服务，提高了配置效率和使用便利性。

- **Related PR**: [#566](https://github.com/higress-group/higress-console/pull/566) \
  **Contributor**: @OuterCyrex \
  **Change Log**: 新增了自定义Qwen服务的支持，包括启用互联网搜索和上传文件ID等功能。主要在前端和后端代码中增加了相应的支持逻辑。 \
  **Feature Value**: 该功能允许用户配置自定义的Qwen服务，增强了系统的灵活性和扩展性，满足更多个性化需求。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#620](https://github.com/higress-group/higress-console/pull/620) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR修复了sortWasmPluginMatchRules逻辑中的拼写错误，确保了代码的正确性和一致性。 \
  **Feature Value**: 修正了排序规则处理过程中的文本错误，提升了系统的稳定性和可靠性，避免了因拼写问题导致的潜在故障。

- **Related PR**: [#619](https://github.com/higress-group/higress-console/pull/619) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR移除了从AiRoute转换为ConfigMap的数据JSON中的版本信息，因为该信息已经在ConfigMap的元数据中保存。 \
  **Feature Value**: 通过消除冗余数据，提高了配置管理的一致性和准确性，简化了用户在处理ConfigMap时的操作复杂度。

- **Related PR**: [#618](https://github.com/higress-group/higress-console/pull/618) \
  **Contributor**: @CH3CHO \
  **Change Log**: 重构了SystemController中的API认证逻辑，通过引入新的AllowAnonymous注解来修复安全漏洞，确保系统控制器的API调用更加安全可靠。 \
  **Feature Value**: 此次更新解决了SystemController中存在的安全问题，提高了系统的安全性，防止未经授权的访问，保障了用户数据的安全性和隐私。

- **Related PR**: [#617](https://github.com/higress-group/higress-console/pull/617) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了前端控制台中的多个错误，包括列表元素缺少唯一key警告、图片加载违反内容安全策略以及消费者名称字段类型不正确的问题。 \
  **Feature Value**: 通过解决这些前端错误，提高了用户体验和应用程序的稳定性，减少了因错误而导致的潜在问题。

- **Related PR**: [#614](https://github.com/higress-group/higress-console/pull/614) \
  **Contributor**: @lc0138 \
  **Change Log**: 此PR修正了ServiceSource类中type字段的错误类型，并添加了字典值校验，确保该字段只能接受预定义的有效值。 \
  **Feature Value**: 修复服务来源类型字段的不正确设置问题，提高了系统稳定性和数据准确性，避免因类型错误引发的潜在运行时错误。

- **Related PR**: [#613](https://github.com/higress-group/higress-console/pull/613) \
  **Contributor**: @lc0138 \
  **Change Log**: 该PR通过修改前端document.tsx文件，增加了15行代码来修复CSP等安全风险问题。 \
  **Feature Value**: 此修复增强了应用程序的安全性，有效防止了潜在的跨站脚本攻击和其他与内容安全策略相关的威胁，提升了用户体验和数据保护。

- **Related PR**: [#611](https://github.com/higress-group/higress-console/pull/611) \
  **Contributor**: @qshuai \
  **Change Log**: 修正了LlmProvidersController类中API文档注解的拼写错误，确保了API文档的准确性。 \
  **Feature Value**: 该PR修复了一个小但重要的文档问题，提升了API文档的质量，使开发者能够更准确地理解API的功能。

- **Related PR**: [#609](https://github.com/higress-group/higress-console/pull/609) \
  **Contributor**: @CH3CHO \
  **Change Log**: 该PR修复了Consumer接口中name字段类型错误的问题，将布尔类型更正为字符串类型。 \
  **Feature Value**: 此修复确保了Consumer名称能够正确存储和显示，避免了因类型不匹配导致的数据处理错误，提升了系统的稳定性和用户数据的准确性。

- **Related PR**: [#605](https://github.com/higress-group/higress-console/pull/605) \
  **Contributor**: @SaladDay \
  **Change Log**: 修正了AI路由名称的验证规则，使其支持点号，并统一了界面提示与实际验证逻辑，同时调整了错误提示信息。 \
  **Feature Value**: 解决了用户在设置AI路由名称时遇到的不一致问题，提高了系统的可用性和用户体验。

- **Related PR**: [#552](https://github.com/higress-group/higress-console/pull/552) \
  **Contributor**: @lcfang \
  **Change Log**: 新增vport属性以解决服务后端端口变化导致的路由配置失效问题，通过在服务注册时配置vport属性来确保兼容性。 \
  **Feature Value**: 解决了因服务实例端口不一致导致的路由配置失效问题，提高了系统的稳定性和用户体验。

### 📚 文档更新 (Documentation)

- **Related PR**: [#610](https://github.com/higress-group/higress-console/pull/610) \
  **Contributor**: @heimanba \
  **Change Log**: 更新了前端灰度插件文档中的配置字段说明，包括将rewrite、backendVersion和enabled字段更改为非必填项，并修正了部分文本描述以确保术语的一致性和准确性。 \
  **Feature Value**: 通过提高配置的灵活性和兼容性并确保文档的准确性和一致性，此更改使得用户能够更容易理解和使用前端灰度插件，降低了学习成本。

---

## 📊 发布统计

- 🚀 新功能: 7项
- 🐛 Bug修复: 10项
- 📚 文档更新: 1项

**总计**: 18项更改

感谢所有贡献者的辛勤付出！🎉


