# Logistics Tracking Query

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00048162

# MCP Server Function Overview Document

## Function Overview

The **logistics-tracking-query** server is primarily used to handle queries related to express delivery, including but not limited to automatically identifying the courier company associated with a tracking number and obtaining detailed transportation path information for a specified package. This service is very useful for applications that need to integrate information from multiple different courier service providers, as it simplifies the process of connecting to multiple independent APIs and provides a unified data access interface.

## Tool Introduction

### Tracking Number to Identify Courier Company

- **Purpose**: Intelligently match the corresponding courier service provider based on the tracking number provided by the user.
- **Use Case**: Suitable for e-commerce platforms, order management systems, and other scenarios where it is necessary to quickly determine the courier company used by customers.
- **Parameter Description**:
  - `mailNo` (Required): The tracking number input by the user, used to find the associated courier company information.

### Logistics Tracking Query

- **Purpose**: Call the API to obtain the latest logistics updates based on the tracking number and related information.
- **Use Case**: Suitable for all services that wish to provide real-time package location updates to their users, such as shopping websites and third-party logistics monitoring applications.
- **Parameter Description**:
  - `cpCode` (Required): The unique identifier code of the courier company.
  - `mailNo` (Required): The tracking number to be queried.
  - `phone` (Optional): When involving SF Express or Fengwang, provide the full phone number or the last four digits of the recipient's/sender's phone number to enhance verification accuracy.

Together, these two tools effectively provide a one-stop solution for users, from identifying the courier company to tracking the entire status of the package, greatly enhancing the user experience while also reducing the integration burden for developers.