# traffic-tag插件说明文档

本文档提供了 `traffic-tag` 插件的配置选项的详细信息。此配置用于管理基于灰度发布和测试目的定义的特定规则的流量标记。

## 功能说明

`traffic-tag` 插件允许根据权重或特定请求内容通过添加特定请求头的方式对请求流量进行标记。它支持复杂的逻辑来确定如何根据用户定义的标准标记流量。

## 配置字段

此部分提供了配置字段的详细描述。

| 字段名称        | 类型     | 默认值 | 是否必填 | 描述                                                         |
|----------------|----------|-------|---------|-------------------------------------------------------------|
| `conditionGroups` | array of object  | -     | 否      | 定义基于内容的标记条件组，详细结构见**条件组配置**。             |
| `weightGroups`    | array of object  | -     | 否      | 定义基于权重的标记条件组，详细结构见**权重组配置**。             |
| `defaultTagKey`   | string   | -     | 否      | 默认的标记键名，当未匹配到任何条件时使用。当且仅当同时配置了**defaultTagVal**时生效      |
| `defaultTagVal` | string   | -     | 否      | 默认的标记值，当未匹配到任何条件时使用。当且仅当同时配置了**defaultTagKey**时生效      |

### 条件组配置
`conditionGroups` 中每一项的配置字段说明如下：

| 字段名称      | 类型   | 默认值 | 是否必填 | 描述                                                         |
|--------------|--------|-------|---------|-------------------------------------------------------------|
| `headerName` | string | -     | 是      | 要添加或修改的 HTTP 头名称。                                  |
| `headerValue`| string | -     | 是      | HTTP 头的值。                                                |
| `logic`      | string | -     | 是      | 条件组中的逻辑关系，支持 `and`、`or`，必须为小写字母。         |
| `conditions` | array of object  | -     | 是      | 描述具体的标记条件，详细结构如下。                    |
---

`conditions` 中每一项的配置字段说明如下：

| 字段名称        | 类型   | 默认值 | 是否必填 | 描述                                                         |
|----------------|--------|-------|---------|-------------------------------------------------------------|
| `conditionType`| string | -     | 是      | 条件类型，支持 `header`、`parameter`、`cookie`。                 |
| `key`          | string | -     | 是      | 条件的关键字。                                               |
| `operator`     | string | -     | 是      | 操作符，支持 `equal`、`not_equal`、`prefix`、`in`、`not_in`、`regex`、`percentage`。  |
| `value`        | array of string  | -     | 是      | 条件的值，**仅当**操作符为 `in` 和 `not_in` 时支持配置多个值。 |

