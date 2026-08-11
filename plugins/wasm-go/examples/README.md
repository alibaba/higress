# Go Wasm plugin examples

This directory contains reference implementations and development examples.
They are intentionally excluded from official plugin release discovery, which
only scans `plugins/wasm-go/extensions/`.

The following examples are Go counterparts of official C++ plugins:

- `basic-auth`
- `bot-detect`
- `custom-response`
- `jwt-auth`
- `key-auth`
- `model-mapper`
- `model-router`
- `request-block`
- `sni-misdirect`

Build one from `plugins/wasm-go/` with:

```bash
PLUGIN_ROOT=examples PLUGIN_NAME=request-block make build
```

These examples may keep historical `VERSION` and `.buildrc` files for local
compatibility. Their presence does not make them release artifacts.
