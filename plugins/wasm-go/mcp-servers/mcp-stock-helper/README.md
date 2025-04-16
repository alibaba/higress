# Stock Assistant

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00065924

# Overview

## Function Overview

The `stock-helper` server is a multifunctional API service designed for the stock, futures, and foreign exchange markets. It provides various tools, including candlestick charts, quotes, rankings, and more, to help users obtain real-time and historical data, perform technical analysis, and support decision-making. Through these tools, users can easily access information related to A-shares, H-shares, U.S. stocks, global indices, domestic and international futures, and the foreign exchange market.

## Tool Introduction

### A-Share Candlestick Charts
- **Purpose**: Provides candlestick data for A-shares at different time intervals (e.g., 1 minute, 5 minutes, daily).
- **Use Cases**: Technical analysis, trading strategy formulation, historical data backtesting, etc.
- **Parameters**:
  - `limit`: Number of records to return, default is 10.
  - `ma`: Moving average lines to return, optional values are 5, 10, 15, 20, 25, 30.
  - `symbol`: Security code, e.g., sh688193.
  - `type`: Candlestick type, such as 1 minute, 5 minutes, daily, etc.

### A-Share Adjusted Candlestick Charts
- **Purpose**: Provides adjusted candlestick data for A-shares.
- **Use Cases**: Long-term investment analysis, fundamental analysis, etc.
- **Parameters**:
  - `fuquan`: Adjustment status, 0 for no adjustment, 1 for forward adjustment, 2 for backward adjustment.
  - `limit`: Number of records to return, default is 10.
  - `symbol`: Security code, e.g., sh688193.
  - `type`: Candlestick type, such as 1 minute, 5 minutes, daily, etc.

### A-Share Quotes
- **Purpose**: Provides real-time quote information for A-shares.
- **Use Cases**: Real-time monitoring, quick trading decisions, etc.
- **Parameters**:
  - `symbol`: Security codes, separated by commas, e.g., sz000002,bj430047.

### A-Share Rankings
- **Purpose**: Provides rankings of A-shares based on specific conditions (e.g., price change rate, trading volume, etc.).
- **Use Cases**: Identifying hot stocks, market trend analysis, etc.
- **Parameters**:
  - `asc`: Sorting order, 0 for descending (from high to low), 1 for ascending (from low to high), default is 0.
  - `limit`: Number of records per page, maximum is 100, default is 10.
  - `market`: Market code, such as Shanghai and Shenzhen A-shares, GEM, etc.
  - `page`: Page number, default is 1.
  - `sort`: Sorting field, such as price change rate, trading volume, etc.

### Global Index Candlestick Charts
- **Purpose**: Provides candlestick data for global indices at different time intervals (e.g., daily, weekly, monthly).
- **Use Cases**: Global market analysis, asset allocation, etc.
- **Parameters**:
  - `limit`: Number of records to return, default is 10.
  - `symbol`: Index security code, see the code table for details.
  - `type`: Candlestick type, such as daily, weekly, monthly, etc.

### Global Index Quotes
- **Purpose**: Provides real-time quote information for global indices.
- **Use Cases**: Real-time monitoring, quick trading decisions, etc.
- **Parameters**:
  - `symbol`: Index security code, see the code table for details.

### Domestic Futures Candlestick Charts
- **Purpose**: Provides candlestick data for domestic futures at different time intervals (e.g., 1 minute, 5 minutes, daily).
- **Use Cases**: Futures market analysis, trading strategy formulation, etc.
- **Parameters**:
  - `limit`: Number of records to return, default is 10.
  - `symbol`: Futures security code, see the code table for details.
  - `type`: Candlestick type, such as 1 minute, 5 minutes, daily, etc.

### Domestic Futures Contracts
- **Purpose**: Provides information about domestic futures contracts.
- **Use Cases**: Contract selection, risk management, etc.
- **Parameters**:
  - `symbol`: Futures security code, see the code table for details.

### Domestic Futures Quotes
- **Purpose**: Provides real-time quote information for domestic futures.
- **Use Cases**: Real-time monitoring, quick trading decisions, etc.
- **Parameters**:
  - `symbol`: Futures security code, see the code table for details.

