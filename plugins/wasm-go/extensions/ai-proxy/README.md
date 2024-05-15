---
title: AI 代理
keywords: [ higress,ai,proxy,rag ]
description: AI 代理插件配置参考
---

## 功能说明

`AI 代理`插件实现了基于 OpenAI API 契约的 AI 代理功能。目前支持 OpenAI、Azure OpenAI、月之暗面（Moonshot）和通义千问等 AI
服务提供商。

## 配置字段

### 基本配置

| 名称         | 数据类型   | 填写要求 | 默认值 | 描述               |
|------------|--------|------|-----|------------------|
| `provider` | object | 必填   | -   | 配置目标 AI 服务提供商的信息 |

`provider`的配置字段说明如下：

| 名称             | 数据类型                    | 填写要求 | 默认值 | 描述                                                                               |
|----------------|-------------------------|------|-----|----------------------------------------------------------------------------------|
| `type`         | string                  | 必填   | -   | AI 服务提供商名称。目前支持以下取值：openai, azure, moonshot, qwen                                |
| `apiTokens`    | array of string         | 必填   | -   | 用于在访问 AI 服务时进行认证的令牌。如果配置了多个 token，插件会在请求时随机进行选择。部分服务提供商只支持配置一个 token。            |
| `timeout`      | number                  | 非必填  | -   | 访问 AI 服务的超时时间。单位为毫秒。默认值为 120000，即 2 分钟                                           |
| `modelMapping` | map of string to string | 非必填  | -   | AI 模型映射表，用于将请求中的模型名称映射为服务提供商支持模型名称。<br/>可以使用 "*" 为键来配置通用兜底映射关系                   |
| `protocol`     | string                  | 非必填  | -   | 插件对外提供的 API 接口契约。目前支持以下取值：openai（默认值，使用 OpenAI 的接口契约）、original（使用目标服务提供商的原始接口契约） |
| `context`      | object                  | 非必填  | -   | 配置 AI 对话上下文信息                                                                    |

`context`的配置字段说明如下：

| 名称            | 数据类型   | 填写要求 | 默认值 | 描述                               |
|---------------|--------|------|-----|----------------------------------|
| `fileUrl`     | string | 必填   | -   | 保存 AI 对话上下文的文件 URL。仅支持纯文本类型的文件内容 |
| `serviceName` | string | 必填   | -   | URL 所对应的 Higress 后端服务完整名称        |
| `servicePort` | number | 必填   | -   | URL 所对应的 Higress 后端服务访问端口        |

### 提供商特有配置

#### OpenAI

OpenAI 所对应的 `type` 为 `openai`。它并无特有的配置字段。

#### Azure OpenAI

Azure OpenAI 所对应的 `type` 为 `azure`。它特有的配置字段如下：

| 名称                | 数据类型   | 填写要求 | 默认值 | 描述                                           |
|-------------------|--------|------|-----|----------------------------------------------|
| `azureServiceUrl` | string | 必填   | -   | Azure OpenAI 服务的 URL，须包含 `api-version` 查询参数。 |

**注意：** Azure OpenAI 只支持配置一个 API Token。

#### 月之暗面（Moonshot）

月之暗面所对应的 `type` 为 `moonshot`。它特有的配置字段如下：

| 名称               | 数据类型   | 填写要求 | 默认值 | 描述                                                          |
|------------------|--------|------|-----|-------------------------------------------------------------|
| `moonshotFileId` | string | 非必填  | -   | 通过文件接口上传至月之暗面的文件 ID，其内容将被用做 AI 对话的上下文。不可与 `context` 字段同时配置。 |

#### 通义千问（Qwen）

通义千问所对应的 `type` 为 `qwen`。它并无特有的配置字段。

## 用法示例

### 使用 OpenAI 协议代理 Azure OpenAI 服务

使用最基本的 Azure OpenAI 服务，不配置任何上下文。

**配置信息**

```yaml
provider:
  type: azure
  apiTokens:
    - "YOUR_AZURE_OPENAI_API_TOKEN"
  azureServiceUrl: "https://YOUR_RESOURCE_NAME.openai.azure.com/openai/deployments/YOUR_DEPLOYMENT_NAME/chat/completions?api-version=2024-02-15-preview",
```

**请求示例**

```json
{
  "model": "gpt-3",
  "messages": [
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ],
  "temperature": 0.3
}
```

**响应示例**

