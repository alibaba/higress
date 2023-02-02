<p>
   English | <a href="README.md">中文</a>
</p>

# Description
`custom-response` plugin implements a function of sending custom responses, including custom HTTP response status codes, HTTP response headers and HTTP response body, which can be used in the scenarios of response mocking and sending a custom response for specific status codes, such as customizing the response for rate-limited requests.

# Configuration Fields

| Name | Type | Requirement |  Default Value | Description |
| -------- | -------- | -------- | -------- | -------- |
|  status_code    |  number     |  Optional      |   200  |  Custom HTTP response status code   |
|  headers     |  array of string      |  Optional     |   -  |  Custom HTTP response header. Key and value shall be separated using `=`.   |
|  body      |  string    |  Optional     |   -   |  Custom HTTP response body  |
|  enable_on_status   |  array of number    |   Optional     |  -  | The original response status code to match. Generate the custom response only the actual status code matches the configuration. Ignore the status code match if left unconfigured.   |

# Configuration Samples

## Mock Responses

```yaml
status_code: 200
headers:
- Content-Type=application/json
- Hello=World
body: "{\"hello\":\"world\"}"

```

According to the configuration above, all the requests will get the following custom response:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Hello: World
Content-Length: 17

{"hello":"world"}
```

## Send a Custom Response when Rate-Limited

```yaml
enable_on_status: 
- 429
status_code: 302
headers:
- Location=https://example.com
```

When rate-limited, normally gateway will return a status code of `429` . Now, rate-limited requests will get the following custom response:

```text
HTTP/1.1 302 Found
Location: https://example.com
```

So based on the 302 redirecting mechanism provided by browsers, this can redirect rate-limited users to other pages, for example, a static page hosted on CDN.

If you'd like to send other responses when rate-limited, please add other fields into the configuration, referring to the Mock Responses scenario.

## Only Enabled for Specific Routes or Domains
```yaml
# Use _rules_ field for fine-grained rule configurations 
_rules_:
# Rule 1: Match by route name
- _match_route_:
  - route-a
  - route-b
  body: "{\"hello\":\"world\"}"
# Rule 2: Match by domain
- _match_domain_:
  - "*.example.com"
  - test.com
  enable_on_status: 
  - 429
  status_code: 200
  headers:
  - Content-Type=application/json
  body: "{\"errmsg\": \"rate limited\"}"
```
In the rule sample of `_match_route_`, `route-a` and `route-b` are the route names provided when creating a new gateway route. When the current route names matches the configuration, the rule following shall be applied.
In the rule sample of `_match_domain_`, `*.example.com` and `test.com` are the domain names used for request matching. When the current domain name matches the configuration, the rule following shall be applied.
All rules shall be checked following the order of items in the `_rules_` field, The first matched rule will be applied. All remained will be ignored.
