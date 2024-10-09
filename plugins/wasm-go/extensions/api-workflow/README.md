---
title: API 工作流
keywords: [ API工作流 ]
description: API 工作流插件配置参考
---
## 功能说明
`api工作流 `实现了可编排的API workflow 插件，支持根据配置定义生成DAG并执行工作流

## 配置说明

| 名称       | 数据类型   | 填写要求 | 默认值 | 描述     | 备注 |
|----------|--------|------| --- |--------|----|
| workflow | object | 必填   |     | DAG的定义 |    |
| env      | object | 选填   |     | 一些环境变量 |    |

`env`object的配置字段说明如下：

| 名称       | 数据类型   | 填写要求 | 默认值  | 描述        | 备注 |
|----------|--------|------|------|-----------|--|
| timeout | int    | 选填   | 5000 | 每次请求的过期时间 | 单位是毫秒(ms) |
| max_depth | int    | 选填   | 100  | 工作流最大迭代次数 |  |


`workflow`object的配置字段说明如下：

| 名称    | 数据类型                 | 填写要求 | 默认值 | 描述        | 备注 |
|-------|----------------------| ---- | --- |-----------|----|
| nodes | array of node object | 选填   |     | DAG的定义的节点 |    |
| edges | array of edge object | 必填   |     | DAG的定义的边  |    |

`edge` object的配置字段说明如下：

| 名称          | 数据类型   | 填写要求 | 默认值 | 描述                                             |
|-------------| ------ | ---- | --- |------------------------------------------------|
| source      | string | 必填   | -   | 上一步的操作，必须是定义的node的name，或者初始化工作流的start          |
| target      | string | 必填   | -   | 当前的操作，必须是定义的node的name，或者结束工作流的关键字 end continue | |
| conditional | string | 选填   | -   | 这一步是否执行的判断条件                                   |

`node` object的配置字段说明如下：

| 名称              | 数据类型                               | 填写要求 | 默认值 | 描述                            | 备注                            |
| --------------- |------------------------------------|---| --- |-------------------------------|-------------------------------|
| name            | string                             | 必填 | -   | node名称                        | 全局唯一                          |
| service_name    | string                             | 必填 | -   | higress配置的服务名称                |                               |
| service_port    | int                                | 选填 | 80  | higress配置的服务端口                |                               |
| service_domain  | string                             | 选填 |     | higress配置的服务domain            |                               |
| service_path    | string                             | 必填 |     | 请求的path                       |                               |
| service_headers | array of header object             | 选填 |     | 请求的头                          |                               |
| service_body_replace_keys| array of bodyReplaceKeyPair object | 选填|   请求body模板替换键值对  | 用来构造请求| 如果为空，则直接使用service_body_tmpl请求 |
| service_body_tmpl   | string                             | 选填 |     | 请求的body模板                     |                               |
| service_method  | string                             | 必填 |     | 请求的方法                         | GET，POST                      |

`header` object 的配置字段说明如下：

| 名称    | 数据类型                   | 填写要求 | 默认值 | 描述        | 备注        |
|-------|------------------------|---| --- |-----------| --------- |
| key   | string                 | 必填 | -   | 头文件的key   |           |
| value | string                 | 必填 | -   | 头文件的value |           |

`bodyReplaceKeyPair` object 配置说明

| 名称   | 数据类型                   | 填写要求 | 默认值 | 描述        | 备注 |
|------|------------------------|---| --- |-----------|--|
| from | string                 | 必填 | -   | 描述数据从哪获得  |  |
| to   | string                 | 必填 | -   | 描述数据最后放到那 |  |



## 用法示例

我们把工作流抽象成DAG配置文件，加上控制流和数据流更方便的控制流程和构造请求。

![img](img/img.png)



### DAG的定义

#### 边edge
描述操作如何编排

样例
```yaml
  edges:
    - source: start
      target: A
    - source: start
      target: B
    - source: start
      target: C
    - source: A
      target: D
    - source: B
      target: D
    - source: C
      target: D
    - source: D
      target: end
      conditional: "gt {{D||check}} 0.9"
    - source: D
      target: E
      conditional: "lt {{D||check}} 0.9"
    - source: E
      target: end
```
#### 控制流 conditional 和 target
##### 分支 conditional
插件执行到conditional的定义不为空的步骤`edge`时，会根据表达式定义判断这步是否执行，如果判断为否，会跳过这个分支。
表达式可使用参数，用{{xxx}}标注，具体定义见数据流`模板和变量`
支持比较表达式和例子如下：
`eq arg1 arg2`： arg1 == arg2时为true 不只是数字，支持string
`lt arg1 arg2`： arg1 < arg2时为true 
`le arg1 arg2`： arg1 <= arg2时为true 
`gt arg1 arg2`： arg1 > arg2时为true 
`ge arg1 arg2`： arg1 >= arg2时为true
`and arg1 arg2`: arg1 && arg2
`or arg1 arg2`: arg1 || arg2
`contain arg1 arg2`: arg1 包含 arg2时为true
支持and 和 or的嵌套 比如 `and (eq 1 1) (or (contain hello hi) (lt 1 2))`

