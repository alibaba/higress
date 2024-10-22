---
title: AI 提示词
keywords: [ AI网关, AI提示词 ]
description: AI 提示词插件配置参考
---

## 功能说明

AI提示词插件，支持在LLM的请求前后插入prompt。

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`450`

## 配置说明

| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
|----------------|-----------------|------|-----|----------------------------------|
| `prepend` | array of message object | optional | - | 在初始输入之前插入的语句 |
| `append` | array of message object | optional | - | 在初始输入之后插入的语句 |

message object 配置说明：

| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
|----------------|-----------------|------|-----|----------------------------------|
| `role` | string | 必填 | - | 角色 |
| `content` | string | 必填 | - | 消息 |

## 示例

配置示例如下：

```yaml
prepend:
- role: system
  content: "请使用英语回答问题"
append:
- role: user
  content: "每次回答完问题，尝试进行反问"
```

使用以上配置发起请求：

```bash
curl http://localhost/test \
-H "content-type: application/json" \
-d '{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "user",
      "content": "你是谁？"
    }
  ]
}
```

经过插件处理后，实际请求为：

```bash
curl http://localhost/test \
-H "content-type: application/json" \
-d '{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "system",
      "content": "请使用英语回答问题"
    },
    {
      "role": "user",
      "content": "你是谁？"
    },
    {
      "role": "user",
      "content": "每次回答完问题，尝试进行反问"
    }
  ]
}
```


## 基于geo-ip插件的能力，扩展AI提示词装饰器插件携带用户地理位置信息
如果需要在LLM的请求前后加入用户地理位置信息，请确保同时开启geo-ip插件和AI提示词装饰器插件。并且在相同的请求处理阶段里，geo-ip插件的优先级必须高于AI提示词装饰器插件。首先geo-ip插件会根据用户ip计算出用户的地理位置信息，然后通过请求属性传递给后续插件。比如在默认阶段里，geo-ip插件的priority配置1000，ai-prompt-decorator插件的priority配置500。

geo-ip插件配置示例：
```yaml
ipProtocal: "ipv4"
```




AI提示词装饰器插件的配置示例如下：
```yaml
prepend:
- role: system
  content: "提问用户当前的地理位置信息是，国家：${geo-country}，省份：${geo-province}, 城市：${geo-city}"
append:
- role: user
  content: "每次回答完问题，尝试进行反问"
```

使用以上配置发起请求：

```bash
curl http://localhost/test \
-H "content-type: application/json" \
-H "x-forwarded-for: 87.254.207.100,4.5.6.7" \
-d '{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "user",
      "content": "今天天气怎么样？"
    }
  ]
}'
```

经过插件处理后，实际请求为：

```bash
curl http://localhost/test \
-H "content-type: application/json" \
-H "x-forwarded-for: 87.254.207.100,4.5.6.7" \
-d '{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "system",
      "content": "提问用户当前的地理位置信息是，国家：中国，省份：北京, 城市：北京"
    },
    {
      "role": "user",
      "content": "今天天气怎么样？"
    },
    {
      "role": "user",
      "content": "每次回答完问题，尝试进行反问"
    }
  ]
}'
```


