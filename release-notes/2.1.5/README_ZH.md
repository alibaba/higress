# Higress


## 📋 本次发布概览

本次发布包含 **41** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 20项
- **Bug修复**: 15项
- **重构优化**: 2项
- **文档更新**: 4项

### ⭐ 重点关注

本次发布包含 **9** 项重要更新，建议重点关注：

- **fix: The mcp to rest capability of the mcp server supports returning status without returning a body from the backend, and instead responds via sse** ([#2445](https://github.com/alibaba/higress/pull/2445)): 提升了MCP服务器在处理特定REST请求时的稳定性与兼容性，避免因后端无返回体导致异常，增强用户体验一致性。
- **feat(mcp/sse): support passthourgh the query parameter in sse server to the rest api server ** ([#2460](https://github.com/alibaba/higress/pull/2460)): 增强了SSE功能的灵活性，使查询参数能够正确传递，提升系统兼容性和用户体验。
- **fix too much logs when nacos is not avaiable** ([#2469](https://github.com/alibaba/higress/pull/2469)): 提升了系统稳定性，避免因日志错误导致的程序崩溃，同时减少了无效日志输出，提高日志可读性和系统性能。
- **feat: support for wanxiang image/video generation in ai-proxy & ai-statistics** ([#2378](https://github.com/alibaba/higress/pull/2378)): 支持异步图像/视频生成，提升AI服务能力；配置优化避免日志统计错误，增强系统兼容性与稳定性。
- **feat: add DB MCP Server execute, list tables, describe table tools** ([#2506](https://github.com/alibaba/higress/pull/2506)): 用户可更便捷地执行SQL语句、列出表名及描述表结构，提升了数据库管理与调试效率。
- **fix(ai-proxy): fix bedrock Sigv4 mismatch** ([#2402](https://github.com/alibaba/higress/pull/2402)): 确保AWS SigV4签名机制正确运行，避免因modelId解码错误导致API调用失败，提升系统稳定性与安全性。
- **feat: add mcp-router plugin** ([#2409](https://github.com/alibaba/higress/pull/2409)): 提供统一的网关聚合能力，使得多个MCP后端服务可以通过单一入口进行访问，简化了客户端配置，提升了服务整合与扩展的灵活性。
- **feat(ai-proxy): add support for OpenAI Fine-Tuning API** ([#2424](https://github.com/alibaba/higress/pull/2424)): 用户现在可以使用OpenAI的微调API功能，从而更灵活地定制模型并在特定任务上提升模型表现。
- **feat: add default route support for wanx image&video synthesis** ([#2431](https://github.com/alibaba/higress/pull/2431)): 增强了路由功能，使用户能更高效地调用Wanx图像和视频合成接口，提升系统灵活性与易用性。

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. fix: The mcp to rest capability of the mcp server supports returning status without returning a body from the backend, and instead responds via sse

**相关PR**: [#2445](https://github.com/alibaba/higress/pull/2445) | **贡献者**: [johnlanni](https://github.com/johnlanni)

**使用背景**

在微服务和云原生架构中，MCP（Mesh Configuration Protocol）服务器负责将配置信息通过REST接口传递给下游服务。在某些情况下，后端可能只需要返回HTTP状态码而无需返回响应体。然而，原有实现要求必须包含响应体，导致资源浪费或与某些下游客户端不兼容。此外，传统的HTTP响应模式在实时性和流式数据传输方面存在不足，限制了MCP服务器在高并发、低延迟场景下的表现。此次修复满足了对REST接口更灵活响应机制的需求，目标用户包括基于MCP进行控制平面集成的开发者和运维人员。

**功能详述**

该PR重构了MCP工具库中的`makeHttpResponse`函数，使HTTP响应可以仅返回状态码而不包含响应体，同时集成了SSE（Server-Sent Events）机制以实现流式响应。这一改进使得MCP服务器能够更灵活地处理客户端请求，特别是在状态反馈或事件通知等场景下。技术上，去除了响应处理中强制携带body的限制，同时在回调函数中移除了`sendDirectly`参数，表明响应逻辑已统一为事件驱动。代码变更涉及多个MCP工具模块，包括对go.mod和go.sum依赖版本的更新，以确保兼容性和稳定性。

**使用方式**

启用该功能无需额外配置，MCP服务器默认支持在无响应体的情况下返回状态码，并通过SSE进行响应。开发者在实现自定义MCP工具时，应确保回调函数遵循新的签名格式，即不再包含`sendDirectly`参数。典型使用场景包括状态检查接口、异步任务通知以及事件流推送等。最佳实践建议对返回的HTTP状态码进行标准化处理，并结合SSE客户端事件监听机制提升响应性能和用户体验。

**功能价值**

本次修复显著提升了MCP服务器在处理REST请求时的灵活性和性能。通过支持无body响应和SSE机制，简化了状态反馈流程，减少了不必要的网络开销，提高了系统吞吐量。同时，该改进增强了MCP服务器与流式客户端之间的兼容性，使其更适用于实时控制、异步通知和事件驱动的架构场景。此外，统一的响应机制降低了开发者实现和维护MCP工具的复杂度，提升了整体生态的可扩展性及稳定性。

---

### 2. feat(mcp/sse): support passthourgh the query parameter in sse server to the rest api server 

**相关PR**: [#2460](https://github.com/alibaba/higress/pull/2460) | **贡献者**: [erasernoob](https://github.com/erasernoob)

**使用背景**

在基于SSE（Server-Sent Events）实现的实时消息推送场景中，前端通常会通过查询参数传递上下文信息，例如用户标识或会话状态。然而，原始设计中这些参数无法有效传递到后端的REST API服务器，导致上下文信息丢失，影响后端处理逻辑。该功能的引入解决了这一问题，使得SSE连接中的查询参数能够被正确转发到消息处理接口，从而实现更完整的请求上下文传递。目标用户包括使用Higress进行SSE代理的开发者和需要通过查询参数携带会话状态的前后端服务。

**功能详述**

该PR主要实现了SSE连接建立时的查询参数透传功能。具体来说，在`filter.go`中，从原始请求URL中提取查询参数，并将其附加到构造的messageEndpoint地址上；在`sse.go`中，使用`net/url`包以更安全的方式拼接URL和参数，避免手动拼接带来的格式错误。相较于之前的硬编码方式，这一改进增强了对参数的支持和处理可靠性。关键技术点包括：使用`url.Parse`和`Query()`方法解析和拼接URL参数，以及在构造messageEndpoint时保留原始查询参数。这一功能增强了SSE代理的能力，与原有逻辑无缝集成，提升了系统的可扩展性。

**使用方式**

该功能默认启用，无需额外配置。在使用SSE代理时，只需在前端发起SSE连接时在URL中附带查询参数（如`/sse?userId=123`），这些参数将自动透传至对应的REST API消息处理接口。典型使用场景包括：基于查询参数进行用户身份识别、会话控制或动态路由。建议在使用时确保参数的合法性与安全性，避免注入攻击。最佳实践是结合身份认证机制，在后端校验关键参数的有效性。

**功能价值**

该功能显著提升了SSE代理的灵活性和实用性，使得后端服务能够基于完整的请求上下文进行处理。它增强了SSE连接的可用性，支持更丰富的业务场景，例如个性化消息推送和多租户会话管理。在系统层面上，提高了代理服务的通用性和兼容性，降低了前端与网关之间的耦合度。生态层面，这一改进使得Higress在支持实时通信场景（如聊天、通知、数据看板等）方面更具竞争力，为构建更丰富的微服务集成方案提供了坚实基础。

---

### 3. fix too much logs when nacos is not avaiable

**相关PR**: [#2469](https://github.com/alibaba/higress/pull/2469) | **贡献者**: [luoxiner](https://github.com/luoxiner)

**使用背景**

在MCP服务发现模块中，当注册中心Nacos处于不可用状态时，系统会尝试不断拉取配置信息以维持服务发现能力。此时，如果Nacos无法正常响应，会触发日志记录逻辑。由于原有代码中存在日志记录参数缺失的问题，不仅导致日志中关键错误信息缺失，还可能引发运行时panic。此外，日志频繁输出也增加了系统I/O压力，影响整体可观测性和性能。目标用户主要是使用MCP架构并集成Nacos作为服务发现机制的系统运维人员和开发人员。

**功能详述**

此PR主要修复了两个问题：一是client.go中日志记录调用格式字符串与参数数量不匹配，第144行缺少err参数，第149行多了一个%v但没有传入参数，容易导致运行时panic；二是优化日志输出逻辑，在出现错误或空结果时不再持续重试，而是选择退出循环，避免日志风暴。同时，watcher.go中增加了日志滚动配置的默认参数，包括单个日志文件最大大小（64MB）和最大备份数量（3），提升日志管理的可控性。该修复通过修改日志调用方式并调整日志级别控制策略实现，与现有功能兼容且无需额外配置。

**使用方式**

本PR的修复是自动生效的，无需手动启用或配置。典型使用场景包括MCP服务启动、Nacos服务异常切换或网络中断等情况下的服务发现流程。用户只需正常部署MCP服务并与Nacos集成即可。在Nacos不可用时，系统将显著减少不必要的日志输出，并避免因日志格式错误导致程序崩溃。最佳实践是结合日志监控系统观察日志量变化，确保系统具备足够的日志容错能力。注意：如果自定义了日志路径或日志级别，需确保logrolling配置的兼容性。

**功能价值**

此项修复提升了系统的健壮性和日志处理的可靠性，有效避免了Nacos异常时的日志风暴和panic问题，从而增强了服务发现机制的稳定性。通过引入日志滚动配置参数，也增强了对日志存储空间和生命周期的控制能力，降低了运维成本。此外，该修复有助于提升MCP架构在复杂网络环境中的可靠性，进一步增强了服务注册与发现模块的健壮性，对整个生态系统的稳定性具有积极意义。

---

### 4. feat: support for wanxiang image/video generation in ai-proxy & ai-statistics

**相关PR**: [#2378](https://github.com/alibaba/higress/pull/2378) | **贡献者**: [mirror58229](https://github.com/mirror58229)

**使用背景**

此PR解决了在使用万相（WanXiang）AIGC服务时，对异步生成任务的代理和日志统计支持不完善的问题。万相提供的文本生成图像或视频的功能具有较长的处理时延，因此需要异步提交和查询任务状态的能力。此外，万相的API协议与OpenAI标准不同，导致日志中model、token等字段提取失败，影响监控与统计功能。目标用户主要为使用阿里云万相服务的AI平台管理员和开发者，他们需要稳定、可扩展的AIGC调用与可观测能力。

**功能详述**

PR在ai-proxy中新增了对万相异步AIGC接口的识别与路由支持，包括两个API路径：`/api/v1/services/aigc`用于任务提交，`/api/v1/tasks`用于任务状态查询。同时，在ai-statistics模块中新增`disable_openai_usage`配置项，用于关闭OpenAI兼容格式的日志字段提取逻辑，避免因万相API非标准响应结构导致的错误。代码中通过新增路由匹配规则和配置逻辑实现，确保现有OpenAI兼容服务的配置不会受到影响，同时提升万相服务的兼容性和可观测性。

**使用方式**

在配置ai-proxy时，将万相服务的API路径映射为`/api/v1/services/aigc`（生成请求）和`/api/v1/tasks`（状态查询）。在ai-statistics插件配置中，若使用非OpenAI协议（如万相），设置`disable_openai_usage: true`以避免日志解析错误。典型使用流程包括：提交文本生成图像请求、异步轮询任务状态、记录任务完成情况。最佳实践包括合理配置日志字段、监控任务延迟，并确保异步路径配置准确无误，避免路径匹配错误影响服务。

**功能价值**

该功能提升了平台对异步AIGC生成任务的支持能力，增强了对非标准协议服务（如万相）的兼容性，避免日志统计错误并提升可观测性。同时，通过结构化配置支持，简化了平台管理与监控，提升了服务的稳定性与可维护性。对于AI平台生态而言，新增的万相接口支持有助于拓展AIGC应用场景，提升平台对图像、视频生成类服务的统一接入能力和可观测性水平。

---

### 5. feat: add DB MCP Server execute, list tables, describe table tools

**相关PR**: [#2506](https://github.com/alibaba/higress/pull/2506) | **贡献者**: [hongzhouzi](https://github.com/hongzhouzi)

**使用背景**

随着MCP（Model Control Protocol）服务在数据库连接与交互场景中的广泛应用，用户对于数据库的即时控制能力提出了更高要求。原有实现仅支持只读SQL查询，缺乏执行变更语句、获取表结构信息的能力，导致在复杂场景下功能受限。例如，开发者在调试或部署阶段需要查看数据库表结构，运营人员需要批量执行变更SQL，这些场景均无法通过现有功能支持。目标用户主要包括数据库开发者、AI应用集成人员以及系统运维人员，他们需要通过MCP服务实现更灵活、更全面的数据库操作。

**功能详述**

本PR新增了三个核心工具：execute（执行SQL语句，如INSERT/UPDATE/DELETE）、list tables（列出所有表名）、describe table（获取指定表的字段结构）。技术实现上，在db.go中引入了数据库类型常量（如MYSQL、POSTGRES等）以提高可维护性；在tools.go中分别实现了四个工具的处理函数，如HandleExecuteTool用于处理写操作，HandleListTablesTool和HandleDescribeTableTool则分别调用底层GORM能力获取表信息和结构。server.go中通过AddTool方法将新增工具注册至MCP Server，并统一描述信息格式，提升一致性。社区反馈指出存在SQL注入风险、重复错误处理等问题，建议后续优化。

**使用方式**

要启用这些功能，需在MCP Server配置文件中正确设置数据库连接信息（DSN）与类型（如mysql、postgres）。用户可通过MCP客户端调用新增工具：
- execute：传入SQL语句参数，执行INSERT/UPDATE/DELETE操作；
- list tables：无需参数，直接调用即可返回所有表名；
- describe table：传入表名参数，获取该表的字段、类型、约束等信息。
典型使用场景包括：在部署脚本中自动执行初始化SQL、通过UI界面查看表结构、在调试阶段检查数据库状态等。使用时应注意权限控制，避免非授权访问执行SQL；同时建议参数化查询以防止SQL注入问题。

**功能价值**

本次功能增强显著提升了MCP Server对数据库的交互能力，使其能够满足更复杂的应用需求。execute工具支持写操作，弥补了原有只读查询的局限；list tables和describe table工具则增强了对数据库结构的感知能力，为自动化运维和可视化界面提供了基础支持。从系统角度看，这些工具提高了数据库调试和管理的便捷性，减少了手动干预，提升了整体稳定性。此外，通过对数据库类型进行常量定义，增强了代码可读性与可维护性。虽然仍存在SQL注入和重复错误处理等问题，但已为后续优化打下基础，是MCP Server生态完善的重要一步。

---

### 6. fix(ai-proxy): fix bedrock Sigv4 mismatch

**相关PR**: [#2402](https://github.com/alibaba/higress/pull/2402) | **贡献者**: [HecarimV](https://github.com/HecarimV)

**使用背景**

AWS Bedrock 是 Amazon 提供的全托管基础模型服务，允许用户通过统一接口访问多种模型。在使用该服务时，请求必须通过 Sigv4 签名机制进行身份验证。然而，之前的实现中对 URL 路径的编码不符合 AWS IAM 文档中有关 Sigv4 的编码规范，导致签名验证失败，表现为 403 Forbidden 或其他认证错误。此问题影响了使用 AI Proxy 代理 AWS Bedrock 服务的用户，特别是在需要路径参数的场景下。目标用户主要为使用 AI Proxy 作为 AWS Bedrock 服务前端、希望统一 API 代理层并实现协议兼容的开发者和系统架构师。

**功能详述**

此 PR 主要修复了 Sigv4 签名中对路径部分（Canonical URI）的编码问题。根据 AWS IAM 文档中的规范，路径部分应保留斜杠 `/` 并对其他特殊字符进行 RFC 3986 兼容的编码，且编码字符使用大写形式。代码中新增了 `encodeSigV4Path` 函数，按段对路径进行处理，并使用 `url.PathEscape` 实现正确编码。此外，修复了 `modelId` 的解码逻辑，避免潜在的数据污染风险。该改动确保了签名的正确性，解决了 #2396 提出的身份验证不匹配问题。与原有实现相比，增强了与 AWS 服务的兼容性，减少了因签名错误导致的无效请求。

**使用方式**

要使用此功能，需在 AI Proxy 的配置中设置类型为 `bedrock`，并提供 `awsAccessKey`、`awsSecretKey` 和 `awsRegion` 参数。例如配置如下：

```yaml
provider:
  type: bedrock
  awsAccessKey: "YOUR_AWS_ACCESS_KEY_ID"
  awsSecretKey: "YOUR_AWS_SECRET_ACCESS_KEY"
  awsRegion: "YOUR_AWS_REGION"
```

典型使用场景包括通过 OpenAI 协议代理 AWS Bedrock 服务，实现统一的 API 接口接入。请求示例中，可使用标准的 OpenAI 格式调用 Bedrock 模型。使用时应确保 AWS 凭证具备访问目标模型的权限，并正确配置区域信息。最佳实践包括定期更换密钥、使用 IAM 角色管理访问权限，并结合 VPC 等安全机制增强安全性。

**功能价值**

此次修复显著提升了 AI Proxy 对 AWS Bedrock 服务的集成稳定性，解决了常见的 Sigv4 签名不匹配问题，使用户能够更可靠地使用统一代理层访问 AWS 模型服务。同时增强了系统的安全性和兼容性，降低了因签名错误导致的服务不可用风险。对于需要多模型后端统一管理、协议兼容的场景，该改进具有重要价值，提升了 AI Proxy 在企业级 AI 服务网关场景中的实用性与可靠性。

---

### 7. feat: add mcp-router plugin

**相关PR**: [#2409](https://github.com/alibaba/higress/pull/2409) | **贡献者**: [johnlanni](https://github.com/johnlanni)

**使用背景**

在当前的MCP架构中，一个MCP Server通常只对应一个后端服务实例，这种一一对应的关系限制了创建统一的MCP端点的能力。对于需要集成多个后端工具的AI代理来说，这种限制导致客户端需要管理多个MCP端点，增加了复杂性和维护成本。因此，需要一个动态路由机制，使得一个MCP网关能够根据请求内容将工具调用路由到不同的后端服务。mcp-router插件正是为解决这一问题而设计的，它允许客户端通过单一入口调用不同服务上的MCP工具。

**功能详述**

mcp-router插件通过解析tools/call请求中的工具名称，判断是否需要路由到特定的后端MCP服务器。若工具名称带有前缀（如server-name/tool-name），则插件会根据配置的路由规则将请求重新定向到对应的后端服务器。插件使用Wasm-Go编写，并通过修改请求头和请求体来实现动态路由。技术实现上，它利用了JSON-RPC协议中对方法名称的解析以及网关路由引擎的二次处理机制，动态修改请求的目标域名和路径，从而实现多后端服务的无缝集成。

**使用方式**

要使用mcp-router插件，需在Higress的路由配置中启用该插件并配置服务器路由规则。配置时需提供每个后端MCP服务器的name、domain和path信息。例如：在higress-plugins.yaml中定义服务器列表，包括服务器名称、域名和路径。客户端在发起tools/call请求时，只需在工具名称中添加服务器前缀（如`server-name/tool-name`），插件会自动将请求路由到指定的后端服务器。最佳实践是确保name字段与服务器配置一致，并合理配置domain和path以避免路由错误。

**功能价值**

mcp-router插件为用户提供了一个统一的MCP工具调用入口，简化了客户端对多个后端服务的管理。通过动态路由，用户无需维护多个端点，提升了系统的可扩展性和易用性。此外，该插件增强了Higress作为MCP网关的能力，使其能够更灵活地支持工具组合和复杂的服务架构，为AI代理提供了更强大的后端支持。插件的设计也具备良好的可维护性，便于未来扩展更多路由策略和增强配置管理能力。

---

### 8. feat(ai-proxy): add support for OpenAI Fine-Tuning API

**相关PR**: [#2424](https://github.com/alibaba/higress/pull/2424) | **贡献者**: [wydream](https://github.com/wydream)

**使用背景**

OpenAI的Fine-Tuning API允许用户根据特定数据集对基础模型进行微调，以生成具备垂直领域能力的定制模型。随着用户对大模型定制化需求的增长，AI代理平台需要提供对微调任务端到端的支持，包括任务创建、状态监控、事件日志获取、检查点管理等功能。该功能主要面向需要进行模型优化、定制的AI工程师和数据科学家，帮助他们在一个统一的代理层中高效管理微调流程，而无需直接对接底层API。目标用户包括企业级AI应用开发者、模型训练团队及MLOps运维人员。

**功能详述**

该PR在ai-proxy中新增了对OpenAI Fine-Tuning API的完整路由映射和默认能力配置，具体实现包括：在main.go中添加了对多个Fine-Tuning相关路径的路由识别逻辑，如创建任务、获取事件、取消任务、暂停任务、检查点权限管理等；在provider/openai.go中更新了默认能力映射表，将新增的API名称常量对应到实际路径；在provider/provider.go中定义了多个新的ApiName常量和路径常量，确保语义一致性；在util/http.go中引入了多个正则表达式用于路径匹配和参数提取，如提取微调任务ID、检查点ID、权限ID等。代码变更通过结构化映射和正则匹配机制实现了对复杂路径结构的精确识别。尽管社区反馈指出部分常量命名存在拼写或语义不一致的问题，但整体实现已满足基本功能需求。

**使用方式**

该功能默认集成在OpenAI代理模块中，无需额外启用。用户可通过代理层访问以下Fine-Tuning API路径：创建微调任务（POST /v1/fine_tuning/jobs）、列出任务（GET /v1/fine_tuning/jobs）、获取任务详情（GET /v1/fine_tuning/jobs/{job_id}）、获取事件日志（GET /v1/fine_tuning/jobs/{job_id}/events）、获取检查点（GET /v1/fine_tuning/jobs/{job_id}/checkpoints）、取消任务（POST /v1/fine_tuning/jobs/{job_id}/cancel）等。典型使用场景包括：企业内部模型微调服务管理、训练日志监控、检查点权限配置、任务暂停与恢复等。使用时应确保任务ID正确传入，注意路径匹配正则对格式的严格要求，并在使用权限管理接口时注意访问控制策略。

**功能价值**

此功能增强了AI代理平台的模型训练支持能力，使用户能够在统一的代理层中完成微调任务的全生命周期管理。通过集成Fine-Tuning API，用户无需直接对接底层OpenAI服务，即可实现任务自动化编排、状态监控与日志分析，提升开发和运维效率。同时，支持检查点权限管理与任务控制操作（如暂停、恢复、取消），增强了对敏感模型训练流程的安全控制。对于平台生态而言，这一功能扩展了AI代理在模型定制化场景中的应用边界，有助于构建更完整的AI工程化工具链，提升平台的竞争力和用户粘性。

---

### 9. feat: add default route support for wanx image&video synthesis

**相关PR**: [#2431](https://github.com/alibaba/higress/pull/2431) | **贡献者**: [mirror58229](https://github.com/mirror58229)

**使用背景**

AI平台在不断发展过程中，逐渐支持多种生成式AI能力，包括文本生成、图像生成、语音合成，以及视频生成等。model-mapper和model-router是AI代理网关中的关键插件，用于根据请求中的模型参数进行路由决策。在本PR之前，这两个插件并未支持图像合成（image-synthesis）和视频合成（video-synthesis）接口，导致在处理这类请求时缺乏路由映射能力，无法正确引导请求到后端服务。因此，此功能的引入解决了对新AI生成能力支持不足的问题，目标用户主要为AI服务网关的运维人员和需要集成图像/视频生成功能的开发人员。

**功能详述**

该PR主要扩展了model-mapper和model-router插件对WanX图像合成和视频合成接口的支持。具体而言，在两个插件的配置项enableOnPathSuffix的默认值中，新增了路径后缀/image-synthesis和/video-synthesis，以确保针对这两个接口的请求可以被正确识别并进行模型参数解析和路由决策。这一实现通过修改插件源码中的默认配置数组和文档中的配置说明完成，确保了功能的一致性和文档实时更新。与现有功能相比，此次扩展并未引入新的配置字段，而是基于已有的enableOnPathSuffix机制进行扩展，维持了配置逻辑的一致性和可维护性。

**使用方式**

启用该功能无需额外配置变更，系统默认会将路径为/image-synthesis和/video-synthesis的请求纳入模型映射和路由流程。用户可继续使用现有的modelMapping配置规则，为图像和视频生成接口定义特定的模型映射策略。典型使用场景包括：当AI网关需要代理WanX等平台的图像/视频生成API时，通过model-mapper自动识别请求模型参数并完成映射，再由model-router将请求路由至对应的后端服务。最佳实践建议用户定期检查并根据实际需求自定义enableOnPathSuffix配置，以确保接口的安全性和灵活性。

**功能价值**

该功能显著增强了AI网关对多媒体生成能力的支持，使平台在对接图像合成、视频合成等新型AI服务时具备更高的兼容性和扩展性。通过自动化的模型参数识别和路由分发，降低了集成新模型所需的配置复杂度，提升了整体开发和运维效率。同时，该功能进一步完善了插件的适用场景，使其能够覆盖AI代理网关中更广泛的服务类型。此外，通过更新文档和默认配置，也提升了系统的易用性和可维护性，为后续的AI功能扩展提供了良好基础。

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#2536](https://github.com/alibaba/higress/pull/2536)
  **Contributor**: johnlanni
  **Change Log**: 本次PR主要完成了版本号从2.1.5-rc.1升级到2.1.5，涉及Makefile、VERSION和Helm Chart相关文件的更新，标志2.1.5版本正式发布。
  **Feature Value**: 该版本发布提供了最新的稳定版本供用户部署和使用，保证用户能够获取最新的功能增强和问题修复，提升产品体验和稳定性。

- **Related PR**: [#2533](https://github.com/alibaba/higress/pull/2533)
  **Contributor**: johnlanni
  **Change Log**: 新增 ai-proxy 对 subPath 字段的配置支持，提升路径处理灵活性，并同步更新中英文文档描述，增强功能可用性。
  **Feature Value**: 用户可通过配置 subPath 前缀优化请求路径处理逻辑，提升 AI 代理插件对复杂路由场景的适配能力。

- **Related PR**: [#2531](https://github.com/alibaba/higress/pull/2531)
  **Contributor**: rinfx
  **Change Log**: 添加了三个针对LLM服务的负载均衡策略：最小负载、基于Redis的全局最少请求、prompt前缀匹配策略，通过WASM插件实现。
  **Feature Value**: 为LLM服务提供更智能的负载均衡选项，提升系统资源利用率和响应效率，同时支持KV Cache复用，优化推理性能。

- **Related PR**: [#2516](https://github.com/alibaba/higress/pull/2516)
  **Contributor**: HecarimV
  **Change Log**: PR在AI Proxy组件中为Bedrock API请求添加了系统消息处理功能，包括在请求负载结构中新增System字段，并更新请求构建逻辑以支持系统消息的条件包含。
  **Feature Value**: 该功能使用户可通过Bedrock API发送系统消息，从而更灵活地控制对话上下文与模型行为，提升AI交互的准确性和实用性。

- **Related PR**: [#2509](https://github.com/alibaba/higress/pull/2509)
  **Contributor**: daixijun
  **Change Log**: 添加了对 OpenAI responses 接口 Body 的处理，并新增火山方舟大模型 responses 接口支持。
  **Feature Value**: 提升 AI 代理功能，支持 Doubao 模型响应处理，增强系统扩展性和模型适配能力。

- **Related PR**: [#2488](https://github.com/alibaba/higress/pull/2488)
  **Contributor**: rinfx
  **Change Log**: 新增 `trace_span_key` 和 `as_seperate_log_field` 配置项，分别用于区分日志与Span属性的键名，以及控制日志字段是否单独记录。
  **Feature Value**: 提升日志与追踪配置灵活性，使用户能更清晰地管理和查询日志及分布式追踪数据，增强系统可观测性。

- **Related PR**: [#2485](https://github.com/alibaba/higress/pull/2485)
  **Contributor**: johnlanni
  **Change Log**: 该PR为mcp server插件增加了errorResponseTemplate功能，用于在后端HTTP状态码大于300时自定义错误响应模板。
  **Feature Value**: 通过支持自定义错误响应模板，提升了用户对错误处理的灵活性和体验，增强了系统的可配置性与适应性。

- **Related PR**: [#2450](https://github.com/alibaba/higress/pull/2450)
  **Contributor**: kenneth-bro
  **Change Log**: 新增了Investoday MCP Server板块行情功能，包含行业和概念板块的实时行情及成分股数据，覆盖关键市场指标。
  **Feature Value**: 为智能投研和市场热点追踪提供实时、全面的行业与概念板块数据支持，提升用户对市场动态的洞察力。

- **Related PR**: [#2446](https://github.com/alibaba/higress/pull/2446)
  **Contributor**: johnlanni
  **Change Log**: 更新了版本号至v2.1.5-rc.1，同时修改了Helm Chart中的应用版本信息，开始支持新的发布版本。
  **Feature Value**: 为用户提供新版本的试用支持，帮助其及时获取最新的功能与改进，提升使用体验。

- **Related PR**: [#2404](https://github.com/alibaba/higress/pull/2404)
  **Contributor**: 007gzs
  **Change Log**: 新增支持reasoning_content字段和返回多条index分组，提升AI数据掩码功能在流模式下的灵活性和兼容性。
  **Feature Value**: 用户可更高效处理多组响应数据，并兼容OpenAI的n参数特性，提升系统扩展性与使用便捷性。

- **Related PR**: [#2391](https://github.com/alibaba/higress/pull/2391)
  **Contributor**: daixijun
  **Change Log**: 调整AI代理的流式响应结构，使在usage、logprobs、finish_reason字段为空时输出null，保持与OpenAI官方接口一致。
  **Feature Value**: 提升系统兼容性与一致性，使用户在使用不同AI模型时获得统一的响应格式，减少后续处理逻辑的复杂性。

- **Related PR**: [#2389](https://github.com/alibaba/higress/pull/2389)
  **Contributor**: NorthernBob
  **Change Log**: 新增了插件服务器对 Kubernetes 一键部署的支持，并配置了插件的默认下载 URL，涉及 Helm Chart 模板和服务配置的新增与调整。
  **Feature Value**: 简化了 Higress 插件服务器在 Kubernetes 环境下的部署流程，提升了用户部署效率和使用体验。

- **Related PR**: [#2343](https://github.com/alibaba/higress/pull/2343)
  **Contributor**: hourmoneys
  **Change Log**: 新增基于AI的投标信息工具MCP服务，包含英文和中文的说明文档、配置文件，提供根据关键字查询标讯列表的功能。
  **Feature Value**: 帮助企业快速获取精准的标讯信息，提高投标效率与中标概率，并优化标讯查询的使用体验。

- **Related PR**: [#1925](https://github.com/alibaba/higress/pull/1925)
  **Contributor**: kai2321
  **Change Log**: 实现了AI-IMAGE-READER插件，对接OCR服务，支持阿里云模型服务dashscope的qwen-vl，新增中文和英文文档说明以及相关配置。
  **Feature Value**: 用户可通过该插件调用OCR服务提取图像中的文字内容，提升了AI网关对图像内容处理的能力，增强了平台的功能扩展性。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#2524](https://github.com/alibaba/higress/pull/2524)
  **Contributor**: daixijun
  **Change Log**: 修复了`stream_options`参数在非OpenAI流式接口中导致报错的问题，限制其仅在openai/v1/chatcompletions接口生效。
  **Feature Value**: 防止了因错误添加`stream_options`参数导致的接口报错，提升了系统稳定性与参数处理的准确性。

- **Related PR**: [#2514](https://github.com/alibaba/higress/pull/2514)
  **Contributor**: daixijun
  **Change Log**: 注释掉 values.yaml 中 tracing.skywalking 的默认值，避免在配置其他 tracing 类型时 helm upgrade 自动添加 skywalking 配置导致报错。
  **Feature Value**: 修复了在使用 helm 升级时因自动注入无效 tracing 配置导致的错误，提升了配置灵活性和用户体验。

- **Related PR**: [#2497](https://github.com/alibaba/higress/pull/2497)
  **Contributor**: johnlanni
  **Change Log**: 修复了在构造和发送请求时，当配置的URL路径包含URL编码部分时解码行为不正确的问题。
  **Feature Value**: 解决了因URL路径解码错误导致的请求异常问题，提升了系统的稳定性和兼容性，确保用户请求正确处理。

- **Related PR**: [#2480](https://github.com/alibaba/higress/pull/2480)
  **Contributor**: HecarimV
  **Change Log**: 修复了AWS Bedrock请求构建时未初始化AdditionalModelRequestFields导致的空指针异常问题，并完善了相关文档的表格格式。
  **Feature Value**: 提升了AI代理调用AWS Bedrock服务的稳定性，确保用户请求能正确携带额外参数，避免运行时崩溃。

- **Related PR**: [#2475](https://github.com/alibaba/higress/pull/2475)
  **Contributor**: daixijun
  **Change Log**: 修复了 openai provider 在配置 openaiCustomUrl 为单个接口时，customPath 传递错误导致 404 的问题。
  **Feature Value**: 提升了 openai provider 兼容不同接口路径（如千问的 /compatible-mode/v1）的适配能力，确保请求正常处理。

- **Related PR**: [#2443](https://github.com/alibaba/higress/pull/2443)
  **Contributor**: Colstuwjx
  **Change Log**: 修复了controller service account缺少annotations配置的问题，允许用户通过annotations绑定AWS IAM角色。
  **Feature Value**: 增强了controller SA的灵活性，使用户能够通过AWS IAM进行资源身份验证，提高了系统集成能力。

- **Related PR**: [#2441](https://github.com/alibaba/higress/pull/2441)
  **Contributor**: wydream
  **Change Log**: 统一API名称常量的命名规范，修复ApiName映射问题，修正了多个拼写错误，确保API路径匹配正确。
  **Feature Value**: 提升API调用的准确性和稳定性，避免因路径拼写错误导致的404错误或功能失效，改善用户体验。

- **Related PR**: [#2440](https://github.com/alibaba/higress/pull/2440)
  **Contributor**: johnlanni
  **Change Log**: 修复了Istio中启用一致性哈希时rds缓存不生效的问题，并修复了envoy wasm的abi获取接口。
  **Feature Value**: 提升Istio在一致性哈希场景下的稳定性和功能可用性，增强envoy wasm的功能兼容性。

- **Related PR**: [#2423](https://github.com/alibaba/higress/pull/2423)
  **Contributor**: johnlanni
  **Change Log**: 修复了在配置SSE转发的MCP服务器时可能导致控制器崩溃的问题，优化了相关代码健壮性。
  **Feature Value**: 解决了潜在的控制器崩溃问题，提升了系统稳定性，保障了用户服务的持续可用性。

- **Related PR**: [#2408](https://github.com/alibaba/higress/pull/2408)
  **Contributor**: daixijun
  **Change Log**: 修复Gemini提供者中finishReason缺失的问题，将STOP转换为小写并与OpenAI API保持一致，同时修复流式响应中finishReason内容缺失。
  **Feature Value**: 提升AI代理的兼容性和稳定性，确保用户在使用Gemini API时能正确获取finishReason信息，避免潜在错误和体验问题。

- **Related PR**: [#2405](https://github.com/alibaba/higress/pull/2405)
  **Contributor**: Erica177
  **Change Log**: 修复了多个文件中`McpStreamableProtocol`常量的拼写错误，确保协议支持映射、上游类型映射和路由重写逻辑正确。
  **Feature Value**: 修正拼写错误后，确保了MCP协议支持的一致性和正确性，避免因拼写问题导致的协议识别失败或映射异常，提升系统稳定性。

- **Related PR**: [#2398](https://github.com/alibaba/higress/pull/2398)
  **Contributor**: Erica177
  **Change Log**: 修复了McpStreambleProtocol常量的拼写错误，并将processServerConfig函数中硬编码的命名空间值替换为常量。
  **Feature Value**: 提升了代码逻辑的正确性与可维护性，避免因拼写错误或硬编码值导致的潜在运行时问题。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#2458](https://github.com/alibaba/higress/pull/2458)
  **Contributor**: johnlanni
  **Change Log**: 将MCP服务器的依赖从higress的wasm-go仓库切换到独立的wasm-go仓库，涉及多个模块的路径调整和依赖更新。
  **Feature Value**: 提升代码维护性和独立性，减少模块间耦合，为用户提供更稳定和高效的WASM功能支持。

- **Related PR**: [#2403](https://github.com/alibaba/higress/pull/2403)
  **Contributor**: johnlanni
  **Change Log**: 统一MCP会话过滤器中的行结束标记，提升代码一致性与可维护性。
  **Feature Value**: 通过统一行结束标记减少混淆，提高代码可读性和维护性，对用户无直接影响。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2503](https://github.com/alibaba/higress/pull/2503)
  **Contributor**: CH3CHO
  **Change Log**: 修正了ai-proxy插件README文档中的配置属性名称拼写错误，将`vertexGeminiSafetySetting`更正为`geminiSafetySetting`。
  **Feature Value**: 提升文档准确性与规范性，避免用户因配置项名称错误导致的配置问题，增强使用体验。

- **Related PR**: [#2433](https://github.com/alibaba/higress/pull/2433)
  **Contributor**: johnlanni
  **Change Log**: 添加了Higress 2.1.4版本的发布说明，包括对Google Cloud Vertex AI服务的支持等新功能介绍，并补充了相关中文文档内容。
  **Feature Value**: 为用户提供清晰的版本更新信息，帮助用户快速了解2.1.4版本的新功能、改进和使用变化，提升使用体验和升级效率。

- **Related PR**: [#2418](https://github.com/alibaba/higress/pull/2418)
  **Contributor**: xuruidong
  **Change Log**: 修复了 mcp-servers 的 README_zh.md 中的断开链接，更新了 GJSON Template 语法的引用链接。
  **Feature Value**: 提高了文档的准确性与可读性，确保用户能正确访问相关工具和文档资源，提升使用体验。

- **Related PR**: [#2327](https://github.com/alibaba/higress/pull/2327)
  **Contributor**: hourmoneys
  **Change Log**: 新增了mcp-server的文档，包括工具功能说明和配置文件的更新。具体涉及城市残保金年份和基数查询、社保计算等工具的描述配置。
  **Feature Value**: 为用户提供清晰的mcp-server功能说明与配置指导，提升易用性与接入效率，帮助开发者快速理解与使用相关工具。

---

## 📊 发布统计

- 🚀 新功能: 20项
- 🐛 Bug修复: 15项
- ♻️ 重构优化: 2项
- 📚 文档更新: 4项

**总计**: 41项更改（包含9项重要更新）

感谢所有贡献者的辛勤付出！🎉
\n
# Higress Console


## 📋 本次发布概览

本次发布包含 **8** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 3项
- **Bug修复**: 3项
- **文档更新**: 1项
- **测试改进**: 1项

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#540](https://github.com/higress-group/higress-console/pull/540)
  **Contributor**: CH3CHO
  **Change Log**: 新增对Vertex LLM provider类型的支持，包括认证机制与配置界面实现，扩展了AI服务接入能力
  **Feature Value**: 用户可通过Vertex AI平台接入Google Gemini等先进模型，提升AI服务的可扩展性与多云支持能力

- **Related PR**: [#530](https://github.com/higress-group/higress-console/pull/530)
  **Contributor**: Thomas-Eliot
  **Change Log**: 该PR主要实现了MCP Server的控制台管理功能，通过新增及修改多个模块代码，增强了服务端的配置管理与操作能力。
  **Feature Value**: 为用户提供MCP Server的可视化管理功能，提升配置操作便捷性与服务管理效率，对系统扩展性有积极影响。

- **Related PR**: [#529](https://github.com/higress-group/higress-console/pull/529)
  **Contributor**: CH3CHO
  **Change Log**: 新增AI路由上游的多模型映射规则配置功能，通过弹窗形式支持高级配置编辑，提升路由策略的灵活性。
  **Feature Value**: 用户可针对不同模型定义映射规则，增强AI路由配置的多样性和精准度，提升服务适配能力和使用体验。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#537](https://github.com/higress-group/higress-console/pull/537)
  **Contributor**: CH3CHO
  **Change Log**: 修复了`URL.parse`函数的兼容性问题，替换为`new URL()`以支持更多浏览器版本。
  **Feature Value**: 提升了应用在不同浏览器环境下的兼容性与稳定性，确保功能可正常使用。

- **Related PR**: [#528](https://github.com/higress-group/higress-console/pull/528)
  **Contributor**: cr7258
  **Change Log**: 将PVC访问模式的默认值从`rwxSupported: true`更改为`false`，以匹配更常用的`ReadWriteOnce`模式，并避免不必要的配置。
  **Feature Value**: 优化默认配置，减少资源浪费和潜在的配置错误，提升用户部署的稳定性和合理性，同时允许需要多副本的用户手动启用`ReadWriteMany`。

- **Related PR**: [#525](https://github.com/higress-group/higress-console/pull/525)
  **Contributor**: NorthernBob
  **Change Log**: 该PR将配置中的字段名从"UrlPattern"更正为"urlPattern"，修正了命名格式的一致性问题。
  **Feature Value**: 通过统一字段命名规范，提升了配置的可读性和维护性，避免了因命名不一致导致的潜在错误。

### 📚 文档更新 (Documentation)

- **Related PR**: [#538](https://github.com/higress-group/higress-console/pull/538)
  **Contributor**: zhangjingcn
  **Change Log**: 更新了MCP服务器插件文档，修正了errorResponseTemplate的触发条件描述，修复了GJSON路径中的错误转义问题。
  **Feature Value**: 帮助用户正确配置错误响应模板，避免因状态码判断错误导致模板误触发，提高配置准确性和易用性。

### 🧪 测试改进 (Testing)

- **Related PR**: [#526](https://github.com/higress-group/higress-console/pull/526)
  **Contributor**: CH3CHO
  **Change Log**: 新增了一个单元测试用例，用于检查Wasm插件镜像是否为最新版本，通过比较当前使用的镜像标签和最新标签的清单实现。
  **Feature Value**: 确保Wasm插件镜像保持最新，避免潜在的安全和功能问题，提升系统的稳定性和安全性。

---

## 📊 发布统计

- 🚀 新功能: 3项
- 🐛 Bug修复: 3项
- 📚 文档更新: 1项
- 🧪 测试改进: 1项

**总计**: 8项更改

感谢所有贡献者的辛勤付出！🎉
\n
