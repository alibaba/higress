---
title: APISIX HMAC 认证
keywords: [higress,hmac auth,apisix]
description: APISIX HMAC 认证插件配置参考
---

## 功能说明

`hmac-auth-apisix` 插件兼容 Apache APISIX 的 HMAC 认证机制，通过 HMAC 算法为 HTTP 请求生成防篡改的数字签名，实现请求的身份认证和权限控制。该插件完全兼容 Apache APISIX HMAC 认证插件的配置和签名算法，签名生成方法可参考 [Apache APISIX HMAC 认证文档](https://apisix.apache.org/docs/apisix/plugins/hmac-auth/)

## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`330`

## 配置字段

**注意：**

- 在一个规则里，鉴权配置和认证配置不可同时存在
- 对于通过认证鉴权的请求，请求的 header 会被添加一个 `X-Mse-Consumer` 字段，用以标识调用者的名称

### 认证配置

| 名称                    | 数据类型        | 填写要求                   | 默认值                                      | 描述                                                         |
| ----------------------- | --------------- | -------------------------- | ------------------------------------------- | ------------------------------------------------------------ |
| `global_auth`           | bool            | 选填（**仅实例级别配置**） | -                                           | 只能在实例级别配置，若配置为 true，则全局生效认证机制；若配置为 false，则只对做了配置的域名和路由生效认证机制，若不配置则仅当没有域名和路由配置时全局生效（兼容老用户使用习惯） |
| `consumers`             | array of object | 必填                       | -                                           | 配置服务的调用者，用于对请求进行认证                         |
| `allowed_algorithms`    | array of string | 选填                       | ["hmac-sha1", "hmac-sha256", "hmac-sha512"] | 允许的 HMAC 算法列表。有效值为 "hmac-sha1"、"hmac-sha256" 和 "hmac-sha512" 的组合 |
| `clock_skew`            | number          | 选填                       | 300                                         | 客户端请求的时间戳与 Higress 服务器当前时间之间允许的最大时间差（以秒为单位）。这有助于解决客户端和服务器之间的时间同步差异，并防止重放攻击。时间戳将根据 Date 头中的时间（必须为 GMT 格式）进行计算。如果配置为0，会跳过该校验 |
| `signed_headers`        | array of string | 选填                       | -                                           | 客户端请求的 HMAC 签名中应包含的 HMAC 签名头列表             |
| `validate_request_body` | boolean         | 选填                       | false                                       | 如果为 true，则验证请求正文的完整性，以确保在传输过程中没有被篡改。具体来说，插件会创建一个 SHA-256 的 base64 编码 digest，并将其与 `Digest` 头进行比较。如果 `Digest` 头丢失或 digest 不匹配，验证将失败 |
| `hide_credentials`      | boolean         | 选填                       | false                                       | 如果为 true，则不会将授权请求头传递给上游服务                |
| `anonymous_consumer`    | string          | 选填                       | -                                           | 匿名消费者名称。如果已配置，则允许匿名用户绕过身份验证       |


`consumers`中每一项的配置字段说明如下：

| 名称         | 数据类型 | 填写要求 | 默认值       | 描述                                           |
| ------------ | -------- | -------- | ------------ | ---------------------------------------------- |
| `access_key` | string   | 必填     | -            | 消费者的唯一标识符，用于标识相关配置，例如密钥 |
| `secret_key` | string   | 必填     | -            | 用于生成 HMAC 的密钥                           |
| `name`       | string   | 选填     | `access_key` | 配置该 consumer 的名称                         |

### 鉴权配置（非必需）

| 名称    | 数据类型        | 填写要求                 | 默认值 | 描述                                                         |
| ------- | --------------- | ------------------------ | ------ | ------------------------------------------------------------ |
| `allow` | array of string | 选填(**非实例级别配置**) | -      | 只能在路由或域名等细粒度规则上配置，对于符合匹配条件的请求，配置允许访问的 consumer，从而实现细粒度的权限控制 |

## 配置示例

### 全局配置认证和路由粒度鉴权

以下配置用于对网关特定路由或域名开启 Hmac Auth 认证和鉴权。**注意：access_key 字段不可重复**

#### 示例1：基础路由与域名鉴权配置

**实例级别插件配置**：
```yaml
global_auth: false
consumers:
- name: consumer1
  access_key: consumer1-key
  secret_key: 2bda943c-ba2b-11ec-ba07-00163e1250b5
- name: consumer2
  access_key: consumer2-key
  secret_key: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
```

**路由级配置**（适用于 route-a 和 route-b）：
```yaml
allow: 
- consumer1  # 仅允许consumer1访问
```

**域名级配置**（适用于 `*.example.com` 和 `test.com`）：
```yaml
allow:
- consumer2  # 仅允许consumer2访问
```

**配置说明**：
- 路由名称（如 route-a、route-b）对应网关路由创建时定义的名称，匹配时仅允许consumer1访问
- 域名匹配（如 `*.example.com`、`test.com`）用于过滤请求域名，匹配时仅允许consumer2访问
- 未在allow列表中的调用者将被拒绝访问

**请求与响应示例**：

1. **验证通过场景**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date",signature="G2+60rCCHQCQDZOailnKHLCEy++P1Pa5OEP1bG4QlRo="' \
-H 'Date:Sat, 30 Aug 2025 00:52:39 GMT' \
-H 'Content-Type: application/json' \
-d '{}'
```
- 响应：返回后端服务正常响应
- 附加信息：认证通过后会自动添加请求头 `X-Mse-Consumer: consumer1` 传递给后端

2. **请求方法修改导致验签失败**
```shell
curl -X PUT 'http://localhost:8082/foo' \  # 此处将POST改为PUT
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date",signature="G2+60rCCHQCQDZOailnKHLCEy++P1Pa5OEP1bG4QlRo="' \
-H 'Date:Sat, 30 Aug 2025 00:52:39 GMT' \
-H 'Content-Type: application/json' \
-d '{}'
```
- 响应：`401 Unauthorized`
- 错误信息：`{"message":"client request can't be validated: Invalid signature"}`

3. **不在允许列表中的调用者**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer2-key",algorithm="hmac-sha256",headers="@request-target date",signature="5sqSbDX9b91dQsfQra2hpluM7O6/yhS7oLcKPQylyCo="' \
-H 'Date:Sat, 30 Aug 2025 00:54:18 GMT' \
-H 'Content-Type: application/json' \
-d '{}'
```
- 响应：`401 Unauthorized`
- 错误信息：`{"message":"client request can't be validated: consumer 'consumer2' is not allowed"}`

4. **时间戳过期**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization: Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date",signature="gvIUwoYNiK57w6xX2g1Ntpk8lfgD7z+jgom434r5qwg="' \
-H 'Date: Sat, 30 Aug 2025 00:40:21 GMT' \  # 过期的时间戳
-H 'Content-Type: application/json' \
-d '{}'
```
- 响应：`401 Unauthorized`
- 错误信息：`{"message":"client request can't be validated: Clock skew exceeded"}`

#### 示例2：带自定义签名头与请求体验证的配置

**实例级别插件配置**：
```yaml
global_auth: false
consumers:
- name: consumer1
  access_key: consumer1-key
  secret_key: 2bda943c-ba2b-11ec-ba07-00163e1250b5
- name: consumer2
  access_key: consumer2-key
  secret_key: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
signed_headers:  # 需要纳入签名的自定义请求头
- X-Custom-Header-A
- X-Custom-Header-B
validate_request_body: true  # 启用请求体签名校验
```

**请求与响应示例**：

1. **验证通过场景**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date x-custom-header-a x-custom-header-b",signature="+xCWYCmidq3Sisn08N54NWaau5vSY9qEanWoO9HD4mA="' \
-H 'Date:Sat, 30 Aug 2025 01:04:06 GMT' \
-H 'Digest:SHA-256=RBNvo1WzZ4oRRq0W9+hknpT7T8If536DEMBg9hyq/4o=' \  # 请求体摘要
-H 'X-Custom-Header-A:test1' \
-H 'X-Custom-Header-B:test2' \
-H 'Content-Type: application/json' \
-d '{}'
```
- 响应：返回后端服务正常响应

2. **缺少签名头**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date x-custom-header-a x-custom-header-b",signature="+xCWYCmidq3Sisn08N54NWaau5vSY9qEanWoO9HD4mA="' \
-H 'Date:Sat, 30 Aug 2025 01:04:06 GMT' \
-H 'Digest:SHA-256=RBNvo1WzZ4oRRq0W9+hknpT7T8If536DEMBg9hyq/4o=' \
-H 'X-Custom-Header-B:test2' \  # 缺少X-Custom-Header-A
-H 'Content-Type: application/json' \
-d '{}'
```
- 响应：`401 Unauthorized`
- 错误信息：`{"message":"client request can't be validated: expected header "X-Custom-Header-A" missing in signing"}`

3. **请求体被篡改**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date x-custom-header-a x-custom-header-b",signature="dSbv6pdQOcgkN89TmSxiT8F9nypbPUqAR2E7ELL8K2s="' \
-H 'Date:Sat, 30 Aug 2025 01:10:17 GMT' \
-H 'Digest:SHA-256=RBNvo1WzZ4oRRq0W9+hknpT7T8If536DEMBg9hyq/4o=' \  # 与实际body不匹配
-H 'X-Custom-Header-A:test1' \
-H 'X-Custom-Header-B:test2' \
-H 'Content-Type: application/json' \
-d '{"key":"value"}'  # 篡改后的请求体
```
- 响应：`401 Unauthorized`
- 错误信息：`{"message":"client request can't be validated: Invalid digest"}`

### 网关实例级别开启全局认证

以下配置将在网关实例级别开启 Hmac Auth 认证，**所有请求必须经过认证才能访问**：

```yaml
global_auth: true  # 开启全局认证
consumers:
- name: consumer1
  access_key: consumer1-key
  secret_key: 2bda943c-ba2b-11ec-ba07-00163e1250b5
- name: consumer2
  access_key: consumer2-key
  secret_key: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
```

**说明**：当 `global_auth: true` 时，所有访问网关的请求都需要携带有效的认证信息，未认证的请求将被直接拒绝