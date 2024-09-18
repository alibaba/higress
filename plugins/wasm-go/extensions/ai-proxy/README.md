---
title: AI 代理
keywords: [ AI网关, AI代理 ]
description: AI 代理插件配置参考
---

## 功能说明

`AI 代理`插件实现了基于 OpenAI API 契约的 AI 代理功能。目前支持 OpenAI、Azure OpenAI、月之暗面（Moonshot）和通义千问等 AI
服务提供商。

> **注意：**

> 请求路径后缀匹配 `/v1/chat/completions` 时，对应文生文场景，会用 OpenAI 的文生文协议解析请求 Body，再转换为对应 LLM 厂商的文生文协议

> 请求路径后缀匹配 `/v1/embeddings` 时，对应文本向量场景，会用 OpenAI 的文本向量协议解析请求 Body，再转换为对应 LLM 厂商的文本向量协议

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`100`


## 配置字段

### 基本配置

| 名称         | 数据类型   | 填写要求 | 默认值 | 描述               |
|------------|--------|------|-----|------------------|
| `provider` | object | 必填   | -   | 配置目标 AI 服务提供商的信息 |

`provider`的配置字段说明如下：

| 名称           | 数据类型        | 填写要求 | 默认值 | 描述                                                                                                                                                                                                                                                           |
| -------------- | --------------- | -------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------                                                                                                  |
| `type`         | string          | 必填     | -      | AI 服务提供商名称                                                                                                                                                                                                                                              |
| `apiTokens`    | array of string | 非必填   | -      | 用于在访问 AI 服务时进行认证的令牌。如果配置了多个 token，插件会在请求时随机进行选择。部分服务提供商只支持配置一个 token。                                                                                                                                     |
| `timeout`      | number          | 非必填   | -      | 访问 AI 服务的超时时间。单位为毫秒。默认值为 120000，即 2 分钟                                                                                                                                                                                                 |
| `modelMapping` | map of string   | 非必填   | -      | AI 模型映射表，用于将请求中的模型名称映射为服务提供商支持模型名称。<br/>1. 支持前缀匹配。例如用 "gpt-3-*" 匹配所有名称以“gpt-3-”开头的模型；<br/>2. 支持使用 "*" 为键来配置通用兜底映射关系；<br/>3. 如果映射的目标名称为空字符串 ""，则表示保留原模型名称。 |
| `protocol`     | string          | 非必填   | -      | 插件对外提供的 API 接口契约。目前支持以下取值：openai（默认值，使用 OpenAI 的接口契约）、original（使用目标服务提供商的原始接口契约）                                                                                                                          |
| `context`      | object          | 非必填   | -      | 配置 AI 对话上下文信息                                                                                                                                                                                                                                         |
| `customSettings` | array of customSetting | 非必填   | -      | 为AI请求指定覆盖或者填充参数                                                                                                                                                                                                                                 |

`context`的配置字段说明如下：

| 名称            | 数据类型   | 填写要求 | 默认值 | 描述                               |
|---------------|--------|------|-----|----------------------------------|
| `fileUrl`     | string | 必填   | -   | 保存 AI 对话上下文的文件 URL。仅支持纯文本类型的文件内容 |
| `serviceName` | string | 必填   | -   | URL 所对应的 Higress 后端服务完整名称        |
| `servicePort` | number | 必填   | -   | URL 所对应的 Higress 后端服务访问端口        |


`customSettings`的配置字段说明如下：

| 名称        | 数据类型              | 填写要求 | 默认值 | 描述                                                                                                                         |
| ----------- | --------------------- | -------- | ------ | ---------------------------------------------------------------------------------------------------------------------------- |
| `name`      | string                | 必填     | -      | 想要设置的参数的名称，例如`max_tokens`                                                                                       |
| `value`     | string/int/float/bool | 必填     | -      | 想要设置的参数的值，例如0                                                                                                    |
| `mode`      | string                | 非必填   | "auto" | 参数设置的模式，可以设置为"auto"或者"raw"，如果为"auto"则会自动根据协议对参数名做改写，如果为"raw"则不会有任何改写和限制检查 |
| `overwrite` | bool                  | 非必填   | true   | 如果为false则只在用户没有设置这个参数时填充参数，否则会直接覆盖用户原有的参数设置                                            |


