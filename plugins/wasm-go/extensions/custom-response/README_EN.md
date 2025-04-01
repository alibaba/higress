---
title: Custom Response
keywords: [higress, custom response]
description: Custom response plugin configuration reference
---
## Function Description
The `custom-response` plugin supports the configuration of custom responses, including custom HTTP response status codes, HTTP response headers, and HTTP response bodies. It can be used for Mock responses or for providing custom responses based on specific status codes, such as implementing custom responses when triggering the gateway rate-limiting policy.

## Running Attributes
Plugin Execution Phase: `Authentication Phase`

Plugin Execution Priority: `910`

## Configuration Fields
| Name | Data Type | Requirements | Default Value | Description                                                                                        |
| -------- | -------- | -------- | -------- |----------------------------------------------------------------------------------------------------|
|  status_code    |  number     |  Optional      |   200  | Custom HTTP response status code                                                                   |
|  headers     |  array of string      |  Optional     |   -  | Custom HTTP response headers, keys and values separated by `=`                                     |
|  body      |  string    |  Optional     |   -   | Custom HTTP response body                                                                          |
|  enable_on_status   |  array of number    |   Required     |  -  | Match original status codes to generate custom responses; if not specified, the plugin not enabled |

## Configuration Example
### Mock Response Scenario
```yaml
- body: '{"hello":"world 200"}'
  enable_on_status:
    - 200
    - 201
  headers:
    - key1=value1
    - key2=value2
  status_code: 200
- body: '{"hello":"world 404"}'
  enable_on_status:
    - 404
  headers:
    - key1=value1
    - key2=value2
  status_code: 200
```
With this configuration, 200/201 response will return the following custom response:
```text
HTTP/1.1 200 OK
Content-Type: application/json
key1: value1
key2: value2
Content-Length: 21

{"hello":"world 200"}
```
With this configuration, 404 response will return the following custom response:
```text
HTTP/1.1 200 OK
Content-Type: application/json
key1: value1
key2: value2
Content-Length: 21

{"hello":"world 404"}
```
### Custom Response on Rate Limiting
```yaml
enable_on_status:
- 429
status_code: 302
headers:
- Location=https://example.com
```
When the gateway rate limiting is triggered, it generally returns the `429` status code, and the request will return the following custom response:
```text
HTTP/1.1 302 Found
Location: https://example.com
```
This achieves the goal of redirecting users who have been rate-limited to another page based on the browser's 302 redirect mechanism, which could be a static page on a CDN.

If you wish to return other responses normally when rate limiting is triggered, just refer to the Mock response scenario to configure the relevant fields accordingly.
