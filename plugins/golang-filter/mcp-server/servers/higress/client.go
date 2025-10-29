package higress

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// HigressClient handles Higress Console API connections and operations
type HigressClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

func NewHigressClient(baseURL string) *HigressClient {
	client := &HigressClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	api.LogInfof("Higress Console client initialized: %s", baseURL)

	return client
}

func (c *HigressClient) Get(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, "GET", path, nil)
}

func (c *HigressClient) Post(ctx context.Context, path string, data interface{}) ([]byte, error) {
	return c.request(ctx, "POST", path, data)
}

func (c *HigressClient) Put(ctx context.Context, path string, data interface{}) ([]byte, error) {
	return c.request(ctx, "PUT", path, data)
}

func (c *HigressClient) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, "DELETE", path, nil)
}

// DeleteWithBody performs a DELETE request with a request body
func (c *HigressClient) DeleteWithBody(ctx context.Context, path string, data interface{}) ([]byte, error) {
	return c.request(ctx, "DELETE", path, data)
}

func (c *HigressClient) request(ctx context.Context, method, path string, data interface{}) ([]byte, error) {
	url := c.baseURL + path

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
		api.LogDebugf("Higress API %s %s: %s", method, url, string(jsonData))
	} else {
		api.LogDebugf("Higress API %s %s", method, url)
	}

	// Create context with timeout if not already set
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Try to get Authorization header from context first (passthrough from MCP client)
	if authHeader, ok := common.GetAuthHeader(ctx); ok && authHeader != "" {
		req.Header.Set("Authorization", authHeader)
		api.LogDebugf("Higress API request: Using Authorization header from context for %s %s", method, path)
	} else {
		api.LogWarnf("Higress API request: No authentication credentials available for %s %s", method, path)
		return nil, fmt.Errorf("no authentication credentials available for %s %s", method, path)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, nil
}
