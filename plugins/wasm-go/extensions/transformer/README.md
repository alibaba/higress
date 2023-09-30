# 功能说明
`transformer` 插件可以对请求/响应头、请求查询参数、请求/响应体参数进行转换，支持的转换操作类型包括删除、重命名、更新、添加、追加、映射、去重。


# 配置字段

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
|  type |  string  | 必填，可选值为 `request`, `response` |   -  |  指定转换器类型  |
| rules |  array of object  | 选填     |   -  | 指定转换操作类型以及请求/响应头、请求查询参数、请求/响应体参数的转换规则 |

`rules`中每一项的配置字段说明如下：

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
| operate |  string  | 必填，可选值为 `remove`, `rename`, `replace`, `add`, `append`, `map`, `dedupe` |   -  |  指定转换操作类型，支持的操作类型有删除 (remove)、重命名 (rename)、更新 (replace)、添加 (add)、追加 (append)、映射 (map)、去重 (dedupe)，当存在多项不同类型的转换规则时，按照上述操作类型顺序依次执行  |
|  headers  |  array of object  | 选填     |  -  | 指定请求/响应头转换规则 |
| querys |  array of object  | 选填     |   -  | 指定请求查询参数转换规则 |
| body | array of object | 选填 | - | 指定请求/响应体参数转换规则，请求体转换允许 content-type 为 `application/json`, `application/x-www-form-urlencoded`, `multipart/form-data`；响应体转换仅允许 content-type 为 `application/json` |

