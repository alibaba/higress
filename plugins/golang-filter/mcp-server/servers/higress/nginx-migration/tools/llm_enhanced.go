// Package tools provides LLM-enhanced nginx configuration migration tools for Higress
// Added on 2025.9.29 - LLM enhanced migration tools implementation
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/mark3labs/mcp-go/mcp"
)

// GenerateCompletion sends a request to OpenAI API and returns the response
func (c *OpenAIClient) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	request := OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert in Nginx configuration migration to Higress. Provide accurate, well-structured responses for configuration conversion tasks.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1, // Low temperature for consistent, factual responses
		MaxTokens:   2000,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenAI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// RegisterLLMEnhancedMigrationTools registers LLM-enhanced nginx migration tools
func RegisterLLMEnhancedMigrationTools(mcpServer *common.MCPServer, client *higress.HigressClient, llmClient *OpenAIClient) {
	// LLM-enhanced Nginx configuration analysis
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("llm_analyze_nginx_config", "Use LLM to analyze complex Nginx configurations and provide migration insights", getLLMAnalyzeNginxConfigSchema()),
		handleLLMAnalyzeNginxConfig(llmClient),
	)

	// LLM-enhanced configuration conversion
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("llm_convert_complex_config", "Use LLM to convert complex Nginx configurations that standard parsers cannot handle", getLLMConvertComplexConfigSchema()),
		handleLLMConvertComplexConfig(llmClient),
	)

	// LLM-enhanced plugin migration assistance
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("llm_assist_plugin_migration", "Use LLM to assist in migrating complex Nginx Lua plugins to Higress WASM", getLLMAssistPluginMigrationSchema()),
		handleLLMAssistPluginMigration(llmClient),
	)

	// LLM-enhanced migration validation
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("llm_validate_migration", "Use LLM to validate and suggest improvements for migration results", getLLMValidateMigrationSchema()),
		handleLLMValidateMigration(llmClient),
	)

	api.LogInfof("LLM-enhanced migration tools registered successfully")
}

// Handler functions for LLM-enhanced tools
func handleLLMAnalyzeNginxConfig(llmClient *OpenAIClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		configContent, ok := arguments["config_content"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'config_content' argument")
		}

		prompt := fmt.Sprintf(`Analyze this Nginx configuration and provide detailed insights for migration to Higress:

Configuration:
%s

Please provide:
1. Summary of server blocks and their purposes
2. Complex directives that may need special handling
3. Potential migration challenges
4. Recommended migration approach
5. Higress-equivalent configurations

Format your response as structured JSON with clear sections.`, configContent)

		response, err := llmClient.GenerateCompletion(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("LLM analysis failed: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("LLM Analysis Results:\n%s", response),
				},
			},
		}, nil
	}
}

func handleLLMConvertComplexConfig(llmClient *OpenAIClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		configContent, ok := arguments["config_content"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'config_content' argument")
		}

		namespace := "default"
		if ns, ok := arguments["namespace"].(string); ok {
			namespace = ns
		}

		prompt := fmt.Sprintf(`Convert this complex Nginx configuration to Higress format:

Nginx Configuration:
%s

Target Namespace: %s

Please provide:
1. Equivalent Higress route configurations
2. Required Kubernetes YAML manifests
3. Any additional Higress plugins needed
4. Configuration notes and warnings

Generate valid Kubernetes YAML that can be directly applied.`, configContent, namespace)

		response, err := llmClient.GenerateCompletion(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("LLM conversion failed: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("LLM Complex Configuration Conversion:\n%s", response),
				},
			},
		}, nil
	}
}

func handleLLMAssistPluginMigration(llmClient *OpenAIClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		luaCode, ok := arguments["lua_code"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'lua_code' argument")
		}

		targetLanguage := "rust"
		if lang, ok := arguments["target_language"].(string); ok {
			targetLanguage = lang
		}

		prompt := fmt.Sprintf(`Help migrate this Nginx Lua plugin to Higress WASM format:

Lua Code:
%s

Target Language: %s

Please provide:
1. Analysis of the Lua plugin functionality
2. Equivalent WASM plugin structure in %s
3. Configuration mapping
4. Implementation guidance
5. Potential compatibility issues

Provide practical, implementable code examples.`, luaCode, targetLanguage, targetLanguage)

		response, err := llmClient.GenerateCompletion(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("LLM plugin migration assistance failed: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("LLM Plugin Migration Assistance:\n%s", response),
				},
			},
		}, nil
	}
}

func handleLLMValidateMigration(llmClient *OpenAIClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		originalConfig, ok := arguments["original_config"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'original_config' argument")
		}

		migratedConfig, ok := arguments["migrated_config"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'migrated_config' argument")
		}

		prompt := fmt.Sprintf(`Validate this Nginx to Higress migration:

Original Nginx Configuration:
%s

Migrated Higress Configuration:
%s

Please provide:
1. Completeness assessment (what's covered, what's missing)
2. Functional equivalence analysis
3. Performance considerations
4. Security implications
5. Recommended improvements
6. Migration quality score (1-10)

Provide detailed feedback for production readiness.`, originalConfig, migratedConfig)

		response, err := llmClient.GenerateCompletion(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("LLM migration validation failed: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("LLM Migration Validation:\n%s", response),
				},
			},
		}, nil
	}
}

// Schema functions for LLM-enhanced tools
func getLLMAnalyzeNginxConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"config_content": {
				"type": "string",
				"description": "The Nginx configuration content to analyze"
			}
		},
		"required": ["config_content"],
		"additionalProperties": false
	}`)
}

func getLLMConvertComplexConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"config_content": {
				"type": "string",
				"description": "The complex Nginx configuration to convert"
			},
			"namespace": {
				"type": "string",
				"description": "Kubernetes namespace for the converted configuration",
				"default": "default"
			}
		},
		"required": ["config_content"],
		"additionalProperties": false
	}`)
}

func getLLMAssistPluginMigrationSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"lua_code": {
				"type": "string",
				"description": "The Nginx Lua plugin code to migrate"
			},
			"target_language": {
				"type": "string",
				"enum": ["rust", "go", "cpp", "assemblyscript"],
				"description": "Target language for WASM plugin",
				"default": "rust"
			}
		},
		"required": ["lua_code"],
		"additionalProperties": false
	}`)
}

func getLLMValidateMigrationSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"original_config": {
				"type": "string",
				"description": "Original Nginx configuration"
			},
			"migrated_config": {
				"type": "string",
				"description": "Migrated Higress configuration"
			}
		},
		"required": ["original_config", "migrated_config"],
		"additionalProperties": false
	}`)
}
