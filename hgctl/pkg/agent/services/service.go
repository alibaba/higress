package services

import (
	"fmt"
)

func HandleAddServiceSource(client *HigressClient, body interface{}) ([]byte, error) {
	data, ok := body.(map[string]interface{})
	fmt.Printf("request body: %v\n", data)
	if !ok {
		return nil, fmt.Errorf("failed to parse request body")
	}
	// Validate
	if _, ok := data["name"]; !ok {
		return nil, fmt.Errorf("missing required field 'name' in body")
	}
	if _, ok := data["type"]; !ok {
		return nil, fmt.Errorf("missing required field 'type' in body")
	}
	if _, ok := data["domain"]; !ok {
		return nil, fmt.Errorf("missing required field 'domain' in body")
	}
	if _, ok := data["port"]; !ok {
		return nil, fmt.Errorf("missing required field 'port' in body")
	}

	resp, err := client.Post("/v1/service-sources", data)
	if err != nil {
		return nil, fmt.Errorf("failed to add service source: %w", err)
	}
	// res := make(map[string]interface{})

	return resp, nil
}

// add MCP server to higress console, example request body as followed:
//
//	{
//	  "name": "mcp-deepwiki",
//	  "description": "",
//	  "type": "DIRECT_ROUTE",
//	  "service": "hgctl-deepwiki.dns:443",
//	  "upstreamPathPrefix": "/mcp",
//	  "services": [
//	    {
//	      "name": "hgctl-deepwiki.dns",
//	      "port": 443,
//	      "version": "1.0",
//	      "weight": 100
//	    }
//	  ]
//	}
func HandleAddMCPServer(client *HigressClient, body interface{}) ([]byte, error) {
	data, ok := body.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse request body")
	}
	// Validate
	if _, ok := data["name"]; !ok {
		return nil, fmt.Errorf("missing required field 'name' in body")
	}
	if _, ok := data["type"]; !ok {
		return nil, fmt.Errorf("missing required field 'type' in body")
	}
	if _, ok := data["service"]; !ok {
		return nil, fmt.Errorf("missing required field 'service' in body")
	}
	if _, ok := data["upstreamPathPrefix"]; !ok {
		return nil, fmt.Errorf("missing required field 'upstreamPathPrefix' in body")
	}

	_, ok = data["services"]
	if !ok {
		return nil, fmt.Errorf("missing required field 'port' in body")
	}

	resp, err := client.Put("/v1/mcpServer", data)
	if err != nil {
		return nil, fmt.Errorf("failed to add service source: %w", err)
	}

	return resp, nil
}
