## Function Description
The `model-router` plugin implements the functionality of routing based on the `model` parameter in the LLM protocol.

## Runtime Properties

Plugin Execution Phase: `Default Phase`
Plugin Execution Priority: `260`

## Configuration Fields

| Name               | Data Type   | Filling Requirement | Default Value         | Description                                           |
| -------------------- | ------------- | --------------------- | ---------------------- | ----------------------------------------------------- |
| `enable`            | bool        | Optional             | false                 | Whether to enable routing based on the `model` parameter |
| `model_key`         | string      | Optional             | model                 | The location of the `model` parameter in the request body |
| `add_header_key`    | string      | Optional             | x-higress-llm-provider | The header where the parsed provider name from the `model` parameter will be placed |

## Effect Description

To enable routing based on the `model` parameter, use the following configuration:

```yaml
enable: true
```

After enabling, the plugin extracts the provider part (if any) from the `model` parameter in the request, and sets it in the `x-higress-llm-provider` request header for subsequent routing. It also rewrites the `model` parameter to the model name part. For example, the original LLM request body is:

```json
{
    "model": "openai/gpt-4o",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "What is the GitHub address for the main repository of the Higress project?"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
```

After processing by the plugin, the following request header (which can be used for routing matching) will be added:

`x-higress-llm-provider: openai`

The original LLM request body will be modified to:

```json
{
    "model": "gpt-4o",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "What is the GitHub address for the main repository of the Higress project?"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
```
