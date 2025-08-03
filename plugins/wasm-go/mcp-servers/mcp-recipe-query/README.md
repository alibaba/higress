# Recipe Query

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00045093

# MCP Server Function Overview

This document introduces the main functions and purposes of the MCP server, as well as the specific use cases for each tool.

## Function Overview

The MCP server is primarily used to provide recipe-related API services. Through these APIs, users can perform various types of queries, including searching recipes by category, retrieving detailed information by ID, viewing a list of recipe categories, and searching recipes using keywords. Additionally, all requests must carry specific authentication information to ensure secure access.

## Tool Introduction

### Search by Category
- **Description**: This tool supports finding results that meet certain criteria from a vast collection of recipes, allowing searches based on different categories or keywords. Each record includes main ingredients, auxiliary ingredients, and detailed preparation steps.
- **Parameters**:
  - `classid` (Required): The specified category identifier.
  - `num` (Required): The number of results to return.
  - `start` (Optional): The starting position in the result set, default is 0.
- **Use Case**: Very useful when you need to quickly find multiple recipes within a specific category.

### Query Details by ID
- **Description**: Allows users to obtain more detailed information based on the provided recipe ID.
- **Parameters**:
  - `id` (Required): The unique identifier of the target recipe.
- **Use Case**: Suitable for situations where you already know the exact ID of a dish and want to learn all its details.

### Recipe Categories
- **Description**: Lists all recipe categories, helping developers understand the data structure and build richer application interfaces accordingly.
- **Parameters**: None
- **Use Case**: Crucial for applications that need to display all available categories or allow users to select their preferred categories.

### Recipe Search
- **Description**: Allows full-text search operations across the entire database using keywords.
- **Parameters**:
  - `keyword` (Required): The search term used for matching.
  - `num` (Required): The number of search results to return.
  - `start` (Optional): The starting position in the search results, default is 0.
- **Use Case**: Ideal for applications that aim to find relevant recipes through simple keyword input.

The above provides a brief introduction to the various functions and services offered by the MCP server. By effectively utilizing these tools, developers can easily create food applications that meet different needs.