示例：
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