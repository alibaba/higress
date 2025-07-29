# Tourist Attraction Query

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi022105

# tourist-attraction-query Server Configuration Documentation

This document aims to provide a clear guide to the functionality and usage of the `tourist-attraction-query` server. Through this guide, users can learn how to use the service to query information about tourist attractions and how to interpret API responses.

## Overview of Functionality

`tourist-attraction-query` is a platform designed specifically for travel enthusiasts, allowing users to search for detailed information about tourist attractions based on different search criteria (such as city, province, or specific attraction name). This service not only returns basic information about the attractions (such as address, contact details, etc.), but also provides practical advice such as the best times to visit and ticket prices, greatly enriching the process of planning a trip.

## Tool Introduction

### Attraction Information Query

**Purpose**: This tool is primarily used to help users quickly locate and obtain specific details about their desired travel destinations. Users can set filtering conditions according to their needs to get more precise results.

**Use Cases**:
- When you need to plan a trip but are unfamiliar with the destination.
- If you want to compare differences between various attractions to make a choice.
- It is also very useful for tourists who want to learn more about the cultural background or historical stories of a particular place.

#### Parameter Description
- **city** (City): Enter the name of the city you wish to query.
- **page** (Page Number): Specify the page number of the data you want to view.
- **province** (Province): Provide the provincial administrative division to narrow down the search scope.
- **spot** (Attraction Name): Directly enter the name of the attraction you are interested in for an exact match.

#### Request Example
```yaml
url: https://scenicspot.market.alicloudapi.com/lianzhuo/scenicspot
method: GET
headers:
  Authorization: "APPCODE {{.config.appCode}}"
  X-Ca-Nonce: "{{uuidv4}}"
```

#### Response Structure
The response will be returned in JSON format and will include the following fields:

- **data**: An object containing the actual data.
  - **record[]**: Each record is an object representing the information of one attraction.
    - **addr**: Address
    - **grade**: Grade
    - **lat**: Latitude
    - **lng**: Longitude
    - **opentime**: Opening Time
    - **spot**: Name
    - **tel**: Contact Phone
    - **type**: Type
    - **url**: Official Website Link
    - **visittime**: Recommended Visit Time
  - **totalcount**: Total number of items
  - **totalpage**: Total number of pages
- **resp**: A message containing the request status.
  - **RespCode**: Status Code
  - **RespMsg**: Description

This is a brief introduction to the `tourist-attraction-query` server and its main functionalities. We hope this document helps you better understand and use our service!