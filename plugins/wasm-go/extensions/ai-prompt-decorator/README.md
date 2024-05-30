示例：

```yaml
http_filters:
- name: test
  typed_config:
    "@type": type.googleapis.com/udpa.type.v1.TypedStruct
    type_url: type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
    value:
      config:
        name: wasmdemo
        vm_config:
          runtime: envoy.wasm.runtime.v8
          code:
            local:
              filename: main.wasm
        configuration:
          "@type": "type.googleapis.com/google.protobuf.StringValue"
          value: |
            {
              "decorators": [
                {
                  "name": "hangzhou-guide",
                  "decorator": {
                    "prepend": [
                      {
                        "role": "system",
                        "content": "You will always respond in the Chinese language."
                      },
                      {
                        "role": "user",
                        "content": "Assume you are from Hangzhou."
                      }
                    ],
                    "append": [
                      {
                        "role": "user",
                        "content": "Don't introduce Hangzhou's food."
                      }
                    ]
                  }
                }
              ]
            }
```


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
        "content": "Please introduce you home."
      }
    ]
  }'
  ```