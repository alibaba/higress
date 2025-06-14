---
title: AI IMAGE READER
keywords: [ AI GATEWAY, AI IMAGE READER ]
description: AI IMAGE READER Plugin Configuration Reference
---

## Function Description

By integrating with OCR services to implement AI-IMAGE-READER, currently, it supports Alibaba Cloud's qwen-vl-ocr model under Dashscope for OCR services, and the process is shown in the figure below:<img src=".\ai-image-reader-en.png"> 

## Running Attributes

Plugin execution phase：`Default Phase`
Plugin execution priority：`400`

## Configuration Description

| Name                 | Data Type      | Requirement | Default Value | Description                                                  |
| -------------------- | -------------- | ----------- | ------------- | ------------------------------------------------------------ |
| `apiKey`             | string         | Required    | -             | Token for authenticating access to OCR services.             |
| `serviceName`        | string         | Required    | -             | Name of the backend OCR service.                             |
| `servicePort`        | int            | Required    | -             | Port of the backend OCR service.                             |
| `provider`           | string         | Required    | -             | Provider of the backend OCR service (e.g., alibaba cloud dashscope). |
| `model`              | stringRequired | Required    | -             | Model name of the backend OCR service (e.g., qwen-vl-ocr).   |
| `timeoutMillisecond` | int            | Required    | 30000         | API call timeout duration (milliseconds).                    |

## Example

```yaml
"apiKey": "YOUR_API_KEY",
"serviceName": "dashscope.dns",
"servicePort": "443"
"provider": "alibaba cloud dashscope",
"model": "qwen-vl-ocr",
"timeoutMillisecond": 30000
```

Request to follow the OpenAI API protocol specifications:

Pass images via URL:

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

Pass images via Base64:

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

The following is an example of using ai-image-reader for enhancement. The original request was:

```
What is the content of the image?
```

The result returned by the LLM without processing from the ai-image-reader plugin is:

```
Sorry, as a text-based AI assistant, I cannot view image content. You can describe the content of the image, and I will do my best to help you identify it.
```

The result returned by the LLM after processing by the ai-image-reader plugin is:

```
Thank you for sharing the image! Mastering shell scripting is highly beneficial for Linux system administrators as it automates tasks, boosts efficiency, and cuts down manual work. For home Linux users, command-line skills are equally important for quick and efficient operations. This book will teach you to handle system management tasks with shell scripts and operate in the Linux command line. Hope it aids your Linux system management learning! Feel free to ask if you have more questions.
```