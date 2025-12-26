package common

// Himarket Product Type
type ProductType string

const (
	MCP_SERVER ProductType = "MCP_SERVER"
	MODEL_API  ProductType = "MODEL_API"
	REST_API   ProductType = "REST_API"
	AGENT_API  ProductType = "AGENT_API"
)
