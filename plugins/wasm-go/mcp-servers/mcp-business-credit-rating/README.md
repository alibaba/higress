# Enterprise Credit Rating

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00067564

# MCP Server Configuration Overview

## Function Overview
This MCP server is primarily used to handle query requests related to enterprise credit ratings. By interacting with specific APIs available on the Alibaba Cloud Marketplace, this service can return detailed credit rating information of a company based on provided information such as the company name, registration number, or social credit code. This allows users to conveniently obtain the latest credit status of target companies, including but not limited to bond credit ratings, entity ratings, and rating outlooks.

## Tool Overview

### Enterprise Credit Rating
- **Purpose**: Provides an interface for querying the credit rating information of specified enterprises.
- **Use Cases**: Suitable for scenarios where a comprehensive understanding of an enterprise's credit status is needed, such as when financial institutions decide whether to provide loans to a company; or when suppliers investigate the creditworthiness of potential clients before choosing partners.

#### Parameter Description
- **keyword** (Required): The search keyword, which can be the company name, registration number, or social credit code.
- **pageNum**: The page number in the request pagination, defaulting to 1.
- **pageSize**: The number of result items per page, with a default value of 10.

#### Request Template
- **URL**: `https://slyhonour.market.alicloudapi.com/credit/rating`
- **Method**: GET
- **Headers**:
  - Authorization: Use the application code as the authentication method
  - X-Ca-Nonce: A unique identifier generated automatically

#### Response Structure
- **code**: Status code
- **data**:
  - **items[]**:
    - alias: Rating company alias
    - bondCreditLevel: Bond credit level
    - gid: Global ID
    - logo: Rating company logo
    - ratingCompanyName: Rating company name
    - ratingDate: Rating date
    - ratingOutlook: Rating outlook
    - subjectLevel: Subject level
  - orderNo: Order number
  - total: Total number of records
- **msg**: Message content returned
- **success**: Boolean flag indicating whether the operation was successful

This tool provides detailed enterprise credit assessment data, helping users to quickly and accurately evaluate a company's financial health and its ability to repay debts.