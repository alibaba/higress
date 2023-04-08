# Features
The `key-auth` plug-in implements the authentication function based on the API Key, supports parsing the API Key from the URL parameter or request header of the HTTP request, and verifies whether the API Key has permission to access.

# Configuration field

|   Name      | Data Type       |               Parameter requirements                     | Default|     Description                                                                                           |
| ----------- | --------------- | -------------------------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------- |
| `consumers` | array of object | Required                                                 | -      | Configure the caller of the service to authenticate the request.                                          |
| `keys`      | array of string | Required                                                 | -      | The name of the source field of the API Key, which can be a URL parameter or an HTTP request header name. |
| `in_query`  | bool            | At least one of `in_query` and `in_header` must be true. | true   | When configured true, the gateway will try to parse the API Key from the URL parameters.                  |
| `in_header` | bool            | The same as above.                                       | true   | The same as above.                                                                                        |
| `_rules_`   | array of object | Optional                                                 | -      | Configure the access list of a specific route or domain name for authenticating requests.                 |


The configuration fields of each item in `consumers` are described as follows:

| Name         | Data Type | Parameter requirements | Default | Description                                  |
| ------------ | --------- | -----------------------| ------  | -------------------------------------------  |
| `credential` | string    | Required               | -       | Configure the consumer's access credentials. |
| `name`       | string    | Required               | -       | Configure the name of the consumer.          |

The configuration fields of each item in `_rules_` are described as follows:

| Name             | Data Type       | Parameter requirements                                                 | Default| Description                                               |
| ---------------- | --------------- | ---------------------------------------------------------------------  | ------ | -------------------------------------------------- |
| `_match_route_`  | array of string | Optional，Optionally fill in one of `_match_route_`, `_match_domain_`. | -      | Configure the route name to match.                               |
| `_match_domain_` | array of string | Optional，Optionally fill in one of `_match_route_`, `_match_domain_`. | -      | Configure the domain name to match.                                  |
| `allow`          | array of string | Required                                                               | -      | For requests that meet the matching conditions, configure the name of the consumer that is allowed to access. |

**Warning：**
- If the `_rules_` field is not configured, authentication will be enabled for all routes of the current gateway instance by default;
- For a request that passes authentication, an `X-Mse-Consumer` field will be added to the request header to identify the name of the caller.

# Example configuration

## Enabled for specific routes or domains

The following configuration will enable Key Auth authentication and authentication for gateway-specific routes or domain names. Note that the `credential` field can not be repeated.

```yaml
consumers:
- credential: 2bda943c-ba2b-11ec-ba07-00163e1250b5
  name: consumer1
- credential: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
  name: consumer2
keys:
- apikey
in_query: true
# Use the _rules_ field for fine-grained rule configuration
_rules_:
# Rule 1: Match by route name to take effect
- _match_route_:
  - route-a
  - route-b
  allow:
  - consumer1
# Rule 2: Take effect by domain name matching
- _match_domain_:
  - "*.example.com"
  - test.com
  allow:
  - consumer2
```

The `route-a` and `route-b` specified in `_match_route_` in this example are the route names filled in when creating the gateway route. When these two routes are matched, calls whose `name` is `consumer1` will be allowed Access by callers, other callers are not allowed to access;

`*.example.com` and `test.com` specified in `_match_domain_` in this example are used to match the domain name of the request. When the domain name matches, the caller whose `name` is `consumer2` will be allowed to access, and other calls access is not allowed.

### Depending on this configuration, the following requests would allow access：

Assume that the following request will match the route-a route:

**Set the API Key in the url parameter**
```bash
curl  http://xxx.hello.com/test?apikey=2bda943c-ba2b-11ec-ba07-00163e1250b5
```
**Set the API Key in the http request header**
```bash
curl  http://xxx.hello.com/test -H 'x-api-key: 2bda943c-ba2b-11ec-ba07-00163e1250b5'
```

After the authentication is passed, an `X-Mse-Consumer` field will be added to the header of the request. In this example, its value is `consumer1`, which is used to identify the name of the caller.

### The following requests will deny access：

**The request does not provide an API Key, return 401**
```bash
curl  http://xxx.hello.com/test
```
**The API Key provided by the request is not authorized to access, return 401**
```bash
curl  http://xxx.hello.com/test?apikey=926d90ac-ba2e-11ec-ab68-00163e1250b5
```

**The caller matched according to the API Key provided in the request has no access rights, return 403**
```bash
# consumer2 is not in the allow list of route-a
curl  http://xxx.hello.com/test?apikey=c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
```

## Gateway instance level enabled

The following configuration does not specify the `_rules_` field, so Key Auth authentication will be enabled at the gateway instance level.

```yaml
consumers:
- credential: 2bda943c-ba2b-11ec-ba07-00163e1250b5
  name: consumer1
- credential: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
  name: consumer2
keys:
- apikey
in_query: true
```

# Error code

| HTTP status code | Error information                                          | Reason                                        |
| ---------------- | ---------------------------------------------------------  | --------------------------------------------  |
| 401              | No API key found in request.                               | API not provided by request Key.              |
| 401              | Request denied by Key Auth check. Invalid API key.         | Current API Key access is not allowed.        |
| 403              | Request denied by Basic Auth check. Unauthorized consumer. | The requested caller does not have access.    |

