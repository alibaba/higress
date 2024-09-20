---
title: AI Agent
keywords: [ AI Gateway, AI Agent ]
description: AI Agent plugin configuration reference
---
## Functional Description
A customizable API AI Agent that supports configuring HTTP method types as GET and POST APIs. Supports multiple dialogue rounds, streaming and non-streaming modes.  
The agent flow chart is as follows:  
![ai-agent](https://github.com/user-attachments/assets/b0761a0c-1afa-496c-a98e-bb9f38b340f8)  

## Runtime Properties
Plugin execution phase: `Default Phase`  
Plugin execution priority: `20`  

## Configuration Fields

### Basic Configuration
| Name             | Data Type | Requirement | Default Value | Description                      |
|------------------|-----------|-------------|---------------|----------------------------------|
| `llm`            | object    | Required    | -             | Configuration information for AI service provider  |
| `apis`           | object    | Required    | -             | Configuration information for external API service provider  |
| `promptTemplate` | object    | Optional    | -             | Configuration information for Agent ReAct template  |

The configuration fields for `llm` are as follows:  
| Name               | Data Type | Requirement | Default Value | Description                         |
|--------------------|-----------|-------------|---------------|-------------------------------------|
| `apiKey`           | string    | Required    | -             | Token for authentication when accessing large model services.  |
| `serviceName`      | string    | Required    | -             | Name of the large model service                      |
| `servicePort`      | int       | Required    | -             | Port of the large model service                   |
| `domain`           | string    | Required    | -             | Domain for accessing the large model service       |
| `path`             | string    | Required    | -             | Path for accessing the large model service         |
| `model`            | string    | Required    | -             | Model name for accessing the large model service     |
| `maxIterations`    | int       | Required    | 15            | Maximum steps before ending the execution loop         |
| `maxExecutionTime` | int       | Required    | 50000         | Timeout for each request to the large model, in milliseconds |
| `maxTokens`        | int       | Required    | 1000          | Token limit for each request to the large model       |

The configuration fields for `apis` are as follows:  
| Name            | Data Type | Requirement | Default Value | Description                         |
|-----------------|-----------|-------------|---------------|-------------------------------------|
| `apiProvider`   | object    | Required    | -             | Information about the external API service  |
| `api`           | string    | Required    | -             | OpenAPI documentation of the tool   |

The configuration fields for `apiProvider` are as follows:  
| Name              | Data Type | Requirement | Default Value | Description                                      |
|-------------------|-----------|-------------|---------------|--------------------------------------------------|
| `apiKey`          | object    | Optional    | -             | Token for authentication when accessing external API services.  |
| `maxExecutionTime`| int       | Optional    | 50000         | Timeout for each request to the API, in milliseconds|
| `serviceName`     | string    | Required    | -             | Name of the external API service                    |
| `servicePort`     | int       | Required    | -             | Port of the external API service                    |
| `domain`          | string    | Required    | -             | Domain for accessing the external API               |

The configuration fields for `apiKey` are as follows:  
| Name              | Data Type | Requirement | Default Value | Description                                                                          |
|-------------------|-----------|-------------|---------------|-------------------------------------------------------------------------------------|
| `in`              | string    | Optional    | none          | Whether the authentication token for accessing the external API service is in the header or in the query; If the API does not have a token, fill in none.   |
| `name`            | string    | Optional    | -             | The name of the token for authentication when accessing the external API service. |
| `value`           | string    | Optional    | -             | The value of the token for authentication when accessing the external API service.  |

The configuration fields for `promptTemplate` are as follows:  
| Name            | Data Type | Requirement | Default Value | Description                                        |
|-----------------|-----------|-------------|---------------|----------------------------------------------------|
| `language`      | string    | Optional    | EN            | Language type of the Agent ReAct template, including CH and EN. |
| `chTemplate`    | object    | Optional    | -             | Agent ReAct Chinese template                      |
| `enTemplate`    | object    | Optional    | -             | Agent ReAct English template                       |

The configuration fields for `chTemplate` and `enTemplate` are as follows:  
| Name            | Data Type | Requirement | Default Value | Description                                       |
|-----------------|-----------|-------------|---------------|---------------------------------------------------|
| `question`      | string    | Optional    | -             | The question part of the Agent ReAct template       |
| `thought1`      | string    | Optional    | -             | The thought1 part of the Agent ReAct template       |
| `observation`   | string    | Optional    | -             | The observation part of the Agent ReAct template     |
| `thought2`      | string    | Optional    | -             | The thought2 part of the Agent ReAct template       |

## Usage Example
**Configuration Information**  
```yaml  
llm:  
  apiKey: xxxxxxxxxxxxxxxxxx  
  domain: dashscope.aliyuncs.com  
  serviceName: dashscope.dns  
  servicePort: 443  
  path: /compatible-mode/v1/chat/completions  
  model: qwen-max-0403  
  maxIterations: 2  
promptTemplate:  
  language: CH  
apis:  
- apiProvider:  
    domain: restapi.amap.com  
    serviceName: geo.dns  
    servicePort: 80  
    apiKey:  
      in: query  
      name: key  
      value: xxxxxxxxxxxxxxx  
  api: |  
    openapi: 3.1.0  
    info:  
      title: Amap  
      description: Get related information of POI  
      version: v1.0.0  
    servers:  
      - url: https://restapi.amap.com  
    paths:  
      /v5/place/text:  
        get:  
          description: Get latitude and longitude coordinates based on POI name  
          operationId: get_location_coordinate  
          parameters:  
            - name: keywords  
              in: query  
              description: POI name, must be in Chinese  
              required: true  
              schema:  
                type: string  
            - name: region  
              in: query  
              description: The name of the region where the POI is located, must be in Chinese  
              required: true  
              schema:  
                type: string  
          deprecated: false  
      /v5/place/around:  
        get:  
          description: Search for POI near the given coordinates  
          operationId: search_nearby_pois  
          parameters:  
            - name: keywords  
              in: query  
              description: Keywords for the target POI  
              required: true  
              schema:  
                type: string  
            - name: location  
              in: query  
              description: Latitude and longitude of the center point, separated by a comma  
              required: true  
              schema:  
                type: string  
          deprecated: false  
    components:  
      schemas: {}  
- apiProvider:  
    domain: api.seniverse.com  
    serviceName: seniverse.dns  
    servicePort: 80  
    apiKey:  
      in: query  
      name: key  
      value: xxxxxxxxxxxxxxx  
  api: |  
    openapi: 3.1.0  
    info:  
      title: XZWeather  
      description: Get weather related information  
      version: v1.0.0  
    servers:  
      - url: https://api.seniverse.com  
    paths:  
      /v3/weather/now.json:  
        get:  
          description: Get weather conditions for a specified city  
          operationId: get_weather_now  
          parameters:  
            - name: location  
              in: query  
              description: The city to query  
              required: true  
              schema:  
                type: string  
            - name: language  
              in: query  
              description: Language used for the weather query results  
              required: true  
              schema:  
                type: string  
                default: zh-Hans  
                enum:  
                  - zh-Hans  
                  - en  
                  - ja  
            - name: unit  
              in: query  
              description: Units of temperature, available in Celsius and Fahrenheit  
              required: true  
              schema:  
                type: string  
                default: c  
                enum:  
                  - c  
                  - f  
          deprecated: false  
    components:  
      schemas: {}  
- apiProvider:  
    apiKey:  
      in: "header"  
      name: "DeepL-Auth-Key"  
      value: "73xxxxxxxxxxxxxxx:fx"  
    domain: "api-free.deepl.com"  
    serviceName: "deepl.dns"  
    servicePort: 443  
  api: |  
    openapi: 3.1.0  
    info:  
      title: DeepL API Documentation  
      description: The DeepL API provides programmatic access to DeepL’s machine translation technology.  
      version: v1.0.0  
    servers:  
      - url: https://api-free.deepl.com/v2  
    paths:  
      /translate:  
        post:  
          summary: Request Translation  
          operationId: translateText  
          requestBody:  
            required: true  
            content:  
              application/json:  
                schema:  
                  type: object  
                  required:  
                    - text  
                    - target_lang  
                  properties:  
                    text:  
                      description: |  
                        Text to be translated. Only UTF-8-encoded plain text is supported. 
                        The parameter may be specified up to 50 times in a single request. 
                        Translations are returned in the same order as they are requested.  
                      type: array  
                      maxItems: 50  
                      items:  
                        type: string  
                        example: Hello, World!  
                    target_lang:  
                      description: The language into which the text should be translated.  
                      type: string  
                      enum:  
                        - BG  
                        - CS  
                        - DA  
                        - DE  
                        - EL  
                        - EN-GB  
                        - EN-US  
                        - ES  
                        - ET  
                        - FI  
                        - FR  
                        - HU  
                        - ID  
                        - IT  
                        - JA  
                        - KO  
                        - LT  
                        - LV  
                        - NB  
                        - NL  
                        - PL  
                        - PT-BR  
                        - PT-PT  
                        - RO  
                        - RU  
                        - SK  
                        - SL  
                        - SV  
                        - TR  
                        - UK  
                        - ZH  
                        - ZH-HANS  
                      example: DE  
    components:  
      schemas: {}  
```  
This example configures three services demonstrating both GET and POST types of tools. The GET type tools include Amap and XZWeather, while the POST type tool is the DeepL translation. All three services need to be properly configured in the Higress service with DNS domain names and should be healthy.  
Amap provides two tools, one for obtaining the coordinates of a specified location and the other for searching for points of interest near the coordinates. Document: https://lbs.amap.com/api/webservice/guide/api-advanced/newpoisearch  
XZWeather provides one tool to get real-time weather conditions for a specified city, supporting results in Chinese, English, and Japanese, as well as representations in Celsius and Fahrenheit. Document: https://seniverse.yuque.com/hyper_data/api_v3/nyiu3t  
DeepL provides one tool for translating given sentences, supporting multiple languages. Document: https://developers.deepl.com/docs/v/zh/api-reference/translate?fallback=true  

Below are test cases. For stability, it is recommended to maintain a stable version of the large model. The example used here is qwen-max-0403:  
**Request Example**  
```shell  
curl 'http://<replace with gateway public IP>/api/openai/v1/chat/completions' \  
-H 'Accept: application/json, text/event-stream' \  
-H 'Content-Type: application/json' \  
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"I want to have coffee near the Xinshi Building in Jinan, please recommend a few."}],"presence_penalty":0,"temperature":0,"top_p":0}'  
```  
**Response Example**  
```json  
{"id":"139487e7-96a0-9b13-91b4-290fb79ac992","choices":[{"index":0,"message":{"role":"assistant","content":" Near the Xinshi Building in Jinan, you can choose from the following coffee shops:\n1. luckin coffee 瑞幸咖啡(鑫盛大厦店), located in the lobby of Xinshi Building, No. 1299 Xinluo Avenue;\n2. 三庆齐盛广场挪瓦咖啡(三庆·齐盛广场店), located 60 meters southwest of the intersection of Xinluo Avenue and Yingxiu Road;\n3. luckin coffee 瑞幸咖啡(三庆·齐盛广场店), located at No. 1267 Yingxiu Road;\n4. 库迪咖啡(齐鲁软件园店), located in the commercial space of Building 4, Sanqing Qisheng Plaza, Xinluo Avenue;\n5. 库迪咖啡(美莲广场店), located at L117, Meilian Plaza, No. 1166 Xinluo Avenue, High-tech Zone; and a few other options. I hope these suggestions help!"},"finish_reason":"stop"}],"created":1723172296,"model":"qwen-max-0403","object":"chat.completion","usage":{"prompt_tokens":886,"completion_tokens":50,"total_tokens":936}}  
```  
**Request Example**  
```shell  
curl 'http://<replace with gateway public IP>/api/openai/v1/chat/completions' \  
-H 'Accept: application/json, text/event-stream' \  
-H 'Content-Type: application/json' \  
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"What is the current weather in Jinan?"}],"presence_penalty":0,"temperature":0,"top_p":0}'  
```  
**Response Example**  
```json  
{"id":"ebd6ea91-8e38-9e14-9a5b-90178d2edea4","choices":[{"index":0,"message":{"role":"assistant","content":" The current weather condition in Jinan is overcast, with a temperature of 31°C. This information was last updated on August 9, 2024, at 15:12 (Beijing time)."},"finish_reason":"stop"}],"created":1723187991,"model":"qwen-max-0403","object":"chat.completion","usage":{"prompt_tokens":890,"completion_tokens":56,"total_tokens":946}}  
```  
**Request Example**  
```shell  
curl 'http://<replace with gateway public IP>/api/openai/v1/chat/completions' \  
-H 'Accept: application/json, text/event-stream' \  
-H 'Content-Type: application/json' \  
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"What is the current weather in Jinan?"},{"role":"assistant","content":" The current weather condition in Jinan is overcast, with a temperature of 31°C. This information was last updated on August 9, 2024, at 15:12 (Beijing time)."},{"role":"user","content":"BeiJing?"}],"presence_penalty":0,"temperature":0,"top_p":0}'  
```  
**Response Example**  
```json  
{"id":"ebd6ea91-8e38-9e14-9a5b-90178d2edea4","choices":[{"index":0,"message":{"role":"assistant","content":" The current weather condition in Beijing is overcast, with a temperature of 19°C. This information was last updated on Sep 12, 2024, at 22:17 (Beijing time)."},"finish_reason":"stop"}],"created":1723187991,"model":"qwen-max-0403","object":"chat.completion","usage":{"prompt_tokens":999,"completion_tokens":76,"total_tokens":1075}}  
```  
**Request Example**  
```shell  
curl 'http://<replace with gateway public IP>/api/openai/v1/chat/completions' \  
-H 'Accept: application/json, text/event-stream' \  
-H 'Content-Type: application/json' \  
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"What is the current weather in Jinan? Please indicate in Fahrenheit and respond in Japanese."}],"presence_penalty":0,"temperature":0,"top_p":0}'  
```  
**Response Example**  
```json  
{"id":"ebd6ea91-8e38-9e14-9a5b-90178d2edea4","choices":[{"index":0,"message":{"role":"assistant","content":" 現在の济南の天気は曇りで、気温は88°Fです。この情報は2024年8月9日15時12分（東京時間）に更新されました。"},"finish_reason":"stop"}],"created":1723187991,"model":"qwen-max-0403","object":"chat.completion","usage":{"prompt_tokens":890,"completion_tokens":56,"total_tokens":946}}  
```  
**Request Example**  
```shell  
curl 'http://<replace with gateway public IP>/api/openai/v1/chat/completions' \  
-H 'Accept: application/json, text/event-stream' \  
-H 'Content-Type: application/json' \  
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"Help me translate the following sentence into German: \"Hail Hydra!\""}],"presence_penalty":0,"temperature":0,"top_p":0}'  
```  
**Response Example**  
```json  
{"id":"65dcf12c-61ff-9e68-bffa-44fc9e6070d5","choices":[{"index":0,"message":{"role":"assistant","content":" The German translation of \"Hail Hydra!\" is \"Hoch lebe Hydra!\"."},"finish_reason":"stop"}],"created":1724043865,"model":"qwen-max-0403","object":"chat.completion","usage":{"prompt_tokens":908,"completion_tokens":52,"total_tokens":960}}  
```  

