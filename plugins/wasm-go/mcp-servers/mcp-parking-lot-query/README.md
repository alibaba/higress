# Parking Lot Inquiry

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00050817

# MCP Server Function Overview Document

## Function Overview
This MCP server is primarily responsible for handling parking lot information query requests. It supports retrieving detailed parking lot data based on geographical location, city name, or specific parking lot ID through the interfaces provided by the Alibaba Cloud API Marketplace. This service is suitable for applications that need to integrate real-time parking information, such as navigation software and smart city management systems.

## Tool Introduction

### Parking Lot Inquiry_Based on Surroundings
- **Purpose**: This tool allows users to search for nearby parking lots based on specified geographic coordinates (latitude and longitude) and an optional distance range.
- **Use Case**: Suitable for mobile app or website development to provide users with parking options near their current location.
- **Parameter Description**:
  - `lat` (Required): Latitude value
  - `lng` (Required): Longitude value
  - `distance`: Search radius, default is 1000 meters
  - `page`: Page number of the results, default is the first page
  - `size`: Number of results per page, default is 10 records

### Parking Lot Inquiry_Based on City
- **Purpose**: This function is used to retrieve a list of parking lots by city name.
- **Use Case**: Very useful when an application needs to display all available parking lots within a specific city.
- **Parameter Description**:
  - `city` (Required): The name of the city to be queried
  - `page`: Page number of the results, default is the first page
  - `size`: Page size, default is 10 items per page

### Parking Lot Inquiry_Details
- **Purpose**: Using the unique identifier (ID) of a parking lot, this API can be called to obtain all detailed information about a specific parking lot.
- **Use Case**: Suitable for situations where detailed information about a single parking lot is needed, such as prices, location descriptions, etc.
- **Parameter Description**:
  - `id` (Required): The unique identifier of the target parking lot

Each tool is configured with the corresponding request template, including URL, HTTP method, and necessary header information, and defines the response format to help developers understand and parse the returned data structure. These tools collectively form a powerful parking lot information service system that can meet different levels of information needs.