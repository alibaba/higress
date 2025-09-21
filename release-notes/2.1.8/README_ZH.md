# Higress


## 📋 本次发布概览

本次发布包含 **29** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 13项
- **Bug修复**: 7项
- **重构优化**: 5项
- **文档更新**: 3项
- **测试改进**: 1项

### ⭐ 重点关注

本次发布包含 **3** 项重要更新，建议重点关注：

- **feat(gzip): add gzip configuration support and update default settings** ([#2867](https://github.com/alibaba/higress/pull/2867)): 通过启用Gzip功能并调整其默认参数，用户可以更灵活地控制资源压缩，从而优化网站性能和用户体验。
- **feat: add rag mcp server** ([#2930](https://github.com/alibaba/higress/pull/2930)): 此PR为用户引入了一个新的MCP服务器，能够帮助用户更有效地管理和检索知识，增强了系统的功能性和实用性。
- **refactor(mcp): use ECDS for golang filter configuration to avoid connection drain** ([#2931](https://github.com/alibaba/higress/pull/2931)): 改进了MCP服务器配置生成逻辑，利用ECDS服务来动态发现和加载过滤器配置，从而提升了系统的稳定性和用户体验。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat(gzip): add gzip configuration support and update default settings

**相关PR**: [#2867](https://github.com/alibaba/higress/pull/2867) | **贡献者**: [@Aias00](https://github.com/Aias00)

**使用背景**

在现代Web应用中，数据传输效率是一个关键因素。未压缩的HTTP响应可能会导致较高的带宽消耗和较长的加载时间，影响用户访问体验。此功能主要面向需要优化网络性能的应用开发者和运维人员。它解决了因响应数据过大而导致的传输延迟问题，特别是在移动或低带宽网络环境下尤为重要。此外，对于云原生环境中的微服务架构，Gzip压缩有助于减少跨服务调用时的网络开销，提高整体系统的响应速度。

**功能详述**

该PR为Higress网关添加了Gzip压缩配置支持，并更新了默认设置。具体实现包括：1. 在`helm/core/values.yaml`文件中新增了Gzip压缩相关的配置项，如是否启用、最小内容长度等；2. 修改了`pkg/ingress/kube/configmap/gzip.go`文件中的默认值，将Gzip压缩默认开启；3. 更新了相关测试用例以验证新功能的正确性。核心技术要点在于合理配置Gzip参数（例如压缩级别、窗口位数等）来平衡压缩率与CPU使用率之间的关系，确保在不显著增加服务器负载的情况下获得最佳压缩效果。

**使用方式**

要启用并配置Gzip压缩功能，用户可以通过修改`values.yaml`文件中的相应字段来定制化其行为。例如，设置`gzip.enable: true`以激活压缩功能，调整`minContentLength`来指定参与压缩的内容最小字节长度等。典型的使用场景是在部署新的微服务或者优化现有服务的网络性能时开启Gzip。需要注意的是，虽然Gzip能有效减小传输数据量，但过度压缩可能对服务器性能造成负面影响，因此建议根据实际情况调整压缩策略。

**功能价值**

此功能为用户带来了几个重要优势：1. 显著减少了HTTP响应的数据体积，加快了页面加载速度，提升了用户体验；2. 降低了网络流量成本，尤其适合移动设备或带宽有限的情况；3. 改善了系统整体性能，特别是在处理大量小文件请求时表现尤为突出。通过提供灵活的配置选项，使得管理员能够根据实际需求调整压缩强度，以达到最优的性能表现。这对于构建高效、响应迅速的Web服务至关重要。

---

### 2. feat: add rag mcp server

**相关PR**: [#2930](https://github.com/alibaba/higress/pull/2930) | **贡献者**: [@2456868764](https://github.com/2456868764)

**使用背景**

随着大规模语言模型（LLM）的发展，知识管理和检索变得越来越重要。传统的知识管理系统往往难以处理复杂的文本数据，并且缺乏与LLM的集成。Higress RAG MCP服务器解决了这一问题，它提供了一个统一的接口来管理和检索知识，并集成了多个LLM提供商（如OpenAI、DashScope等）。该功能的目标用户群体包括需要高效管理和检索知识的企业、开发者和研究人员。

**功能详述**

PR实现了RAG MCP服务器，添加了多个新文件和依赖项。主要功能包括知识管理（创建、删除和列出知识块）、搜索和聊天功能。核心技术要点包括使用递归分块器对文本进行分割，支持多种嵌入模型（如OpenAI和DashScope），以及与向量数据库（如Milvus）的集成。代码变更中新增了多个Go模块依赖，如`github.com/dlclark/regexp2`、`github.com/milvus-io/milvus-sdk-go/v2`和`github.com/pkoukk/tiktoken-go`，这些依赖分别用于正则表达式处理、向量数据库操作和文本编码。此外，还新增了HTTP客户端和配置结构，确保了系统的灵活性和可配置性。

**使用方式**

要启用和配置RAG MCP服务器，首先需要在`higress-config` ConfigMap中启用MCP服务器并设置相应的路径和匹配规则。然后，根据需求配置RAG系统的基础参数，如分块器类型、块大小和重叠，以及LLM和嵌入模型的相关信息。典型的使用场景包括从文本创建知识块、搜索相关知识和与LLM进行聊天交互。例如，可以通过发送POST请求到`/mcp-servers/rag/create-chunks-from-text`来创建知识块，或通过`/mcp-servers/rag/search`进行搜索。注意检查配置文件中的必填项，如API密钥和模型名称，以避免运行时错误。

**功能价值**

RAG MCP服务器为用户提供了强大的知识管理和检索能力，显著提升了系统的智能化水平。通过集成多种LLM和嵌入模型，用户可以根据具体需求选择最合适的工具。此外，灵活的配置选项使得系统能够适应不同的网络环境和数据规模。该功能不仅提高了系统的性能和稳定性，还增强了用户体验，使得知识管理和检索变得更加高效和便捷。对于生态而言，RAG MCP服务器为Higress平台增加了新的核心功能，进一步巩固了其在智能应用领域的领先地位。

---

### 3. refactor(mcp): use ECDS for golang filter configuration to avoid connection drain

**相关PR**: [#2931](https://github.com/alibaba/higress/pull/2931) | **贡献者**: [@johnlanni](https://github.com/johnlanni)

**使用背景**

在当前实现中，golang过滤器配置直接嵌入到HTTP_FILTER补丁中。这会导致配置更改时出现连接中断问题，主要原因包括Go map的顺序不一致以及HTTP_FILTER更新触发的监听器配置更改。为了解决这个问题，PR引入了Extension Configuration Discovery Service (ECDS)来分离HTTP_FILTER和实际的golang过滤器配置。目标用户群体主要是需要高可用性和低延迟的网络服务运维人员。

**功能详述**

这次重构将配置分为两部分：HTTP_FILTER仅包含带有config_discovery引用的过滤器，而EXTENSION_CONFIG则包含实际的golang过滤器配置。具体实现上，对`constructMcpSessionStruct`和`constructMcpServerStruct`方法进行了更新，以返回与EXTENSION_CONFIG兼容的格式。此外，还更新了单元测试以匹配新的配置结构。这项技术改进通过使用ECDS来解耦过滤器配置和HTTP_FILTER，从而避免了由于配置更改而导致的连接中断问题。

**使用方式**

要启用此功能，用户无需进行额外配置，因为这是内部实现的变化。典型使用场景是当用户需要频繁更新golang过滤器配置时，系统能够自动处理配置更新而不影响现有连接。尽管如此，用户仍需确保其环境支持ECDS，并且相关的Envoy代理版本兼容。最佳实践建议定期检查日志和监控数据，以验证配置更新过程中的连接稳定性。

**功能价值**

此次重构显著提升了系统的稳定性和可靠性，通过消除配置更改时的连接中断问题，使得系统更加健壮。对于依赖于低延迟和高可用性的应用场景来说，这一点尤为重要。此外，使用ECDS还增强了代码的可维护性和可扩展性，为未来的功能开发奠定了良好的基础。总体而言，这一改动不仅解决了现有的问题，也为整个生态系统带来了长远的好处。

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#2926](https://github.com/alibaba/higress/pull/2926) \
  **Contributor**: @rinfx \
  **Change Log**: 引入了对多模态、函数调用和思考的支持，扩展了vertex-ai的功能，主要修改了vertex相关的代码，增强了处理能力。 \
  **Feature Value**: 新增的多模态支持、函数调用以及思考功能极大地丰富了vertex-ai的应用场景，使得用户能够更加灵活地利用AI进行复杂任务处理，提高了用户体验。

- **Related PR**: [#2917](https://github.com/alibaba/higress/pull/2917) \
  **Contributor**: @Aias00 \
  **Change Log**: 新增了对Fireworks AI的支持，包括在代理插件中集成其服务以及相关的测试用例。 \
  **Feature Value**: 通过支持Fireworks AI，用户能够利用该AI服务的能力，增强了系统的多样性和灵活性，满足更多场景下的需求。

- **Related PR**: [#2907](https://github.com/alibaba/higress/pull/2907) \
  **Contributor**: @Aias00 \
  **Change Log**: 此PR通过升级wasm-go库支持outputSchema功能，更新了相关依赖版本。 \
  **Feature Value**: 增加了对outputSchema的支持，提升了插件处理数据输出格式的能力，使用户能够更灵活地定义和处理输出内容。

- **Related PR**: [#2897](https://github.com/alibaba/higress/pull/2897) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR在ai-proxy bedrock中引入了多模态支持和thinking功能，通过更新bedrock.go文件中的逻辑来实现。 \
  **Feature Value**: 新增的多模态及thinking功能扩展了ai-proxy bedrock的能力范围，使得用户能够利用更丰富的交互模式和思考流程，提升了系统的灵活性与实用性。

- **Related PR**: [#2891](https://github.com/alibaba/higress/pull/2891) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR实现了AI内容安全插件支持为不同消费者配置特定的检测服务，通过新增配置项如requestCheckService与responseCheckService来实现灵活的服务定制。 \
  **Feature Value**: 这项更新允许用户根据需求为不同的服务或客户端指定独立的内容安全检查策略，从而提高了系统的灵活性和安全性，更好地满足了多样化的业务场景。

- **Related PR**: [#2883](https://github.com/alibaba/higress/pull/2883) \
  **Contributor**: @Aias00 \
  **Change Log**: 此PR增加了对美团Longcat的支持，包括实现相应的请求处理逻辑与测试用例。 \
  **Feature Value**: 为用户提供更多AI服务提供商的选择，增强插件的适用性和灵活性，满足不同场景下的需求。

- **Related PR**: [#2844](https://github.com/alibaba/higress/pull/2844) \
  **Contributor**: @Aias00 \
  **Change Log**: 该PR增强了基于一致性哈希的负载均衡算法，新增了useSourceIp支持。通过使用源IP地址代替请求头中的信息来进行更精确的负载分配。 \
  **Feature Value**: 此功能改进了负载均衡策略，使得服务可以根据客户端的真实IP地址进行更合理和一致的路由选择，从而提高系统的稳定性和性能。

- **Related PR**: [#2843](https://github.com/alibaba/higress/pull/2843) \
  **Contributor**: @erasernoob \
  **Change Log**: 新增了对NVIDIA Triton Server的支持，包括配置信息和相关代码实现。 \
  **Feature Value**: 此功能允许用户通过OpenAI协议代理与NVIDIA Triton Inference Server进行交互，扩展了AI服务的兼容性和灵活性。

- **Related PR**: [#2806](https://github.com/alibaba/higress/pull/2806) \
  **Contributor**: @C-zhaozhou \
  **Change Log**: PR通过更新ai-security-guard插件，使其能够兼容MultiModalGuard接口，从而支持更多类型的内容安全检测。 \
  **Feature Value**: 增强了插件的多功能性，使得用户可以利用多模态API进行更全面的内容安全检查，提升了系统的灵活性和实用性。

- **Related PR**: [#2727](https://github.com/alibaba/higress/pull/2727) \
  **Contributor**: @Aias00 \
  **Change Log**: 此PR为OpenAI服务添加了端到端测试，包括非流式和流式请求的测试用例。 \
  **Feature Value**: 增加了对OpenAI集成的验证手段，确保了非流式与流式API请求的正确性和稳定性，提升了系统的可靠性和用户体验。

- **Related PR**: [#2593](https://github.com/alibaba/higress/pull/2593) \
  **Contributor**: @Xscaperrr \
  **Change Log**: 本PR通过在EnvoyFilter中添加WorkloadSelector字段，限制了过滤器的作用范围，确保其仅对Higress Gateway生效，从而避免与同命名空间内的Istio组件产生冲突。 \
  **Feature Value**: 这一改进增强了配置的精确控制能力，防止了不希望的影响扩散到其他服务或网关上，提升了系统稳定性和安全性。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#2938](https://github.com/alibaba/higress/pull/2938) \
  **Contributor**: @wydream \
  **Change Log**: 此PR修复了由于缺少AttackLevel字段支持导致的MultiModalGuard模式下提示攻击检测不工作的问题，通过更新数据结构和文档解决了该问题。 \
  **Feature Value**: 修复后，系统能够正确识别并阻止不同级别的提示攻击，提升了MultiModalGuard的安全性与可靠性，确保用户可以更安全地使用相关功能。

- **Related PR**: [#2904](https://github.com/alibaba/higress/pull/2904) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR修复了在初始化HTTP请求上下文时，原始的Authorization头可能被覆盖的问题。通过无条件地从请求头中获取并存储Authorization头到上下文中，确保了认证信息不会丢失。 \
  **Feature Value**: 解决了因原始认证头被意外覆盖而导致的安全问题和认证失败风险，提升了系统的稳定性和安全性，保障了用户数据的安全访问。

- **Related PR**: [#2899](https://github.com/alibaba/higress/pull/2899) \
  **Contributor**: @Jing-ze \
  **Change Log**: 此PR优化了MCP服务器主机模式匹配，减少了运行时解析开销，并修复了SSE消息格式化问题，移除了未使用的字段以提高内存使用效率。 \
  **Feature Value**: 通过优化主机模式匹配和解决SSE消息中的换行符问题，提高了系统性能和消息准确性，提升了用户体验。

- **Related PR**: [#2892](https://github.com/alibaba/higress/pull/2892) \
  **Contributor**: @johnlanni \
  **Change Log**: 修复了Claude工具返回数组格式的content字段导致的JSON解码错误，并移除了重复的代码结构以提高代码质量。 \
  **Feature Value**: 这一更改解决了特定场景下的数据解析问题，提高了系统的稳定性和开发者的调试效率，通过减少冗余代码还简化了维护工作。

- **Related PR**: [#2882](https://github.com/alibaba/higress/pull/2882) \
  **Contributor**: @johnlanni \
  **Change Log**: 修复了Claude协议在流式传输场景下的转换错误，改进了工具调用状态跟踪和连接阻塞问题。 \
  **Feature Value**: 提高了Claude到OpenAI双向转换的可靠性，解决了流式响应处理中的关键问题，提升了用户体验。

- **Related PR**: [#2865](https://github.com/alibaba/higress/pull/2865) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: 修复了当SSE事件被分割成多个块时，SSE连接会被阻塞的问题，通过添加缓存机制来处理多块事件。 \
  **Feature Value**: 此修复确保了即使SSE响应被拆分为多个部分，数据传输也不会中断，从而提升了系统的稳定性和用户体验。

- **Related PR**: [#2859](https://github.com/alibaba/higress/pull/2859) \
  **Contributor**: @lcfang \
  **Change Log**: 通过在mcpbridge中新增vport元素，固化serviceEntry中的端口配置，确保当后端服务实例端口发生变化时，路由规则不会失效。同时调整了相关数据结构和逻辑。 \
  **Feature Value**: 解决了因后端服务实例端口不一致导致的兼容性问题，提高了系统的稳定性和一致性，减少了因端口变化引起的服务中断，提升了用户体验。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#2933](https://github.com/alibaba/higress/pull/2933) \
  **Contributor**: @rinfx \
  **Change Log**: 移除了bedrock和vertex模块中重复的think标签定义，减少了冗余代码并提高了代码的可维护性。 \
  **Feature Value**: 通过消除冗余代码，该PR提升了项目的整洁度与维护效率，间接地为使用者提供了一个更加稳定可靠的系统环境。

- **Related PR**: [#2927](https://github.com/alibaba/higress/pull/2927) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR修改了`getAPIName`函数中对API名称提取逻辑的条件判断，将原来的固定长度检查改为至少包含3部分的更灵活方式。 \
  **Feature Value**: 通过放宽API名称提取的限制条件，提高了系统对于不同格式API字符串的兼容性和鲁棒性，减少了因格式不完全匹配导致的错误。

- **Related PR**: [#2922](https://github.com/alibaba/higress/pull/2922) \
  **Contributor**: @daixijun \
  **Change Log**: 将Higress SDK引用的包名从github.com/alibaba/higress升级到v2版本，确保项目能够引用最新的SDK版本。 \
  **Feature Value**: 通过更新包名至最新版本，解决了因版本不匹配导致的依赖引入问题，提升了项目的兼容性和可维护性。

- **Related PR**: [#2890](https://github.com/alibaba/higress/pull/2890) \
  **Contributor**: @johnlanni \
  **Change Log**: 引入了HostMatcher结构体，使用专门的匹配类型替代了基于正则表达式的匹配方式，并实现了端口剥离逻辑以正确处理带有端口号的主机头。 \
  **Feature Value**: 通过改进主机匹配逻辑提升了代码的可维护性和执行效率，使得系统在处理复杂主机头时更加准确高效，间接增强了系统的稳定性和响应速度。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2912](https://github.com/alibaba/higress/pull/2912) \
  **Contributor**: @hanxiantao \
  **Change Log**: 该PR优化了hmac-auth-apisix插件的中英文文档，增加了路由名称与域名匹配配置说明，并对文档结构进行了调整以提高可读性。 \
  **Feature Value**: 通过详细说明hmac-auth-apisix插件的使用方法和配置规则，帮助开发者更好地理解和使用该认证机制，从而提升API网关的安全性和用户体验。

- **Related PR**: [#2880](https://github.com/alibaba/higress/pull/2880) \
  **Contributor**: @a6d9a6m \
  **Change Log**: 此PR修复了README.md及其日语和中文版本中的语法错误，确保文档的准确性和一致性。 \
  **Feature Value**: 通过修正文档中的错误，提高了用户对项目的理解度，增强了用户体验并维护了项目的专业形象。

- **Related PR**: [#2873](https://github.com/alibaba/higress/pull/2873) \
  **Contributor**: @CH3CHO \
  **Change Log**: 在问题模板中添加了日志和配置获取方法的相关说明，帮助用户更好地提供故障排查所需信息。 \
  **Feature Value**: 通过改进问题模板，引导用户提供更全面的日志和配置信息，有助于提高问题定位的效率，使维护者能够更快地解决问题。

### 🧪 测试改进 (Testing)

- **Related PR**: [#2928](https://github.com/alibaba/higress/pull/2928) \
  **Contributor**: @rinfx \
  **Change Log**: 更新了ai-security-guard插件的单元测试用例，增加了新的测试场景并优化了现有测试逻辑。 \
  **Feature Value**: 通过增强和更新测试案例，提高了ai-security-guard功能的稳定性和可靠性，确保新特性或修复不会影响到现有的安全功能。

---

## 📊 发布统计

- 🚀 新功能: 13项
- 🐛 Bug修复: 7项
- ♻️ 重构优化: 5项
- 📚 文档更新: 3项
- 🧪 测试改进: 1项

**总计**: 29项更改（包含3项重要更新）

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

- **feat: Support using a known service in OpenAI LLM provider** ([#589](https://github.com/higress-group/higress-console/pull/589)): 新增的支持能够使用户更方便地集成和使用OpenAI的LLM服务，提升了系统的灵活性与可用性，为用户提供更多选择。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat: Support using a known service in OpenAI LLM provider

**相关PR**: [#589](https://github.com/higress-group/higress-console/pull/589) | **贡献者**: [@CH3CHO](https://github.com/CH3CHO)

**使用背景**

在当前系统中，用户在配置OpenAI LLM提供商时只能使用默认的服务地址。然而，在实际应用中，用户可能需要使用自己的代理服务器或其他已知服务来与OpenAI进行交互。这种需求可能是出于性能优化、安全性增强或特定业务需求考虑。例如，企业可能希望将所有对外API调用通过内部代理服务器路由，以确保数据的安全性和合规性。此外，有些用户可能已经在其基础设施中部署了OpenAI服务的镜像或代理，因此需要一种灵活的方式来指定这些服务。目标用户群体主要是开发人员、系统管理员以及需要高度定制化OpenAI服务的企业。

**功能详述**

此次更新主要实现了以下功能：
1. **支持自定义服务配置**：新增了`buildServiceSource`和`buildUpstreamService`方法，允许用户通过配置指定自定义的OpenAI服务。如果用户提供了自定义服务配置，则直接使用该配置而不创建新的服务源。
2. **增强Wasm插件管理**：在`WasmPluginInstanceService`接口中增加了删除插件实例的方法，支持传入`internal`参数，这使得对内部资源的管理更加灵活。
3. **国际化资源检查**：在前端国际化资源检查中新增了相关键值，以支持新添加的功能在不同语言环境下的正确显示。核心技术创新在于提供了一种灵活且统一的方式处理自定义服务配置，并且通过增加对内部资源管理的支持，提高了系统的可维护性。

**使用方式**

启用和配置此功能的具体步骤如下：
1. 在配置文件中找到OpenAI LLM提供商的相关设置部分。
2. 添加或修改`openaiCustomServiceHost`和`openaiCustomServicePath`字段，分别指定自定义服务的主机名和路径。例如：
   ```json
   {
     "provider": "OpenAI",
     "openaiCustomServiceHost": "api.openai.internal",
     "openaiCustomServicePath": "/v1"
   }
   ```
3. 如果需要进一步控制内部资源的管理，可以在删除Wasm插件实例时传递`internal`参数，如`wasmPluginInstanceService.delete(WasmPluginInstanceScope.ROUTE, routeName, BuiltInPluginName.MODEL_MAPPER, true);`。
4. 系统会根据提供的自定义服务配置自动连接到指定的服务地址，从而实现更灵活的集成方式。注意事项包括确保自定义服务地址的有效性和可达性，同时要确保相关的安全策略已正确配置。

**功能价值**

这一新功能为用户带来了显著的好处：
1. **灵活性提升**：用户可以根据自身需求选择不同的OpenAI服务配置方法，无论是使用外部服务还是内部代理。
2. **性能优化**：通过使用内部代理或本地镜像服务，可以减少网络延迟，提高响应速度。
3. **安全性增强**：对于有严格安全要求的企业来说，能够通过内部网络访问OpenAI服务有助于确保数据传输的安全性。
4. **更好的用户体验**：前端界面提供了相应的多语言支持，确保用户在不同语言环境下能够顺畅地配置和使用该功能。整体上，这一功能不仅满足了用户的多样化需求，还进一步提升了整个系统的可靠性和用户满意度。

---

## 📝 完整变更日志

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#591](https://github.com/higress-group/higress-console/pull/591) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修正了路由重写配置的条件检查逻辑，确保在启用状态下，必须同时提供host和newPath.path的有效值，解决了由于缺少必填项导致的错误。 \
  **Feature Value**: 通过修复路由重写配置中的验证逻辑问题，提高了系统的稳定性和用户体验，避免了因配置不完整而引发的功能异常。

- **Related PR**: [#590](https://github.com/higress-group/higress-console/pull/590) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了Route.customLabels处理逻辑中的错误，排除内置标签以确保在更新过程中可以正确移除。 \
  **Feature Value**: 此修复解决了自定义标签与内置标签混淆的问题，提升了系统的准确性和用户体验，特别是在进行路由配置更新时。

### 📚 文档更新 (Documentation)

- **Related PR**: [#595](https://github.com/higress-group/higress-console/pull/595) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR移除了README.md中与项目无关的部分描述，并添加了代码格式规范说明，总共修改了72行。 \
  **Feature Value**: 通过清理不必要的信息并加入格式指南，提高了文档的质量和可读性，帮助开发者更好地理解和遵循项目的贡献规则。

---

## 📊 发布统计

- 🚀 新功能: 1项
- 🐛 Bug修复: 2项
- 📚 文档更新: 1项

**总计**: 4项更改（包含1项重要更新）

感谢所有贡献者的辛勤付出！🎉


