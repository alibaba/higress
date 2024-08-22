# 功能说明

`geo-ip`本插件实现了通过用户ip查询出地理位置信息，然后通过请求属性和新添加的请求头把地理位置信息传递给后续插件。

# 配置字段
| 名称            | 数据类型     | 填写要求   |  默认值  | 描述 |
| --------        | --------    | -------- | -------- | -------- |
|  ipProtocol     |  string     |  必填     |   ipv4    |  ip协议版本   |

# 配置示例

```yaml
ipProtocol: ipv4
```

# 生成geoCidr.txt的说明

在generateCidr目录里包含的ip.merge.txt文件是github上ip2region项目的全世界的ip网段库。 ipRange2Cidr.go 是把ip网段转换成多个cidr的程序。转换出的cidr 和地理位置信息存在 /data/geoCidr.txt文件里。geo-ip插件会在Higress启动读配置阶段读取geoCidr.txt文件并且解析到radixtree数据结构的内存里，以便以后查询用户ip对应的地理位置信息。转换程序运行命令如下：

```basg
go run generateCidr/ipRange2Cidr.go
```

# property 的使用方式
在geo-ip插件里调用proxywasm.SetProperty() 分别把country、city、province、isp设置进请求属性里，以便后续插件可以调用proxywasm.GetProperty()获取该请求的用户ip对应的地理信息。

# ip网段转换成cidr列表的单元测试
在 generateCidr 目录里的  ipRange2Cidr_test.go  是ip网段转换成cidr 列表的单元测试程序。在 generateCidr 目录里运行命令 go test 。通过的情况显示如下：

``bash
PASS
ok      higress/plugins/wasm-go/extensions/geo-ip/generateCidr  0.018s
```
