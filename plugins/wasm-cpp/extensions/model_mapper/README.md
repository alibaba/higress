## 功能说明

`model-mapper`插件实现了基于LLM协议中的model参数路由的功能。

## 配置字段

| 名称                         | 数据类型            | 填写要求 | 默认值                      | 描述                                                                                                                                                           |
|----------------------------|-----------------|------|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `modelKey`                 | string          | 选填   | model                    | 请求body中model参数的位置                                                                                                                                            |
| `modelMapping`             | map of string   | 选填   | -                        | 默认 AI 模型映射表，用于将请求中的模型名称映射为服务提供商支持模型名称。<br/>1. 支持前缀匹配。例如用 "gpt-3-*" 匹配所有名称以“gpt-3-”开头的模型；<br/>2. 支持使用 "*" 为键来配置通用兜底映射关系；<br/>3. 如果映射的目标名称为空字符串 ""，则表示保留原模型名称。 |
| `conditionalModelMappings` | array of object | 选填   | -                        | 带有生效条件的 AI 模型映射表。按照配置顺序依次判断是否满足生效条件。若请求未匹配到任何带有生效条件的映射表，则生效默认 AI 模型映射表。                                                                                      |
| `enableOnPathSuffix`       | array of string | 选填   | ["/v1/chat/completions"] | 只对这些特定路径后缀的请求生效                                                                                                                                              |

`conditionalModelMappings`的配置字段说明如下：

| 名称             | 数据类型            | 填写要求 | 默认值 | 描述                                                                                                                                                        |
|----------------|-----------------|------|-----|-----------------------------------------------------------------------------------------------------------------------------------------------------------|
| `consumers`    | array of string | 必填   | -   | 生效本条规则的 consumer 名称。consumer 名称来自于前置认证插件（如 key-auth、basic-auth 等）的认证结果                                                                                    |
| `modelMapping` | map of string   | 选填   | -   | AI 模型映射表，用于将请求中的模型名称映射为服务提供商支持模型名称。<br/>1. 支持前缀匹配。例如用 "gpt-3-*" 匹配所有名称以“gpt-3-”开头的模型；<br/>2. 支持使用 "*" 为键来配置通用兜底映射关系；<br/>3. 如果映射的目标名称为空字符串 ""，则表示保留原模型名称。 |

## 效果说明

如下配置

```yaml
modelMapping:
  'gpt-4-*': "qwen-max"
  'gpt-4o': "qwen-vl-plus"
  '*': "qwen-turbo"
conditionalModelMappings:
  - consumers:
      - "consumer1"
    modelMapping:
      'qwen-*': "qwen-max"
      '*': "qwen-turbo"
```

开启后，默认情况下 `gpt-4-` 开头的模型参数会被改写为 `qwen-max`, `gpt-4o` 会被改写为 `qwen-vl-plus`，其他所有模型会被改写为
`qwen-turbo`。

例如原本的请求是：

```json
{
    "model": "gpt-4o",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "higress项目主仓库的github地址是什么"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
```


经过这个插件后，原始的 LLM 请求体将被改成：

```json
{
    "model": "qwen-vl-plus",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "higress项目主仓库的github地址是什么"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
```

如果前置认证插件认证结果表明请求来自 `consumer1`，则 `qwen-*` 开头的模型参数会被改写为 `qwen-max`，其他所有模型会被改写为 `qwen-turbo`。
