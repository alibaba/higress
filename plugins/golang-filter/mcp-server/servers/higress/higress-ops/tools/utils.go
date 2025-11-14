package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// FormatJSONResponse formats a JSON response for better readability
func FormatJSONResponse(data []byte) (string, error) {
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		// If not valid JSON, return as-is
		return string(data), nil
	}

	formatted, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return string(data), nil
	}

	return string(formatted), nil
}

// CreateToolResult creates a standardized tool result with formatted content
func CreateToolResult(data []byte, contentType string) (*mcp.CallToolResult, error) {
	var content string
	var err error

	if contentType == "json" || strings.Contains(string(data), "{") {
		content, err = FormatJSONResponse(data)
		if err != nil {
			return nil, fmt.Errorf("failed to format JSON response: %w", err)
		}
	} else {
		content = string(data)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: content,
			},
		},
	}, nil
}

// CreateErrorResult creates an error result for tool calls
func CreateErrorResult(message string) (*mcp.CallToolResult, error) {
	return nil, fmt.Errorf(message)
}

// GetStringParam safely extracts a string parameter from arguments
func GetStringParam(arguments map[string]interface{}, key string, defaultValue string) string {
	if value, ok := arguments[key].(string); ok {
		return value
	}
	return defaultValue
}

// GetBoolParam safely extracts a boolean parameter from arguments
func GetBoolParam(arguments map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := arguments[key].(bool); ok {
		return value
	}
	return defaultValue
}

// ValidateRequiredParams validates that required parameters are present
func ValidateRequiredParams(arguments map[string]interface{}, requiredParams []string) error {
	for _, param := range requiredParams {
		if _, ok := arguments[param]; !ok {
			return fmt.Errorf("missing required parameter: %s", param)
		}
	}
	return nil
}

// CreateSimpleSchema creates a simple JSON schema for tools with no parameters
func CreateSimpleSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": [],
		"additionalProperties": false
	}`)
}

// CreateParameterSchema creates a JSON schema for tools with specific parameters
func CreateParameterSchema(properties map[string]interface{}, required []string) json.RawMessage {
	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}

	schemaBytes, _ := json.Marshal(schema)
	return json.RawMessage(schemaBytes)
}
