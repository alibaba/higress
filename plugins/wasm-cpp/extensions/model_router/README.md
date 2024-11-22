## 功能说明
`model-router`插件实现了基于LLM协议中的model参数路由的功能

## 配置字段

| 名称                 | 数据类型        | 填写要求                | 默认值                   | 描述                                                  |
| -----------          | --------------- | ----------------------- | ------                   | -------------------------------------------           |
| `modelKey`           | string          | 选填                    | model                    | 请求body中model参数的位置                             |
| `addProviderHeader`  | string          | 选填                    | -                        | 从model参数中解析出的provider名字放到哪个请求header中 |
| `modelToHeader`      | string          | 选填                    | -                        | 直接将model参数放到哪个请求header中                   |
| `enableOnPathSuffix` | array of string | 选填                    | ["/v1/chat/completions"] | 只对这些特定路径后缀的请求生效                        |

## 运行属性

插件执行阶段：认证阶段
插件执行优先级：900

## 效果说明

### 基于 model 参数进行路由

需要做如下配置：

```yaml
modelToHeader: x-higress-llm-model
```

插件会将请求中 model 参数提取出来，设置到 x-higress-llm-model 这个请求 header 中，用于后续路由，举例来说，原生的 LLM 请求体是：

```json
{
    "model": "qwen-long",
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

经过这个插件后，将添加下面这个请求头(可以用于路由匹配)：

x-higress-llm-model: qwen-long

### 提取 model 参数中的 provider 字段用于路由

> 注意这种模式需要客户端在 model 参数中通过`/`分隔的方式，来指定 provider

需要做如下配置：

```yaml
addProviderHeader: x-higress-llm-provider
```

插件会将请求中 model 参数的 provider 部分（如果有）提取出来，设置到 x-higress-llm-provider 这个请求 header 中，用于后续路由，并将 model 参数重写为模型名称部分。举例来说，原生的 LLM 请求体是：

```json
{
    "model": "dashscope/qwen-long",
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

经过这个插件后，将添加下面这个请求头(可以用于路由匹配)：

x-higress-llm-provider: dashscope

原始的 LLM 请求体将被改成：

```json
{
    "model": "qwen-long",
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
