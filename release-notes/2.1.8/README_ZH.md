# Higress


## 📋 本次发布概览

本次发布包含 **30** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 13项
- **Bug修复**: 7项
- **重构优化**: 5项
- **文档更新**: 4项
- **测试改进**: 1项

### ⭐ 重点关注

本次发布包含 **2** 项重要更新，建议重点关注：

- **feat: add rag mcp server** ([#2930](https://github.com/alibaba/higress/pull/2930)): 通过引入RAG MCP服务器，为用户提供了一种新的方式来管理与检索知识，增强了系统的功能性和实用性。
- **refactor(mcp): use ECDS for golang filter configuration to avoid connection drain** ([#2931](https://github.com/alibaba/higress/pull/2931)): 采用ECDS进行过滤器配置避免了直接嵌入golang过滤器配置带来的不稳定因素，提高了系统的稳定性和可维护性，对用户而言减少了不必要的服务中断。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat: add rag mcp server

**相关PR**: [#2930](https://github.com/alibaba/higress/pull/2930) | **贡献者**: [@2456868764](https://github.com/2456868764)

**使用背景**

在现代应用中，知识管理和检索变得越来越重要。许多系统需要快速、准确地从大量文本数据中提取和检索信息。RAG (Retrieval-Augmented Generation) 技术结合了检索和生成模型，能够有效提升知识管理的效率和准确性。本PR引入了一个Model Context Protocol (MCP) 服务器，专门用于知识管理和检索，满足了用户对高效信息处理的需求。目标用户群体包括需要处理大量文本数据的企业和开发者，尤其是在自然语言处理（NLP）和机器学习领域。

**功能详述**

该PR实现了RAG MCP服务器，新增了多个功能模块，包括知识管理、块管理、搜索和聊天功能。核心功能包括：
1. **知识管理**：支持从文本创建知识块。
2. **块管理**：提供列表显示和删除知识块的功能。
3. **搜索**：支持基于关键词的搜索功能。
4. **聊天功能**：允许用户发送聊天消息并获取响应。
技术实现上，该服务器使用了多种外部库，如`github.com/dlclark/regexp2`、`github.com/milvus-io/milvus-sdk-go/v2`和`github.com/pkoukk/tiktoken-go`，这些库提供了正则表达式处理、向量数据库管理和文本编码等功能。关键代码变更包括新增HTTP客户端、配置文件和多个处理函数，确保了系统的灵活性和可配置性。

**使用方式**

启用和配置RAG MCP服务器的步骤如下：
1. 在`higress-config`配置文件中启用MCP服务器，并设置相应的路径和配置项。
2. 配置RAG系统的基础参数，如分块器类型、块大小和重叠等。
3. 配置LLM（大语言模型）提供商及其API密钥、模型名称等。
4. 配置嵌入模型提供商及其API密钥、模型名称等。
5. 配置向量数据库提供商及其连接信息。
示例配置如下：
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
注意事项：
- 确保所有配置项正确无误，特别是API密钥和模型名称。
- 在生产环境中，建议对超时时间等参数进行适当调整以适应不同网络环境。

**功能价值**

RAG MCP服务器为用户提供了一套完整的知识管理和检索解决方案，提升了系统的智能化和自动化水平。具体好处包括：
1. **提高效率**：通过集成的知识管理和检索功能，用户可以快速处理和检索大量文本数据，节省时间和资源。
2. **增强准确性**：结合RAG技术，系统能够更准确地提取和检索信息，减少错误率。
3. **灵活配置**：提供了丰富的配置选项，用户可以根据实际需求进行灵活调整，满足不同场景下的需求。
4. **扩展性强**：支持多种提供商和模型，方便用户根据业务需求选择合适的组件和技术栈。
5. **稳定性提升**：通过详细的配置验证和错误处理机制，确保系统的稳定性和健壮性。

---

### 2. refactor(mcp): use ECDS for golang filter configuration to avoid connection drain

**相关PR**: [#2931](https://github.com/alibaba/higress/pull/2931) | **贡献者**: [@johnlanni](https://github.com/johnlanni)

**使用背景**

当前实现中，Golang过滤器配置直接嵌入在HTTP_FILTER补丁中，这会导致配置更改时出现连接耗尽的问题。主要原因是Go map在`map[string]any`字段中的排序不一致，以及HTTP_FILTER更新触发的监听器配置更改。这个问题影响了系统的稳定性和用户体验。目标用户群体是使用Higress进行服务网格管理的开发者和运维人员。

**功能详述**

此PR将配置分为两部分：HTTP_FILTER仅包含带有`config_discovery`的过滤器引用，而EXTENSION_CONFIG则包含实际的Golang过滤器配置。通过这种方式，配置更改不会直接导致连接耗尽。具体实现包括更新`constructMcpSessionStruct`和`constructMcpServerStruct`方法以返回与EXTENSION_CONFIG兼容的格式，并更新单元测试以匹配新的配置结构。核心技术创新在于利用ECDS机制分离配置，使配置更改更加平滑。

**使用方式**

启用和配置这个功能不需要额外的操作，因为它是在后台自动处理的。典型的使用场景是在Higress中配置Golang过滤器时，系统会自动将其分为HTTP_FILTER和EXTENSION_CONFIG两部分。用户只需按照常规方式配置Golang过滤器即可。需要注意的是，在升级到新版本时，确保所有相关的配置文件都已更新，并且在生产环境中进行充分的测试，以确保配置更改不会引入其他问题。

**功能价值**

通过分离配置并使用ECDS，此功能消除了配置更改时的连接耗尽问题，显著提高了系统的稳定性和用户体验。此外，这种设计使得配置更易于管理和维护，减少了因配置更改引起的潜在问题。对于大规模的服务网格部署，这一改进尤为重要，因为它可以减少因配置更改导致的服务中断，从而提高整体系统的可靠性和可用性。

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#2926](https://github.com/alibaba/higress/pull/2926) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR在vertex-ai中添加了对多模态、函数调用和思考的支持，涉及引入正则表达式库及处理逻辑的改进。 \
  **Feature Value**: 通过增加新功能，使得vertex-ai能够更好地支持复杂场景下的应用需求，如多模态数据处理和更灵活的功能调用方式，提升了系统的灵活性与实用性。

- **Related PR**: [#2917](https://github.com/alibaba/higress/pull/2917) \
  **Contributor**: @Aias00 \
  **Change Log**: 此次PR新增了对Fireworks AI的支持，扩展了AI代理插件的功能，包括必要的配置文件和测试代码的添加。 \
  **Feature Value**: 增加对Fireworks AI的支持使用户能够利用该平台提供的AI功能，拓宽了应用程序可以集成的AI服务范围，增强了用户体验。

- **Related PR**: [#2907](https://github.com/alibaba/higress/pull/2907) \
  **Contributor**: @Aias00 \
  **Change Log**: 此PR升级了wasm-go以支持outputSchema功能，涉及jsonrpc-converter和oidc插件的依赖更新。 \
  **Feature Value**: 通过支持outputSchema，增强了wasm-go插件的功能性和灵活性，使用户能够更方便地处理和定义输出数据结构。

- **Related PR**: [#2897](https://github.com/alibaba/higress/pull/2897) \
  **Contributor**: @rinfx \
  **Change Log**: 此次PR为ai-proxy bedrock添加了多模态支持及thinking功能，通过扩展bedrock.go中的相关代码来实现。 \
  **Feature Value**: 新增的多模态和thinking支持丰富了ai-proxy的功能集，使得用户能够利用更先进的AI技术处理复杂场景，提升了系统的灵活性与实用性。

- **Related PR**: [#2891](https://github.com/alibaba/higress/pull/2891) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR在AI内容安全插件中添加了针对不同消费者配置特定检测服务的功能，允许用户根据需求自定义请求和响应的检查规则。 \
  **Feature Value**: 通过支持为不同消费者设置独立的检测服务，该功能增强了系统的灵活性与安全性，使用户能够更精确地控制内容审查过程，从而满足多样化的安全策略需求。

- **Related PR**: [#2883](https://github.com/alibaba/higress/pull/2883) \
  **Contributor**: @Aias00 \
  **Change Log**: 此PR为美团Longcat增加了支持，包括实现与Longcat平台的集成和相关的单元测试。 \
  **Feature Value**: 新增对美团Longcat的支持扩展了插件的功能范围，使得用户能够利用更多AI服务提供商的技术，增强了应用的灵活性和多样性。

- **Related PR**: [#2867](https://github.com/alibaba/higress/pull/2867) \
  **Contributor**: @Aias00 \
  **Change Log**: 此PR新增了Gzip配置支持，并更新了默认设置。通过在Helm配置文件中添加gzip选项，用户可以自定义压缩参数以优化响应性能。 \
  **Feature Value**: 增加了对Gzip配置的支持，使得用户可以根据需求调整HTTP响应的压缩级别，有助于减少传输的数据量，加快页面加载速度，提升用户体验。

- **Related PR**: [#2844](https://github.com/alibaba/higress/pull/2844) \
  **Contributor**: @Aias00 \
  **Change Log**: 此PR通过支持useSourceIp增强了负载均衡的一致性哈希算法，修改了相关的Go代码文件以及添加了一个示例配置文件。 \
  **Feature Value**: 新增的useSourceIp选项允许用户基于源IP地址进行一致性哈希负载均衡，这有助于提高服务在特定网络条件下的稳定性和可靠性。

- **Related PR**: [#2843](https://github.com/alibaba/higress/pull/2843) \
  **Contributor**: @erasernoob \
  **Change Log**: 此PR为AI代理插件添加了NVIDIA Triton服务器支持，包括相关配置说明和代码实现。 \
  **Feature Value**: 新增对Triton服务器的支持扩展了AI代理插件的功能集，使用户能够利用高性能的机器学习推理服务。

- **Related PR**: [#2806](https://github.com/alibaba/higress/pull/2806) \
  **Contributor**: @C-zhaozhou \
  **Change Log**: 此PR使ai-security-guard兼容MultiModalGuard接口，增加了多模态API的支持，并更新了相关文档。 \
  **Feature Value**: 通过支持多模态API，增强了ai-security-guard的功能，使其能够处理更复杂的内容安全场景，提升了用户体验和安全性。

- **Related PR**: [#2727](https://github.com/alibaba/higress/pull/2727) \
  **Contributor**: @Aias00 \
  **Change Log**: 本PR为OpenAI添加了端到端测试支持，包括非流式和流式请求的测试用例。 \
  **Feature Value**: 新增的OpenAI端到端测试有助于确保系统在处理不同类型的请求时保持稳定性和准确性，提升了用户体验。

- **Related PR**: [#2593](https://github.com/alibaba/higress/pull/2593) \
  **Contributor**: @Xscaperrr \
  **Change Log**: 增加了WorkloadSelector字段以限制EnvoyFilter的作用范围，确保在存在开源istio环境下不影响同命名空间的其他组件。 \
  **Feature Value**: 通过限定EnvoyFilter仅作用于Higress Gateway，避免了对环境内其他istio gateway/sidecar造成干扰，提升了配置的安全性和隔离性。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#2938](https://github.com/alibaba/higress/pull/2938) \
  **Contributor**: @wydream \
  **Change Log**: 此PR修复了MultiModalGuard模式下因缺少AttackLevel字段支持而导致的提示攻击检测失效问题，确保所有级别的攻击都能被正确识别。 \
  **Feature Value**: 通过增加对AttackLevel字段的支持，提高了系统安全性，防止高风险级别的提示攻击未被拦截的情况发生，保障了用户体验和安全。

- **Related PR**: [#2904](https://github.com/alibaba/higress/pull/2904) \
  **Contributor**: @johnlanni \
  **Change Log**: 修复了在处理HTTP请求时，原始Authorization头可能被覆盖的问题。通过无条件保存并检查非空后再写入上下文，确保认证信息的准确性和安全性。 \
  **Feature Value**: 该修复提升了系统的安全性和稳定性，避免了因认证信息丢失而导致的潜在认证失败或安全漏洞问题，增强了用户体验和信任度。

- **Related PR**: [#2899](https://github.com/alibaba/higress/pull/2899) \
  **Contributor**: @Jing-ze \
  **Change Log**: 此PR对MCP服务器进行了优化，包括提前解析主机模式以减少运行时开销和移除未使用的DomainList字段。同时修复了SSE消息格式问题，特别是处理多余换行符的问题。 \
  **Feature Value**: 通过提高模式匹配效率和内存使用率，以及修正SSE消息中的错误，提升了用户体验和服务稳定性，确保了数据传输的正确性和完整性。

- **Related PR**: [#2892](https://github.com/alibaba/higress/pull/2892) \
  **Contributor**: @johnlanni \
  **Change Log**: 修正了Claude API返回数组格式content时的JSON解组错误，并移除了重复的代码结构，提升了代码质量和维护性。 \
  **Feature Value**: 解决了由于不正确的数据类型而导致的消息解析失败问题，增强了系统的稳定性和用户体验，对于使用数组作为content格式的用户来说，这修复确保了消息处理流程的顺畅。

- **Related PR**: [#2882](https://github.com/alibaba/higress/pull/2882) \
  **Contributor**: @johnlanni \
  **Change Log**: 解决了Claude流式响应转换逻辑中的SSE事件分块问题，改进了协议自动转换和工具调用状态跟踪。 \
  **Feature Value**: 提高了Claude与OpenAI兼容提供者之间的双向转换可靠性，避免了连接阻塞，增强了用户体验。

- **Related PR**: [#2865](https://github.com/alibaba/higress/pull/2865) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: 该PR解决了当SSE事件被分割成多个chunk时，SSE连接会被阻塞的问题。通过在代理mcp server场景下增加缓存机制来确保数据流处理的连续性。 \
  **Feature Value**: 修复了可能导致SSE连接中断的问题，增强了系统的稳定性和用户体验。用户不再会因为网络条件或服务器响应方式而遇到数据接收不完整的情况。

- **Related PR**: [#2859](https://github.com/alibaba/higress/pull/2859) \
  **Contributor**: @lcfang \
  **Change Log**: 此PR通过在mcpbridge中新增vport元素，解决了当注册服务实例端口不一致时路由配置失效的问题。主要改动包括更新CRD定义、protobuf文件及相关生成代码。 \
  **Feature Value**: 该功能确保了即使后端实例端口发生变化，服务的路由配置也能保持有效，从而提高了系统的稳定性和兼容性，为用户提供了更加可靠的服务体验。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#2933](https://github.com/alibaba/higress/pull/2933) \
  **Contributor**: @rinfx \
  **Change Log**: 移除了bedrock和vertex中重复的think标签，减少了冗余代码，提高了代码的可读性和维护性。 \
  **Feature Value**: 通过去除不必要的重复代码，提升了项目的整体质量和开发效率，使得代码结构更加清晰，方便后续的维护和扩展。

- **Related PR**: [#2927](https://github.com/alibaba/higress/pull/2927) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR修改了ai-statistics插件中API名称提取逻辑，将检查条件从固定长度5调整为至少3个部分，以提高灵活性和兼容性。 \
  **Feature Value**: 通过放宽API字符串分割的限制条件，增强了系统对不同格式API字符串的支持能力，提升了系统的适应性和稳定性。

- **Related PR**: [#2922](https://github.com/alibaba/higress/pull/2922) \
  **Contributor**: @daixijun \
  **Change Log**: 该PR将项目中引用的Higress SDK包名从github.com/alibaba/higress升级为github.com/alibaba/higress/v2，以兼容最新版本。 \
  **Feature Value**: 通过更新包名，确保项目可以引入并使用Higress的最新功能和改进，提升开发效率和代码质量。

- **Related PR**: [#2890](https://github.com/alibaba/higress/pull/2890) \
  **Contributor**: @johnlanni \
  **Change Log**: 重构了`matchDomain`函数，引入HostMatcher结构及匹配类型，替换正则表达式以简单字符串操作提高性能，并实现端口剥离逻辑。 \
  **Feature Value**: 通过优化主机匹配逻辑提高了系统性能和代码可维护性，使得处理包含端口号的主机头更加准确高效，提升了用户体验。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2915](https://github.com/alibaba/higress/pull/2915) \
  **Contributor**: @a6d9a6m \
  **Change Log**: 修复了README_JP.md中的一个失效链接，并在README.md中添加了缺失的部分，使多语言文档内容更加一致。 \
  **Feature Value**: 提高了文档的准确性和一致性，帮助用户更容易地找到相关信息，提升了用户体验。

- **Related PR**: [#2912](https://github.com/alibaba/higress/pull/2912) \
  **Contributor**: @hanxiantao \
  **Change Log**: 优化了hmac-auth-apisix插件的英文和中文文档，增加了更多配置说明细节，提升了文档清晰度。 \
  **Feature Value**: 通过更详细的文档解释，帮助开发者更好地理解和使用hmac-auth-apisix插件，提高了用户体验。

- **Related PR**: [#2880](https://github.com/alibaba/higress/pull/2880) \
  **Contributor**: @a6d9a6m \
  **Change Log**: 此PR修复了README.md、README_JP.md和README_ZH.md文件中的语法错误，确保文档的正确性和一致性。 \
  **Feature Value**: 通过修正文档中的语言错误，提升了文档的质量与可读性，帮助用户更好地理解项目信息。

- **Related PR**: [#2873](https://github.com/alibaba/higress/pull/2873) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR在非崩溃安全漏洞问题模板中增加了获取Higress运行时日志和配置的方法，帮助更好地调查问题。 \
  **Feature Value**: 通过提供更详细的日志和配置信息，用户可以更容易地诊断和解决问题，提高了问题处理的效率和准确性。

### 🧪 测试改进 (Testing)

- **Related PR**: [#2928](https://github.com/alibaba/higress/pull/2928) \
  **Contributor**: @rinfx \
  **Change Log**: 该PR更新了ai-security-guard组件的测试代码，增加了新的测试用例并调整了一些现有的测试逻辑。 \
  **Feature Value**: 通过改进ai-security-guard的测试覆盖率和准确性，提高了整个项目的稳定性和可靠性，有助于开发者更好地理解和维护相关功能。

---

## 📊 发布统计

- 🚀 新功能: 13项
- 🐛 Bug修复: 7项
- ♻️ 重构优化: 5项
- 📚 文档更新: 4项
- 🧪 测试改进: 1项

**总计**: 30项更改（包含2项重要更新）

感谢所有贡献者的辛勤付出！🎉


# Higress Console


## 📋 本次发布概览

本次发布包含 **4** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 1项
- **Bug修复**: 2项
- **文档更新**: 1项

### ⭐ 重点关注

本次发布包含 **1** 项重要更新，建议重点关注：

- **feat: Support using a known service in OpenAI LLM provider** ([#589](https://github.com/higress-group/higress-console/pull/589)): 该功能允许用户在OpenAI LLM提供者中利用现有的服务资源，从而扩展了系统的灵活性和可用性，为用户提供更多选择。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat: Support using a known service in OpenAI LLM provider

**相关PR**: [#589](https://github.com/higress-group/higress-console/pull/589) | **贡献者**: [@CH3CHO](https://github.com/CH3CHO)

**使用背景**

在许多应用场景中，开发者可能希望使用自定义的OpenAI服务实例，而不是默认的服务。这可能是由于特定的安全要求、性能优化或基础设施限制。此PR通过引入对已知服务的支持，满足了这些需求。目标用户群体包括需要高度定制化配置的企业级用户和技术专家。此功能解决了用户无法灵活选择和配置OpenAI服务的问题，提升了系统的适应性和用户体验。

**功能详述**

该PR主要实现了以下功能：1. 允许用户在配置OpenAI LLM提供者时指定自定义的服务。2. 修改了`OpenaiLlmProviderHandler`类，添加了`buildServiceSource`和`buildUpstreamService`方法，以处理自定义服务的逻辑。3. 在`WasmPluginInstanceService`接口中新增了带`internal`参数的删除方法，以支持更细粒度的控制。4. 更新了前端国际化资源文件，增加了与自定义服务相关的提示信息。核心技术要点在于对现有架构的扩展，使得系统能够识别并使用用户提供的自定义服务，同时保持了向后兼容性。

**使用方式**

启用和配置这个功能非常简单。首先，在创建或更新LLM提供者时，选择“自定义OpenAI服务”选项，并填写相应的服务主机和服务路径。然后，系统会自动使用这些自定义配置来连接OpenAI服务。典型的使用场景包括企业内部部署的OpenAI服务实例，或者需要特定安全策略的环境。注意事项包括确保输入的URL是有效的，并且服务主机和服务路径正确。最佳实践是进行充分的测试，确保自定义配置能够正常工作。

**功能价值**

这一新功能显著提升了系统的灵活性和可配置性，使用户能够根据自身需求选择最合适的OpenAI服务。对于需要高度定制化的企业级用户来说，这种灵活性尤为重要。此外，通过支持自定义服务，系统可以更好地集成到现有的基础设施中，提高了整体的稳定性和性能。这对于维护和扩展大型应用系统具有重要意义。总体而言，这一功能不仅增强了用户体验，还为系统带来了更高的可扩展性和可靠性。

---

## 📝 完整变更日志

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#591](https://github.com/higress-group/higress-console/pull/591) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR修复了在启用路由重写配置时未正确验证必填字段的问题，确保`host`和`newPath.path`都必须提供有效值以避免配置错误。 \
  **Feature Value**: 通过修正路由重写的验证逻辑，防止因配置不完整而导致的潜在错误，提升了系统的稳定性和用户体验。

- **Related PR**: [#590](https://github.com/higress-group/higress-console/pull/590) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修正了Route.customLabels处理逻辑中的错误，确保内置标签在更新时能够被正确排除。 \
  **Feature Value**: 解决了自定义标签与内置标签冲突的问题，保证了用户在更新路由设置时的灵活性和准确性。

### 📚 文档更新 (Documentation)

- **Related PR**: [#595](https://github.com/higress-group/higress-console/pull/595) \
  **Contributor**: @CH3CHO \
  **Change Log**: 移除了README.md中与项目无关的描述，并添加了代码格式指南，使得文档更加专注于项目本身。 \
  **Feature Value**: 通过更新README.md，使用户能够更清晰地了解项目的结构和代码规范要求，有助于新贡献者快速上手。

---

## 📊 发布统计

- 🚀 新功能: 1项
- 🐛 Bug修复: 2项
- 📚 文档更新: 1项

**总计**: 4项更改（包含1项重要更新）

感谢所有贡献者的辛勤付出！🎉


