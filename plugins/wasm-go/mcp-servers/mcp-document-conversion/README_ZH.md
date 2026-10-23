# 文档转换

API认证需要的APP Code请在阿里云API市场申请: https://market.aliyun.com/apimarket/detail/cmapi00067671

## 什么是云市场API MCP服务

阿里云云市场是生态伙伴的交易服务平台，我们致力于为合作伙伴提供覆盖上云、商业化和售卖的全链路服务，帮助客户高效获取、部署和管理优质生态产品。云市场的API服务涵盖以下几个类目：应用开发、身份验证与金融、车辆交通与物流、企业服务、短信与运营商、AI应用与OCR、生活服务。
云市场API依托Higress提供MCP服务，您只需在云市场完成订阅并获取AppCode，通过Higress MCP Server进行配置，即可无缝集成云市场API服务。

## 如何在使用云市场API MCP服务

1. 进入API详情页，订阅该API。您可以优先使用免费试用。
2. 前往云市场用户控制台，使用阿里云账号登陆后查看已订阅API服务的AppCode，并配置到Higress MCP Server的配置中。注意：在阿里云市场订阅API服务后，您将获得AppCode。对于您订阅的所有API服务，此AppCode是相同的，您只需使用这一个AppCode即可访问所有已订阅的API服务。
3. 云市场用户控制台会实时展示已订阅的预付费API服务的可用额度，如您免费试用额度已用完，您可以选择重新订阅。

# MCP服务器配置文档

## 功能简介
该MCP服务器主要用于提供文件格式转换服务，支持将PDF文件转换为Word、PPT或Excel格式，以及将Word、Excel、PPT和txt等常见办公文档格式转换成PDF。此外，还提供了查询文件转换结果的功能，以便用户能够跟踪其请求的状态。通过这些工具，用户可以轻松地在不同文档类型之间进行转换，并且可以根据需要添加水印以保护文档内容。

## 工具简介

### PDF转文档
此工具允许用户将PDF文件转换成多种Microsoft Office文档格式，包括但不限于Word (.docx, .doc), PowerPoint (.pptx, .ppt) 和 Excel (.xlsx, .xls) 文件。它非常适合那些需要从固定布局的PDF中提取信息并编辑的情况。
- **callBackUrl**: 用于接收转换完成后通知的回调地址。
- **fileUrl**: 指向待转换PDF文件的URL链接；对文件大小及页数有一定限制。
- **type**: 指定目标输出文件格式。

### 文档转PDF
利用此功能，用户可以从Word、Excel、PPT甚至纯文本文件创建PDF版本。这对于希望确保跨平台兼容性或增加安全性的场景非常有用。除了基本的转换能力外，还支持添加自定义文字或图片水印到生成的PDF中。
- **callBackUrl**: 接收转换状态更新的通知地址。
- **fileUrl**: 需要被转换成PDF的源文件链接；不同类型文件有不同的最大尺寸限制。
- **watermarkColor**, **watermarkFontName**, **watermarkFontSize**, **watermarkImage**, **watermarkLocation**, **watermarkRotation**, **watermarkText**, **watermarkTransparency**: 这些参数共同定义了如何在最终PDF文档中显示水印效果。

### 文档转换结果查询
当一个转换任务提交后，可能需要一段时间才能完成。此API允许用户检查特定转换请求的状态，从而了解转换是否成功以及获取任何已生成文件的下载链接。
- **convertTaskId**: 之前发起的转换请求所返回的任务标识符，用于追踪处理进度。

以上工具均通过POST请求调用，并要求设置适当的内容类型头部(Content-Type: application/x-www-form-urlencoded)及授权信息(Authorization: APPCODE [appCode])来访问阿里云市场上的相关服务端点。每个响应都包含有关操作成功与否的信息，以及如果适用的话，还会提供转换后文件的直接链接。
