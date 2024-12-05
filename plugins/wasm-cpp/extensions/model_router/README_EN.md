## Function Description
The `model-router` plugin implements the function of routing based on the model parameter in the LLM protocol.

## Configuration Fields

| Name                 | Data Type        | Filling Requirement                | Default Value                   | Description                                                  |
| -----------          | --------------- | -----------------------            | ------                          | -------------------------------------------                  |
| `modelKey`           | string           | Optional                           | model                           | The location of the model parameter in the request body       |
| `addProviderHeader`  | string           | Optional                           | -                               | Which request header to place the provider name parsed from the model parameter |
| `modelToHeader`      | string           | Optional                           | -                               | Which request header to directly place the model parameter    |
| `enableOnPathSuffix` | array of string  | Optional                           | ["/v1/chat/completions"]        | Only effective for requests with these specific path suffixes |

## Runtime Attributes

Plugin execution phase: Authentication phase
Plugin execution priority: 900

## Effect Description

### Routing Based on the model Parameter

The following configuration is required:

```yaml
modelToHeader: x-higress-llm-model
```

The plugin will extract the model parameter from the request and set it in the x-higress-llm-model request header, which can be used for subsequent routing. For example, the original LLM request body:

```json
{
    "model": "qwen-long",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "What is the GitHub address of the main repository for the higress project"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
```

After processing by this plugin, the following request header (which can be used for route matching) will be added:

x-higress-llm-model: qwen-long

### Extracting the provider Field from the model Parameter for Routing

> Note that this mode requires the client to specify the provider using a `/` separator in the model parameter.

The following configuration is required:

```yaml
addProviderHeader: x-higress-llm-provider
```

The plugin will extract the provider part (if present) from the model parameter in the request and set it in the x-higress-llm-provider request header, which can be used for subsequent routing, and rewrite the model parameter to the model name part. For example, the original LLM request body:

```json
{
    "model": "dashscope/qwen-long",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "What is the GitHub address of the main repository for the higress project"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
```

After processing by this plugin, the following request header (which can be used for route matching) will be added:

x-higress-llm-provider: dashscope

The original LLM request body will be changed to:

```json
{
    "model": "qwen-long",
    "frequency_penalty": 0,
    "max_tokens": 800,
    "stream": false,
    "messages": [{
        "role": "user",
        "content": "What is the GitHub address of the main repository for the higress project"
    }],
    "presence_penalty": 0,
    "temperature": 0.7,
    "top_p": 0.95
}
