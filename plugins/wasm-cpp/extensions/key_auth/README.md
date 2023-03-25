# 功能说明
`key-auth`插件实现了基于 API Key 进行认证鉴权的功能，支持从 HTTP 请求的 URL 参数或者请求头解析 API Key，同时验证该 API Key 是否有权限访问。

# 配置字段

| 名称        | 数据类型        | 填写要求                                    | 默认值 | 描述                                                        |
| ----------- | --------------- | ------------------------------------------- | ------ | ----------------------------------------------------------- |
| `consumers` | array of object | 必填                                        | -      | 配置服务的调用者，用于对请求进行认证                        |
| `keys`      | array of string | 必填                                        | -      | API Key 的来源字段名称，可以是 URL 参数或者 HTTP 请求头名称 |
| `in_query`  | bool            | `in_query` 和 `in_header` 至少有一个为 true | true   | 配置 true 时，网关会尝试从 URL 参数中解析 API Key           |
| `in_header` | bool            | `in_query` 和 `in_header` 至少有一个为 true | true   | 配置 true 时，网关会尝试从 HTTP 请求头中解析 API Key        |
| `_rules_`   | array of object | 选填                                        | -      | 配置特定路由或域名的访问权限列表，用于对请求进行鉴权        |

`consumers`中每一项的配置字段说明如下：

| 名称         | 数据类型 | 填写要求 | 默认值 | 描述                     |
| ------------ | -------- | -------- | ------ | ------------------------ |
| `credential` | string   | 必填     | -      | 配置该consumer的访问凭证 |
| `name`       | string   | 必填     | -      | 配置该consumer的名称     |

`_rules_` 中每一项的配置字段说明如下：

| 名称             | 数据类型        | 填写要求                                          | 默认值 | 描述                                               |
| ---------------- | --------------- | ------------------------------------------------- | ------ | -------------------------------------------------- |
| `_match_route_`  | array of string | 选填，`_match_route_`，`_match_domain_`中选填一项 | -      | 配置要匹配的路由名称                               |
| `_match_domain_` | array of string | 选填，`_match_route_`，`_match_domain_`中选填一项 | -      | 配置要匹配的域名                                   |
| `allow`          | array of string | 必填                                              | -      | 对于符合匹配条件的请求，配置允许访问的consumer名称 |

**注意：**
- 若不配置`_rules_`字段，则默认对当前网关实例的所有路由开启认证；
- 对于通过认证鉴权的请求，请求的header会被添加一个`X-Mse-Consumer`字段，用以标识调用者的名称。

# 配置示例

## 对特定路由或域名开启

以下配置将对网关特定路由或域名开启 Key Auth 认证和鉴权，注意`credential`字段不能重复

```yaml
consumers:
- credential: 2bda943c-ba2b-11ec-ba07-00163e1250b5
  name: consumer1
- credential: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
  name: consumer2
keys:
- apikey
in_query: true
# 使用 _rules_ 字段进行细粒度规则配置
_rules_:
# 规则一：按路由名称匹配生效
- _match_route_:
  - route-a
  - route-b
  allow:
  - consumer1
# 规则二：按域名匹配生效
- _match_domain_:
  - "*.example.com"
  - test.com
  allow:
  - consumer2
```

此例 `_match_route_` 中指定的 `route-a` 和 `route-b` 即在创建网关路由时填写的路由名称，当匹配到这两个路由时，将允许`name`为`consumer1`的调用者访问，其他调用者不允许访问；

此例 `_match_domain_` 中指定的 `*.example.com` 和 `test.com` 用于匹配请求的域名，当发现域名匹配时，将允许`name`为`consumer2`的调用者访问，其他调用者不允许访问。

### 根据该配置，下列请求可以允许访问：

假设以下请求会匹配到route-a这条路由

**将 API Key 设置在 url 参数中**
```bash
curl  http://xxx.hello.com/test?apikey=2bda943c-ba2b-11ec-ba07-00163e1250b5
```
**将 API Key 设置在 http 请求头中**
```bash
curl  http://xxx.hello.com/test -H 'x-api-key: 2bda943c-ba2b-11ec-ba07-00163e1250b5'
```

认证鉴权通过后，请求的header中会被添加一个`X-Mse-Consumer`字段，在此例中其值为`consumer1`，用以标识调用方的名称

### 下列请求将拒绝访问：

**请求未提供 API Key，返回401**
```bash
curl  http://xxx.hello.com/test
```
**请求提供的 API Key 无权访问，返回401**
```bash
curl  http://xxx.hello.com/test?apikey=926d90ac-ba2e-11ec-ab68-00163e1250b5
```

**根据请求提供的 API Key匹配到的调用者无访问权限，返回403**
```bash
# consumer2不在route-a的allow列表里
curl  http://xxx.hello.com/test?apikey=c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
```

## 网关实例级别开启

以下配置未指定`_rules_`字段，因此将对网关实例级别开启 Key Auth 认证

```yaml
consumers:
- credential: 2bda943c-ba2b-11ec-ba07-00163e1250b5
  name: consumer1
- credential: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
  name: consumer2
keys:
- apikey
in_query: true
```

# 相关错误码

| HTTP 状态码 | 出错信息                                                  | 原因说明                |
| ----------- | --------------------------------------------------------- | ----------------------- |
| 401         | No API key found in request                               | 请求未提供 API Key      |
| 401         | Request denied by Key Auth check. Invalid API key         | 不允许当前 API Key 访问 |
| 403         | Request denied by Basic Auth check. Unauthorized consumer | 请求的调用方无访问权限  |
