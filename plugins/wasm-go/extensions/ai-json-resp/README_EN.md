---
title: AI JSON Formatting
keywords: [ AI Gateway, AI JSON Formatting ]
description: AI JSON Formatting plugin configuration reference
---
## Function Description
LLM structured response plugin, used to structure AI responses according to the default or user-configured Json Schema for subsequent plugin processing. Note that only `non-streaming responses` are currently supported.

## Running Attributes
Plugin execution phase: `default phase`
Plugin execution priority: `150`

### Configuration Description
| Name | Type | Requirement | Default | **Description** |
| --- | --- | --- | --- | --- |
| serviceName | str |  required | - | AI service or gateway service name that supports AI-Proxy |
| serviceDomain | str |  optional | - | AI service or gateway service domain/IP address that supports AI-Proxy |
| servicePath | str |  optional | '/v1/chat/completions' | AI service or gateway service base path that supports AI-Proxy |
| serviceUrl | str |  optional | - | AI service or gateway service URL that supports AI-Proxy; the plugin will automatically extract domain and path to fill in the unconfigured serviceDomain or servicePath |
| servicePort | int |  optional | 443 | Gateway service port |
| serviceTimeout | int |  optional | 50000 | Default request timeout |
| maxRetry | int |  optional | 3 | Number of retry attempts when the answer cannot be correctly extracted and formatted |
| contentPath | str |  optional | "choices.0.message.content” | gpath path to extract the response result from the LLM answer |
| jsonSchema | str (json) |  optional | - | The jsonSchema against which the request is validated; if empty, only valid Json format responses are returned |
| enableSwagger | bool |  optional | false | Whether to enable the Swagger protocol for validation |
| enableOas3 | bool |  optional | true | Whether to enable the Oas3 protocol for validation |
| enableContentDisposition | bool | optional | true | Whether to enable the Content-Disposition header; if enabled, the response header will include `Content-Disposition: attachment; filename="response.json"` |

> For performance reasons, the maximum supported Json Schema depth is 6 by default. Json Schemas exceeding this depth will not be used to validate responses; the plugin will only check if the returned response is a valid Json format.

### Request and Return Parameter Description
- **Request Parameters**: The request format for this plugin is the OpenAI request format, including the `model` and `messages` fields, where `model` is the AI model name and `messages` is a list of conversation messages, each containing `role` and `content` fields, with `role` being the message role and `content` being the message content.
  ```json
  {
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "give me a api doc for add the variable x to x+5"}
    ]
  }
  ```
  Other request parameters should refer to the corresponding documentation of the configured AI service or gateway service.

- **Return Parameters**:
  - Returns a `Json format response` that satisfies the constraints of the defined Json Schema.
  - If no Json Schema is defined, returns a valid `Json format response`.
  - If an internal error occurs, returns `{ "Code": 10XX, "Msg": "Error message" }`.

## Request Example
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

## Return Example
### Normal Return
Under normal circumstances, the system should return JSON data validated by the JSON Schema. If no JSON Schema is configured, the system will return legally valid JSON data that complies with JSON standards.
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

### Exception Return
In case of an error, the return status code is `500`, and the return content is a JSON format error message. It contains two fields: error code `Code` and error message `Msg`.
```json
{
  "Code": 1006,
  "Msg": "retry count exceed max retry count"
}
```

### Error Code Description
| Error Code | Description |
| --- | --- |
| 1001 | The configured Json Schema is not in a valid Json format |
| 1002 | The configured Json Schema compilation failed; it is not a valid Json Schema format or depth exceeds jsonSchemaMaxDepth while rejectOnDepthExceeded is true |
| 1003 | Unable to extract valid Json from the response |
| 1004 | Response is an empty string |
| 1005 | Response does not conform to the Json Schema definition |
| 1006 | Retry count exceeds the maximum limit |
| 1007 | Unable to retrieve the response content; may be due to upstream service configuration errors or incorrect ContentPath path to get the content |
| 1008 | serviceDomain is empty; please note that either serviceDomain or serviceUrl cannot be empty at the same time |

## Service Configuration Description
This plugin requires configuration of upstream services to support automatic retry mechanisms in case of exceptions. Supported configurations mainly include `AI services supporting OpenAI interfaces` or `local gateway services`.

### AI Services Supporting OpenAI Interfaces
Taking Qwen as an example, the basic configuration is as follows:
```yaml
serviceName: qwen
serviceDomain: dashscope.aliyuncs.com
apiKey: [Your API Key]
servicePath: /compatible-mode/v1/chat/completions
jsonSchema:
  title: ReasoningSchema
  type: object
  properties:
    reasoning_steps:
      type: array
      items:
        type: string
      description: The reasoning steps leading to the final conclusion.
    answer:
      type: string
      description: The final answer, taking into account the reasoning steps.
  required:
    - reasoning_steps
    - answer
  additionalProperties: false
```

### Local Gateway Services
To reuse already configured services, this plugin also supports configuring local gateway services. For example, if the gateway has already configured the AI-proxy service, it can be directly configured as follows:

1. Create a service with a fixed IP address of 127.0.0.1:80, for example, localservice.static.
2. Add the service configuration for localservice.static in the configuration file.
```yaml
serviceName: localservice
serviceDomain: 127.0.0.1
servicePort: 80
```
3. Automatically extract request Path, Header, and other information.
The plugin will automatically extract request Path, Header, and other information to avoid repetitive configuration for the AI service.