### Foreign Exchange Candlestick Charts
- **Purpose**: Provides candlestick data for foreign exchange at different time intervals (e.g., 1 minute, 5 minutes, daily).
- **Use Cases**: Foreign exchange market analysis, trading strategy formulation, etc.
- **Parameters**:
  - `limit`: Number of records to return, default is 10.
  - `symbol`: Security code, such as FXINDEX, see the code table for details.
  - `type`: Candlestick type, such as 1 minute, 5 minutes, daily, etc.

### Foreign Exchange Quotes
- **Purpose**: Provides real-time quote information for foreign exchange.
- **Use Cases**: Real-time monitoring, quick trading decisions, etc.
- **Parameters**:
  - `symbol`: Security code, such as FXINDEX,CNYRUB, see the code table for details.

### International Futures Candlestick Charts
- **Purpose**: Provides candlestick data for international futures at different time intervals (e.g., 1 minute, 5 minutes, daily).
- **Use Cases**: Futures market analysis, trading strategy formulation, etc.
- **Parameters**:
  - `limit`: Number of records to return, default is 10.
  - `symbol`: Futures security code, see the code table for details.
  - `type`: Candlestick type, such as 1 minute, 5 minutes, daily, etc.

### International Futures Contracts
- **Purpose**: Provides information about international futures contracts.
- **Use Cases**: Contract selection, risk management, etc.
- **Parameters**:
  - `symbol`: Futures security code, see the code table for details.

### International Futures Quotes
- **Purpose**: Provides real-time quote information for international futures.
- **Use Cases**: Real-time monitoring, quick trading decisions, etc.
- **Parameters**:
  - `symbol`: Futures security code, see the code table for details.

### H-Share Candlestick Charts
- **Purpose**: Provides candlestick data for H-shares at different time intervals (e.g., 1 minute, 5 minutes, daily).
- **Use Cases**: H-share market analysis, trading strategy formulation, etc.
- **Parameters**:
  - `limit`: Number of records to return, default is 10.
  - `symbol`: Security code, such as 08026.
  - `type`: Candlestick type, such as 1 minute, 5 minutes, daily, etc.

### H-Share Quotes
- **Purpose**: Provides real-time quote information for H-shares.
- **Use Cases**: Real-time monitoring, quick trading decisions, etc.
- **Parameters**:
  - `symbol`: Security codes, separated by commas, e.g., 08026,02203.

### H-Share Rankings
- **Purpose**: Provides rankings of H-shares based on specific conditions (e.g., price change rate, trading volume, etc.).
- **Use Cases**: Identifying hot stocks, market trend analysis, etc.
- **Parameters**:
  - `asc`: Sorting order, 0 for descending (from high to low), 1 for ascending (from low to high), default is 0.
  - `limit`: Number of records per page, maximum is 100, default is 10.
  - `page`: Page number, default is 1.
  - `sort`: Sorting field, such as price change rate, trading volume, etc.

### U.S. Stock Candlestick Charts
- **Purpose**: Provides candlestick data for U.S. stocks at different time intervals (e.g., 1 minute, 5 minutes, daily).
- **Use Cases**: U.S. stock market analysis, trading strategy formulation, etc.
- **Parameters**:
  - `limit`: Number of records to return, default is 10.
  - `symbol`: Security code.
  - `type`: Candlestick type, such as 1 minute, 5 minutes, daily, etc.

### U.S. Stock Information
- **Purpose**: Provides information about U.S. stocks.
- **Use Cases**: Stock selection, risk management, etc.
- **Parameters**:
  - `market`: Market, such as NYSE, NASDAQ.

### U.S. Stock Quotes
- **Purpose**: Provides real-time quote information for U.S. stocks.
- **Use Cases**: Real-time monitoring, quick trading decisions, etc.
- **Parameters**:
  - `symbol`: Security codes, separated by commas, e.g., INTC,AAPL.

### U.S. Stock Rankings
- **Purpose**: Provides rankings of U.S. stocks based on specific conditions (e.g., price change rate, trading volume, etc.).
- **Use Cases**: Identifying hot stocks, market trend analysis, etc.
- **Parameters**:
  - `asc`: Sorting order, 0 for descending (from high to low), 1 for ascending (from low to high), default is 0.
  - `limit`: Number of records per page, maximum is 100, default is 10.
  - `market`: Market code, such as all U.S. stocks, tech stocks, Chinese concept stocks, etc.
  - `page`: Page number, default is 1.
  - `sort`: Sorting field, such as price change rate, trading volume, etc.