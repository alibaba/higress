# Higress


## 📋 本次发布概览

本次发布包含 **33** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 14项
- **Bug修复**: 5项
- **重构优化**: 8项
- **文档更新**: 5项
- **测试改进**: 1项

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#2679](https://github.com/alibaba/higress/pull/2679)
  **Contributor**: erasernoob
  **Change Log**: 此PR增加了对外部FQDN的支持，通过在mirror注解中添加了新的配置项，并且更新了相关的测试用例来确保新功能的正确性。
  **Feature Value**: 用户现在可以利用外部服务作为镜像目标，这增强了系统的灵活性和可扩展性，允许更广泛的服务集成场景。

- **Related PR**: [#2667](https://github.com/alibaba/higress/pull/2667)
  **Contributor**: hanxiantao
  **Change Log**: 此PR为AI Token限流插件添加了设置全局路由限流阈值的功能，并改进了cluster-key-rate-limit和ai-token-ratelimit插件的基础逻辑与日志提示。
  **Feature Value**: 新功能允许用户更灵活地控制流量，通过设置全局限流阈值来防止过载，提高系统的稳定性和可用性，同时优化了用户体验。

- **Related PR**: [#2652](https://github.com/alibaba/higress/pull/2652)
  **Contributor**: OxalisCu
  **Change Log**: 此PR为AI代理插件添加了对LLM流式请求首字节超时的支持，通过在配置中引入firstByteTimeout参数来控制。
  **Feature Value**: 此功能增强了系统处理长时间无响应的LLM服务的能力，提升了用户体验和系统的稳定性。

- **Related PR**: [#2650](https://github.com/alibaba/higress/pull/2650)
  **Contributor**: zhangjingcn
  **Change Log**: 此PR实现了从Nacos MCP注册中心获取ErrorResponseTemplate配置的功能，通过修改mcp_model.go和watcher.go文件来达成。
  **Feature Value**: 这一功能让使用MCP注册中心的用户能够更方便地获取错误响应模板，从而提高了系统的灵活性与可维护性。

- **Related PR**: [#2649](https://github.com/alibaba/higress/pull/2649)
  **Contributor**: CH3CHO
  **Change Log**: 此PR引入了对Azure OpenAI URL不同格式的支持，包括三种新的URL配置方式，并确保了`api-version`参数始终被要求。
  **Feature Value**: 增强了系统的灵活性和兼容性，使得用户可以更方便地配置与Azure OpenAI服务的连接，从而支持更多场景下的使用。

- **Related PR**: [#2648](https://github.com/alibaba/higress/pull/2648)
  **Contributor**: daixijun
  **Change Log**: 此PR为qwen Provider添加了对anthropic /v1/messages接口的支持，通过在qwen.go文件中引入新的依赖并修改相关代码逻辑实现。
  **Feature Value**: 新增的功能扩展了qwen Provider的能力范围，使得用户能够利用Anthropic API进行消息处理，增强了系统的灵活性和适用性。

- **Related PR**: [#2639](https://github.com/alibaba/higress/pull/2639)
  **Contributor**: johnlanni
  **Change Log**: 此PR通过在特定插件中禁用路由重定向，优化了请求处理流程。主要改动涉及多个WASM插件的相关文件，确保不需要重新匹配的插件不会触发额外的路由逻辑。
  **Feature Value**: 对于使用这些插件的用户而言，该功能可以提高API网关性能，减少不必要的资源消耗，从而加快响应速度并提升整体系统效率。

- **Related PR**: [#2585](https://github.com/alibaba/higress/pull/2585)
  **Contributor**: akolotov
  **Change Log**: 此PR为Blockscout MCP服务器添加了配置文件，包括详细的YAML配置和对应的README文档。
  **Feature Value**: 通过集成Blockscout MCP服务器，用户可以更方便地监控和分析EVM兼容的区块链网络，提升了系统的可观察性和用户体验。

- **Related PR**: [#2551](https://github.com/alibaba/higress/pull/2551)
  **Contributor**: daixijun
  **Change Log**: 本PR添加了对Anthropic和Gemini API的支持，包括anthropic/v1/messages、anthropic/v1/complete以及gemini/v1beta/generatecontent等接口。
  **Feature Value**: 通过支持更多AI服务提供商的API，用户能够更灵活地选择和集成不同的AI功能，从而扩展应用的能力并提升用户体验。

- **Related PR**: [#2542](https://github.com/alibaba/higress/pull/2542)
  **Contributor**: daixijun
  **Change Log**: 此PR添加了对images、audio及responses接口Token使用情况的统计功能，并将UnifySSEChunk与GetTokenUsage定义为公共工具函数，以减少重复代码。
  **Feature Value**: 新功能允许用户更好地监控和管理API Token的使用，特别是对于多媒体文件处理相关的接口，增强了系统的可观察性和成本控制能力。

- **Related PR**: [#2537](https://github.com/alibaba/higress/pull/2537)
  **Contributor**: wydream
  **Change Log**: 此PR为Qwen模型添加了文本重排支持，新增了API接口qwen/v1/rerank，并在provider.go和qwen.go文件中进行了相应的更新。
  **Feature Value**: 通过引入Qwen的文本重排功能，用户现在可以利用这一新特性进行更高效的信息检索与排序，提升应用的数据处理能力。

- **Related PR**: [#2535](https://github.com/alibaba/higress/pull/2535)
  **Contributor**: wydream
  **Change Log**: 此PR引入了`basePath`和`basePathHandling`选项，允许灵活处理请求路径。通过设置`removePrefix`或`prepend`模式来控制基础路径的行为，以适应不同场景下的路由需求。
  **Feature Value**: 新增功能使得API网关能够更好地处理具有前缀的URL，从而简化后端服务对路径的处理逻辑，提升了服务配置的灵活性和用户体验。

- **Related PR**: [#2517](https://github.com/alibaba/higress/pull/2517)
  **Contributor**: cr7258
  **Change Log**: 此PR通过golang-filter重新实现了Higress API MCP服务器，新增了路由管理、服务来源和插件资源管理等功能。
  **Feature Value**: 新功能提供了更灵活的Higress资源配置方式，用户可以更方便地管理路由和服务来源，提升了系统的可维护性和扩展性。

- **Related PR**: [#2499](https://github.com/alibaba/higress/pull/2499)
  **Contributor**: heimanba
  **Change Log**: 新增useManifestAsEntry配置项，更新了GrayConfig结构体，并修改了相关的处理逻辑和文档。主要变更包括在GrayConfig中添加UseManifestAsEntry字段、更新HTML响应处理逻辑以及更新README文档。
  **Feature Value**: 该功能允许用户通过配置useManifestAsEntry来控制首页请求是否缓存，增强了灰度发布时的灵活性，确保在特定场景下首页请求能够按照预期不被缓存，从而保证了灰度策略的有效性。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#2687](https://github.com/alibaba/higress/pull/2687)
  **Contributor**: Thomas-Eliot
  **Change Log**: 该PR修复了在使用mcp client工具describeTable时遇到的SQL错误，通过引入字符串处理库strings来修正问题。
  **Feature Value**: 此修复确保了用户能够正确地获取表结构信息，提升了从Postgres到MCP Server迁移过程中工具的稳定性和可靠性。

- **Related PR**: [#2662](https://github.com/alibaba/higress/pull/2662)
  **Contributor**: johnlanni
  **Change Log**: 修复了proxy-wasm-cpp-host中的内存泄漏问题以及ppv2启用时端口映射不匹配导致的404错误，通过更新相关配置和优化查找逻辑实现。
  **Feature Value**: 解决了因内存泄漏造成的资源浪费及服务中断风险，提升了系统的稳定性和性能；同时保证了在复杂网络环境下的正确路由，增强了用户体验。

- **Related PR**: [#2656](https://github.com/alibaba/higress/pull/2656)
  **Contributor**: co63oc
  **Change Log**: 此PR修正了多个文件中的拼写错误，包括变量名、函数名、接口方法名以及文档中的拼写问题，提高了代码的可读性和一致性。
  **Feature Value**: 通过修复这些拼写错误，提高了代码质量和用户体验，减少了潜在的逻辑错误和运行时异常，确保了系统的稳定性和可靠性。

- **Related PR**: [#2623](https://github.com/alibaba/higress/pull/2623)
  **Contributor**: Guo-Chenxu
  **Change Log**: 修复了在处理特殊字符翻译时可能出现的问题，确保了版本发布说明中不会因特殊字符导致JSON格式错误。
  **Feature Value**: 通过解决特殊字符翻译问题，提高了系统稳定性和用户体验，保证了版本发布说明的准确性和可读性。

- **Related PR**: [#2507](https://github.com/alibaba/higress/pull/2507)
  **Contributor**: hongzhouzi
  **Change Log**: 修正了在arm64架构机器上编译golang-filter.so时出现的错误，确保正确安装对应架构的工具链。
  **Feature Value**: 解决了特定架构下编译问题，保证了跨平台兼容性，提升了用户体验和开发效率。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#2688](https://github.com/alibaba/higress/pull/2688)
  **Contributor**: johnlanni
  **Change Log**: 该PR更新了CI/CD工作流中的OSS上传工具，并将版本号从v2.1.5升级到v2.1.6，同时对Makefile进行了小幅调整。
  **Feature Value**: 通过更新CI/CD流程中的工具和版本号，提高了项目的维护性和一致性，确保用户能够获取到最新的稳定版本。

- **Related PR**: [#2673](https://github.com/alibaba/higress/pull/2673)
  **Contributor**: johnlanni
  **Change Log**: 改进了`findEndpointUrl`函数，使其能够处理多个SSE消息，支持在遇到非'endpoint'事件时继续处理后续消息。
  **Feature Value**: 增强了MCP端点解析器的健壮性和灵活性，确保即使在接收到其他类型的消息后也能正确解析出'endpoint'信息，提升了系统的稳定性和用户体验。

- **Related PR**: [#2661](https://github.com/alibaba/higress/pull/2661)
  **Contributor**: johnlanni
  **Change Log**: 该PR对DNS服务域名验证正则表达式进行了调整，放宽了域名格式的限制条件。
  **Feature Value**: 通过放宽域名验证规则，提高了系统的灵活性和兼容性，使得更多样化的域名能够被接受，从而为用户提供更广泛的使用场景。

- **Related PR**: [#2615](https://github.com/alibaba/higress/pull/2615)
  **Contributor**: johnlanni
  **Change Log**: 该PR移除了wasm-go相关插件构建过程中的一些不再需要的变量和配置，简化了Dockerfile、Makefile及相关扩展文件的内容。
  **Feature Value**: 通过清理冗余代码来保持项目整洁，使得维护更加容易，同时也减少了潜在的错误来源。对于用户来说，这有助于提高系统的稳定性和可维护性。

- **Related PR**: [#2600](https://github.com/alibaba/higress/pull/2600)
  **Contributor**: johnlanni
  **Change Log**: 更新了wasm-go构建镜像中Go的版本至1.24.4，移除了DockerfileBuilder中的部分注释和过时信息。
  **Feature Value**: 通过升级Go版本，提高了构建过程的安全性和稳定性，确保开发者能够使用最新功能和技术改进。

- **Related PR**: [#2598](https://github.com/alibaba/higress/pull/2598)
  **Contributor**: johnlanni
  **Change Log**: 此PR更新了wasm-go构建器镜像中的Go版本至1.24.4，移除了Dockerfile中对特定架构的支持描述。
  **Feature Value**: 通过升级Go版本并简化Dockerfile内容，提升了构建环境的一致性和稳定性，间接促进了使用该构建器开发的WASM插件性能和兼容性。

- **Related PR**: [#2564](https://github.com/alibaba/higress/pull/2564)
  **Contributor**: rinfx
  **Change Log**: 优化了请求数计数逻辑的位置，确保在异常情况下也能正确处理；同时改进了Redis Lua脚本的逻辑，包括修正了字符串比较问题和配置参数判断错误。
  **Feature Value**: 通过将最小请求数-1的逻辑移至streamdone中，并修复Lua脚本中的类型转换和逻辑错误，提升了系统的稳定性和准确性，减少了潜在的运行时错误。

- **Related PR**: [#2532](https://github.com/alibaba/higress/pull/2532)
  **Contributor**: erasernoob
  **Change Log**: 该PR迁移了WASM Go插件到新的SDK和Go 1.24版本，更新了CI/CD配置文件和其他相关代码以确保兼容性和构建正确性。
  **Feature Value**: 通过升级Go版本和SDK，提升了插件的性能和稳定性，为未来的功能开发打下基础，同时减少了潜在的编译和运行时错误。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2675](https://github.com/alibaba/higress/pull/2675)
  **Contributor**: Aias00
  **Change Log**: 此PR修复了四个文档文件中的无效链接，确保用户能够访问到正确的外部资源。
  **Feature Value**: 通过修正死链，提高了文档的准确性和可用性，使得开发者可以更顺畅地获取所需信息，从而提升用户体验。

- **Related PR**: [#2668](https://github.com/alibaba/higress/pull/2668)
  **Contributor**: Aias00
  **Change Log**: PR更新了Rust Wasm插件开发框架的README文件，提供了详细的开发指南，包括环境要求、构建步骤和测试方法。
  **Feature Value**: 此更新极大地提高了项目的可维护性和易用性，使新开发者能够快速上手并理解如何使用Rust进行Wasm插件开发。

- **Related PR**: [#2647](https://github.com/alibaba/higress/pull/2647)
  **Contributor**: Guo-Chenxu
  **Change Log**: 此PR增加了新贡献者列表和完整的变更日志，并在markdown中添加了强制换行功能，以改善文档的组织和可读性。
  **Feature Value**: 通过增加新贡献者列表和完整变更日志，提高了项目的透明度和对社区成员的认可；同时优化了文档格式，使信息展示更加清晰易读。

- **Related PR**: [#2635](https://github.com/alibaba/higress/pull/2635)
  **Contributor**: github-actions[bot]
  **Change Log**: 此PR为2.1.5版本添加了中英文版的发布说明，总结了41项更新内容，涵盖新功能、Bug修复、性能优化等多个方面。
  **Feature Value**: 通过提供详细的发布说明，帮助用户了解版本更新的具体内容及其带来的改进，提升用户体验和系统的透明度。

- **Related PR**: [#2586](https://github.com/alibaba/higress/pull/2586)
  **Contributor**: erasernoob
  **Change Log**: 更新了wasm-go插件相关的README文件，移除了TinyGo相关配置，并将Go版本要求从1.18升级到了1.24以支持wasm构建特性。
  **Feature Value**: 通过更新文档，用户可以获得最新的编译环境需求信息，避免使用过时的工具版本导致编译失败，从而提高开发效率和体验。

### 🧪 测试改进 (Testing)

- **Related PR**: [#2596](https://github.com/alibaba/higress/pull/2596)
  **Contributor**: Guo-Chenxu
  **Change Log**: 该PR引入了一个新的GitHub Action工作流，用于在发布新版本时自动生成并提交release notes。通过设置必要的secrets，确保了自动化流程的安全性和可靠性。
  **Feature Value**: 此功能大大简化了版本发布的流程，减少了手动编写和提交release notes的工作量，提高了团队的效率，并保证了文档的一致性与准确性。

---

## 📊 发布统计

- 🚀 新功能: 14项
- 🐛 Bug修复: 5项
- ♻️ 重构优化: 8项
- 📚 文档更新: 5项
- 🧪 测试改进: 1项

**总计**: 33项更改

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
  **Change Log**: 此PR实现了在一个路由或AI路由中配置多个路由的功能，通过修改后端服务和前端组件来支持这一新特性。
  **Feature Value**: 允许用户在一个路由或AI路由下配置多个路由，简化了复杂场景下的路由管理，提高了灵活性与用户体验。

- **Related PR**: [#560](https://github.com/higress-group/higress-console/pull/560)
  **Contributor**: Erica177
  **Change Log**: 为多个插件添加了JSON Schema，包括AI代理、缓存、数据屏蔽、历史记录和意图识别等，定义了详细的配置属性。
  **Feature Value**: 通过引入JSON Schema，增强了插件的配置管理能力，使得用户可以更直观地理解和设置插件参数，提升了配置的准确性和易用性。

- **Related PR**: [#555](https://github.com/higress-group/higress-console/pull/555)
  **Contributor**: hongzhouzi
  **Change Log**: 新增了DB MCP Server的执行、列表展示表和描述表工具配置功能，确保控制台中显示的配置与higress-gateway中的设置保持一致。
  **Feature Value**: 用户现在能够通过控制台查看并管理DB MCP Server的详细配置信息，提高了系统的可维护性和一致性。

- **Related PR**: [#550](https://github.com/higress-group/higress-console/pull/550)
  **Contributor**: CH3CHO
  **Change Log**: 该PR更新了AI路由配置逻辑，以确保在更新特定类型的LLM提供者时能够正确同步路由配置。
  **Feature Value**: 通过此功能更新，用户可以更准确地管理其AI服务的路由配置，尤其是在变更LLM供应商类型后，保证了服务名称的一致性与可用性。

- **Related PR**: [#547](https://github.com/higress-group/higress-console/pull/547)
  **Contributor**: CH3CHO
  **Change Log**: PR通过在系统配置页面中引入撤销/重做功能，增强了用户对配置更改的控制能力，主要修改了CodeEditor组件和相关页面逻辑。
  **Feature Value**: 新增的撤销/重做功能允许用户轻松回滚或恢复配置更改，提高了用户体验并减少了误操作的风险。

- **Related PR**: [#543](https://github.com/higress-group/higress-console/pull/543)
  **Contributor**: erasernoob
  **Change Log**: 此PR将插件版本从1.0.0升级到了2.0.0，涉及对plugins.properties文件的更新。
  **Feature Value**: 通过升级插件版本至2.0.0，用户可以享受到新版本带来的性能改进和功能增强，提升了系统的整体稳定性和用户体验。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#559](https://github.com/higress-group/higress-console/pull/559)
  **Contributor**: KarlManong
  **Change Log**: 该PR修正了项目中文件的换行符格式，确保所有非二进制及cmd文件使用LF结尾，提高了代码的一致性和跨平台兼容性。
  **Feature Value**: 统一文件换行符为LF有助于避免因不同操作系统引起的文件差异问题，提升了项目的可维护性和用户体验。

- **Related PR**: [#554](https://github.com/higress-group/higress-console/pull/554)
  **Contributor**: CH3CHO
  **Change Log**: 修复了LLM提供商管理模块中的两个UI问题：1. Google Vertex服务端点缺少方案；2. 取消新增提供商操作后确保表单状态重置。
  **Feature Value**: 通过修复这些UI错误，提高了用户在管理LLM提供商时的体验，减少了因界面问题导致的操作失误，增强了系统的可用性和稳定性。

- **Related PR**: [#549](https://github.com/higress-group/higress-console/pull/549)
  **Contributor**: CH3CHO
  **Change Log**: 此PR确保在打开配置编辑抽屉时总是加载最新的插件配置，通过更新特定文件中的几行代码来实现。
  **Feature Value**: 修复了可能显示过时插件配置的问题，保证用户每次查看或编辑时都能看到最新设置，提升了用户体验的一致性和准确性。

- **Related PR**: [#548](https://github.com/higress-group/higress-console/pull/548)
  **Contributor**: CH3CHO
  **Change Log**: 此PR修复了Wasm镜像URL提交前的前后空格问题，通过去除URL中的多余空格来确保其格式正确。
  **Feature Value**: 该修复提高了系统的鲁棒性和用户输入容错率，确保即使用户在输入URL时意外添加了空格也能成功提交。

- **Related PR**: [#544](https://github.com/higress-group/higress-console/pull/544)
  **Contributor**: CH3CHO
  **Change Log**: 修复了启用认证但未选择消费者时显示的错误消息不正确的问题，通过更新翻译文件和调整组件代码来确保错误信息准确无误。
  **Feature Value**: 提高了用户界面的信息准确性，避免因误导性错误消息给用户带来的困惑，提升了用户体验。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#551](https://github.com/higress-group/higress-console/pull/551)
  **Contributor**: JayLi52
  **Change Log**: 移除了数据库配置中主机和端口字段的禁用状态，将API网关默认URL从https改为http，并更新了MCP详细页面中的API网关URL显示逻辑。
  **Feature Value**: 这些改动提高了系统的灵活性，允许用户自定义数据库连接信息，并通过简化URL协议来确保一致性和兼容性。

---

## 📊 发布统计

- 🚀 新功能: 6项
- 🐛 Bug修复: 5项
- ♻️ 重构优化: 1项

**总计**: 12项更改

感谢所有贡献者的辛勤付出！🎉


