---
title: IP 地理位置
keywords: [higress,geo ip]
description: IP 地理位置插件配置参考
---


## 功能说明

`geo-ip`本插件实现了通过用户ip查询出地理位置信息，然后通过请求属性和新添加的请求头把地理位置信息传递给后续插件。

## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`440`

## 配置字段
| 名称            | 数据类型     | 填写要求    |  默认值          | 描述      |
| --------        | --------    | --------   | --------          | -------- |
|  ip_protocol    |  string     |  否        |   ipv4             |  可选值：1. ipv4：只对ipv4用户请求查找地理位置信息，传递给后续插件。而ipv6用户的请求会跳过该插件，继续由后续插件处理。 2. ipv6：(未来实现后)只对ipv6用户查找地理位置信息，传递给后续插件。而ipv4用户的请求会跳过该插件，继续由后续插件处理。（目前是跳过插件，请求由后续插件处理。）
|  ip_source_type |  string     |  否        |   origin-source    |  可选值：1. 对端socket ip：`origin-source`; 2. 通过header获取：`header`  |
|  ip_header_name |  string     |  否        |   x-forwarded-for  |  当`ip_source_type`为`header`时，指定自定义IP来源头                      |


## 配置示例

```yaml
ip_protocol: ipv4
ip_source_type: header
ip_header_name: X-Real-Ip
```

## 生成geoCidr.txt的说明

在generateCidr目录里包含的ip.merge.txt文件是github上ip2region项目的全世界的ip网段库。 ipRange2Cidr.go 是把ip网段转换成多个cidr的程序。转换出的cidr 和地理位置信息存在 /data/geoCidr.txt文件里。geo-ip插件会在Higress启动读配置阶段读取geoCidr.txt文件并且解析到radixtree数据结构的内存里，以便以后查询用户ip对应的地理位置信息。转换程序运行命令如下：

```bash
go run generateCidr/ipRange2Cidr.go
```

## property 的使用方式
在geo-ip插件里调用proxywasm.SetProperty() 分别把country、city、province、isp设置进请求属性里，以便后续插件可以调用proxywasm.GetProperty()获取该请求的用户ip对应的地理信息。
