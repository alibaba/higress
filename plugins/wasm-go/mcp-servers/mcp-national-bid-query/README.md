# National Tender and Bid Inquiry

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00066410

# MCP Server Function and Tool Introduction

## Function Overview
This MCP server is primarily used for processing and querying information related to tender and bid projects. By integrating a series of specific tools, it provides comprehensive services from list retrieval to detailed information acquisition. Each tool is designed for different business needs, aiming to improve data access efficiency and simplify information management processes.

## Tool Introduction

### Tender and Bid Project List Query
This tool allows users to filter and retrieve a list of tender and bid projects based on various conditions. It is suitable for broad searches or initial market trend understanding.
- **Parameter Description**:
  - `cityCode`: City code
  - `classId`: Information category (1: Tender, 2: Bid)
  - `endDate`: End date of the query
  - `keyword`: Keyword search
  - `pageIndex`: Current page number
  - `pageSize`: Number of items per page
  - `provinceCode`: Province code
  - `searchMode`: Search mode (1: All, 2: Title, 3: Content)
  - `searchType`: Search type (1: Smart subscription, 2: Precise subscription, 3: Advanced definition)
  - `startDate`: Start date of the query

### Structured Query for Tender and Bid Projects
This tool provides detailed structured information queries for individual tender and bid projects. It is particularly useful for gaining in-depth insights into specific project details.
- **Parameter Description**:
  - `id`: Unique project identifier
  - `publishTime`: Project publication date

### Detailed Query for Tender and Bid Projects
This tool is used to obtain comprehensive information about a specified tender or bid project, including but not limited to contact persons and contact details. It is very useful for those who need the most detailed information.
- **Parameter Description**:
  - `id`: Project ID
  - `publishTime`: Publication time

Each tool supports invocation via the POST method and must include the `Content-Type`, `Authorization`, and `X-Ca-Nonce` headers. The `Authorization` value should be replaced with a valid `APPCODE`. The response will be returned in JSON format, containing fields such as status code (`code`), message (`msg`), and relevant data (`data`), with the specific content depending on the selected tool.