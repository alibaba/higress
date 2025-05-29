# Bid Tools MCP Server

A tool for querying bidding information data across the network. It provides high-quality bidding information daily through RPA+(AI + manual backup) quality inspection. The bidding digital human of Junrun Technology is an agent service that helps enterprises obtain high-quality bidding information every day. Our vision is "Winning more with fewer bids, achieving mutual benefits between enterprises and bids", aiming to maximize the winning rate of enterprises by pushing only the most suitable bidding information for your enterprise.

## Data Source Coverage

| 站点类型         | 站点数量（10万+ ） | 采集源地址数量（29万+ ） | 信息发布量占比 |
|------------------|----------|------------|----------------|
| 企业招标         | -        |-           | 13.1183%       |
| 电子采购平台     | -        | -          | 33.0614%       |
| 金融银行         | -        | -          | 0.0757%        |
| 高等教育         | -        | -          | 0.9645%        |
| 教育局(网)       | -        | -          | 0.0173%        |
| 医院             | -        | -          | 0.8030%        |
| 其他卫生机构     | -        | -          | 0.0167%        |
| 人民政府         | -        | -          | 8.6457%        |
| 公共资源中心     | -        | -          | 11.6248%       |
| 政府采购中心     | -        | -          | 27.8139%       |
| 工程交易中心     | -        | -          | 3.5826%        |
| 行政服务中心     | -        | -          | 0.2761%        |


- **Data Source Collection Process**:
Over 100,000 first-release information source stations, over 290,000 collection channel addresses, and over 10,000 new media official accounts sources (continuously updated). More than 1,000 new active first-release information sources are added every month, and about 51,000 first-release information sources have published bidding and procurement data in the past year.

- **Data Quality Assurance**:
There is a large amount of duplicate information in the first-release information source data, with a duplication rate of about 40%. The intelligent recognition accuracy of element items is about 90%. Therefore, manual assistance is essential. The team uses AI intelligent recognition + manual processing to reduce the duplication rate to about 3% (while competitors are in the range of 8%-15%). At the same time, the data is cleaned to ensure its accuracy and completeness. The team maintains a manual team of over a hundred people for the collection and quality inspection of bidding and procurement information, ensuring the rapid resolution of source issues, reducing the information duplication rate, and verifying the extract data elements to maintain high-level information processing.

- **Data Update Frequency**:
Data is crawled in real-time and updated dynamically every day to ensure its timeliness and accuracy.

## Features

- `getBidlist`: Query bidding information across the network based on keywords. Input keywords and return the query results.
- `getBidinfo`: Query the details of bidding information based on the bidding ID. Input the bidding ID and return the details of the bidding information.
- `Email`: An offline-configured enterprise bidding matching email notification service. It creates a unique enterprise profile based on the enterprise's industry, business scope, and keyword groups. Then, it uses the Qwen3 large model of Alibaba's Bailian platform to deeply analyze daily bidding information and provide accurate bidding matching services for enterprises.

## Tutorial

### Get API Key
1. Register an account [Create an ID](https://moonai-bid.junrunrenli.com/#/login?src=higress)
2. Send an email to: yuanpeng@junrunrenli.com with the subject: MCP and the content: Apply for the bidding tool service, and provide your account.

### Knowledge Base
1. The clearer the enterprise's exclusive profile, the better the matching effect. Through continuous optimization, our knowledge base will be continuously improved to provide more accurate bidding matching services for enterprises.

### Future Plans (Stay Tuned)
1. Competitor Winning Analysis Service. Analyze the winning situations of competitors and provide more accurate winning decision support for enterprises.
2. Customer Bidding Analysis Service. Analyze the historical bidding situations of customers and provide more accurate bidding information support for enterprises to prepare for winning bids in advance.

### Configure API Key
In the `mcp-server.yaml` file, set the `jr-api-key` field to a valid API key.

### Integrate into MCP Client
On the user's MCP Client interface, add the relevant configuration to the MCP Server list.

```json
"mcpServers": {
    "jr-agent-bid": {
      "url": "https://agent-bid.junrunrenli.com/sse?jr-api-key={jr-api-key}",
    }
}```