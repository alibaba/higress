---
title: 请求响应编辑
keywords: [ higress,request,response,edit ]
description: 请求响应编辑插件使用说明
---

## 功能说明

`traffic-editor` 插件可以对请求/响应头进行修改，支持的修改操作类型包括删除、重命名、更新、添加、追加、映射、去重。

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`100`

## 配置字段

| 字段名                | 类型                                      | 必填 | 说明                  |
|--------------------|-----------------------------------------|----|---------------------|
| defaultConfig      | object (CommandSet)                     | 否  | 默认命令集配置，无条件执行的编辑操作  |
| conditionalConfigs | array of object (ConditionalCommandSet) | 否  | 条件命令集配置，按条件执行不同编辑操作 |

### CommandSet 结构

| 字段名            | 类型                        | 必填 | 默认值   | 说明         |
|----------------|---------------------------|----|-------|------------|
| disableReroute | bool                      | 否  | false | 是否禁用自动路由重选 |
| commands       | array of object (Command) | 是  | -     | 编辑命令列表     |

### ConditionalCommandSet 结构

| 字段名        | 类型                        | 必填 | 说明                  |
|------------|---------------------------|----|---------------------|
| conditions | array                     | 是  | 条件列表，见下表            |
| commands   | array of object (Command) | 是  | 命令列表，结构同 CommandSet |

#### Command 结构

| 字段名  | 类型     | 必填 | 说明                |
|------|--------|----|-------------------|
| type | string | 是  | 命令类型。其他配置字段由类型决定。 |

##### set 命令

功能为将某个字段设置为指定值。`type` 字段值为 `set`。

其它字段如下：

| 字段名    | 类型           | 必填 | 说明     |
|--------|--------------|----|--------|
| target | object (Ref) | 是  | 目标字段信息 |
| value  | string       | 是  | 要设置的值  |

##### concat 命令

功能为将多个值拼接后赋值给目标字段。`type` 字段值为 `concat`。

其它字段如下：

| 字段名    | 类型                    | 必填 | 说明                       |
|--------|-----------------------|----|--------------------------|
| target | object (Ref)          | 是  | 目标字段信息                   |
| values | array of (string/Ref) | 是  | 要拼接的值列表，可以是字符串或字段引用（Ref） |

##### copy 命令

功能为将源字段的值复制到目标字段。`type` 字段值为 `copy`。

其它字段如下：

| 字段名    | 类型           | 必填 | 说明     |
|--------|--------------|----|--------|
| source | object (Ref) | 是  | 源字段信息  |
| target | object (Ref) | 是  | 目标字段信息 |

##### delete 命令

功能为删除指定字段。`type` 字段值为 `delete`。

其它字段如下：

| 字段名    | 类型           | 必填 | 说明       |
|--------|--------------|----|----------|
| target | object (Ref) | 是  | 要删除的字段信息 |

##### rename 命令

功能为将字段重命名。`type` 字段值为 `rename`。

其它字段如下：

| 字段名    | 类型           | 必填 | 说明    |
|--------|--------------|----|-------|
| source | object (Ref) | 是  | 原字段信息 |
| target | object (Ref) | 是  | 新字段信息 |

#### Condition 结构

| 字段名  | 类型     | 必填 | 说明                |
|------|--------|----|-------------------|
| type | string | 是  | 条件类型。其他配置字段由类型决定。 |

##### equals 条件

判断某字段值是否等于指定值。`type` 字段值为 `equals`。

| 字段名    | 类型           | 必填 | 说明      |
|--------|--------------|----|---------|
| value1 | object (Ref) | 是  | 参与比较的字段 |
| value2 | string       | 是  | 目标值     |

##### prefix 条件

判断某字段值是否以指定前缀开头。`type` 字段值为 `prefix`。

| 字段名    | 类型           | 必填 | 说明      |
|--------|--------------|----|---------|
| value  | object (Ref) | 是  | 参与比较的字段 |
| prefix | string       | 是  | 前缀字符串   |

##### suffix 条件

判断某字段值是否以指定后缀结尾。`type` 字段值为 `suffix`。

| 字段名    | 类型           | 必填 | 说明      |
|--------|--------------|----|---------|
| value  | object (Ref) | 是  | 参与比较的字段 |
| suffix | string       | 是  | 后缀字符串   |

##### contains 条件

判断某字段值是否包含指定子串。`type` 字段值为 `contains`。

| 字段名    | 类型           | 必填 | 说明      |
|--------|--------------|----|---------|
| value  | object (Ref) | 是  | 参与比较的字段 |
| substr | string       | 是  | 子串      |

##### regex 条件

判断某字段值是否匹配指定正则表达式。`type` 字段值为 `regex`。

| 字段名     | 类型           | 必填 | 说明      |
|---------|--------------|----|---------|
| value   | object (Ref) | 是  | 参与比较的字段 |
| pattern | string       | 是  | 正则表达式   |

#### Ref 结构

用于标识一个请求或响应中的字段。

| 字段名  | 类型     | 必填 | 说明                                                           |
|------|--------|----|--------------------------------------------------------------|
| type | string | 是  | 字段类型。可选值有：`request_header`、`request_query`、`response_header` |
| name | string | 是  | 字段名称                                                         |

### 示例配置

```json
{
  "defaultConfig": {
    "disableReroute": false,
    "commands": [
      {
        "type": "set",
        "target": {
          "type": "request_header",
          "name": "x-user"
        },
        "value": "admin"
      },
      {
        "type": "delete",
        "target": {
          "type": "request_header",
          "name": "x-dummy"
        }
      }
    ]
  },
  "conditionalConfigs": [
    {
      "conditions": [
        {
          "type": "equals",
          "value1": {
            "type": "request_query",
            "name": "id"
          },
          "value2": "1"
        }
      ],
      "commands": [
        {
          "type": "set",
          "target": {
            "type": "response_header",
            "name": "x-id"
          },
          "value": "1"
        }
      ]
    }
  ]
}
```
