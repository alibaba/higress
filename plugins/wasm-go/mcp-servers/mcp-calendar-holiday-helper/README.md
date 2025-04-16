# Chinese Almanac/Holiday Helper

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00066017

# MCP Server Configuration Document

## Function Overview
The `calendar-holiday-helper` server is a service platform focused on providing holiday-related information and almanac fortune queries. It supports various API calls to obtain data including but not limited to holiday lists, detailed holiday information for specific dates, and almanac information based on traditional Chinese culture. These services are very useful for individuals and organizations that need to schedule activities based on specific dates or want to know the auspiciousness of a particular day.

## Tool Introduction

### 1. Holiday List
- **Description**: This tool is used to list all holidays within a specified year.
- **Use Case**: Suitable for businesses planning annual holidays, the travel industry formulating promotional plans, etc.
- **Parameter Description**:
  - `year` (string): The year to query, defaulting to the current year. For non-current years, it also returns the current year's holiday data; next year's data can only be queried in December of the current year.

### 2. Holiday Details
- **Description**: This tool provides detailed holiday information for a specific date (defaulting to the current day).
- **Use Case**: Suitable for individuals or teams who want to know if a particular day is a holiday and its specific name.
- **Parameter Description**:
  - `date` (string): The date to query, defaulting to the current day.
  - `needDesc` (string): Whether to return a brief description of public holidays, international days, and traditional Chinese festivals, with a value of 1 indicating to return, defaulting to not returning.

### 3. Almanac Fortune (New Version) - Auspicious Times
- **Description**: Provides a daily auspicious time query service based on the traditional Chinese calendar.
- **Use Case**: Particularly useful for those who believe in choosing auspicious times for important decisions.
- **Parameter Description**:
  - `date` (string, required): The date to query, in the format yyyyMMdd.

### 4. Almanac Fortune (New Version) - Auspicious Deities and Inauspicious Spirits
- **Description**: Displays information about auspicious deities and inauspicious spirits affecting fortune on a specific date.
- **Use Case**: Helps users avoid unfavorable factors and seize favorable opportunities.
- **Parameter Description**:
  - `date` (string, required): The date to query, in the format yyyyMMdd.

### 5. Almanac Fortune (New Version) - Almanac
- **Description**: A comprehensive calendar service integrating the lunar calendar, Gregorian calendar, and other relevant astronomical information.
- **Use Case**: Widely used for arranging various customary activities in daily life.
- **Parameter Description**:
  - `date` (string, required): The date to query, in the format yyyyMMdd.

The above is an overview of the main tools and services provided by the `calendar-holiday-helper` server. By making reasonable use of these tools, users can more effectively manage their time and adjust their activity schedules as needed.