<p>
   <a href="README_EN.md">English</a> | 中文
</p>

# 功能说明
`basic-auth`插件实现了基于 HTTP Basic Auth 标准进行认证鉴权的功能

# 配置字段

| 名称        | 数据类型        | 填写要求 | 默认值 | 描述                                                 |
| ----------- | --------------- | -------- | ------ | ---------------------------------------------------- |
| `consumers` | array of object | 必填     | -      | 配置服务的调用者，用于对请求进行认证                 |
| `_rules_`   | array of object | 选填     | -      | 配置特定路由或域名的访问权限列表，用于对请求进行鉴权 |

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

## 对特定路由或域名开启认证和鉴权

以下配置将对网关特定路由或域名开启 Basic Auth 认证和鉴权，注意凭证信息中的用户名和密码之间使用":"分隔，`credential`字段不能重复

```yaml
# 使用 _rules_ 字段进行细粒度规则配置
consumers:
- credential: 'admin:123456'
  name: consumer1
- credential: 'guest:abc'
  name: consumer2
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

**请求指定用户名密码**

```bash
# 假设以下请求将会匹配到route-a路由
# 使用 curl 的 -u 参数指定
curl -u admin:123456  http://xxx.hello.com/test
# 或者直接指定 Authorization 请求头，用户名密码使用 base64 编码
curl -H 'Authorization: Basic YWRtaW46MTIzNDU2'  http://xxx.hello.com/test
```

认证鉴权通过后，请求的header中会被添加一个`X-Mse-Consumer`字段，在此例中其值为`consumer1`，用以标识调用方的名称

### 下列请求将拒绝访问：

**请求未提供用户名密码，返回401**
```bash
curl  http://xxx.hello.com/test
```
**请求提供的用户名密码错误，返回401**
```bash
curl -u admin:abc  http://xxx.hello.com/test
```
**根据请求的用户名和密码匹配到的调用者无访问权限，返回403**
```bash
# consumer2不在route-a的allow列表里
curl -u guest:abc  http://xxx.hello.com/test
```

## 网关实例级别开启

以下配置未指定`_rules_`字段，因此将对网关实例级别开启 Basic Auth 认证

```yaml
consumers:
- credential: 'admin:123456'
  name: consumer1
- credential: 'guest:abc'
  name: consumer2
```

# 相关错误码

| HTTP 状态码 | 出错信息                                                                       | 原因说明               |
| ----------- | ------------------------------------------------------------------------------ | ---------------------- |
| 401         | Request denied by Basic Auth check. No Basic Authentication information found. | 请求未提供凭证         |
| 401         | Request denied by Basic Auth check. Invalid username and/or password           | 请求凭证无效           |
| 403         | Request denied by Basic Auth check. Unauthorized consumer                      | 请求的调用方无访问权限 |
