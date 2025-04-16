# Train Ticket Query

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi011240

# MCP Server Function Overview

This document aims to introduce the main functions of the MCP server and the tools used for configuring these functions, which are primarily designed for train ticket information query services. By calling different API interfaces, users can obtain detailed train schedules, ticket prices, and seat information.

## Function Overview

The `train-ticket-query` server mainly provides three train ticket query services tailored to different needs. These include ticket availability queries, station-to-station journey queries, and specific train information queries. Each tool is designed with specific parameters to meet user query requirements and returns structured data results, making it easy for further processing or display to end-users. This system is suitable for application developers or travel planning platforms that need to integrate train ticket-related information.

## Tool Introduction

### 1. Ticket Availability Query Interface
- **Purpose**: Used to retrieve available train schedules and their detailed information based on the departure location, destination, and date.
- **Use Case**: When a user wants to know the available train options from one city to another, this interface can quickly provide a list of all eligible options.
- **Request Parameters**:
  - `date`: Query date (required)
  - `end`: Destination (required)
  - `start`: Departure location (required)

### 2. Station-to-Station Query Interface
- **Purpose**: Provides a search function for direct connection routes between the starting point and the endpoint, allowing the option to specify whether to consider only high-speed rail options.
- **Use Case**: Suitable for situations where users want to know if there are direct trains between two locations or are only interested in high-speed rail services.
- **Request Parameters**:
  - `date`: Query date
  - `end`: Destination (required)
  - `ishigh`: Whether to show only high-speed trains (0 for no, 1 for yes)
  - `start`: Departure location (required)

### 3. Train Number Query Interface
- **Purpose**: Allows users to input a specific train number to get details of all stops along the route.
- **Use Case**: Very useful for passengers who have already decided which train to take but want more information about the stations en route.
- **Request Parameters**:
  - `date`: Query date
  - `trainno`: Train number (required)

This is an overview of the functionalities provided by the `train-ticket-query` server. Each tool has its unique use cases and can effectively help developers build richer and more practical transportation and travel-related applications.