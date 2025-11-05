---
title: AI IMAGE READER
keywords: [ AI网关, AI IMAGE READER ]
description: AI IMAGE READER 插件配置参考

---

## 功能说明

通过对接OCR服务实现AI-IMAGE-READER，目前支持阿里云模型服务灵积（dashscope）的qwen-vl-ocr模型提供OCR服务，流程如图所示：

<img src=".\ai-image-reader.png"> 

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`400`


## 配置说明

| 名称          | 数据类型 | 填写要求 | 默认值 | 描述                                   |
| ------------- | -------- | -------- | ------ | -------------------------------------- |
| `apiKey`      | string   | 必填     | -      | 用于在访问OCR服务时进行认证的令牌。    |
| `type`        | string   | 必填     | -      | 后端OCR服务提供商类型（例如dashscope） |
| `serviceHost` | string   | 必填     | -      | 后端OCR服务域名                        |
| `serviceName` | string   | 必填     | -      | 后端OCR服务名                          |
| `servicePort` | int      | 必填     | -      | 后端OCR服务端口                        |
| `model`       | string   | 必填     | -      | 后端OCR服务模型名称（例如qwen-vl-ocr） |
| `timeout`     | int      | 选填     | 10000  | API调用超时时间（毫秒）                |

## 示例

```yaml
"apiKey": "YOUR_API_KEY",
"type": "dashscope",
"model": "qwen-vl-ocr",
"timeout": 10000,
"serviceHost": "dashscope.aliyuncs.com",
"serviceName": "dashscope",
"servicePort": "443"
```

请求遵循openai api协议规范:

URL传递图片：

```
messages=[{
    "role": "user",
    "content": [
        {"type": "text", "text": "What's in this image?"},
        {
            "type": "image_url",
            "image_url": {
                "url": "https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20241108/ctdzex/biaozhun.jpg",
            },
        },
    ],
}],
```

Base64编码传递图片：

```
messages=[
    {
        "role": "user",
        "content": [
            { "type": "text", "text": "what's in this image?" },
            {
                "type": "image_url",
                "image_url": {
                    "url": f"data:image/jpeg;base64,{base64_image}",
                },
            },
        ],
    }
],
```

以下为使用ai-image-reader进行增强的例子，原始请求为：

```
图片内容是什么？
```

未经过ai-image-reader插件处理LLM返回的结果为：

```
对不起，作为一个文本AI助手，我无法查看图片内容。您可以描述一下图片的内容，我可以尽力帮助您识别。
```

经过ai-image-reader插件处理后LLM返回的结果为：

```
非常感谢您分享的图片内容！根据您提供的文字信息，学习编写shell脚本对Linux系统管理员来说是非常有益的。通过自动化系统管理任务，可以提高效率并减少手动操作的时间。对于家用Linux爱好者来说，了解如何在命令行下操作也是很重要的，因为在某些情况下，命令行操作可能更为便捷和高效。在本书中，您将学习如何运用shell脚本处理系统管理任务，以及如何在Linux命令行下进行操作。希望这本书能够帮助您更好地理解和应用Linux系统管理和操作的知识！如果您有任何其他问题或需要进一步帮助，请随时告诉我。
```