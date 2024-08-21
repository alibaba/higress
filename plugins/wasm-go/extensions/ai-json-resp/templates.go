package main

var ApiTemp = map[string]interface{}{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type":    "object",
	"properties": map[string]interface{}{
		"apiVersion": map[string]interface{}{
			"type":        "string",
			"description": "API version being used",
		},
		"request": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"endpoint": map[string]interface{}{
					"type":        "string",
					"description": "API endpoint being requested",
				},
				"port": map[string]interface{}{
					"type":        "integer",
					"description": "Port number to access the API",
					"minimum":     1,
					"maximum":     65535,
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "HTTP method to be used",
					"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
				},
				"headers": map[string]interface{}{
					"type":                 "object",
					"description":          "HTTP headers for the request",
					"additionalProperties": map[string]interface{}{"type": "string"},
				},
				"body": map[string]interface{}{
					"type":                 "object",
					"description":          "The JSON payload to be sent with the request",
					"additionalProperties": true,
				},
			},
			"required": []string{"endpoint", "port", "method"},
		},
	},
	"required":             []string{"apiVersion", "request"},
	"additionalProperties": false,
}
