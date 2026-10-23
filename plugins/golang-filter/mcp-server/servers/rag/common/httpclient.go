package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient handles HTTP API connections and operations
type HTTPClient struct {
	baseURL    string
	headers    map[string]string
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP client with base URL and optional headers
func NewHTTPClient(baseURL string, headers map[string]string) *HTTPClient {
	client := &HTTPClient{
		baseURL: baseURL,
		headers: make(map[string]string),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Copy headers to avoid external modification
	if headers != nil {
		for k, v := range headers {
			client.headers[k] = v
		}
	}

	return client
}

// SetHeader sets a header for all requests
func (c *HTTPClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// SetHeaders sets multiple headers for all requests
func (c *HTTPClient) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		c.headers[k] = v
	}
}

// RemoveHeader removes a header
func (c *HTTPClient) RemoveHeader(key string) {
	delete(c.headers, key)
}

// Get performs a GET request
func (c *HTTPClient) Get(path string) ([]byte, error) {
	return c.request("GET", path, nil)
}

// Post performs a POST request
func (c *HTTPClient) Post(path string, data interface{}) ([]byte, error) {
	return c.request("POST", path, data)
}

// Put performs a PUT request
func (c *HTTPClient) Put(path string, data interface{}) ([]byte, error) {
	return c.request("PUT", path, data)
}

// Delete performs a DELETE request
func (c *HTTPClient) Delete(path string) ([]byte, error) {
	return c.request("DELETE", path, nil)
}

// Patch performs a PATCH request
func (c *HTTPClient) Patch(path string, data interface{}) ([]byte, error) {
	return c.request("PATCH", path, data)
}

// RequestWithHeaders performs a request with additional headers for this request only
func (c *HTTPClient) RequestWithHeaders(method, path string, data interface{}, additionalHeaders map[string]string) ([]byte, error) {
	return c.requestWithHeaders(method, path, data, additionalHeaders)
}

func (c *HTTPClient) request(method, path string, data interface{}) ([]byte, error) {
	return c.requestWithHeaders(method, path, data, nil)
}

func (c *HTTPClient) requestWithHeaders(method, path string, data interface{}, additionalHeaders map[string]string) ([]byte, error) {
	url := c.baseURL + path

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Set additional headers for this request
	if additionalHeaders != nil {
		for k, v := range additionalHeaders {
			req.Header.Set(k, v)
		}
	}

	// Set Content-Type for requests with body
	if data != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

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

// SetTimeout sets the HTTP client timeout
func (c *HTTPClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// GetBaseURL returns the base URL
func (c *HTTPClient) GetBaseURL() string {
	return c.baseURL
}

// SetBaseURL sets the base URL
func (c *HTTPClient) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}