# traffic-tag插件配置文档

本文档提供了 `traffic-tag` 插件的配置选项的详细信息。此配置用于管理基于灰度发布和测试目的定义的特定规则的流量标记。

## 配置概览

`traffic-tag` 插件允许根据百分比或特定内容条件定义流量标记规则。它支持复杂的逻辑来确定如何根据用户定义的标准标记流量。

## 字段描述
此部分提供了配置字段的详细描述。

| 字段名称      | 类型     | 必填    | 默认值       | 有效值                  | 描述                                                                     |
|-------------|--------|-------|-----------|----------------------|------------------------------------------------------------------------|
| `taggingType` | string    | 是     | 无         | `content`, `percentage` | **目前仅支持`content`。** 确定应用的标记规则类型，可以是基于百分比或特定内容条件。     |
| `percentage`  | object     | 否     | 无         | -                    | 定义要标记的流量百分比，仅在 `taggingType` 设置为 `percentage` 时使用。此字段包含子字段 `value`。 |
| `matchLogic`  | string    | 否     | `or`       | `and`, `or`          | 应用于 `content` 条件中的条件之间的逻辑操作符。默认为 `or`。         |
| `tagValue`    | string    | 是     | 无         | -                    | 如果所有条件都满足，则应用于 `x-mse-tag` 标头的值。            |
| `conditions`  | array[condition]     | 是     | 无         | -                    | 定义如何匹配要标记的请求的条件列表。              |

#### `condition` 字段
| 字段名称   | 类型     | 必填 | 默认值 | 有效值                         | 描述                                                         |
|----------|--------|----|-------|-----------------------------|------------------------------------------------------------|
| `type`    | string    | 是  | 无     | `header`, `parameter`, `cookie`, ~~` body `~~ | 匹配条件的类型。                      |
| `key`     | string    | 否  | 无     | -                           | 对于像 `header`, `parameter` 这样的条件，要检查的键。             |
| `matchType` | string    | 是  | ==     | `prefix`, `==`, `!=`, `in`, `noIn`, `regex`, `percentage`      | 用于条件的匹配方式。            |
| `value`   | []string    | 是  | 无     | -                           | 根据 `matchType` 可能变化的匹配值。除匹配类型"in"和"notin"外，默认value只有一个值。当matchType == "percentage"时，value只能是单个数字。  |

## 本地测试
本地测试环境部署参考 https://higress.io/zh-cn/docs/user/wasm-go#%E4%B8%89%E6%9C%AC%E5%9C%B0%E8%B0%83%E8%AF%95

### 测试用例-1：多条件逻辑或
插件部分配置如下，完整配置文件见`./test/envoy-test-1.yaml`
```
{
"taggingType": "content",
"percent": 50,
"matchLogic": "or",
"tagKey": "x-mse-tag",
"tagValue": "gray",
"conditions": [
{
    "keyType": "header",
    "key": "X-User-Type",
    "matchType": "prefix",
    "value": ["test"]
},
{
    "keyType": "header",
    "key": "feature",
    "matchType": "==",
    "value": ["new-ui"]
},
{
    "keyType": "parameter",
    "key": "role",
    "matchType": "in",
    "value": ["admin", "super-admin"]
}
]
}
// 配置描述：当请求头中的X-User-Type以test开头，请求头中的feature等于new-ui，请求参数中的role等于admin或super-admin三个条件满足其一时，将请求标记为gray
```

