# Higress


## 📋 本次发布概览

本次发布包含 **42** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 21项
- **Bug修复**: 14项
- **重构优化**: 4项
- **文档更新**: 2项
- **测试改进**: 1项

### ⭐ 重点关注

本次发布包含 **3** 项重要更新，建议重点关注：

- **feat: add MCP SSE stateful session load balancer support** ([#2818](https://github.com/alibaba/higress/pull/2818)): 此功能使得基于SSE协议的MCP服务能够更好地保持客户端与服务器之间的持久连接，增强用户体验和应用性能，特别是在需要维持长时间连接以进行数据推送的场景中。
- **feat: Support adding a proxy server in between when forwarding requests to upstream** ([#2710](https://github.com/alibaba/higress/pull/2710)): 此功能允许用户在转发请求到上游服务时使用代理服务器，增强了系统的灵活性和安全性，适用于需要通过特定代理进行通信的场景。
- **feat(ai-proxy): add auto protocol compatibility for OpenAI and Claude APIs** ([#2810](https://github.com/alibaba/higress/pull/2810)): 通过自动协议检测与转换，使得所有AI Provider可以同时兼容OpenAI协议和Claude协议，可以丝滑对接Claude Code

详细信息请查看下方重要功能详述部分。

---

## 🌟 重要功能详述

以下是本次发布中的重要功能和改进的详细说明：

### 1. feat: add MCP SSE stateful session load balancer support

**相关PR**: [#2818](https://github.com/alibaba/higress/pull/2818) | **贡献者**: [@johnlanni](https://github.com/johnlanni)

**使用背景**

随着实时通信需求的增长，Server-Sent Events (SSE) 成为了许多应用的关键技术。然而，在分布式系统中，如何确保同一个客户端的请求始终被路由到相同的后端服务以保持会话状态成为了一个挑战。传统的负载均衡策略无法满足这一需求。本功能针对这一问题，引入了MCP SSE状态会话负载均衡支持。通过在`higress.io/load-balance`注解中指定`mcp-sse`类型，用户可以轻松实现SSE连接的状态会话管理。目标用户群体主要是需要在分布式环境中进行实时数据推送的应用开发者和服务提供商。

**功能详述**

本次PR主要实现了以下功能：
1. **扩展`load-balance`注解**：在`loadbalance.go`文件中增加了对`mcp-sse`值的支持，并在`LoadBalanceConfig`结构体中添加了`McpSseStateful`字段。
2. **简化配置**：用户只需在`higress.io/load-balance`注解中设置`mcp-sse`，即可启用该功能，无需额外配置。
3. **后台地址编码**：当启用了MCP SSE状态会话负载均衡后，后端地址将被Base64编码并嵌入到SSE消息的会话ID中。这样可以确保客户端能够正确地识别和维护会话。
核心技术创新在于通过EnvoyFilter动态生成SSE会话相关的配置，从而实现状态会话管理。

**使用方式**

要使用此功能，用户需要按照以下步骤操作：
1. **启用功能**：在Ingress资源中添加`higress.io/load-balance: mcp-sse`注解。
2. **配置示例**：
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: sse-ingress
  annotations:
    higress.io/load-balance: mcp-sse
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /mcp-servers/test/sse
        pathType: Prefix
        backend:
          service:
            name: sse-service
            port:
              number: 80
```
3. **测试**：通过`curl`命令访问SSE端点，检查返回的消息中是否包含正确的会话ID。
注意事项：
- 确保后端服务能够处理Base64编码的会话ID。
- 避免频繁更改后端服务部署，以免影响会话的一致性。

**功能价值**

此功能为用户带来了以下具体好处：
1. **会话一致性**：确保同一个客户端的请求始终被路由到相同的后端服务，从而保持会话状态的一致性。
2. **简化配置**：通过简单的注解配置即可启用功能，降低了用户的配置复杂度。
3. **提升用户体验**：对于依赖于SSE的应用，如实时通知、股票行情等，能够提供更稳定和一致的服务体验。
4. **降低运维成本**：减少了因会话不一致导致的错误和故障，降低了运维团队的工作负担。

---

### 2. feat: Support adding a proxy server in between when forwarding requests to upstream

**相关PR**: [#2710](https://github.com/alibaba/higress/pull/2710) | **贡献者**: [@CH3CHO](https://github.com/CH3CHO)

**使用背景**

在现代微服务架构中，特别是在复杂的网络环境中，直接将请求从客户端转发到后端服务可能会遇到各种问题，如网络安全、性能瓶颈等。引入中间代理服务器可以有效解决这些问题，例如通过代理服务器进行流量控制、负载均衡、SSL卸载等操作。此外，在某些情况下，企业可能需要使用特定的代理服务器来满足合规性和安全要求。此功能的目标用户群体主要是需要在复杂网络环境中优化请求转发路径的企业和开发者们。

**功能详述**

该PR主要实现了在McpBridge资源中配置一个或多个代理服务器，并允许为每个注册表配置指定的代理服务器。具体实现包括：1. 在`McpBridge`资源定义中添加了`proxies`字段用于配置代理服务器列表，以及在`registries`项中添加了`proxyName`字段以关联代理服务器与注册表。2. 当创建或更新`McpBridge`资源时，系统会根据配置自动生成相应的EnvoyFilter资源，这些资源定义了如何将请求转发至指定的代理服务器。3. 此外，还生成了针对每个绑定有代理的服务的EnvoyFilter，确保它们能够正确地指向对应代理服务器上的本地监听器。整个技术实现基于Envoy的高级路由能力，展示了项目在处理复杂网络拓扑方面的强大功能。

**使用方式**

启用此功能首先需要在`McpBridge`资源中配置至少一个代理服务器。这可以通过向`spec.proxies`数组中添加新的`ProxyConfig`对象完成，每个对象需包含诸如`name`、`serverAddress`、`serverPort`等必要信息。接着，对于希望使用代理服务器的注册表条目，只需在其`proxyName`字段中引用已定义的代理名称即可。一旦配置好，系统会自动处理所有相关的EnvoyFilter生成工作。值得注意的是，在实际部署前应该仔细检查配置文件的正确性，避免因错误配置导致的服务不可用等问题。

**功能价值**

新增加的代理服务器支持功能极大地增强了系统的网络灵活性，使得用户可以根据自身需求灵活地调整请求转发路径。比如，通过设置不同的代理服务器，可以轻松实现多地域间的数据传输优化；同时，借助于代理层提供的额外安全特性（如SSL加密），也大大提高了整个系统的安全性。另外，这一功能还有助于简化运维管理，尤其是在需要频繁调整网络架构的情况下，通过简单的配置更改就能快速响应变化，无需对底层基础架构做出重大修改。总而言之，这项改进不仅扩展了项目的适用范围，也为用户提供了更强有力的工具来应对日益复杂的网络挑战。

---

### 3. feat(ai-proxy): add auto protocol compatibility for OpenAI and Claude APIs

**相关PR**: [#2810](https://github.com/alibaba/higress/pull/2810) | **贡献者**: [@johnlanni](https://github.com/johnlanni)

**使用背景**

在AI代理插件中，用户可能需要同时与多个AI服务提供商（如OpenAI和Anthropic Claude）进行交互。这些提供商通常使用不同的API协议，导致用户在切换服务时需要手动配置协议类型，增加了复杂性和出错的可能性。此功能解决了这一问题，使用户能够无缝地使用不同提供商的服务，而无需关心底层协议的差异。目标用户群体是那些希望简化AI服务集成过程的开发者和企业。

**功能详述**

本PR实现了自动协议兼容功能，核心技术创新在于自动检测请求路径并根据目标提供商的能力智能地进行协议转换。具体来说，当请求路径为`/v1/chat/completions`时，识别为OpenAI协议；当请求路径为`/v1/messages`时，识别为Claude协议。如果目标提供商不支持原生Claude协议，插件会将请求从Claude格式转换为OpenAI格式，反之亦然。在`main.go`文件中，新增了基于请求路径的自动协议检测逻辑，并在必要时进行路径替换。此外，新增了`claude_to_openai.go`文件，用于实现Claude到OpenAI协议的具体转换逻辑。

**使用方式**

启用此功能非常简单，用户只需像往常一样发送请求即可，无需额外配置。例如，对于OpenAI协议的请求，URL为`http://your-domain/v1/chat/completions`，而对于Claude协议的请求，URL为`http://your-domain/v1/messages`。插件会自动检测并处理协议转换。如果目标提供商不支持Claude协议，插件会将其转换为OpenAI格式。示例配置如下：

```yaml
provider:
  type: claude  # 原生支持Claude协议的供应商
  apiTokens:
    - 'YOUR_CLAUDE_API_TOKEN'
  version: '2023-06-01'
```

注意事项：确保正确配置API令牌和版本号，以便插件能够正确识别和处理请求。

**功能价值**

此功能显著提升了AI代理插件的易用性和灵活性，减少了用户的配置负担。通过自动协议检测和智能转换，用户可以更轻松地在不同AI服务提供商之间切换，而无需担心协议兼容性问题。这不仅提高了开发效率，还增强了系统的稳定性和可靠性。此外，该功能还支持流式响应，进一步扩展了其应用场景，特别是在需要实时交互的场景中。总之，此功能为用户提供了一种更高效、更简便的方式来集成和管理多AI服务提供商。

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#2847](https://github.com/alibaba/higress/pull/2847) \
  **Contributor**: @Erica177 \
  **Change Log**: 此PR为Nacos MCP添加了安全模式，涉及对mcp_model.go和watcher.go文件的修改，包括新增和调整配置项。 \
  **Feature Value**: 通过增加安全模式支持，提升了Nacos MCP服务的安全性，允许用户在更安全的环境下管理其微服务配置。

- **Related PR**: [#2842](https://github.com/alibaba/higress/pull/2842) \
  **Contributor**: @hanxiantao \
  **Change Log**: 为hmac-auth-apisix插件添加了详细的中文和英文文档，并增加了相应的测试用例以确保新添加的功能稳定可靠。 \
  **Feature Value**: 通过增加文档与测试，提高了hmac-auth-apisix插件的可用性和稳定性，帮助用户更好地理解和使用HMAC认证机制，增强API的安全性。

- **Related PR**: [#2823](https://github.com/alibaba/higress/pull/2823) \
  **Contributor**: @johnlanni \
  **Change Log**: 新增了OpenRouter作为AI服务提供商，支持通过统一API访问多种AI模型。核心实现包括chat completions和text completions的支持。 \
  **Feature Value**: 通过引入OpenRouter，用户可以更灵活地选择不同的AI模型并进行交互，简化了跨平台使用AI服务的复杂性，提升了用户体验。

- **Related PR**: [#2815](https://github.com/alibaba/higress/pull/2815) \
  **Contributor**: @hanxiantao \
  **Change Log**: 本PR添加了hmac-auth-apisix插件，实现了对API请求的身份验证功能。通过HMAC算法生成签名来验证请求的完整性与真实性。 \
  **Feature Value**: 新增的hmac-auth-apisix插件增强了系统的安全性，确保只有经过身份验证的客户端才能访问受保护的资源，提升了用户体验和系统防护能力。

- **Related PR**: [#2808](https://github.com/alibaba/higress/pull/2808) \
  **Contributor**: @daixijun \
  **Change Log**: 添加了Anthropic API和OpenAI v1/models接口的支持，扩展了DeepSeek的兼容性和功能范围。 \
  **Feature Value**: 引入的新支持使得用户能够利用更多的人工智能服务选项，增强了系统的灵活性和实用性。

- **Related PR**: [#2805](https://github.com/alibaba/higress/pull/2805) \
  **Contributor**: @johnlanni \
  **Change Log**: 新增了JSON-RPC协议转换插件，能够从MCP协议中提取请求和响应信息到头部，便于进一步的观察、限流、认证等处理。 \
  **Feature Value**: 该功能允许用户在A2A协议中利用JSON-RPC进行更高级别的策略控制，如身份验证和流量管理，从而提高了系统的灵活性与安全性。

- **Related PR**: [#2788](https://github.com/alibaba/higress/pull/2788) \
  **Contributor**: @zat366 \
  **Change Log**: 此PR更新了mcp-server中的依赖项github.com/higress-group/wasm-go，以支持MCP插件响应图片。通过更新go.mod和go.sum文件实现。 \
  **Feature Value**: 新增功能允许MCP插件处理并响应图像数据，增强了系统的多媒体处理能力，为用户提供更丰富的内容展示选项。

- **Related PR**: [#2769](https://github.com/alibaba/higress/pull/2769) \
  **Contributor**: @github-actions[bot] \
  **Change Log**: 此PR在helm文件夹中更新了CRD文件，增加了关于proxies的新属性定义。 \
  **Feature Value**: 通过更新CRD文件增加新属性，使Kubernetes资源定义更加丰富和完善，提升了系统的配置灵活性和可扩展性。

- **Related PR**: [#2761](https://github.com/alibaba/higress/pull/2761) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR引入了两种新的去重策略：SPLIT_AND_RETAIN_FIRST和SPLIT_AND_RETAIN_LAST，分别用于保留逗号分隔的头部值的第一个和最后一个元素。 \
  **Feature Value**: 新策略为用户提供了更细粒度的控制选项，允许他们在去重操作时根据需求选择保留特定位置的数据，从而更好地满足多样化的需求。

- **Related PR**: [#2739](https://github.com/alibaba/higress/pull/2739) \
  **Contributor**: @WeixinX \
  **Change Log**: 新增了一个插件配置字段`reroute`，允许用户控制是否禁用路由重新选择。这一功能通过修改主要配置文件及添加相关测试用例来实现。 \
  **Feature Value**: 该功能为用户提供了一种方式来精细化控制请求处理过程中的路由行为，增强了系统的灵活性和可配置性，满足了特定场景下的需求。

- **Related PR**: [#2730](https://github.com/alibaba/higress/pull/2730) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR为Bedrock服务添加了工具使用支持，通过修改bedrock.go等文件中的结构体和逻辑，使系统能够处理与工具相关的请求。 \
  **Feature Value**: 新增加的功能允许用户在Bedrock环境下有效地利用工具调用能力，提升了系统的灵活性和功能性，更好地满足了需要集成外部工具的应用场景需求。

- **Related PR**: [#2729](https://github.com/alibaba/higress/pull/2729) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR增加了AI统计插件中每个value的长度限制，当超过设定长度时自动截断。这有助于减少处理大文件（如base64编码的图片、视频）时的内存占用。 \
  **Feature Value**: 通过限制和截断过长的数据值，该功能可以有效防止因记录大型媒体文件导致的内存溢出问题，从而提高系统的稳定性和性能表现。

- **Related PR**: [#2713](https://github.com/alibaba/higress/pull/2713) \
  **Contributor**: @Aias00 \
  **Change Log**: 此PR为AI代理添加了Grok提供商支持，包括新增Grok Go文件实现与更新相关文档。 \
  **Feature Value**: 通过集成Grok作为新的AI提供商，用户现在可以利用Grok的AI能力来处理请求，增加了系统的灵活性和功能多样性。

- **Related PR**: [#2712](https://github.com/alibaba/higress/pull/2712) \
  **Contributor**: @SCMRCORE \
  **Change Log**: 增加了对Gemini模型thinking功能的支持，特别针对2.5 Flash、2.5 Pro和2.5 Flash-Lite三种模型进行了适配。 \
  **Feature Value**: 增强了AI代理插件的功能性，允许用户利用特定的Gemini模型进行更复杂的思考任务，提升了用户体验与应用范围。

- **Related PR**: [#2704](https://github.com/alibaba/higress/pull/2704) \
  **Contributor**: @hanxiantao \
  **Change Log**: 该PR实现了Rust WASM插件支持Redis数据库配置选项的功能，同时改进了demo-wasm以从Wasm插件配置中获取Redis配置。 \
  **Feature Value**: 此功能允许开发者在使用Rust WASM插件时能够更加灵活地配置和集成Redis数据库，提高了开发效率和应用的可配置性。

- **Related PR**: [#2698](https://github.com/alibaba/higress/pull/2698) \
  **Contributor**: @erasernoob \
  **Change Log**: 实现了Gemini模型对多模态的支持，增加了处理图片和文本的能力。通过引入新依赖和修改现有代码逻辑增强功能。 \
  **Feature Value**: 增强了AI代理插件的功能，使其能够支持更复杂的多模态数据处理，为用户提供更加丰富和灵活的AI服务体验。

- **Related PR**: [#2696](https://github.com/alibaba/higress/pull/2696) \
  **Contributor**: @rinfx \
  **Change Log**: 此PR在启用内容安全插件时引入了流式响应支持，通过`bufferLimit`参数调整检测频率，提高了内容检测的灵活性和效率。 \
  **Feature Value**: 新增的流式响应功能允许用户更高效地处理内容安全检测，减少延迟，提高用户体验，特别适用于需要实时反馈的应用场景。

- **Related PR**: [#2671](https://github.com/alibaba/higress/pull/2671) \
  **Contributor**: @Aias00 \
  **Change Log**: 实现了路径后缀和内容类型过滤功能，以解决ai-statistics插件的性能与资源管理问题。通过引入SkipProcessing机制，避免了对所有请求的无差别处理，减少了不必要的响应体缓存。 \
  **Feature Value**: 增强了AI统计插件的选择性处理能力，提升了系统性能并优化了资源使用效率，对于大量且复杂的API请求场景尤其有益，可显著改善用户体验。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#2816](https://github.com/alibaba/higress/pull/2816) \
  **Contributor**: @Asnowww \
  **Change Log**: 该PR修正了文件`scanners-user-agents.data`中的拼写错误，将'scannr'更正为'scanner'。 \
  **Feature Value**: 修正文档中的拼写错误可以提高文档的准确性和可读性，有助于用户更好地理解和使用相关功能。

- **Related PR**: [#2799](https://github.com/alibaba/higress/pull/2799) \
  **Contributor**: @erasernoob \
  **Change Log**: 修正了wasm-go-build插件构建命令，确保编译时包含了目录中的所有文件，解决了由于依赖关系导致的编译失败问题。 \
  **Feature Value**: 通过修复编译命令，避免因缺少必要文件而引起的编译错误，提升了构建过程的稳定性和可靠性，为开发者提供了更好的开发体验。

- **Related PR**: [#2787](https://github.com/alibaba/higress/pull/2787) \
  **Contributor**: @co63oc \
  **Change Log**: 修复了RegisteTickFunc函数的拼写错误，确保了定时任务注册功能的正确性。通过更正关键函数名，避免了潜在的功能失效问题。 \
  **Feature Value**: 修正了因拼写错误导致的定时任务无法正常注册的问题，提升了系统的稳定性和可靠性，保障了依赖于定时任务执行的应用程序按预期运行。

- **Related PR**: [#2786](https://github.com/alibaba/higress/pull/2786) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR移除了mcp-session过滤器处理SSE传输请求时的'accept-encoding'头部，解决了无法正确处理压缩响应体数据的问题。 \
  **Feature Value**: 该修复确保了MCP服务器上游使用SSE传输时能够正常工作，避免了因压缩导致的数据解析错误，提升了系统的稳定性和可靠性。

- **Related PR**: [#2782](https://github.com/alibaba/higress/pull/2782) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR修复了Azure URL配置组件被意外更改的问题，通过定义新的枚举类型azureServiceUrlType来确保URL组件的正确性和一致性。 \
  **Feature Value**: 该修复保证了用户在使用AI代理时能够保持他们对Azure服务URL的原始配置，避免因错误更改而导致的服务调用失败或不一致问题。

- **Related PR**: [#2757](https://github.com/alibaba/higress/pull/2757) \
  **Contributor**: @Jing-ze \
  **Change Log**: 修复了mcp server构建Envoy过滤器单元测试的问题，确保了测试用例的正确性和稳定性。 \
  **Feature Value**: 通过修复单元测试中的错误，增强了代码的可靠性和可维护性，帮助开发者更好地进行后续开发和调试工作。

- **Related PR**: [#2755](https://github.com/alibaba/higress/pull/2755) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR修复了在ip-restriction配置中添加重复IP时抛出错误的问题，通过忽略已存在IP的错误并显示iptree返回的具体错误详情。 \
  **Feature Value**: 允许在IP限制列表中存在重复项，提高了配置灵活性和用户体验，同时确保其他类型的错误仍能得到有效处理。

- **Related PR**: [#2754](https://github.com/alibaba/higress/pull/2754) \
  **Contributor**: @Jing-ze \
  **Change Log**: 修正了golang-filter中解码数据时的停止和缓冲问题，确保数据处理流程更加稳定。 \
  **Feature Value**: 解决了数据解码过程中的错误，提高了系统的可靠性和用户体验，避免了潜在的数据丢失或处理异常。

- **Related PR**: [#2743](https://github.com/alibaba/higress/pull/2743) \
  **Contributor**: @Jing-ze \
  **Change Log**: 修复了设置ip_source_type为origin-source时的错误，确保了IP限制功能可以正确地根据源类型进行配置。 \
  **Feature Value**: 此修复解决了在特定条件下IP源类型设置不正确的问题，提高了系统的稳定性和安全性，让用户能够更可靠地使用IP限制功能。

- **Related PR**: [#2723](https://github.com/alibaba/higress/pull/2723) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修正了C++ Wasm插件中_match_service_规则由于使用错误属性名导致的功能异常问题，通过修改为正确的属性名称使规则恢复正常。 \
  **Feature Value**: 解决了因匹配规则错误而导致的服务路由问题，提高了系统的稳定性和准确性，确保用户能够正确地访问到所需的服务。

- **Related PR**: [#2706](https://github.com/alibaba/higress/pull/2706) \
  **Contributor**: @WeixinX \
  **Change Log**: 修复了transformer在替换键不存在时执行添加操作的问题，并增加了映射操作的测试用例，确保从headers/querys到body以及从body到headers/querys的转换正确。 \
  **Feature Value**: 此修复提高了系统的稳定性和可靠性，防止了错误的数据操作，增强了用户对数据处理逻辑的信心，提升了用户体验。

- **Related PR**: [#2663](https://github.com/alibaba/higress/pull/2663) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了bedrock模型名称转义逻辑中的错误，移除了请求体中不必要的URL编码处理，并确保返回的响应与预期一致。 \
  **Feature Value**: 通过修正名称转义逻辑问题，提高了系统的稳定性和兼容性，确保了用户在使用过程中不会遇到由于转义不匹配导致的问题。

- **Related PR**: [#2653](https://github.com/alibaba/higress/pull/2653) \
  **Contributor**: @johnlanni \
  **Change Log**: 修复了当使用Bedrock时，AI路由回退功能失效的问题。通过确保即使在headers为nil的情况下也能正确获取路径来避免空指针异常。 \
  **Feature Value**: 此修复解决了特定条件下签名验证失败导致的请求被拒绝问题，提高了系统的稳定性和可靠性，确保用户可以顺利访问服务。

- **Related PR**: [#2628](https://github.com/alibaba/higress/pull/2628) \
  **Contributor**: @co63oc \
  **Change Log**: 此PR修正了多个文件中的拼写错误，共涉及5个文件36行代码的修改，确保了文档和注释的准确性。 \
  **Feature Value**: 修正拼写错误提高了代码库的专业性，使开发者在阅读文档时能够更准确地理解内容，从而减少因误解导致的错误。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#2777](https://github.com/alibaba/higress/pull/2777) \
  **Contributor**: @StarryVae \
  **Change Log**: 更新了ai-prompt-decorator插件至新的封装API，改进了初始化配置与请求头处理方法的调用方式。 \
  **Feature Value**: 此次重构提高了代码的一致性和可维护性，使得开发者能够更方便地集成和使用ai-prompt-decorator功能。

- **Related PR**: [#2773](https://github.com/alibaba/higress/pull/2773) \
  **Contributor**: @CH3CHO \
  **Change Log**: 重构了ai-proxy中的路径到API名称的映射逻辑，引入正则表达式简化映射过程，并新增了测试用例以验证功能正确性。 \
  **Feature Value**: 通过优化路径映射逻辑结构，提高了代码可维护性和扩展性，使得支持更多路径变得更加容易，间接提升了系统的灵活性和用户体验。

- **Related PR**: [#2740](https://github.com/alibaba/higress/pull/2740) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR将`ai-statistics`组件中部分日志级别从`warn`下调至`info`，以更准确地反映这些日志信息的实际重要性。 \
  **Feature Value**: 通过调整日志级别，使日志记录更加符合实际需求，有助于降低用户在查看日志时的误报警率，提升用户体验。

- **Related PR**: [#2711](https://github.com/alibaba/higress/pull/2711) \
  **Contributor**: @johnlanni \
  **Change Log**: 本PR将mcp server和tool中使用斜杠作为连接符的方式废弃，改为使用更符合函数命名规范的格式。这包括更新了部分代码库中的依赖版本，并对相关文件进行了调整。 \
  **Feature Value**: 通过遵循标准的函数命名约定，这次改动增强了代码的一致性和可读性，有助于降低未来维护的成本，同时也减少了由于不合规命名导致的潜在错误。

### 📚 文档更新 (Documentation)

- **Related PR**: [#2770](https://github.com/alibaba/higress/pull/2770) \
  **Contributor**: @co63oc \
  **Change Log**: 修正了多个文件中的拼写错误，包括测试文件、README以及Go代码中的变量名和配置项名称。 \
  **Feature Value**: 提高了文档的准确性和可读性，确保了代码的一致性和用户体验。对于使用该插件的用户来说，这些更改有助于避免因拼写错误导致的混淆或配置问题。

### 🧪 测试改进 (Testing)

- **Related PR**: [#2809](https://github.com/alibaba/higress/pull/2809) \
  **Contributor**: @Jing-ze \
  **Change Log**: 新增了针对多个Wasm扩展的单元测试，并引入了CI/CD工作流来自动化这些测试，确保代码质量和稳定性。 \
  **Feature Value**: 提高了Wasm插件的可靠性，通过增加全面的单元测试和自动化CI/CD流程，帮助开发者更快地发现和修复问题，提升了用户体验。

---

## 📊 发布统计

- 🚀 新功能: 21项
- 🐛 Bug修复: 14项
- ♻️ 重构优化: 4项
- 📚 文档更新: 2项
- 🧪 测试改进: 1项

**总计**: 42项更改（包含3项重要更新）

感谢所有贡献者的辛勤付出！🎉


# Higress Console


## 📋 本次发布概览

本次发布包含 **12** 项更新，涵盖了功能增强、Bug修复、性能优化等多个方面。

### 更新内容分布

- **新功能**: 5项
- **Bug修复**: 5项
- **重构优化**: 2项

---

## 📝 完整变更日志

### 🚀 新功能 (Features)

- **Related PR**: [#585](https://github.com/higress-group/higress-console/pull/585) \
  **Contributor**: @johnlanni \
  **Change Log**: 此PR增加了新的AI服务提供商，并更新了可用模型列表，包括对翻译文件的更新以支持新增加的提供商。 \
  **Feature Value**: 通过引入更多AI服务提供商及更新模型列表，用户现在可以访问更广泛的服务选项，提升了系统的灵活性和实用性。

- **Related PR**: [#582](https://github.com/higress-group/higress-console/pull/582) \
  **Contributor**: @Thomas-Eliot \
  **Change Log**: 新增了对ai-load-balancer插件的支持，使其能够在higress-console中进行可视化配置，并定义了其在系统中的优先级。 \
  **Feature Value**: 通过提供白屏配置选项，极大提升了用户对于AI负载均衡器的管理效率与灵活性，降低了使用门槛。

- **Related PR**: [#579](https://github.com/higress-group/higress-console/pull/579) \
  **Contributor**: @JayLi52 \
  **Change Log**: 本次更新为MCP服务器管理功能添加了对PostgreSQL和ClickHouse数据库的支持，同时优化了MySQL数据库连接字符串格式，并修复了一些数据库连接相关的问题。 \
  **Feature Value**: 新增的数据库支持扩大了MCP的应用范围，使用户能够更灵活地选择适合其需求的数据库类型，提升了系统的兼容性和用户体验。

- **Related PR**: [#572](https://github.com/higress-group/higress-console/pull/572) \
  **Contributor**: @CH3CHO \
  **Change Log**: 此PR添加了管理代理服务器的功能，包括新类和服务控制器，允许用户配置和管理代理服务器。 \
  **Feature Value**: 通过新增的支持，用户能够更灵活地管理和配置代理服务器，提高了系统的灵活性和可用性。

- **Related PR**: [#565](https://github.com/higress-group/higress-console/pull/565) \
  **Contributor**: @Aias00 \
  **Change Log**: 该PR改进了MCP服务器管理任务6和7，包括更新README.md文档、修改系统服务实现代码及优化ConfigMap处理逻辑。 \
  **Feature Value**: 通过改进MCP服务器管理功能，提高了系统的稳定性和可维护性，简化了用户对Higress配置的管理，提升了用户体验。

### 🐛 Bug修复 (Bug Fixes)

- **Related PR**: [#584](https://github.com/higress-group/higress-console/pull/584) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了在启用认证但没有允许的消费者时出现的错误，包括不正确地清空允许消费者列表和显示错误的认证状态。 \
  **Feature Value**: 确保即使在没有允许消费者的情况下，认证功能也能正常工作，并且用户界面能够准确反映当前的认证状态。

- **Related PR**: [#581](https://github.com/higress-group/higress-console/pull/581) \
  **Contributor**: @hongzhouzi \
  **Change Log**: 修复了更新openapi mcp server时出现的NPE异常，并修正了PostgreSQL枚举值以确保与Higress中的常量一致。 \
  **Feature Value**: 通过解决NPE问题提高了系统的稳定性和可靠性，同时枚举值的一致性改进了配置管理的准确性，减少了潜在的错误源。

- **Related PR**: [#577](https://github.com/higress-group/higress-console/pull/577) \
  **Contributor**: @CH3CHO \
  **Change Log**: 同步了前后端对域名正则表达式的校验模式，确保长顶级域名如`test.internal`能够被接受，涉及少量代码修改和新增测试用例。 \
  **Feature Value**: 解决了因前后端使用的域名验证规则不一致导致的部分合法域名无法通过的问题，提升了系统的兼容性和用户体验。

- **Related PR**: [#574](https://github.com/higress-group/higress-console/pull/574) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修复了根据internal标志过滤V1alpha1WasmPlugin时的逻辑错误，确保非内部实例不会被误返回。 \
  **Feature Value**: 提高了系统准确性，确保用户获取到正确的插件实例列表，避免了因逻辑错误导致的数据不一致问题。

- **Related PR**: [#570](https://github.com/higress-group/higress-console/pull/570) \
  **Contributor**: @CH3CHO \
  **Change Log**: 修正了一个拼写错误，该错误导致在编辑OpenAI类型的LLM提供者时出现'Cannot read properties of undefined'的报错。 \
  **Feature Value**: 通过修复此问题，避免了用户在配置OpenAI服务提供商时遇到运行时错误，提升了系统的稳定性和用户体验。

### ♻️ 重构优化 (Refactoring)

- **Related PR**: [#573](https://github.com/higress-group/higress-console/pull/573) \
  **Contributor**: @CH3CHO \
  **Change Log**: 重构了MCP服务器集成的认证模块，使得常规路由和MCP服务器可以共享相同的认证逻辑。主要变更包括在多个文件中添加、删除及修改代码。 \
  **Feature Value**: 通过重构认证模块，实现了认证逻辑的统一化，提升了代码的可维护性和复用性，减少了重复代码，有助于提高系统的整体稳定性和性能。

- **Related PR**: [#571](https://github.com/higress-group/higress-console/pull/571) \
  **Contributor**: @JayLi52 \
  **Change Log**: 通过更新Monaco编辑器的引入方式，并配置按需加载，优化了EditToolDrawer、McpServerCommand和MCPDetail组件中的性能。 \
  **Feature Value**: 提高了应用加载速度与响应效率，减少了不必要的资源消耗，提升了用户体验。

---

## 📊 发布统计

- 🚀 新功能: 5项
- 🐛 Bug修复: 5项
- ♻️ 重构优化: 2项

**总计**: 12项更改

感谢所有贡献者的辛勤付出！🎉


