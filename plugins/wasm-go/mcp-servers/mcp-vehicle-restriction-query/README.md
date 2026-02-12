# Vehicle Restriction Query

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi011138

# MCP Server Configuration Document

This document aims to provide a brief introduction to the main functions and tools of the `vehicle-restriction-query` MCP server. This server focuses on providing query services related to urban vehicle restriction information, implemented through two primary interfaces: city restriction queries and obtaining a list of supported cities.

## Function Overview

The `vehicle-restriction-query` server is primarily used to provide services that allow querying specific city's vehicle restriction policy information based on city codes and dates. Additionally, it allows users to obtain a list of all currently supported city codes and names. These features are particularly useful for applications that need to understand or comply with local traffic rules, such as map applications and navigation systems.

## Tool Introduction

### City Restriction Query Interface

- **Purpose**: Retrieve the vehicle restriction details of a specified city based on the provided city code and date.
- **Use Case**: Suitable for any application or service that needs to update its information about vehicle restrictions in a specific area in real-time.
- **Parameter Description**:
  - `city`: Required, represents the unique identifier of the queried city.
  - `date`: Required, defaults to the current day; used to specify the exact date for the query.
- **Request Method**: GET
- **Response Structure**:
  - Includes detailed information such as restricted areas, applicable dates, and restricted license plate numbers.
  - Each response field has a clearly defined data type, making it easy to parse and process.

### Get Cities Interface

- **Purpose**: Returns a list containing all queryable city codes and their corresponding Chinese names.
- **Use Case**: Very useful when an application needs to display a dropdown menu for users to select different cities.
- **Parameter Description**:
  - No additional parameters are required for this interface.
- **Request Method**: GET
- **Response Structure**:
  - Provides a simple JSON array format result, where each element contains a mapping of city code (`city`) and city name (`cityname`).
  - This structure makes it easy for the client to convert the data into a suitable form for display.

The above is a basic overview of the functions and services provided by the `vehicle-restriction-query` server. We hope this document will help developers better understand and utilize these API interfaces.