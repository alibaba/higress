# Shebao Tools MCP Server

An implementation of the Model Context Protocol (MCP) server that integrates social security, housing provident fund, disability insurance, income tax, work injury compensation, and work death compensation calculation functions.

## Features

- Calculate social security and housing provident fund fees based on city information. Input the city name and salary information to get detailed calculation results.
- Calculate disability insurance based on enterprise scale. Input the number of employees and average salary of the enterprise to get the calculation result.
- Calculate income tax payment based on individual salary. Input the individual salary to get the payment amount.
- Calculate work injury compensation based on work injury situation. Input the work injury level and salary information to get the compensation amount.
- Calculate work death compensation based on work death situation. Input relevant information to get the compensation amount.

## Tutorial

### Configure API Key

In the `mcp-server.yaml` file, set the `apikey` field to a valid API key.

### Integrate into MCP Client

On the user's MCP Client interface, add the relevant configuration to the MCP Server list.

```json
"mcpServers": {
    "wolframalpha": {
      "url": "https://open-api.junrunrenli.com/agent/tools?jr-api-key={apikey}",
    }
}
