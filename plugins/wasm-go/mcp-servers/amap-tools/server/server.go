// server/server.go
package server

import (
	"encoding/json"
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

// Define your server configuration structure
type AmapMCPServer struct {
    ApiKey string `json:"apiKey"`
    // Add other configuration fields as needed
}

// Validate the configuration
func (s AmapMCPServer) ConfigHasError() error {
    if s.ApiKey == "" {
        return errors.New("missing api key")
    }
    return nil
}

// Parse configuration from JSON
func ParseFromConfig(configBytes []byte, server *AmapMCPServer) error {
    return json.Unmarshal(configBytes, server)
}

// Parse configuration from HTTP request
func ParseFromRequest(ctx wrapper.HttpContext, server *AmapMCPServer) error {
    return ctx.ParseMCPServerConfig(server)
}