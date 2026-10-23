# Oil Price Inquiry

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00062739

# MCP Server Configuration Function Overview

This document aims to provide a clear overview of the functions and a brief introduction to the tools supported by the `oil-price-query` MCP server. Through this documentation, users can better understand how to utilize these services to obtain the information they need.

## Function Overview

The `oil-price-query` server is primarily responsible for handling requests related to oil prices across different regions in China. It fetches the latest oil price data through the Alibaba Cloud Marketplace API and returns it in a structured format to the caller. This service is particularly suitable for applications that require real-time monitoring or analysis of fuel price trends in various regions, such as car refueling apps, logistics cost estimation systems, etc.

## Tool Introduction

### Today's Oil Price

- **Purpose**: This tool allows users to query the prices of various types of gasoline for a specified province on the current date.
- **Use Cases**: Suitable for individual users who want to know the latest oil prices in their area or other regions of interest; businesses can use it for cost control and budget planning by referring to the latest fuel cost standards.
- **Parameter Description**:
  - `prov`: The name of the province to be queried, such as "Beijing" or "Guangxi". This parameter must be specified in the URL query string.
  
- **Request Example**:
  ```http
  GET https://smjryjcx.market.alicloudapi.com/oil/price?prov=Beijing
  Authorization: APPCODE <your_app_code_here>
  X-Ca-Nonce: <random_uuid_value>
  ```

- **Response Structure**:
  - **code**: Status code (integer)
  - **data**: Data object
    - **data.list[]**: List of oil prices
      - **ct**: Update time (string)
      - **p0**: Price of diesel 0 (string)
      - **p89**: Price of gasoline 89 (string)
      - **p90**: Price of gasoline 90 (string)
      - **p92**: Price of gasoline 92 (string)
      - **p93**: Price of gasoline 93 (string)
      - **p95**: Price of gasoline 95 (string)
      - **p97**: Price of gasoline 97 (string)
      - **p98**: Price of gasoline 98 (string)
      - **prov**: Queried province (string)
    - **orderNo**: Order number (string)
    - **ret_code**: Return status code, 0 indicates success (integer)
  - **msg**: Response message (string)
  - **success**: Whether the request was successful (boolean)

Please replace `<your_app_code_here>` with your valid AppCode obtained from Alibaba Cloud. Additionally, generate a new random UUID for each request to use as the value of the `X-Ca-Nonce` header.