# 商品条码查询

API认证需要的APP Code请在阿里云API市场申请: https://market.aliyun.com/apimarket/detail/cmapi011032

## 什么是云市场API MCP服务

阿里云云市场是生态伙伴的交易服务平台，我们致力于为合作伙伴提供覆盖上云、商业化和售卖的全链路服务，帮助客户高效获取、部署和管理优质生态产品。云市场的API服务涵盖以下几个类目：应用开发、身份验证与金融、车辆交通与物流、企业服务、短信与运营商、AI应用与OCR、生活服务。
云市场API依托Higress提供MCP服务，您只需在云市场完成订阅并获取AppCode，通过Higress MCP Server进行配置，即可无缝集成云市场API服务。

## 如何在使用云市场API MCP服务

1. 进入API详情页，订阅该API。您可以优先使用免费试用。
2. 前往云市场用户控制台，使用阿里云账号登陆后查看已订阅API服务的AppCode，并配置到Higress MCP Server的配置中。注意：在阿里云市场订阅API服务后，您将获得AppCode。对于您订阅的所有API服务，此AppCode是相同的，您只需使用这一个AppCode即可访问所有已订阅的API服务。
3. 云市场用户控制台会实时展示已订阅的预付费API服务的可用额度，如您免费试用额度已用完，您可以选择重新订阅。

# MCP服务器配置文档

## 功能简介

`product-barcode-query` 是一个专门用于查询国内商品条形码信息的服务。它支持通过API调用来获取与指定条形码相关的商品详情，包括但不限于商品名称、品牌、价格等关键信息。这项服务特别适用于需要快速准确地访问大量商品数据的应用场景，如电商平台、库存管理系统或是消费者权益保护平台等。

## 工具简介

### 商品条码查询

- **用途**：该工具允许用户输入中国标准的商品条形码（以69开头），并返回相应的商品信息。
- **使用场景**：非常适合于那些希望根据条形码快速检索到具体商品资料的企业或个人开发者。例如，在线购物网站可以通过此工具在用户扫描商品条形码后立即显示相关产品详情；零售商亦可利用此功能加强其库存管理系统的效率和准确性。

#### 参数说明
- `code`: 国内商品条形码（必须以69开头）。这是发起请求时必需提供的参数之一。
  - 类型: 字符串
  - 必填: 是
  - 位置: 查询字符串

#### 请求模板
- **URL**: `https://barcode14.market.alicloudapi.com/barcode`
- **方法**: GET
- **头部信息**:
  - `Authorization`: APPCODE {{.config.appCode}}
  - `X-Ca-Nonce`: '{{uuidv4}}'

#### 响应结构
响应将以JSON格式返回，并包含以下主要字段：
- `showapi_res_body`: 包含了实际的商品信息。
  - `showapi_res_body.code`: 条形码
  - `showapi_res_body.engName`: 英文名称
  - `showapi_res_body.flag`: 查询结果标志
  - `showapi_res_body.goodsName`: 商品名称
  - `showapi_res_body.goodsType`: 商品分类
  - `showapi_res_body.img`: 图片地址
  - `showapi_res_body.manuName`: 厂商
  - `showapi_res_body.note`: 备注信息
  - `showapi_res_body.price`: 参考价格(单位:元)
  - `showapi_res_body.remark`: 查询结果备注
  - `showapi_res_body.ret_code`: 返回代码
  - `showapi_res_body.spec`: 规格
  - `showapi_res_body.sptmImg`: 条码图片
  - `showapi_res_body.trademark`: 商标/品牌名称
  - `showapi_res_body.ycg`: 原产地
- `showapi_res_code`: 响应状态码
- `showapi_res_error`: 错误信息（如果存在的话）

以上即为`product-barcode-query`服务及其相关工具的基本概述。希望这份文档能够帮助您更有效地使用这些资源。