> **说明：当 `operator` 为 `regex` 时，使用的正则表达式引擎是 [RE2](https://github.com/google/re2)。详情请参阅 [RE2 官方文档](https://github.com/google/re2/wiki/Syntax)。

### 权重组配置

`weightGroups` 中每一项的配置字段说明如下：

| 字段名称      | 类型     | 默认值 | 是否必填 | 描述                                                         |
|--------------|----------|-------|---------|-------------------------------------------------------------|
| `headerName` | string   | -     | 是      | 要添加或修改的 HTTP 头名称。                                  |
| `headerValue`| string   | -     | 是      | HTTP 头的值。                                                |
| `weight`     | integer  | -     | 是      | 流量权重百分比。                                             |                                           

### 操作符说明
| 操作符      | 描述                                      |
|-------------|------------------------------------------|
| `equal`        | 精确匹配，值需要完全相等                  |
| `not_equal`        | 不等匹配，值不相等时满足条件              |
| `prefix`    | 前缀匹配，指定值是实际值的前缀时满足条件  |
| `in`        | 包含匹配，实际值需要在指定的列表中        |
| `not_in`    | 排除匹配，实际值不在指定的列表中时满足条件|
| `regex`     | 正则表达式匹配，按照正则表达式规则匹配    |
| `percentage`| 百分比匹配，原理：`hash(get(key)) % 100 < value` 成立时满足条件|

> **提示：关于`percentage`和`weight`的区别**
>
> - **`percentage`操作符**：用于条件表达式中，基于指定的百分比和指定的键值对来判断是否执行某个操作。对于一个相同的键值对，多次匹配的结果是幂等的，即这一次命中条件，下一次也会命中。
> - **`weight`字段**：用于定义不同处理路径的流量权重。在基于权重的流量标记中，`weight`确定了某个路径应接收的流量比例。与`percentage`不同的是，由于没有指定固定的对比依据而是基于随机权重分布，同一个请求的多次匹配可能匹配多个结果。
>
> 使用`percentage`进行条件匹配时，判断每个请求是否满足特定百分比条件；而`weight`则是静态随机分配整体流量的比例。

## 配置示例
### 对特定路由开启
`_match_route_` 中指定的 `route-a` 和 `route-b` 即在创建网关路由时填写的路由名称，当匹配到这两个路由时，将使用此段配置；

当配置了多个规则时，配置的匹配生效顺序将按照 `_rules_` 下规则的排列顺序，第一个规则匹配后生效对应配置，后续规则将被忽略。

**例1: 基于内容的匹配**

按照下例的配置，路由`route-a`和`route-a`命中的请求中，同时满足请求头`role` 的值是`user`、`viwer`、`editor`其中之一且存在查询参数`foo=bar`的请求将被添加请求头`x-mse-tag: gray`。由于配置了`defaultTagKey`和`defaultTagVal`，当未匹配到任何条件时，请求将被添加请求头`x-mse-tag: base`。
```yaml
# 使用 _rules_ 字段进行细粒度规则配置
_rules_:
- _match_route_:
  - route-a
  - route-b
  defaultTagKey: x-mse-tag
  defaultTagVal: base
  conditionGroups:
    - headerName: x-mse-tag
      headerValue: gray
      logic: and
      conditions:
        - conditionType: header
          key: role
          operator: in
          value:
            - user
            - viewer
            - editor
        - conditionType: parameter
          key: foo
          operator: equal
          value:
          - bar
```
**例子2: 基于权重的匹配**

按照下列配置，请求将有30%几率被添加请求头`x-mse-tag: gray`，30%几率被添加请求头`x-mse-tag: blue`，40%几率不添加请求头。

```yaml
_rules_:
- _match_route_:
  - route-a
  - route-b
  # 权重总和为100，下例中未配置的40权重将不添加header
  weightGroups:
    - headerName: x-mse-tag
      headerValue: gray
      weight: 30
    - headerName: x-mse-tag
      headerValue: blue
      weight: 30
```
### 对特定域名开启
 `_match_domain_` 中指定的 `*.example.com` 和 `test.com` 用于匹配请求的域名，当发现域名匹配时，将使用此段配置；

 按照下例配置，对于目标为`*.example.com`和`test.com`的请求，当含有请求头`role`且其值以`user`为前缀，比如`role: user_common`，请求将被添加header`x-mse-tag: blue`。

```yaml
_rules_:
- _match_domain_:
  - "*.example.com"
  - test.com
  conditionGroups:
    - headerName: x-mse-tag
      headerValue: blue
      logic: and
      conditions:
        - conditionType: header
          key: role
          operator: prefix
          value:
            - user
```

### 网关实例级别开启
以下配置未指定_rules_字段，因此将对网关实例级别生效。
可按照基于内容或基于权重的匹配单独配置`conditionGroups`或`weightGroups`，若同时配置，插件将先按照`conditionGroups`中的配置条件进行匹配，匹配成功添加相应header后跳过后续逻辑。若所有`conditionGroups`都未匹配成功，则进入`weightGroups`按照权重配置添加header。
```yaml
conditionGroups:
  - headerName: x-mse-tag-1
    headerValue: gray
    # logic为or，则conditions中任一条件满足就命中匹配
    logic: or
    conditions:
      - conditionType: header
        key: foo
        operator: equal
        value:
          - bar
      - conditionType: cookie
        key: x-user-type
        operator: prefix
        value:
          - test
  - headerName: x-mse-tag-2
    headerValue: blue
    # logic为and，需要conditions中所有条件满足才命中匹配
    logic: and
    conditions:
      - conditionType: header
        key: x-type
        operator: in
        value:
          - type1
          - type2
          - type3
      - conditionType: header
        key: x-mod
        operator: regex
        value:
          - "^[a-zA-Z0-9]{8}$"
  - headerName: x-mse-tag-3
    headerValue: green
    logic: and
    conditions:
      - conditionType: header
        key: user_id
        operator: percentage
        value:
          - 60
# 权重总和为100，下例中未配置的40权重将不添加header
weightGroups:
  - headerName: x-mse-tag
    headerValue: gray
    weight: 30
  - headerName: x-mse-tag
    headerValue: base
    weight: 30
```
