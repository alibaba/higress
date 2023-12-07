<p>
   English | <a href="README.md">中文</a>
</p>

# Description
`key-rate-limit` plugin implements a rate-limiting function based on specific key-values. The key-values may come from URL parameters or HTTP headers.

# Configuration Fields

| Name | Type | Requirement |  Default Value | Description |
| -------- | -------- | -------- | -------- | -------- |
|  limit_by_header     |  string     | Optional. Choose one from following: `limit_by_header`, `limit_by_param`. |   -  |  The name of HTTP header used to obtain key-value used in rate-limiting. |
|  limit_by_param     |  string     | Optional. Choose one from following: `limit_by_header`, `limit_by_param`. |   -  |  The name of URL parameter used to obtain key-value used in rate-limiting.   |
|  limit_keys     |  array of object     | Required     |   -  |  Rate-limiting thresholds when matching specific key-values |

Field descriptions of `limit_keys` items:

| Name | Type | Requirement |  Default Value | Description |
| -------- | -------- | -------- | -------- | -------- |
|  key     |  string     | Required     |   -  |  Value to match of the specific key |
|  query_per_second     |  number     | Optional. Choose one from following: `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day`. |   -  |  Number of requests allowed per second |
|  query_per_minute     |  number     | Optional. Choose one from following: `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day`. |   -  |  Number of requests allowed per minute |
|  query_per_hour     |  number     | Optional. Choose one from following: `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day`. |   -  |  Number of requests allowed per hour |
|  query_per_day     |  number     | Optional. Choose one from following: `query_per_second`, `query_per_minute`, `query_per_hour`, `query_per_day`. |   -  |  Number of requests allowed per day |

# Configuration Samples

## Use query parameter `apikey` for rate-limiting
```yaml
limit_by_param: apikey
limit_keys:
- key: 9a342114-ba8a-11ec-b1bf-00163e1250b5
  query_per_second: 10
- key: a6a6d7f2-ba8a-11ec-bec2-00163e1250b5
  query_per_minute: 100
```

## Use HTTP header parameter `x-ca-key` for rate-limiting
```yaml
limit_by_header: x-ca-key
limit_keys:
- key: 102234
  query_per_second: 10
- key: 308239
  query_per_hour: 10
```

## Error Codes

| HTTP code | Error Message                                                                         | Reason              |
| ----------- |--------------------------------------------------------------------------------| ---------------------- |
| 429         | Too many requests. | Too many requests.          |