custom-setting会遵循如下表格，根据`name`和协议来替换对应的字段，用户需要填写表格中`settingName`列中存在的值。例如用户将`name`设置为`max_tokens`，在openai协议中会替换`max_tokens`，在gemini中会替换`maxOutputTokens`。
`none`表示该协议不支持此参数。如果`name`不在此表格中或者对应协议不支持此参数，同时没有设置raw模式，则配置不会生效。


| settingName | openai      | baidu             | spark       | qwen        | gemini          | hunyuan     | claude      | minimax            |
| ----------- | ----------- | ----------------- | ----------- | ----------- | --------------- | ----------- | ----------- | ------------------ |
| max_tokens  | max_tokens  | max_output_tokens | max_tokens  | max_tokens  | maxOutputTokens | none        | max_tokens  | tokens_to_generate |
| temperature | temperature | temperature       | temperature | temperature | temperature     | Temperature | temperature | temperature        |
| top_p       | top_p       | top_p             | none        | top_p       | topP            | TopP        | top_p       | top_p              |
| top_k       | none        | none              | top_k       | none        | topK            | none        | top_k       | none               |
| seed        | seed        | none              | none        | seed        | none            | none        | none        | none               |

如果启用了raw模式，custom-setting会直接用输入的`name`和`value`去更改请求中的json内容，而不对参数名称做任何限制和修改。
对于大多数协议，custom-setting都会在json内容的根路径修改或者填充参数。对于`qwen`协议，ai-proxy会在json的`parameters`子路径下做配置。对于`gemini`协议，则会在`generation_config`子路径下做配置。


### 提供商特有配置

#### OpenAI

OpenAI 所对应的 `type` 为 `openai`。它特有的配置字段如下:

| 名称              | 数据类型 | 填写要求 | 默认值 | 描述                                                                          |
|-------------------|----------|----------|--------|-------------------------------------------------------------------------------|
| `openaiCustomUrl` | string   | 非必填   | -      | 基于OpenAI协议的自定义后端URL，例如: www.example.com/myai/v1/chat/completions |
| `responseJsonSchema` | object | 非必填 | - | 预先定义OpenAI响应需满足的Json Schema, 注意目前仅特定的几种模型支持该用法|


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

通义千问所对应的 `type` 为 `qwen`。它特有的配置字段如下：

| 名称                 | 数据类型            | 填写要求 | 默认值 | 描述                                                               |
|--------------------|-----------------|------|-----|------------------------------------------------------------------|
| `qwenEnableSearch` | boolean         | 非必填  | -   | 是否启用通义千问内置的互联网搜索功能。                          |
| `qwenFileIds`      | array of string | 非必填  | -   | 通过文件接口上传至Dashscope的文件 ID，其内容将被用做 AI 对话的上下文。不可与 `context` 字段同时配置。 |

#### 百川智能 (Baichuan AI)

百川智能所对应的 `type` 为 `baichuan` 。它并无特有的配置字段。

#### 零一万物（Yi）

零一万物所对应的 `type` 为 `yi`。它并无特有的配置字段。

#### 智谱AI（Zhipu AI）

智谱AI所对应的 `type` 为 `zhipuai`。它并无特有的配置字段。

#### DeepSeek（DeepSeek）

DeepSeek所对应的 `type` 为 `deepseek`。它并无特有的配置字段。

#### Groq

Groq 所对应的 `type` 为 `groq`。它并无特有的配置字段。

#### 文心一言（Baidu）

文心一言所对应的 `type` 为 `baidu`。它并无特有的配置字段。

#### 360智脑

360智脑所对应的 `type` 为 `ai360`。它并无特有的配置字段。

#### Mistral

Mistral 所对应的 `type` 为 `mistral`。它并无特有的配置字段。

#### MiniMax

MiniMax所对应的 `type` 为 `minimax`。它特有的配置字段如下：

