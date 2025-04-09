package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// MCPConfigHandler handles configuration requests for MCP server
type MCPConfigHandler struct {
	configStore ConfigStore
	callbacks   api.FilterCallbackHandler
}

// NewMCPConfigHandler creates a new instance of MCP configuration handler
func NewMCPConfigHandler(redisClient *internal.RedisClient, callbacks api.FilterCallbackHandler) *MCPConfigHandler {
	return &MCPConfigHandler{
		configStore: NewRedisConfigStore(redisClient),
		callbacks:   callbacks,
	}
}

// HandleConfigRequest processes configuration requests
func (h *MCPConfigHandler) HandleConfigRequest(path string, method string, body []byte) bool {
	// Check if it's a configuration request
	if !strings.HasSuffix(path, "/config") {
		return false
	}

	// Extract serverName and uid from path
	pathParts := strings.Split(strings.TrimSuffix(path, "/config"), "/")
	if len(pathParts) < 2 {
		h.sendErrorResponse(http.StatusBadRequest, "INVALID_PATH", "Invalid path format")
		return true
	}
	uid := pathParts[len(pathParts)-1]
	serverName := pathParts[len(pathParts)-2]

	switch method {
	case http.MethodGet:
		return h.handleGetConfig(serverName, uid)
	case http.MethodPost:
		return h.handleStoreConfig(serverName, uid, body)
	default:
		h.sendErrorResponse(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return true
	}
}

// handleGetConfig handles configuration retrieval requests
func (h *MCPConfigHandler) handleGetConfig(serverName string, uid string) bool {
	config, err := h.configStore.GetConfig(serverName, uid)
	if err != nil {
		api.LogErrorf("Failed to get config for server %s, uid %s: %v", serverName, uid, err)
		h.sendErrorResponse(http.StatusInternalServerError, "CONFIG_ERROR", fmt.Sprintf("Failed to get configuration: %s", err.Error()))
		return true
	}

	response := struct {
		Success bool              `json:"success"`
		Config  map[string]string `json:"config"`
	}{
		Success: true,
		Config:  config,
	}

	responseBytes, _ := json.Marshal(response)
	h.callbacks.DecoderFilterCallbacks().SendLocalReply(
		http.StatusOK,
		string(responseBytes),
		nil, 0, "",
	)
	return true
}

// handleStoreConfig handles configuration storage requests
func (h *MCPConfigHandler) handleStoreConfig(serverName string, uid string, body []byte) bool {
	// Parse request body
	var requestBody struct {
		Config map[string]string `json:"config"`
	}
	if err := json.Unmarshal(body, &requestBody); err != nil {
		api.LogErrorf("Invalid request format for server %s, uid %s: %v", serverName, uid, err)
		h.sendErrorResponse(http.StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("Invalid request format: %s", err.Error()))
		return true
	}

	response, err := h.configStore.StoreConfig(serverName, uid, requestBody.Config)
	if err != nil {
		api.LogErrorf("Failed to store config for server %s, uid %s: %v", serverName, uid, err)
		h.sendErrorResponse(http.StatusInternalServerError, "CONFIG_ERROR", fmt.Sprintf("Failed to store configuration: %s", err.Error()))
		return true
	}

	responseBytes, _ := json.Marshal(response)
	h.callbacks.DecoderFilterCallbacks().SendLocalReply(
		http.StatusOK,
		string(responseBytes),
		nil, 0, "",
	)
	return true
}

// sendErrorResponse sends an error response with the specified status, code and message
func (h *MCPConfigHandler) sendErrorResponse(status int, code string, message string) {
	response := &ConfigResponse{
		Success: false,
		Error: &struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.callbacks.DecoderFilterCallbacks().SendLocalReply(
		status,
		string(responseBytes),
		nil, 0, "",
	)
}

// GetEncodedConfig retrieves and encodes the configuration for a given server and uid
func (h *MCPConfigHandler) GetEncodedConfig(serverName string, uid string) (string, error) {
	conf, err := h.configStore.GetConfig(serverName, uid)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	// Check if config exists and is not empty
	if config, ok := conf["config"]; ok && len(config) > 0 {
		// Convert config map to JSON string
		configBytes, err := json.Marshal(config)
		if err != nil {
			return "", fmt.Errorf("failed to marshal config: %w", err)
		}
		// Encode JSON string to base64
		return base64.StdEncoding.EncodeToString(configBytes), nil
	}

	return "", nil
}
