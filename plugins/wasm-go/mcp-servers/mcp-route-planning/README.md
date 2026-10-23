# Route Planning

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00065113

# MCP Server Function Overview Document

## Function Overview
The MCP server offers a comprehensive set of path planning solutions, supporting route planning for multiple modes of transportation including public transit, walking, bicycle riding, electric bike riding, driving, and more. It also provides a function for measuring travel distances. These services are designed to help users choose the most suitable travel options based on different needs and preferences, thereby improving travel efficiency and saving time. Additionally, the MCP server takes into account factors such as real-time traffic conditions and personal preference settings to ensure that the provided routes are the best possible solutions.

## Tool Introduction

### Public Transit Route Planning
- **Purpose**: Based on the user's starting point and destination information, it calculates one or more reasonable bus routes.
- **Usage Scenarios**: Suitable for passengers who need to travel by bus, especially when users are unfamiliar with local bus routes or wish to find the fastest and most economical travel plan.
- **Parameter Description**:
  - `alternativeRoute`: Returns a specified number of route options.
  - `date` and `time`: Specifies the exact date and time for the query.
  - `destCityCode`, `origCityCode`: Specifies the destination city code and the origin city code, respectively.
  - `destination`, `origin`: The latitude and longitude coordinates of the destination and origin, respectively.
  - `maxTrans`: Limits the maximum number of transfers.
  - Additional parameters allow for further customization of the query conditions, such as whether to consider night buses (`nightFlag`), specific strategies (`strategy`), etc.

### Walking Route Planning
- **Purpose**: Provides users with the best walking route from one location to another.
- **Usage Scenarios**: Suitable for short trips or situations where driving is not feasible.
- **Main Parameters**:
  - `alternativeRoute`: An optional parameter to obtain multiple alternative routes.
  - `destination` and `origin`: Required fields that define the start and end points of the journey.
  - `showFields`: Controls which fields are included in the returned results.

### Electric Bike/Bicycle Route Planning
- **Purpose**: Designed for electric bicycle or regular bicycle riders, providing optimized route suggestions.
- **Applicable Situations**: Particularly suitable for daily commuting or leisure cycling activities.
- **Configuration Options**: Similar to walking route planning, it also supports setting multiple alternative routes (`alternativeRoute`) and other personalized settings.

### Travel Distance Measurement
- **Objective**: Used to estimate the straight-line distance or the distance according to a specific mode of transportation between two geographic locations.
- **Application Scenarios**: Very useful for users who need to know the approximate distance between two places.
- **Key Parameters**: The `type` parameter determines whether the calculation method is direct distance or the actual travel distance based on a specific mode of transportation.

### Driving Route Planning
- **Function Description**: Given the starting point and destination, the system will recommend the best driving route by considering current traffic conditions, traffic restrictions, and other factors.
- **Target Audience**: Aimed at private car owners or professional drivers, especially those who frequently need to drive long distances.
- **Important Features**: Supports advanced options such as setting avoidance areas (`avoidpolygons`), special vehicle types (`carType`), and different driving strategies (`strategy`).

The above is a basic introduction to the various services supported by the MCP server. Each tool has its unique application area and can be fine-tuned to meet more specific needs by adjusting the corresponding parameters.