# APISIX HMAC Authentication
keywords: [higress, hmac auth, apisix]
description: Configuration Reference for APISIX HMAC Authentication Plugin
---

## Feature Description
The `hmac-auth-apisix` plugin is compatible with Apache APISIX's HMAC authentication mechanism. It generates tamper-proof digital signatures for HTTP requests using the HMAC algorithm, enabling request identity authentication and permission control. This plugin is fully compatible with the configuration and signature algorithm of the Apache APISIX HMAC Authentication Plugin. For signature generation methods, please refer to the [Apache APISIX HMAC Authentication Documentation](https://apisix.apache.org/docs/apisix/plugins/hmac-auth/).


## Operational Attributes
- Plugin Execution Phase: `Authentication Phase`
- Plugin Execution Priority: `330`


## Configuration Fields
**Note:**
- In a single rule, authentication configuration and authorization configuration cannot coexist.
- For requests that pass authentication and authorization, a `X-Mse-Consumer` field will be added to the request header to identify the caller's name.


### Authentication Configuration

| Name                    | Data Type        | Requirements                              | Default Value                                      | Description                                                                                                                                                                                                 |
| ----------------------- | ---------------- | ----------------------------------------- | --------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `global_auth`           | bool             | Optional (**Instance-level configuration only**) | -                                                   | Can only be configured at the instance level. If set to `true`, the authentication mechanism takes effect globally; if set to `false`, authentication only applies to domains and routes with specific configurations. If not configured, it takes effect globally only when there are no domain or route configurations (to maintain compatibility with legacy user habits). |
| `consumers`             | array of object  | Required                                  | -                                                   | Configures service callers for request authentication.                                                                                                                                                     |
| `allowed_algorithms`    | array of string  | Optional                                  | ["hmac-sha1", "hmac-sha256", "hmac-sha512"]        | List of allowed HMAC algorithms. Valid values are combinations of "hmac-sha1", "hmac-sha256", and "hmac-sha512".                                                                                           |
| `clock_skew`            | number           | Optional                                  | 300                                                 | Maximum allowed time difference (in seconds) between the timestamp of the client request and the current time of the Higress server. This helps resolve time synchronization differences between the client and server and prevents replay attacks. The timestamp is calculated based on the time in the `Date` header (must be in GMT format). If set to `0`, this check is skipped. |
| `signed_headers`        | array of string  | Optional                                  | -                                                   | List of HTTP headers that should be included in the HMAC signature of the client request.                                                                                                                  |
| `validate_request_body` | boolean          | Optional                                  | false                                               | If set to `true`, the integrity of the request body is verified to ensure no tampering during transmission. Specifically, the plugin creates a SHA-256 base64-encoded digest and compares it with the `Digest` header. Verification fails if the `Digest` header is missing or the digest does not match. |
| `hide_credentials`      | boolean          | Optional                                  | false                                               | If set to `true`, the authorization request header will not be passed to the upstream service.                                                                                                              |
| `anonymous_consumer`    | string           | Optional                                  | -                                                   | Name of the anonymous consumer. If configured, anonymous users are allowed to bypass identity authentication.                                                                                              |


### Configuration Fields for Each Item in `consumers`

| Name         | Data Type | Requirements | Default Value | Description                                                                 |
|--------------|-----------|--------------|---------------|-----------------------------------------------------------------------------|
| `access_key` | string    | Required     | -             | A unique identifier for the consumer, used to reference configurations such as the secret key. |
| `secret_key` | string    | Required     | -             | Secret key used to generate the HMAC signature.                             |
| `name`       | string    | Optional     | `access_key`  | Name of the consumer.                                                       |


### Authorization Configuration (Non-essential)

| Name    | Data Type        | Requirements                              | Default Value | Description                                                                                                                                 |
|---------|------------------| ----------------------------------------- |---------------|---------------------------------------------------------------------------------------------------------------------------------------------|
| `allow` | array of string  | Optional (**Non-instance-level configuration only**) | -             | Can only be configured in fine-grained rules such as routes or domains. For requests that match the criteria, it configures the consumers allowed to access, enabling fine-grained permission control. |


## Configuration Examples

### Global Authentication Configuration and Route-level Authorization
The following configuration enables HMAC Auth authentication and authorization for specific routes or domains of the gateway. **Note: The `access_key` field must be unique.**


#### Example 1: Basic Route and Domain Authorization Configuration
**Instance-level Plugin Configuration**:
```yaml
global_auth: false
consumers:
- name: consumer1
  access_key: consumer1-key
  secret_key: 2bda943c-ba2b-11ec-ba07-00163e1250b5
- name: consumer2
  access_key: consumer2-key
  secret_key: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
```

**Route-level Configuration** (Applicable to `route-a` and `route-b`):
```yaml
allow: 
- consumer1  # Only consumer1 is allowed to access
```

**Domain-level Configuration** (Applicable to `*.example.com` and `test.com`):
```yaml
allow:
- consumer2  # Only consumer2 is allowed to access
```

**Configuration Description**:
- Route names (e.g., `route-a`, `route-b`) correspond to the names defined when creating gateway routes. Only `consumer1` is allowed to access when the route matches.
- Domain matching (e.g., `*.example.com`, `test.com`) is used to filter request domains. Only `consumer2` is allowed to access when the domain matches.
- Callers not in the `allow` list will be denied access.


**Request and Response Examples**:

1. **Successful Verification Scenario**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date",signature="G2+60rCCHQCQDZOailnKHLCEy++P1Pa5OEP1bG4QlRo="' \
-H 'Date:Sat, 30 Aug 2025 00:52:39 GMT' \
-H 'Content-Type: application/json' \
-d '{}'
```
- Response: Returns a normal response from the backend service.
- Additional Info: After successful authentication, the request header `X-Mse-Consumer: consumer1` is automatically added and passed to the backend.


2. **Signature Verification Failure Due to Modified Request Method**
```shell
curl -X PUT 'http://localhost:8082/foo' \  # Changed from POST to PUT here
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date",signature="G2+60rCCHQCQDZOailnKHLCEy++P1Pa5OEP1bG4QlRo="' \
-H 'Date:Sat, 30 Aug 2025 00:52:39 GMT' \
-H 'Content-Type: application/json' \
-d '{}'
```
- Response: `401 Unauthorized`
- Error Message: `{"message":"client request can't be validated: Invalid signature"}`


3. **Caller Not in the Allow List**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer2-key",algorithm="hmac-sha256",headers="@request-target date",signature="5sqSbDX9b91dQsfQra2hpluM7O6/yhS7oLcKPQylyCo="' \
-H 'Date:Sat, 30 Aug 2025 00:54:18 GMT' \
-H 'Content-Type: application/json' \
-d '{}'
```
- Response: `401 Unauthorized`
- Error Message: `{"message":"client request can't be validated: consumer 'consumer2' is not allowed"}`


4. **Expired Timestamp**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization: Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date",signature="gvIUwoYNiK57w6xX2g1Ntpk8lfgD7z+jgom434r5qwg="' \
-H 'Date: Sat, 30 Aug 2025 00:40:21 GMT' \  # Expired timestamp
-H 'Content-Type: application/json' \
-d '{}'
```
- Response: `401 Unauthorized`
- Error Message: `{"message":"client request can't be validated: Clock skew exceeded"}`


#### Example 2: Configuration with Custom Signature Headers and Request Body Verification
**Instance-level Plugin Configuration**:
```yaml
global_auth: false
consumers:
- name: consumer1
  access_key: consumer1-key
  secret_key: 2bda943c-ba2b-11ec-ba07-00163e1250b5
- name: consumer2
  access_key: consumer2-key
  secret_key: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
signed_headers:  # Custom request headers to be included in the signature
- X-Custom-Header-A
- X-Custom-Header-B
validate_request_body: true  # Enable request body signature verification
```


**Request and Response Examples**:

1. **Successful Verification Scenario**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date x-custom-header-a x-custom-header-b",signature="+xCWYCmidq3Sisn08N54NWaau5vSY9qEanWoO9HD4mA="' \
-H 'Date:Sat, 30 Aug 2025 01:04:06 GMT' \
-H 'Digest:SHA-256=RBNvo1WzZ4oRRq0W9+hknpT7T8If536DEMBg9hyq/4o=' \  # Request body digest
-H 'X-Custom-Header-A:test1' \
-H 'X-Custom-Header-B:test2' \
-H 'Content-Type: application/json' \
-d '{}'
```
- Response: Returns a normal response from the backend service.


2. **Missing Signature Header**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date x-custom-header-a x-custom-header-b",signature="+xCWYCmidq3Sisn08N54NWaau5vSY9qEanWoO9HD4mA="' \
-H 'Date:Sat, 30 Aug 2025 01:04:06 GMT' \
-H 'Digest:SHA-256=RBNvo1WzZ4oRRq0W9+hknpT7T8If536DEMBg9hyq/4o=' \
-H 'X-Custom-Header-B:test2' \  # Missing X-Custom-Header-A
-H 'Content-Type: application/json' \
-d '{}'
```
- Response: `401 Unauthorized`
- Error Message: `{"message":"client request can't be validated: expected header \"X-Custom-Header-A\" missing in signing"}`


3. **Tampered Request Body**
```shell
curl -X POST 'http://localhost:8082/foo' \
-H 'Authorization:Signature keyId="consumer1-key",algorithm="hmac-sha256",headers="@request-target date x-custom-header-a x-custom-header-b",signature="dSbv6pdQOcgkN89TmSxiT8F9nypbPUqAR2E7ELL8K2s="' \
-H 'Date:Sat, 30 Aug 2025 01:10:17 GMT' \
-H 'Digest:SHA-256=RBNvo1WzZ4oRRq0W9+hknpT7T8If536DEMBg9hyq/4o=' \  # Mismatches the actual body
-H 'X-Custom-Header-A:test1' \
-H 'X-Custom-Header-B:test2' \
-H 'Content-Type: application/json' \
-d '{"key":"value"}'  # Tampered request body
```
- Response: `401 Unauthorized`
- Error Message: `{"message":"client request can't be validated: Invalid digest"}`


### Enabling Global Authentication at the Gateway Instance Level
The following configuration enables HMAC Auth authentication at the gateway instance level. **All requests must pass authentication to access**:

```yaml
global_auth: true  # Enable global authentication
consumers:
- name: consumer1
  access_key: consumer1-key
  secret_key: 2bda943c-ba2b-11ec-ba07-00163e1250b5
- name: consumer2
  access_key: consumer2-key
  secret_key: c8c8e9ca-558e-4a2d-bb62-e700dcc40e35
```

**Description**: When `global_auth: true`, all requests accessing the gateway must carry valid authentication information. Unauthenticated requests will be directly rejected.