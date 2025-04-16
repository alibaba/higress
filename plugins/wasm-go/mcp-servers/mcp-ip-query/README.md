# IP location query

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00054907

# MCP Server Configuration Function Overview

## Function Overview

This service can analyze the user's location based on their IP address. It can automatically obtain the IP without requiring the user to actively provide it (based on the API gateway's capabilities). The location information can also be used to determine the user's timezone, allowing for the provision of accurate local time based on their location.

## Tool Introduction

### Enhanced IP Address Query

- **Purpose**: This tool allows users to input an IP address (supports IPv6) and then returns the corresponding detailed location information.
- **Use Cases**: Suitable for application development that requires precise geographic positioning of visitors or clients, such as online advertising placement and content localization services.
- **Request Parameters**:
  - `ip` (required): The IP address to be queried.
- **Response Structure**: Returns data in JSON format, containing the status code of the query result, specific geographical information (e.g., city name, district code), status message, and the task ID of this request.
- **Notes**: In addition to basic location information, it also includes latitude and longitude coordinates.

### Precise IP Address Query

- **Purpose**: Compared to the previous version, this tool provides more dimensional information along with basic geographical details, such as the operator's name, time zone, and does not return latitude and longitude data for IPv4 addresses.
- **Use Cases**: Suitable for services that not only care about where the visitor is from but also need to understand the characteristics of their network environment, such as cybersecurity monitoring systems or the design of globally distributed applications.
- **Request Parameters**:
  - `ip` (required): The IP address to be queried.
- **Response Structure**: Also presented in JSON format, the content is rich and diverse, covering everything from continent to postal code, and also retains the task identifier for tracking.
- **Features**: Enhances the comprehensiveness of the information, especially with the different handling methods for IPv4 and IPv6, making it a more flexible choice.
