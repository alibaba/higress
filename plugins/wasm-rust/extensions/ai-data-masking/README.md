# 功能说明
- 内置敏感词，提问和回答中包含敏感词时进行拦截
- 支持自定义敏感词
- 实现了基于用户配置的规则进行敏感数据替换，保证敏感数据不出域
- 替换支持正则变量，使用场景：手机号码只隐藏中间4位
- 用户输入的数据进入大模型前进行脱敏，某些场景可以对替换结果进行记录，大模型返回的数据可以恢复为原始值，让用户看起来更友好

# 配置字段

| 名称 | 数据类型 | 默认值 | 描述 |
| -------- | --------  | -------- | -------- |
|  system_deny            | bool            |   -  |  开启内置拦截规则  数据来源：https://github.com/houbb/sensitive-word/tree/master/src/main/resources  |
|  deny_code              | int             | 200  |  拦截时http状态码   |
|  deny_message           | bool            | 提问或回答中包含敏感词，已被屏蔽 |  拦截时ai返回消息   |
|  deny_words             | array of string | []   |  自定义敏感词列表  |
|  replace_roles          | array           |   -  |  自定义敏感词正则替换  |
|  replace_roles.regex    | string          |   -  |  规则正则(内置GROK规则)  |
|  replace_roles.type     | [replace, hash] |   -  |  替换类型  |
|  replace_roles.restore  | bool            | false|  是否恢复  |
|  replace_roles.value    | string          |   -  |  替换值（支持正则变量）  |

# 配置示例

```yaml
system_deny: true
deny_code: 200
deny_message: "提问或回答中包含敏感词，已被屏蔽"
deny_words: 
  - "自定义敏感词1"
  - "自定义敏感词2"
replace_roles:
  - regex: "%{MOBILE}"
    type: "replace"
    value: "****"
    # 手机号  13800138000 -> ****
  - regex: "%{EMAILLOCALPART}@%{HOSTNAME:domain}"
    type: "replace"
    restore: true
    value: "****@$domain"
    # 电子邮箱  admin@gmail.com -> ****@gmail.com
  - regex: "%{IP}"
    type: "replace"
    restore: true
    value: "***.***.***.***"
    # ip 192.168.0.1 -> ***.***.***.***
  - regex: "%{IDCARD}"
    type: "replace"
    value: "****"
    # 身份证号 110000000000000000 -> ****
  - regex: "sk-[0-9a-zA-Z]*"
    restore: true
    type: "hash"
    # hash sk-12345 -> 9cb495455da32f41567dab1d07f1973d
    # hash后的值提供给大模型，从大模型返回的数据中会将hash值还原为原始值
```

# 替换测试

## 用户请求内容

  请将 `curl http://172.20.5.14/api/openai/v1/chat/completions -H "Authorization: sk-12345" -H "Auth: test@gmail.com"` 改成post方式

## 处理后请求大模型内容

  `curl http://***.***.***.***/api/openai/v1/chat/completions -H "Authorization: 48a7e98a91d93896d8dac522c5853948" -H "Auth: ****@gmail.com"` 改成post方式

## 大模型返回内容

  您想要将一个 `curl` 的 GET 请求转换为 POST 请求，并且这个请求是向一个特定的 API 发送数据。下面是修改后的 `curl` 命令，以 POST 方式发送：

```sh
curl -X POST \
     -H "Authorization: 48a7e98a91d93896d8dac522c5853948" \
     -H "Auth: ****@gmail.com" \
     -H "Content-Type: application/json" \
     -d '{"key":"value"}' \
     http://***.***.***.***/api/openai/v1/chat/completions
```

这里做了如下几个修改:

- `-X POST` 设置请求方式为 POST。
- `-H "Content-Type: application/json"` 设置请求头中的 `Content-Type` 为 `application/json`，这通常用来告诉服务器您发送的数据格式是 JSON。
- `-d '{"key":"value"}'` 这里设置了要发送的数据，`'{"key":"value"}'` 是一个简单的 JSON 对象示例。您需要将其替换为您实际想要发送的数据。

请注意，您需要将 `"key":"value"` 替换为您实际要发送的数据内容。如果您的 API 接受不同的数据结构或者需要特定的字段，请根据实际情况调整这部分内容。

## 处理后返回用户内容

  您想要将一个 `curl` 的 GET 请求转换为 POST 请求，并且这个请求是向一个特定的 API 发送数据。下面是修改后的 `curl` 命令，以 POST 方式发送：

```sh
curl -X POST \
     -H "Authorization: sk-12345" \
     -H "Auth: test@gmail.com" \
     -H "Content-Type: application/json" \
     -d '{"key":"value"}' \
     http://172.20.5.14/api/openai/v1/chat/completions
```

这里做了如下几个修改:

- `-X POST` 设置请求方式为 POST。
- `-H "Content-Type: application/json"` 设置请求头中的 `Content-Type` 为 `application/json`，这通常用来告诉服务器您发送的数据格式是 JSON。
- `-d '{"key":"value"}'` 这里设置了要发送的数据，`'{"key":"value"}'` 是一个简单的 JSON 对象示例。您需要将其替换为您实际想要发送的数据。

请注意，您需要将 `"key":"value"` 替换为您实际要发送的数据内容。如果您的 API 接受不同的数据结构或者需要特定的字段，请根据实际情况调整这部分内容。