| 名称             | 数据类型 | 填写要求                                                     | 默认值 | 描述                                                         |
| ---------------- | -------- | ------------------------------------------------------------ | ------ | ------------------------------------------------------------ |
| `minimaxGroupId` | string   | 当使用`abab6.5-chat`, `abab6.5s-chat`, `abab5.5s-chat`, `abab5.5-chat`四种模型时必填 | -      | 当使用`abab6.5-chat`, `abab6.5s-chat`, `abab5.5s-chat`, `abab5.5-chat`四种模型时会使用ChatCompletion Pro，需要设置groupID |

#### Anthropic Claude

Anthropic Claude 所对应的 `type` 为 `claude`。它特有的配置字段如下：

| 名称        | 数据类型   | 填写要求 | 默认值 | 描述                               |
|-----------|--------|------|-----|----------------------------------|
| `claudeVersion` | string | 可选   | -   | Claude 服务的 API 版本，默认为 2023-06-01 |

#### Ollama

Ollama 所对应的 `type` 为 `ollama`。它特有的配置字段如下：

| 名称                | 数据类型   | 填写要求 | 默认值 | 描述                                           |
|-------------------|--------|------|-----|----------------------------------------------|
| `ollamaServerHost` | string | 必填   | -   | Ollama 服务器的主机地址 |
| `ollamaServerPort` | number | 必填   | -   | Ollama 服务器的端口号，默认为11434 |

#### 混元

混元所对应的 `type` 为 `hunyuan`。它特有的配置字段如下：

| 名称                | 数据类型   | 填写要求 | 默认值 | 描述                                           |
|-------------------|--------|------|-----|----------------------------------------------|
| `hunyuanAuthId` | string | 必填   | -   | 混元用于v3版本认证的id |
| `hunyuanAuthKey` | string | 必填   | -   | 混元用于v3版本认证的key |

#### 阶跃星辰 (Stepfun)

阶跃星辰所对应的 `type` 为 `stepfun`。它并无特有的配置字段。

#### Cloudflare Workers AI

Cloudflare Workers AI 所对应的 `type` 为 `cloudflare`。它特有的配置字段如下：

