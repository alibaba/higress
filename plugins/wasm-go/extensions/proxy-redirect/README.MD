# 功能说明

`proxy-redirect` 插件设置上游服务响应的 "Location" 或 "Refresh" 头字段的值。

# 配置字段

| 名称               | 数据类型            | 填写要求     | 默认值                              | 描述                                        |
|------------------|-----------------|----------|----------------------------------|-------------------------------------------|
| http_status_code | array of uint   | Optional | `[201, 301, 302, 303, 307, 308]` | 要处理的 HTTP Status Code 列表                  |
| proxy_redirect   | array of string | Optional | `[default]`                      | proxy_redirect 策略, array 的每一个 Item 表示一条策略 |

`proxy_redirect` 的基本语法：

```json
{
  "proxy_redirect": [
    "default",
    "redirect replacement"
  ]
}
```

"proxy_redirect" 的值类型是 `[]String`。
每一个 Item 表示一条 proxy_redirect 规则，值必须为 `off`, `default` 或 "redirect replacement" 中之一。
插件会选择第一个可执行的规则执行。
当 `proxy_redirect` 的值的长度为 0 时，按 `{"proxy_redirect": ["default"]}` 处理。

# 配置示例

## 示例 1

上游服务响应：

```yaml
Location: http://localhost:8000/two/some/uri/
```

插件配置：

```json
{
  "proxy_redirect": [
    "http://localhost:8000/two/ http://frontend/one/"
  ]
}
```

客户端获取到的响应：

```yaml
Location: http://frontend/one/some/uri/
```

## 示例 2 _replacement_ 中省略服务名称

上游服务响应：

```yaml
Location: http://localhost:8000/two/some/uri/
```

插件配置

```yaml
{
  "proxy_redirect": [
    "http://localhost:8000/two/ /"
  ]
}
```

等价于

```json
{
  "proxy_redirect": [
    "http://localhost:8000/two/ http://frontend/"
  ]
}
```

客户端获取到的响应：

```yaml
Location: http://frontend/some/uri
```

## 示例 3 _default_

上游服务响应：

```yaml
Location: http://localhost:8000/one/two/three
```

插件配置：

```json
{
  "proxy_direct": [
    "default"
  ]
}
```

此时插件会简单地推测 path_rewrite 规则，决定如何修改 `Location`.
客户端请求的原始路径记为 "_original_path_"，上游服务接收到的请求路径记为 "_upstream_path_"，则有

### 情形1 _original_path_ 和 _upstream_path_ 完全一直时不修改 `Location`

```yaml
original_path: /one/two/three
upstream_path: /one/two/three 
```

如上，此时不修改 `Location`.

### 情形2 _original_path_ 和 _upstream_path_ 无公共后缀时不修改 `Location`

```yaml
original_path: /one/two/three
upstream_path: /a/b/c
```

此时不修改 `Location` .

### 情形3 _original_path_ 和 _upstream_path_ 有公共后缀

```yaml
original_path: /one/two/three
upstream_path: /a/two/three
```

如上，_original_path_ 和 _upstream_path_ 的公共后缀是 `/two/three`, 那么推测 Route 的匹配和重写规则为

```yaml
- match:
    prefix: "/one/"
  route:
    prefix_rewrite: "/a/"
```

则 `Location` 的匹配和重写规则可以写作：

```yaml
- match:
    prefix: "/a/"
  location:
    prefix_rewrite: "/one/"
```

> 注意: 以上写法仅作表述用, Higress 和 Envoy 均无此配置。

那么当 `Location` 的值能匹配上述规则时将被修改，如 `Location: /a/some/path` -> `Location: /one/some/path`；
当 `Location` 的值不能匹配上述规则时不被修改。

## 示例 4 正则捕获和引用

可以使用正则表达式指定指令。在这种情况下，"redirect" 应该以 “~” 符号开头以进行区分大小写的匹配，
或者以“~*”符号开头以进行不区分大小写的匹配。正则表达式可以包含命名和位置捕获，"replacement" 可以引用它们：

```json
{
  "proxy_redirect": [
    ",~^(http://[^:]+):\\d+(/.+)$ $1$2"
  ]
}
```

```json
{
  "proxy_redirect": [
    "~*/user/([^/]+)/(.+)$      http://$1.example.com/$2"
  ]
}
```

# 注意

该插件功能基本对标 Nginx "proxy_redirect", 但不支持 "proxy_redirect: off";
Nginx "proxy_redirect: default" 视 "proxy_pass" 决定如何处理 `Location`, 而本插件根据推测的路由匹配和重写规则决定如何处理 `Location`.
由于 "proxy_redirect: default" 依靠简单地推测路由匹配和重写规则, 所以对 `Location` 的修改并不一定总是符合期待，建议谨慎使用。
