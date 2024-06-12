# 功能说明

`key-cluster-rate-limit`插件实现了基于特定键值实现集群限流，键值来源可以是 URL 参数、HTTP 请求头、客户端 IP 地址



# 配置说明

| 配置项                  | 类型   | 必填 | 默认值 | 说明 |
| ----------------------- | ------ | ---- | ------ | ---- |
| rule_name               | string | 是 | - | 限流规则名称，根据限流规则名称和限流的客户端IP段来拼装redis key |
| limit_by_header         | string          | 否，`limit_by_header`,`limit_by_param`,`limit_by_per_ip` 中选填一项 | -                 | 配置获取限流键值的来源 http 请求头名称                       |
| limit_by_param          | string          | 否，`limit_by_header`,`limit_by_param`,`limit_by_per_ip` 中选填一项 | -                 | 配置获取限流键值的来源 URL 参数名称                          |
| limit_by_per_ip         | string          | 否，`limit_by_header`,`limit_by_param`,`limit_by_per_ip` 中选填一项 | -                 | 配置获取限流键值的来源 IP 参数名称，从请求头获取，以`from-header-对应的header名`，示例：`from-header-x-forwarded-for`，直接获取对端socket ip，配置为`from-remote-addr` |
| limit_keys              | array of object | 是 | - | 配置匹配键值后的限流次数 |
| show_limit_quota_header | bool | 否 | false | 响应头中是否显示`X-RateLimit-Limit`（限制的总请求数）和`X-RateLimit-Remaining`（剩余还可以发送的请求数） |
| rejected_code           | int | 否 | 429 | 请求被限流时，返回的HTTP状态码 |
| rejected_msg            | string | 否 | Too many requests | 请求被限流时，返回的响应体 |
| redis                   | object          | 是                                                           | -                 | redis相关配置                                                |

`limit_keys`中每一项的配置字段说明

| 配置项           | 类型   | 必填                                                         | 默认值 | 说明               |
| ---------------- | ------ | ------------------------------------------------------------ | ------ | ------------------ |
| key              | string | 是                                                           | -      | 匹配的键值         |
| query_per_second | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每秒请求次数   |
| query_per_minute | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每分钟请求次数 |
| query_per_hour   | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每小时请求次数 |
| query_per_day    | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每天请求次数   |

`redis`中每一项的配置字段说明

| 配置项       | 类型   | 必填 | 默认值                                                     | 说明                        |
| ------------ | ------ | ---- | ---------------------------------------------------------- | --------------------------- |
| service_name | string | 必填 | -                                                          | 输入redis服务的注册名称     |
| service_port | int    | 否   | 服务类型为固定地址（static service）默认值为80，其他为6379 | 输入redis服务的服务端口     |
| username     | string | 否   | -                                                          | redis用户名                 |
| password     | string | 否   | -                                                          | redis密码                   |
| timeout      | int    | 否   | 1000                                                       | redis连接超时时间，单位毫秒 |



# 配置示例

## 识别请求参数 apikey，进行区别限流
```yaml
rule_name: limit_by_param_apikey
limit_by_param: apikey
limit_keys:
- key: 9a342114-ba8a-11ec-b1bf-00163e1250b5
  query_per_second: 10
- key: a6a6d7f2-ba8a-11ec-bec2-00163e1250b5
  query_per_minute: 100
redis:
  service_name: redis.static
show_limit_quota_header: true
```

## 识别请求头 x-ca-key，进行区别限流
```yaml
rule_name: limit_by_param_x-ca-key
limit_by_header: x-ca-key
limit_keys:
- key: 102234
  query_per_second: 10
- key: 308239
  query_per_hour: 10
redis:
  service_name: redis.static
show_limit_quota_header: true  
```

## 根据请求头 x-forwarded-for 获取对端IP，进行区别限流

```yaml
rule_name: limit_by_per_ip_from-header-x-forwarded-for
limit_by_per_ip: from-header-x-forwarded-for
limit_keys:
	# 精确ip
- key: 1.1.1.1
  query_per_day: 10
  # ip段，符合这个ip段的ip，每个ip 100qps
- key: 1.1.1.0/24
  query_per_day: 100
  # 兜底用，即默认每个ip 1000qps
- key: 0.0.0.0/0
  query_per_day: 1000
redis:
  service_name: redis.static
show_limit_quota_header: true  
```

## 对特定路由或域名开启

```yaml
# 使用_rules_字段进行细粒度规则配置
_rules_:
# 规则一：按路由名称匹配生效
- _match_route_:
  - route-a
  - route-b
  rule_name: limit_rule1
  limit_by_per_ip: from-header-x-forwarded-for
  limit_keys:
    # 精确ip
  - key: 1.1.1.1
    query_per_day: 10
    # ip段，符合这个ip段的ip，每个ip 100qps
  - key: 1.1.1.0/24
    query_per_day: 100
    # 兜底用，即默认每个ip 1000qps
  - key: 0.0.0.0/0
    query_per_day: 1000
  redis:
  	service_name: redis.static
# 规则二：按域名匹配生效
- _match_domain_:
  - "*.example.com"
  - test.com
  rule_name: limit_rule2
  limit_by_param: apikey
  limit_keys:
  - key: 9a342114-ba8a-11ec-b1bf-00163e1250b5
    query_per_second: 10
  - key: a6a6d7f2-ba8a-11ec-bec2-00163e1250b5
    query_per_minute: 100
  redis:
  	service_name: redis.static
  show_limit_quota_header: true 
```
此例 `_match_route_` 中指定的 `route-a` 和 `route-b` 即在创建网关路由时填写的路由名称，当匹配到这两个路由时，将使用此段配置；
此例 `_match_domain_` 中指定的 `*.example.com` 和 `test.com` 用于匹配请求的域名，当发现域名匹配时，将使用此段配置；
配置的匹配生效顺序，将按照 `_rules_` 下规则的排列顺序，匹配第一个规则后生效对应配置，后续规则将被忽略
