## 简介
---
title: AI 搜索增强
keywords: [higress,ai search]
description: higress 支持通过集成搜索引擎（Google/Bing等）的实时结果，增强DeepSeek-R1等模型等回答准确性和时效性
---

## 功能说明

`ai-search`插件通过集成搜索引擎（Google/Bing等）的实时结果，增强AI模型的回答准确性和时效性。插件会自动将搜索结果注入到提示模板中，并根据配置决定是否在最终回答中添加引用来源。

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`440`

## 配置字段

| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
|------|----------|----------|--------|------|
| needReference | bool | 选填 | false | 是否在回答中添加引用来源 |
| referenceFormat | string | 当needReference=true时必填 | `"**References:**\n%s"` | 引用内容格式，必须包含%s占位符 |
| defaultLang | string | 选填 | "zh-CN" | 默认搜索语言代码（如zh-CN/en-US） |
| promptTemplate | string | 选填 | 内置模板 | 提示模板，必须包含`{search_results}`和`{question}`占位符 |
| searchFrom | array of object | 必填 | - | 参考下面搜索引擎配置，至少配置一个引擎 |

## 搜索引擎通用配置

| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
|------|----------|----------|--------|------|
| type | string | 必填 | - | 引擎类型（google/being） |
| apiKey | string | 必填 | - | 搜索引擎API密钥 |
| cx | string | Google引擎必填 | - | Google自定义搜索引擎ID，用于指定搜索范围 |
| serviceName | string | 必填 | - | 后端服务名称 |
| servicePort | number | 必填 | - | 后端服务端口 |
| count | number | 选填 | 10 | 单次搜索返回结果数量 |
| timeoutMillisecond | number | 选填 | 5000 | API调用超时时间（毫秒） |
| optionArgs | map | 选填 | - | 搜索引擎特定参数（key-value格式） |

## 配置示例

### 基础配置（单搜索引擎）

```yaml
needReference: true
searchFrom:
- type: google
  apiKey: "your-google-api-key"
  cx: "search-engine-id"
  serviceName: "google-search-service"
  servicePort: 443
  count: 5
  optionArgs:
    fileType: "pdf"
```

### 多搜索引擎配置

```yaml
defaultLang: "en-US"
promptTemplate: |
  # Search Results:
  {search_results}
  
  # Please answer this question: 
  {question}
searchFrom:
- type: google
  apiKey: "google-key"
  cx: "github-search-id"  # 专门搜索GitHub内容的搜索引擎ID
  serviceName: "google-svc"
  servicePort: 443
- type: google
  apiKey: "google-key"
  cx: "news-search-id"    # 专门搜索Google News内容的搜索引擎ID 
  serviceName: "google-svc"
  servicePort: 443
- type: being
  apiKey: "bing-key"
  serviceName: "bing-svc"
  servicePort: 80
  optionArgs:
    answerCount: "5"
```

### 自定义引用格式

```yaml
needReference: true
referenceFormat: "### 数据来源\n%s"
searchFrom: 
- type: being
  apiKey: "your-bing-key"
  serviceName: "search-service"
  servicePort: 8080
```

## 注意事项

1. 必须包含`{search_results}`和`{question}`占位符
2. 可选使用`{cur_date}`插入当前日期（格式：2006年1月2日）
3. 默认模板包含搜索结果处理指引和回答规范
4. 模板长度建议控制在2000字符以内
5. 多个搜索引擎是并行查询，总超时时间 = 所有搜索引擎配置中最大timeoutMillisecond值 + 处理时间


