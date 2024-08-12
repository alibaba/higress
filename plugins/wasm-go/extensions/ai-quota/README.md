# 功能说明

`ai-qutoa`插件实现给特定 consumer 根据分配固定的 quota 进行 quota 策略限流，同时支持 quota 管理能力，包括查询 quota 、刷新 quota、增减 quota。



# 配置说明

| 名称                 | 数据类型            | 填写要求                                 | 默认值 | 描述                                         |
|--------------------|-----------------|--------------------------------------| ---- |--------------------------------------------|
| `consumers`        | array of object | 必填                                   | -    | 配置服务的调用者，用于对请求进行认证                         |
| `keys`             | array of string | 必填                                   | -    | credential 的来源字段名称，可以是 URL 参数或者 HTTP 请求头名称 |
| `in_query`         | bool            | `in_query` 和 `in_header` 至少有一个为 true | true | 配置 true 时，网关会尝试从 URL 参数中解析 credential      |
| `in_header`        | bool            | `in_query` 和 `in_header` 至少有一个为 true | true | 配置 true 时，网关会尝试从 HTTP 请求头中解析 credential    |
| `redis_key_prefix` | string          |  选填                                     |   chat_quota:   | qutoa redis key 前缀                         |
| `admin_consumer`   | string          | 必填                                   |      | 管理 quota 管理身份的 consumer 名称                 |
| `admin_path`       | string          | 选填                                   |   /quota   | 管理 quota 请求 path 前缀                        |
| `redis`            | object          | 是                                    |      | redis相关配置                                  |


`consumers`中每一项的配置字段说明如下：

| 名称         | 数据类型 | 填写要求 | 默认值 | 描述                     |
| ------------ | -------- | -------- | ------ | ------------------------ |
| `credential` | string   | 必填     | -      | 配置该consumer的访问凭证 |
| `name`       | string   | 必填     | -      | 配置该consumer的名称     |



`redis`中每一项的配置字段说明

| 配置项       | 类型   | 必填 | 默认值                                                     | 说明                        |
| ------------ | ------ | ---- | ---------------------------------------------------------- | --------------------------- |
| service_name | string | 必填 | -                                                          | redis 服务名称，带服务类型的完整 FQDN 名称，例如 my-redis.dns、redis.my-ns.svc.cluster.local     |
| service_port | int    | 否   | 服务类型为固定地址（static service）默认值为80，其他为6379 | 输入redis服务的服务端口     |
| username     | string | 否   | -                                                          | redis用户名                 |
| password     | string | 否   | -                                                          | redis密码                   |
| timeout      | int    | 否   | 1000                                                       | redis连接超时时间，单位毫秒 |



# 配置示例

## 识别请求参数 apikey，进行区别限流
```yaml
consumers:
- credential: "Bearer credential1"
  name: consumer1
- credential: "Bearer credential2"
  name: consumer2
- credential: "Bearer credential3"
  name: consumer3
keys:
- authorization
in_header: true
redis_key_prefix: "chat_quota:"
admin_consumer: consumer3
admin_path: /quota
redis:
  service_name: redis-service.default.svc.cluster.local
  service_port: 6379
  timeout: 2000
```

##  刷新 quota

如果当前请求 url 的后缀符合 admin_path，例如插件在 example.com/v1/chat/completions 这个路由上生效，那么更新 quota 可以通过
curl https://example.com/v1/chat/completions/quota/refresh -H "Authorization: Bearer credential3" -d "consumer=consumer1&quota=10000" 

Redis 中 key 为 chat_quota:consumer1 的值就会被刷新为 10000

## 查询 quota

查询特定用户的 quota 可以通过 curl https://example.com/v1/chat/completions/quota?consumer=consumer1 -H "Authorization: Bearer credential3"
将返回： {"quota": 10000, "consumer": "consumer1"}

## 增减 quota 

增减特定用户的 quota 可以通过 curl https://example.com/v1/chat/completions/quota/delta -d "consumer=consumer1&value=100" -H "Authorization: Bearer credential3"
这样 Redis 中 Key 为 chat_quota:consumer1 的值就会增加100，可以支持负数，则减去对应值。

