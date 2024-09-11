# Traffic-Tag Plugin Documentation

This document provides detailed information about the configuration options for the `traffic-tag` plugin. This configuration is used to manage traffic tags based on rules defined for canary releases and testing purposes.

## Feature Description

The `traffic-tag` plugin allows for tagging request traffic by adding specific request headers based on weight or specific request content. It supports complex logic to determine how to tag traffic based on user-defined criteria.

## Configuration Fields

This section provides a detailed description of the configuration fields.

| Field Name        | Type             | Default | Required | Description                                                       |
|-------------------|------------------|---------|----------|-------------------------------------------------------------------|
| `conditionGroups` | array of objects | -       | No       | Defines groups of content-based tagging conditions, see **Condition Group Configuration**. |
| `weightGroups`    | array of objects | -       | No       | Defines groups of weight-based tagging conditions, see **Weight Group Configuration**. |
| `defaultTagKey`   | string   | -     | No      | The default tag key name used when no conditions are matched. It only takes effect when **defaultTagValue** is also configured.      |
| `defaultTagValue` | string   | -     | No      | The default tag value used when no conditions are matched. It only takes effect when **defaultTagKey** is also configured.      |

### Condition Group Configuration

Each item in `conditionGroups` has the following configuration fields:

| Field Name   | Type   | Default | Required | Description                                                      |
|--------------|--------|---------|----------|------------------------------------------------------------------|
| `headerName` | string | -       | Yes      | The name of the HTTP header to add or modify.                    |
| `headerValue`| string | -       | Yes      | The value of the HTTP header.                                    |
| `logic`      | string | -       | Yes      | The logical relation within the condition group, supports `and`, `or`, all in lowercase. |
| `conditions` | array of objects | - | Yes | Describes specific tagging conditions, detailed structure below. |
---

The configuration fields for each item in `conditions` are as follows:

| Field Name     | Type   | Default | Required | Description                                                         |
|----------------|--------|---------|----------|---------------------------------------------------------------------|
| `conditionType`| string | -       | Yes      | The type of condition, supports `header`, `parameter`, `cookie`.    |
| `key`          | string | -       | Yes      | The keyword for the condition.                                      |
| `operator`     | string | -       | Yes      | The operator, supports `equal`, `not_equal`, `prefix`, `in`, `not_in`, `regex`, `percentage`. |
| `value`        | array of strings | - | Yes | The value for the condition, **only** supports multiple values for `in` and `not_in`. |

> **Note: When `operator` is `regex`, the regex engine used is [RE2](https://github.com/google/re2). For more details, see the [RE2 official documentation](https://github.com/google/re2/wiki/Syntax).**

### Weight Group Configuration

Each item in `weightGroups` has the following configuration fields:

| Field Name   | Type    | Default | Required | Description                                            |
|--------------|---------|---------|----------|--------------------------------------------------------|
| `headerName` | string  | -       | Yes      | The name of the HTTP header to add or modify.          |
| `headerValue`| string  | -       | Yes      | The value of the HTTP header.                          |
| `weight`     | integer | -       | Yes      | The percentage of traffic weight.                      |

### Operator Descriptions

| Operator     | Description                                             |
|--------------|---------------------------------------------------------|
| `equal`      | Exact match, the value must be completely equal.        |
| `not_equal`  | Not equal match, condition is met when values are not equal. |
| `prefix`     | Prefix match, condition is met when the specified value is a prefix of the actual value. |
| `in`         | Inclusion match, actual value needs to be in the specified list. |
| `not_in`     | Exclusion match, condition is met when the actual value is not in the specified list. |
| `regex`      | Regex match, matches according to regex rules.          |
| `percentage` | Percentage match, the principle: `hash(get(key)) % 100 < value` satisfies the condition. |

> **Tip: Differences between `percentage` and `weight`**
>
> - **`percentage` operator**: Used in condition expressions, based on a specified percentage and specified key-value pair to determine whether to perform a certain action. For the same key-value pair, the result of multiple matches is idempotent, i.e., if it hits the condition this time, it will also hit next time.
> - **`weight` field**: Used to define the traffic weight of different treatment paths. In weight-based traffic marking, `weight` determines the proportion of traffic that a path should receive. Unlike `percentage`, which is based on a fixed comparison basis, `weight` involves a random distribution of traffic weight, so the same request may match multiple outcomes in multiple matches.
>
> When using `percentage` for condition matching, each request is evaluated to see if it meets a specific percentage condition; whereas `weight` statically randomly allocates the proportion of overall traffic.

## Configuration Example

**Example 1: Content-Based Matching**

According to the following configuration, requests that meet both the condition of the request header `role` being one of `user`, `viewer`, or `editor` and the query parameter `foo=bar` will have the request header `x-mse-tag: gray` added. Since `defaultTagKey` and `defaultTagVal` are configured, when no conditions are matched, the request header `x-mse-tag: base` will be added to the request.

```yaml
defaultTagKey: x-mse-tag
defaultTagVal: base
conditionGroups:
  - headerName: x-mse-tag
    headerValue: gray
    logic: and
    conditions:
      - conditionType: header
        key: role
        operator: in
        value:
          - user
          - viewer
          - editor
      - conditionType: parameter
        key: foo
        operator: equal
        value:
        - bar
```

**Example 2: Weight-Based Matching**

According to the following configuration, requests have a 30% chance of having the request header `x-mse-tag: gray` added, a 30% chance of having `x-mse-tag: blue` added, and a 40% chance of not having any header added.

```yaml
# The total weight sums to 100, the unconfigured 40 weight will not add a header
weightGroups:
  - headerName: x-mse-tag
    headerValue: gray
    weight: 30
  - headerName: x-mse-tag
    headerValue: blue
    weight: 30
```
