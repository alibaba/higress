# Historical Stock Data

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi010845

# MCP Server Function Overview Document

## Function Overview
The MCP server primarily provides stock market-related data query services, including but not limited to historical data, real-time data, and technical indicators. Through a series of API interfaces, users can conveniently obtain stock and index information from the Shanghai, Shenzhen, and Hong Kong markets. These tools and services are designed to help investors and developers quickly access the necessary data for analysis, strategy formulation, or application development.

## Tool Introduction

### BOLL Band Query
- **Purpose**: Used to query BOLL band data for stocks in the Shanghai, Shenzhen, and Hong Kong markets.
- **Use Case**: Suitable for technical analysis of the price fluctuation range of a specific stock over a certain period.
- **Parameters**:
  - `begin_date`: Start date, default is the current day.
  - `code`: The stock code to be queried, required.
  - `end_date`: End date, default is the current day, with a maximum time span of one quarter.
  - `fqtype`: Type of adjustment, default is no adjustment.

### Single Stock Historical Data Reference
- **Purpose**: Provides historical data reference for a single stock from the previous trading day.
- **Use Case**: Suitable for understanding the performance of a particular stock on the most recent trading day.
- **Parameters**:
  - `code`: Stock code, supports pinyin initials, required.
  - `needIndex`: Whether to return index information, optional.
  - `need_k_pic`: Whether to return the K-line chart URL, optional.

### Major Index Historical Monthly Line Query
- **Purpose**: Queries historical K-line chart data for major indices.
- **Use Case**: Very useful for technical analysts who want to view the trend of the major index over a specific period.
- **Parameters**:
  - `beginDay`: Start time, default is the current day.
  - `code`: Index code, required.
  - `time`: Query period, default is 5-minute K-line.

### Batch Historical Data Reference Query
- **Purpose**: Batch retrieval of historical data for multiple stocks.
- **Use Case**: Particularly useful when needing to analyze the historical performance of multiple stocks simultaneously.
- **Parameters**:
  - `needIndex`: Whether to return information on the four major stock indices, optional.
  - `stocks`: Multiple stock codes separated by commas, up to 20 codes, required.

### Shanghai and Shenzhen KDJ Query
- **Purpose**: Queries the KDJ stochastic indicator for stocks in the Shanghai and Shenzhen markets.
- **Use Case**: Used to evaluate the strength of stock price trends and potential turning points.
- **Parameters**:
  - `code`: Stock code, required.
  - `end`: End date, required.
  - `fqtype`: Type of adjustment, default is no adjustment.
  - `start`: Start date, required.

### Shanghai and Shenzhen MACD Data Query
- **Purpose**: Retrieves the MACD indicator values for a stock.
- **Use Case**: MACD is a commonly used trend-following momentum indicator for identifying potential buy or sell signals.
- **Parameters**:
  - `code`: Stock code.
  - `end`: End time.
  - `fqtype`: Type of adjustment, default is no adjustment.
  - `start`: Start time.

### Stock List Query
- **Purpose**: Lists all stocks that meet specified conditions.
- **Use Case**: Used when you need to filter and display a list of stocks by market or other criteria.
- **Parameters**:
  - `market`: Market abbreviation, supports sh, sz, hk.
  - `page`: Page number, with a maximum of 50 records per page.

The above is just an introduction to some of the tools. The complete tool list includes more modules designed for different needs. Each tool has detailed parameter descriptions and expected response structures to ensure that users can accurately call the API and parse the returned data.