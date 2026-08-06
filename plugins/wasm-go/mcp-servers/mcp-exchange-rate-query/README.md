# Exchange Rate Inquiry

The APP Code required for API authentication can be applied for at the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi011221

# MCP Server Function Overview Document

## Function Overview
This MCP server, named `exchange-rate-query`, primarily provides foreign exchange rate inquiry and conversion services. By integrating with the exchange rate API available on the Alibaba Cloud Marketplace, it offers users information including but not limited to the foreign exchange rates of the top ten banks, the exchange rate between a single currency and other currencies, all supported currency names, and specific currency conversions. This service is particularly suitable for application scenarios requiring real-time access to the latest exchange rate data, such as financial analysis software, international trade platforms, or personal finance assistants.

## Tool Introduction

### Foreign Exchange Rates of the Top Ten Banks
- **Purpose**: This tool allows users to query the foreign exchange rates provided by the top ten major banks in China.
- **Use Case**: It is useful for situations where one needs to understand the differences in foreign exchange prices among different banks, such as comparing fees and exchange rates before making an international transfer to choose the best option.
- **Parameter Description**:
  - `bank`: Specifies the bank code to be queried, defaulting to Bank of China (BOC).

### Single Currency Inquiry Interface
- **Purpose**: Used to obtain the current exchange rates of a specific currency relative to other multiple currencies and their last update time.
- **Use Case**: It is very useful when you need to check the value of a base currency against other currencies worldwide.
- **Parameter Description**:
  - `currency`: A required parameter that specifies the code of the base currency to be queried.

### All Currencies Inquiry Interface
- **Purpose**: Lists all supported currencies and their corresponding full names.
- **Use Case**: As one of the initialization steps when building applications involving multi-currency transactions to obtain a complete list of currencies.
- **Parameter Description**: No additional input parameters are needed.

### Exchange Rate Conversion Interface
- **Purpose**: Converts the value of one unit of currency into another based on a given amount.
- **Use Case**: It is ideal for applications that need to quickly calculate the value conversion between different currencies in cross-border transactions.
- **Parameter Description**:
  - `amount`: A required field indicating the amount to be converted.
  - `from`: A required field specifying the source currency code; if left blank, it defaults to Chinese Yuan (CNY) or US Dollar (USD).
  - `to`: A required field specifying the target currency code; similarly, if not specified, it defaults to CNY or USD as the output currency.