##### 结束和执行工作流 target
当target为`name`,执行name的操作
当target 为`end`，直接返回source的结果，结束工作流
当target 为`continue`，结束工作流，将请求放行到下一个plugin

#### 数据流

进入plugin的数据（request body）,会根据构造模板json`node.service_body_tmpl`和`node.service_body_replace_keys`构造请求body，并把执行后结果存在key为`nodeName`的上下文里，只支持json格式的数据。

##### 模板和变量
在工作流的配置文件中
###### edge.conditional
配置文件的定义中，`edge.conditional`  支持模板和变量，方便根据数据流的数据来构建请求数据
在模板里使用变量来代表数据和过滤。变量使用`{{str1||str2}}`包裹，使用`||`分隔，str1代表使用那个node的输出数据，str2代表如何取数据，过滤表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串，`@all`代表全都要

例子
```yaml
conditional: "lt {{D||check}} 0.9"
```
node D 的返回值是
```json
{"check": 0.99}
```
解析后的表达式 `lt 0.99 0.9`

###### node.service_body_tmpl 和 node.service_body_replace_keys
这组配置用来构造请求body，`node.service_body_tmpl`是模板json ，`node.service_body_replace_keys`用来描述如何填充模板json，是一个object的数组，from标识数据从哪里来，to表示填充的位置
`from`是使用`str1||str2`的字符串，str1代表使用那个node的执行返回数据，str2代表如何取数据，表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
`to`标识数据放哪,表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法来描述填充位置，使用的是sjson来拼接json，填充到`tool.service_body_tmpl` 的模板json里
当`node.service_body_replace_keys`为空时，代表直接发送`node.service_body_tmpl`

例子
```yaml
    service_body_tmpl:
      embeddings: 
        result: ""
      msg: ""
      sk: "sk-xxxxxx"
    service_body_replace_keys:
      - to "embeddings.result"
        from "A||output.embeddings.0.embedding"
      - to "msg"
        from "B||@all"
```
`A`节点的输出是
```json
{"embeddings":  {"output":{"embeddings":[{"embedding":[0.014398524595686043],"text_index":0}]},"usage":{"total_tokens":12},"request_id":"2a5229bc-53d9-91ca-bce2-00ae5e01a1d3"}}
```
`B`节点的输出是
```json
["higress项目主仓库的github地址是什么"]
```
根据 service_body_tmpl 和 service_body_replace_keys 构造的request body如下
```json
{"embeddings":{"result":"[0.014398524595686043，......]"},"msg":["higress项目主仓库的github地址是什么"],"sk":"sk-xxxxxx"}
```



### node的定义

具体执行的单元，封装了httpCall，提供http的访问能力，获取各种api的能力。request body支持自主构建。

