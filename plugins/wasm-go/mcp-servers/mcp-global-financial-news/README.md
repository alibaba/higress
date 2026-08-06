# Global Financial News Briefs

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi027789

# MCP Server Configuration Documentation

## Overview
This MCP server provides a series of financial-related APIs, aimed at offering users global financial information. This information includes, but is not limited to, financial holiday calendars, real-time news briefs, economic data release calendars, and important financial events. Through these tools, users can promptly obtain the latest financial information, helping them make more accurate decisions in their investments.

## Tool Introduction

### Global Financial Holiday Calendar
- **Purpose**: This tool provides financial holiday schedules for countries/regions worldwide, which is very useful for investors who need to know if a specific date is a working day or a holiday.
- **Use Case**: When planning cross-border transactions, this feature can be used to avoid delays caused by non-working days.
- **Parameters**:
  - `date` (required): The specific date to query.
  - `year` (required): The year.

### Global Real-Time Financial News Briefs
- **Purpose**: Provides the latest global financial news updates, allowing users to quickly learn about major financial events happening around the world.
- **Use Case**: Suitable for professionals who wish to continuously track financial market dynamics.
- **Parameters**:
  - `lastOutId`: The output ID from the last request, used for paginated loading of more content.
  - `size`: The number of messages per page.

### Global Real-Time Financial News and Economic Data
- **Purpose**: In addition to providing the latest financial news, it also includes important economic indicator data such as GDP growth rate, unemployment rate, and other key statistics.
- **Use Case**: Suitable for researchers who need to conduct comprehensive analysis of economic conditions and market trends.
- **Parameters**:
  - `lastOutId`: Same as above.
  - `size`: Same as above.

### Global Economic Data Calendar
- **Purpose**: Lists upcoming economic reports and their expected values, helping investors predict market trends.
- **Use Case**: Particularly valuable for those who want to prepare in advance and adjust their investment portfolios.
- **Parameters**:
  - `date` (required): The date of interest.
  - `year` (required): The corresponding year.

### Global Financial Events Calendar
- **Purpose**: Records the schedule of important meetings, speeches, and other activities that impact the global economy.
- **Use Case**: Very important for entrepreneurs or analysts who are concerned with changes in international financial policies.
- **Parameters**:
  - `date` (required): The day to view significant events.
  - `year` (required): The specified year.

Each tool supports invocation via the HTTP GET method and requires the inclusion of authentication information (`Authorization: APPCODE`) and a randomly generated nonce value in the request header to ensure security. The returned data format is JSON, making it easy to parse and process.