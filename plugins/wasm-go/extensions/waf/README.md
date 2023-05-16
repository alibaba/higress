# Coraza Proxy WASM

Web Application Firewall WASM filter built on top of [Coraza](https://github.com/corazawaf/coraza) and implementing the [proxy-wasm ABI](https://github.com/proxy-wasm/spec). It can be loaded directly from Envoy or also used as an Istio plugin.

## Getting started

`go run mage.go -l` lists all the available commands:

```bash
▶ go run mage.go -l
Targets:
  build*             builds the Coraza wasm plugin.
  check              runs lint and tests.
  coverage           runs tests with coverage and race detector enabled.
  doc                runs godoc, access at http://localhost:6060
  e2e                runs e2e tests with a built plugin against the example deployment.
  format             formats code in this repository.
  ftw                runs ftw tests with a built plugin and Envoy.
  lint               verifies code quality.
  runExample         spins up the test environment, access at http://localhost:8080.
  teardownExample    tears down the test environment.
  test               runs all unit tests.
  updateLibs         updates the C++ filter dependencies.

* default target
```

### Building the filter

```bash
go run mage.go build
```

You will find the WASM plugin under `./build/main.wasm`.

For performance purposes, some libs are built from non-Go implementations. The compiled polyglot wasm libs are already checked in under [./lib/](./lib/). It is possible to rely on the Dockerfiles under [./buildtools/](./buildtools/) if you wish to rebuild them from scratch
using `go run mage.go updateLibs`.

### Running the filter in an Envoy process

In order to run the coraza-proxy-wasm we need to spin up an envoy configuration including this as the filter config

```yaml
    ...

    filter_chains:
    - filters:
        - name: envoy.filters.network.http_connection_manager
            typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
            stat_prefix: ingress_http
            codec_type: auto
            route_config:
                ...
            http_filters:
                - name: envoy.filters.http.wasm
                typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
                    config:
                    name: "coraza-filter"
                    root_id: ""
                    configuration:
                        "@type": "type.googleapis.com/google.protobuf.StringValue"
                        value: |
                        {
                            "rules": [
                                "SecDebugLogLevel 9",
                                "SecRuleEngine On",
                                "SecRule REQUEST_URI \"@streq /admin\" \"id:101,phase:1,t:lowercase,deny\""
                            ]
                        }
                    vm_config:
                        runtime: "envoy.wasm.runtime.v8"
                        vm_id: "coraza-filter_vm_id"
                        code:
                        local:
                            filename: "build/main.wasm"
```

### Using CRS

[Core Rule Set](https://github.com/coreruleset/coreruleset) comes embedded in the extension, in order to use it in the config, you just need to include it directly in the rules:

Loading entire coreruleset:

```yaml
configuration:
    "@type": "type.googleapis.com/google.protobuf.StringValue"
    value: |
    {
        "rules": [ "SecDebugLogLevel 9", "SecRuleEngine On", "Include @owasp_crs/*.conf" ]
    }
```

Loading some pieces:

```yaml
configuration:
    "@type": "type.googleapis.com/google.protobuf.StringValue"
    value: |
    {
        "rules": [ "SecDebugLogLevel 9", "SecRuleEngine On", "Include @owasp_crs/REQUEST-901-INITIALIZATION.conf" ]
    }
```

#### Recommendations using CRS with proxy-wasm

- In order to mitigate as much as possible malicious requests (or connections open) sent upstream, it is recommended to keep the [CRS Early Blocking](https://coreruleset.org/20220302/the-case-for-early-blocking/) feature enabled (SecAction [`900120`](./wasmplugin/rules/crs-setup.conf.example)).

### Running go-ftw (CRS Regression tests)

The following command runs the [go-ftw](https://github.com/coreruleset/go-ftw) test suite against the filter with the CRS fully loaded.

```bash
go run mage.go ftw
```

Take a look at its config file [ftw.yml](./ftw/ftw.yml) for details about tests currently excluded.

One can also run a single test by executing:

```bash
FTW_INCLUDE=920410 go run mage.go ftw
```

## Example: Spinning up the coraza-wasm-filter for manual tests

Once the filter is built, via the commands `mage runExample` and `mage teardownExample` you can spin up and tear down the test environment. Envoy with the coraza-wasm filter will be reachable at `localhost:8080`. The filter is configured with the CRS loaded working in Anomaly Scoring mode. For details and locally tweaking the configuration refer to [@demo-conf](./wasmplugin/rules/coraza-demo.conf) and [@crs-setup-demo-conf](./wasmplugin/rules/crs-setup-demo.conf).
In order to monitor envoy logs while performing requests you can run:

- Envoy logs: `docker-compose -f ./example/docker-compose.yml logs -f envoy-logs`.
- Critical wasm (audit) logs: `docker-compose -f ./example/docker-compose.yml logs -f wasm-logs`

### Manual requests

Run `./e2e/e2e-example.sh` in order to run the following requests against the just set up test environment, otherwise manually execute and tweak them to grasp the behaviour of the filter:

```bash
# True positive requests:
# Custom rule phase 1
curl -I 'http://localhost:8080/admin'
# Custom rule phase 2
curl -i -X POST 'http://localhost:8080/anything' --data "maliciouspayload"
# Custom rule phase 3
curl -I 'http://localhost:8080/status/406'
# Custom rule phase 4
curl -i -X POST 'http://localhost:8080/anything' --data "responsebodycode"
# XSS phase 1
curl -I 'http://localhost:8080/anything?arg=<script>alert(0)</script>'
# SQLI phase 2 (reading the body request)
curl -i -X POST 'http://localhost:8080/anything' --data "1%27%20ORDER%20BY%203--%2B"
# Triggers a CRS scanner detection rule (913100)
curl -I --user-agent "Grabber/0.1 (X11; U; Linux i686; en-US; rv:1.7)" -H "Host: localhost" -H "Accept: text/xml,application/xml,application/xhtml+xml,text/html;q=0.9,text/plain;q=0.8,image/png,*/*;q=0.5" localhost:8080

# True negative requests:
# A GET request with a harmless argument
curl -I 'http://localhost:8080/anything?arg=arg_1'
# A payload (reading the body request)
curl -i -X POST 'http://localhost:8080/anything' --data "This is a payload"
# An harmless response body
curl -i -X POST 'http://localhost:8080/anything' --data "Hello world"
# An usual user-agent
curl -I --user-agent "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36" localhost:8080
```

### WAF Metrics

Metrics are exposed in the prometheus format under `localhost:8082` (admin cluster in the envoy config).

```bash
curl -s localhost:8082/stats/prometheus | grep waf_filter
```

and we get the metrics with the corresponding tags:

```bash
# TYPE waf_filter_tx_interruptions counter
waf_filter_tx_interruptions{phase="http_request_body",rule_id="949110"} 1
waf_filter_tx_interruptions{phase="http_response_headers",rule_id="949110"} 3
# TYPE waf_filter_tx_total counter
waf_filter_tx_total{} 7
```
