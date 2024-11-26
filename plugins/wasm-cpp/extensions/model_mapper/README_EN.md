## Function Description
The `model-mapper` plugin implements the functionality of routing based on the model parameter in the LLM protocol.

## Configuration Fields

| Name                 | Data Type        | Filling Requirement                | Default Value                   | Description                                                                                                                                                                                                                                                         |
| -----------          | --------------- | -----------------------            | ------                          | -------------------------------------------                                                                                                                                                                                                                  |
| `modelKey`           | string          | Optional                           | model                           | The location of the model parameter in the request body.                                                                                                                                                                                                            |
| `modelMapping`       | map of string   | Optional                           | -                               | AI model mapping table, used to map the model names in the request to the model names supported by the service provider.<br/>1. Supports prefix matching. For example, use "gpt-3-*" to match all models whose names start with “gpt-3-”;<br/>2. Supports using "*" as the key to configure a generic fallback mapping relationship;<br/>3. If the target name in the mapping is an empty string "", it means to keep the original model name. |
| `enableOnPathSuffix` | array of string | Optional                           | ["/v1/chat/completions"]        | Only applies to requests with these specific path suffixes.                                                                                                                                           |

## Runtime Properties

Plugin execution phase: Authentication phase
Plugin execution priority: 800

## Effect Description

With the following configuration:

```yaml
modelMapping:
  'gpt-4-*': "qwen-max"
  'gpt-4o': "qwen-vl-plus"
  '*': "qwen-turbo"
```

After enabling, model parameters starting with `gpt-4-` will be rewritten to `qwen-max`, `gpt-4o` will be rewritten to `qwen-vl-plus`, and all other models will be rewritten to `qwen-turbo`.

For example, if the original request was:

```json
{
    "model": "gpt-4o",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "What is the GitHub address of the main repository for the higress project?"
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
        "content": "What is the GitHub address of the main repository for the higress project?"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
```
