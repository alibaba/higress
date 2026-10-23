# Fund Data Query

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi026966

# MCP Server Function Overview Document

## Function Overview
This MCP server, named `fund-data-query`, is primarily used to provide fund-related data query services. It achieves the acquisition of various types of fund information through a series of tools, including but not limited to lists of funds on sale, fund valuation data, and fund announcements. These tools all use the HTTP GET method to send requests to specified URLs and require specific application codes (appCode) for authorization verification. The data format returned by each tool is JSON.

## Tool Overview

### List of Funds on Sale
- **Purpose**: Used to obtain basic information about all purchasable funds currently available in the market.
- **Usage Scenario**: When users or systems need to display the latest fund products for selection, this interface can be called to fetch the most recent data.
- **Parameter Description**:
  - `limit`: Number of items displayed per page.
  - `page`: The current page number being viewed.

### Fund Valuation Data
- **Purpose**: Provides the latest valuation details for a single fund.
- **Usage Scenario**: Suitable for individual investors to track the performance of a specific fund they hold.
- **Parameter Description**:
  - `fundcode`: The specific fund code to be queried.

### Fund Announcement Data
- **Purpose**: Lists all official notifications or important matters issued by a specific fund.
- **Usage Scenario**: Helps investors stay informed about significant changes in their invested funds.
- **Parameter Description**:
  - `fundcode`: Target fund code.
  - `limit`: Maximum number of results to return.
  - `page`: Pagination index.

### Fund Dividend and Distribution
- **Purpose**: Displays the dividend records of a particular fund over a certain period.
- **Usage Scenario**: Very useful for investors who are concerned with the distribution strategy of returns.
- **Parameter Description**:
  - `fundcode`: The fund code of the object to be queried.

### Fund Historical Managers
- **Purpose**: Provides information about all fund managers who have managed a given fund in the past.
- **Usage Scenario**: Consider the impact of different managers during different time periods when evaluating the long-term performance of a fund.
- **Parameter Description**:
  - `fundcode`: Identifier of the specific fund.

... [Some tool introductions are omitted here for brevity]

### New Fund Issuance List
- **Purpose**: Lists newly issued fund products.
- **Usage Scenario**: Provides references for users looking for new investment opportunities.
- **Parameter Description**:
  - `limit`: The maximum number of records to return in a single request.
  - `page`: The page number of the data requested.
  - `saleStatus`: Whether to show only funds that are currently on sale or also include those not yet open for subscription. The default value is true.

The above is a brief description of the main tools provided by the MCP server, along with their basic functions and usage scenarios. Each tool is designed with corresponding API endpoints, and developers can call the appropriate interfaces based on actual needs to obtain the required information.