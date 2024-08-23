---
title: AI Agent
keywords: [ AI网关, AI Agent ]
description: AI Agent插件配置参考
---

## 功能说明
一个可定制化的 API AI Agent，目前第一版本只支持配置 http method 类型为 GET 的 API，且只支持非流式模式。agent流程图如下：
![ai-agent](https://github.com/user-attachments/assets/b0761a0c-1afa-496c-a98e-bb9f38b340f8)

由于 Agent 是多轮对话场景，需要维护历史对话记录，本版本目前是在内存中维护历史对话记录，因此只支持单机。后续会支持通过 redis 存储回话记录

## 配置字段

### 基本配置
| 名称             | 数据类型   | 填写要求 | 默认值 | 描述                       |
|------------------|-----------|---------|--------|----------------------------|
| `llm`            | object    | 必填    | -      | 配置 AI 服务提供商的信息     |
| `apis`           | object    | 必填    | -      | 配置外部 API 服务提供商的信息 |
| `promptTemplate` | object    | 非必填  | -      | 配置 Agent ReAct 模板的信息  |

`llm`的配置字段说明如下：

| 名称            | 数据类型   | 填写要求 | 默认值 | 描述                               |
|-----------------|-----------|---------|--------|-----------------------------------|
| `apiKey`        | string    | 必填    | -      | 用于在访问大模型服务时进行认证的令牌。|
| `serviceName`   | string    | 必填    | -      | 大模型服务名                       |
| `servicePort`   | int       | 必填    | -      | 大模型服务端口                     |
| `domain`        | string    | 必填    | -      | 访问大模型服务时域名                |
| `path`          | string    | 必填    | -      | 访问大模型服务时路径                |
| `model`         | string    | 必填    | -      | 访问大模型服务时模型名              |

`apis`的配置字段说明如下：

| 名称            | 数据类型   | 填写要求 | 默认值 | 描述                               |
|-----------------|-----------|---------|--------|-----------------------------------|
| `apiProvider`   | object    | 必填     | -     | 外部 API 服务信息                   |
| `api`           | string    | 必填     | -     | 工具的 OpenAPI 文档                 |

`apiProvider`的配置字段说明如下：

| 名称            | 数据类型   | 填写要求 | 默认值 | 描述                                      |
|-----------------|-----------|---------|--------|------------------------------------------|
| `apiKey`        | object    | 非必填   | -     | 用于在访问外部 API 服务时进行认证的令牌。    |
| `serviceName`   | string    | 必填     | -     | 访问外部 API 服务名                        |
| `servicePort`   | int       | 必填     | -     | 访问外部 API 服务端口                      |
| `domain`        | string    | 必填     | -     | 访访问外部 API 时域名                      |

`apiKey`的配置字段说明如下：

| 名称              | 数据类型 | 填写要求    | 默认值  | 描述                                                                          |
|-------------------|---------|------------|--------|-------------------------------------------------------------------------------|
| `in`              | string  | 非必填     | header | 在访问外部 API 服务时进行认证的令牌是放在 header 中还是放在 query 中，默认是 header。
| `name`            | string  | 非必填     | -      | 用于在访问外部 API 服务时进行认证的令牌的名称。 |
| `value`           | string  | 非必填     | -      | 用于在访问外部 API 服务时进行认证的令牌的值。   |

`promptTemplate`的配置字段说明如下：

| 名称            | 数据类型   | 填写要求   | 默认值 | 描述                                        |
|-----------------|-----------|-----------|--------|--------------------------------------------|
| `language`      | string    | 非必填     | EN    | Agent ReAct 模板的语言类型，包括 CH 和 EN 两种|
| `chTemplate`    | object    | 非必填     | -     | Agent ReAct 中文模板                         |
| `enTemplate`    | object    | 非必填     | -     | Agent ReAct 英文模板                         |

`chTemplate`和`enTemplate`的配置字段说明如下：

| 名称            | 数据类型   | 填写要求   | 默认值 | 描述                                         |
|-----------------|-----------|-----------|--------|---------------------------------------------|
| `question`      | string    | 非必填     | -      | Agent ReAct 模板的 question 部分             |
| `thought1`      | string    | 非必填     | -      | Agent ReAct 模板的 thought1 部分             |
| `actionInput`   | string    | 非必填     | -      | Agent ReAct 模板的 actionInput 部分          |
| `observation`   | string    | 非必填     | -      | Agent ReAct 模板的 observation 部分          |
| `thought2`      | string    | 非必填     | -      | Agent ReAct 模板的 thought2 部分             |
| `finalAnswer`   | string    | 非必填     | -      | Agent ReAct 模板的 finalAnswer 部分          |
| `begin`         | string    | 非必填     | -      | Agent ReAct 模板的 begin 部分                |

## 用法示例

**配置信息**

```yaml
llm:
  apiKey: xxxxxxxxxxxxxxxxxx
  domain: dashscope.aliyuncs.com
  serviceName: dashscope.dns
  servicePort: 443
  path: /compatible-mode/v1/chat/completions
  model: qwen-max-0403
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
      title: 高德地图
      description: 获取 POI 的相关信息
      version: v1.0.0
    servers:
      - url: https://restapi.amap.com
    paths:
      /v5/place/text:
        get:
          description: 根据POI名称，获得POI的经纬度坐标
          operationId: get_location_coordinate
          parameters:
            - name: keywords
              in: query
              description: POI名称，必须是中文
              required: true
              schema:
                type: string
            - name: region
              in: query
              description: POI所在的区域名，必须是中文
              required: true
              schema:
                type: string
          deprecated: false
      /v5/place/around:
        get:
          description: 搜索给定坐标附近的POI
          operationId: search_nearby_pois
          parameters:
            - name: keywords
              in: query
              description: 目标POI的关键字
              required: true
              schema:
                type: string
            - name: location
              in: query
              description: 中心点的经度和纬度，用逗号隔开
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
      title: 心知天气
      description: 获取 天气预办相关信息
      version: v1.0.0
    servers:
      - url: https://api.seniverse.com
    paths:
      /v3/weather/now.json:
        get:
          description: 获取指定城市的天气实况
          operationId: get_weather_now
          parameters:
            - name: location
              in: query
              description: 所查询的城市
              required: true
              schema:
                type: string
            - name: language
              in: query
              description: 返回天气查询结果所使用的语言
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
              description: 表示温度的的单位，有摄氏度和华氏度两种
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
```

本示例配置了两个服务，一个是高德地图，另一个是心知天气，两个服务都需要现在Higress的服务中以DNS域名的方式配置好，并确保健康。
高德地图提供了两个工具，分别是获取指定地点的坐标，以及搜索坐标附近的感兴趣的地点。文档：https://lbs.amap.com/api/webservice/guide/api-advanced/newpoisearch
心知天气提供了一个工具，用于获取指定城市的实时天气情况，支持中文，英文，日语返回，以及摄氏度和华氏度的表示。文档：https://seniverse.yuque.com/hyper_data/api_v3/nyiu3t


以下为测试用例，为了效果的稳定性，建议保持大模型版本的稳定，本例子中使用的qwen-max-0403：

**请求示例**

```shell
curl 'http://<这里换成网关公网IP>/api/openai/v1/chat/completions' \
-H 'Accept: application/json, text/event-stream' \
-H 'Content-Type: application/json' \
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"我想在济南市鑫盛大厦附近喝咖啡，给我推荐几个"}],"presence_penalty":0,"temperature":0,"top_p":0}'
```

**响应示例**

```json
{"id":"139487e7-96a0-9b13-91b4-290fb79ac992","choices":[{"index":0,"message":{"role":"assistant","content":" 在济南市鑫盛大厦附近，您可以选择以下咖啡店：\n1. luckin coffee 瑞幸咖啡(鑫盛大厦店)，位于新泺大街1299号鑫盛大厦2号楼大堂；\n2. 三庆齐盛广场挪瓦咖啡(三庆·齐盛广场店)，位于新泺大街与颖秀路交叉口西南60米；\n3. luckin coffee 瑞幸咖啡(三庆·齐盛广场店)，位于颖秀路1267号；\n4. 库迪咖啡(齐鲁软件园店)，位于新泺大街三庆齐盛广场4号楼底商；\n5. 库迪咖啡(美莲广场店)，位于高新区新泺大街1166号美莲广场L117号；以及其他一些选项。希望这些建议对您有所帮助！"},"finish_reason":"stop"}],"created":1723172296,"model":"qwen-max-0403","object":"chat.completion","usage":{"prompt_tokens":886,"completion_tokens":50,"total_tokens":936}}
```

**请求示例**

```shell
curl 'http://<这里换成网关公网IP>/api/openai/v1/chat/completions' \
-H 'Accept: application/json, text/event-stream' \
-H 'Content-Type: application/json' \
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"济南市现在的天气情况如何？"}],"presence_penalty":0,"temperature":0,"top_p":0}'
```

**响应示例**

```json
{"id":"ebd6ea91-8e38-9e14-9a5b-90178d2edea4","choices":[{"index":0,"message":{"role":"assistant","content":" 济南市现在的天气状况为阴天，温度为31℃。此信息最后更新于2024年8月9日15时12分（北京时间）。"},"finish_reason":"stop"}],"created":1723187991,"model":"qwen-max-0403","object":"chat.completion","usage":{"prompt_tokens":890,"completion_tokens":56,"total_tokens":946}}
```

**请求示例**

```shell
curl 'http://<这里换成网关公网IP>/api/openai/v1/chat/completions' \
-H 'Accept: application/json, text/event-stream' \
-H 'Content-Type: application/json' \
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"济南市现在的天气情况如何？用华氏度表示，用日语回答"}],"presence_penalty":0,"temperature":0,"top_p":0}'
```

**响应示例**

```json
{"id":"ebd6ea91-8e38-9e14-9a5b-90178d2edea4","choices":[{"index":0,"message":{"role":"assistant","content":" 济南市の現在の天気は雨曇りで、気温は88°Fです。この情報は2024年8月9日15時12分（東京時間）に更新されました。"},"finish_reason":"stop"}],"created":1723187991,"model":"qwen-max-0403","object":"chat.completion","usage":{"prompt_tokens":890,"completion_tokens":56,"total_tokens":946}}
```