| 名称                | 数据类型   | 填写要求 | 默认值 | 描述                                                                                                                         |
|-------------------|--------|------|-----|----------------------------------------------------------------------------------------------------------------------------|
| `cloudflareAccountId` | string | 必填   | -   | [Cloudflare Account ID](https://developers.cloudflare.com/workers-ai/get-started/rest-api/#1-get-api-token-and-account-id) |

#### 星火 (Spark)

星火所对应的 `type` 为 `spark`。它并无特有的配置字段。

讯飞星火认知大模型的`apiTokens`字段值为`APIKey:APISecret`。即填入自己的APIKey与APISecret，并以`:`分隔。

#### Gemini

Gemini 所对应的 `type` 为 `gemini`。它特有的配置字段如下：

| 名称                  | 数据类型 | 填写要求 | 默认值 | 描述                                                                                              |
| --------------------- | -------- | -------- |-----|-------------------------------------------------------------------------------------------------|
| `geminiSafetySetting` | map of string   | 非必填     | -   | Gemini AI内容过滤和安全级别设定。参考[Safety settings](https://ai.google.dev/gemini-api/docs/safety-settings) |

#### DeepL

DeepL 所对应的 `type` 为 `deepl`。它特有的配置字段如下：

| 名称         | 数据类型 | 填写要求 | 默认值 | 描述                         |
| ------------ | -------- | -------- | ------ | ---------------------------- |
| `targetLang` | string   | 必填     | -      | DeepL 翻译服务需要的目标语种 |

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
    'gpt-4-*': "qwen-max"
    'gpt-4o': "qwen-vl-plus"
    'text-embedding-v1': 'text-embedding-v1'
    '*': "qwen-turbo"
```

**AI 对话请求示例**

URL: http://your-domain/v1/chat/completions

请求示例：

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

响应示例：

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

**多模态模型 API 请求示例（适用于 `qwen-vl-plus` 和 `qwen-vl-max` 模型）**

URL: http://your-domain/v1/chat/completions

请求示例：

```json
{
    "model": "gpt-4o",
    "messages": [
        {
            "role": "user",
            "content": [
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "https://dashscope.oss-cn-beijing.aliyuncs.com/images/dog_and_girl.jpeg"
                    }
                },
                {
                    "type": "text",
                    "text": "这个图片是哪里？"
                }
            ]
        }
    ],
    "temperature": 0.3
}
```

响应示例：

```json
{
    "id": "17c5955d-af9c-9f28-bbde-293a9c9a3515",
    "choices": [
        {
            "index": 0,
            "message": {
                "role": "assistant",
                "content": [
                    {
                        "text": "这张照片显示的是一位女士和一只狗在海滩上。由于我无法获取具体的地理位置信息，所以不能确定这是哪个地方的海滩。但是从视觉内容来看，它可能是一个位于沿海地区的沙滩海岸线，并且有海浪拍打着岸边。这样的场景在全球许多美丽的海滨地区都可以找到。如果您需要更精确的信息，请提供更多的背景或细节描述。"
                    }
                ]
            },
            "finish_reason": "stop"
        }
    ],
    "created": 1723949230,
    "model": "qwen-vl-plus",
    "object": "chat.completion",
    "usage": {
        "prompt_tokens": 1279,
        "completion_tokens": 78
    }
}
```

**文本向量请求示例**

URL: http://your-domain/v1/embeddings

请求示例：

```json
{
  "model": "text-embedding-v1",
  "input": "Hello"
}
```

响应示例：

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [
        -1.0437825918197632,
        5.208984375,
        3.0483806133270264,
        -1.7897135019302368,
        -2.0107421875,
        ...,
        0.8125,
        -1.1759847402572632,
        0.8174641728401184,
        1.0432943105697632,
        -0.5885213017463684
      ]
    }
  ],
  "model": "text-embedding-v1",
  "usage": {
    "prompt_tokens": 1,
    "total_tokens": 1
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

### 使用通义千问配合其原生的文件上下文

提前上传文件至通义千问，以文件内容作为上下文使用其 AI 服务。

**配置信息**

```yaml
provider:
  type: qwen
  apiTokens:
    - "YOUR_QWEN_API_TOKEN"
  modelMapping:
    "*": "qwen-long" # 通义千问的文件上下文只能在 qwen-long 模型下使用
  qwenFileIds:
  - "file-fe-xxx"
  - "file-fe-yyy"
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
  "output": {
    "choices": [
      {
        "finish_reason": "stop",
        "message": {
          "role": "assistant",
          "content": "您上传了两个文件，`context.txt` 和 `context_2.txt`，它们似乎都包含了关于xxxx"
        }
      }
    ]
  },
  "usage": {
    "total_tokens": 2023,
    "output_tokens": 530,
    "input_tokens": 1493
  },
  "request_id": "187e99ba-5b64-9ffe-8f69-01dafbaf6ed7"
}
```

### 使用original协议代理百炼智能体应用
**配置信息**

```yaml
provider:
  type: qwen
  apiTokens:
    - "YOUR_DASHSCOPE_API_TOKEN"
  protocol: original
