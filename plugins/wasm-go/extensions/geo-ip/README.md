# 功能说明
`geo-ip`本插件实现了通过用户ip查询出地理位置信息，然后通过请求属性和新添加的请求头把地理位置信息传递给后续插件。


# 配置字段
| 名称            | 数据类型     | 填写要求   |  默认值  | 描述 |
| --------        | --------    | -------- | -------- | -------- |
|  ipProtocol     |  string     |  必填     |   ipv4    |  ip协议版本   |


# 配置示例
```yaml
ipProtocol: ipv4


# 运行代码把ip网段转换成多个cidr地址，生成geoCidr.txt
在generateCidr目录里包含的ip.merge.txt文件是github上ip2region项目的全世界的ip网段库。main.go 是把ip网段转换成多个cidr的程序。转换出的cidr 和地理位置信息存在 /data/geoCidr.txt文件里。geo-ip插件会在Higress启动读配置阶段读取geoCidr.txt文件并且解析到radixtree数据结构的内存里，以便以后查询用户ip对应的地理位置信息。转换程序运行命令如下：

go run generateCidr/main.go










