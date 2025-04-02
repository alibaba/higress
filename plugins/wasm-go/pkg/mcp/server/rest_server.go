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
	Required    bool          `json:"required,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
}

// RestToolHeader represents an HTTP header
type RestToolHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RestToolRequestTemplate defines how to construct the HTTP request
type RestToolRequestTemplate struct {
	URL        string           `json:"url"`
	Method     string           `json:"method"`
	Headers    []RestToolHeader `json:"headers"`
	Body       string           `json:"body"`
	ArgsToBody bool             `json:"argsToBody,omitempty"`
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

	if err := json.Unmarshal(params, &newTool.arguments); err != nil {
		log.Warnf("Failed to parse tool arguments: %v", err)
	}

	// Apply default values for missing arguments
	for _, arg := range t.toolConfig.Args {
		if _, exists := newTool.arguments[arg.Name]; !exists && arg.Default != nil {
			newTool.arguments[arg.Name] = arg.Default
		}
	}

	return newTool
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
	url, err := executeTemplate(t.toolConfig.parsedURLTemplate, templateDataBytes)
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

	// Execute request body if needed
	var requestBody []byte
	var hasJsonContentType bool

	// Check if any header is Content-Type: application/json
	for _, header := range headers {
		if header[0] == "Content-Type" && (header[1] == "application/json" || header[1] == "application/json; charset=utf-8") {
			hasJsonContentType = true
			break
		}
	}

	if t.toolConfig.RequestTemplate.ArgsToBody {
		// Use args directly as the request body
		argsJson, err := json.Marshal(t.arguments)
		if err != nil {
			return fmt.Errorf("error marshaling args to JSON: %v", err)
		}
		requestBody = argsJson

		// Add JSON content type if not already present
		if !hasJsonContentType {
			headers = append(headers, [2]string{"Content-Type", "application/json; charset=utf-8"})
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
	ctx.RouteCall(t.toolConfig.RequestTemplate.Method, url, headers, requestBody,
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
			"type":        "string",
			"description": arg.Description,
		}

		// Add enum if specified
		if arg.Enum != nil && len(arg.Enum) > 0 {
			argSchema["enum"] = arg.Enum
		}

		// Add default if specified
		if arg.Default != nil {
			argSchema["default"] = arg.Default
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
