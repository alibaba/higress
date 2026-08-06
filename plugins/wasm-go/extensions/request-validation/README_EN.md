---
title: Request Protocol Validation
keywords: [higress,request validation]
description: Configuration reference for request protocol validation plugin
---
## Function Description
The `request-validation` plugin is used to validate requests forwarded to upstream services in advance. This plugin utilizes the `JSON Schema` mechanism for data validation, capable of validating both the body and header data of requests.

## Execution Attributes
Plugin Execution Phase: `Authentication Phase`  
Plugin Execution Priority: `220`

## Configuration Fields
| Name            | Data Type | Requirements | Default Value | Description                                       |
|-----------------|-----------|--------------|---------------|---------------------------------------------------|
| header_schema    | object    | Optional     | -             | Configuration for JSON Schema to validate request headers |
| body_schema      | object    | Optional     | -             | Configuration for JSON Schema to validate request body   |
| rejected_code    | number    | Optional     | 403           | HTTP status code returned when the request is rejected   |
| rejected_msg     | string    | Optional     | -             | HTTP response body returned when the request is rejected  |
| enable_swagger   | bool      | Optional     | false         | Configuration to enable Swagger documentation validation   |
| enable_oas3      | bool      | Optional     | false         | Configuration to enable OAS3 documentation validation      |

**Validation rules for header and body are the same, below is an example using body.**

## Configuration Examples
### Enumeration (Enum) Validation
```yaml
body_schema:
  type: object
  required:
    - enum_payload
  properties:
    enum_payload:
      type: string
      enum:
        - "enum_string_1"
        - "enum_string_2"
      default: "enum_string_1"
```

### Boolean Validation
```yaml
body_schema:
  type: object
  required:
    - boolean_payload
  properties:
    boolean_payload:
      type: boolean
      default: true
```

### Number Range (Number or Integer) Validation
```yaml
body_schema:
  type: object
  required:
    - integer_payload
  properties:
    integer_payload:
      type: integer
      minimum: 1
      maximum: 10
```

### String Length Validation
```yaml
body_schema:
  type: object
  required:
    - string_payload
  properties:
    string_payload:
      type: string
      minLength: 1
      maxLength: 10
```

### Regular Expression (Regex) Validation
```yaml
body_schema:
  type: object
  required:
    - regex_payload
  properties:
    regex_payload:
      type: string
      minLength: 1
      maxLength: 10
      pattern: "^[a-zA-Z0-9_]+$"
```

### Array Validation
```yaml
body_schema:
  type: object
  required:
    - array_payload
  properties:
    array_payload:
      type: array
      minItems: 1
      items:
        type: integer
        minimum: 1
        maximum: 10
      uniqueItems: true
      default: [1, 2, 3]
```

### Combined Validation
```yaml
body_schema:
  type: object
  required:
    - boolean_payload
    - array_payload
    - regex_payload
  properties:
    boolean_payload:
      type: boolean
    array_payload:
      type: array
      minItems: 1
      items:
          type: integer
          minimum: 1
          maximum: 10
      uniqueItems: true
      default: [1, 2, 3]
    regex_payload:
      type: string
      minLength: 1
      maxLength: 10
      pattern: "^[a-zA-Z0-9_]+$"
```

### Custom Rejection Message
```yaml
body_schema:
  type: object
  required:
    - boolean_payload
  properties:
    boolean_payload:
      type: boolean
rejected_code: 403
rejected_msg: "Request rejected"
```
