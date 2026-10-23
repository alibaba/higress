# Book Query

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00066353

## Overview

The `book-query` service is primarily used to query detailed information about books using their ISBN numbers. This service accepts a request containing an ISBN number and sends a request to an external API to retrieve all available data related to that ISBN, including but not limited to the author, publication date, and publisher. This feature is very useful for library management systems, online bookstores, and other applications that need to quickly look up book details based on ISBN.

## Tool Introduction

### ISBN Number Query
- **Purpose**: This tool allows users to input an ISBN number and obtain comprehensive information about the corresponding book.
- **Use Cases**:
  - When developers or system integrators need to provide users with a book search function based on ISBN.
  - For those who want to quickly learn all relevant information about a book (such as author, edition, price, etc.) through its ISBN.
  - As a basic data query method when building platforms that involve managing or selling a large number of books.

#### Parameter Description
- `isbn`: The ISBN number provided by the user, which is a string. This is the only required piece of information in the query process, used to locate a specific book record.

#### Request Example
- **URL**: https://lhisbnshcx.market.alicloudapi.com/isbn/query
- **Method**: POST
- **Headers**:
  - Content-Type: application/x-www-form-urlencoded
  - Authorization: Use the APP CODE for authentication
  - X-Ca-Nonce: A generated random UUID value to ensure the uniqueness of each request

#### Response Structure
The response will be returned in JSON format and will include the following main fields:
- `code`: The status code returned by the interface, different from the HTTP status code.
- `data`: An object containing specific book information.
  - `details[]`: An array of specific book details, where each element represents a record and includes various attributes such as author, title, and publisher.
- `msg`: A description message corresponding to the returned status code.
- `taskNo`: The task order number, which can be used for subsequent service provider verification.

This is a brief overview of the MCP server and its components mentioned in the YAML configuration file. Through these tools and services, effective querying and management of book information can be achieved.