# 功能说明
`key-auth`插件基于wasm-go实现了插件用于将身份验证密钥（API 密钥）添加到路由或服务。X-API-KEY无效或不存在直接拒绝并返回401

# 配置字段
|  名称 |  数据类型 | 填写要求  | 描述  |
| ------------ | ------------ | ------------ | ------------ |
|  key_auth_name   | string    | 选填  |   配置header中key name. 默认: X-API-KEY |
|  key_auth_tokens | []string  | 必填  |   配置可放通Tokens|

# 配置示例
```yaml
...
    - config:
        key_auth_name: X-API-KEY
        key_auth_tokens:
          - 9a1150e86674dcdd4ea664143563428e
          - b49b8ed4cecb2f68eee0354d8c915bfc
...
```
此例`key_auth_name`中请求头名称;`key_auth_tokens`是可放通的token集合；