样例
```yaml
  nodes:
    - name: "A"
      service_domain: "dashscope.aliyuncs.com"
      service_name: "dashscope"
      service_port: 443
      service_path: "/api/v1/services/embeddings/text-embedding/text-embedding"
      service_method: "POST"
      service_body_tmpl:
        model: "text-embedding-v2"
        input:
          texts: ""
        parameters:
          text_type: "query"
      service_body_replace_keys:
        - from: "start||messages.#(role==user)#.content"
          to: "input.texts"
      service_headers:
        - key: "Authorization"
          value: "Bearer sk-b98f462xxxxxxxx"
        - key: "Content-Type"
          value:  "application/json"
```
这是请求官方 text-embedding-v2模型的请求样例 具体请求可以看 https://help.aliyun.com/zh/dashscope/developer-reference/text-embedding-api-details?spm=a2c22.12281978.0.0.4d596ea2lRn8xW
### 一个工作流的例子
从三个节点ABC获取信息，等到数据都就位了，再执行D。 并根据D的输出判断是否需要执行E还是直接结束
![dag.png](img/dag.png)
start的返回值(请求plugin的body)
```json
{
  "model":"qwen-7b-chat-xft",
  "frequency_penalty":0,
  "max_tokens":800,
  "stream":false,
  "messages": [{"role":"user","content":"higress项目主仓库的github地址是什么"}],
  "presence_penalty":0,"temperature":0.7,"top_p":0.95
}
```
A的返回值是
```json
{
    "output":{
        "embeddings": [
          {
             "text_index": 0,
             "embedding": [-0.006929283495992422,-0.005336422007530928]
          }, 
          {
             "text_index": 1,
             "embedding": [-0.006929283495992422,-0.005336422007530928]
          },
          {
             "text_index": 2,
             "embedding": [-0.006929283495992422,-0.005336422007530928]
          },
          {
             "text_index": 3,
             "embedding": [-0.006929283495992422,-0.005336422007530928]
          }
        ]
    },
    "usage":{
        "total_tokens":12
    },
    "request_id":"d89c06fb-46a1-47b6-acb9-bfb17f814969"
}
```
B的返回值是
```json
{"llm":"this is b"}
```
C的返回值是
```json
{
  "get": "this is c"
}
```
D的返回值是
```json
{"check": 0.99, "llm":{}}
```
E的返回值是
```json
{"save": "ok", "date":{}}
```
这个工作流的配置文件如下：
```yaml
env:
  max_depth: 100
  timeout: 3000
workflow:
  edges:
    - source: start
      target: A
    - source: start
      target: B
    - source: start
      target: C
    - source: A
      target: D
    - source: B
      target: D
    - source: C
      target: D
    - source: D
      target: end
      conditional: "lt {{D||check}} 0.9"
    - source: D
      target: E
      conditional: "gt {{D||check}} 0.9"
    - source: E
      target: end
  nodes:
    - name: "A"
      service_domain: "dashscope.aliyuncs.com"
      service_name: "dashscope"
      service_port: 443
      service_path: "/api/v1/services/embeddings/text-embedding/text-embedding"
      service_method: "POST"
      service_body_tmpl:
        model: "text-embedding-v2"
        input:
          texts: ""
        parameters:
          text_type: "query"
      service_body_replace_keys:
        - from: "start||messages.#(role==user)#.content"
          to: "input.texts"
      service_headers:
        - key: "Authorization"
          value: "Bearer sk-b98f462xxxxxxxx"
        - key: "Content-Type"
          value:  "application/json"
    - name: "B"
      service_body_tmpl:
        embeddings: "default"
        msg: "default request body"
        sk: "sk-xxxxxx"
      service_body_replace_keys:
      service_headers:
        - key: "AK"
          value: "ak-xxxxxxxxxxxxxxxxxxxx"
        - key: "Content-Type"
          value:  "application/json"
      service_method: "POST"
      service_name: "whoai.static"
      service_path: "/llm"
      service_port: 80
    - name: "C"
      service_method: "GET"
      service_name: "whoai.static"
      service_path: "/get"
      service_port: 80
    - name: "D"
      service_headers:
      service_method: "POST"
      service_name: "whoai.static"
      service_path: "/check_cache"
      service_port: 80
      service_body_tmpl:
        A_result: ""
        B_result: ""
        C_result: ""
      service_body_replace_keys:
        - from: "A||output.embeddings.0.embedding.0"
          to: "A_result"
        - from: "B||llm"
          to: "B_result"
        - from: "C||get"
          to: "C_result"
    - name: "E"
      service_method: "POST"
      service_name: "whoai.static"
      service_path: "/save_cache"
      service_port: 80
      service_body_tmpl:
        save: ""
      service_body_replace_keys:
        - from: "D||llm"
          to: "save"
```
执行请求
```bash
curl  -v '127.0.0.1:8080'  -H 'Accept: application/json, text/event-stream'  -H 'Content-Type: application/json'--data-raw '{"model":"qwen-7b-chat-xft","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"higress项目主仓库的github地址是什么"}],"presence_penalty":0,"temperature":0.7,"top_p":0.95}'
```

执行后的简略debug日志,可以看到工作流等到前置的ABC流程执行完毕后，根据返回值构建了D的body` {"A_result":0.007155838584362588,"B_result":"this is b","C_result":"this is c"}`;执行D后，根据D的返回值`{"check": 0.99, "llm":{}}`进行条件判断，最终继续执行了E`gt 0.99 0.9`，然后结束流程
```bash
[api-workflow] workflow exec task,source is start,target is A, body is {"input":{"texts":["higress项目主仓库的github地址是什么"]},"model":"text-embedding-v2","parameters":{"text_type":"query"}},header is [[Authorization Bearer sk-b98f4628125xxxxxxxxxxxxxxxx] [Content-Type application/json]]
[api-workflow] workflow exec task,source is start,target is B, body is {"embeddings":"default","msg":"default request body","sk":"sk-xxxxxx"},header is [[AK ak-xxxxxxxxxxxxxxxxxxxx] [Content-Type application/json]]
[api-workflow] workflow exec task,source is start,target is C, body is ,header is []
[api-workflow] source is B,target is D,stauts is map[A:0 B:0 C:0 D:2 E:1]
[api-workflow] source is C,target is D,stauts is map[A:0 B:0 C:0 D:1 E:1]
[api-workflow] source is A,target is D,stauts is map[A:0 B:0 C:0 D:0 E:1]
[api-workflow] workflow exec task,source is A,target is D, body is,header is []
[api-workflow] source is D,target is end,workflow is pass
[api-workflow] source is D,target is E,stauts is map[A:0 B:0 C:0 D:0 E:0]
[api-workflow] workflow exec task,source is D,target is E, body is {"save":"{\"A_result\":0.007155838584362588,\"B_result\":\"this is b\",\"C_result\":\"this is c\"}"},header is [] 
[api-workflow] source is E,target is end,workflow is end
```
