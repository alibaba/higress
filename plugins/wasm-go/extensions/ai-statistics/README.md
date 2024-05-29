# AI可观测
## Metrics
示例：
```
wasmcustom.route.dashscope.upstream.qwen.model.qwen-turbo.input_token: 28
wasmcustom.route.dashscope.upstream.qwen.model.qwen-turbo.output_token: 52
```

## Logs
示例：
```yaml
inline_string: '{"model":"%FILTER_STATE(wasm.model:PLAIN)%","input_token":"%FILTER_STATE(wasm.input_token:PLAIN)%"}'
```