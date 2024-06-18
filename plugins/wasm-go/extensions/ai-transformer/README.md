# 简介
低代码开发插件，通过LLM对请求/响应的header以及body进行修改。

# 配置说明
| Name | Type | Requirement | Default | Description |
| :- | :-  | :-  | :- | :- |
| request.enable | bool | requried | - | 是否在request阶段开启转换 |
| request.prompt | string | requried | - | request阶段转换使用的prompt |
| response.enable | string | requried | - | 是否在response阶段开启转换 |
| response.prompt | string | requried | - | response阶段转换使用的prompt |
| provider.serviceName | string | requried | - | DNS类型的服务名，目前仅支持通义千问 |
| provider.domain | string | requried | - | LLM服务域名 |
| provider.apiKey | string | requried | - | 阿里云dashscope服务的API Key |

# 配置示例
```yaml
request:
    enable: false
    prompt: "如果请求path是以/httpbin开头的，帮我去掉/httpbin前缀，其他的不要改。"
response: 
    enable: true
    prompt: "帮我修改以下HTTP应答信息，要求：1. content-type修改为application/json；2. body由xml转化为json；3. 移除content-length。"
provider: 
    serviceName: qwen
    domain: dashscope.aliyuncs.com
    apiKey: xxxxxxxxxxxxx
```

访问原始的httbin的/xml接口，结果为：
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

使用以上配置，通过网关访问httpbin的/xml接口，结果为：
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
