package higress_ops

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
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
func NewOpsClient(istiodURL, envoyAdminURL, namespace string) *OpsClient {
	if namespace == "" {
		namespace = "higress-system"
	}

	client := &OpsClient{
		istiodURL:     istiodURL,
		envoyAdminURL: envoyAdminURL,
		namespace:     namespace,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	return client
}

// GetIstiodDebug calls Istiod debug endpoints
func (c *OpsClient) GetIstiodDebug(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, c.istiodURL, path)
}

// GetEnvoyAdmin calls Envoy admin endpoints
func (c *OpsClient) GetEnvoyAdmin(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, c.envoyAdminURL, path)
}

// GetIstiodDebugWithParams calls Istiod debug endpoints with query parameters
func (c *OpsClient) GetIstiodDebugWithParams(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	return c.requestWithParams(ctx, c.istiodURL, path, params)
}

// GetEnvoyAdminWithParams calls Envoy admin endpoints with query parameters
func (c *OpsClient) GetEnvoyAdminWithParams(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	return c.requestWithParams(ctx, c.envoyAdminURL, path, params)
}

func (c *OpsClient) request(ctx context.Context, baseURL, path string) ([]byte, error) {
	return c.requestWithParams(ctx, baseURL, path, nil)
}

func (c *OpsClient) requestWithParams(ctx context.Context, baseURL, path string, params map[string]string) ([]byte, error) {
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

	// Use the provided context, or create a new one if nil
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// Try to get Istiod token from context first (passthrough from MCP client)
	// This is only applied for Istiod requests, not Envoy admin
	if c.isBaseURL(baseURL, c.istiodURL) {
		if istiodToken, ok := common.GetIstiodToken(ctx); ok && istiodToken != "" {
			req.Header.Set("Authorization", "Bearer "+istiodToken)
			api.LogInfof("Istiod API request: Using X-Istiod-Token from context for %s", path)
		} else {
			api.LogWarnf("Istiod API request: No authentication token available for %s. Request may fail with 401", path)
		}
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
