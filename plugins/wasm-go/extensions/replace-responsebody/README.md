# 功能说明
`replace-responsebody`插件实现了匹配指定响应状态码和响应内容体并替换返回自定义响应

# 配置字段

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
|  statuscode     |  string     |  必填     |   -  |  匹配响应状态码   |
|  responsebody     |  string     | 必填    |   -  |   匹配响应体   |
|  replace.statuscode     |  string     |  必填     |   -  |  替换后的响应状态码   |
|  replace.responsebody     |  string     | 必填    |   -  |   替换后的响应体   |
|  replace.responseheader     |  array of object     | 选填    |   -  |   需要替换的header 例如：replace.responseheader.content-type:  "application/json"   |

# 配置示例
```yaml
rules:
- match:
    statuscode: "403"
    responsebody: "RBAC: access denied"
    replace:
      statuscode: "401"
      responsebody: "{\"code\":401,\"message\":\"User is not authenticated\"}"
      responseheader: 
        - content-type:  "application/json"
- match:
    statuscode: "503"
    responsebody: "no healthy upstream"
    replace:
      statuscode: "404"
      responsebody: "{\"code\":503,\"message\":\"No Healthy Service\"}"
      responseheader:
        - content-type:  "application/json;charset=UTF-8"
        - content-language: "zh-CN"
```

## 示例说明：
以上配置示例作用于当前实例全局生效

此例match下指定的statuscode和responsebody将被替换为同级中replace下的statuscode和responsebody，如需要增加或替换header则可以在replace下添加header字段

当某个请求的响应状态码是403并且响应内容是RBAC: access denied的则替换状态码为401和响应内容为json格式的内容"{\"code\":401,\"message\":\"User is not authenticated\"}"



