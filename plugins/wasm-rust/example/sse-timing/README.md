## Proxy-Wasm plugin example: SSE Timing

Proxy-Wasm plugin that traces Server-Side Event(SSE) duration from request start.

### Building

```sh
$ make
```

### Using in Envoy

This example can be run with [`docker compose`](https://docs.docker.com/compose/install/)
and has a matching Envoy configuration.

```sh
$ docker compose up
```

#### Access granted.

Send HTTP request to `localhost:10000/`:

```sh
$ curl localhost:10000/
```
