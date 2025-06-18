# Sector Market Overview

Integrates the latest real-time market data for industry and concept sectors, including detailed information on constituent stocks. It covers key market indicators such as index level, price change percentage, trading volume, total market capitalization, sector rankings, and leading stocks. Designed for intelligent investment research and market trend tracking, it provides comprehensive insights into sector dynamics and constituent stock performance.

## Tool Overview
### Real-Time Daily Market Data for an Industry get_industry_realtime_quote
Enter an industry code to obtain the latest real-time data for that industry, including index level, price change percentage, trading volume, total market capitalization, number of constituent stocks, counts of stocks that hit limit up / rose / fell / remained flat / total stocks, sector ranking by performance, and leading stock information. This is used for real-time tracking of overall industry performance.


### Real-Time Daily Market Data for an Industry and Its Constituent Stocks get_industry_stock_realtime_quote
Enter an industry code to obtain the latest overall real-time market data for that industry—including index level, price change percentage, trading volume, total market capitalization, number of constituent stocks, performance ranking, and leading stock—along with detailed real-time data for all related constituent stocks, such as stock code, name, opening price, current price, price change percentage, and highest/lowest prices. This enables comprehensive analysis of the industry’s and its constituents’ latest market performance.


### Real-Time Daily Market Data for a Concept get_concept_realtime_quote
Enter a concept type (e.g., Juyuan, CLS) and concept code to retrieve the latest real-time market data for the concept sector, including sector name, price change percentage, one-week change, total market capitalization, number of constituent stocks, counts of stocks that hit limit up / rose / fell / remained flat, performance ranking, and leading stock information. This enables efficient tracking of trending market concept sectors.

### Real-Time Daily Market Data for a Concept and Its Constituent Stocks get_concept_stock_realtime_quote
Retrieve the latest real-time market data for a specified concept sector, including concept name, price change percentage, one-week change, total market capitalization, performance ranking, number of constituent stocks, counts of stocks that hit limit up / rose / fell / remained flat, and leading stock information. It also provides real-time data for all constituent stocks under the concept, such as stock code, name, market, opening price, current price, price change percentage, and highest/lowest prices—enabling quick insight into both the overall concept and its constituent stocks’ latest market performance.

## Usage Guide
### Apply for an APP Code
Visit the [Investoday Data Marketplace](https://data-api.investoday.net/mcp) to apply for your AppCode.

### Configuration Examples
```json
"mcpServers": {
    "industry-quote": {
      "url": "https://mcp.higress.ai/mcp-plate-quote/{appCode}/sse"
    }
}
```

