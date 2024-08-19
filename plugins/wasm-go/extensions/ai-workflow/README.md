## 功能说明
可编排的AI workflow 插件，当前版本支持http工具编排，后续将添加官方model的封装，更方便调用。

## 配置说明

| 名称     | 数据类型                     | 填写要求 | 默认值 | 描述      | 备注 |
| ------ |--------------------------| ---- | --- | ------- |----|
| tools  | array of object          | 选填   |     | 工具的定义   |    |
| dsl    | array of workflow object | 必填   |     | 工作流的定义  |    |
|        |                          |      |     |         |    |

dsl.workflow object 配置说明：


| 名称          | 数据类型   | 填写要求 | 默认值 | 描述                                                    |
| ----------- | ------ | ---- | --- | ----------------------------------------------------- |
| source      | string | 必填   | -   | 上一步的操作，必须是定义的model或者tool的name，或者初始化工作流的start          |
| target      | string | 必填   | -   | 当前的操作，必须是定义的model或者tool的name，或者结束工作流的关键字 end continue |
| input       | string | 选填   | -   | 进入的数据过滤方式，使用gjson的表达式                                 |
| output      | string | 选填   | -   | 执行后的数据过滤方式，使用gjson的表达式                                |
| conditional | string | 选填   | -   | 这一步是否执行的判断条件                                          |

tool object 配置说明：

| 名称              | 数据类型    | 填写要求 | 默认值 | 描述                 | 备注                  |
| --------------- | ------- |------|-----| ------------------ |---------------------|
| name            | string  | 必填   | -   | 工具名称               | 全局唯一                |
| key             | string  | 选填   | -   | apikey             |                     |
| service_name    | string  | 必填   | -   | higress配置的服务名称     |                     |
| service_port    | int     | 选填   | 80  | higress配置的服务端口     |                     |
| service_domain  | string  | 选填   |     | higress配置的服务domain |                     |
| service_path    | string  | 选填   | /   | 请求的path            |                     |
| service_headers | 二维array | 选填   |     | 请求的头               |                     |
| service_body    | string  | 选填   |     | 请求的body模板          |                     |
| service_type    | string  | 必填   |     | 请求的类型              | static，domain       |
| service_method  | string  | 必填   |     | 请求的方法              | GET,POST,PUT,DELETE |


## 设计如下

我们把工作流抽象成dsl配置文件，附加定义了官方模型和工具方便调用。同时加上控制流和数据流更方便的控制流程和构造请求。


