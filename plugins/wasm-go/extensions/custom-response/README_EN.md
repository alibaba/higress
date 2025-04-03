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
### New version - Supports multiple returns
| Name                  | Data Type            | Requirements     | Default Value | Description         |
|---------------------|-----------------|----------|-----|------------|
| rules              | array of object           | Required | -   | rule array |

The configuration field description of `rules` is as follows：

| Name               | Data Type       | Requirements | Default Value | Description                                                                                                                                                                             |
|--------------------|-----------------|--------------|-----|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `status_code`      | number          | Optional     | 200 | Custom HTTP response status code                                                                                                                                                        |
| `headers`          | array of string | Optional     | -   | Custom HTTP response headers, keys and values separated by `=`                                                                                                                          |
| `body`             | string          | Optional     | -   | Custom HTTP response body                                                                                                                                                               |
| `enable_on_status` | array of number | Optional     | -   | Match original status codes to generate custom responses; if not specified, the original status code is not checked, take the first rule with enable_on_status null as the default rule |
| `prefix_on_status` | array of string | Optional     | -   | Fuzzy matching of raw status codes to generate custom responses. You can fill in '2xx' to match the status code between 200 and 299, and '20x' to match the status code between 200 and 209, with x representing any digit |

Matching priority: enable_on_status > prefix_on_status > Default configuration (the first enable_on_status parameter is null)

## Old version - Only one return is supported
| Name | Data Type | Requirements | Default Value | Description                                                                                        |
| -------- | -------- | -------- | -------- |----------------------------------------------------------------------------------------------------|
|  `status_code`    |  number     |  Optional      |   200  | Custom HTTP response status code                                                                   |
|  `headers`     |  array of string      |  Optional     |   -  | Custom HTTP response headers, keys and values separated by `=`                                     |
|  `body`      |  string    |  Optional     |   -   | Custom HTTP response body                                                                          |
|  `enable_on_status`   |  array of number    |   Optional     |  -  | Match original status codes to generate custom responses; if not specified, the original status code is not checked |


## Configuration Example

### Different status codes for different response scenarios

```yaml
rules:
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

According to this configuration 200 201 requests will return a custom response as follows：

```text
HTTP/1.1 200 OK
Content-Type: application/json
key1: value1
key2: value2
Content-Length: 21

{"hello":"world 200"}
```
According to this configuration 404 requests will return a custom response as follows：

```text
HTTP/1.1 200 OK
Content-Type: application/json
key1: value1
key2: value2
Content-Length: 21

{"hello":"world 400"}
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

### Fuzzy matching scene

```yaml
rules:
  - body: '{"hello":"world 2xx"}'
    prefix_on_status:
      - '2xx'
    headers:
      - key1=value1
      - key2=value2
    status_code: 200
  - body: '{"hello":"world 40x"}'
    prefix_on_status:
      - '40x'
    headers:
      - key1=value1
      - key2=value2
    status_code: 200
```

According to this configuration, the status code between 200-299 will return a custom reply as follows：

```text
HTTP/1.1 200 OK
Content-Type: application/json
key1: value1
key2: value2
Content-Length: 21

{"hello":"world 2xx"}
```
According to this configuration, the status code between 401-409 will return a custom reply as follows：

```text
HTTP/1.1 200 OK
Content-Type: application/json
key1: value1
key2: value2
Content-Length: 21

{"hello":"world 40x"}
```

### Mock Response Scenario
```yaml
enable_on_status:
  - 200
status_code: 200
headers:
  - Content-Type=application/json
  - Hello=World
body: "{\"hello\":\"world\"}"
```
With this configuration, 200/201 response will return the following custom response:
```text
HTTP/1.1 200 OK
Content-Type: application/json
key1: value1
key2: value2
Content-Length: 21

{"hello":"world"}
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
