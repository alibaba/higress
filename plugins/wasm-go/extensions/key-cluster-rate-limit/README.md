# 功能说明

` key-cluster-rate-limit`根据客户端IP地址实现集群限流功能



# 配置说明

| 配置项                  | 类型   | 必填 | 默认值 | 说明 |
| ----------------------- | ------ | ---- | ------ | ---- |
| rule_name               | string | 是 | - | 限流规则名称，根据限流规则名称和限流的客户端IP段来拼装redis key |
| ip_source_type          | string | 否 | origin-source | 可选值：1. `origin-source`：对端socket ip 2. `header`：通过header获取 |
| ip_header_name          | string | 否 | x-forwarded-for | 当`ip_source_type`为`header`时，指定自定义IP来源头 |
| limit_keys              | array of object | 是 | - | 配置匹配客户端IP后的限流次数 |
| show_limit_quota_header | bool | 否 | false | 响应头中是否显示`X-RateLimit-Limit`（限制的总请求数）和`X-RateLimit-Remaining`（剩余还可以发送的请求数） |
| rejected_code           | int | 否 | 429 | 请求被限流时，返回的HTTP状态码 |
| rejected_msg            | string | 否 | Too many requests | 请求被限流时，返回的响应体 |
| redis_service_source | string | 必填   | -          | 类型为固定ip或者DNS，输入redis服务的注册来源                             |
| redis_service_name | string | 必填   | -          | 输入redis服务的注册名称                                        |
| redis_service_port | int    | 必填   | -          | 输入redis服务的服务端口                                        |
| redis_service_host | string | 必填   | -          | 当类型为固定ip时必须填写，输入redis服务的主机名                          |
| redis_service_domain | string | 必填   | -          | 当类型为DNS时必须填写，输入redis服务的domain |
| redis_username | string | 否 | - | redis用户名 |
| redis_password | string | 否 | - | redis密码 |
| redis_timeout | int | 否 | 1000 | redis连接超时时间，单位毫秒 |

`limit_keys`中每一项的配置字段说明

| 配置项           | 类型   | 必填                                                         | 默认值 | 说明                        |
| ---------------- | ------ | ------------------------------------------------------------ | ------ | --------------------------- |
| key              | string | 否                                                           | -      | 匹配的客户端IP段或IP地址    |
| query_per_second | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | 1000   | redis连接超时时间，单位毫秒 |
| query_per_minute | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每秒请求次数            |
| query_per_hour   | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每分钟请求次数          |
| query_per_day    | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每天请求次数            |



# 配置示例

## 对特定路由或域名开启
```yaml
# 使用_rules_字段进行细粒度规则配置
_rules_:
# 规则一：按路由名称匹配生效
- _match_route_:
  - route-a
  - route-b
  ip_source_type: header
  rule_name: test
  redis_service_name: redis
  redis_service_host: redis
  redis_service_port: 80
  redis_service_source: ip
  limit_keys:
    # 精确ip
    - key: 1.1.1.1
      query_per_second: 10
    # ip段，符合这个ip段的ip，每个ip 100qps
    - key: 1.1.1.0/24
      query_per_second: 100
    # 兜底用，即默认每个ip 1000qps
    - key: 0.0.0.0/0
      query_per_second: 1000  
# 规则二：按域名匹配生效
- _match_domain_:
  - "*.example.com"
  - test.com
  ip_source_type: header
  rule_name: test
  redis_service_name: redis
  redis_service_host: redis
  redis_service_port: 80
  redis_service_source: ip
  limit_keys:
    # 精确ip
    - key: 1.1.1.1
      query_per_second: 10
    # ip段，符合这个ip段的ip，每个ip 100qps
    - key: 1.1.1.0/24
      query_per_second: 100
    # 兜底用，即默认每个ip 1000qps
    - key: 0.0.0.0/0
      query_per_second: 1000   
```
此例 `_match_route_` 中指定的 `route-a` 和 `route-b` 即在创建网关路由时填写的路由名称，当匹配到这两个路由时，将使用此段配置；
此例 `_match_domain_` 中指定的 `*.example.com` 和 `test.com` 用于匹配请求的域名，当发现域名匹配时，将使用此段配置；
配置的匹配生效顺序，将按照 `_rules_` 下规则的排列顺序，匹配第一个规则后生效对应配置，后续规则将被忽略