![image](https://github.com/user-attachments/assets/43f07b89-2f4d-4a02-a07d-9e6aaa9d2a7d)


## dsl的定义

workflow用来描述工作流的编排过程，分为控制流和数据流两部分介绍。

| 名称          | 数据类型   | 填写要求 | 默认值 | 描述                                                    |     |
| ----------- | ------ | ---- | --- | ----------------------------------------------------- | --- |
| source      | string | 必填   | -   | 上一步的操作，必须是定义的model或者tool的name，或者初始化工作流的start          |     |
| target      | string | 必填   | -   | 当前的操作，必须是定义的model或者tool的name，或者结束工作流的关键字 end continue |     |
| input       | string | 选填   | -   | 进入的数据过滤方式，使用gjson的表达式                                 |     |
| output      | string | 选填   | -   | 执行后的数据过滤方式，使用gjson的表达式                                |     |
| conditional | string | 选填   | -   | 这一步是否执行的判断条件                                          |     |
样例
```yaml
dsl:
  workflow:
    - conditional: "gt {{check-cache-output||check}} 0.9"
      source: "check-cache"
      target: "llm"
    - conditional: "lt {{check-cache-output||check}} 0.9"
      source: "check-cache"
      target: "end"
```
### 控制流 conditional 和 target
#### 分支 conditional
插件根据定义workflow一步步执行，当conditional的定义不为空，表明该操作是分支之一，通常分支只有一个可以执行，剩下的分支在执行会直接跳过
参数可以使用{{xxx}}标注，具体定义见数据流`模板和变量`


支持比较表达式如下：
```
eq arg1 arg2： arg1 == arg2时为true 
ne arg1 arg2： arg1 != arg2时为true 
lt arg1 arg2： arg1 < arg2时为true 
le arg1 arg2： arg1 <= arg2时为true 
gt arg1 arg2： arg1 > arg2时为true 
ge arg1 arg2： arg1 >= arg2时为true
```

#### 结束和执行工作流 target
当target为`name`,执行name的操作
当target 为`end`，直接返回source的结果，结束工作流
当target 为`continue`，结束工作流，将请求放行到下一个plugin

### 数据流

进入plugin的数据（request body）， 会依次传递给所有的执行动作（model和tool），同时把输入（input）和输出（output）的数据都记录在上下文，只支持json格式的数据。
过滤表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串

#### 模板和变量

##### workflow.conditional
配置文件的定义中，`workflow.conditional`  支持模板，方便根据数据流的数据来构建
在模板里使用变量来代表数据和过滤。变量使用`{{str1||str2}}`包裹，使用`||`分隔，str1代表使用前面命名为name的操作(tool/model)。`name-input`或者`name-output` 代表它的输入和输出，str2代表如何取数据，过滤表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串，`@all`代表全都要

例子
```yaml
conditional: "lt {{check-cache-output||check}} 0.9"
```
返回值是
```json
{"check": 0.99}
```
会从最近一次check-cache的工具返回值中取出数据，构造表达式

##### tool.service_body_tmpl 和 service_body_replace_keys
这组配置用来构造请求body，`tool.service_body_tmpl`是模板文件 ，`service_body_replace_keys`用来描述如何填充body，是一个二维数组，每个数组两个元素，前面一个表示填充的位置，后面一个标识数据从哪里来。
表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串

例子
```yaml
    service_body_tmpl:
      embeddings: 
        result: ""
      msg: ""
      sk: "sk-xxxxxx"
    service_body_replace_keys:
      - - "embeddings.result"
        - "embedding-output||output.embeddings.0.embedding"
      - - "msg"
        - "embedding-input||@all"
```
`embedding`模块的输出是
```json
{"embeddings":  {"output":{"embeddings":[{"embedding":[0.014398524595686043],"text_index":0}]},"usage":{"total_tokens":12},"request_id":"2a5229bc-53d9-91ca-bce2-00ae5e01a1d3"}}
```
`embedding`模块的输入是
```json
["higress项目主仓库的github地址是什么"]
```
解析后的例子
```json
{"embeddings":{"result":"[0.014398524595686043，......]"},"msg":["higress项目主仓库的github地址是什么"],"sk":"sk-xxxxxx"}
```
#### workflow.input 和 workflow.output
这两个会依据定义对进入或流出的数据进行过滤,使用gjson的表达式， [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
workflow.input 不为空的话， 在数据进入之后执行，
workflow.output不为空的话，在数据处理完之后执行


## tool的定义
外部工具，封装了httpCall，所以外部工具提供http的访问能力，获取各种api的能力。request body支持模板。

| 名称              | 数据类型          | 填写要求 | 默认值 | 描述                 | 备注            |
| --------------- | ------------- |---| --- | ------------------ | ------------- |
| name            | string        | 必填 | -   | 工具名称               | 全局唯一          |
| key             | string        | 选填 | -   | apikey             |               |
| service_name    | string        | 必填 | -   | higress配置的服务名称     |               |
| service_port    | int           | 选填 | 80  | higress配置的服务端口     |               |
| service_domain  | string        | 选填 |     | higress配置的服务domain |               |
| service_path    | string        | 必填 |     | 请求的path            |               |
| service_headers | 二维 string array | 选填 |     | 请求的头               |               |
| service_body_replace_keys|二维 string array| 选填|   请求body模板替换键值对  |   用来构造请求。前面一个表示填充的位置，后面一个标识数据从哪里|               |
| service_body_tmpl   | string        | 选填 |     | 请求的body模板          |               |
| service_type    | string        | 必填 |     | 请求的类型              | static，domain |
| service_method  | string        | 必填 |     | 请求的方法              | GET，POST      |
样例
```yaml
tools:
  - name: "embedding"
    service_domain: "dashscope.aliyuncs.com"
    service_name: "dashscope"
    service_port: 443
    service_type: "domain"
    service_path: "/api/v1/services/embeddings/text-embedding/text-embedding"
    service_method: "POST"
    service_body_tmpl:
      model: "text-embedding-v2"
      input:
        texts: ""
      parameters:
        text_type: "query"
    service_body_replace_keys :
      - - "input.texts"
        - "embedding-input||@all"
    service_headers:
      - - "Authorization"
        - "Bearer sk-b98f462xxxxxxxx"
      - - "Content-Type"
        - "application/json"
```
这是请求官方 text-embedding-v2模型的请求样例
## 一个工作流的例子

![image](https://github.com/user-attachments/assets/ef326f40-dc85-4936-8a9d-0398de1a848f)

whoai.static的web代码如下，使用gin模拟http工具
```go
r := gin.Default()
gin.SetMode(gin.DebugMode)	
r.POST("/check_cache", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		log.Println(string(body))
		c.JSON(200, gin.H{"check": 0.99})

	})
	r.POST("/save_cache", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		result := gjson.ParseBytes(body)
		get := result.Get("llm-result")
		r2 := result.Get("msg")
		log.Println(result.Get("llm-result").Exists(), result.Get("msg").Exists(), get, r2)
		log.Println(gin.H{"save": map[string]string{"llm-result": gjson.GetBytes(body, "llm-result").Raw}})

		c.JSON(200, gin.H{"save": map[string]string{"llm-result": gjson.GetBytes(body, "llm-result").Raw}})

	})
	r.POST("/llm", func(c *gin.Context) {

		c.JSON(200, gin.H{"llm": "hello,world"})

	})
```

配置文件如下：
```yaml
dsl:
  workflow:
    - input: "messages.#(role==user)#.content"
      output: ""
      source: "start"
      target: "embedding"
    - source: "embedding"
      target: "check-cache"
    - conditional: "gt {{check-cache-output||check}} 0.9"
      source: "check-cache"
      target: "llm"
    - conditional: "lt {{check-cache-output||check}} 0.9"
      source: "check-cache"
      target: "end"
    - source: "llm"
      target: "save-cache"
      output: "save.llm-result"
    - source: "save-cache"
      target: "end"
tools:
  - name: "embedding"
    service_domain: "dashscope.aliyuncs.com"
    service_name: "dashscope"
    service_port: 443
    service_type: "domain"
    service_path: "/api/v1/services/embeddings/text-embedding/text-embedding"
    service_method: "POST"
    service_body_tmpl:
      model: "text-embedding-v2"
      input:
        texts: ""
      parameters:
        text_type: "query"
    service_body_replace_keys :
      - - "input.texts"
        - "embedding-input||@all"
    service_headers:
      - - "Authorization"
        - "Bearer sk-b98f462xxxxxxxx"
      - - "Content-Type"
        - "application/json"
  - name: "check-cache"
    service_body_tmpl:
      embeddings: "embedding-output||output.embeddings.0.embedding"
      msg: "embedding-input||@all"
      sk: "sk-xxxxxx"
    service_body_replace_keys:
      - - "embeddings"
        - "check-cache-input||@all"
      - - "msg"
        - "embedding-input||@all"
    service_headers:
      - - "AK"
        - "ak-xxxxxxxxxxxxxxxxxxxx"
      - - "Content-Type"
        - "application/json"
    service_method: "POST"
    service_name: "whoai.static"
    service_path: "/check_cache"
    service_port: 80
    service_type: "static"
  - name: "save-cache"
    service_body_tmpl:
      embeddings: ""
      msg: ""
      llm-result: ""
      sk: "sk-ggggggg"
    service_body_replace_keys:
      - - "embeddings"
        - "check-cache-input||@all"
      - - "msg"
        - "embedding-input||@all"
      - - "llm-result"
        - "llm-output||@all"
    service_headers:
      - - "AK"
        - "ak-xxxxxxxxxxxxxxxxxxxx"
      - - "Content-Type"
        - "application/json"
    service_method: "POST"
    service_name: "whoai.static"
    service_path: "/save_cache"
    service_port: 80
    service_type: "static"
  - key: "ak-xxxxxxxxxxxxx"
    name: "llm"
    service_headers:
      - - "AK"
        - "ak-xxxxxxxxxxxxxxxxxxxx"
      - - "Content-Type"
        - "application/json"
    service_method: "POST"
    service_name: "whoai.static"
    service_path: "/llm"
    service_port: 80
    service_type: "static"
    service_body_tmpl:
      chat: "fffffffff"
      sk: "sk-cccccccccccc"
```

运行结果如下
```shell
[Envoy (Epoch 0)] [2024-08-15 12:14:15.732][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] wl is {start embedding 0x811d70 map[] messages.#(role==user)#.content  }
[Envoy (Epoch 0)] [2024-08-15 12:14:15.732][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] body is {"model":"text-embedding-v2","input":{"texts":["[\"higress项目主仓库的github地址是什么\"]"]},"parameters":{"text_type":"query"}},header is [[Authorization Bearer sk-xxxxxxxxxxxxx] [Content-Type application/json]]
[Envoy (Epoch 0)] [2024-08-15 12:14:15.949][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] wl is {embedding check-cache 0x811b90 map[]   }
[Envoy (Epoch 0)] [2024-08-15 12:14:15.953][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] body is {"embeddings":"[0.014398524595686043，......]","msg":"["higress项目主仓库的github地址是什么"]","sk":"sk-xxxxxx"},header is [[AK ak-xxxxxxxxxxxxxxxxxxxx] [Content-Type application/json]]
[Envoy (Epoch 0)] [2024-08-15 12:14:15.960][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] wl is {check-cache llm 0x8119b0 map[]   gt {{check-cache-output||check}} 0.9}
[Envoy (Epoch 0)] [2024-08-15 12:14:15.960][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] ExecConditional is gt 0.99 0.9 
[Envoy (Epoch 0)] [2024-08-15 12:14:15.961][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] body is {"chat":"["higress项目 主仓库的github地址是什么"]","sk":"sk-cccccccccccc"},header is [[AK ak-xxxxxxxxxxxxxxxxxxxx] [Content-Type application/json]]
[Envoy (Epoch 0)] [2024-08-15 12:14:15.964][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] wl is {check-cache end 0x8117d0 map[]   lt {{check-cache-output||check}} 0.9}
[Envoy (Epoch 0)] [2024-08-15 12:14:15.964][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] ExecConditional is lt 0.99 0.9 
[Envoy (Epoch 0)] [2024-08-15 12:14:15.964][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] wl is pass
[Envoy (Epoch 0)] [2024-08-15 12:14:15.964][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] wl is {llm save-cache 0x8115f0 map[]   }
[Envoy (Epoch 0)] [2024-08-15 12:14:15.965][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] body is {"embeddings":"{"output":{"embeddings":[{"embedding":[0.014398524595686043,.......],"text_index":0}]},"usage":{"total_tokens":12},"request_id":"2a5229bc-53d9-91ca-bce2-00ae5e01a1d3"}","msg":"["higress项目主仓库的github地址是什么"]","sk":"sk-ggggggg"},header is [[AK ak-xxxxxxxxxxxxxxxxxxxx] [Content-Type application/json]]
[Envoy (Epoch 0)] [2024-08-15 12:14:15.965][255][debug][wasm] wasm log higress-system.wl-1.0.0: [ai-workflow] wl is end
```

# 下一步工作计划

## model的定义 (todo)
官方模型，本质也是使用http请求，下一步计划封装了一些请求内容，方便使用。

| 名称              | 数据类型         | 填写要求 | 默认值 | 描述                               | 备注             |
| --------------- | ------------ | ---- | --- | -------------------------------- | -------------- |
| name            | string       | 必填   | -   | 名称，全局唯一                          |                |
| model_type      | string       | 必填   | -   | llm,embedding,audio,image,rerank |                |
| model_name      | string       | 必填   | -   | 官方模型的名称，比如text-embedding-v2      |                |
| key             | string       | 必填   |     | 官方key                            |                |
| service_name    | string       | 必填   | -   | higress配置的服务名称                   |                |
| service_port    | int          | 必填   |     | higress配置的服务端口                   |                |
| service_domain  | string       | 必填   |     | higress配置的服务domain               |                |
| embedding_input | string array | 选填   |     | embedding 模型 输入参数                | embedding 模型必填 |
| chart_message   | string       | 选填   |     | llm的Prompt模板                     | llm模型必填        |
样例
```yaml
models:
  - chart_message:
    embedding_input:
      - "{{embedding-input||@all}}"
    key: "sk-b98f4628125e4f178f7c340exxxxxxx"
    model_name: "text-embedding-v2"
    model_type: "embeddings"
    name: "embedding"
    service_domain: "dashscope.aliyuncs.com"
    service_name: "dashscope"
    service_port: 443
```