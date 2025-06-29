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

## 插件注册

在开发新的 Golang Filter 时，需要在`main.go` 的 `init()` 函数中注册你的插件。注册时需要提供插件名称、Filter 工厂函数和配置解析器：

```go
func init() {
    envoyHttp.RegisterHttpFilterFactoryAndConfigParser(
        "your-plugin-name",    // 插件名称
        yourFilterFactory,     // Filter 工厂函数
        &yourConfigParser{},   // 配置解析器
    )
}
```

## 配置示例

多个 Golang Filter 插件可以共同编译到一个 `golang-filter.so` 文件中，通过 `plugin_name` 来指定要使用的插件。配置示例如下：

```yaml
http_filters:
- name: envoy.filters.http.golang
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config
    library_id: your-plugin-name
    library_path: "./golang-filter.so"  # 包含多个插件的共享库文件
    plugin_name: your-plugin-name       # 指定要使用的插件名称，需要与 init() 函数中注册的插件名称保持一致
    plugin_config:
      "@type": type.googleapis.com/xds.type.v3.TypedStruct
      value:
          your_config_here: value
```

## 快速构建

使用以下命令可以快速构建 golang filter 插件：

```bash
make build
```

如果是 arm64 架构，请设置 `GOARCH=arm64`：

```bash
make build GOARCH=arm64
```
