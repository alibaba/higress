<p>
   <a href="README_EN.md"> English </a> | 中文
</p>

# 功能说明
`key-rate-limit`插件实现了基于特定键值实现限流，键值来源可以是 URL 参数、HTTP 请求头

# 配置字段

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
|  limit_by_header     |  string     | 选填，`limit_by_header`,`limit_by_param` 中选填一项     |   -  |  配置获取限流键值的来源 http 请求头名称   |
|  limit_by_param     |  string     | 选填，`limit_by_header`,`limit_by_param` 中选填一项     |   -  |  配置获取限流键值的来源 URL 参数名称   |
|  limit_keys     |  array of object     | 必填     |   -  |  配置匹配键值后的限流次数   |
| `global_auth` | bool | 选填     | -      | 若配置为true，则全局生效认证机制; 若配置为false，则只对做了配置的域名和路由生效认证机制; 若不配置则仅当没有域名和路由配置时全局生效（兼容机制）  |


`limit_keys`中每一项的配置字段说明

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
|  key     |  string     | 必填     |   -  |  匹配的键值 |
|  query_per_second     |  number     | 选填，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项     |   -  |  允许每秒请求次数 |
|  query_per_minute     |  number     | 选填，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项     |   -  |  允许每分钟请求次数 |
|  query_per_hour     |  number     | 选填，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项     |   -  |  允许每小时请求次数 |
|  query_per_day     |  number     | 选填，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项     |   -  |  允许每天请求次数 |

# 配置示例

## 识别请求参数 apikey，进行区别限流
```yaml
limit_by_param: apikey
limit_keys:
- key: 9a342114-ba8a-11ec-b1bf-00163e1250b5
  query_per_second: 10
- key: a6a6d7f2-ba8a-11ec-bec2-00163e1250b5
  query_per_minute: 100
```

## 识别请求头 x-ca-key，进行区别限流
```yaml
limit_by_header: x-ca-key
limit_keys:
- key: 102234
  query_per_second: 10
- key: 308239
  query_per_hour: 10

```
## 相关错误码

| HTTP 状态码 | 出错信息                                                                           | 原因说明               |
| ----------- |--------------------------------------------------------------------------------| ---------------------- |
| 429         | Too many requests. | 请求过多         |
