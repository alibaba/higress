// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	template "github.com/higress-group/gjson_template"
	"github.com/tidwall/sjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/utils"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

// RestToolArg represents an argument for a REST tool
type RestToolArg struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        string        `json:"type,omitempty"` // JSON Schema type: string, number, integer, boolean, array, object
	Required    bool          `json:"required,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	// For array type
	Items interface{} `json:"items,omitempty"`
	// For object type
	Properties interface{} `json:"properties,omitempty"`
}

// RestToolHeader represents an HTTP header
type RestToolHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RestToolRequestTemplate defines how to construct the HTTP request
type RestToolRequestTemplate struct {
	URL            string           `json:"url"`
	Method         string           `json:"method"`
	Headers        []RestToolHeader `json:"headers"`
	Body           string           `json:"body"`
	ArgsToJsonBody bool             `json:"argsToJsonBody,omitempty"` // Use args as JSON body
	ArgsToUrlParam bool             `json:"argsToUrlParam,omitempty"` // Add args to URL parameters
	ArgsToFormBody bool             `json:"argsToFormBody,omitempty"` // Use args as form-urlencoded body
}

// RestToolResponseTemplate defines how to transform the HTTP response
type RestToolResponseTemplate struct {
	Body string `json:"body"`
}

// RestTool represents a REST API that can be called as an MCP tool
type RestTool struct {
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	Args             []RestToolArg            `json:"args"`
	RequestTemplate  RestToolRequestTemplate  `json:"requestTemplate"`
	ResponseTemplate RestToolResponseTemplate `json:"responseTemplate"`

	// Parsed templates (not from JSON)
	parsedURLTemplate      *template.Template
	parsedHeaderTemplates  map[string]*template.Template
	parsedBodyTemplate     *template.Template
	parsedResponseTemplate *template.Template
}

// templateFuncs returns the template functions map
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
		// Add more helper functions as needed
	}
}

// parseTemplates parses all templates in the tool configuration
func (t *RestTool) parseTemplates() error {
	var err error

	// Validate args configuration - only one of the three options can be true
	argsOptionCount := 0
	if t.RequestTemplate.ArgsToJsonBody {
		argsOptionCount++
	}
	if t.RequestTemplate.ArgsToUrlParam {
		argsOptionCount++
	}
	if t.RequestTemplate.ArgsToFormBody {
		argsOptionCount++
	}
	if argsOptionCount > 1 {
		return fmt.Errorf("only one of argsToJsonBody, argsToUrlParam, or argsToFormBody can be set to true")
	}

	// Parse URL template
	t.parsedURLTemplate, err = template.New("url").Funcs(templateFuncs()).Parse(t.RequestTemplate.URL)
	if err != nil {
		return fmt.Errorf("error parsing URL template: %v", err)
	}

	// Parse header templates
	t.parsedHeaderTemplates = make(map[string]*template.Template)
	for i, header := range t.RequestTemplate.Headers {
		tmplName := fmt.Sprintf("header_%d", i)
		t.parsedHeaderTemplates[header.Key], err = template.New(tmplName).Funcs(templateFuncs()).Parse(header.Value)
		if err != nil {
			return fmt.Errorf("error parsing header template for %s: %v", header.Key, err)
		}
	}

	// Parse body template if present
	if t.RequestTemplate.Body != "" {
		t.parsedBodyTemplate, err = template.New("body").Funcs(templateFuncs()).Parse(t.RequestTemplate.Body)
		if err != nil {
			return fmt.Errorf("error parsing body template: %v", err)
		}
	}

	// Parse response template if present
	if t.ResponseTemplate.Body != "" {
		t.parsedResponseTemplate, err = template.New("response").Funcs(templateFuncs()).Parse(t.ResponseTemplate.Body)
		if err != nil {
			return fmt.Errorf("error parsing response template: %v", err)
		}
	}

	return nil
}

// executeTemplate executes a parsed template with the given data
func executeTemplate(tmpl *template.Template, data []byte) (string, error) {
	if tmpl == nil {
		return "", errors.New("template is nil")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// RestMCPServer implements Server interface for REST-to-MCP conversion
type RestMCPServer struct {
	base        BaseMCPServer
	toolsConfig map[string]RestTool // Store original tool configs for template rendering
}

// NewRestMCPServer creates a new REST-to-MCP server
func NewRestMCPServer() *RestMCPServer {
	return &RestMCPServer{
		base:        NewBaseMCPServer(),
		toolsConfig: make(map[string]RestTool),
	}
}

// AddMCPTool implements Server interface
func (s *RestMCPServer) AddMCPTool(name string, tool Tool) Server {
	s.base.AddMCPTool(name, tool)
	return s
}

// AddRestTool adds a REST tool configuration
func (s *RestMCPServer) AddRestTool(toolConfig RestTool) error {
	// Parse templates at configuration time
	if err := toolConfig.parseTemplates(); err != nil {
		return err
	}

	s.toolsConfig[toolConfig.Name] = toolConfig
	s.base.AddMCPTool(toolConfig.Name, &RestMCPTool{
		name:       toolConfig.Name,
		toolConfig: toolConfig,
	})

	return nil
}

// GetMCPTools implements Server interface
func (s *RestMCPServer) GetMCPTools() map[string]Tool {
	return s.base.GetMCPTools()
}

// SetConfig implements Server interface
func (s *RestMCPServer) SetConfig(config []byte) {
	s.base.SetConfig(config)
}

// GetConfig implements Server interface
func (s *RestMCPServer) GetConfig(v any) {
	s.base.GetConfig(v)
}

// Clone implements Server interface
func (s *RestMCPServer) Clone() Server {
	newServer := &RestMCPServer{
		base:        s.base.CloneBase(),
		toolsConfig: make(map[string]RestTool),
	}
	for k, v := range s.toolsConfig {
		newServer.toolsConfig[k] = v
	}
	return newServer
}

// GetToolConfig returns the REST tool configuration for a given tool name
func (s *RestMCPServer) GetToolConfig(name string) (RestTool, bool) {
	config, ok := s.toolsConfig[name]
	return config, ok
}

// RestMCPTool implements Tool interface for REST-to-MCP
type RestMCPTool struct {
	name       string
	toolConfig RestTool
	arguments  map[string]interface{}
}

// Create implements Tool interface
func (t *RestMCPTool) Create(params []byte) Tool {
	newTool := &RestMCPTool{
		name:       t.name,
		toolConfig: t.toolConfig,
		arguments:  make(map[string]interface{}),
	}

	// Parse raw arguments
	var rawArgs map[string]interface{}
	if err := json.Unmarshal(params, &rawArgs); err != nil {
		log.Warnf("Failed to parse tool arguments: %v", err)
	}

	// Process arguments with type conversion
	for _, arg := range t.toolConfig.Args {
		// Check if argument was provided
		rawValue, exists := rawArgs[arg.Name]
		if !exists {
			// Apply default if available
			if arg.Default != nil {
				newTool.arguments[arg.Name] = arg.Default
			}
			continue
		}

		// Convert value based on type
		switch arg.Type {
		case "boolean":
			// Convert to boolean
			switch v := rawValue.(type) {
			case bool:
				newTool.arguments[arg.Name] = v
			case string:
				if v == "true" {
					newTool.arguments[arg.Name] = true
				} else if v == "false" {
					newTool.arguments[arg.Name] = false
				} else {
					newTool.arguments[arg.Name] = rawValue
				}
			default:
				newTool.arguments[arg.Name] = rawValue
			}
		case "integer":
			// Convert to integer
			switch v := rawValue.(type) {
			case float64:
				newTool.arguments[arg.Name] = int(v)
			case string:
				if intVal, err := json.Number(v).Int64(); err == nil {
					newTool.arguments[arg.Name] = int(intVal)
				} else {
					newTool.arguments[arg.Name] = rawValue
				}
			default:
				newTool.arguments[arg.Name] = rawValue
			}
		case "number":
			// Convert to number (float64)
			switch v := rawValue.(type) {
			case string:
				if floatVal, err := json.Number(v).Float64(); err == nil {
					newTool.arguments[arg.Name] = floatVal
				} else {
					newTool.arguments[arg.Name] = rawValue
				}
			default:
				newTool.arguments[arg.Name] = rawValue
			}
		default:
			// For string, array, object, or unspecified types, use as is
			newTool.arguments[arg.Name] = rawValue
		}
	}

	return newTool
}

// convertArgToString converts an argument value to a string representation
func convertArgToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool, int, int64, float64:
		return fmt.Sprintf("%v", v)
	default:
		// For complex types, try to marshal to JSON
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	}
}

// hasContentType checks if the headers contain a specific content type
func hasContentType(headers [][2]string, contentTypeSubstr string) bool {
	for _, header := range headers {
		if strings.EqualFold(header[0], "Content-Type") && strings.Contains(strings.ToLower(header[1]), contentTypeSubstr) {
			return true
		}
	}
	return false
}

// Call implements Tool interface
func (t *RestMCPTool) Call(httpCtx HttpContext, server Server) error {
	ctx := httpCtx.(wrapper.HttpContext)

	// Get server config
	var config map[string]interface{}
	server.GetConfig(&config)

	var templateDataBytes []byte
	templateDataBytes, _ = sjson.SetBytes(templateDataBytes, "config", config)
	templateDataBytes, _ = sjson.SetBytes(templateDataBytes, "args", t.arguments)

	// Execute URL template
	urlStr, err := executeTemplate(t.toolConfig.parsedURLTemplate, templateDataBytes)
	if err != nil {
		return fmt.Errorf("error executing URL template: %v", err)
	}

	// Execute headers
	headers := make([][2]string, 0, len(t.toolConfig.RequestTemplate.Headers))
	for _, header := range t.toolConfig.RequestTemplate.Headers {
		tmpl, ok := t.toolConfig.parsedHeaderTemplates[header.Key]
		if !ok {
			return fmt.Errorf("header template not found for %s", header.Key)
		}

		value, err := executeTemplate(tmpl, templateDataBytes)
		if err != nil {
			return fmt.Errorf("error executing header template: %v", err)
		}
		headers = append(headers, [2]string{header.Key, value})
	}

	// Check for existing content types
	hasJsonContentType := hasContentType(headers, "application/json")
	hasFormContentType := hasContentType(headers, "application/x-www-form-urlencoded")

	// Process URL parameters if argsToUrlParam is true
	if t.toolConfig.RequestTemplate.ArgsToUrlParam {
		// Parse the URL to add parameters
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return fmt.Errorf("error parsing URL: %v", err)
		}

		// Get existing query values
		query := parsedURL.Query()

		// Add arguments to query parameters
		for key, value := range t.arguments {
			query.Set(key, convertArgToString(value))
		}

		// Update the URL with the new query string
		parsedURL.RawQuery = query.Encode()
		urlStr = parsedURL.String()
	}

	// Prepare request body
	var requestBody []byte

	if t.toolConfig.RequestTemplate.ArgsToJsonBody {
		// Use args directly as JSON in the request body
		argsJson, err := json.Marshal(t.arguments)
		if err != nil {
			return fmt.Errorf("error marshaling args to JSON: %v", err)
		}
		requestBody = argsJson

		// Add JSON content type if not already present
		if !hasJsonContentType {
			headers = append(headers, [2]string{"Content-Type", "application/json; charset=utf-8"})
		}
	} else if t.toolConfig.RequestTemplate.ArgsToFormBody {
		// Use args as form-urlencoded body
		formValues := url.Values{}
		for key, value := range t.arguments {
			formValues.Set(key, convertArgToString(value))
		}

		requestBody = []byte(formValues.Encode())

		// Add form content type if not already present
		if !hasFormContentType {
			headers = append(headers, [2]string{"Content-Type", "application/x-www-form-urlencoded"})
		}
	} else if t.toolConfig.parsedBodyTemplate != nil {
		body, err := executeTemplate(t.toolConfig.parsedBodyTemplate, templateDataBytes)
		if err != nil {
			return fmt.Errorf("error executing body template: %v", err)
		}
		requestBody = []byte(body)

		// Check if body is JSON and add content type if needed
		trimmedBody := bytes.TrimSpace(requestBody)
		if !hasJsonContentType && len(trimmedBody) > 0 &&
			((trimmedBody[0] == '{' && trimmedBody[len(trimmedBody)-1] == '}') ||
				(trimmedBody[0] == '[' && trimmedBody[len(trimmedBody)-1] == ']')) {
			// Try to parse as JSON to confirm
			var js interface{}
			if json.Unmarshal(trimmedBody, &js) == nil {
				headers = append(headers, [2]string{"Content-Type", "application/json; charset=utf-8"})
			}
		}
	}

	// Make HTTP request
	ctx.RouteCall(t.toolConfig.RequestTemplate.Method, urlStr, headers, requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("call failed, status: %d", statusCode))
				return
			}

			// Execute response template
			if t.toolConfig.parsedResponseTemplate != nil {
				result, err := executeTemplate(t.toolConfig.parsedResponseTemplate, responseBody)
				if err != nil {
					utils.OnMCPToolCallError(ctx, fmt.Errorf("error executing response template: %v", err))
					return
				}
				utils.SendMCPToolTextResult(ctx, result)
			} else {
				// Just return raw response as JSON string
				utils.SendMCPToolTextResult(ctx, string(responseBody))
			}
		})

	return nil
}

// Description implements Tool interface
func (t *RestMCPTool) Description() string {
	return t.toolConfig.Description
}

// InputSchema implements Tool interface
func (t *RestMCPTool) InputSchema() map[string]any {
	// Convert tool args to JSON schema
	properties := make(map[string]interface{})
	required := []string{}

	for _, arg := range t.toolConfig.Args {
		argSchema := map[string]interface{}{
			"description": arg.Description,
		}

		// Set type (default to string if not specified)
		argType := arg.Type
		if argType == "" {
			argType = "string"
		}
		argSchema["type"] = argType

		// Add enum if specified
		if arg.Enum != nil && len(arg.Enum) > 0 {
			argSchema["enum"] = arg.Enum
		}

		// Add default if specified
		if arg.Default != nil {
			argSchema["default"] = arg.Default
		}

		// Add items for array type
		if argType == "array" && arg.Items != nil {
			argSchema["items"] = arg.Items
		}

		// Add properties for object type
		if argType == "object" && arg.Properties != nil {
			argSchema["properties"] = arg.Properties
		}

		properties[arg.Name] = argSchema

		// Add to required list if needed
		if arg.Required {
			required = append(required, arg.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	// Add required field only if there are required properties
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}
