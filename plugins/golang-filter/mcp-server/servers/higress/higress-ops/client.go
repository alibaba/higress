package higress_ops

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// OpsClient handles Istio/Envoy debug API connections and operations
type OpsClient struct {
	istiodURL     string
	envoyAdminURL string
	namespace     string
	istiodToken   string // Istiod authentication token (audience: istio-ca)
	httpClient    *http.Client
}

// NewOpsClient creates a new ops client for Istio/Envoy debug interfaces
func NewOpsClient(istiodURL, envoyAdminURL, namespace, istiodToken string) *OpsClient {
	if namespace == "" {
		namespace = "higress-system"
	}

	client := &OpsClient{
		istiodURL:     istiodURL,
		envoyAdminURL: envoyAdminURL,
		namespace:     namespace,
		istiodToken:   istiodToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if istiodToken != "" {
		api.LogInfof("Istio/Envoy Ops client initialized: istiod=%s, envoy=%s, namespace=%s (with authentication token)",
			istiodURL, envoyAdminURL, namespace)
	} else {
		api.LogInfof("Istio/Envoy Ops client initialized: istiod=%s, envoy=%s, namespace=%s (no authentication token)",
			istiodURL, envoyAdminURL, namespace)
		api.LogWarnf("No Istiod authentication token provided. Cross-pod Istiod API requests may fail with 401 errors.")
	}

	return client
}

// GetIstiodDebug calls Istiod debug endpoints
func (c *OpsClient) GetIstiodDebug(path string) ([]byte, error) {
	return c.request(c.istiodURL, path)
}

// GetEnvoyAdmin calls Envoy admin endpoints
func (c *OpsClient) GetEnvoyAdmin(path string) ([]byte, error) {
	return c.request(c.envoyAdminURL, path)
}

// GetIstiodDebugWithParams calls Istiod debug endpoints with query parameters
func (c *OpsClient) GetIstiodDebugWithParams(path string, params map[string]string) ([]byte, error) {
	return c.requestWithParams(c.istiodURL, path, params)
}

// GetEnvoyAdminWithParams calls Envoy admin endpoints with query parameters
func (c *OpsClient) GetEnvoyAdminWithParams(path string, params map[string]string) ([]byte, error) {
	return c.requestWithParams(c.envoyAdminURL, path, params)
}

func (c *OpsClient) request(baseURL, path string) ([]byte, error) {
	return c.requestWithParams(baseURL, path, nil)
}

func (c *OpsClient) requestWithParams(baseURL, path string, params map[string]string) ([]byte, error) {
	fullURL := baseURL + path

	// Add query parameters if provided
	if len(params) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL %s: %w", fullURL, err)
		}

		q := u.Query()
		for key, value := range params {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	api.LogDebugf("Ops API GET %s", fullURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// Add Istiod authentication token if configured
	// Istiod requires JWT token with audience "istio-ca" for cross-pod access
	if c.istiodToken != "" && c.isBaseURL(baseURL, c.istiodURL) {
		req.Header.Set("Authorization", "Bearer "+c.istiodToken)
		api.LogDebugf("Added Istiod authentication token for request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, nil
}

// GetNamespace returns the configured namespace
func (c *OpsClient) GetNamespace() string {
	return c.namespace
}

// GetIstiodURL returns the Istiod URL
func (c *OpsClient) GetIstiodURL() string {
	return c.istiodURL
}

// GetEnvoyAdminURL returns the Envoy admin URL
func (c *OpsClient) GetEnvoyAdminURL() string {
	return c.envoyAdminURL
}

// isBaseURL checks if the baseURL matches the targetURL (for determining if token is needed)
func (c *OpsClient) isBaseURL(baseURL, targetURL string) bool {
	return baseURL == targetURL
}
