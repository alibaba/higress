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
### Enable for Specific Routes
`_match_route_` specifies the route names `route-a` and `route-b` filled during the creation of gateway routes; when these routes are matched, this configuration will be used;

When multiple rules are configured, the effective order of matching follows the arrangement of rules under `_rules_`, with the first matched rule applying the corresponding configuration, and subsequent rules being ignored.

**Example 1: Content-Based Matching**

According to the following configuration, requests to routes `route-a` and `route-a` that meet both the condition of the request header `role` being one of `user`, `viewer`, or `editor` and the query parameter `foo=bar` will have the request header `x-mse-tag: gray` added. Since `defaultTagKey` and `defaultTagVal` are configured, when no conditions are matched, the request header `x-mse-tag: base` will be added to the request.
```yaml
# Use the _rules_ field for fine-grained rule configuration
_rules_:
- _match_route_:
  - route-a
  - route-b
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
_rules_:
- _match_route_:
  - route-a
  - route-b
  # The total weight sums to 100, the unconfigured 40 weight will not add a header
  weightGroups:
    - headerName: x-mse-tag
      headerValue: gray
      weight: 30
    - headerName: x-mse-tag
      headerValue: blue
      weight: 30
```

### Enable for Specific Domains
`_match_domain_` specifies the domains `*.example.com` and `test.com` for matching request domains; when a matching domain is found, this configuration will be used;

According to the following configuration, for requests targeted at `*.example.com` and `test.com`, if they contain the request header `role` and its value begins with `user`, such as `role: user_common`, the request will have the header `x-mse-tag: blue` added.

```yaml
_rules_:
- _match_domain_:
  - "*.example.com"
  - test.com
  conditionGroups:
    - headerName: x-mse-tag
      headerValue: blue
      logic: and
      conditions:
        - conditionType: header
          key: role
          operator: prefix
          value:
            - user
```

### Enable for Gateway Instances
The following configuration does not specify the rules field, thus it will be effective at the gateway instance level.
It is possible to configure `conditionGroups` or `weightGroups` separately based on content or weight matching; if both are configured, the plugin will first match according to the conditions in `conditionGroups`, and if a match is successful, add the corresponding header and skip subsequent logic. If all `conditionGroups` do not match successfully, then `weightGroups` will add headers according to weight configuration.

```yaml
conditionGroups:
  - headerName: x-mse-tag-1
    headerValue: gray
    # logic为or，则conditions中任一条件满足就命中匹配
    logic: or
    conditions:
      - conditionType: header
        key: x-user-type
        operator: prefix
        value:
          - test
      - conditionType: cookie
        key: foo
        operator: equal
        value:
          - bar
  - headerName: x-mse-tag-2
    headerValue: blue
    # logic为and，需要conditions中所有条件满足才命中匹配
    logic: and
    conditions:
      - conditionType: header
        key: x-type
        operator: in
        value:
          - type1
          - type2
          - type3
      - conditionType: header
        key: x-mod
        operator: regex
        value:
          - "^[a-zA-Z0-9]{8}$"
  - headerName: x-mse-tag-3
    headerValue: green
    logic: and
    conditions:
      - conditionType: header
        key: user_id
        operator: percentage
        value:
          - 60
# 权重总和为100，下例中未配置的40权重将不添加header
weightGroups:
  - headerName: x-higress-canary
    headerValue: gray
    weight: 30
  - headerName: x-higress-canary
    headerValue: base
    weight: 30
```