# Golang HTTP Filter

[English](./README_en.md) | 简体中文

## 简介

Golang HTTP Filter 允许开发者使用 Go 语言编写自定义的 Envoy Filter。该框架支持在请求和响应流程中执行 Golang 代码，使 Envoy 的扩展开发变得更加简单。最重要的是，使用此框架开发的 Go 插件可以独立于 Envoy 进行编译，这大大提高了开发和部署的灵活性。

> **注意** Golang Filter 需要 Higress 2.1.0 或更高版本才能使用。
## 特性

- 支持在HTTP请求和响应流程中执行 Go 代码
- 支持插件独立编译，无需重新编译 Envoy
- 提供简洁的 API 接口
- 支持请求/响应头部修改
- 支持请求/响应体修改
- 支持同步请求

## 快速开始

请参考 [Envoy Golang HTTP Filter 示例](https://github.com/envoyproxy/examples/tree/main/golang-http) 了解如何开发和运行一个基本的 Golang Filter。

## 配置示例

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


## 快速构建

使用以下命令可以快速构建 golang filter 插件:

```bash
GO_FILTER_NAME=mcp-server make build
```
