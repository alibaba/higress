# MCP-GUARD 架构图集

## 1. 整体系统架构

```mermaid
graph TB
    %% 客户端层
    subgraph Client["🖥️ 客户端层"]
        C1[tenantA<br/>白金客户]
        C2[tenantB<br/>标准客户]
        C3[未授权用户]
    end

    %% 网关层
    subgraph Gateway["🌐 Higress Gateway (数据面)"]
        E[Envoy 代理]
        subgraph Filter["🔍 HTTP Filter Chain"]
            MG[mcp-guard<br/>Wasm插件<br/>优先级: 0]
            AP[ai-proxy<br/>Wasm插件<br/>优先级: 100]
            R[Envoy Router]
        end
    end

    %% 控制层
    subgraph Control["⚙️ Higress Controller (控制面)"]
        IC[Ingress Config<br/>转换/聚合]
        WP[WasmPlugin<br/>控制器]
        XDS[xDS Server<br/>配置分发]
    end

    %% 外部服务
    subgraph Service["🚀 后端服务"]
        AI[DeepSeek AI]
        TEST[测试后端]
    end

    %% 认证层
    subgraph Auth["🔐 认证层"]
        JWT[jwt-authn<br/>或 jwt-auth]
    end

    %% 连接关系
    C1 -->|HTTP(S)| E
    C2 -->|HTTP(S)| E
    C3 -->|HTTP(S)| E

    E --> Filter
    MG -->|授权通过| AP
    AP -->|转发| R
    R -->|下游调用| Service

    IC -->|WasmPlugin| XDS
    XDS -->|动态配置| E

    Auth -.->|注入身份| MG

    style MG fill:#ff6b6b,stroke:#d63031,stroke-width:3px,color:#fff
    style C1 fill:#4ecdc4,stroke:#00b894,color:#000
    style C2 fill:#45b7d1,stroke:#0984e3,color:#000
    style C3 fill:#e17055,stroke:#d63031,color:#fff
```

## 2. 请求处理时序图

### 场景1: 授权访问 (tenantA 访问 summarize)

```mermaid
sequenceDiagram
    participant C as Client
    participant E as Envoy
    participant G as mcp-guard
    participant A as ai-proxy
    participant S as Backend

    Note over C,S: 授权访问流程
    C->>E: POST /v1/text:summarize<br/>X-Subject: tenantA<br/>X-MCP-Capability: cap.text.summarize
    E->>G: onHttpRequestHeaders
    G->>G: 检查权限<br/>intersection([summarize], [summarize,translate]) = [summarize]
    G->>E: 允许继续 (ActionContinue)
    E->>A: 转发请求
    A->>S: 调用AI服务
    S-->>A: 返回结果
    A-->>E: 流式响应
    E-->>C: 200 OK + 摘要结果
```

### 场景2: 越权访问 (tenantB 访问 translate)

```mermaid
sequenceDiagram
    participant C as Client
    participant E as Envoy
    participant G as mcp-guard

    Note over C,G: 越权访问流程
    C->>E: POST /v1/text:translate<br/>X-Subject: tenantB<br/>X-MCP-Capability: cap.text.translate
    E->>G: onHttpRequestHeaders
    G->>G: 检查权限<br/>intersection([translate], [summarize]) = []
    G->>E: 拒绝请求 (SendHttpResponse 403)
    E-->>C: 403 Forbidden<br/>mcp-guard deny: reason=no-effective-capability
```

## 3. 权限判定模型

```mermaid
graph TD
    A[请求进入 mcp-guard] --> B[提取身份主体<br/>X-Subject]
    B --> C[提取路由路径<br/>/v1/text:summarize]
    C --> D[提取请求能力<br/>X-MCP-Capability]
    D --> E[获取主体权限集<br/>tenantA: [summarize, translate]]
    D --> F[获取路由允许权限集<br/>summarize路由: [summarize]]

    E --> G[计算交集<br/>intersection()]
    F --> G

    G --> H{交集为空?}
    H -->|是| I[返回 403<br/>reason: no-effective-capability]
    H -->|否| J{请求能力为空?}
    J -->|是| K[允许访问<br/>继续后续过滤链]
    J -->|否| L{请求能力在交集中?}
    L -->|否| M[返回 403<br/>reason: requested-cap-not-allowed]
    L -->|是| K

    K --> N[交由 ai-proxy 处理]
    I --> O[终止请求]
    M --> O

    style G fill:#74b9ff,stroke:#0984e3,stroke-width:2px,color:#000
    style K fill:#00b894,stroke:#00b894,stroke-width:2px,color:#000
    style I fill:#ff7675,stroke:#d63031,stroke-width:2px,color:#fff
    style M fill:#ff7675,stroke:#d63031,stroke-width:2px,color:#fff
```

## 4. Wasm插件技术架构

```mermaid
graph LR
    subgraph "Go 源代码"
        A[main.go<br/>插件入口点]
        B[config/config.go<br/>配置解析]
        C[decision/decision.go<br/>授权判定逻辑]
        D[proxy-wasm-go-sdk<br/>Wasm SDK]
    end

    subgraph "编译构建"
        E[go build<br/>wasip1/wasm]
        F[plugin.wasm<br/>5.4MB]
    end

    subgraph "Envoy 运行时"
        G[Envoy Core<br/>代理核心]
        H[V8 Wasm VM<br/>虚拟机]
        I[HTTP Filter<br/>过滤器链]
    end

    A --> E
    B --> E
    C --> E
    D --> E

    E --> F
    F -->|动态加载| H
    H -->|执行| I
    I -->|集成| G

    style F fill:#fdcb6e,stroke:#e17055,stroke-width:3px,color:#000
    style H fill:#a29bfe,stroke:#6c5ce7,stroke-width:2px,color:#000
```

