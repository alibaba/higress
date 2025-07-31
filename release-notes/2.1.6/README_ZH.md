# Higress


## 📋 本次发布概览

本次发布包含 **31** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 13项
- **Bug修复**: 5项
- **重构优化**: 7项
- **文档更新**: 5项
- **测试改进**: 1项

### ⭐ 重点关注

本次发布包含 **2** 项重要更新，建议重点关注：

- **feat: Add Higress API MCP server** ([#2517](https://github.com/alibaba/higress/pull/2517)): 新增的Higress API MCP服务器功能增强了AI Agent对Higress资源的管理能力，支持通过MCP进行路由和服务的增删改查操作，提升了系统的灵活性和可维护性。
- **Migrate WASM Go Plugins to New SDK and Go 1.24** ([#2532](https://github.com/alibaba/higress/pull/2532)): 将开发 Wasm Go 插件的底层编译依赖从 TinyGo 替换为了原生的 Go 1.24，提高了插件的兼容性和性能，确保了与最新技术栈的一致性，为用户提供更稳定和高效的插件支持。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat: Add Higress API MCP server

**相关PR**: [#2517](https://github.com/alibaba/higress/pull/2517) | **贡献者**: [@cr7258](https://github.com/cr7258)

**使用背景**

在现代微服务架构中，API网关作为入口点，需要灵活且强大的配置管理能力。Higress作为一个高性能的API网关，提供了丰富的功能来管理路由、服务来源和插件。然而，现有的配置管理方式可能不够灵活，无法满足复杂的运维需求。为了解决这个问题，PR #2517引入了Higress API MCP Server，通过Higress Console API提供了一种新的配置管理方式。该功能主要针对需要对Higress进行高级配置和动态管理的运维人员和开发者。

**功能详述**

此次变更实现了Higress API MCP Server，通过golang-filter重新实现了一个MCP服务器，能够调用Higress Console API进行路由、服务来源和插件的管理。具体实现包括：
1. 新增了HigressClient类，用于处理与Higress Console API的交互。
2. 实现了多种管理工具，如路由管理（list-routes, get-route, add-route, update-route）、服务来源管理（list-service-sources, get-service-source, add-service-source, update-service-source）和插件管理（get-plugin, delete-plugin, update-request-block-plugin）。
3. 修改了相关配置文件和README文档，提供了详细的配置示例和使用说明。
4. 代码变更涉及多个文件，包括`config.go`、`client.go`、`server.go`等，确保了功能的完整性和可扩展性。

**使用方式**

启用和配置Higress API MCP Server的步骤如下：
1. 在Higress的ConfigMap中添加MCP Server的配置，指定Higress Console的URL地址、用户名和密码。
2. 启动Higress Gateway时，确保`mcpServer.enable`设置为`true`。
3. 使用提供的工具命令（如`list-routes`、`add-route`等）进行路由、服务来源和插件的管理。
4. 配置示例：
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: higress-config
  namespace: higress-system
data:
  higress: |-
    mcpServer:
      sse_path_suffix: /sse
      enable: true
      servers:
        - name: higress-api-mcp-server
          path: /higress-api
          type: higress-api
          config:
            higressURL: http://higress-console.higress-system.svc.cluster.local
            username: admin
            password: <password>
```
注意事项：
- 确保Higress Console的URL地址、用户名和密码正确无误。
- 配置中的密码建议使用环境变量或加密存储，以提高安全性。

**功能价值**

Higress API MCP Server为用户带来了以下具体好处：
1. **提升运维效率**：通过统一的MCP接口，用户可以更方便地通过AI Agent管理和配置Higress的各项资源，减少手动操作的复杂性和出错率。
2. **增强系统灵活性**：支持动态管理和更新路由、服务来源和插件，使系统更加灵活，能够快速响应业务需求的变化。
3. **提高系统稳定性**：通过自动化的配置管理，减少了人为错误的可能性，从而提高了系统的稳定性和可靠性。
4. **易于集成**：Higress API MCP Server的设计使得其易于与其他AI系统和工具集成（例如Cursor，CherryStudio等），便于构建完整的自动化运维体系。

---

### 2. Migrate WASM Go Plugins to New SDK and Go 1.24

**相关PR**: [#2532](https://github.com/alibaba/higress/pull/2532) | **贡献者**: [@erasernoob](https://github.com/erasernoob)

**使用背景**

随着Go语言的发展，新版本提供了许多性能优化和安全改进。本PR旨在将WASM Go插件从旧的SDK迁移到新的SDK，并将Go版本升级到1.24。这不仅解决了旧版本中的一些已知问题，还为未来的功能扩展和性能优化铺平了道路。目标用户群体包括使用Higress进行微服务管理和流量控制的开发者和运维人员。

**功能详述**

此PR主要实现了以下功能：1) 更新了构建和测试插件的工作流文件，以支持新的Go版本；2) 修改了Dockerfile和Makefile，移除了对TinyGo的支持，改为直接使用标准Go编译器生成WASM文件；3) 更新了go.mod文件，引用了新的包路径和版本；4) 调整了日志库的导入路径，统一使用新的日志库。这些变更使得插件能够更好地利用Go 1.24的新特性，如更好的垃圾回收机制和更高效的编译器优化。此外，移除对TinyGo的支持简化了构建过程，减少了潜在的兼容性问题。

**使用方式**

要启用和配置这个功能，首先需要确保你的开发环境已经安装了Go 1.24。然后，你可以通过修改项目的Makefile和Dockerfile来指定新的构建参数。例如，在Makefile中设置`GO_VERSION ?= 1.24.0`，在Dockerfile中使用`ARG BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.0-oras1.0.0`。典型的使用场景是当你需要在Higress中部署新的WASM插件时。最佳实践包括定期更新依赖库至最新版本，并确保所有相关代码都已适配新版本。

**功能价值**

此次重构为用户带来了多方面的好处：1) 提高了插件的运行效率和稳定性，得益于Go 1.24的新特性和优化；2) 简化了构建流程，减少了对第三方工具（如TinyGo）的依赖，降低了维护成本；3) 统一了代码风格和依赖管理，提高了项目的可读性和可维护性；4) 增强了系统的安全性，通过采用最新的Go版本修复了一些已知的安全漏洞。这些改进使得Higress生态系统更加健壮，为用户提供了一个更加强大和可靠的微服务管理平台。

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#2679](https://github.com/alibaba/higress/pull/2679) \
  **Contributor**: @erasernoob \
  **Change Log**: 此PR在镜像注解中增加了对外部服务FQDN的支持，并为此新增了相应的测试用例，以确保新功能的正确性和稳定性。 \
  **Feature Value**: 允许用户通过指定外部FQDN作为镜像目标，提升了系统的灵活性和适用范围，便于集成更多外部资源。

- **Related PR**: [#2667](https://github.com/alibaba/higress/pull/2667) \
  **Contributor**: @hanxiantao \
  **Change Log**: 此PR为AI Token限流插件添加了支持设置全局路由限流阈值的功能，同时优化了与cluster-key-rate-limit插件相关的基础逻辑，并改进了日志提示。 \
  **Feature Value**: 通过增加对全局限流阈值的支持，使得用户可以更灵活地管理流量，避免了单一路由因流量过大而影响整个系统的稳定性。

- **Related PR**: [#2652](https://github.com/alibaba/higress/pull/2652) \
  **Contributor**: @OxalisCu \
  **Change Log**: 此PR在ai-proxy插件中为LLM流式请求添加了首字节超时支持，通过修改provider.go文件实现了这一功能。 \
  **Feature Value**: 该功能允许用户为LLM流式请求设置首字节超时时间，提高了系统的稳定性和用户体验。

- **Related PR**: [#2650](https://github.com/alibaba/higress/pull/2650) \
  **Contributor**: @zhangjingcn \
  **Change Log**: 此PR实现了从Nacos MCP注册中心获取ErrorResponseTemplate配置的功能，通过修改mcp_model.go和watcher.go两个文件来支持新的元数据处理。 \
  **Feature Value**: 这项功能增强了系统与Nacos MCP注册中心的集成度，使得在遇到错误时能够使用自定义的响应模板，提升了错误处理的灵活性和用户体验。

- **Related PR**: [#2649](https://github.com/alibaba/higress/pull/2649) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR增加了对Azure OpenAI三种不同URL格式的支持，并确保了`api-version`参数总是必需的。通过修改和添加多个Go文件中的代码，包括处理请求头、路径解析等关键部分。 \
  **Feature Value**: 增强了插件与Azure OpenAI服务集成的能力，允许用户采用更多样化的URL配置方式来部署其模型，从而提高了系统的灵活性和兼容性。

- **Related PR**: [#2648](https://github.com/alibaba/higress/pull/2648) \
  **Contributor**: @daixijun \
  **Change Log**: 该PR实现了qwen Provider对anthropic /v1/messages接口的支持，通过在qwen.go文件中添加了相关的代码逻辑。 \
  **Feature Value**: 新增了对于Anthropic消息接口的支持，使得用户能够利用Qwen代理更多人工智能服务，从而扩展了系统的应用范围和功能。

- **Related PR**: [#2585](https://github.com/alibaba/higress/pull/2585) \
  **Contributor**: @akolotov \
  **Change Log**: 此PR为Blockscout MCP服务器提供了配置文件，包括详细的README文档和YAML格式的配置设置。 \
  **Feature Value**: 通过集成Blockscout MCP服务器，用户可以更方便地检查和分析EVM兼容的区块链，提升了系统的功能性和用户体验。

- **Related PR**: [#2551](https://github.com/alibaba/higress/pull/2551) \
  **Contributor**: @daixijun \
  **Change Log**: 此PR为AI代理插件添加了对Anthropic和Gemini API的支持，扩展了系统处理不同来源AI请求的能力。 \
  **Feature Value**: 通过引入新的API支持，用户可以更灵活地选择使用不同的AI服务提供商，增强了系统的多样性和可用性。

- **Related PR**: [#2542](https://github.com/alibaba/higress/pull/2542) \
  **Contributor**: @daixijun \
  **Change Log**: 该PR新增了对images、audio、responses接口Token使用的统计功能，并将相关工具函数定义为公共函数，以减少重复代码。 \
  **Feature Value**: 通过支持更多接口的Token使用统计，用户能够更全面地了解和管理资源消耗情况，从而优化成本控制。

- **Related PR**: [#2537](https://github.com/alibaba/higress/pull/2537) \
  **Contributor**: @wydream \
  **Change Log**: 此PR添加了对Qwen模型的文字重排序功能支持，通过在AI代理插件中引入新的API路径来实现。 \
  **Feature Value**: 新增的Qwen文字重排序功能扩展了平台的文本处理能力，使用户能够利用更先进的模型进行内容优化和排序。

- **Related PR**: [#2535](https://github.com/alibaba/higress/pull/2535) \
  **Contributor**: @wydream \
  **Change Log**: 此PR引入了`basePath`和`basePathHandling`选项，用于灵活处理请求路径。通过设置`removePrefix`或`prepend`来决定如何使用basePath。 \
  **Feature Value**: 新增的选项使用户能够更灵活地管理API网关与后端服务之间的路径映射，增强了系统的适应性和灵活性。

- **Related PR**: [#2499](https://github.com/alibaba/higress/pull/2499) \
  **Contributor**: @heimanba \
  **Change Log**: 此PR在GrayConfig结构中引入了UseManifestAsEntry字段，并更新了相关函数以支持该配置，同时修改了README文档并调整了HTML响应处理逻辑。 \
  **Feature Value**: 新增的useManifestAsEntry配置项允许用户更灵活地控制首页请求是否使用缓存，从而增强了系统的灵活性和用户体验。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#2687](https://github.com/alibaba/higress/pull/2687) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: 修复了当mcp client使用describeTable工具时出现的SQL错误，确保了对Postgres表描述功能的正确性。 \
  **Feature Value**: 此修复提高了系统的稳定性和可靠性，确保用户在使用mcp-server与Postgres数据库交互时能够准确获取表信息，提升了用户体验。

- **Related PR**: [#2662](https://github.com/alibaba/higress/pull/2662) \
  **Contributor**: @johnlanni \
  **Change Log**: 解决了Envoy中的两个问题：修复了proxy-wasm-cpp-host中的内存泄漏，以及当ppv2启用时端口映射不正确导致的404错误。 \
  **Feature Value**: 通过修复内存泄漏和端口映射问题，提高了系统的稳定性和可靠性，减少了资源浪费，并确保了正确的路由配置。

- **Related PR**: [#2656](https://github.com/alibaba/higress/pull/2656) \
  **Contributor**: @co63oc \
  **Change Log**: 此PR修正了多个文件中的拼写错误，包括常量名、函数名和插件名称等，确保了代码的一致性和可读性。 \
  **Feature Value**: 通过修复这些拼写错误，提高了代码质量，避免了由于命名不一致导致的潜在逻辑错误或编译失败，增强了系统的稳定性和用户体验。

- **Related PR**: [#2623](https://github.com/alibaba/higress/pull/2623) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: 修复了特殊字符在翻译过程中导致的问题，通过调整生成和处理JSON数据的方法来避免潜在的JSON结构破坏。 \
  **Feature Value**: 该修复确保了包含特殊字符的内容能够正确地被处理和显示，从而提升了系统的稳定性和用户体验。

- **Related PR**: [#2507](https://github.com/alibaba/higress/pull/2507) \
  **Contributor**: @hongzhouzi \
  **Change Log**: 修正了在arm64架构上编译golang-filter.so时因安装了x86工具链而导致的错误，通过确保安装与目标架构匹配的工具来解决问题。 \
  **Feature Value**: 此修复解决了特定硬件架构（arm64）上的编译问题，使得项目能够在更多种类的处理器上成功构建，增加了软件的兼容性和用户基础。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#2673](https://github.com/alibaba/higress/pull/2673) \
  **Contributor**: @johnlanni \
  **Change Log**: 改进了`findEndpointUrl`函数，使其能够处理多个SSE消息，而不仅仅是第一个。这涉及代码逻辑的优化和新增单元测试。 \
  **Feature Value**: 增强了MCP端点解析器的功能，使其更加健壮，可以更好地兼容不同后端服务发送的消息格式，提升了系统的稳定性和用户体验。

- **Related PR**: [#2661](https://github.com/alibaba/higress/pull/2661) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR放宽了DNS服务域名验证规则，通过修改正则表达式来允许更灵活的域名格式。 \
  **Feature Value**: 放宽域名验证有助于提高系统的灵活性和兼容性，使用户能够使用更多样化的域名配置，从而提升用户体验。

- **Related PR**: [#2639](https://github.com/alibaba/higress/pull/2639) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR通过在特定插件中禁用重新路由，优化了请求处理流程。具体地，在不需要重新匹配路由的官方插件中统一设置了ctx.DisableReroute。 \
  **Feature Value**: 优化了插件的性能，减少了不必要的路由重定向，提升了应用的整体效率和响应速度，为用户提供了更流畅的体验。

- **Related PR**: [#2615](https://github.com/alibaba/higress/pull/2615) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR移除了wasm-go插件的Dockerfile和Makefile中的EXTRA_TAGS变量，并更新了相关配置文件，简化了构建过程。 \
  **Feature Value**: 通过清理不再使用的配置项，该改动使得项目结构更加简洁清晰，有助于减少潜在的维护成本，同时保持了现有功能的稳定性。

- **Related PR**: [#2598](https://github.com/alibaba/higress/pull/2598) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR将WASM构建器镜像中的Go版本更新至1.24.4，同时简化了DockerfileBuilder文件的内容。 \
  **Feature Value**: 通过升级Go版本并清理不必要代码，提升了构建环境的性能与安全性，使用户能够利用最新版Go语言特性和修复的安全漏洞。

- **Related PR**: [#2564](https://github.com/alibaba/higress/pull/2564) \
  **Contributor**: @rinfx \
  **Change Log**: 优化了最小请求数逻辑的位置，将其移至streamdone中处理，并改进了Redis Lua脚本中的计数比较逻辑。 \
  **Feature Value**: 提高了系统在异常情况下的稳定性和准确性，确保请求计数和负载均衡策略的正确实施，提升用户体验。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2675](https://github.com/alibaba/higress/pull/2675) \
  **Contributor**: @Aias00 \
  **Change Log**: 修复了项目文档中的一些死链，确保用户能够访问到正确的链接，提高了文档的可用性和准确性。 \
  **Feature Value**: 通过修复文档中的死链，用户可以更容易地找到和使用相关资源，提升了用户体验和文档的整体质量。

- **Related PR**: [#2668](https://github.com/alibaba/higress/pull/2668) \
  **Contributor**: @Aias00 \
  **Change Log**: 改进了Rust插件开发框架的README文档，新增详细的开发指南，包括环境要求、构建步骤和测试方法。 \
  **Feature Value**: 提高了项目的可维护性和易用性，使新开发者能够快速上手，更好地理解和使用Rust Wasm插件开发框架。

- **Related PR**: [#2647](https://github.com/alibaba/higress/pull/2647) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: 此PR增加了New Contributors和full changelog部分，并引入了markdown强制换行，以改善文档的可读性和完整性。 \
  **Feature Value**: 通过增加贡献者名单和完整的变更日志，以及改进Markdown格式，使项目文档更加清晰易读，方便用户了解最新更新及参与者的贡献。

- **Related PR**: [#2635](https://github.com/alibaba/higress/pull/2635) \
  **Contributor**: @github-actions[bot] \
  **Change Log**: 此PR为Higress 2.1.5版本添加了详尽的发布说明，包括新功能、Bug修复和性能优化等内容。 \
  **Feature Value**: 通过提供详细的发布信息，用户可以更好地了解Higress的新特性及改进点，从而更有效地使用该软件。

- **Related PR**: [#2586](https://github.com/alibaba/higress/pull/2586) \
  **Contributor**: @erasernoob \
  **Change Log**: 更新了wasm-go插件的README文件，移除了TinyGo相关配置，并调整了Go版本要求至1.24以上以支持wasm构建特性，同时清理了不再使用的代码路径。 \
  **Feature Value**: 通过更新文档和环境配置要求，确保开发者能够正确设置其开发环境来编译wasm-go插件，这有助于避免因使用不兼容的语言版本或依赖项而导致的问题。

### 🧪 测试改进 (Testing)

- **Related PR**: [#2596](https://github.com/alibaba/higress/pull/2596) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: 本PR通过添加一个新的GitHub Actions工作流文件，实现了在发版时自动生成release notes并提交PR的功能。该流程基于higress-report-agent实现。 \
  **Feature Value**: 此功能极大地简化了发布过程中的文档维护工作，提高了团队的工作效率，并确保每次版本发布都有详细的变更记录供用户参考。

---

## 📊 发布统计

- 🚀 新功能: 13项
- 🐛 Bug修复: 5项
- ♻️ 重构优化: 7项
- 📚 文档更新: 5项
- 🧪 测试改进: 1项

**总计**: 31项更改（包含2项重要更新）

感谢所有贡献者的辛勤付出！🎉


# Higress Console


## 📋 本次发布概览

本次发布包含 **12** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 6项
- **Bug修复**: 5项
- **重构优化**: 1项

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#562](https://github.com/higress-group/higress-console/pull/562) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR实现了在单一路由或AI路由中配置多个路由的功能，修改了后端和前端相关代码，并增强了Kubernetes模型转换器。 \
  **Feature Value**: 支持在一个路由配置中添加多个子路由，为用户提供更灵活的路由管理能力，提升了系统的配置灵活性和用户体验。

- **Related PR**: [#560](https://github.com/higress-group/higress-console/pull/560) \
  **Contributor**: @Erica177 \
  **Change Log**: 此PR为多个插件添加了JSON Schema，包括AI代理、AI缓存等，定义了插件配置的结构和属性，有助于提高配置的规范性和可读性。 \
  **Feature Value**: 通过引入JSON Schema，用户可以更清晰地理解每个插件的配置项及其作用，从而简化配置过程并减少错误配置的风险，提升用户体验。

- **Related PR**: [#555](https://github.com/higress-group/higress-console/pull/555) \
  **Contributor**: @hongzhouzi \
  **Change Log**: 新增了DB MCP Server的执行、列表展示及表描述工具配置功能，确保控制台与higress-gateway中配置的一致性。 \
  **Feature Value**: 用户现在可以通过控制台查看和管理DB MCP Server工具的配置信息，增强了系统的可视化管理和一致性。

- **Related PR**: [#550](https://github.com/higress-group/higress-console/pull/550) \
  **Contributor**: @CH3CHO \
  **Change Log**: 该PR更新了在特定类型的LLM提供者更新后AI路由配置的逻辑，确保上游服务名称变更时路由能够正确同步。 \
  **Feature Value**: 通过自动更新AI路由配置来适应某些LLM提供者类型更改后的服务名称变化，提升了系统的灵活性和稳定性，减少了手动调整的需求。

- **Related PR**: [#547](https://github.com/higress-group/higress-console/pull/547) \
  **Contributor**: @CH3CHO \
  **Change Log**: 在系统配置页面中增加了撤销/重做功能，通过引入forwardRef和useImperativeHandle来支持代码编辑器组件的新API。 \
  **Feature Value**: 新增的撤销/重做功能提升了用户在进行系统配置时的操作灵活性，减少了误操作带来的不便，提高了用户体验。

- **Related PR**: [#543](https://github.com/higress-group/higress-console/pull/543) \
  **Contributor**: @erasernoob \
  **Change Log**: 此PR将插件版本从1.0.0升级到2.0.0，涉及对plugins.properties文件中的相关条目进行更新。 \
  **Feature Value**: 通过升级插件版本，增强了系统的功能性和兼容性，用户可以享受到新版本带来的性能优化和额外特性。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#559](https://github.com/higress-group/higress-console/pull/559) \
  **Contributor**: @KarlManong \
  **Change Log**: 该PR修正了项目中除二进制及cmd文件外所有文件的行尾符，统一为LF格式，避免因换行符不一致导致的问题。 \
  **Feature Value**: 通过统一文件的行尾符为LF，可以提高代码的一致性和兼容性，减少因换行符差异引起的各种问题，特别是在跨平台开发环境中。

- **Related PR**: [#554](https://github.com/higress-group/higress-console/pull/554) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了LLM提供商管理模块中的UI问题，包括Google Vertex服务端点缺少scheme以及取消新增提供商操作后表单状态未重置的问题。 \
  **Feature Value**: 通过修正这些问题，提升了用户在管理和配置LLM提供商时的体验，确保了界面的一致性和功能的准确性。

- **Related PR**: [#549](https://github.com/higress-group/higress-console/pull/549) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR确保在打开配置编辑抽屉时始终加载最新的插件配置，通过修改useEffect中的数据获取逻辑实现。 \
  **Feature Value**: 修复了可能因配置未及时更新而导致的用户操作基于旧配置的问题，提升了用户体验和系统的响应准确性。

- **Related PR**: [#548](https://github.com/higress-group/higress-console/pull/548) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR修正了Wasm镜像URL提交前未去除首尾空白字符的问题，确保了URL的有效性。 \
  **Feature Value**: 通过移除Wasm镜像URL中的多余空格，提高了数据准确性，避免因格式问题导致的加载失败，提升了用户体验。

- **Related PR**: [#544](https://github.com/higress-group/higress-console/pull/544) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了启用认证但未选择消费者时显示的错误消息不正确的问题，通过更新翻译文件和调整代码逻辑来确保正确的错误提示。 \
  **Feature Value**: 此修复提高了系统的可用性和用户体验，确保用户在配置服务时能够接收到准确的反馈信息，避免了因误导性错误消息导致的混淆。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#551](https://github.com/higress-group/higress-console/pull/551) \
  **Contributor**: @JayLi52 \
  **Change Log**: 移除了数据库配置中主机和端口字段的禁用状态，将API网关默认URL从https改为http，并更新了MCP详细页面中的API网关URL显示逻辑。 \
  **Feature Value**: 这些改动增强了系统的灵活性和用户友好性，允许用户自定义更多配置项，并确保UI与后端行为一致，提升了用户体验。

---

## 📊 发布统计

- 🚀 新功能: 6项
- 🐛 Bug修复: 5项
- ♻️ 重构优化: 1项

**总计**: 12项更改

感谢所有贡献者的辛勤付出！🎉


