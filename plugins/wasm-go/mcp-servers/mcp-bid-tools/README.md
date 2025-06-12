# Bid Tools MCP Server

## What is Junrun Bidding Digital Employee?
- Bidding is an important channel for enterprises to obtain projects and customers. Junrun Bidding Digital Employee can provide comprehensive and accurate bidding information, helping enterprises effectively improve their bidding capabilities.

## What are the problems with traditional bidding information acquisition channels?
- Obtaining bidding information through bidding platforms was a commonly used method in the past. However, there are many sources of bidding information, and it is difficult for general bidding platforms to cover all of them. There is a large amount of duplication in the bidding information from different platforms, making it difficult to deduplicate and filter.
- Traditional bidding information screening only uses keyword filtering, which is not very accurate. A large amount of bidding information still needs to be screened by experienced personnel, resulting in high labor costs.

## What can Junrun Bidding Digital Employee do?
- According to the subscribed bidding information, it pushes accurate bidding information filtered by AI via email every day. You can view the title, details, and download attachments of the bidding information.
- Based on AI capabilities, it analyzes bidding documents and automatically extracts key information, greatly improving the efficiency of bidding decision-making.
- Based on historical bidding information, it provides future bidding predictions to help enterprises plan in advance.
- Analyzes the historical bidding and winning situations of customers to support the formulation of bidding strategies.
- Analyzes the historical bidding and winning situations of competitors to obtain a list of target customers.

## Why choose Junrun Bidding Digital Employee?
### More comprehensive bidding data
- It cooperates with multiple source data enterprises, covering government bidding information, third - party bidding information, and enterprise self - owned website bidding information. In total, it covers more than 100,000 first - release information source stations, more than 290,000 collection channel addresses, and more than 10,000 new media official accounts (continuously updated). Tens of thousands of bidding and procurement information are updated daily, and the coverage rate of publicly available bidding and procurement information across the network can reach over 98% (the remaining 2% is due to new resources and the restructuring of existing resources).
### More accurate screening results
- Based on AI large - scale models, data training, and matching technology, the accuracy is significantly improved compared to traditional keyword search and rule - based judgment.
### Higher bidding decision - making efficiency
- Analysis of historical winning situations and one - click analysis of bidding documents to extract key information effectively assist in bidding decision - making.

## Data Source Coverage

| Site Type         | Number of Sites (100,000+) | Number of Collection Source Addresses (290,000+) | Information Release Volume Proportion |
|------------------|----------|------------|----------------|
| Enterprise Bidding         | -        |-           | 13.1183%       |
| E - Procurement Platform     | -        | -          | 33.0614%       |
| Financial Banks         | -        | -          | 0.0757%        |
| Higher Education         | -        | -          | 0.9645%        |
| Education Bureau (Website)       | -        | -          | 0.0173%        |
| Hospitals             | -        | -          | 0.8030%        |
| Other Healthcare Institutions     | -        | -          | 0.0167%        |
| People's Government         | -        | -          | 8.6457%        |
| Public Resource Centers     | -        | -          | 11.6248%       |
| Government Procurement Centers     | -        | -          | 27.8139%       |
| Project Trading Centers     | -        | -          | 3.5826%        |
| Administrative Service Centers     | -        | -          | 0.2761%        |

## Features

- `getBidlist`: Query bidding information across the network based on keywords. Input multiple keywords and return the bidding query results.
- `getBidinfo`: Query the details of bidding information based on the bidding ID. Input the bidding ID and return the details of the bidding information.
- `Email`: An offline - configured enterprise bidding matching email notification service. It combines the enterprise's industry, business scope, keyword groups, etc. to create a unique enterprise profile. Then, it uses the Qwen3 large model of Alibaba's Bailian platform to deeply analyze daily bidding information and provide accurate bidding matching services for enterprises.

## Tutorial

### Get API Key
1. Register an account [Create an ID](https://moonai-bid.junrunrenli.com?src=higress)
2. Send an email to: yuanpeng@junrunrenli.com with the subject: MCP and the content: Apply for the bidding tool service, and provide your account.

### Knowledge Base (Alibaba Bailian Knowledge Base)
1. The clearer the enterprise's exclusive profile, the better the matching effect. Through continuous optimization, our knowledge base will be continuously improved to provide more accurate bidding matching services for enterprises.

### Configure API Key
In the `mcp - server.yaml` file, set the `jr - api - key` field to a valid API key.

### Integrate into MCP Client
On the user's MCP Client interface, add the relevant configuration to the MCP Server list.
