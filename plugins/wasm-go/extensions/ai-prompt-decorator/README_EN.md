---
title: AI Prompts
keywords: [ AI Gateway, AI Prompts ]
description: AI Prompts plugin configuration reference
---
## Function Description
The AI Prompts plugin allows inserting prompts before and after requests in LLM.

## Execution Properties
Plugin execution phase: `Default Phase`  
Plugin execution priority: `450`

## Configuration Description
| Name          | Data Type            | Requirement | Default Value | Description                          |
|---------------|----------------------|-------------|---------------|--------------------------------------|
| `prepend`     | array of message object | optional   | -             | Statements inserted before the initial input |
| `append`      | array of message object | optional   | -             | Statements inserted after the initial input |

Message object configuration description:
| Name      | Data Type   | Requirement | Default Value | Description |
|-----------|-------------|-------------|---------------|-------------|
| `role`    | string      | required    | -             | Role        |
| `content` | string      | required    | -             | Message     |

## Example
An example configuration is as follows:
```yaml
prepend:
- role: system
  content: "Please answer the questions in English."
append:
- role: user
  content: "After answering each question, try to ask a follow-up question."
```

Using the above configuration to initiate a request:
```bash
curl http://localhost/test \
-H "content-type: application/json" \
-d '{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "user",
      "content": "Who are you?"
    }
  ]
}
```

After processing through the plugin, the actual request will be:
```bash
curl http://localhost/test \
-H "content-type: application/json" \
-d '{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "system",
      "content": "Please answer the questions in English."
    },
    {
      "role": "user",
      "content": "Who are you?"
    },
    {
      "role": "user",
      "content": "After answering each question, try to ask a follow-up question."
    }
  ]
}
```

## Based on the geo-ip plugin's capabilities, extend AI Prompt Decorator plugin to carry user geographic location information.
If you need to include user geographic location information before and after the LLM's requests, please ensure both the geo-ip plugin and the AI Prompt Decorator plugin are enabled. Moreover, in the same request processing phase, the geo-ip plugin's priority must be higher than that of the AI Prompt Decorator plugin. First, the geo-ip plugin will calculate the user's geographic location information based on the user's IP, and then pass it to subsequent plugins via request attributes. For instance, in the default phase, the geo-ip plugin's priority configuration is 1000, while the ai-prompt-decorator plugin's priority configuration is 500.

Example configuration for the geo-ip plugin:
```yaml
ipProtocal: "ipv4"
```

An example configuration for the AI Prompt Decorator plugin is as follows:
```yaml
prepend:
- role: system
  content: "The user's current geographic location is, country: ${geo-country}, province: ${geo-province}, city: ${geo-city}."
append:
- role: user
  content: "After answering each question, try to ask a follow-up question."
```

Using the above configuration to initiate a request:
```bash
curl http://localhost/test \
-H "content-type: application/json" \
-H "x-forwarded-for: 87.254.207.100,4.5.6.7" \
-d '{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "user",
      "content": "How is the weather today?"
    }
  ]
}'
```

After processing through the plugin, the actual request will be:
```bash
curl http://localhost/test \
-H "content-type: application/json" \
-H "x-forwarded-for: 87.254.207.100,4.5.6.7" \
-d '{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "system",
      "content": "The user's current geographic location is, country: China, province: Beijing, city: Beijing."
    },
    {
      "role": "user",
      "content": "How is the weather today?"
    },
    {
      "role": "user",
      "content": "After answering each question, try to ask a follow-up question."
    }
  ]
}'
```
