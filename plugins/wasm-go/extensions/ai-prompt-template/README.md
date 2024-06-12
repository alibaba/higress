# 简介
AI提示词模板，用于快速构建同类型的AI请求。

# 配置示例
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

使用以上配置的请求示例：

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Host: api.openai.com" \
  -H "Template-Enable: true" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SK" \
  -d '{
    "template": "developer-chat"
    "properties": {
      "program": "quick sort"
      "language": "python"
    }
  }'
```