使用curl通过网关访问httpbin，可以看到经过网关处理后的请求头的内容。
```
// 包含请求参数'role=super-admin'
curl --location --request GET 'http://127.0.0.1:10000/get?role=super-admin'
```
```
// 包含请求头'X-User-Type: testtttt'
curl --location --request GET 'http://127.0.0.1:10000/get' \
--header 'X-User-Type: testtttt'
```
```
// 包含请求头'feature: new-ui'
curl --location --request GET 'http://127.0.0.1:10000/get' \
--header 'feature: new-ui'
```
```
// 不满足条件，网关处理后未添加'x-mse-tag'
curl --location --request GET 'http://127.0.0.1:10000/get' \
--header 'feature: old-ui'

{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Accept-Encoding": "gzip, deflate, br",
    "Feature": "old-ui",
    "Host": "127.0.0.1:10000",
    "Original-Host": "127.0.0.1:10000",
    "Req-Start-Time": "1718779564653",
    "User-Agent": "Apifox/1.0.0 (https://apifox.com)",
    "X-Envoy-Expected-Rq-Timeout-Ms": "15000"
  },
  "origin": "172.19.0.3",
  "url": "https://127.0.0.1:10000/get"
}
```

### 测试用例-2：多条件逻辑与
插件部分配置如下：
```
{
    "taggingType": "content",
    "matchLogic": "and",
    "tagKey": "x-mse-tag",
    "tagValue": "special",
    "conditions": [
    {
        "keyType": "header",
        "key": "User-Id",
        "matchType": "prefix",
        "value": ["super"]
    },
    {
        "keyType": "parameter",
        "key": "session",
        "matchType": "!=",
        "value": ["expired"]
    },
    {
        "keyType": "cookie",
        "key": "user_role",
        "matchType": "in",
        "value": ["admin", "editor"]
    }
    ]
}
// 配置描述：这个配置文件设置了三个条件，必须同时满足这些条件才会标记流量
```

发送测试请求：
```
// 满足所有条件
curl --location --request GET 'http://127.0.0.1:10000/get?session=valid' \
--header 'User-Id: superMan' \
--header 'Cookie: user_role=editor'
```

```
// session=expired, 不添加tag
curl --location --request GET 'http://127.0.0.1:10000/get?session=expired' \
--header 'User-Id: superMan' \
--header 'Cookie: user_role=editor'
```

### 测试用例-3：百分比
插件部分配置如下：
```
{
"taggingType": "content",
"matchLogic": "or",
"tagKey": "x-mse-tag",
"tagValue": "beta-test",
"conditions": [
{
    "keyType": "header",
    "key": "User-Id",
    "matchType": "percentage",
    "value": ["60"]
}
]
}
// hash("User-Id".value) % 100, 结果小于阈值则添加tag
```

使用curl通过网关访问httpbin，可以看到经过网关处理后的请求头的内容。

```
// 满足，添加tag
curl --location --request GET 'http://127.0.0.1:10000/get' \
--header 'User-Id: qwqwqwqdd2' 
```

```
// 不满足，不添加tag
curl --location --request GET 'http://127.0.0.1:10000/get' \
--header 'User-Id: qwqwqwqdd4' 
```

### 测试用例-4: 正则匹配
插件部分配置如下：
```
{
    "taggingType": "content",
    "matchLogic": "or",
    "tagKey": "x-mse-tag",
    "tagValue": "regex-example",
    "conditions": [
    {
        "keyType": "header",
        "key": "User-Agent",
        "matchType": "regex",
        "value": ["^Mozilla/5\\.0.*"]
    },
    {
        "keyType": "header",
        "key": "Accept-Language",
        "matchType": "regex",
        "value": ["^en-US.*", "^en-GB.*"]
    },
    {
        "keyType": "parameter",
        "key": "session",
        "matchType": "regex",
        "value": ["^[a-zA-Z0-9]{32}$"]
    },
    {
        "keyType": "cookie",
        "key": "session_id",
        "matchType": "regex",
        "value": ["^[a-zA-Z0-9]{16}$"]
    }
    ]
}
```

使用curl通过网关访问httpbin，可以看到经过网关处理后的请求头的内容。

```
// 满足全部条件，添加tag
curl --location --request GET 'http://127.0.0.1:10000/get?session=12345678901234567890123456789012' \
--header 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)' \
--header 'Accept-Language: en-US' \
--header 'Cookie: session_id=abcd1234abcd1234' \
```