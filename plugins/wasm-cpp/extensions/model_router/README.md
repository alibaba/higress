## 功能说明
`model-router`插件实现了基于LLM协议中的model参数路由的功能

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`260`

## 配置字段

| 名称             | 数据类型        | 填写要求                | 默认值                 | 描述                                                  |
| -----------      | --------------- | ----------------------- | ------                 | -------------------------------------------           |
| `enable`         | bool            | 选填                    | false                  | 是否开启基于model参数路由                             |
| `model_key`      | string          | 选填                    | model                  | 请求body中model参数的位置                             |
| `add_header_key` | string          | 选填                    | x-higress-llm-provider | 从model参数中解析出的provider名字放到哪个请求header中 |


## 效果说明

如下开启基于model参数路由的功能：

```yaml
enable: true
```

开启后，插件将请求中 model 参数的 provider 部分（如果有）提取出来，设置到 x-higress-llm-provider 这个请求 header 中，用于后续路由，并将 model 参数重写为模型名称部分。举例来说，原生的 LLM 请求体是：

```json
{
    "model": "qwen/qwen-long",
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

x-higress-llm-provider: qwen

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
