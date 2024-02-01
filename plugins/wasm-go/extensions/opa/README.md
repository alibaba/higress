# 功能说明

该插件实现了 `OPA` 策略控制

# 该教程使用k8s，[k8s配置文件](../../../../test/e2e/conformance/tests/go-wasm-opa.yaml)

支持client `k8s,nacos,ip,route` 策略去访问

## 配置字段

| 字段            | 数据类型   | 填写要求 | 默认值 | 描述                                   |
|---------------|--------|------|-----|--------------------------------------|
| policy        | string | 必填   | -   | opa 策略                               |
| timeout       | string | 必填   | -   | 访问超时时间设置                             |
| serviceSource | string | 必填   | -   | k8s,nacos,ip,route                   |
| host          | string | 非必填  | -   | 服务主机（serviceSource为`ip`必填）           |
| serviceName   | string | 非必填  | -   | 服务名称（serviceSource为`k8s,nacos,ip`必填） |
| servicePort   | string | 非必填  | -   | 服务端口（serviceSource为`k8s,nacos,ip`必填） |
| namespace     | string | 非必填  | -   | 服务端口（serviceSource为`k8s,nacos`必填）    |

这是一个用于OPA认证配置的表格，确保在提供所有必要的信息时遵循上述指导。

## 配置示例

```yaml
serviceSource: k8s
serviceName: opa
servicePort: 8181
namespace: higress-backend
policy: example1
timeout: 5s
```

# 在宿主机上执行OPA的流程

## 启动opa服务

```shell
docker run -d --name opa -p 8181:8181 openpolicyagent/opa:0.35.0 run -s
```

## 创建opa策略

```shell
curl -X PUT '127.0.0.1:8181/v1/policies/example1' \
  -H 'Content-Type: text/plain' \
  -d 'package example1

import input.request

default allow = false

allow {
    # HTTP method must GET
    request.method == "GET"
}'
```

## 查询策略

```shell
curl -X POST '127.0.0.1:8181/v1/data/example1/allow' \
  -H 'Content-Type: application/json' \
  -d '{"input":{"request":{"method":"GET"}}}'
```

# 测试插件

## 打包 WASM 插件

> 在 `wasm-go` 目录下把Dockerfile文件改成`PLUGIN_NAME=opa`，然后执行以下命令

```shell
docker build -t build-wasm-opa --build-arg GOPROXY=https://goproxy.cn,direct --platform=linux/amd64 .
```

## 拷贝插件

> 在当前的目录执行以下命令，将插件拷贝当前的目录

```shell
docker cp wasm-opa:/plugin.wasm .
```

## 运行插件

> 运行前修改envoy.yaml 这两个字段 `OPA_SERVER` `OPA_PORT` 替换宿主机上的IP和端口

```shell
docker compose up
```

## 使用curl测试插件

```shell
curl http://127.0.0.1:10000/get -X GET -v
```

```shell
curl http://127.0.0.1:10000/get -X POST -v
```