```

**请求实例**
```json
{
  "input": {
      "prompt": "介绍一下Dubbo"
  },
  "parameters":  {},
  "debug": {}
}
```

**响应实例**

```json
{
    "output": {
        "finish_reason": "stop",
        "session_id": "677e7e8fbb874e1b84792b65042e1599",
        "text": "Apache Dubbo 是一个..."
    },
    "usage": {
        "models": [
            {
                "output_tokens": 449,
                "model_id": "qwen-max",
                "input_tokens": 282
            }
        ]
    },
    "request_id": "b59e45e3-5af4-91df-b7c6-9d746fd3297c"
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

### 使用 OpenAI 协议代理 Groq 服务

**配置信息**

```yaml
provider:
  type: groq
  apiTokens:
    - "YOUR_GROQ_API_TOKEN"
```

**请求示例**

```json
{
  "model": "llama3-8b-8192",
  "messages": [
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ]
}
```

**响应示例**

```json
{
  "id": "chatcmpl-26733989-6c52-4056-b7a9-5da791bd7102",
  "object": "chat.completion",
  "created": 1715917967,
  "model": "llama3-8b-8192",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "😊 Ni Hao! (That's \"hello\" in Chinese!)\n\nI am LLaMA, an AI assistant developed by Meta AI that can understand and respond to human input in a conversational manner. I'm not a human, but a computer program designed to simulate conversations and answer questions to the best of my ability. I'm happy to chat with you in Chinese or help with any questions or topics you'd like to discuss! 😊"
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 16,
    "prompt_time": 0.005,
    "completion_tokens": 89,
    "completion_time": 0.104,
    "total_tokens": 105,
    "total_time": 0.109
  },
  "system_fingerprint": "fp_dadc9d6142",
  "x_groq": {
    "id": "req_01hy2awmcxfpwbq56qh6svm7qz"
  }
}
```

### 使用 OpenAI 协议代理 Claude 服务

**配置信息**

```yaml
provider:
  type: claude
  apiTokens:
    - "YOUR_CLAUDE_API_TOKEN"
  version: "2023-06-01"
```

**请求示例**

```json
{
  "model": "claude-3-opus-20240229",
  "max_tokens": 1024,
  "messages": [
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ]
}
```

**响应示例**

```json
{
  "id": "msg_01Jt3GzyjuzymnxmZERJguLK",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "您好,我是一个由人工智能公司Anthropic开发的聊天助手。我的名字叫Claude,是一个聪明友善、知识渊博的对话系统。很高兴认识您!我可以就各种话题与您聊天,回答问题,提供建议和帮助。我会尽最大努力给您有帮助的回复。希望我们能有个愉快的交流!"
      },
      "finish_reason": "stop"
    }
  ],
  "created": 1717385918,
  "model": "claude-3-opus-20240229",
  "object": "chat.completion",
  "usage": {
    "prompt_tokens": 16,
    "completion_tokens": 126,
    "total_tokens": 142
  }
}
```
### 使用 OpenAI 协议代理混元服务

**配置信息**

```yaml
provider:
  type: "hunyuan"
  hunyuanAuthKey: "<YOUR AUTH KEY>"
  apiTokens:
    - ""
  hunyuanAuthId: "<YOUR AUTH ID>"
  timeout: 1200000
  modelMapping:
    "*": "hunyuan-lite"
```

**请求示例**
请求脚本：
```sh

curl --location 'http://<your higress domain>/v1/chat/completions' \
--header 'Content-Type:  application/json' \
--data '{
  "model": "gpt-3",
  "messages": [
    {
      "role": "system",
      "content": "你是一个名专业的开发人员！"
    },
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ],
  "temperature": 0.3,
  "stream": false
}'
```

**响应示例**

```json
{
    "id": "fd140c3e-0b69-4b19-849b-d354d32a6162",
    "choices": [
        {
            "index": 0,
            "delta": {
                "role": "assistant",
                "content": "你好！我是一名专业的开发人员。"
            },
            "finish_reason": "stop"
        }
    ],
    "created": 1717493117,
    "model": "hunyuan-lite",
    "object": "chat.completion",
    "usage": {
        "prompt_tokens": 15,
        "completion_tokens": 9,
        "total_tokens": 24
    }
}
```

### 使用 OpenAI 协议代理百度文心一言服务

**配置信息**

```yaml
provider:
  type: baidu
  apiTokens:
    - "YOUR_BAIDU_API_TOKEN"
  modelMapping:
    'gpt-3': "ERNIE-4.0"
    '*': "ERNIE-4.0"
```

**请求示例**

```json
{
    "model": "gpt-4-turbo",
    "messages": [
        {
            "role": "user",
            "content": "你好，你是谁？"
        }
    ],
    "stream": false
}
```

**响应示例**

```json
{
    "id": "as-e90yfg1pk1",
    "choices": [
        {
            "index": 0,
            "message": {
                "role": "assistant",
                "content": "你好，我是文心一言，英文名是ERNIE Bot。我能够与人对话互动，回答问题，协助创作，高效便捷地帮助人们获取信息、知识和灵感。"
            },
            "finish_reason": "stop"
        }
    ],
    "created": 1717251488,
    "model": "ERNIE-4.0",
    "object": "chat.completion",
    "usage": {
        "prompt_tokens": 4,
        "completion_tokens": 33,
        "total_tokens": 37
    }
}
```

### 使用 OpenAI 协议代理MiniMax服务

**配置信息**

```yaml
provider:
  type: minimax
  apiTokens:
    - "YOUR_MINIMAX_API_TOKEN"
  modelMapping:
    "gpt-3": "abab6.5g-chat"
    "gpt-4": "abab6.5-chat"
    "*": "abab6.5g-chat"
  minimaxGroupId: "YOUR_MINIMAX_GROUP_ID"
```

**请求示例**

```json
{
    "model": "gpt-4-turbo",
    "messages": [
        {
            "role": "user",
            "content": "你好，你是谁？"
        }
    ],
    "stream": false
}
```

**响应示例**

```json
{
    "id": "02b2251f8c6c09d68c1743f07c72afd7",
    "choices": [
        {
            "finish_reason": "stop",
            "index": 0,
            "message": {
                "content": "你好！我是MM智能助理，一款由MiniMax自研的大型语言模型。我可以帮助你解答问题，提供信息，进行对话等。有什么可以帮助你的吗？",
                "role": "assistant"
            }
        }
    ],
    "created": 1717760544,
    "model": "abab6.5s-chat",
    "object": "chat.completion",
    "usage": {
        "total_tokens": 106
    },
    "input_sensitive": false,
    "output_sensitive": false,
    "input_sensitive_type": 0,
    "output_sensitive_type": 0,
    "base_resp": {
        "status_code": 0,
        "status_msg": ""
    }
}
```

### 使用 OpenAI 协议代理360智脑服务

**配置信息**

```yaml
provider:
  type: ai360
  apiTokens:
    - "YOUR_MINIMAX_API_TOKEN"
  modelMapping:
    "gpt-4o": "360gpt-turbo-responsibility-8k"
    "gpt-4": "360gpt2-pro"
    "gpt-3.5": "360gpt-turbo"
    "text-embedding-3-small": "embedding_s1_v1.2"
    "*": "360gpt-pro"
```

**请求示例**

```json
{
  "model": "gpt-4o",
  "messages": [
    {
      "role": "system",
      "content": "你是一个专业的开发人员！"
    },
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ]
}
```

**响应示例**

```json
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "你好，我是360智脑，一个大型语言模型。我可以帮助回答各种问题、提供信息、进行对话等。有什么可以帮助你的吗？"
      },
      "finish_reason": "",
      "index": 0
    }
  ],
  "created": 1724257207,
  "id": "5e5c94a2-d989-40b5-9965-5b971db941fe",
  "model": "360gpt-turbo",
  "object": "",
  "usage": {
    "completion_tokens": 33,
    "prompt_tokens": 24,
    "total_tokens": 57
  },
  "messages": [
    {
      "role": "system",
      "content": "你是一个专业的开发人员！"
    },
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ],
  "context": null
}
```

**文本向量请求示例**

URL: http://your-domain/v1/embeddings

请求示例：

```json
{
  "input":["你好"],
  "model":"text-embedding-3-small"
}
```

响应示例：

```json
{
  "data": [
    {
      "embedding": [
        -0.011237,
        -0.015433,
        ...,
        -0.028946,
        -0.052778,
        0.003768,
        -0.007917,
        -0.042201
      ],
      "index": 0,
      "object": ""
    }
  ],
  "model": "embedding_s1_v1.2",
  "object": "",
  "usage": {
    "prompt_tokens": 2,
    "total_tokens": 2
  }
}
```

### 使用 OpenAI 协议代理 Cloudflare Workers AI 服务

**配置信息**

```yaml
provider:
  type: cloudflare
  apiTokens:
    - "YOUR_WORKERS_AI_API_TOKEN"
  cloudflareAccountId: "YOUR_CLOUDFLARE_ACCOUNT_ID"
  modelMapping:
    "*": "@cf/meta/llama-3-8b-instruct"
```

**请求示例**

```json
{
  "model": "gpt-3.5",
  "max_tokens": 1024,
  "messages": [
    {
      "role": "user",
      "content": "Who are you?"
    }
  ]
}
```

**响应示例**

```json
{
  "id": "id-1720367803430",
  "object": "chat.completion",
  "created": 1720367803,
  "model": "@cf/meta/llama-3-8b-instruct",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "I am LLaMA, an AI assistant developed by Meta AI that can understand and respond to human input in a conversational manner. I'm not a human, but a computer program designed to simulate conversation and answer questions to the best of my knowledge. I can be used to generate text on a wide range of topics, from science and history to entertainment and culture.\n\nI'm a large language model, which means I've been trained on a massive dataset of text from the internet and can generate human-like responses. I can understand natural language and respond accordingly, making me suitable for tasks such as:\n\n* Answering questions on various topics\n* Generating text based on a given prompt\n* Translating text from one language to another\n* Summarizing long pieces of text\n* Creating chatbot dialogues\n\nI'm constantly learning and improving, so the more conversations I have with users like you, the better I'll become."
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ]
}
```

### 使用 OpenAI 协议代理Spark服务

**配置信息**

```yaml
provider:
  type: spark
  apiTokens:
    - "APIKey:APISecret"
  modelMapping:
    "gpt-4o": "generalv3.5"
    "gpt-4": "generalv3"
    "*": "general"
```

**请求示例**

```json
{
    "model": "gpt-4o",
    "messages": [
        {
            "role": "system",
            "content": "你是一名专业的开发人员！"
        },
        {
            "role": "user",
            "content": "你好，你是谁？"
        }
    ],
    "stream": false
}
```

**响应示例**

```json
{
    "id": "cha000c23c6@dx190ef0b4b96b8f2532",
    "choices": [
        {
            "index": 0,
            "message": {
                "role": "assistant",
                "content": "你好！我是一名专业的开发人员，擅长编程和解决技术问题。有什么我可以帮助你的吗？"
            }
        }
    ],
    "created": 1721997415,
    "model": "generalv3.5",
    "object": "chat.completion",
    "usage": {
        "prompt_tokens": 10,
        "completion_tokens": 19,
        "total_tokens": 29
    }
}
```

### 使用 OpenAI 协议代理 gemini 服务

**配置信息**

```yaml
provider:
  type: gemini
  apiTokens:
    - "YOUR_GEMINI_API_TOKEN"
  modelMapping:
    "*": "gemini-pro"
  geminiSafetySetting:
    "HARM_CATEGORY_SEXUALLY_EXPLICIT" :"BLOCK_NONE"
    "HARM_CATEGORY_HATE_SPEECH" :"BLOCK_NONE"
    "HARM_CATEGORY_HARASSMENT" :"BLOCK_NONE"
    "HARM_CATEGORY_DANGEROUS_CONTENT" :"BLOCK_NONE"
```

**请求示例**

```json
{
    "model": "gpt-3.5",
    "messages": [
        {
            "role": "user",
            "content": "Who are you?"
        }
    ],
    "stream": false
}
```

**响应示例**

```json
{
    "id": "chatcmpl-b010867c-0d3f-40ba-95fd-4e8030551aeb",
    "choices": [
        {
            "index": 0,
            "message": {
                "role": "assistant",
                "content": "I am a large multi-modal model, trained by Google. I am designed to provide information and answer questions to the best of my abilities."
            },
            "finish_reason": "stop"
        }
    ],
    "created": 1722756984,
    "model": "gemini-pro",
    "object": "chat.completion",
    "usage": {
        "prompt_tokens": 5,
        "completion_tokens": 29,
        "total_tokens": 34
    }
}
```

### 使用 OpenAI 协议代理 DeepL 文本翻译服务

**配置信息**

```yaml
provider:
  type: deepl
  apiTokens:
    - "YOUR_DEEPL_API_TOKEN"
  targetLang: "ZH"
```

**请求示例**
此处 `model` 表示 DeepL 的服务类型，只能填 `Free` 或 `Pro`。`content` 中设置需要翻译的文本；在 `role: system` 的 `content` 中可以包含可能影响翻译但本身不会被翻译的上下文，例如翻译产品名称时，可以将产品描述作为上下文传递，这种额外的上下文可能会提高翻译的质量。

```json
{
  "model": "Free",
  "messages": [
    {
      "role": "system",
      "content": "money"
    },
    {
      "content": "sit by the bank"
    },
    {
      "content": "a bank in China"
    }
  ]
}
```

**响应示例**
```json
{
  "choices": [
    {
      "index": 0,
      "message": { "name": "EN", "role": "assistant", "content": "坐庄" }
    },
    {
      "index": 1,
      "message": { "name": "EN", "role": "assistant", "content": "中国银行" }
    }
  ],
  "created": 1722747752,
  "model": "Free",
  "object": "chat.completion",
  "usage": {}
}
```

## 完整配置示例

### Kubernetes 示例

以下以使用 OpenAI 协议代理 Groq 服务为例，展示完整的插件配置示例。

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-proxy-groq
  namespace: higress-system
spec:
  matchRules:
  - config:
      provider:
        type: groq
        apiTokens: 
          - "YOUR_API_TOKEN"
    ingress:
    - groq
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-proxy:1.0.0
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    higress.io/backend-protocol: HTTPS
    higress.io/destination: groq.dns
    higress.io/proxy-ssl-name: api.groq.com
    higress.io/proxy-ssl-server-name: "on"
  labels:
    higress.io/resource-definer: higress
  name: groq
  namespace: higress-system
spec:
  ingressClassName: higress
  rules:
  - host: <YOUR-DOMAIN> 
    http:
      paths:
      - backend:
          resource:
            apiGroup: networking.higress.io
            kind: McpBridge
            name: default
        path: /
        pathType: Prefix
---
apiVersion: networking.higress.io/v1
kind: McpBridge
metadata:
  name: default
  namespace: higress-system
spec:
  registries:
  - domain: api.groq.com
    name: groq
    port: 443
    type: dns
```

访问示例：

```bash
curl "http://<YOUR-DOMAIN>/v1/chat/completions" -H "Content-Type: application/json" -d '{
  "model": "llama3-8b-8192",
  "messages": [
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ]
}'
```

### Docker-Compose 示例

`docker-compose.yml` 配置文件：

```yaml
version: '3.7'
services:
  envoy:
    image: higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/envoy:1.20
    entrypoint: /usr/local/bin/envoy
    # 开启了 debug 级别日志方便调试
    command: -c /etc/envoy/envoy.yaml --component-log-level wasm:debug
    networks:
      - higress-net
    ports:
      - "10000:10000"
    volumes:
      - ./envoy.yaml:/etc/envoy/envoy.yaml
      - ./plugin.wasm:/etc/envoy/plugin.wasm
