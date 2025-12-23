# JD Hot Words

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi022081

# MCP Server Configuration Document

## Overview
This MCP server, named `jd-hot-words`, is primarily used to support keyword analysis services in the e-commerce domain. Through the interfaces provided by this server, users can query the ranking information of popular search terms related to specific products on the JD platform. This is of great value for understanding market trends, consumer interest points, and optimizing online marketing strategies.

## Tool Introduction

### JD Product Hot Search
**Purpose**:
The JD product keyword search ranking query tool allows developers or merchants to retrieve a list of the most popular related search terms based on specified product keywords. This feature is particularly useful for businesses that need to keep up with the latest shopping trends, helping them better target their customer base and adjust product promotion strategies.

**Use Cases**:
- **Market Research**: Analyze what types of products potential customers are looking for.
- **SEO Optimization**: Determine which keywords to target for website content SEO.
- **Advertising Campaigns**: Develop more effective online advertising plans based on frequently searched keywords.
- **Inventory Management**: Adjust inventory levels to meet seasonal or trending changes.

#### Parameter Description
- `key`: The product keyword provided by the user, one of the required fields when initiating a request. It is used to specify the category or specific name of the product for which related hot words are desired.

#### Request Format
- **URL**: https://jdgoods.market.alicloudapi.com/jdgoods
- **Method**: GET
- **Headers**:
  - `Authorization`: Use the preset application code (`appCode`) as the authentication token.
  - `X-Ca-Nonce`: A unique identifier generated automatically to ensure the uniqueness of each request.

#### Response Structure
The response will be returned in JSON format and will include the following fields:
- **goodsList[]**: An array of strings containing the descriptions of the most relevant popular products for the given keyword.
- **key**: The original search keyword returned.
- **status**: A message indicating the status of the API call.
- **time**: A timestamp recording the time of the API processing.

This is a brief introduction to the `jd-hot-words` MCP server and its main tool, the JD Product Hot Search. By utilizing these resources, you can effectively monitor and analyze consumer behavior patterns on e-commerce platforms.