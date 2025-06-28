package higress

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// APIResponse represents the standard API response format
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// HigressClient handles Higress Console API connections and operations
type HigressClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewHigressClient creates a new HigressClient instance
func NewHigressClient(baseURL, username, password string) *HigressClient {
	client := &HigressClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	api.LogInfof("Higress Console client initialized: %s", baseURL)

	return client
}

// Get performs a GET request to the Higress Console API
func (c *HigressClient) Get(path string) ([]byte, error) {
	return c.request("GET", path, nil)
}

// Post performs a POST request to the Higress Console API
func (c *HigressClient) Post(path string, data interface{}) ([]byte, error) {
	return c.request("POST", path, data)
}

// Put performs a PUT request to the Higress Console API
func (c *HigressClient) Put(path string, data interface{}) ([]byte, error) {
	return c.request("PUT", path, data)
}

// Ping tests the connection to Higress Console
func (c *HigressClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/v1/routes", nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ping request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ping failed with status: %d", resp.StatusCode)
	}

	return nil
}

// request performs an HTTP request to the Higress Console API
func (c *HigressClient) request(method, path string, data interface{}) ([]byte, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	api.LogDebugf("Higress API response: %d %s", resp.StatusCode, string(respBody))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse and validate API response
	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		// If it's not a standard API response format, return the raw body
		return respBody, nil
	}

	// Check if the API returned success=false
	if !apiResp.Success {
		return nil, fmt.Errorf("API returned error: %s", apiResp.Message)
	}

	return respBody, nil
}
