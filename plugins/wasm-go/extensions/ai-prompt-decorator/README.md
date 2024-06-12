# 简介
AI提示词修饰插件，通过在与大模型发起的请求前后插入指定信息来调整大模型的输出。

# 配置示例
```yaml
decorators:
- name: "hangzhou-guide"
  decorator:
    prepend:
    - role: system
      content: "You will always respond in the Chinese language."
    - role: user
      content: "Assume you are from Hangzhou."
    append:
    - role: user
      content: "Don't introduce Hangzhou's food."
```

使用以上配置发起请求：

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Host: api.openai.com" \
  -H "Decorator: hangzhou-guide" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "Please introduce your home."
      }
    ]
  }'
```

响应如下：

```
{
  "id": "chatcmpl-9UYwQlEg6GwAswEZBDYXl41RU4gab",
  "object": "chat.completion",
  "created": 1717071182,
  "model": "gpt-3.5-turbo-0125",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "杭州是一个美丽的城市，有着悠久的历史和富有特色的文化。这里风景优美，有西湖、雷峰塔等著名景点，吸引着许多游客前来观光。杭州人民热情好客，城市宁静安逸，是一个适合居住和旅游的地方。"
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 49,
    "completion_tokens": 117,
    "total_tokens": 166
  },
  "system_fingerprint": null
}
```