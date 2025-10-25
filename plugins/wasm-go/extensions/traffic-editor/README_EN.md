---
title: Request/Response Editor
keywords: [higress,request,response,edit]
description: Usage guide for the request/response editor plugin
---

## Features

The `traffic-editor` plugin allows you to modify request/response headers. Supported operations include delete, rename, update, add, append, map, and deduplicate.

## Runtime Properties

Plugin execution phase: `UNSPECIFIED`
Plugin execution priority: `100`

## Configuration Fields

| Field Name         | Type                                      | Required | Description                       |
|--------------------|-------------------------------------------|----------|-----------------------------------|
| defaultConfig      | object (CommandSet)                       | No       | Default command set, executed unconditionally |
| conditionalConfigs | array of object (ConditionalCommandSet)   | No       | Conditional command sets, executed based on conditions |

### CommandSet Structure

| Field Name     | Type                        | Required | Default | Description                |
|----------------|----------------------------|----------|---------|----------------------------|
| disableReroute | bool                       | No       | false   | Whether to disable automatic route selection |
| commands       | array of object (Command)  | Yes      | -       | List of edit commands      |

### ConditionalCommandSet Structure

| Field Name  | Type                        | Required | Description                       |
|-------------|----------------------------|----------|-----------------------------------|
| conditions  | array                      | Yes      | List of conditions, see below     |
| commands    | array of object (Command)  | Yes      | List of commands, same as CommandSet |

#### Command Structure

| Field Name | Type   | Required | Description                  |
|------------|--------|----------|------------------------------|
| type       | string | Yes      | Command type, other fields depend on type |

##### set Command

Sets a field to a specified value. `type` field value is `set`.

Other fields:

| Field Name | Type           | Required | Description      |
|------------|---------------|----------|------------------|
| target     | object (Ref)   | Yes      | Target field info|
| value      | string         | Yes      | Value to set     |

##### concat Command

Concatenates multiple values and assigns to the target field. `type` field value is `concat`.

Other fields:

| Field Name | Type                  | Required | Description                                  |
|------------|-----------------------|----------|----------------------------------------------|
| target     | object (Ref)          | Yes      | Target field info                            |
| values     | array of (string/Ref) | Yes      | Values to concatenate, can be string or Ref  |

##### copy Command

Copies the value from the source field to the target field. `type` field value is `copy`.

Other fields:

| Field Name | Type           | Required | Description      |
|------------|---------------|----------|------------------|
| source     | object (Ref)   | Yes      | Source field info|
| target     | object (Ref)   | Yes      | Target field info|

##### delete Command

Deletes the specified field. `type` field value is `delete`.

Other fields:

| Field Name | Type           | Required | Description      |
|------------|---------------|----------|------------------|
| target     | object (Ref)   | Yes      | Field to delete  |

##### rename Command

Renames a field. `type` field value is `rename`.

Other fields:

| Field Name | Type           | Required | Description      |
|------------|---------------|----------|------------------|
| source     | object (Ref)   | Yes      | Original field info|
| target     | object (Ref)   | Yes      | New field info   |

#### Condition Structure

| Field Name | Type   | Required | Description                  |
|------------|--------|----------|------------------------------|
| type       | string | Yes      | Condition type, other fields depend on type |

##### equals Condition

Checks if a field value equals the specified value. `type` field value is `equals`.

| Field Name | Type           | Required | Description      |
|------------|---------------|----------|------------------|
| value1     | object (Ref)   | Yes      | Field to compare |
| value2     | string         | Yes      | Target value     |

##### prefix Condition

Checks if a field value starts with the specified prefix. `type` field value is `prefix`.

| Field Name | Type           | Required | Description      |
|------------|---------------|----------|------------------|
| value      | object (Ref)   | Yes      | Field to compare |
| prefix     | string         | Yes      | Prefix string    |

##### suffix Condition

Checks if a field value ends with the specified suffix. `type` field value is `suffix`.

| Field Name | Type           | Required | Description      |
|------------|---------------|----------|------------------|
| value      | object (Ref)   | Yes      | Field to compare |
| suffix     | string         | Yes      | Suffix string    |

##### contains Condition

Checks if a field value contains the specified substring. `type` field value is `contains`.

| Field Name | Type           | Required | Description      |
|------------|---------------|----------|------------------|
| value      | object (Ref)   | Yes      | Field to compare |
| substr     | string         | Yes      | Substring        |

##### regex Condition

Checks if a field value matches the specified regular expression. `type` field value is `regex`.

| Field Name | Type           | Required | Description      |
|------------|---------------|----------|------------------|
| value      | object (Ref)   | Yes      | Field to compare |
| pattern    | string         | Yes      | Regular expression|

#### Ref Structure

Used to identify a field in the request or response.

| Field Name | Type   | Required | Description                                                      |
|------------|--------|----------|------------------------------------------------------------------|
| type       | string | Yes      | Field type: `request_header`, `request_query`, `response_header` |
| name       | string | Yes      | Field name                                                       |

### Example Configuration

```json
{
  "defaultConfig": {
    "disableReroute": false,
    "commands": [
      { "type": "set", "target": { "type": "request_header", "name": "x-user" }, "value": "admin" },
      { "type": "delete", "target": { "type": "request_header", "name": "x-remove" } }
    ]
  },
  "conditionalConfigs": [
    {
      "conditions": [
        { "type": "equals", "value1": { "type": "request_query", "name": "role" }, "value2": "admin" }
      ],
      "commands": [
        { "type": "set", "target": { "type": "response_header", "name": "x-status" }, "value": "is-admin" }
      ]
    },
    {
      "conditions": [
        { "type": "prefix", "value": { "type": "request_header", "name": "x-path" }, "prefix": "/api/" }
      ],
      "commands": [
        { "type": "rename", "source": { "type": "request_header", "name": "x-old" }, "target": { "type": "request_header", "name": "x-new" } }
      ]
    },
    {
      "conditions": [
        { "type": "suffix", "value": { "type": "request_header", "name": "x-path" }, "suffix": ".json" }
      ],
      "commands": [
        { "type": "copy", "source": { "type": "request_query", "name": "id" }, "target": { "type": "response_header", "name": "x-id" } }
      ]
    },
    {
      "conditions": [
        { "type": "contains", "value": { "type": "request_header", "name": "x-info" }, "substr": "test" }
      ],
      "commands": [
        { "type": "concat", "target": { "type": "response_header", "name": "x-token" }, "values": ["prefix-", { "type": "request_query", "name": "token" }] }
      ]
    },
    {
      "conditions": [
        { "type": "regex", "value": { "type": "request_query", "name": "email" }, "pattern": "^.+@example\\.com$" }
      ],
      "commands": [
        { "type": "delete", "target": { "type": "response_header", "name": "x-temp" } }
      ]
    }
  ]
}
```

