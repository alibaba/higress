# Function Description
The `model-mapper` plugin implements model parameter mapping functionality based on the LLM protocol.

# Configuration Fields

| Name                 | Type            | Requirement             | Default Value            | Description |
| --- | --- | --- | --- | --- |
| `modelKey`           | string          | Optional                | model                    | The position of the model parameter in the request body. |
| `modelMapping`       | map of string   | Optional                | -                        | AI model mapping table, used to map the model name in the request to the model name supported by the service provider.<br/>1. Supports prefix matching. For example, use "gpt-3-*" to match all names starting with "gpt-3-";<br/>2. Supports using "*" as a key to configure a generic fallback mapping;<br/>3. If the target mapping name is an empty string "", it indicates keeping the original model name. |
| `enableOnPathSuffix` | array of string | Optional                | ["/completions","/embeddings","/images/generations","/audio/speech","/fine_tuning/jobs","/moderations","/image-synthesis","/video-synthesis","/rerank","/messages"] | Only effective for requests with these specific path suffixes. |


## Effect Description

Configuration example:

```yaml
modelMapping:
  'gpt-4-*': "qwen-max"
  'gpt-4o': "qwen-vl-plus"
  '*': "qwen-turbo"
```

After enabling, model parameters starting with `gpt-4-` will be replaced with `qwen-max`, `gpt-4o` will be replaced with `qwen-vl-plus`, and all other models will be replaced with `qwen-turbo`.

For example, the original request is:

```json
{
    "model": "gpt-4o",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "What is the github address of the main repository of the higress project"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
```


After processing by this plugin, the original LLM request body will be modified to:

```json
{
    "model": "qwen-vl-plus",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "What is the github address of the main repository of the higress project"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
```
