# Configurations
Route设置:

```yaml
  routes:
    - name: "route_1"
      match:
        prefix: "/headers"
      route:
        cluster: local_server
    - name: "route_2"
      match:
        prefix: "/user-agent"
      route:
        cluster: local_server
    - name: "route_3"
      match:
        prefix: "/"
      route:
        cluster: local_server
```

插件设置：

```yaml
  configuration:
    "@type": "type.googleapis.com/google.protobuf.StringValue"
    value: |
      {
        "secRules": [
          "Include @demo-conf",
          "Include @crs-setup-demo-conf",
          "SecDefaultAction \"phase:3,log,auditlog,pass\"",
          "SecDefaultAction \"phase:4,log,auditlog,pass\"",
          "SecDefaultAction \"phase:5,log,auditlog,pass\"",
          "SecDebugLogLevel 3",
          "Include @owasp_crs/*.conf",
          "SecRule REQUEST_URI \"@streq /admin\" \"id:101,phase:1,t:lowercase,deny\"",
          "SecRule REQUEST_BODY \"@rx maliciouspayload\" \"id:102,phase:2,t:lowercase,deny\"",
          "SecRule RESPONSE_HEADERS::hello \"@streq world\" \"id:103,phase:3,t:lowercase,deny\"",
          "SecRule RESPONSE_BODY \"@contains responsebodycode\" \"id:104,phase:4,t:lowercase,deny\""
        ],
        "_rules_": [
          {
            "_match_route_": [
              "route_1"
            ],
            "secRules": [
              "SecDebugLogLevel 9",
              "SecRuleEngine On",
              "SecAction \"id:102,phase:1,deny\""
            ]
          },
          {
            "_match_route_": [
              "route_2"
            ],
            "secRules": [
              "SecDebugLogLevel 9",
              "SecRuleEngine On",
              "SecAction \"id:102,phase:1,pass\""
            ]
          }
        ]
      }
```

# Test route level WAF Rules
```bash
curl -I localhost:8080/headers # deny
curl -I localhost:8080/user-agent # pass
```

# Phase 1, OnHttpRequestHeaders

Process Request Headers

```bash
curl -I localhost:8080/admin # deny, Process URI

```

# Phase 2, OnHttpRequestBody

Process Request Body

```bash
curl -i -X POST localhost:8080/post -d 'maliciouspayload' # deny
curl -i -X POST localhost:8080/post -d 'hello' # pass
```

# Phase 3, OnHttpResponseHeaders

Process Response Headers


# Phase 4, OnHttpResponseBody

Process Response Body


# Phase 5, OnHttpStreamDone

Process Logging

