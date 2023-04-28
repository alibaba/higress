# 功能说明
`gw-error-format`本插件实现了匹配网关未转发到后端服务时的响应状态码和响应内容体并替换返回自定义响应内容

# 配置字段
| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
|  rules.match.statuscode     |  string     |  必填     |   -  |  匹配响应状态码   |
|  rules.match.responsebody     |  string     | 必填    |   -  |   匹配响应体   |
|  rules.replace.statuscode     |  string     |  必填     |   -  |  替换后的响应状态码   |
|  rules.replace.responsebody     |  string     | 必填    |   -  |   替换后的响应体   |
|  set_header     |  array of object      |  选填     |   -  |  添加/替换响应头，例如：- content-type:  "application/json"   |

# 配置示例
```yaml
rules:
- match:
    statuscode: "403"
    responsebody: "RBAC: access denied"
  replace:
    statuscode: "200"
    responsebody: "{\"code\":401,\"message\":\"User is not authenticated\"}"
- match:
    statuscode: "503"
    responsebody: "no healthy upstream"
  replace:
    statuscode: "200"
    responsebody: "{\"code\":404,\"message\":\"No Healthy Service\"}"
set_header:
- Access-Control-Allow-Credentials: "true"
- Access-Control-Allow-Origin: "*"
- Access-Control-Allow-Headers: "*"
- Access-Control-Allow-Methods: "*"
- Access-Control-Expose-Headers: "*"
- Content-Type:  "application/json;charset=UTF-8"
```

## 示例说明：
以上配置示例作用于当前实例全局生效

match下指定的statuscode和responsebody将被替换为同级中的replace下的statuscode和responsebody

以上示例当某个请求返回的响应状态码是403并且响应内容体是RBAC: access denied的则替换状态码为200和响应内容体为json格式"{"code":401,"message":"User is not authenticated"}"

如果需要新增/替换response header则可以在rules同级中添加set_header字段，当有match下的statuscode匹配上之后会将set_header的内容带在response header


## 小提示：
当envoy网关还未转发至后端服务时response header里面不会带有这个header：x-envoy-upstream-service-time
本插件只在没有获取到此x-envoy-upstream-service-time响应头时生效

