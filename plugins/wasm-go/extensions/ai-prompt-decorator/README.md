# 简介
AI提示词装饰器插件，支持在LLM的请求前后插入prompt。

# 配置说明

| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
|----------------|-----------------|------|-----|----------------------------------|
| `prepend` | array of message object | 必填 | - | 在初始输入之前插入的语句 |
| `append` | array of message object | 必填 | - | 在初始输入之后插入的语句 |

message object 配置说明：

| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
|----------------|-----------------|------|-----|----------------------------------|
| `role` | string | 必填 | - | 角色 |
| `content` | string | 必填 | - | 消息 |

# 示例

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