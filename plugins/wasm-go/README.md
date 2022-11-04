## Intro

This SDK is used to develop the WASM Plugins of Higress.
## Requirements

(need support Go's type parameters)

Go version: >= 1.18

TinyGo version: >= 0.25.0

## Quick Examples

### wasm plugin config

```yaml
# this config will take effect globally (all incoming requests are affected)
block_urls:
- "test"
_rules_:
# matching by route name takes effect
- _match_route_:
  - route-a
  - route-b
  block_bodys:
  - "hello world"
# matching by domain takes effect
- _match_domain_:
  - "*.example.com"
  - test.com
  block_urls:
  - "swagger.html"
  block_bodys:
  - "hello world"
```

### code

[request-block](example/request-block)


### compile to wasm

```bash
tinygo build -o main.wasm -scheduler=none -target=wasi ./main.go
```