## 5. 配置分发机制 (xDS)

```mermaid
sequenceDiagram
    participant Dev as 开发者
    participant K8s as K8s API
    participant Ctrl as Controller
    participant XDS as xDS Server
    participant GW as Gateway/Envoy

    Dev->>K8s: apply WasmPlugin yaml
    K8s->>Ctrl: Watch事件通知
    Ctrl->>Ctrl: 解析配置
    Ctrl->>XDS: 注册WasmPlugin
    XDS->>GW: ADS推送配置
    GW->>GW: 加载plugin.wasm
    GW->>Ctrl: 配置就绪确认

    Note over Dev,GW: 配置变更实时同步，无需重启
```

## 6. 多租户权限模型

```mermaid
graph TD
    subgraph "租户权限配置"
        A[tenantA<br/>白金客户]
        B[tenantB<br/>标准客户]
    end

    subgraph "能力集定义"
        C[cap.text.summarize<br/>文本摘要]
        D[cap.text.translate<br/>文本翻译]
        E[cap.image.moderate<br/>图像审核]
    end

    subgraph "授权映射"
        F[白名单:<br/>tenantA → [C, D]]
        G[白名单:<br/>tenantB → [C]]
    end

    subgraph "路由规则"
        H[/v1/text:summarize<br/>→ [C]]
        I[/v1/text:translate<br/>→ [D]]
        J[/v1/images:moderate<br/>→ [E]]
    end

    A --> F
    B --> G
    F --> H
    F --> I
    G --> H

    C --> H
    D --> I
    E --> J

    style A fill:#4ecdc4,stroke:#00b894,color:#000
    style B fill:#45b7d1,stroke:#0984e3,color:#000
    style F fill:#55efc4,stroke:#00b894,color:#000
    style G fill:#74b9ff,stroke:#0984e3,color:#000
```

## 7. 测试验证流程

```mermaid
graph LR
    subgraph "测试用例"
        A[测试1:<br/>无身份访问]
        B[测试2:<br/>tenantB访问translate]
        C[测试3:<br/>tenantA访问summarize]
        D[测试4:<br/>tenantA访问translate]
    end

    subgraph "期望结果"
        E[403 Forbidden<br/>no-subject]
        F[403 Forbidden<br/>no-effective-capability]
        G[503 Service Unavailable<br/>授权通过]
        H[503 Service Unavailable<br/>授权通过]
    end

    subgraph "实际结果"
        I[✅ 403 no-subject]
        J[✅ 403 no-effective-capability]
        K[✅ 503 upstream]
        L[✅ 503 upstream]
    end

    A --> E --> I
    B --> F --> J
    C --> G --> K
    D --> H --> L

    style I fill:#00b894,stroke:#00b894,color:#fff
    style J fill:#00b894,stroke:#00b894,color:#fff
    style K fill:#00b894,stroke:#00b894,color:#fff
    style L fill:#00b894,stroke:#00b894,color:#fff
```

## 8. 业务价值架构

```mermaid
graph TB
    subgraph "业务价值"
        A[多租户治理]
        B[安全合规]
        C[灵活计费]
        D[零改造接入]
        E[运营效率]
    end

    subgraph "技术实现"
        A1[能力集授权模型<br/>主体 → 能力集]
        A2[路由级权限配置<br/>路径 → 能力集]
        B1[最小权限原则<br/>默认拒绝]
        B2[审计日志追踪<br/>每次访问记录]
        C1[套餐差异化<br/>按能力分层]
        C2[动态权限更新<br/>实时生效]
        D1[ai-proxy协议适配<br/>统一API]
        D2[对客户端透明<br/>无需修改]
        E1[xDS动态配置<br/>毫秒级推送]
        E2[可视化配置<br/>Console界面]
    end

    A --> A1
    A --> A2
    B --> B1
    B --> B2
    C --> C1
    C --> C2
    D --> D1
    D --> D2
    E --> E1
    E --> E2

    style A fill:#ff7675,stroke:#d63031,color:#fff
    style B fill:#fdcb6e,stroke:#e17055,color:#000
    style C fill:#74b9ff,stroke:#0984e3,color:#000
    style D fill:#55efc4,stroke:#00b894,color:#000
    style E fill:#a29bfe,stroke:#6c5ce7,color:#000
```

## 9. 部署架构

```mermaid
graph TB
    subgraph "开发环境"
        A[Localhost<br/>kind cluster]
        A1[Kubernetes 1.25.3]
        A2[Higress 2.1.9-rc.1]
        A3[plugin.wasm 5.4MB]
    end

    subgraph "生产环境"
        B[云原生K8s集群]
        B1[Higress Controller]
        B2[Higress Gateway × N]
        B3[WasmPlugin Registry]
        B4[DeepSeek API]
    end

    subgraph "监控运维"
        C[Prometheus<br/>指标采集]
        D[访问日志<br/>审计追踪]
        E[Higress Console<br/>可视化界面]
    end

    A --> B
    B --> C
    B --> D
    B --> E

    style A fill:#ffeaa7,stroke:#fdcb6e,color:#000
    style B fill:#81ecec,stroke:#00bcd4,color:#000
    style C fill:#b2bec3,stroke:#2d3436,color:#000
```

---

## 图例说明

| 图标 | 含义 |
|------|------|
| 🖥️ | 客户端/用户层 |
| 🌐 | 网关层 |
| ⚙️ | 控制层面 |
| 🚀 | 服务层 |
| 🔐 | 安全认证 |
| 🔍 | 过滤器/中间件 |
| ✅ | 成功/通过 |
| ❌ | 失败/拒绝 |
| 📊 | 数据/配置 |
