# Vehicle Information Inquiry

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00046930

# MCP Server Function Overview Document

## Function Overview
This MCP server primarily serves vehicle information inquiries. By inputting the Vehicle Identification Number (VIN), users can obtain detailed vehicle-related information, including but not limited to brand, model, and body dimensions. This service supports two tools: Basic VIN Vehicle Inquiry and Advanced VIN Vehicle Inquiry. Both provide accurate and comprehensive data feedback, but the advanced version offers a richer set of data fields to meet different levels of need.

## Tool Introduction

### VIN Vehicle Inquiry
- **Purpose**: To retrieve and return related vehicle details based on the provided VIN.
- **Usage Scenarios**: Suitable for situations where there is a need to quickly understand basic information about a specific vehicle, such as in used car markets or auto repair shops.
- **Parameter Description**:
  - `vin` (Required): Represents the chassis number to be queried.

### Advanced VIN Vehicle Inquiry
- **Purpose**: In addition to all the information retrieval capabilities of the standard version, this tool also includes more data points regarding vehicle specifications and technical details.
- **Usage Scenarios**: Ideal for professionals or organizations that require an in-depth understanding of vehicles, such as automobile manufacturers or research institutions.
- **Parameter Description**:
  - `vin` (Required): Refers to the specific chassis number used to initiate the query request.

Each tool defines its request template and response structure, ensuring consistency and predictability in API calls. The response section not only lists all possible data items and their type descriptions but also includes an example of the raw response, helping developers better understand how to parse the actual returned data. Additionally, all HTTP requests will use the POST method and require setting appropriate header information, particularly the `Authorization` header, which is used to verify the client's identity and ensure that only authorized applications can access this sensitive information.