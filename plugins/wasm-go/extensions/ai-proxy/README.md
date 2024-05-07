---
title: AI 代理
keywords: [ higress,ai,proxy,rag ]
description: AI 代理插件配置参考
---

## 功能说明

`AI 代理`插件实现了基于 OpenAI API 契约的 AI 代理功能。目前支持 OpenAI、Azure OpenAI、月之暗面（Moonshot）和通义千问等 AI
服务提供商。

## 配置字段

### 全局配置

#### 基本配置

| 名称          | 数据类型            | 填写要求 | 默认值 | 描述               |
|-------------|-----------------|------|-----|------------------|
| `providers` | array of object | 必填   | -   | 配置目标 AI 服务提供商的信息 |
| `contexts`  | array of object | 非必填  | -   | 配置目标 AI 服务提供商的信息 |

`providers`中每一项的配置字段说明如下：

| 名称             | 数据类型                    | 填写要求 | 默认值 | 描述                                                             |
|----------------|-------------------------|------|-----|----------------------------------------------------------------|
| `id`           | string                  | 必填   | -   | AI 服务提供商的唯一标识                                                  |
| `type`         | string                  | 必填   | -   | AI 服务提供商名称。目前支持以下值：openai, azure, moonshot, qwen               |
| `token`        | string                  | 必填   | -   | 用于在访问 AI 服务时进行认证的令牌                                            |
| `timeout`      | number                  | 非必填  | -   | 访问 AI 服务的超时时间。单位为毫秒。默认值为 120000，即 2 分钟                         |
| `modelMapping` | map of string to string | 非必填  | -   | AI 模型映射表，用于将请求中的模型名称映射为服务提供商支持模型名称。<br/>可以使用 "*" 为键来配置通用兜底映射关系 |

`contexts`中每一项的配置字段说明如下：

| 名称            | 数据类型   | 填写要求 | 默认值 | 描述                               |
|---------------|--------|------|-----|----------------------------------|
| `id`          | string | 必填   | -   | 上下文信息的唯一标识                       |
| `fileUrl`     | string | 必填   | -   | 保存 AI 对话上下文的文件 URL。仅支持纯文本类型的文件内容 |
| `serviceName` | string | 必填   | -   | URL 所对应的 Higress 后端服务完整名称        |
| `servicePort` | number | 必填   | -   | URL 所对应的 Higress 后端服务访问端口        |

#### 提供商特有配置

##### OpenAI

OpenAI 所对应的 `type` 为 `openai`。它并无特有的配置字段。

##### Azure OpenAI

Azure OpenAI 所对应的 `type` 为 `azure`。它特有的配置字段如下：

| 名称                | 数据类型   | 填写要求 | 默认值 | 描述                                           |
|-------------------|--------|------|-----|----------------------------------------------|
| `azureServiceUrl` | string | 必填   | -   | Azure OpenAI 服务的 URL，须包含 `api-version` 查询参数。 |

##### 月之暗面（Moonshot）

月之暗面所对应的 `type` 为 `moonshot`。它特有的配置字段如下：

| 名称               | 数据类型   | 填写要求 | 默认值 | 描述                                                          |
|------------------|--------|------|-----|-------------------------------------------------------------|
| `moonshotFileId` | string | 非必填  | -   | 通过文件接口上传至月之暗面的文件 ID，其内容将被用做 AI 对话的上下文。不可与 `context` 字段同时配置。 |

##### 通义千问（Qwen）

通义千问所对应的 `type` 为 `qwen`。它并无特有的配置字段。

### 域名和路由级配置

| 名称                 | 数据类型   | 填写要求 | 默认值 | 描述               |
|--------------------|--------|------|-----|------------------|
| `activeProvider` | string | 必填   | -   | 当前启用的 AI 服务提供商标识 |
| `activeContext`  | string | 非必填  | -   | 当前启用的 AI 对话上下文标识 |

## 用法示例

### 使用基本的 Azure OpenAI 服务

使用最基本的 Azure OpenAI 服务，不配置任何上下文。

**配置信息**

```yaml
providers:
  - id: az
    type: azure
    apiToken: "YOUR_AZURE_OPENAI_API_TOKEN"
    azureServiceUrl: "https://higress-demo.openai.azure.com/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-15-preview",
activeProvider: az
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

### 使用通义千问配合纯文本上下文信息

使用通义千问服务，同时配置纯文本上下文信息。

**配置信息**

```yaml
providers:
  - id: qw
    type: qwen
    apiToken: "YOUR_QWEN_API_TOKEN"
    modelMapping:
      "*": "qwen-turbo"
contexts:
  - id: ctx
    fileUrl: "http://file.default.svc.cluster.local/ai/context.txt",
    serviceName: "file.dns",
    servicePort: 80
activeProvider: qw
activeContext: ctx
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
providers:
  - id: ms
    type: moonshot
    apiToken:
    moonshotFileId: "YOUR_MOONSHOT_FILE_ID",
    modelMapping:
      "*": "moonshot-v1-32k"
activeProvider: ms
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