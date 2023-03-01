<p>
   English | <a href="README.md">中文</a>
</p>

# Description 
`basic-auth` plugin implements the function of authentication based on the HTTP Basic Auth standard.

# Configuration Fields

| Name        | Type            | Requirement | Default Value | Description                                                 |
| ----------- | --------------- | -------- | ------ | ---------------------------------------------------- |
| `consumers` | array of object | Required     | -      | Caller of the service for authentication of requests |
| `_rules_`   | array of object | Optional     | -      | Configure access permission list for specific routes or domains to authenticate requests |

Filed descriptions of `consumers` items:

| Name         | Type   | Requirement | Default Value | Description                           |
| ------------ | ------ | ----------- | ------------- | ------------------------------------- |
| `credential` | string | Required    | -             | Credential for this consumer's access |
| `name`       | string | Required    | -             | Name of this consumer                 |

Configuration field descriptions for each item in `_rules_` are as follows:

| Field Name            | Data Type        | Requirement                                          | Default | Description                                               |
| ---------------- | --------------- | ------------------------------------------------- | ------ | -------------------------------------------------- |
| `_match_route_`  | array of string | One of `_match_route_` or `_match_domain_` | -      | Configure the routes to match for request authorization                               |
| `_match_domain_` | array of string | One of `_match_route_` , `_match_domain_` | -      | Configure the domains to match for request authorization                                   |
| `allow`          | array of string | Required                                              | -      | Configure the consumer names allowed to access requests that match the match condition |

**Note:**

- If the `_rules_` field is not configured, authentication is enabled for all routes of the current gateway instance by default;
- For authenticated requests,  `X-Mse-Consumer` field will be added to the request header to identify the name of the caller.

# Configuration Samples

## Enable Authentication and Authorization for specific routes or domains

The following configuration will enable Basic Auth authentication and authorization for specific routes or domains of the gateway. Note that the username and password in the credential information are separated by a ":", and the `credential` field cannot be repeated.



```yaml
# use the _rules_ field for fine-grained rule configuration.
consumers:
- credential: 'admin:123456'
  name: consumer1
- credential: 'guest:abc'
  name: consumer2
_rules_:
# rule 1: match by the route name.
  - _match_route_:
    - route-a
    - route-b
    allow:
    - consumer1
# rule 2: match by the domain.
  - _match_domain_:
    - "*.example.com"
    - test.com
    allow:
    - consumer2
```
In this sample, `route-a` and `route-b` specified in `_match_route_` are the route names filled in when creating gateway routes. When these two routes are matched, the caller with `name` as `consumer1` is allowed to access, and other callers are not allowed to access.

The `*.example.com` and `test.com` specified in `_match_domain_` are used to match the domain name of the request. When the domain name is matched, the caller with `name` as `consumer2` is allowed to access, and other callers are not allowed to access.


### According to this configuration, the following requests are allowed:

**Requests with specified username and password**

```bash
# Assuming the following request will match with route-a
# Use -u option of curl to specify the credentials
curl -u admin:123456  http://xxx.hello.com/test
# Or specify the Authorization request header directly with the credentials in base64 encoding
curl -H 'Authorization: Basic YWRtaW46MTIzNDU2'  http://xxx.hello.com/test
```

A `X-Mse-Consumer` field will be added to the headers of the request, and its value in this example is `consumer1`, used to identify the name of the caller when passed authentication and authorization.

### The following requests will be denied:

**Requests without providing username and password, returning 401**
```bash
curl  http://xxx.hello.com/test
```
**Requests with incorrect username or password, returning 401**
```bash
curl -u admin:abc  http://xxx.hello.com/test
```
**Requests matched with a caller who has no access permission, returning 403**
```bash
# consumer2 is not in the allow list of route-a
curl -u guest:abc  http://xxx.hello.com/test
```

## Enable basic auth for gateway instance

The following configuration does not specify the `_rules_` field, so Basic Auth authentication will be effective for the whole gateway instance.

```yaml
consumers:
- credential: 'admin:123456'
  name: consumer1
- credential: 'guest:abc'
  name: consumer2
```

# Error Codes 

| HTTP Status Code | Error Info                                                                       | Reason               |
| ----------- | ------------------------------------------------------------------------------ | ---------------------- |
| 401         | Request denied by Basic Auth check. No Basic Authentication information found. | Credentials not provided in the request        |
| 401         | Request denied by Basic Auth check. Invalid username and/or password           | Invalid username and/or password           |
| 403         | Request denied by Basic Auth check. Unauthorized consumer                      | Unauthorized consumer |