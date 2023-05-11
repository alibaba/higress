# Description
The `jwt-auth` plugin implements authentication and authorization based on JWT (JSON Web Token), supports parsing JWTs from URL parameters, request headers, and Cookie fields from HTTP requests, and verifies whether the token has permission to access.

The difference between this plugin and the JWT authentication in `Security Capabilities->Authentication and Authorization` is that it provides additional capabilities for identifying the caller's identity, supporting the configuration of different JWT credentials for different callers.

# Detailed Description

## 1. Token-based authentication

### 1.1 Introduction
Many open APIs need to identify the identity of the caller and determine whether the requested resource can be returned to the caller based on this identity. Token is a mechanism used for identity verification. Based on this mechanism, the application does not need to retain the user's authentication information or session information on the server, which can realize stateless and distributed web application authorization and provide convenience for application extension.

### 1.2 Process Description

![](process.png)

The above figure is the business process sequence diagram when the gateway uses JWT for authentication, and the following we will describe the steps marked in the figure in detail in words:

1. The client initiates an authentication request to the API gateway, usually carrying the end user's username and password in the request;

2. The gateway forwards the request directly to the backend service;

3. The backend service reads the authentication information (such as the username and password) in the request for verification. After the verification is passed, it uses the private key to generate a standard token and returns it to the gateway;

4. The gateway returns the response with the token to the client, and the client needs to cache this token locally;

5. The client sends a business request to the API gateway, carrying the token in the request;

6. The gateway uses the public key set by the user to verify the token in the request. After the verification is passed, the request is passed through to the backend service;

7. The backend service processes the business and responds;

8. The gateway returns the business response to the client.

In this entire process, the gateway uses the token authentication mechanism to realize the ability of users to use their own user system to authorize their API. Next, we will introduce the structured token JSON Web Token (JWT) used by the gateway to implement token authentication.

### 1.3 JWT

#### 1.3.1 Introduction

JSON Web Token (JWT) is an open standard RFC7519 based on JSON for executing a type of claim to pass between network applications. JWT can generally be used as an independent identity verification token, which can contain user identification, user roles, and permissions information, making it easier to obtain resources from the resource server, and can also add some other necessary declarative information for other business logic, especially suitable for the login scenario of distributed sites.

#### 1.3.2 Composition of JWT

`eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ`

As shown in the example above, JWT is a string consisting of three parts:

- Header
- Payload
- Signature

**Header**

The header of the JWT carries two pieces of information:

- The type of the token, which is JWT
- The algorithm used for encryption

The gateway supports the following encryption algorithms:

```text
ES256, ES384, ES512,
HS256, HS384, HS512,
RS256, RS384, RS512,
PS256, PS384, PS512,
EdDSA
```

The complete header looks like the following JSON:

```js
{
  'typ': 'JWT',
  'alg': 'HS256'
}
```

The header is then Base64-encoded (this encoding can be symmetrically decoded), forming the first part:

`eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9`

**Payload**

The payload is where the actual information is stored. The details are defined below:

```text
iss: The issuer of the token. This indicates who created the token and is a string value.
sub: The subject identifier. This is the unique identifier for the end user provided by the issuer, and is no longer than 255 ASCII characters, and is case-sensitive within the issuer's scope.
aud: The audience(s) of the token, which is an array of case-sensitive strings.
exp: The expiration time of the token, after which the token will be invalidated, is an integer declaration representing the Unix timestamp in seconds.
iat: The issuance time of the token, is an integer declaration representing the Unix timestamp in seconds.
jti: The unique identifier of the token, and the value is unique for every token created by the issuer. It is usually a cryptographically random value to prevent conflicts. This component adds a random entropy that an attacker cannot obtain to the structured token, making it more difficult for the token to be guessed or replayed.
```

Custom fields for a user feature can also be added, such as the example below adding a "name" field for the user's nickname:

```js
{
  "sub": "1234567890",
  "name": "John Doe"
}
```

The payload is then Base64-encoded, forming the second part of the JWT:

`JTdCJTBBJTIwJTIwJTIyc3ViJTIyJTNBJTIwJTIyMTIzNDU2Nzg5MCUyMiUyQyUwQSUyMCUyMCUyMm5hbWUlMjIlM0ElMjAlMjJKb2huJTIwRG9lJTIyJTBBJTdE`

**Signature**

