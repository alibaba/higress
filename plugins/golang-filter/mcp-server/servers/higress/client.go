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

// HigressClient handles Higress Console API connections and operations
type HigressClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

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

func (c *HigressClient) Get(path string) ([]byte, error) {
	return c.request("GET", path, nil)
}

func (c *HigressClient) Post(path string, data interface{}) ([]byte, error) {
	return c.request("POST", path, data)
}

func (c *HigressClient) Put(path string, data interface{}) ([]byte, error) {
	return c.request("PUT", path, data)
}

func (c *HigressClient) Delete(path string) ([]byte, error) {
	return c.request("DELETE", path, nil)
}

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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	var responseJson map[string]interface{}
	if err := json.Unmarshal(respBody, &responseJson); err != nil {
		// If it's not valid JSON, return the raw body
		api.LogDebugf("Response is not valid JSON, returning raw body")
		return respBody, nil
	}

	// If success field exists and is False, it indicates an error
	if success, exists := responseJson["success"]; exists && success == false {
		errorMsg := "Unknown API error"
		if msg, ok := responseJson["message"].(string); ok {
			errorMsg = msg
		}
		api.LogErrorf("Request API error for %s %s: %s", method, path, errorMsg)
		return nil, fmt.Errorf("request API error for %s %s: %s", method, path, errorMsg)
	}

	return respBody, nil
}
