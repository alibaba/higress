# 功能说明
`transformer` 插件可以对请求/响应头、请求查询参数、请求/响应体参数进行转换，支持的转换操作类型包括删除、重命名、更新、添加、追加、映射、去重。


# 配置字段

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
|  type |  string  | 必填，可选值为`request`, `response` |   -  |  指定转换器类型  |
|  dots_in_keys  |  bool  | 选填     |  false  | 转换规则中 JSON 请求/响应体参数的键是否包含点号，例如 foo.bar:value。若为 true，则表示 foo.bar 作为键名，值为 value；否则表示嵌套关系，即 foo 的成员变量 bar 的值为 value |
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
| kv |  string  | 选填 |   -  | 指定键值对，形如 `key` 或 `key:value`                       |
|  value_type  |  string  | 选填，可选值为`boolean`, `number`, `string` |  string  | 当`content-type: application/json`时，该字段指定请求/响应体参数的值类型 |
| host_pattern |  string  | 选填     |   -  | 指定主机名匹配规则，当转换操作类型为 `replace`, `add`, `append` 时有效 |
| path_pattern | string | 选填 | - | 指定路径匹配规则，当转换操作类型为 `replace`, `add`, `append` 时有效  |

注意：

* `request transformer` 支持以下转换对象：请求头部、请求查询参数、请求体（application/json, application/x-www-form-urlencoded, multipart/form-data）
* `response transformer` 支持以下转换对象：响应头部、响应体（application/json）

* 转换操作类型的执行顺序：remove → rename → replace → add → append → map → dedupe
* 当转换对象为 headers 时，`kv` 字段的 key 部分不区分大小写；当为 headers 且为 `rename`, `map` 操作时，`kv` 字段的 value 部分也不区分大小写；而 querys 和 body 的 `kv` 字段均区分大小写
* `dots_in_keys` 和 `value_type` 仅对 content-type 为 application 的请求/响应体有效
* `host_pattern` 和 `path_pathern` 支持 [RE2 语法](https://pkg.go.dev/regexp/syntax)，仅对 `replace`, `add`, `append` 操作有效，且在一项转换规则中两者只能选填其一，若均填写，则 `host_pattern` 生效，而 `path_pattern` 失效



# 转换操作类型

| 操作类型      | kv 参数格式       | 描述                                                         |
| ------------- | ----------------- | ------------------------------------------------------------ |
| 删除 remove   | `key`             | 若存在指定的 `key:value`，则删除；否则无操作                 |
| 重命名 rename | `oldKey:newKey`   | 若存在指定的 `oldKey:value`，则将其键名重命名为 newKey，得到 `newKey:value`；否则无操作 |
| 更新 replace  | `key:newValue`    | 若存在指定的 `key:value`，则将其 value 更新为 newValue，得到 `key:newValue`；否则无操作 |
| 添加 add      | `key:value`       | 若不存在指定的 `key:value`，则添加；否则无操作               |
| 追加 append   | `key:appendValue` | 若存在指定的 `key:value`，则追加 appendValue 得到 `key:[value..., appendValue]`；否则相当于执行 add 操作，得到 `key:appendValue` |
| 映射 map      | `fromKey:toKey`   | 若存在指定的 `fromKey:fromValue`，则将其值 fromValue 映射给 toKey 的值，得到 `toKey:fromValue`，同时保留 `fromKey:fromValue`（注：若 toKey 已存在则其值会被覆盖）；否则无操作 |
| 去重 dedupe   | `key:strategy`    | strategy 可选值为：<br>`RETAIN_UNIQUE`: 按顺序保留所有唯一值，如 `k1:[v1,v2,v3,v3,v2,v1]`，去重后得到 `k1:[1,2,3]` <br>`RETAIN_LAST`: 保留最后一个值，如 `k1:[v1,v2,v3]`，去重后得到 `k1:v3` <br>`RETAIN_FIRST` (default): 保留第一个值，如 `k1:[v1,v2,v3]`，去重后得到 `k1:v1` |




# 配置示例

## Request Transformer

### 转换请求头部

```yaml
type: request
rules:
- operate: remove
  headers:
  - kv: "X-remove"
- operate: rename
  headers:
  - kv: "X-not-renamed:X-renamed"
- operate: replace
  headers:
  - kv: "X-replace:replaced"
- operate: add
  headers:
  - kv: "X-add-append:host-$1"
    host_pattern: "^(.*)\\.com$"
- operate: append
  headers:
  - kv: "X-add-append:path-$1"
    path_pattern: "^.*?\\/(\\w+)[\\?]{0,1}.*$"
- operate: map
  headers:
  - kv: "X-add-append:X-map"
- operate: dedupe
  headers:
  - kv: "X-dedupe-first:RETAIN_FIRST"
  - kv: "X-dedupe-last:RETAIN_LAST"
  - kv: "X-dedupe-unique:RETAIN_UNIQUE"
```

发送请求

```bash
$ curl -v console.higress.io/get -H 'host: foo.bar.com' \
-H 'X-remove: exist' -H 'X-not-renamed:test' -H 'X-replace:not-replaced'\
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
  - kv: "k1"
- operate: rename
  querys:
  - kv: "k2:k2-new"
- operate: replace
  querys:
  - kv: "k2-new:v2-new"
- operate: add
  querys:
  - kv: "k3:v31-$1"
    path_pattern: "^.*?\\/(\\w+)[\\?]{0,1}.*$"
- operate: append
  querys:
  - kv: "k3:v32"
- operate: map
  querys:
  - kv: "k3:k4"
- operate: dedupe
  querys:
  - kv: "k4:RETAIN_FIRST"
```

发送请求

```bash
$ curl -v "console.higress.io/get?k1=v11&k1=v12&k2=v2" -H 'host: foo.bar.com'

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
  - kv: "a1"
- operate: rename
  body: 
  - kv: "a2:a2-new"
- operate: replace
  body:
  - kv: "a3:t3-new"
    value_type: string
- operate: add
  body:
  - kv: "a1-new:t1-new"
    value_type: string
- operate: append
  body:
  - kv: "a1-new:t1-$1-append"
    value_type: string
    host_pattern: "^(.*)\\.com$"
- operate: map
  body:
  - kv: "a1-new:a4"
- operate: dedupe
  body:
  - kv: "a4:RETAIN_FIRST"
```

发送请求：

**1. Content-Type: application/json**

```bash
$ curl -v -x POST console.higress.io/post -H 'host: foo.bar.com' -H 'Content-Type: application/json' -d '{"a1":"t1","a2":"t2","a3":"t3"}'

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
$ curl -v -X POST console.higress.io/post -H 'host: foo.bar.com' -d 'a1=t1&a2=t2&a3=t3'

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
$ curl -v -X POST console.higress.io/post -H 'host: foo.bar.com' -F a1=t1 -F a2=t2 -F a3=t3

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

与 Request Transformer 类似，在此仅演示 `dots_in_keys` 字段的效果：

**1. dots_in_keys: true**

```yaml
type: response
dots_in_keys: true
rules:
- operate: add
  body:
  - kv: foo.bar:value
    value_type: string
```

```bash
$ curl -v console.higress.io/get -H 'host: foo.bar.com'

# httpbin 响应结果
{
 ...
 "foo.bar": "value",
 ...
}
```

**2. dots_in_keys: false**

```yaml
type: response
dots_in_keys: false
rules:
- operate: add
  body:
  - kv: foo.bar:value
    value_type: string
```

```bash
$ curl -v console.higress.io/get -H 'host: foo.bar.com'

# httpbin 响应结果
{
 ...
 "foo": {
  "bar": "value"
 },
 ...
}
```