This part is a string that consists of the Base64-encoded header and Base64-encoded payload concatenated with a period, followed by the encryption of the resulting string using the algorithm specified in the header (where $secret represents the user's private key).

```js
var encodedString = base64UrlEncode(header) + '.' + base64UrlEncode(payload);
var signature = HMACSHA256(encodedString, '$secret');
```

These three parts are then concatenated using periods to form the complete JWT string as shown in the example at the beginning of this section.

#### 1.3.3 Time validity

The gateway will verify the exp field in the token. Once this field has expired, the gateway will consider the token invalid and reject the request directly. The expiration time value must be set.

#### 1.3.4 Several Characteristics of JWT

1. By default, JWT is not encrypted and cannot write secret data into JWT.
2. JWT can not only be used for authentication but also for exchanging information. Using JWT effectively can reduce the number of times servers query the database.
3. The biggest drawback of JWT is that the server cannot invalidate a token during use, or change the token's permission, because the server does not keep the session state. That is, once JWT is issued, it will always be valid before it expires, unless the server deploys additional logic.
4. JWT contains authentication information itself. Once leaked, anyone can obtain all permissions of the token. To reduce theft, the expiration time of JWT should be set relatively short. For some more important permissions, users should be authenticated again when using them.
5. To reduce theft, JWT should not be transmitted in plaintext using the HTTP protocol, and the HTTPS protocol should be used for transmission.

## 2. How to apply the JWT plugin to protect the API of the user system

### 2.1 Generate a pair of JWK (JSON Web Key)

**Method 1: Generate online**

Users can generate the private and public keys used for token generation and verification on this website https://mkjwk.org. The private key is used for the authorization service to issue JWT, and the public key is configured into the JWT plugin for the gateway to verify the signature of the request. Note that the JWKs format configuration used by the gateway requires the public key in the figure below to be placed in the keys structure, such as: `{"keys":[{"kty":"RSA","e":"AQAB",...}]}`

<img src="https://help-static-aliyun-doc.aliyuncs.com/assets/img/zh-CN/2336348951/p135823.png" style="zoom:50%" />

**Method 2: Generate locally**

This article uses a Java example to illustrate, and users of other languages can also find relevant tools to generate key pairs. Create a new Maven project and add the following dependencies:

```xml
<dependency>
     <groupId>org.bitbucket.b_c</groupId>
     <artifactId>jose4j</artifactId>
     <version>0.7.0</version>
</dependency>
```

Use the following code to generate a pair of RSA keys:

```java
RsaJsonWebKey rsaJsonWebKey = RsaJwkGenerator.generateJwk(2048);
final String publicKeyString = rsaJsonWebKey.toJson(JsonWebKey.OutputControlLevel.PUBLIC_ONLY);
final String privateKeyString = rsaJsonWebKey.toJson(JsonWebKey.OutputControlLevel.INCLUDE_PRIVATE);
```

### 2.2 Implement the token issuance authentication service using the private key in JWK

The Keypair JSON string generated online in Section 2.1 (the first of the three boxes) or privateKeyString JSON string generated locally needs to be used as the private key to issue tokens for trusted users to access protected APIs. The specific implementation can refer to the following example. The form of issuing tokens to customers is determined by the user according to the specific business scenario. The function of issuing tokens can be deployed in the production environment and configured as a normal API for visitors to obtain through username and password, or tokens can be generated directly in the local environment and copied to designated users for use.

```java
import java.security.PrivateKey; 
import org.jose4j.json.JsonUtil;
import org.jose4j.jwk.RsaJsonWebKey;
import org.jose4j.jwk.RsaJwkGenerator;
import org.jose4j.jws.AlgorithmIdentifiers;
import org.jose4j.jws.JsonWebSignature;
import org.jose4j.jwt.JwtClaims;
import org.jose4j.jwt.NumericDate;
import org.jose4j.lang.JoseException;
public class GenerateJwtDemo {
    public static void main(String[] args) throws JoseException  {
        String keyId = "uniq_key";
          //Use the Keypair generated in section 2.1 of this article
        String privateKeyJson = "{\n"
            + "  \"kty\": \"RSA\",\n"
            + "  \"d\": "
            +
            "\"O9MJSOgcjjiVMNJ4jmBAh0mRHF_TlaVva70Imghtlgwxl8BLfcf1S8ueN1PD7xV6Cnq8YenSKsfiNOhC6yZ_fjW1syn5raWfj68eR7cjHWjLOvKjwVY33GBPNOvspNhVAFzeqfWneRTBbga53Agb6jjN0SUcZdJgnelzz5JNdOGaLzhacjH6YPJKpbuzCQYPkWtoZHDqWTzCSb4mJ3n0NRTsWy7Pm8LwG_Fd3pACl7JIY38IanPQDLoighFfo-Lriv5z3IdlhwbPnx0tk9sBwQBTRdZ8JkqqYkxUiB06phwr7mAnKEpQJ6HvhZBQ1cCnYZ_nIlrX9-I7qomrlE1UoQ\",\n"
            + "  \"e\": \"AQAB\",\n"
            + "  \"alg\": \"RS256\",\n"
            + "  \"n\": \"vCuB8MgwPZfziMSytEbBoOEwxsG7XI3MaVMoocziP4SjzU4IuWuE_DodbOHQwb_thUru57_Efe"
            +
            "--sfATHEa0Odv5ny3QbByqsvjyeHk6ZE4mSAV9BsHYa6GWAgEZtnDceeeDc0y76utXK2XHhC1Pysi2KG8KAzqDa099Yh7s31AyoueoMnrYTmWfEyDsQL_OAIiwgXakkS5U8QyXmWicCwXntDzkIMh8MjfPskesyli0XQD1AmCXVV3h2Opm1Amx0ggSOOiINUR5YRD6mKo49_cN-nrJWjtwSouqDdxHYP-4c7epuTcdS6kQHiQERBd1ejdpAxV4c0t0FHF7MOy9kw\"\n"
            + "}";
        JwtClaims claims = new JwtClaims();
        claims.setGeneratedJwtId();
        claims.setIssuedAtToNow();
        //Expiration time must be set
        NumericDate date = NumericDate.now();
        date.addSeconds(120*60);
        claims.setExpirationTime(date);
        claims.setNotBeforeMinutesInThePast(1);
        claims.setSubject("YOUR_SUBJECT");
        claims.setAudience("YOUR_AUDIENCE");
        //Add custom parameters, use String type for all values
        claims.setClaim("userId", "1213234");
        claims.setClaim("email", "userEmail@youapp.com");
        JsonWebSignature jws = new JsonWebSignature();
        jws.setAlgorithmHeaderValue(AlgorithmIdentifiers.RSA_USING_SHA256);
        jws.setKeyIdHeaderValue(keyId);
        jws.setPayload(claims.toJson());
        PrivateKey privateKey = new RsaJsonWebKey(JsonUtil.parseJson(privateKeyJson)).getPrivateKey();
     
        jws.setKey(privateKey);
        String jwtResult = jws.getCompactSerialization();
        System.out.println("Generate Json Web token , result is " + jwtResult);
    }
}
```

# Plugin Configuration Guide

## Configuration Fields

| Name        | Data Type  | Required  | Default | Description  |
| ----------- | --------------- | ------------------------------------------- | ------ | ----------------------------------------------------------- |
| `consumers` | array of object | Yes  | -      | Configures callers of the service for authenticating requests |
| `_rules_`   | array of object | Optional   | -      | Configures access control lists for specific routes or domains for authorizing requests        |

The configuration field descriptions for each item in consumers are as follows:

| Name                    | Data Type          | Required | Default | Description|
| ----------------------- | ----------------- | -------- | ------------------------------------------------- | ------------------------ |
| `name`                  | string            | Yes     | -    | Configures the name of this consumer     |
| `jwks`                  | string            | Yes     | -   | Specifies a JSON Web Key Set, as defined in https://www.rfc-editor.org/rfc/rfc7517, consisting of public keys (or symmetric keys) used to verify the signature of JWT  |
| `issuer`                | string            | Yes     | -  | The issuer of the JWT, which should be consistent with the iss field in the payload   |
| `claims_to_headers`     | array of object   | Optional     | -  | Extracts the specified fields from the JWT's payload and sets them to the specified request headers for forwarding to the backend |
| `from_headers`          | array of object   | Optional     | {"name":"Authorization","value_prefix":"Bearer "} | Extracts the JWT from the specified request headers |
| `from_params`           | array of string   | Optional     | access_token    | Extracts the JWT from the specified URL parameters     |
| `from_cookies`          | array of string   | Optional     | -    | Extracts the JWT from the specified cookie(s)     |
| `clock_skew_seconds`    | number            | Optional     | 60     | The amount of clock skew, in seconds, that is allowed when verifying the exp and iat fields of the JWT          |
| `keep_token`            | bool              | Optional     | ture   | Whether to keep the JWT when forwarding it to the backend  |

**Note:**

- The default value is used only if neither `from_headers`, `from_params`, nor `from_cookies` are configured.

The configuration field descriptions for each item in `from_headers` are as follows:

| Name             | Data Type        | Required| Default | Description |
| ---------------- | --------------- | ------- | ------ | --------------------------------------------------------- |
| `name`           | string          | Yes    | -      | Specifies the request header to extract the JWT from |
| `value_prefix`   | string          | Yes    | -      | Removes the specified prefix from the request header's value, leaving the rest as the JWT  |

The configuration field descriptions for each item in `claims_to_headers` are as follows:

| Name             | Data Type        | Required| Default | Description  |
| ---------------- | --------------- | ------- | ------ | --------------------------------------------------------- |
| `claim`          | string          | Yes    | -      | The name of the field in the JWT payload, which must be a string or unsigned integer |
| `header`         | string          | Yes    | -      | Sets the value of the specified request header to the value of the specified field in the payload, for forwarding to the backend  |
| `override`       | bool            | Optional    | true   | If true, overrides an existing header with the same name; if false, appends the header to the existing headers |

The configuration field descriptions for each item in `_rules_` are as follows:

|  Name             | Data Type        | Required| Default | Description  |
| ---------------- | --------------- | ------------------------------------------------- | ------ | -------------------------------------------------- |
| `_match_route_`  | array of string | Optional, select either `_match_route_` or `_match_domain_` | -      | Configures the route names to match|
| `_match_domain_` | array of string | Optional, select either `_match_route_` or `_match_domain_` | -      | Configures the domains to match |
| `allow`          | array of string | Required  | -      | Configures the consumer names allowed to access the matched requests |

**Note:**
- If the `_rules_` field is not configured, authentication and authorization are enabled for all routes of the current gateway instance by default;
- For authenticated and authorized requests, a `X-Mse-Consumer` field is added to the request header to identify the caller's name.

## Configuration Example

### Enable for Specific Routes or Domains

The following configuration enables Jwt Auth authentication and authorization for specific routes or domains of the gateway. If a JWT can match multiple `jwks`, the first matching `consumer` is hit according to the configuration order.

```yaml
consumers:
- name: consumer1
  issuer: abcd
  jwks: |
    {
      "keys": [
        {
          "kty": "oct",
          "kid": "123",
          "k": "hM0k3AbXBPpKOGg__Ql2Obcq7s60myWDpbHXzgKUQdYo7YCRp0gUqkCnbGSvZ2rGEl4YFkKqIqW7mTHdj-bcqXpNr-NOznEyMpVPOIlqG_NWVC3dydBgcsIZIdD-MR2AQceEaxriPA_VmiUCwfwL2Bhs6_i7eolXoY11EapLQtutz0BV6ZxQQ4dYUmct--7PLNb4BWJyQeWu0QfbIthnvhYllyl2dgeLTEJT58wzFz5HeNMNz8ohY5K0XaKAe5cepryqoXLhA-V-O1OjSG8lCNdKS09OY6O0fkyweKEtuDfien5tHHSsHXoAxYEHPFcSRL4bFPLZ0orTt1_4zpyfew",
          "alg": "HS256"
        }
      ]
    }
- name: consumer2
  issuer: abc
  jwks: |
    {
      "keys": [
        {
          "kty": "RSA",
          "e": "AQAB",
          "use": "sig",
          "kid": "123",
          "alg": "RS256",
          "n": "i0B67f1jggT9QJlZ_8QL9QQ56LfurrqDhpuu8BxtVcfxrYmaXaCtqTn7OfCuca7cGHdrJIjq99rz890NmYFZuvhaZ-LMt2iyiSb9LZJAeJmHf7ecguXS_-4x3hvbsrgUDi9tlg7xxbqGYcrco3anmalAFxsbswtu2PAXLtTnUo6aYwZsWA6ksq4FL3-anPNL5oZUgIp3HGyhhLTLdlQcC83jzxbguOim-0OEz-N4fniTYRivK7MlibHKrJfO3xa_6whBS07HW4Ydc37ZN3Rx9Ov3ZyV0idFblU519nUdqp_inXj1eEpynlxH60Ys_aTU2POGZh_25KXGdF_ZC_MSRw"
        }
      ]
    }
# Use the _rules_ field for fine-grained rule configuration
_rules_:
# Rule 1: Effective when matched by route name
- _match_route_:
  - route-a
  - route-b
  allow:
  - consumer1
# Rule 2: Effective when matched by domain name
- _match_domain_:
  - "*.example.com"
  - test.com
  allow:
  - consumer2
```

In this example, the `route-a` and `route-b` specified in `_match_route_` are the names of the routes filled in when creating the gateway route. When these two routes are matched, access will be allowed for the caller with the `name` of `consumer1`, and other callers will not be allowed to access.

The `*.example.com` and `test.com` specified in `_match_domain_` are used to match the domain names of the requests. When a domain name match is found, access will be allowed for the caller with the `name` of `consumer2`, and other callers will not be allowed to access.

#### According to this configuration, the following requests are allowed:

Assuming the following requests will match the route-a route:

**JWT is set in URL parameter**
```bash
curl  'http://xxx.hello.com/test?access_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjEyMyJ9.eyJpc3MiOiJhYmNkIiwic3ViIjoidGVzdCIsImlhdCI6MTY2NTY2MDUyNywiZXhwIjoxODY1NjczODE5fQ.-vBSV0bKeDwQcuS6eeSZN9dLTUnSnZVk8eVCXdooCQ4'
```

**JWT is set in HTTP request header**
```bash
curl  http://xxx.hello.com/test -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjEyMyJ9.eyJpc3MiOiJhYmNkIiwic3ViIjoidGVzdCIsImlhdCI6MTY2NTY2MDUyNywiZXhwIjoxODY1NjczODE5fQ.-vBSV0bKeDwQcuS6eeSZN9dLTUnSnZVk8eVCXdooCQ4'
```

After authentication and authorization, a `X-Mse-Consumer` field will be added in the request header with a value of `consumer1` in this example, to identify the name of the caller.

#### The following requests will be denied:

**Request without JWT provided, returns 401**
```bash
curl  http://xxx.hello.com/test
```

**The consumer matched by the provided JWT in the request does not have access, returns 403**
```bash
# consumer1 is not in the allow list of *.example.com
curl  'http://xxx.example.com/test' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjEyMyJ9.eyJpc3MiOiJhYmNkIiwic3ViIjoidGVzdCIsImlhdCI6MTY2NTY2MDUyNywiZXhwIjoxODY1NjczODE5fQ.-vBSV0bKeDwQcuS6eeSZN9dLTUnSnZVk8eVCXdooCQ4'
```

### Enabling at Gateway Instance Level

The following configuration does not specify the `_rules_` field, so JWT authentication will be enabled at the gateway instance level:

```yaml
consumers:
- name: consumer1
  issuer: abcd
  jwks: |
    {
      "keys": [
        {
          "kty": "oct",
          "kid": "123",
          "k": "hM0k3AbXBPpKOGg__Ql2Obcq7s60myWDpbHXzgKUQdYo7YCRp0gUqkCnbGSvZ2rGEl4YFkKqIqW7mTHdj-bcqXpNr-NOznEyMpVPOIlqG_NWVC3dydBgcsIZIdD-MR2AQceEaxriPA_VmiUCwfwL2Bhs6_i7eolXoY11EapLQtutz0BV6ZxQQ4dYUmct--7PLNb4BWJyQeWu0QfbIthnvhYllyl2dgeLTEJT58wzFz5HeNMNz8ohY5K0XaKAe5cepryqoXLhA-V-O1OjSG8lCNdKS09OY6O0fkyweKEtuDfien5tHHSsHXoAxYEHPFcSRL4bFPLZ0orTt1_4zpyfew",
          "alg": "HS256"
        }
      ]
    }
- name: consumer2
  issuer: abc
  jwks: |
    {
      "keys": [
        {
          "kty": "RSA",
          "e": "AQAB",
          "use": "sig",
          "kid": "123",
          "alg": "RS256",
          "n": "i0B67f1jggT9QJlZ_8QL9QQ56LfurrqDhpuu8BxtVcfxrYmaXaCtqTn7OfCuca7cGHdrJIjq99rz890NmYFZuvhaZ-LMt2iyiSb9LZJAeJmHf7ecguXS_-4x3hvbsrgUDi9tlg7xxbqGYcrco3anmalAFxsbswtu2PAXLtTnUo6aYwZsWA6ksq4FL3-anPNL5oZUgIp3HGyhhLTLdlQcC83jzxbguOim-0OEz-N4fniTYRivK7MlibHKrJfO3xa_6whBS07HW4Ydc37ZN3Rx9Ov3ZyV0idFblU519nUdqp_inXj1eEpynlxH60Ys_aTU2POGZh_25KXGdF_ZC_MSRw"
        }
      ]
    }
```

# Common Error Codes

| HTTP Status Code | Error Message               | Reason Description|
|------------------| ---------------------- | -------------------------------------------------------------------------------- |
| 401              | JWT missing            | The JWT is not provided in the request header. |
| 401              | JWT expired            | The JWT has expired. |
| 401              | JWT verification fails | The JWT payload verification failed, such as the iss mismatch. |
| 403              | Access denied          | Access to the current route is denied. |
