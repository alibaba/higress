# 简介
一个可定制化的API AI Agent，目前第一版本只支持配置http method类型为GET的API。agent执行的流程图如下：
![未命名绘图 drawio](https://github.com/user-attachments/assets/7d2e5b23-99f0-4e80-8524-c4ffa523ddbe)


# 配置说明
| 名称             | 数据类型            | 填写要求 | 默认值 | 描述                                                                               |
|----------------|-----------------|------|-----|----------------------------------------------------------------------------------|
| `dashscope.apiKey` | string | 必填 | - | 用于在访问通义千问服务时进行认证的令牌。 |
| `dashscope.serviceName` | string | 必填 | - | 通义千问服务名 |
| `dashscope.servicePort` | int | 必填 | - | 通义千问服务端口 |
| `dashscope.domain` | string | 必填 | - | 访问通义千问服务时域名 |
| `toolsClientInfo.apiKey` | string | 必填 | - | 用于在访问外部API服务时进行认证的令牌。 |
| `toolsClientInfo.serviceName` | string | 必填 | - | 访问外部API服务名 |
| `toolsClientInfo.servicePort` | int | 必填 | - | 访问外部API服务端口 |
| `toolsClientInfo.domain` | string | 必填 | - | 访访问外部API时域名 |
| `tools.title` | string | 必填 | - | api的服务名,与`toolsClientInfo.serviceName`的内容保持一致 |
| `tools.description_for_model` | string | 必填 | - | 工具的描述 |
| `tools.name_for_model` | string | 必填 | - | 给模型使用的工具名称（工具函数名） |
| `tools.parameters` | string | 必填 | - | 使用OpenAPI规范中的parameters的表示格式表示的工具入参，类型，作用，默认值等，使用yaml格式 |
| `tools.url` | string | 必填 | - | 访问外部API服务时的url，需按照parameters定义的参数顺序，将query参数，拼接到url中 |
| `tools.method` | string | 必填 | - | 访问外部API服务时的http method |

# 示例

```yaml
toolsClientInfo:
- domain: restapi.amap.com #高德地图服务
  serviceName: geo
  servicePort: 80
  apiKey: xxxxxxxxxxxxxxxxxxxxxxx
- domain: api.seniverse.com #心知天气服务
  serviceName: seniverse
  servicePort: 80
  apiKey: xxxxxxxxxxxxx
dashscope:
  apiKey: xxxxxxxxxxxxxxxxxx
  domain: dashscope.aliyuncs.com
  serviceName: dashscope
  servicePort: 443
tools:
- title: "geo"
  description_for_model: "根据POI名称，获得POI的经纬度坐标"
  name_for_model: "get_location_coordinate"
  parameters: |
    parameters:
    - name: apiKey
      in: query
      description: 高德地图的apikey，默认是""
      required: true
      schema:
        type: string
    - name: keywords
      in: query
      description: POI名称，必须是中文
      required: true
      schema:
        type: string
    - name: region
      in: query
      description: POI所在的城市名，必须是中文
      required: true
      schema:
        type: string
  url: "/v5/place/text?key=%s&keywords=%s&region=%s"
  method: "get"
- title: "geo"
  description_for_model: "根据给定的给定坐标附近的POI"
  name_for_model: "search_nearby_pois"
  parameters: |
    parameters:
    - name: apiKey
      in: query
      description: 高德地图的apikey，默认是""
      required: true
      schema:
        type: string
    - name: keywords
      in: query
      description: 目标POI的关键字
      required: true
      schema:
        type: string
    - name: longitude
      in: query
      description: 中心点的经度
      required: true
      schema:
        type: string
    - name: latitude
      in: query
      description: 中心点的纬度
      required: true
      schema:
        type: string
  url: "/v5/place/around?key=%s&keywords=%s&location=%s,%s"
  method: "get"
- title: "seniverse"
  description_for_model: "获取指定城市的天气实况"
  name_for_model: "get_weather_now"
  parameters: |
    parameters:
    - name: apiKey
      in: query
      description: 获取天气实况服务的apikey，默认是""
      required: true
      schema:
        type: string
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
          - zh-Hans #简体中文
          - en #英文
          - ja #日语
    - name: unit
      in: query
      description: 表示温度的的单位，有摄氏度和华氏度两种
      required: true
      schema:
        type: string
        default: c #默认是摄氏度，下面的列表是支持的温度单位，比如摄氏度，则取值c
        enum:
          - c #摄氏度
          - f #华氏度
  url: "/v3/weather/now.json?key=%s&location=%s&language=%s&unit=%s"
  method: "get"
```

本示例配置了两个服务，一个是高德地图，另一个是心知天气，两个服务都需要现在Higress的服务中以DNS域名的方式配置好，并确保健康。
高德地图提供了两个工具，分别是获取指定地点的坐标，以及搜索坐标附近的感兴趣的地点。文档：https://lbs.amap.com/api/webservice/guide/api-advanced/newpoisearch
心知天气提供了一个工具，用于获取指定城市的实时天气情况，支持中文，英文，日语返回，以及摄氏度和华氏度的表示。文档：https://seniverse.yuque.com/hyper_data/api_v3/nyiu3t


以下为测试用例：
```
Example request
```
```shell
curl 'http://<这里换成网关公网IP>/api/openai/v1/chat/completions' \
-H 'Accept: application/json, text/event-stream' \
-H 'Content-Type: application/json' \
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"我想在济南市鑫盛大厦附近喝咖啡，给我推荐几个"}],"presence_penalty":0,"temperature":0,"top_p":0}'
```

```
Example response
```
```json
{"id":"139487e7-96a0-9b13-91b4-290fb79ac992","choices":[{"index":0,"message":{"role":"assistant","content":" 在济南市鑫盛大厦附近，您可以选择以下咖啡店：\n1. luckin coffee 瑞幸咖啡(鑫盛大厦店)，位于新泺大街1299号鑫盛大厦2号楼大堂；\n2. 三庆齐盛广场挪瓦咖啡(三庆·齐盛广场店)，位于新泺大街与颖秀路交叉口西南60米；\n3. luckin coffee 瑞幸咖啡(三庆·齐盛广场店)，位于颖秀路1267号；\n4. 库迪咖啡(齐鲁软件园店)，位于新泺大街三庆齐盛广场4号楼底商；\n5. 库迪咖啡(美莲广场店)，位于高新区新泺大街1166号美莲广场L117号；以及其他一些选项。希望这些建议对您有所帮助！"},"finish_reason":"stop"}],"created":1723172296,"model":"qwen-plus","object":"chat.completion","usage":{"prompt_tokens":886,"completion_tokens":50,"total_tokens":936}}
```

```
Example request
```
```shell
curl 'http://<这里换成网关公网IP>/api/openai/v1/chat/completions' \
-H 'Accept: application/json, text/event-stream' \
-H 'Content-Type: application/json' \
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"济南市现在的天气情况如何？"}],"presence_penalty":0,"temperature":0,"top_p":0}'
```

```
Example response
```
```json
{"id":"ebd6ea91-8e38-9e14-9a5b-90178d2edea4","choices":[{"index":0,"message":{"role":"assistant","content":" 济南市现在的天气状况为阴天，温度为31℃。此信息最后更新于2024年8月9日15时12分（北京时间）。"},"finish_reason":"stop"}],"created":1723187991,"model":"qwen-plus","object":"chat.completion","usage":{"prompt_tokens":890,"completion_tokens":56,"total_tokens":946}}
```

```
Example request
```
```shell
curl 'http://<这里换成网关公网IP>/api/openai/v1/chat/completions' \
-H 'Accept: application/json, text/event-stream' \
-H 'Content-Type: application/json' \
--data-raw '{"model":"qwen","frequency_penalty":0,"max_tokens":800,"stream":false,"messages":[{"role":"user","content":"济南市现在的天气情况如何？用华氏度表示，用日语回答"}],"presence_penalty":0,"temperature":0,"top_p":0}'
```

```
Example response
```
```json
{"id":"ebd6ea91-8e38-9e14-9a5b-90178d2edea4","choices":[{"index":0,"message":{"role":"assistant","content":" 济南市の現在の天気は雨曇りで、気温は88°Fです。この情報は2024年8月9日15時12分（東京時間）に更新されました。"},"finish_reason":"stop"}],"created":1723187991,"model":"qwen-plus","object":"chat.completion","usage":{"prompt_tokens":890,"completion_tokens":56,"total_tokens":946}}
```
