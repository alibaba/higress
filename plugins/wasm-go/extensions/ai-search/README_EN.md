## Introduction
---
title: AI Search Enhancement  
keywords: [higress, ai search]  
description: Higress enhances the accuracy and timeliness of DeepSeek-R1 and other models by integrating real-time search results from search engines (Google/Bing etc.)

## Feature Description

The `ai-search` plugin enhances the accuracy and timeliness of AI model responses by integrating real-time results from search engines (Google/Bing etc.). The plugin automatically injects search results into the prompt template and can optionally add reference sources in the final response based on configuration.

## Runtime Attributes

Plugin execution phase: `Default phase`  
Plugin execution priority: `440`  

## Configuration Fields

| Name | Data Type | Required | Default Value | Description |
|------|-----------|----------|---------------|-------------|
| needReference | bool | Optional | false | Whether to add reference sources in the response |
| referenceFormat | string | Required when needReference=true | `"**References:**\n%s"` | Reference content format, must contain %s placeholder |
| defaultLang | string | Optional | "zh-CN" | Default search language code (e.g. zh-CN/en-US) |
| promptTemplate | string | Optional | Built-in template | Prompt template, must contain `{search_results}` and `{question}` placeholders |
| searchFrom | array of object | Required | - | Refer to search engine configuration below, at least one engine must be configured |

## Search Engine Common Configuration

| Name | Data Type | Required | Default Value | Description |
|------|-----------|----------|---------------|-------------|
| type | string | Required | - | Engine type (google/bing/arxiv) |
| apiKey | string | Required | - | Search engine API key |
| cx | string | Required for Google | - | Google Custom Search Engine ID, used to specify search scope |
| arxivCategory | string | Optional for Arxiv | - | Paper [category](https://arxiv.org/category_taxonomy) to search (e.g. cs.AI, cs.CL) |
| serviceName | string | Required | - | Backend service name |
| servicePort | number | Required | - | Backend service port |
| count | number | Optional | 10 | Number of results per search |
| timeoutMillisecond | number | Optional | 5000 | API call timeout (milliseconds) |
| optionArgs | map | Optional | - | Engine-specific parameters (key-value format) |

## Configuration Examples

### Basic Configuration (Single Search Engine)

```yaml
needReference: true
searchFrom:
- type: google
  apiKey: "your-google-api-key"
  cx: "search-engine-id"
  serviceName: "google-svc.dns"
  servicePort: 443
  count: 5
  optionArgs:
    fileType: "pdf"

### Arxiv Search Configuration

```yaml
searchFrom:
- type: arxiv
  serviceName: "arxiv-svc.dns"
  servicePort: 443
  arxivCategory: "cs.AI"
  count: 10
```

### Multiple Search Engines Configuration

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
  cx: "github-search-id"  # Search engine ID specifically for GitHub content
  serviceName: "google-svc.dns"
  servicePort: 443
- type: google
  apiKey: "google-key"
  cx: "news-search-id"    # Search engine ID specifically for Google News content
  serviceName: "google-svc.dns"
  servicePort: 443
- type: bing
  apiKey: "bing-key"
  serviceName: "bing-svc.dns"
  servicePort: 80
  optionArgs:
    answerCount: "5"
```

### Custom Reference Format

```yaml
needReference: true
referenceFormat: "### Data Sources\n%s"
searchFrom: 
- type: bing
  apiKey: "your-bing-key"
  serviceName: "search-service.dns"
  servicePort: 8080
```

## Notes

1. Prompt template must contain `{search_results}` and `{question}` placeholders，optionally use `{cur_date}` to insert current date (format: January 2, 2006)
2. Default prompt template includes search result processing guidelines and response specifications
3. Multiple search engines query in parallel, total timeout = maximum timeoutMillisecond value among all engine configurations + processing time
4. Arxiv search doesn't require an API key, but you can specify a paper category (arxivCategory) to narrow the search scope
