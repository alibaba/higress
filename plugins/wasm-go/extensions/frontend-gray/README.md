---
title: 前端灰度
keywords: [higress,frontend gray]
description: 前端灰度插件配置参考
---

## 功能说明
`frontend-gray`插件实现了前端用户灰度的的功能，通过此插件，不但可以用于业务`A/B实验`，同时通过`可灰度`配合`可监控`,`可回滚`策略保证系统发布运维的稳定性。

## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`450`


## 配置字段
| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                                 |
|----------------|--------------|----|-----|----------------------------------------------------------------------------------------------------|
| `grayKey`         | string       | 非必填 | -   | 用户ID的唯一标识，可以来自Cookie或者Header中，比如 userid，如果没有填写则使用`rules[].grayTagKey`和`rules[].grayTagValue`过滤灰度规则 |
| `graySubKey`    | string       | 非必填 | -   | 用户身份信息可能以JSON形式透出，比如：`userInfo:{ userCode:"001" }`,当前例子`graySubKey`取值为`userCode`                   |
| `rules`      | array of object | 必填 | -   | 用户定义不同的灰度规则，适配不同的灰度场景                                                                              |
| `rewrite`      | object | 必填 | -   | 重写配置，一般用于OSS/CDN前端部署的重写配置                                                                           |
| `baseDeployment` | object   | 非必填 | -   | 配置Base基线规则的配置                                                                                      |
| `grayDeployments` |  array of object   | 非必填 | -   | 配置Gray灰度的生效规则，以及生效版本                                                                               |

`rules`字段配置说明：

| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                |
|----------------|--------------|------|-----|-----------------------------------------------------------------------------------|
| `name`         | string       | 必填   | -   | 规则名称唯一标识，和`deploy.gray[].name`进行关联生效                                          |
| `grayKeyValue`    | array of string       | 非必填   | -   | 用户ID 白名单列表 |
| `grayTagKey`      | string | 非必填  | -   | 用户分类打标的标签key值，来自Cookie                                                             |
| `grayTagValue` | array of string   | 非必填  | -   | 用户分类打标的标签value值，来自Cookie                                                         |

`rewrite`字段配置说明：
> `indexRouting`首页重写和`fileRouting`文件重写，本质都是前缀匹配，比如`/app1`: `/mfe/app1/{version}/index.html`代表/app1为前缀的请求，路由到`/mfe/app1/{version}/index.html`页面上，其中`{version}`代表版本号，在运行过程中会被`baseDeployment.version`或者`grayDeployments[].version`动态替换。

> `{version}` 作为保留字段，在执行过程中会被`baseDeployment.version`或者`grayDeployments[].version`动态替换前端版本。


| 名称         | 数据类型         | 填写要求 | 默认值 | 描述                           |
|------------|--------------|------|-----|------------------------------|
| `host`     | string       | 非必填  | -   | host地址，如果是OSS则设置为 VPC 内网访问地址 |
| `notFoundUri` | string       | 非必填  | -   | 404 页面配置                     |
| `indexRouting`    | map of string to string       | 非必填  | -   | 用于定义首页重写路由规则。每个键 (Key) 表示首页的路由路径，值 (Value) 则指向重定向的目标文件。例如，键为 `/app1` 对应的值为 `/mfe/app1/{version}/index.html`。生效version为`0.0.1`， 访问路径为 `/app1`，则重定向到 `/mfe/app1/0.0.1/index.html`。                     |
| `fileRouting`     | map of string to string       | 非必填  | -   | 用于定义资源文件重写路由规则。每个键 (Key) 表示资源访问路径，值 (Value) 则指向重定向的目标文件。例如，键为 `/app1/` 对应的值为 `/mfe/app1/{version}`。生效version为`0.0.1`，访问路径为 `/app1/js/a.js`，则重定向到 `/mfe/app1/0.0.1/js/a.js`。                     |

`baseDeployment`字段配置说明：

| 名称             | 数据类型         | 填写要求 | 默认值 | 描述                                                                                |
|----------------|--------------|------|-----|-----------------------------------------------------------------------------------|
| `version`         | string       | 必填   | -   | Base版本的版本号，作为兜底的版本                                         |

`grayDeployments`字段配置说明：

| 名称     | 数据类型   | 填写要求 | 默认值 | 描述                                              |
|--------|--------|------|-----|-------------------------------------------------|
| `version`  | string | 必填   | -   | Gray版本的版本号，如果命中灰度规则，则使用此版本。如果是非CDN部署，在header添加`x-higress-tag`                     |
| `backendVersion`  | string | 必填   | -   | 后端灰度版本，会在`XHR/Fetch`请求的header头添加 `x-mse-tag`到后端 |
| `name` | string | 必填   | -   | 规则名称和`rules[].name`关联，                          |
| `enabled`  | boolean   | 必填   | -   | 是否启动当前灰度规则                                      |

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

### rewrite重写配置
> 一般用于CDN部署场景
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
rewrite:
  host: frontend-gray.oss-cn-shanghai-internal.aliyuncs.com
  notFoundUri: /mfe/app1/dev/404.html
  indexRouting:
    /app1: '/mfe/app1/{version}/index.html'
    /: '/mfe/app1/{version}/index.html',
  fileRouting:
    /: '/mfe/app1/{version}'
    /app1/: '/mfe/app1/{version}'
baseDeployment:
  version: base
grayDeployments:
  - name: beta-user
    version: gray
    enabled: true
```

`{version}`会在运行过程中动态替换为真正的版本

#### indexRouting：首页路由配置
访问 `/app1`, `/app123`,`/app1/index.html`, `/app1/xxx`, `/xxxx` 都会路由到'/mfe/app1/{version}/index.html'

#### fileRouting：文件路由配置
下面文件映射均生效
- `/js/a.js` => `/mfe/app1/v1.0.0/js/a.js`
- `/js/template/a.js` => `/mfe/app1/v1.0.0/js/template/a.js`
- `/app1/js/a.js` => `/mfe/app1/v1.0.0/js/a.js`
- `/app1/js/template/a.js` => `/mfe/app1/v1.0.0/js/template/a.js`