`headers`, `querys`, `body`中每一项的配置字段说明如下：

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述                                                |
| -------- | -------- | -------- | -------- |---------------------------------------------------|
| key |  string  | 选填 |   -  | 指定键，详见[转换操作类型](#转换操作类型) |
| value | string | 选填 | - | 指定值，详见[转换操作类型](#转换操作类型) |
|  value_type  |  string  | 选填，可选值为 `object`, `boolean`, `number`, `string` |  string  | 当`content-type: application/json`时，该字段指定请求/响应体参数的值类型 |
| host_pattern |  string  | 选填     |   -  | 指定请求主机名匹配规则，当转换操作类型为 `replace`, `add`, `append` 时有效 |
| path_pattern | string | 选填 | - | 指定请求路径匹配规则，当转换操作类型为 `replace`, `add`, `append` 时有效 |

注意：

* `request transformer` 支持以下转换对象：请求头部、请求查询参数、请求体（application/json, application/x-www-form-urlencoded, multipart/form-data）
* `response transformer` 支持以下转换对象：响应头部、响应体（application/json）

* 转换操作类型的执行顺序：remove → rename → replace → add → append → map → dedupe
* 当转换对象为 headers 时，` key` 不区分大小写；当为 headers 且为 `rename`, `map` 操作时，`value` 也不区分大小写（因为此时该字段具有 key 含义）；而 querys 和 body 的 `key`, `value` 字段均区分大小写
* `value_type` 仅对 content-type 为 application/json 的请求/响应体有效
* `host_pattern` 和 `path_pathern` 支持 [RE2 语法](https://pkg.go.dev/regexp/syntax)，仅对 `replace`, `add`, `append` 操作有效，且在一项转换规则中两者只能选填其一，若均填写，则 `host_pattern` 生效，而 `path_pattern` 失效



# 转换操作类型

| 操作类型      | key 字段含义 | value 字段含义     | 描述                                                         |
| ------------- | ----------------- |-----| ------------------------------------------------------------ |
| 删除 remove   | 目标 key     |无需设置| 若存在指定的 `key`，则删除；否则无操作                 |
| 重命名 rename | 目标 oldKey |新的 key 名称 newKey| 若存在指定的 `oldKey:value`，则将其键名重命名为 `newKey`，得到 `newKey:value`；否则无操作 |
| 更新 replace  | 目标 key |新的 value 值 newValue| 若存在指定的 `key:value`，则将其 value 更新为 `newValue`，得到 `key:newValue`；否则无操作 |
| 添加 add      | 添加的 key | 添加的 value |若不存在指定的 `key:value`，则添加；否则无操作               |
| 追加 append   | 目标 key |追加的 value值 appendValue| 若存在指定的 `key:value`，则追加 appendValue 得到 `key:[value..., appendValue]`；否则相当于执行 add 操作，得到 `key:appendValue` |
| 映射 map      | 映射来源 fromKey |映射目标 toKey| 若存在指定的 `fromKey:fromValue`，则将其值 fromValue 映射给 toKey 的值，得到 `toKey:fromValue`，同时保留 `fromKey:fromValue`（注：若 toKey 已存在则其值会被覆盖）；否则无操作 |
| 去重 dedupe   | 目标 key |指定去重策略 strategy| `strategy` 可选值为：<br>`RETAIN_UNIQUE`: 按顺序保留所有唯一值，如 `k1:[v1,v2,v3,v3,v2,v1]`，去重后得到 `k1:[v1,v2,v3]` <br>`RETAIN_LAST`: 保留最后一个值，如 `k1:[v1,v2,v3]`，去重后得到 `k1:v3` <br>`RETAIN_FIRST` (default): 保留第一个值，如 `k1:[v1,v2,v3]`，去重后得到 `k1:v1`<br>（注：若去重后只剩下一个元素 v1 时，键值对变为 `k1:v1`, 而不是 `k1:[v1]`） |




# 配置示例

## Request Transformer

### 转换请求头部

```yaml
type: request
rules:
- operate: remove
  headers:
  - key: X-remove
- operate: rename
  headers:
  - key: X-not-renamed
    value: X-renamed
- operate: replace
  headers:
  - key: X-replace
    value: replaced
- operate: add
  headers:
  - key: X-add-append
    value: host-$1
    host_pattern: ^(.*)\.com$
- operate: append
  headers:
  - key: X-add-append
    value: path-$1
    path_pattern: ^.*?\/(\w+)[\?]{0,1}.*$
- operate: map
  headers:
  - key: X-add-append
    value: X-map
- operate: dedupe
  headers:
  - key: X-dedupe-first
    value: RETAIN_FIRST
  - key: X-dedupe-last
    value: RETAIN_LAST
  - key: X-dedupe-unique
    value: RETAIN_UNIQUE
```

发送请求

```bash
$ curl -v console.higress.io/get -H 'host: foo.bar.com' \
-H 'X-remove: exist' -H 'X-not-renamed:test' -H 'X-replace:not-replaced' \
-H 'X-dedupe-first:1' -H 'X-dedupe-first:2' -H 'X-dedupe-first:3' \
-H 'X-dedupe-last:a' -H 'X-dedupe-last:b' -H 'X-dedupe-last:c' \
-H 'X-dedupe-unique:1' -H 'X-dedupe-unique:2' -H 'X-dedupe-unique:3' \
-H 'X-dedupe-unique:3' -H 'X-dedupe-unique:2' -H 'X-dedupe-unique:1'

# httpbin 响应结果
{
  "args": {},
  "headers": {
    ...
    "X-Add-Append": "host-foo.bar,path-get",
    ...
    "X-Dedupe-First": "1",
    "X-Dedupe-Last": "c",
    "X-Dedupe-Unique": "1,2,3",
    ...
    "X-Map": "host-foo.bar,path-get",
    "X-Renamed": "test",
    "X-Replace": "replaced"
  },
  ...
}
```

### 转换请求查询参数

```yaml
type: request
rules:
- operate: remove
  querys:
  - key: k1
- operate: rename
  querys:
  - key: k2
    value: k2-new
- operate: replace
  querys:
  - key: k2-new
    value: v2-new
- operate: add
  querys:
  - key: k3
    value: v31-$1
    path_pattern: ^.*?\/(\w+)[\?]{0,1}.*$
- operate: append
  querys:
  - key: k3
    value: v32
- operate: map
  querys:
  - key: k3
    value: k4
- operate: dedupe
  querys:
  - key: k4
    value: RETAIN_FIRST
```

发送请求

```bash
$ curl -v "console.higress.io/get?k1=v11&k1=v12&k2=v2"

# httpbin 响应结果
{
  "args": {
    "k2-new": "v2-new",
    "k3": [
      "v31-get",
      "v32"
    ],
    "k4": "v31-get"
  },
  ...
  "url": "http://foo.bar.com/get?k2-new=v2-new&k3=v31-get&k3=v32&k4=v31-get"
}
```

### 转换请求体

```yaml
type: request
rules:
- operate: remove
  body:
  - key: a1
- operate: rename
  body: 
  - key: a2
    value: a2-new
- operate: replace
  body:
  - key: a3
    value: t3-new
    value_type: string
- operate: add
  body:
  - key: a1-new
    value: t1-new
    value_type: string
- operate: append
  body:
  - key: a1-new
    value: t1-$1-append
    value_type: string
    host_pattern: ^(.*)\.com$
- operate: map
  body:
  - key: a1-new
    value: a4
- operate: dedupe
  body:
  - key: a4
    value: RETAIN_FIRST
```

发送请求：

**1. Content-Type: application/json**

```bash
$ curl -v -x POST console.higress.io/post -H 'host: foo.bar.com' \
-H 'Content-Type: application/json' -d '{"a1":"t1","a2":"t2","a3":"t3"}'

# httpbin 响应结果
{
  ...
  "headers": {
    ...
    "Content-Type": "application/json",
    ...
  },
  "json": {
    "a1-new": [
      "t1-new",
      "t1-foo.bar-append"
    ],
    "a2-new": "t2",
    "a3": "t3-new",
    "a4": "t1-new"
  },
  ...
}
```

**2. Content-Type: application/x-www-form-urlencoded**

```bash
$ curl -v -X POST console.higress.io/post -H 'host: foo.bar.com' \
-d 'a1=t1&a2=t2&a3=t3'

# httpbin 响应结果
{
  ...
  "form": {
    "a1-new": [
      "t1-new",
      "t1-foo.bar-append"
    ],
    "a2-new": "t2",
    "a3": "t3-new",
    "a4": "t1-new"
  },
  "headers": {
    ...
    "Content-Type": "application/x-www-form-urlencoded",
    ...
  },
  ...
}
```

**3. Content-Type:  multipart/form-data**

```bash
$ curl -v -X POST console.higress.io/post -H 'host: foo.bar.com' \
-F a1=t1 -F a2=t2 -F a3=t3

# httpbin 响应结果
{
  ...
  "form": {
    "a1-new": [
      "t1-new",
      "t1-foo.bar-append"
    ],
    "a2-new": "t2",
    "a3": "t3-new",
    "a4": "t1-new"
  },
  "headers": {
    ...
    "Content-Type": "multipart/form-data; boundary=------------------------1118b3fab5afbc4e",
    ...
  },
  ...
}
```

## Response Transformer

与 Request Transformer 类似，在此仅说明转换 JSON 形式的请求/响应体时的注意事项：

### key 嵌套 `.`

1.通常情况下，指定的 key 中含有 `.` 表示嵌套含义，如下：

```yaml
type: response
rules:
- operate: add
  body:
  - key: foo.bar
    value: value
```

```bash
$ curl -v console.higress.io/get

# httpbin 响应结果
{
 ...
 "foo": {
  "bar": "value"
 },
 ...
}
```

2.当使用 `\.` 对 key 中的 `.` 进行转义后，表示非嵌套含义，如下：

> 当使用双引号括住字符串时使用 `\\.` 进行转义

```yaml
type: response
rules:
- operate: add
  body:
  - key: foo\.bar
    value: value
```

```bash
$ curl -v console.higress.io/get

# httpbin 响应结果
{
 ...
 "foo.bar": "value",
 ...
}
```

### 访问数组元素 `.index`

可以通过数组下标 `array.index 访问数组元素，下标从 0 开始：

```json
{
  "users": [
    {
      "123": { "name": "zhangsan", "age": 18 }
    },
    {
      "456": { "name": "lisi", "age": 19 }
    }
  ]
}
```

1.移除 `user` 第一个元素：

```yaml
type: request
rules:
- operate: remove
  body:
  - key: users.0
```

```bash
$ curl -v -X POST console.higress.io/post \
-H 'Content-Type: application/json' \
-d '{"users":[{"123":{"name":"zhangsan"}},{"456":{"name":"lisi"}}]}'

# httpbin 响应结果
{
  ...
  "json": {
    "users": [
      {
        "456": {
          "name": "lisi"
        }
      }
    ]
  },
  ...
}
```

2.将 `users` 第一个元素的 key 为 `123` 重命名为 `msg`:

```yaml
type: request
rules:
- operate: rename
  body:
  - key: users.0.123
    value: users.0.first
```

```bash
$ curl -v -X POST console.higress.io/post \
-H 'Content-Type: application/json' \
-d '{"users":[{"123":{"name":"zhangsan"}},{"456":{"name":"lisi"}}]}'


# httpbin 响应结果
{
  ...
  "json": {
    "users": [
      {
        "msg": {
          "name": "zhangsan"
        }
      },
      {
        "456": {
          "name": "lisi"
        }
      }
    ]
  },
  ...
}
```

### 遍历数组元素 `.#`

可以使用 `array.#` 对数组进行遍历操作：

> ❗️该操作目前只能用在 replace 上，请勿在其他转换中尝试该操作，以免造成无法预知的结果

```json
{
  "users": [
    {
      "name": "zhangsan", 
      "age": 18
    },
    {
      "name": "lisi",
      "age": 19
    }
  ]
}
```

```yaml
type: request
rules:
- operate: replace
  body:
  - key: users.#.age
    value: 20
```

```bash
$ curl -v -X POST console.higress.io/post \
-H 'Content-Type: application/json' \
-d '{"users":[{"name":"zhangsan","age":18},{"name":"lisi","age":19}]}'


# httpbin 响应结果
{
  ...
  "json": {
    "users": [
      {
        "age": "20",
        "name": "zhangsan"
      },
      {
        "age": "20",
        "name": "lisi"
      }
    ]
  },
  ...
}
```
