## 功能说明

`mocking` 插件用于模拟 API。当执行该插件时，它将返回符合匹配条件指定格式的模拟数据，并且请求不会转发到上游

## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`205`

## 配置说明

| 配置项                     | 类型   | 必填 | 默认值               | 说明                                                                          |
|-------------------------| ------ |----|-------------------|-----------------------------------------------------------------------------|
| responses               | array of object | 是  | -                 | mocking插件响应集,可以指定多条件响应                                                      |
| with_mock_header        | bool | 否  | true              | 当设置为 true 时，将添加响应头 x-mock-by: higress。设置为 false 时则不添加该响应头                   |


`responses` 中每一项的配置字段说明。

| 配置项                | 类型              | 必填 | 默认值 | 说明                                                                                  |
| --------------------- |-----------------|----|---|-------------------------------------------------------------------------------------|
| trigger       | object          | 否  | - | 匹配条件集                                                                               |
| body        | string          | 是  | {"hello":"world"}  | 响应客户端的response body                                                                 |
| headers     | array of object | 否  | [{"key":"content-type","value":"application/json"}]  | 响应客户端的response headers                                                              |
| status_code       | int             | 否  | 200 | 响应客户端的http code                                                                     |

`trigger` 中每一项的配置字段说明。

| 配置项           | 类型   | 必填                                                                             | 默认值 | 说明                      |
| ---------------- | ------ |--------------------------------------------------------------------------------| ------ |-------------------------|
| headers              | array of object | 否                                                                              | -      | 匹配的request headers      |
| queries | array of object    | array of object | 否      | 匹配的request query params |

`trigger.headers` 中每一项的配置字段说明。

| 配置项       | 类型     | 必填 | 默认值 | 说明       |
| ------------ |--------|----|-----|----------|
| key | string | 否  | -   | 请求头key   |
| value | string | 否  | -   | 请求头value |

`trigger.queries` 中每一项的配置字段说明。

| 配置项       | 类型     | 必填 | 默认值 | 说明        |
| ------------ |--------|--|-----|-----------|
| key | string | 否 | -   | 请求参数key   |
| value | string | 否 | -   | 请求参数value |

`response.headers` 中每一项的配置字段说明。

| 配置项       | 类型     | 必填 | 默认值 | 说明         |
| ------------ |--------|----|-----|------------|
| key | string | 否  | -   | 新增响应头 key  |
| value | string | 否 | -   | 新增响应头value |

## 配置示例

### 插件示例配置

```yaml
responses:
  -
    trigger:
      headers:
        -
          key: header1
          value: value1
      queries:
        -
          key: queryKey1
          value: queryValue1
    body: "test"
    headers:
      -
        key: "content-type"
        value: "text/plain"
    status_code: 200
```
若trigger中的指定条件都没有满足,则会返回默认的消息体`{"hello":"world"}`和响应header`content-type:application/json`
