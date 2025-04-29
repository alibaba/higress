# Golang HTTP Filter

English | [简体中文](./README.md)

## Introduction

The Golang HTTP Filter allows developers to write custom Envoy Filters using the Go language. This framework supports executing Golang code during both request and response flows, making it easier to extend Envoy. Most importantly, Go plugins developed using this framework can be compiled independently of Envoy, which greatly enhances development and deployment flexibility.

> **注意** Golang Filter require Higress version 2.1.0 or higher to be used.
## Features

- Support for Golang code execution in both request and response flows
- Independent plugin compilation without rebuilding Envoy
- Simple and clean API interface
- Request/response header modification
- Request/response body modification
- Synchronous request support

## Quick Start

Please refer to [Envoy Golang HTTP Filter Example](https://github.com/envoyproxy/examples/tree/main/golang-http) to learn how to develop and run a basic Golang Filter.

## Plugin Registration

When developing a new Golang Filter, you need to register your plugin in the `init()` function of `main.go`. The registration requires a plugin name, Filter factory function, and configuration parser:

```go
func init() {
    envoyHttp.RegisterHttpFilterFactoryAndConfigParser(
        "your-plugin-name",    // Plugin name
        yourFilterFactory,     // Filter factory function
        &yourConfigParser{},   // Configuration parser
    )
}
```

## Configuration Example

Multiple Golang Filter plugins can be compiled into a single `golang-filter.so` file, and the desired plugin can be specified using `plugin_name`. Here's an example configuration:

```yaml
http_filters:
- name: envoy.filters.http.golang
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config
    library_id: your-plugin-name
    library_path: "./golang-filter.so"  # Shared library file containing multiple plugins
    plugin_name: your-plugin-name       # Specify which plugin to use, must match the name registered in init()
    plugin_config:
      "@type": type.googleapis.com/xds.type.v3.TypedStruct
      value:
          your_config_here: value
```

## Quick Build

Use the following command to quickly build the golang filter plugin:

```bash
make build
``` 