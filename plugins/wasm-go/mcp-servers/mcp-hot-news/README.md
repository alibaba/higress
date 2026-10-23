# Hot News

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi011178

# MCP Server Configuration Documentation

This document aims to provide users with detailed information about the MCP server `hot-news`, including its main functionalities and the specific uses and application scenarios of the integrated tools.

## Function Overview

The MCP server `hot-news` is primarily used for managing and providing news-related API services. It supports various operations such as searching for news based on keywords, fetching the latest message lists by channel, and querying available news channels. Through these features, users can easily access the latest news information and filter out content of interest according to their needs. Additionally, the server supports custom application code settings (appCode), enhancing security and flexibility.

## Tool Introduction

### Search News Interface
- **Purpose**: Allows users to retrieve relevant news entries based on specific keywords.
- **Use Case**: Very useful when users need to quickly find information related to a particular topic or event.
- **Request Parameters**:
  - `keyword`: Required, specifies the keyword to search for.
- **Response Structure**: Returns a list of all news entries associated with the keyword, each record containing details such as title, timestamp, and source link.

### Get News Interface
- **Purpose**: Pulls a specified number of new articles from a specific channel.
- **Use Case**: Suitable for browsing the latest updates in a specific field (e.g., technology, sports).
- **Request Parameters**:
  - `channel`: Required, used to select the target news channel.
  - `num`: Optional, default value is 10, maximum can be set to 40, specifies the number of results to return.
  - `start`: Optional, defaults to 0, indicates the starting index of the records.
- **Response Format**: The returned dataset includes an overview of the specified number of articles in the selected channel, with each item accompanied by detailed metadata.

### Get News Channels Interface
- **Purpose**: Lists all available news channels.
- **Use Case**: Helps developers understand which news categories are currently supported by the system, making it easier to correctly fill in the `channel` field when calling other APIs.
- **Request Parameters**: None
- **Response Content**: A simple array of strings, where each element represents a unique news channel name.

The above provides a basic introduction to the `hot-news` server and its built-in tools. We hope this will help you better understand and utilize this service!