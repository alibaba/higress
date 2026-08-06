# Product Barcode Query

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi011032

# MCP Server Configuration Document

## Function Overview

`product-barcode-query` is a specialized service for querying domestic product barcode information. It supports obtaining details related to a specified barcode through API calls, including but not limited to product name, brand, price, and other key information. This service is particularly suitable for applications that require quick and accurate access to large amounts of product data, such as e-commerce platforms, inventory management systems, or consumer rights protection platforms.

## Tool Introduction

### Product Barcode Query

- **Purpose**: This tool allows users to input a Chinese standard product barcode (starting with 69) and returns the corresponding product information.
- **Use Cases**: It is ideal for businesses or individual developers who wish to quickly retrieve specific product details based on barcodes. For example, online shopping websites can use this tool to immediately display relevant product details after a user scans a product barcode; retailers can also use this feature to enhance the efficiency and accuracy of their inventory management systems.

#### Parameter Description
- `code`: Domestic product barcode (must start with 69). This is one of the required parameters when making a request.
  - Type: String
  - Required: Yes
  - Location: Query string

#### Request Template
- **URL**: `https://barcode14.market.alicloudapi.com/barcode`
- **Method**: GET
- **Headers**:
  - `Authorization`: APPCODE {{.config.appCode}}
  - `X-Ca-Nonce`: '{{uuidv4}}'

#### Response Structure
The response will be returned in JSON format and will include the following main fields:
- `showapi_res_body`: Contains the actual product information.
  - `showapi_res_body.code`: Barcode
  - `showapi_res_body.engName`: English name
  - `showapi_res_body.flag`: Query result flag
  - `showapi_res_body.goodsName`: Product name
  - `showapi_res_body.goodsType`: Product category
  - `showapi_res_body.img`: Image URL
  - `showapi_res_body.manuName`: Manufacturer
  - `showapi_res_body.note`: Note information
  - `showapi_res_body.price`: Reference price (unit: RMB)
  - `showapi_res_body.remark`: Query result remarks
  - `showapi_res_body.ret_code`: Return code
  - `showapi_res_body.spec`: Specifications
  - `showapi_res_body.sptmImg`: Barcode image
  - `showapi_res_body.trademark`: Trademark/Brand name
  - `showapi_res_body.ycg`: Place of origin
- `showapi_res_code`: Response status code
- `showapi_res_error`: Error message (if any)

The above is a basic overview of the `product-barcode-query` service and its related tools. We hope this document helps you use these resources more effectively.