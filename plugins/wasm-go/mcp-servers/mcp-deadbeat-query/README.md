# Deadbeat Inquiry

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00047480

# MCP Server Configuration Document

## Overview
This MCP server, named `deadbeat-query`, primarily serves as an external interface that allows users to query whether a person is a dishonest debtor by submitting basic personal information such as name, ID number, and mobile phone number. This service is particularly useful for financial institutions, credit departments, and other entities that need to perform risk control, helping these organizations quickly understand the credit status of potential customers and make more informed decisions.

## Tool Introduction
### Dishonest Debtor Information Inquiry
- **Purpose**: This tool is specifically designed to check if there are any records of a person being a dishonest debtor based on the provided personal information.
- **Use Cases**: It is applicable in various fields such as bank loan approvals, lease business reviews, and employment background checks, where it can effectively assess the creditworthiness of relevant individuals.

#### Input Parameters
- `idcard_number`: Required, represents the identification number of the individual being queried.
- `mobile_number`: Required, indicates the mobile phone number used by the individual being queried.
- `name`: Required, specifies the name of the person to be queried.

#### Request Example
- **URL**: `https://jumjokk.market.alicloudapi.com/personal/disenforcement`
- **Method**: POST
- **Headers**:
  - `Content-Type: application/x-www-form-urlencoded`
  - `Authorization: APPCODE <appCode value>` (Replace `<appCode value>` with the actual application code)
  - `X-Ca-Nonce: <randomly generated unique identifier>`

#### Response Structure
The response will include the following fields:
- `code`: An integer indicating the status code of the API call result.
- `data`: An object type, which contains specific details of the dishonesty records when data is returned.
  - `caseCount`: An integer showing the number of related cases found.
  - `caseList`: An array listing all the details of the related cases.
    - Each element includes information such as age (`age`), area name (`areaname`), business entity (`buesinessentity`), etc.
- `msg`: A string providing a message or error prompt regarding this query operation.
- `taskNo`: A string, a unique task number that can be used to track the processing of this request.

Please note that to ensure privacy, security, and legal compliance, please make sure you have obtained the appropriate authorization and comply with local laws and regulations before using this service.