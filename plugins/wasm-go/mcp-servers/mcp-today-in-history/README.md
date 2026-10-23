# Today in History

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi011517

# Introduction to MCP Server Functions and Tools

## Function Overview
This MCP server project, named "today-in-history," aims to provide a query service for historical events that occurred on specific dates. By integrating with external APIs, users can obtain information about historical events on a particular date or the current date, including but not limited to the specific events, related image links, and detailed event descriptions. This service is not only suitable as an educational resource in the education sector but also useful for news media industries when creating special reports.

## Tool Introduction

### Today in History
- **Purpose**: This tool allows users to query significant historical events that occurred on a specified date. It supports on-demand loading of more detailed background information about the events.
- **Use Cases**:
  - Educational institutions can use this tool to enrich classroom content and increase students' interest in history.
  - News websites or applications can display interesting historical facts daily to attract readers.
  - Individual users can also explore important events that happened on past days using this tool.

#### Parameter Description
- `date` (string, location: query parameter)
  - Description: The specific date to query. If not provided, it defaults to the current date.
- `needContent` (string, location: query parameter)
  - Description: Whether to return detailed event information. Set to "1" to include, "0" to exclude.

#### Request Template
- **URL**: https://today15.market.alicloudapi.com/today-of-history
- **Method**: GET
- **Headers**:
  - Authorization: Use the `appCode` value from the configuration file for authentication.
  - X-Ca-Nonce: A unique identifier generated automatically to ensure the uniqueness of each request.

#### Response Structure
The response will be returned in JSON format and will include the following main parts:

- **showapi_res_body** (object): Contains the actual data information
  - **list** (array): A series of historical event entries
    - **content** (string): If `needContent=1` is set, this will display the detailed content of the event.
    - **day** (integer): The day of the event
    - **img** (string): Link to the related image
    - **month** (integer): The month
    - **title** (string): Title of the event
    - **year** (string): The year
  - **ret_code** (integer): Return status code, 0 indicates success
- **showapi_res_code** (integer): Overall response status code, 0 means the request was processed successfully
- **showapi_res_error** (string): Error message provided if an error occurs; otherwise, it is empty

This concludes the basic introduction to the "today-in-history" MCP server and its core tools. We hope this information helps you better understand and utilize this service!