```json
{
  "choices": [
    {
      "content_filter_results": {
        "hate": {
          "filtered": false,
          "severity": "safe"
        },
        "self_harm": {
          "filtered": false,
          "severity": "safe"
        },
        "sexual": {
          "filtered": false,
          "severity": "safe"
        },
        "violence": {
          "filtered": false,
          "severity": "safe"
        }
      },
      "finish_reason": "stop",
      "index": 0,
      "logprobs": null,
      "message": {
        "content": "你好！我是一个AI助手，可以回答你的问题和提供帮助。有什么我可以帮到你的吗？",
        "role": "assistant"
      }
    }
  ],
  "created": 1714807624,
  "id": "chatcmpl-abcdefg1234567890",
  "model": "gpt-35-turbo-16k",
  "object": "chat.completion",
  "prompt_filter_results": [
    {
      "prompt_index": 0,
      "content_filter_results": {
        "hate": {
          "filtered": false,
          "severity": "safe"
        },
        "self_harm": {
          "filtered": false,
          "severity": "safe"
        },
        "sexual": {
          "filtered": false,
          "severity": "safe"
        },
        "violence": {
          "filtered": false,
          "severity": "safe"
        }
      }
    }
  ],
  "system_fingerprint": null,
  "usage": {
    "completion_tokens": 40,
    "prompt_tokens": 15,
    "total_tokens": 55
  }
}
```

### 使用 OpenAI 协议代理通义千问服务

使用通义千问服务，并配置从 OpenAI 大模型到通义千问的模型映射关系。

**配置信息**

```yaml
provider:
  type: qwen
  apiTokens:
    - "YOUR_QWEN_API_TOKEN"
  modelMapping:
    'gpt-3': "qwen-turbo"
    'gpt-35-turbo': "qwen-plus"
    'gpt-4-turbo': "qwen-max"
    '*': "qwen-turbo"
```

**请求示例**

```json
{
  "model": "gpt-3",
  "messages": [
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ],
  "temperature": 0.3
}
```

**响应示例**

```json
{
  "id": "c2518bd3-0f46-97d1-be34-bb5777cb3108",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "我是通义千问，由阿里云开发的AI助手。我可以回答各种问题、提供信息和与用户进行对话。有什么我可以帮助你的吗？"
      },
      "finish_reason": "stop"
    }
  ],
  "created": 1715175072,
  "model": "qwen-turbo",
  "object": "chat.completion",
  "usage": {
    "prompt_tokens": 24,
    "completion_tokens": 33,
    "total_tokens": 57
  }
}
```

### 使用通义千问配合纯文本上下文信息

使用通义千问服务，同时配置纯文本上下文信息。

**配置信息**

```yaml
provider:
  type: qwen
  apiTokens:
    - "YOUR_QWEN_API_TOKEN"
  modelMapping:
    "*": "qwen-turbo"
  context:
    - fileUrl: "http://file.default.svc.cluster.local/ai/context.txt",
      serviceName: "file.dns",
      servicePort: 80
```

**请求示例**

```json
{
  "model": "gpt-3",
  "messages": [
    {
      "role": "user",
      "content": "请概述文案内容"
    }
  ],
  "temperature": 0.3
}
```

**响应示例**

```json
{
  "id": "cmpl-77861a17681f4987ab8270dbf8001936",
  "object": "chat.completion",
  "created": 9756990,
  "model": "moonshot-v1-128k",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "这份文案是一份关于..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20181,
    "completion_tokens": 439,
    "total_tokens": 20620
  }
}
```

### 使用月之暗面配合其原生的文件上下文

提前上传文件至月之暗面，以文件内容作为上下文使用其 AI 服务。

**配置信息**

```yaml
provider:
  type: moonshot
  apiTokens:
    - "YOUR_MOONSHOT_API_TOKEN"
  moonshotFileId: "YOUR_MOONSHOT_FILE_ID",
  modelMapping:
    '*': "moonshot-v1-32k"
```

**请求示例**

```json
{
  "model": "gpt-4-turbo",
  "messages": [
    {
      "role": "user",
      "content": "请概述文案内容"
    }
  ],
  "temperature": 0.3
}
```

**响应示例**

```json
{
  "id": "cmpl-e5ca873642ca4f5d8b178c1742f9a8e8",
  "object": "chat.completion",
  "created": 1872961,
  "model": "moonshot-v1-128k",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "文案内容是关于一个名为“xxxx”的支付平台..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 11,
    "completion_tokens": 498,
    "total_tokens": 509
  }
}
```
