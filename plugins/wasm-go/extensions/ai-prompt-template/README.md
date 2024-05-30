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
              "templates": [
                {
                  "name": "developer-chat",
                  "template": {
                    "model": "gpt-3.5-turbo",
                    "messages": [
                      {
                        "role": "system",
                        "content": "You are a {{program}} expert, in {{language}} programming language."
                      },
                      {
                        "role": "user",
                        "content": "Write me a {{program}} program."
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