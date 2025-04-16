# Taobao Hot Words

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi022144

# MCP Server Configuration Document

## Function Overview

The `taobao-hot-words` server primarily serves merchants and operators on e-commerce platforms by providing a feature to query the ranking of search keywords within Taobao. It processes and analyzes data based on real-time search frequencies, allowing merchants to grasp the performance of specific keywords in the market, including but not limited to their rankings and distribution characteristics. Additionally, it supports users in specifying any keyword and returning a list of the top 10 most relevant keywords, sorted by relevance from highest to lowest. This function is crucial for optimizing product titles and enhancing search visibility.

## Tool Introduction

### Taobao Hot Words

**Purpose**: As a tool focused on analyzing internal search behavior on the Taobao platform, "Taobao Hot Words" helps merchants or developers understand the popularity and trend changes of specific terms on Taobao. By analyzing this data, users can gain insights into the changing patterns of consumer interests, providing a basis for adjusting product promotion strategies.

**Use Cases**:
- When evaluating whether a new product name or marketing campaign slogan is sufficiently attractive to the target customer base;
- Before formulating an SEO (Search Engine Optimization) plan, to obtain more information about potential hot keywords;
- To regularly monitor the performance of certain key business terms in the market, enabling timely responses;
- To find inspiration for creating new advertising slogans or improving the effectiveness of existing copy.

**Parameter Description**:
- `key` (Required): The specific keyword the user wants to query, of type string.

**Request Template**:
- **URL**: `http://tbhot.market.alicloudapi.com/tbhot10`
- **Method**: GET
- **Headers**:
  - `Authorization`: Use the APP Code for authentication.
  - `X-Ca-Nonce`: A security token used to prevent replay attacks, with a value that is a randomly generated UUID.

**Response Structure**:
- `goodsList`: Contains a list of products related to the queried keyword, stored as an array.
  - `goodsList[]`: Each element is a string representing a single product entry.
- `key`: Displays the actual query keyword submitted to the API.
- `status`: A code indicating the status of the request.
- `time`: A timestamp recording when the API responded.

This is the basic introduction and detailed explanation of the core tool "Taobao Hot Words" of the `taobao-hot-words` server. We hope this guide will be helpful to you!