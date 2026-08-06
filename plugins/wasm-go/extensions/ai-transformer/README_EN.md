---
title: AI Request-Response Transformation
keywords: [higress, AI transformer]
description: AI Request-Response Transformation plugin configuration reference
---
## Function Description
The AI Request-Response Transformation plugin modifies the header and body of requests/responses using LLM.

## Execution Attributes
Plugin execution phase: `Authentication Phase`  
Plugin execution priority: `410`  

## Configuration Description
| Name | Type | Requirement | Default | Description |
| :- | :-  | :-  | :- | :- |
| request.enable | bool | required | - | Whether to enable transformation in the request phase |
| request.prompt | string | required | - | Prompt used for transformation in the request phase |
| response.enable | bool | required | - | Whether to enable transformation in the response phase |
| response.prompt | string | required | - | Prompt used for transformation in the response phase |
| provider.serviceName | string | required | - | DNS type service name, currently only supports Qwen |
| provider.domain | string | required | - | LLM service domain |
| provider.apiKey | string | required | - | Alibaba Cloud Dashscope service API Key |

## Configuration Example
```yaml
request:
    enable: false
    prompt: "If the request path starts with /httpbin, please remove the /httpbin prefix and do not change anything else."
response:
    enable: true
    prompt: "Please modify the following HTTP response information with the requirements: 1. change content-type to application/json; 2. convert body from xml to json; 3. remove content-length."
provider:
    serviceName: qwen
    domain: dashscope.aliyuncs.com
    apiKey: xxxxxxxxxxxxx
```

Accessing the original httpbin's /xml interface yields:
```
<?xml version='1.0' encoding='us-ascii'?>
<!--  A SAMPLE set of slides  -->
<slideshow
    title="Sample Slide Show"
    date="Date of publication"
    author="Yours Truly"
    >
    <!-- TITLE SLIDE -->
    <slide type="all">
      <title>Wake up to WonderWidgets!</title>
    </slide>
    <!-- OVERVIEW -->
    <slide type="all">
        <title>Overview</title>
        <item>Why <em>WonderWidgets</em> are great</item>
        <item/>
        <item>Who <em>buys</em> WonderWidgets</item>
    </slide>
</slideshow>
```

Using the above configuration, accessing the httpbin's /xml interface through the gateway yields:
```
{
  "slideshow": {
    "title": "Sample Slide Show",
    "date": "Date of publication",
    "author": "Yours Truly",
    "slides": [
      {
        "type": "all",
        "title": "Wake up to WonderWidgets!"
      },
      {
        "type": "all",
        "title": "Overview",
        "items": [
          "Why <em>WonderWidgets</em> are great",
          "",
          "Who <em>buys</em> WonderWidgets"
        ]
      }
    ]
  }
}
```
