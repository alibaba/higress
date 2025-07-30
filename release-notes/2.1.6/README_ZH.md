# Higress


## 📋 本次发布概览

本次发布包含 **31** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 14项
- **Bug修复**: 5项
- **重构优化**: 6项
- **文档更新**: 5项
- **测试改进**: 1项

### ⭐ 重点关注

本次发布包含 **2** 项重要更新，建议重点关注：

- **feat: Add Higress API MCP server** ([#2517](https://github.com/alibaba/higress/pull/2517)): 新功能为用户提供了一种管理和配置Higress资源的新方式，增强了系统的灵活性和可扩展性，使得用户可以更方便地进行路由和服务管理。
- **Migrate WASM Go Plugins to New SDK and Go 1.24** ([#2532](https://github.com/alibaba/higress/pull/2532)): 通过迁移至新SDK和Go版本，提高了代码质量和兼容性，减少了潜在的编译错误和运行时问题，提升了系统的稳定性和性能。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat: Add Higress API MCP server

**相关PR**: [#2517](https://github.com/alibaba/higress/pull/2517) | **贡献者**: [cr7258](https://github.com/cr7258)

**使用背景**

在微服务架构中，有效的路由和服务管理是关键。Higress作为一个高性能的API网关，需要一个强大的工具来管理其内部资源。现有的管理方式可能不够灵活或难以扩展。为了解决这个问题，PR #2517引入了一个新的Higress API MCP Server，通过调用Higress Console API来实现对路由、服务来源和插件等资源的集中管理。这不仅提高了系统的可操作性，还增强了用户体验，特别适合那些需要频繁调整和优化API网关配置的运维团队。

**功能详述**

该功能通过golang-filter重新实现了Higress Ops MCP Server，并新增了Higress API MCP Server。主要技术实现包括：
1. 使用Go语言编写客户端`HigressClient`，支持HTTP请求的基本操作（GET, POST, PUT, DELETE）。
2. 实现了多个API工具，如路由管理（list-routes, get-route, add-route, update-route）、服务来源管理（list-service-sources, get-service-source, add-service-source, update-service-source）和插件管理（get-plugin, delete-plugin, update-request-block-plugin）。
3. 提供了详细的配置参数说明，包括Higress Console的URL、用户名、密码等。
4. 代码变更涉及多个文件，包括README文档、配置文件、客户端实现和工具注册等，共计1546行代码变更。

**使用方式**

启用和配置Higress API MCP Server的步骤如下：
1. 在Higress Gateway的配置文件中添加MCP Server的相关配置，例如设置`higressURL`、`username`和`password`等参数。
2. 构建Higress Gateway镜像时，确保包含`golang-filter.so`插件，可以使用`make build-gateway-local`命令。
3. 启动Higress Gateway后，通过配置的路径访问Higress API MCP Server，例如`/higress-api`。
4. 使用提供的API工具进行路由、服务来源和插件的管理。例如，通过`list-routes`获取所有路由信息，通过`add-route`添加新的路由。
注意事项：
- 确保Higress Console的URL、用户名和密码正确无误。
- 在生产环境中，建议使用环境变量或加密存储来保护敏感信息。
- 遵循最佳实践，定期检查和更新配置以保持系统的安全性和稳定性。

**功能价值**

Higress API MCP Server带来了以下显著优势：
1. **提升管理效率**：通过统一的API接口，用户可以轻松地管理和配置Higress的各种资源，减少了手动操作的时间和复杂性。
2. **增强系统灵活性**：支持动态调整路由和服务来源，使得系统能够快速响应业务需求的变化。
3. **提高安全性**：通过严格的参数验证和错误处理机制，确保了系统的稳定性和安全性。
4. **简化运维工作**：提供了详细的日志记录和错误处理，便于运维人员快速定位和解决问题。
5. **促进生态发展**：作为Higress生态系统的一部分，这一功能的引入将进一步推动社区的发展和完善，为用户提供更多便捷的工具和解决方案。

---

### 2. Migrate WASM Go Plugins to New SDK and Go 1.24

**相关PR**: [#2532](https://github.com/alibaba/higress/pull/2532) | **贡献者**: [erasernoob](https://github.com/erasernoob)

**使用背景**

此PR解决了在使用旧版Go SDK和较低版本Go语言时遇到的问题。随着Go 1.24的发布，新版本提供了更好的性能、安全性和稳定性。同时，新的WASM Go SDK也带来了更多的功能和改进。目标用户群体是使用Higress进行WebAssembly插件开发的开发者，他们需要一个更现代、更高效的开发环境。此外，对于维护者来说，统一的依赖管理和构建流程可以减少维护成本。

**功能详述**

具体实现了将所有WASM Go插件从旧的SDK迁移到新的SDK，并将Go版本升级到1.24。主要技术要点包括：
1. 更新Dockerfile和GitHub Actions工作流，以支持新的Go版本和构建参数。
2. 修改go.mod文件，更新依赖项并移除不再需要的依赖。
3. 修复了由于日志包变更导致的日志类型不匹配问题。
4. 优化了构建脚本，移除了不必要的main函数，并确保init()函数符合proxy-wasm-go-sdk规范。
5. 添加了资源清理和错误处理逻辑，提高了代码的健壮性。

**使用方式**

要启用和配置此功能，用户需要更新其项目中的相关文件：
1. 更新Dockerfile，使用新的构建参数（如GOOS=wasip1 GOARCH=wasm）。
2. 更新go.mod文件，确保Go版本为1.24，并移除不再需要的依赖。
3. 更新项目中的日志调用，确保使用新的日志实例。
4. 检查所有回调函数和方法参数，确保它们与新的日志类型和其他参数顺序一致。
典型的使用场景包括：
- 开发新的WASM Go插件
- 升级现有插件以利用新版本的性能和功能
注意事项：
- 确保所有依赖项都已更新，并且没有遗漏的旧依赖哈希。
- 检查所有日志调用和回调函数，确保类型和参数顺序一致。

**功能价值**

此次更新为用户带来了以下好处：
1. **性能提升**：Go 1.24带来了显著的性能改进，特别是在并发和内存管理方面，使插件运行更快、更高效。
2. **稳定性增强**：新的SDK和Go版本提供了更好的错误处理和资源管理，减少了运行时崩溃的风险。
3. **安全性加强**：新版本的Go和SDK修复了多个安全漏洞，提高了插件的安全性。
4. **易用性提升**：统一的依赖管理和构建流程简化了开发和部署过程，降低了维护成本。
5. **生态兼容性**：通过采用最新的SDK和工具链，插件能够更好地与其他现代WebAssembly生态系统组件集成，增强了整个生态系统的互操作性。

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#2679](https://github.com/alibaba/higress/pull/2679)
  **Contributor**: erasernoob
  **Change Log**: 该PR实现了在镜像注解中支持外部FQDN的功能，并添加了相应的测试用例，主要改动位于mirror.go和mirror_test.go文件。
  **Feature Value**: 此功能允许用户通过镜像注解配置外部服务，增强了系统的灵活性和可扩展性，使用户能够更方便地与外部服务集成。

- **Related PR**: [#2667](https://github.com/alibaba/higress/pull/2667)
  **Contributor**: hanxiantao
  **Change Log**: 增加了AI Token限流插件支持为整个路由设置限流阈值的功能，并统一了cluster-key-rate-limit和ai-token-ratelimit插件的基础逻辑。
  **Feature Value**: 用户可以更灵活地控制API请求流量，通过设置全局限流来保护后端服务免受过载。同时，配置一致性的改进减少了潜在的配置错误。

- **Related PR**: [#2652](https://github.com/alibaba/higress/pull/2652)
  **Contributor**: OxalisCu
  **Change Log**: 此PR在ai-proxy插件中添加了对LLM流式请求首字节超时的支持，通过在provider.go文件中引入strconv包并修改ProviderConfig结构体实现。
  **Feature Value**: 新增的首字节超时功能允许用户为LLM流式请求设置一个超时时间，若超过该时间未收到响应，则可采取相应措施，提高了系统的灵活性和可靠性。

- **Related PR**: [#2650](https://github.com/alibaba/higress/pull/2650)
  **Contributor**: zhangjingcn
  **Change Log**: 此PR实现了从Nacos MCP注册表中获取ErrorResponseTemplate配置的功能，通过修改mcp_model.go和watcher.go文件来支持新的功能需求。
  **Feature Value**: 新增了从Nacos MCP注册表获取ErrorResponseTemplate的能力，这增强了系统的灵活性与可配置性，使得用户可以根据需要自定义错误响应模板。

- **Related PR**: [#2649](https://github.com/alibaba/higress/pull/2649)
  **Contributor**: CH3CHO
  **Change Log**: 此PR为Azure OpenAI增加了对三种不同URL配置格式的支持，加强了模型映射功能，并确保`api-version`参数始终存在。
  **Feature Value**: 通过支持多种URL格式，用户可以更灵活地配置Azure OpenAI服务，增强了系统的兼容性和易用性，提升了用户体验。

- **Related PR**: [#2648](https://github.com/alibaba/higress/pull/2648)
  **Contributor**: daixijun
  **Change Log**: 此PR在qwen Provider中添加了对anthropic /v1/messages接口的支持，通过修改ai-proxy相关文件实现了这一新功能。
  **Feature Value**: 新增的接口支持扩展了qwen的功能集，使得用户能够利用Anthropic的服务，增强了系统的灵活性和适用性。

- **Related PR**: [#2639](https://github.com/alibaba/higress/pull/2639)
  **Contributor**: johnlanni
  **Change Log**: 此PR在指定的插件中禁用了重路由功能，通过设置ctx.DisableReroute统一控制，确保了对于不需要重新匹配路由的插件能够避免不必要的处理。
  **Feature Value**: 增强了特定插件的功能灵活性和性能，使得这些插件在修改请求头后不再强制进行路由重匹配，从而提高了处理效率和响应速度。

- **Related PR**: [#2585](https://github.com/alibaba/higress/pull/2585)
  **Contributor**: akolotov
  **Change Log**: 此PR增加了Blockscout MCP服务器的配置文件，包括详细的YAML配置和README文档，以支持用户部署并使用该服务。
  **Feature Value**: 通过为用户提供Blockscout MCP服务器的支持，增强了系统对EVM兼容区块链的数据分析能力，有助于提升用户体验和系统的功能性。

- **Related PR**: [#2551](https://github.com/alibaba/higress/pull/2551)
  **Contributor**: daixijun
  **Change Log**: 此PR添加了对Anthropic和Gemini API的支持，具体包括anthropic/v1/messages、anthropic/v1/complete以及gemini/v1beta/generatecontent等接口。
  **Feature Value**: 通过引入Anthropic和Gemini的新API支持，用户可以利用更多AI服务的能力，增强了系统的功能多样性和灵活性，为用户提供更丰富的应用场景。

- **Related PR**: [#2542](https://github.com/alibaba/higress/pull/2542)
  **Contributor**: daixijun
  **Change Log**: 新增了对images、audio、responses接口Token使用情况的统计功能，并将相关工具函数定义为公共函数，以减少重复代码。
  **Feature Value**: 此更新允许用户更好地监控和管理其API Token使用情况，有助于提高资源利用率和成本控制，特别适用于频繁调用这些接口的服务。

- **Related PR**: [#2537](https://github.com/alibaba/higress/pull/2537)
  **Contributor**: wydream
  **Change Log**: 此PR添加了对Qwen模型的文本重排序功能支持，通过在ai-proxy中增加新的API路径实现。
  **Feature Value**: 增加了对Qwen模型的文本重排序能力，使得用户能够利用该模型进行更精确的数据处理和内容管理。

- **Related PR**: [#2535](https://github.com/alibaba/higress/pull/2535)
  **Contributor**: wydream
  **Change Log**: 此PR引入了`basePath`和`basePathHandling`选项，支持灵活处理请求路径。通过设置`removePrefix`或`prepend`来决定如何使用basePath。
  **Feature Value**: 新增功能允许用户根据需要调整请求路径的处理方式，使得API网关能够更好地与后端服务对接，提升了系统的灵活性和可用性。

- **Related PR**: [#2499](https://github.com/alibaba/higress/pull/2499)
  **Contributor**: heimanba
  **Change Log**: 增加了useManifestAsEntry配置支持，更新了GrayConfig结构体及相关处理逻辑，并修改了文档以反映这些变化。
  **Feature Value**: 通过引入useManifestAsEntry配置，用户可以更灵活地控制首页请求的缓存策略，提升了系统的灵活性和用户体验。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#2687](https://github.com/alibaba/higress/pull/2687)
  **Contributor**: Thomas-Eliot
  **Change Log**: 修复了在使用mcp client工具describeTable时出现的SQL错误，确保从Postgres到MCP Server的数据迁移过程中能够正确描述表结构。
  **Feature Value**: 此修复解决了数据迁移过程中的一个关键问题，提高了系统的稳定性和可靠性，使得用户可以更顺畅地完成数据库操作而不会遇到中断。

- **Related PR**: [#2662](https://github.com/alibaba/higress/pull/2662)
  **Contributor**: johnlanni
  **Change Log**: 修复了Envoy中proxy-wasm的内存泄漏问题以及ppv2启用时端口映射不匹配导致的404错误。
  **Feature Value**: 解决了由于内存泄漏和端口映射错误引起的问题，提高了系统的稳定性和用户体验。

- **Related PR**: [#2656](https://github.com/alibaba/higress/pull/2656)
  **Contributor**: co63oc
  **Change Log**: 此PR修正了多处拼写错误，包括变量名、函数名、接口方法名以及文档中的插件名称，提高了代码的可读性和一致性。
  **Feature Value**: 通过修正拼写错误，确保了程序逻辑的正确性和文档的准确性，提升了用户体验和开发者的维护效率。

- **Related PR**: [#2623](https://github.com/alibaba/higress/pull/2623)
  **Contributor**: Guo-Chenxu
  **Change Log**: 修复了由于特殊字符导致的翻译问题，通过调整生成JSON数据的方式避免了潜在的格式错误。
  **Feature Value**: 此修复确保了在处理包含特殊字符的数据时，系统能够正确地生成和解析JSON，提高了系统的稳定性和可靠性。

- **Related PR**: [#2507](https://github.com/alibaba/higress/pull/2507)
  **Contributor**: hongzhouzi
  **Change Log**: 修正了在arm64架构机器上编译golang-filter.so时出现的错误，确保了正确安装对应的架构工具链。
  **Feature Value**: 解决了arm64用户在编译过程中遇到的问题，提高了跨平台兼容性和用户体验。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#2673](https://github.com/alibaba/higress/pull/2673)
  **Contributor**: johnlanni
  **Change Log**: 改进了`findEndpointUrl`函数，使其能够处理多个SSE消息，而不仅限于第一个。通过增加对其他类型消息的容忍度，增强了函数的健壮性和灵活性。
  **Feature Value**: 提高了系统的兼容性，使得即使在遇到非'endpoint'初始消息的情况下也能正确解析出所需的端点URL，从而提升了用户体验和系统的稳定性。

- **Related PR**: [#2661](https://github.com/alibaba/higress/pull/2661)
  **Contributor**: johnlanni
  **Change Log**: 该PR放宽了DNS服务域名验证的正则表达式，允许更灵活的域名格式。通过修改watcher.go文件中的domainRegex变量定义实现。
  **Feature Value**: 通过放宽域名验证规则，使得系统能够支持更多类型的合法域名，增加了系统的兼容性和灵活性，为用户提供更好的使用体验。

- **Related PR**: [#2615](https://github.com/alibaba/higress/pull/2615)
  **Contributor**: johnlanni
  **Change Log**: 该PR移除了与wasm-go插件相关的Dockerfile、Makefile及多个扩展配置文件中不再使用的EXTRA_TAGS变量，简化了构建流程。
  **Feature Value**: 通过清理不必要的配置项，使得项目结构更加简洁清晰，减少了潜在的维护成本，提升了开发效率。

- **Related PR**: [#2598](https://github.com/alibaba/higress/pull/2598)
  **Contributor**: johnlanni
  **Change Log**: 此PR更新了wasm-go构建器镜像中的Go版本至1.24.4，移除了大量旧代码并简化了Dockerfile。
  **Feature Value**: 通过升级Go版本和精简Dockerfile，提高了WASM插件构建过程的效率与安全性，使维护更加便捷。

- **Related PR**: [#2564](https://github.com/alibaba/higress/pull/2564)
  **Contributor**: rinfx
  **Change Log**: PR优化了最小请求数逻辑的位置及Redis Lua脚本，确保请求计数和配置判断的准确性，提高了系统的稳定性和性能。
  **Feature Value**: 通过改进请求计数逻辑与修复潜在的类型转换错误，增强了负载均衡策略的准确性和可靠性，提升了用户体验。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2675](https://github.com/alibaba/higress/pull/2675)
  **Contributor**: Aias00
  **Change Log**: 该PR修复了项目文档中的几个死链接，确保用户能够访问到正确的资源。
  **Feature Value**: 通过修正文档中的失效链接，提升用户体验，确保他们可以顺利获取所需信息，增强了文档的可靠性和可用性。

- **Related PR**: [#2668](https://github.com/alibaba/higress/pull/2668)
  **Contributor**: Aias00
  **Change Log**: 该PR对Rust插件的README进行了大幅改进，新增了详细的开发指南，包括环境要求、构建步骤和测试方法等。
  **Feature Value**: 通过提供详尽的开发文档，新开发者可以更快地理解和上手Rust Wasm插件的开发，提高了项目的可维护性和易用性。

- **Related PR**: [#2647](https://github.com/alibaba/higress/pull/2647)
  **Contributor**: Guo-Chenxu
  **Change Log**: 该PR增加了New Contributors和full changelog部分，同时改进了markdown格式以支持强制换行。
  **Feature Value**: 通过增加贡献者列表及完整变更日志，提升了文档的可读性和信息丰富度，有助于用户更好地了解项目更新动态。

- **Related PR**: [#2635](https://github.com/alibaba/higress/pull/2635)
  **Contributor**: github-actions[bot]
  **Change Log**: 此PR为Higress 2.1.5版本添加了详细的发布说明，包括新功能、Bug修复和性能优化等更新。
  **Feature Value**: 通过提供详细的发布说明，帮助用户了解最新版本的改进和变化，从而更好地使用和维护系统。

- **Related PR**: [#2586](https://github.com/alibaba/higress/pull/2586)
  **Contributor**: erasernoob
  **Change Log**: 更新了关于wasm-go的README文档，移除了TinyGo相关配置，并将Go版本要求更新为1.24。同时调整了Dockerfile中的环境变量设置。
  **Feature Value**: 通过更新文档和依赖信息，确保开发者能够根据最新的要求构建项目，避免因使用过时或不兼容的工具链而导致的问题，提升了开发体验与效率。

### 🧪 测试改进 (Testing)

- **Related PR**: [#2596](https://github.com/alibaba/higress/pull/2596)
  **Contributor**: Guo-Chenxu
  **Change Log**: 新增了GitHub Actions工作流，在每次发布新版本时自动生成release notes并提交PR。通过设置必要的secrets来支持此功能。
  **Feature Value**: 该功能可以自动化生成和更新项目文档，减少维护者的手动操作，提高工作效率，并确保每次发布的变更日志都能及时准确地传达给用户。

---

## 📊 发布统计

- 🚀 新功能: 14项
- 🐛 Bug修复: 5项
- ♻️ 重构优化: 6项
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

- **Related PR**: [#562](https://github.com/higress-group/higress-console/pull/562)
  **Contributor**: CH3CHO
  **Change Log**: 此PR实现了在一个路由或AI路由中配置多个路由的功能，通过修改后端SDK服务和前端组件来支持这一新特性。
  **Feature Value**: 用户现在可以在单个路由定义中配置多个路由规则，这增强了系统的灵活性和可扩展性，使得路由管理更加便捷高效。

- **Related PR**: [#560](https://github.com/higress-group/higress-console/pull/560)
  **Contributor**: Erica177
  **Change Log**: 此PR为多个插件添加了JSON Schema，包括AI代理、AI缓存、AI数据屏蔽、AI历史记录和AI意图识别等，以增强配置的规范性和易用性。
  **Feature Value**: 通过引入JSON Schema，用户可以更直观地理解和配置插件参数，提高开发效率及减少配置错误，从而提升用户体验。

- **Related PR**: [#555](https://github.com/higress-group/higress-console/pull/555)
  **Contributor**: hongzhouzi
  **Change Log**: 此PR增加了DB MCP Server的执行、列出表和描述表工具的功能，并同步了控制台与higress-gateway中的配置。
  **Feature Value**: 新增功能允许用户通过控制台查看DB MCP Server相关工具的配置，提高了系统的可维护性和用户体验。

- **Related PR**: [#550](https://github.com/higress-group/higress-console/pull/550)
  **Contributor**: CH3CHO
  **Change Log**: 此PR实现了在更新具有特定类型的LLM提供商后更新AI路由配置的功能，确保了服务名称更改后的兼容性和正确性。
  **Feature Value**: 通过自动调整AI路由设置来响应LLM供应商的更新，提高了系统的灵活性和维护效率，减少了手动干预的需求。

- **Related PR**: [#547](https://github.com/higress-group/higress-console/pull/547)
  **Contributor**: CH3CHO
  **Change Log**: 在系统配置页面中实现了撤销/重做功能，通过引入forwardRef和useImperativeHandle来管理CodeEditor组件的状态。
  **Feature Value**: 用户现在可以在系统配置页面上撤销或重做更改，提高了配置编辑的灵活性与用户体验。

- **Related PR**: [#543](https://github.com/higress-group/higress-console/pull/543)
  **Contributor**: erasernoob
  **Change Log**: 该PR将插件版本从1.0.0升级到2.0.0，涉及更新配置文件中的插件地址以指向新版本。
  **Feature Value**: 通过升级插件版本，用户可以利用新特性、性能改进以及可能的bug修复，从而提升整体应用的功能性和稳定性。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#559](https://github.com/higress-group/higress-console/pull/559)
  **Contributor**: KarlManong
  **Change Log**: 该PR修复了项目中文件换行符不一致的问题，确保除了二进制及cmd文件外的所有文件均以LF结尾，避免了因换行符差异导致的潜在问题。
  **Feature Value**: 通过统一文件换行符格式，提高了代码的一致性和可移植性，减少了由于不同操作系统换行符差异引起的各种问题，提升了用户体验和开发效率。

- **Related PR**: [#554](https://github.com/higress-group/higress-console/pull/554)
  **Contributor**: CH3CHO
  **Change Log**: 修复了LLM提供者管理模块中的两个UI问题：添加了Google Vertex服务端点中缺失的方案并确保在取消新提供者操作后重置表单状态。
  **Feature Value**: 解决了用户在配置和管理LLM提供者时遇到的问题，提升了用户体验和系统的可用性。

- **Related PR**: [#549](https://github.com/higress-group/higress-console/pull/549)
  **Contributor**: CH3CHO
  **Change Log**: 此PR修复了在打开配置编辑抽屉时未加载最新插件配置的问题，确保用户能够基于最新的配置信息进行修改。
  **Feature Value**: 解决了用户编辑插件配置时可能遇到的信息不同步问题，提高了用户体验和配置管理的准确性。

- **Related PR**: [#548](https://github.com/higress-group/higress-console/pull/548)
  **Contributor**: CH3CHO
  **Change Log**: 修复了提交前Wasm图像URL中存在前后空格的问题，通过修改index.tsx文件中的相关代码实现。
  **Feature Value**: 解决了因URL前后空格导致的潜在错误或失败问题，提高了系统的稳定性和用户体验。

- **Related PR**: [#544](https://github.com/higress-group/higress-console/pull/544)
  **Contributor**: CH3CHO
  **Change Log**: 修复了启用认证但未选择消费者时显示的错误消息不正确的问题，通过更新翻译文件中的文本和移除冗余代码实现。
  **Feature Value**: 提高了系统的准确性和用户体验，确保用户在特定配置下能够接收到正确的反馈信息，避免误导。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#551](https://github.com/higress-group/higress-console/pull/551)
  **Contributor**: JayLi52
  **Change Log**: 移除数据库配置中主机和端口字段的禁用状态，更改API网关默认URL为http，并更新MCP页面上API网关URL显示逻辑。
  **Feature Value**: 用户现在可以编辑数据库配置中的主机和端口字段，同时通过使用新的默认协议提高API网关URL的一致性和可用性。

---

## 📊 发布统计

- 🚀 新功能: 6项
- 🐛 Bug修复: 5项
- ♻️ 重构优化: 1项

**总计**: 12项更改

感谢所有贡献者的辛勤付出！🎉


