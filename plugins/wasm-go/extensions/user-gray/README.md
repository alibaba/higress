# user-gray 前端灰度插件
## 功能说明
`user-gray`插件实现了前端用户灰度的的功能，通过此插件，不但可以用于业务`A/B实验`，同时通过`可灰度`配合`可监控`,`可回滚`策略保证系统发布运维的稳定性。

## 配置字段
| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                |
|----------------|--------------|------|-----|-----------------------------------------------------------------------------------|
| `uid-key`         | string       | 必填   | -   | 用户ID的唯一标识，可以来自Cookie或者Header中，比如 userid                                           |
| `uid-sub-key`    | string       | 非必填   | -   | 用户身份信息可能以JSON形式透出，比如：`userInfo:{ userCode:"001" }`,当前例子`uid-sub-key`取值为`userCode` |
| `rules`      | array of map | 非必填  | -   | 用户定义不同的灰度规则，适配不同的灰度场景                                                             |
| `deploy` | map of map   | 非必填  | -   | 分别配置Base基线和Gary灰度的生效规则，以及生效版本                                                         |

`rules`字段配置说明：

| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                |
|----------------|--------------|------|-----|-----------------------------------------------------------------------------------|
| `name`         | string       | 必填   | -   | 规则名称唯一标识，和`deploy.gray[].name`进行关联生效                                          |
| `uid-value`    | array of string       | 非必填   | -   | 用户ID 白名单列表 |
| `gray-tag-key`      | string | 非必填  | -   | 用户分类打标的标签key值，来自Cookie                                                             |
| `gray-tag-value` | array of string   | 非必填  | -   | 用户分类打标的标签value值，来自Cookie                                                         |


`deploy`字段配置说明：

| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                |
|----------------|--------------|------|-----|-----------------------------------------------------------------------------------|
| `base`         | map of string       | 必填   | -   | 定义Base版本，如果匹配不到灰度版本，默认fallback到当前版本                                         |
| `gray`    | array of string       | 非必填   | -   | 定义Gray版本，如果匹配到灰度规则，则当前的灰度版本生效 |

`deploy.base`字段配置说明：

| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                |
|----------------|--------------|------|-----|-----------------------------------------------------------------------------------|
| `version`         | string       | 必填   | -   | Base版本的版本号，作为兜底的版本                                         |

`deploy.gray`字段配置说明：

| 名称     | 数据类型   | 填写要求 | 默认值 | 描述                         |
|--------|--------|------|-----|----------------------------|
| `version`  | string | 必填   | -   | Gray版本的版本号，如果命中灰度规则，则使用此版本 |
| `name` | string | 必填   | -   | 规则名称和`rules[].name`关联，     |
| `enable`  | boolean   | 必填   | -   | 是否启动当前灰度规则                 |

## 配置示例
### 基础配置
```yml
uid-key: userid
rules:
- name: inner-user
  uid-value:
  - '00000001'
  - '00000005'
- name: beta-user
  uid-value:
  - '00000002'
  - '00000003'
  gray-tag-key: level
  gray-tag-value:
  - level3
  - level5
deploy:
  base:
    version: base
  gray:
  - name: beta-user
    version: gray
    enable: true
```


cookie中的用户唯一标识为 `userid`，当前灰度规则配置了`beta-user`的规则。

当满足下面调试的时候，会使用`versin: gray`版本
- cookie中`userid`等于`00000002`或者`00000003`
- cookie中`level`等于`level3`或者`level5`的用户

否则使用`versin: base`版本

### 用户信息存在JSON中

```yml
uid-key: appInfo
uid-sub-key: userId
rules:
- name: inner-user
  uid-value:
  - '00000001'
  - '00000005'
- name: beta-user
  uid-value:
  - '00000002'
  - '00000003'
  gray-tag-key: level
  gray-tag-value:
  - level3
  - level5
deploy:
  base:
    version: base
  gray:
  - name: beta-user
    version: gray
    enable: true
```

cookie存在`appInfo`的JSON数据，其中包含`userId`字段为当前的唯一标识
当前灰度规则配置了`beta-user`的规则。
当满足下面调试的时候，会使用`versin: gray`版本
- cookie中`userid`等于`00000002`或者`00000003`
- cookie中`level`等于`level3`或者`level5`的用户

否则使用`versin: base`版本