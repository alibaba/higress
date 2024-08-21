## 简介

**Note**

> 需要数据面的proxy wasm版本大于等于0.2.100
> 

> 编译时，需要带上版本的tag，例如：tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags="custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100" ./
> 

> 需要配合 [ai-proxy](../ai-proxy/README.md) 插件使用
> 

LLM响应结构化插件，用于根据默认或用户配置的Json Schema对AI的响应进行结构化，以便后续插件处理。注意目前只支持 `非流式响应`。

### 配置说明

| Name | Type | Requirement | Default | **Description** |
| --- | --- | --- | --- | --- |
| serviceName | str |  required | - | 网关服务名称 |
| serviceDomain | str |  required | - | 网关服务域名/IP地址 |
| servicePort | int |  required | - | 网关服务端口 |
| serviceTimeout | int |  optional | 50000 | 默认请求超时时间 |
| maxRetry | int |  optional | 3 | 若回答无法正确提取格式化时重试次数 |
| contentPath | str |  optional | "choices.0.message.content” | 从LLM回答中提取响应结果的gpath路径 |
| jsonSchema | str (json) |  optional | APITemp, details in the “./templates.go” | 验证请求所参照的jsonSchema |
| enableSwagger | bool |  optional | false | 是否启用Swagger协议进行验证 |
| enableOas3 | bool |  optional | true | 是否启用Oas3协议进行验证 |

### 请求和返回参数说明

- **请求参数**: 请参照ai-proxy的参数请求列表，本插件处理逻辑在ai-proxy返回的响应基础上进行Json提取，以及在提取或者验证失败时自动添加Prompt重试。因此无需特地配置针对本插件的请求参数。
- **返回参数**: 返回满足定义的Json Schema约束的 `Json格式响应` 或 `空字符串`

## 请求示例

```bash
curl -X POST "http://localhost:8001/v1/chat/completions" \
-H "Content-Type: application/json" \
-d '{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "give me a api doc for add the variable x to x+5"}
  ]
}'

```

返回Json为

```json
{
  "apiVersion": "1.0",
  "request": {
    "endpoint": "/add_to_five",
    "method": "POST",
    "port": 8080,
    "headers": {
      "Content-Type": "application/json"
    },
    "body": {
      "x": 7
    }
  }
}
```