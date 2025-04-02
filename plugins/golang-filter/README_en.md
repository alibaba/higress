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

## Configuration Example

```yaml
http_filters:
- name: envoy.filters.http.golang
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config
    library_id: my-go-filter
    library_path: "./my-go-filter.so"
    plugin_name: my-go-filter
    plugin_config:
      "@type": type.googleapis.com/xds.type.v3.TypedStruct
      value:
          your_config_here: value
```

## Quick Build

Use the following command to quickly build the golang filter plugin:

```bash
GO_FILTER_NAME=mcp-server make build
``` 