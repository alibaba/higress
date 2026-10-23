# Enterprise Patent Query

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00049059

# MCP Server Configuration Document

This server is primarily used for querying enterprise patent information, supporting the retrieval of patent lists and detailed information.

## Function Overview
The `business-patent-query` server focuses on providing services related to patent information for enterprises or individual users. Through this service, users can easily search for all relevant patents within a specific technical field, which helps in avoiding infringement of others' intellectual property rights and guiding their own R&D activities. It includes two core functions: patent information list retrieval and patent detail viewing.

## Tool Introduction

### 1. Patent Information List
- **Purpose**: This tool allows users to find related patent lists based on keywords (such as company name, social credit code, etc.).
- **Use Cases**: It is used when a comprehensive understanding of the patent layout of a particular industry or company is needed; it can also be used for market research, competitor analysis, and other areas.
- **Parameter Description**:
  - `dtype`: The format of the returned data, default is JSON.
  - `keyword`: A required parameter, used to specify the search keyword.
  - `pageIndex`: Specifies the page number of the returned results, default is the first page.
  - `pageSize`: Sets the number of results displayed per page, default is 10 records, with a maximum of 10 records.

### 2. Patent Information Details
- **Purpose**: Based on a known patent ID, this tool can obtain the specific details of a single patent.
- **Use Cases**: It is suitable for in-depth study of a specific patent content or when detailed information about a certain technical solution is needed.
- **Parameter Description**:
  - `dtype`: Defines the format of the response data, default is JSON.
  - `id`: A required field, representing the unique identifier of the patent to be queried, typically obtained from the "Patent Information List" interface.

Each tool provides detailed request and response templates to ensure that developers can correctly call the API and handle the returned data. Additionally, the response structure for each tool includes basic information about the requested patent as well as some additional metadata, such as order number, status code, etc., to facilitate tracking the request status and parsing the data.