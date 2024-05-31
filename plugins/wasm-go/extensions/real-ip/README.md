# 功能说明
real-ip插件，根据real_ip_from判断可信地址，从real_ip_header获取真实IP。
功能与[ngx_http_realip_module](http://nginx.org/en/docs/http/ngx_http_realip_module.html)基本一致


# 配置字段
| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
|  real_ip_from     |  array      | 必填     |   -  |   可信的IP地址或CIDR地址块  |
|  real_ip_header     |  string     | 选填     |   X-Real-IP  |  从配置的请求头中获取真实IP   |
|  recursive     |  bool     | 选填     |   false  |  如果为false，真实IP为real_ip_header最右边的IP；如果为true，从real_ip_header的右边开始，第一个不可信IP为真实IP   |

# 配置示例
```yaml
real_ip_from:
- "10.18.233.233"
- "172.18.0.1/24"
- "2001:0db8::/32"
real_ip_header: "X-Forwarded-For"
recursive: true
```