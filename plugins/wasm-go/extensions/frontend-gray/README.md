# frontend-gray 前端灰度插件
## 功能说明
`frontend-gray`插件实现了前端用户灰度的的功能，通过此插件，不但可以用于业务`A/B实验`，同时通过`可灰度`配合`可监控`,`可回滚`策略保证系统发布运维的稳定性。

## 配置字段
| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                |
|----------------|--------------|------|-----|-----------------------------------------------------------------------------------|
| `grayKey`         | string       | 非必填   | -   | 用户ID的唯一标识，可以来自Cookie或者Header中，比如 userid，如果没有填写则使用`rules[].grayTagKey`和`rules[].grayTagValue`过滤灰度规则                                       |
| `graySubKey`    | string       | 非必填   | -   | 用户身份信息可能以JSON形式透出，比如：`userInfo:{ userCode:"001" }`,当前例子`graySubKey`取值为`userCode` |
| `rules`      | array of object | 非必填  | -   | 用户定义不同的灰度规则，适配不同的灰度场景                                                             |
| `baseDeployment` | object   | 非必填  | -   | 配置Base基线规则的配置                                    |
| `grayDeployments` |  array of object   | 非必填  | -   | 配置Gray灰度的生效规则，以及生效版本                                                         |

`rules`字段配置说明：

| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                |
|----------------|--------------|------|-----|-----------------------------------------------------------------------------------|
| `name`         | string       | 必填   | -   | 规则名称唯一标识，和`deploy.gray[].name`进行关联生效                                          |
| `grayKeyValue`    | array of string       | 非必填   | -   | 用户ID 白名单列表 |
| `grayTagKey`      | string | 非必填  | -   | 用户分类打标的标签key值，来自Cookie                                                             |
| `grayTagValue` | array of string   | 非必填  | -   | 用户分类打标的标签value值，来自Cookie                                                         |

`baseDeployment`字段配置说明：

| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                |
|----------------|--------------|------|-----|-----------------------------------------------------------------------------------|
| `version`         | string       | 必填   | -   | Base版本的版本号，作为兜底的版本                                         |

`grayDeployments`字段配置说明：

| 名称     | 数据类型   | 填写要求 | 默认值 | 描述                         |
|--------|--------|------|-----|----------------------------|
| `version`  | string | 必填   | -   | Gray版本的版本号，如果命中灰度规则，则使用此版本 |
| `name` | string | 必填   | -   | 规则名称和`rules[].name`关联，     |
| `enabled`  | boolean   | 必填   | -   | 是否启动当前灰度规则                 |

## 配置示例
### 基础配置
```yml
grayKey: userid
rules:
- name: inner-user
  grayKeyValue:
  - '00000001'
  - '00000005'
- name: beta-user
  grayKeyValue:
  - '00000002'
  - '00000003'
  grayTagKey: level
  grayTagValue:
  - level3
  - level5
baseDeployment:
  version: base
grayDeployments:
  - name: beta-user
    version: gray
    enabled: true
```


cookie中的用户唯一标识为 `userid`，当前灰度规则配置了`beta-user`的规则。

当满足下面调试的时候，会使用`version: gray`版本
- cookie中`userid`等于`00000002`或者`00000003`
- cookie中`level`等于`level3`或者`level5`的用户

否则使用`version: base`版本

### 用户信息存在JSON中

```yml
grayKey: appInfo
graySubKey: userId
rules:
- name: inner-user
  grayKeyValue:
  - '00000001'
  - '00000005'
- name: beta-user
  grayKeyValue:
  - '00000002'
  - '00000003'
  grayTagKey: level
  grayTagValue:
  - level3
  - level5
baseDeployment:
  version: base
grayDeployments:
  - name: beta-user
    version: gray
    enabled: true
```

cookie存在`appInfo`的JSON数据，其中包含`userId`字段为当前的唯一标识
当前灰度规则配置了`beta-user`的规则。
当满足下面调试的时候，会使用`version: gray`版本
- cookie中`userid`等于`00000002`或者`00000003`
- cookie中`level`等于`level3`或者`level5`的用户

否则使用`version: base`版本