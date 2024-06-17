# 简介
AI提示词模板，用于快速构建同类型的AI请求。

# 配置说明
| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
|----------------|-----------------|------|-----|----------------------------------|
| `templates` | array of object | 必填 | - | 模板设置 |

template object 配置说明：

| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
|----------------|-----------------|------|-----|----------------------------------|
| `name` | string | 必填 | - | 模板名称 |
| `template.model` | string | 必填 | - | 模型名称 |
| `template.messages` | array of object | 必填 | - | 大模型输入 |

message object 配置说明：

| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
|----------------|-----------------|------|-----|----------------------------------|
| `role` | string | 必填 | - | 角色 |
| `content` | string | 必填 | - | 消息 |

配置示例如下：

```yaml
templates:
- name: "developer-chat"
  template:
    model: gpt-3.5-turbo
    messages:
    - role: system
      content: "You are a {{program}} expert, in {{language}} programming language."
    - role: user
      content: "Write me a {{program}} program."
```

使用以上配置的请求body示例：

```json
{
  "template": "developer-chat",
  "properties": {
    "program": "quick sort",
    "language": "python"
  }
}
```