networks:
  higress-net: {}
```

`envoy.yaml` 配置文件：

```yaml
admin:
  address:
    socket_address:
      protocol: TCP
      address: 0.0.0.0
      port_value: 9901
static_resources:
  listeners:
    - name: listener_0
      address:
        socket_address:
          protocol: TCP
          address: 0.0.0.0
          port_value: 10000
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                scheme_header_transformation:
                  scheme_to_overwrite: https
                stat_prefix: ingress_http
                # Output envoy logs to stdout
                access_log:
                  - name: envoy.access_loggers.stdout
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
                # Modify as required
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: local_service
                      domains: [ "*" ]
                      routes:
                        - match:
                            prefix: "/"
                          route:
                            cluster: claude
                            timeout: 300s
                http_filters:
                  - name: claude
                    typed_config:
                      "@type": type.googleapis.com/udpa.type.v1.TypedStruct
                      type_url: type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
                      value:
                        config:
                          name: claude
                          vm_config:
                            runtime: envoy.wasm.runtime.v8
                            code:
                              local:
                                filename: /etc/envoy/plugin.wasm
                          configuration:
                            "@type": "type.googleapis.com/google.protobuf.StringValue"
                            value: | # 插件配置
                              {
                                "provider": {
                                  "type": "claude",                                
                                  "apiTokens": [
                                    "YOUR_API_TOKEN"
                                  ]                  
                                }
                              }
                  - name: envoy.filters.http.router
  clusters:
    - name: claude
      connect_timeout: 30s
      type: LOGICAL_DNS
      dns_lookup_family: V4_ONLY
      lb_policy: ROUND_ROBIN
      load_assignment:
        cluster_name: claude
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: api.anthropic.com # API 服务地址
                      port_value: 443
      transport_socket:
        name: envoy.transport_sockets.tls
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
          "sni": "api.anthropic.com"
```

访问示例：

```bash
curl "http://localhost:10000/v1/chat/completions"  -H "Content-Type: application/json"  -d '{
  "model": "claude-3-opus-20240229",
  "max_tokens": 1024,
  "messages": [
    {
      "role": "user",
      "content": "你好，你是谁？"
    }
  ]
}'
```
