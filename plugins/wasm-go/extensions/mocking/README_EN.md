## Functionality Description

The `mocking` plugin is used to simulate APIs. When this plugin is executed, it will return mock data in a specified format that meets the matching criteria, and the request will not be forwarded to the upstream.

## Runtime Attributes

Plugin execution phase: `default phase`
Plugin execution priority: `205`

## Configuration Description

| Configuration Item      | Type            | Required | Default Value      | Description                                                                   |
|-------------------------|----------------|----------|-------------------|-------------------------------------------------------------------------------|
| responses               | array of object | Yes     | -                 | Collection of responses for the mocking plugin, allowing specification of multiple condition responses      |
| with_mock_header        | bool            | No      | true              | When set to true, the response header "x-mock-by: higress" will be added. When set to false, this response header will not be added. |

Description of each configuration field in the `responses`.

| Configuration Item      | Type              | Required | Default Value                                       | Description                                                                    |
|-------------------------|------------------|----------|-----------------------------------------------------|--------------------------------------------------------------------------------|
| trigger                 | object           | No       | -                                                   | Collection of matching criteria                                                |
| body                    | string           | No       | {"hello":"world"}                                   | Response body sent to the client                                               |
| headers                 | array of object  | No       | [{"key":"content-type","value":"application/json"}] | Response headers sent to the client                                            |
| status_code             | int              | No       | 200                                                 | HTTP response code sent to the client                                          |

Description of each configuration field in the `trigger`.

| Configuration Item      | Type            | Required  | Default Value | Description             |
|-------------------------|-----------------|-----------|---------------|--------------------------|
| headers                 | array of object | No        | -             | Matching request headers |
| queries                 | array of object | No        | -             | Matching request query params |

Description of each configuration field in `trigger.headers`.

| Configuration Item      | Type    | Required | Default Value | Description         |
|-------------------------|---------|----------|---------------|---------------------|
| key                     | string  | No       | -             | Request header key  |
| value                   | string  | No       | -             | Request header value |

Description of each configuration field in `trigger.queries`.

| Configuration Item      | Type    | Required | Default Value | Description           |
|-------------------------|---------|----------|---------------|-----------------------|
| key                     | string  | No      | -             | Request parameter key |
| value                   | string  | No       | -             | Request parameter value |

Description of each configuration field in `response.headers`.

| Configuration Item      | Type    | Required | Default Value | Description                |
|-------------------------|---------|----------|---------------|----------------------------|
| key                     | string  | No       | -             | New response header key    |
| value                   | string  | No       | -             | New response header value  |

## Configuration Example

### Plugin Example Configuration

```yaml
responses:
  -
    trigger:
      headers:
        -
          key: header1
          value: value1
      queries:
        -
          key: queryKey1
          value: queryValue1
    body: "test"
    headers:
      -
        key: "content-type"
        value: "text/plain"
    status_code: 200
```
If none of the specified conditions in the trigger are met, the default message body `{"hello":"world"}` will be returned along with the response header`content-type: application/json`.
