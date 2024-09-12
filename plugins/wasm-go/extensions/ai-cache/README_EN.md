---
title: AI Cache
keywords: [higress,ai cache]
description: AI Cache Plugin Configuration Reference
---
## Function Description
LLM result caching plugin, the default configuration can be directly used for result caching under the OpenAI protocol, and it supports caching of both streaming and non-streaming responses.

## Runtime Properties
Plugin Execution Phase: `Authentication Phase`
Plugin Execution Priority: `10`

## Configuration Description
| Name                              | Type     | Requirement | Default                                                                                                                                                                                                                                                 | Description                                                                                                |
| --------                          | -------- | --------    | --------                                                                                                                                                                                                                                                | --------                                                                                                   |
| cacheKeyFrom.requestBody          | string   | optional    | "messages.@reverse.0.content"                                                                                                                                                                                                                           | Extracts a string from the request Body based on [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) syntax     |
| cacheValueFrom.responseBody       | string   | optional    | "choices.0.message.content"                                                                                                                                                                                                                             | Extracts a string from the response Body based on [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) syntax     |
| cacheStreamValueFrom.responseBody | string   | optional    | "choices.0.delta.content"                                                                                                                                                                                                                               | Extracts a string from the streaming response Body based on [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) syntax |
| cacheKeyPrefix                    | string   | optional    | "higress-ai-cache:"                                                                                                                                                                                                                                     | Prefix for the Redis cache key                                                                                         |
| cacheTTL                          | integer  | optional    | 0                                                                                                                                                                                                                                                       | Cache expiration time in seconds, default value is 0, which means never expire                                                            |
| redis.serviceName                 | string   | required    | -                                                                                                                                                                                                                                                       | The complete FQDN name of the Redis service, including the service type, e.g., my-redis.dns, redis.my-ns.svc.cluster.local               |
| redis.servicePort                 | integer  | optional    | 6379                                                                                                                                                                                                                                                    | Redis service port                                                                                             |
| redis.timeout                     | integer  | optional    | 1000                                                                                                                                                                                                                                                    | Timeout for requests to Redis, in milliseconds                                                                          |
| redis.username                    | string   | optional    | -                                                                                                                                                                                                                                                       | Username for logging into Redis                                                                                        |
| redis.password                    | string   | optional    | -                                                                                                                                                                                                                                                       | Password for logging into Redis                                                                                          |
| returnResponseTemplate            | string   | optional    | `{"id":"from-cache","choices":[%s],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`                                                                                                     | Template for returning HTTP response, with %s marking the part to be replaced by cache value                                              |
| returnStreamResponseTemplate      | string   | optional    | `data:{"id":"from-cache","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}\n\ndata:[DONE]\n\n` | Template for returning streaming HTTP response, with %s marking the part to be replaced by cache value                                          |

## Configuration Example
```yaml  
redis:  
  serviceName: my-redis.dns  
  timeout: 2000  
```  

## Advanced Usage
The current default cache key is based on the GJSON PATH expression: `messages.@reverse.0.content`, meaning to get the content of the first item after reversing the messages array;  
GJSON PATH supports conditional syntax, for instance, if you want to take the content of the last role as user as the key, it can be written as: `messages.@reverse.#(role=="user").content`;  
If you want to concatenate all the content with role as user into an array as the key, it can be written as: `messages.@reverse.#(role=="user")#.content`;  
It also supports pipeline syntax, for example, if you want to take the second role as user as the key, it can be written as: `messages.@reverse.#(role=="user")#.content|1`.  
For more usage, you can refer to the [official documentation](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) and use the [GJSON Playground](https://gjson.dev/) for syntax testing.
