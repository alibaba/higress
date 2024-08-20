## 简介

**Note**

> 需要数据面的proxy wasm版本大于等于0.2.100
> 

> 编译时，需要带上版本的tag，例如：tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags="custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100" ./
> 

> 需搭配[AI-Proxy插件](https://github.com/alibaba/higress/tree/main/plugins/wasm-go/extensions/ai-proxy)，并且需要配置 `provider = openai`, `model=gpt-4o-2024-08-06` 以支持结构化输出接口
> 

AI 结构化Json文档生成和验证，可根据自然语言描述生成Json文档或者Json Schema，或根据预先定义的JsonSchema验证Json文档是否正确，并给出解释。

## 配置说明

| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| model | string | optional | "gpt-4o-2024-08-06" | 指定的模型服务，注意需选择支持结构化输出接口的模型 |
| enable_swagger | bool | optional | false | 是否启用swagger验证文档 |
| enable_oas3 | bool | optional | true | 是否启用oas3验证文档 |
| custom_askjson | object | optional | “” | 生成Json文档时使用的JsonSchema约束，此设置会覆盖默认设置 |
| custom_askjsonschema | object | optional | “” | 生成Json Schema约束时使用的JsonSchema约束，此设置会覆盖默认设置 |
| custom_askverify | object | optional | “” | 当验证Json文档不符合给定的Json Schema约束时，传入后续模型服务使用的JsonSchema约束，此设置会覆盖默认设置 |

## 配置示例

```yaml
Model: "gpt-4o-2024-08-06"
EnableOas3: true

```

## 请求格式

| Name | Type | Requirement | Description |
| --- | --- | --- | --- |
| desc | string | required | 自然语言描述，需要用户提供，单独提供可以根据默认Json Schema生成Json文档 |
| json_doc | string | optional | json文档，和 desc 搭配可以生成相应的Json Schema约束 |
| json_schema | string | optional | [type=val] 用于验证 json_doc 是否满足约束 [type=gen] 和 json_doc 一起提示LLM生成相应的Json Schema约束 |
| type | string | required | 指定为 gen 或者 val ，分别代表生成和验证模式 |

返回参数

| Name | Type | Requirement | Description |
| --- | --- | --- | --- |
| reason | string | required | LLM后端的自然语言回答 |
| json | string | optional | 返回的json文档获json schema约束 |
| listOfCases | string | optional | 如果生成Json文档（API文档），相应给出案例说明 |

## 请求示例

### 1. 根据自然语言描述生成Json文档，并给出解释 （type=gen)

```bash
curl -X POST "http://loalhost:8001/v1/chat/completions" \
-H "Content-Type: application/json" \
-d '{
  "model": "gpt-4o-2024-08-06",
  "desc": "帮我设计API接口：在请求的变量上加5，返回加和",
  "type": "gen"
}'
```

> 默认使用 `templates\askjson.go` 中定义的json schema来生成Json文档，可以使用配置中的 `custom_askjson` 来替换
> 

返回示例：

```json
{
  "json": {
    "api_version": "1.0",
    "endpoint": "/addFive",
    "method": "POST",
    "parameters": [
      {
        "name": "number",
        "type": "integer"
      }
    ],
    "response": {
      "message": "The sum of the input number and 5.",
      "status": "success"
    }
  },
  "listOfCases": [
    {
      "caseDescription": "Add 5 to the positive integer 10",
      "caseName": "Adding to positive integer",
      "input": "number=10",
      "output": "15"
    },
    {
      "caseDescription": "Add 5 to zero",
      "caseName": "Adding to zero",
      "input": "number=0",
      "output": "5"
    },
    {
      "caseDescription": "Add 5 to negative integer -3",
      "caseName": "Adding to negative integer",
      "input": "number=-3",
      "output": "2"
    }
  ],
  "reason": "This API is designed to take an integer input, add the number 5 to it, and return the result. It demonstrates a simple arithmetic operation applied to an API input parameter."
}
```

### 2. 根据Json文档生成Json Schema，并给出解释 (type=gen)

```bash
curl -X POST "http://localhost:8001/v1/chat/completions" \
-H "Content-Type: application/json" \
-d '{
  "model": "gpt-4o-2024-08-06",
  "desc": "我希望实现一个 JSON Schema，用于约束 Kibana 配置文件的结构。请根据以下测试用例帮助我编写该 JSON Schema。",
  "json_doc": "{ \"attributes\": { \"title\": \"Sample Dashboard\", \"description\": \"This is a sample dashboard.\", \"panelsJSON\": \"[{\\\"panelIndex\\\":\\\"1\\\",\\\"gridData\\\":{\\\"x\\\":0,\\\"y\\\":0,\\\"w\\\":24,\\\"h\\\":15},\\\"type\\\":\\\"visualization\\\",\\\"id\\\":\\\"1\\\"}]\", \"optionsJSON\": \"{\\\"darkTheme\\\":false}\", \"version\": 1, \"timeRestore\": false, \"kibanaSavedObjectMeta\": { \"searchSourceJSON\": \"{\\\"query\\\":{\\\"query\\\":\\\"\\\",\\\"language\\\":\\\"lucene\\\"},\\\"filter\\\":[]}\" } }, \"type\": \"dashboard\" }",
  "type": "gen"
}'

```

> 默认使用 `templates\askjson.go` 中定义的json schema来生成Json文档，可以使用配置中的 `custom_askjsonschema` 来替换，注意由于现有的结构化输出接口不支持所有的Json Schema语法，所以这里生成的Json Schema仅为String类型，用户使用时需要根据实际情况进行验证并调整
> 

```json
{
  "json": "{\n  \"$schema\": \"http://json-schema.org/draft-07/schema#\",\n  \"type\": \"object\",\n  \"properties\": {\n    \"attributes\": {\n      \"type\": \"object\",\n      \"properties\": {\n        \"title\": {\n          \"type\": \"string\"\n        },\n        \"description\": {\n          \"type\": \"string\"\n        },\n        \"panelsJSON\": {\n          \"type\": \"string\",\n          \"description\": \"Serialized JSON string of panels configuration\"\n        },\n        \"optionsJSON\": {\n          \"type\": \"string\",\n          \"description\": \"Serialized JSON string of dashboard options\"\n        },\n        \"version\": {\n          \"type\": \"integer\"\n        },\n        \"timeRestore\": {\n          \"type\": \"boolean\"\n        },\n        \"kibanaSavedObjectMeta\": {\n          \"type\": \"object\",\n          \"properties\": {\n            \"searchSourceJSON\": {\n              \"type\": \"string\",\n              \"description\": \"Serialized JSON string of search source configuration\"\n            }\n          },\n          \"required\": [\"searchSourceJSON\"]\n        }\n      },\n      \"required\": [\"title\", \"panelsJSON\", \"version\", \"kibanaSavedObjectMeta\"]\n    },\n    \"type\": {\n      \"type\": \"string\",\n      \"enum\": [\"dashboard\"]\n    }\n  },\n  \"required\": [\"attributes\", \"type\"]\n}",
  "reason": "The JSON schema defines the required structure for a Kibana configuration file based on the provided example, ensuring correct data types and required fields."
}
```

### 3. 给定Json Schema检验Json文档是否正确，并给出解释 (type=val)

```bash
curl -X POST "http://localhost:8001/v1/chat/completions" \
-H "Content-Type: application/json" \
-d '{
  "desc": "请帮我检查一下这个case和对应的jsonschema是否匹配请检查以下提供的案例及其对应的JSON Schema是否匹配。",
  "json_doc": "{ \"attributes\": { \"title\": \"Sample Dashboard\", \"description\": \"This is a sample dashboard.\", \"panelsJSON\": \"[{\\\"panelIndex\\\":\\\"1\\\",\\\"gridData\\\":{\\\"x\\\":0,\\\"y\\\":0,\\\"w\\\":24,\\\"h\\\":15},\\\"type\\\":\\\"visualization\\\",\\\"id\\\":\\\"1\\\"}]\", \"optionsJSON\": \"{\\\"darkTheme\\\":false}\", \"version\": 1, \"timeRestore\": false, \"kibanaSavedObjectMeta\": { \"searchSourceJSON\": \"{\\\"query\\\":{\\\"query\\\":\\\"\\\",\\\"language\\\":\\\"lucene\\\"},\\\"filter\\\":[]}\" } }, \"type\": \"dashboard\" }",
  "json_schema": "{ \"type\": \"object\", \"properties\": { \"attributes\": { \"type\": \"object\", \"properties\": { \"title\": { \"type\": \"string\" }, \"description\": { \"type\": \"string\" }, \"panelsJSON\": { \"type\": \"string\" }, \"optionsJSON\": { \"type\": \"string\" }, \"version\": { \"type\": \"integer\" }, \"timeRestore\": { \"type\": \"boolean\" }, \"kibanaSavedObjectMeta\": { \"type\": \"object\", \"properties\": { \"searchSourceJSON\": { \"type\": \"string\" } }, \"required\": [\"searchSourceJSON\"] } }, \"required\": [\"title\", \"description\", \"panelsJSON\", \"optionsJSON\", \"version\", \"timeRestore\", \"kibanaSavedObjectMeta\"] }, \"type\": { \"type\": \"string\" } }, \"required\": [\"attributes\", \"type\"] }",
  "type": "val"
}'
```

当检验通过

```json
{"reason": "case is valid"}
```

当检验不通过

> 这里的 `json` 代表尝试更正后的 `json_doc` , `reason` 代表对不匹配原因的解释
> 

```json
{
  "json": "{\n  \"attributes\": { \n    \"title\": \"Sample Dashboard\", \n    \"description\": \"This is a sample dashboard.\", \n    \"panelsJSON\": \"[{\\\"panelIndex\\\":\\\"1\\\",\\\"gridData\\\":{\\\"x\\\":0,\\\"y\\\":0,\\\"w\\\":24,\\\"h\\\":15},\\\"type\\\":\\\"visualization\\\",\\\"id\\\":\\\"1\\\"}]\", \n    \"optionsJSON\": \"{\\\"darkTheme\\\":false}\", \n    \"version\": 1, \n    \"timeRestore\": false, \n    \"kibanaSavedObjectMeta\": { \n      \"searchSourceJSON\": \"{\\\"query\\\":{\\\"query\\\":\\\"\\\",\\\"language\\\":\\\"lucene\\\"},\\\"filter\\\":[]}\" \n    } \n  }, \n  \"type\": \"dashboard\" \n}",
  "reason": "The JSON case does not match the schema because a key in the 'attributes' object is incorrectly named. In the JSON case, 'tile' is used instead of 'title', which is the required key name as per the JSON schema. To fix this, we need to rename 'tile' to 'title', ensuring that all required keys in the schema are present and correctly named. After this correction, all keys in the JSON case will align with those specified in the JSON schema."
}

```

当提供的JsonSchema格式不正确

```json
{"reason": "failed to compile json schema, please check the json schema you provided"}
```