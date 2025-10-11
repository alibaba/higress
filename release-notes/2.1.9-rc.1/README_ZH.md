# Higress


## 📋 本次发布概览

本次发布包含 **11** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 3项
- **Bug修复**: 5项
- **重构优化**: 1项
- **文档更新**: 2项

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#2978](https://github.com/alibaba/higress/pull/2978) \
  **Contributor**: @rinfx \
  **Change Log**: 在key-auth插件中，无论认证是否通过，在确定消费者名称后都会记录下来。这通过向HTTP请求头添加X-Mse-Consumer字段来实现。 \
  **Feature Value**: 该功能允许系统更早地获取并记录消费者的名称，这对于日志记录和后续处理非常重要，可以提高系统的可追踪性和透明度。

- **Related PR**: [#2968](https://github.com/alibaba/higress/pull/2968) \
  **Contributor**: @2456868764 \
  **Change Log**: 此PR引入了矢量数据库的映射核心功能，包括字段映射系统和索引配置管理，支持多种索引类型。 \
  **Feature Value**: 通过提供灵活的字段映射和索引配置能力，使得用户能够更方便地与不同数据库架构进行集成，提升了系统的兼容性和灵活性。

- **Related PR**: [#2943](https://github.com/alibaba/higress/pull/2943) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: 增加了自定义系统提示的功能，让用户在生成发布笔记时能添加个性化说明。通过修改GitHub Actions工作流文件实现。 \
  **Feature Value**: 此功能允许用户在生成发布笔记时加入自定义的系统提示，增强了发布笔记的灵活性与信息丰富度，提升了用户体验。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#2973](https://github.com/alibaba/higress/pull/2973) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR修复了Higress 2.1.8版本中`mcp-session`过滤器不支持将`match_rule_domain`设置为空字符串的问题，通过使用通配符来匹配所有域以消除兼容性风险。 \
  **Feature Value**: 解决了因特定配置导致的兼容性问题，确保用户在升级或配置过程中不会遇到由空字符串设置引起的错误，提升了系统的稳定性和用户体验。

- **Related PR**: [#2952](https://github.com/alibaba/higress/pull/2952) \
  **Contributor**: @Erica177 \
  **Change Log**: 修正了ToolSecurity结构体中Id字段的JSON标签，从type改为id，确保数据序列化时正确映射。 \
  **Feature Value**: 此修复解决了因字段映射错误导致的数据不一致问题，提高了系统的稳定性和数据准确性。

- **Related PR**: [#2948](https://github.com/alibaba/higress/pull/2948) \
  **Contributor**: @johnlanni \
  **Change Log**: 修复了Azure服务URL类型检测逻辑，支持自定义完整路径。增强了Azure OpenAI响应API处理能力，并改进了流式事件解析中的边缘情况。 \
  **Feature Value**: 确保了与Azure OpenAI服务的兼容性更强，提升了错误处理能力和用户体验，特别是在使用非标准路径或流式响应时。

- **Related PR**: [#2942](https://github.com/alibaba/higress/pull/2942) \
  **Contributor**: @2456868764 \
  **Change Log**: 修复了LLM提供者为空的问题，并优化了文档和提示信息。具体包括更新README.md以改进说明，调整LLM默认模型。 \
  **Feature Value**: 通过增强LLM提供者的初始化健壮性并优化相关文档，提高了系统的稳定性和用户体验，使用户能更清晰地了解系统配置和使用方法。

- **Related PR**: [#2941](https://github.com/alibaba/higress/pull/2941) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR修复了与旧配置兼容性的问题，确保系统能够正确处理过时的配置参数，避免因配置变更导致的潜在错误。 \
  **Feature Value**: 通过支持旧版本配置，增强了系统的向后兼容能力，减少了因升级或配置调整给用户带来的不便，提升了用户体验。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#2945](https://github.com/alibaba/higress/pull/2945) \
  **Contributor**: @rinfx \
  **Change Log**: 优化全局最小请求数选pod的逻辑，更新了ai-load-balancer相关的Lua脚本代码，减少了不必要的检查和提高了性能。 \
  **Feature Value**: 通过改进负载均衡策略中的最小请求数算法，提升了系统的响应速度与资源分配效率，使用户可以更高效地利用集群资源。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2965](https://github.com/alibaba/higress/pull/2965) \
  **Contributor**: @CH3CHO \
  **Change Log**: 更新了ai-proxy插件README文件中azureServiceUrl字段的描述，以提供更清晰准确的信息。 \
  **Feature Value**: 通过改进文档中的描述，使得用户能够更好地理解如何配置Azure OpenAI服务URL，从而提高使用体验和配置准确性。

- **Related PR**: [#2940](https://github.com/alibaba/higress/pull/2940) \
  **Contributor**: @github-actions[bot] \
  **Change Log**: 此PR为2.1.8版本添加了英文和中文的发布说明文档，详细记录了该版本中的30项更新。 \
  **Feature Value**: 通过提供详细的发布笔记，用户能够更容易地理解新版本中包含的功能改进、Bug修复等信息，从而更好地利用新特性。

---

## 📊 发布统计

- 🚀 新功能: 3项
- 🐛 Bug修复: 5项
- ♻️ 重构优化: 1项
- 📚 文档更新: 2项

**总计**: 11项更改

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

- **feat: Support using a known service in OpenAI LLM provider** ([#589](https://github.com/higress-group/higress-console/pull/589)): 此功能允许用户在OpenAI LLM中使用预定义的服务，从而提高开发效率与灵活性，满足更广泛的应用场景需求。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat: Support using a known service in OpenAI LLM provider

**相关PR**: [#589](https://github.com/higress-group/higress-console/pull/589) | **贡献者**: [@CH3CHO](https://github.com/CH3CHO)

**使用背景**

随着越来越多的组织和服务开始采用大型语言模型（LLM），对这些模型的访问和管理变得尤为重要。特别是在需要与特定已知服务进行集成的情况下，例如内部部署的OpenAI API服务器或自定义的API端点。此功能解决了在Higress系统中直接支持自定义OpenAI服务的需求，使用户能够更灵活地配置和使用他们的服务。目标用户群体包括但不限于开发者、运维人员以及需要高度定制化解决方案的企业。

**功能详述**

此次更新主要集中在`OpenaiLlmProviderHandler`类中，引入了自定义服务源的支持。通过添加新的配置项如`openaiCustomServiceName`和`openaiCustomServicePort`，用户现在可以直接指定其自定义OpenAI服务的详细信息。此外，代码还改进了处理逻辑，如果指定了自定义上游服务，则不会为默认服务创建服务源。这种设计不仅简化了配置流程，也提高了系统的可扩展性。技术上，这是通过重写`buildServiceSource`和`buildUpstreamService`方法来实现的，其中包含了对用户自定义设置的检查。

**使用方式**

要启用并配置此新功能，用户首先需要在其OpenAI LLM提供商设置中提供必要的自定义服务信息。这通常涉及到填写诸如自定义服务名称、主机地址及端口号等字段。具体步骤通常是：1. 在Higress控制台或对应的配置文件中找到相关LLM提供者的设置部分；2. 根据提示填入相应的自定义服务详情；3. 保存更改。一个典型的应用场景可能是公司希望使用自己内部托管的OpenAI接口而不是官方提供的公共接口。需要注意的是，确保所提供的自定义服务地址是准确无误的，并且网络可达非常重要。

**功能价值**

这项功能极大地丰富了Higress平台对于不同环境下的适应能力，特别是对于那些需要高度定制化的应用场景来说尤为关键。它不仅提升了用户体验——让配置过程变得更加直观简单，同时也促进了整体系统的稳定性和安全性，因为现在可以直接利用受信任的内部资源。从长远来看，这样的增强有助于构建更加健壮的生态系统，鼓励更多创新性的应用开发。

---

## 📝 完整变更日志

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#591](https://github.com/higress-group/higress-console/pull/591) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了启用路由重写时未正确验证必填字段的问题，确保`host`和`newPath.path`在启用状态下都必须提供有效值。 \
  **Feature Value**: 此修复提高了系统配置的准确性与健壮性，防止因配置不完整导致的功能异常，提升了用户体验。

- **Related PR**: [#590](https://github.com/higress-group/higress-console/pull/590) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了Route.customLabels处理逻辑中的错误，确保内置标签在更新时能够被正确排除。 \
  **Feature Value**: 解决了用户在更新Route时自定义标签与内置标签冲突的问题，提升了系统的稳定性和用户体验。

### 📚 文档更新 (Documentation)

- **Related PR**: [#595](https://github.com/higress-group/higress-console/pull/595) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR更新了README.md文件，移除了非项目级别的描述，并添加了代码格式指南。 \
  **Feature Value**: 通过清理无关信息并提供格式化建议，帮助开发者更清晰地理解项目文档，促进了代码贡献的一致性和可读性。

---

## 📊 发布统计

- 🚀 新功能: 1项
- 🐛 Bug修复: 2项
- 📚 文档更新: 1项

**总计**: 4项更改（包含1项重要更新）

感谢所有贡献者的辛勤付出！🎉


