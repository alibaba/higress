package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// baiduProvider is the provider for baidu service.
const (
	baiduDomain             = "qianfan.baidubce.com"
	baiduChatCompletionPath = "/v2/chat/completions"
	baiduApiTokenDomain     = "iam.bj.baidubce.com"
	baiduApiTokenPort       = 443
	baiduApiTokenPath       = "/v1/BCE-BEARER/token"
	// refresh apiToken every 1 hour
	baiduApiTokenRefreshInterval = 3600
	// authorizationString expires in 30 minutes, authorizationString is used to generate apiToken
	// the default expiration time of apiToken is 24 hours
	baiduAuthorizationStringExpirationSeconds = 1800
	bce_prefix                                = "x-bce-"
)

type baiduProviderInitializer struct{}

func (g *baiduProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.baiduAccessKeyAndSecret == nil || len(config.baiduAccessKeyAndSecret) == 0 {
		return errors.New("no baiduAccessKeyAndSecret found in provider config")
	}
	if config.baiduApiTokenServiceName == "" {
		return errors.New("no baiduApiTokenServiceName found in provider config")
	}
	if !config.failover.enabled {
		config.useGlobalApiToken = true
	}
	return nil
}

func (g *baiduProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &baiduProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type baiduProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (g *baiduProvider) GetProviderType() string {
	return providerTypeBaidu
}

func (g *baiduProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	g.config.handleRequestHeaders(g, ctx, apiName, log)
	return nil
}

func (g *baiduProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return g.config.handleRequestBody(g, g.contextCache, ctx, apiName, body, log)
}

func (g *baiduProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, baiduChatCompletionPath)
	util.OverwriteRequestHostHeader(headers, baiduDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+g.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (g *baiduProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, baiduChatCompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}

func generateAuthorizationString(accessKeyAndSecret string, expirationInSeconds int) string {
	c := strings.Split(accessKeyAndSecret, ":")
	credentials := BceCredentials{
		AccessKeyId:     c[0],
		SecretAccessKey: c[1],
	}
	httpMethod := "GET"
	path := baiduApiTokenPath
	headers := map[string]string{"host": baiduApiTokenDomain}
	timestamp := time.Now().Unix()

	headersToSign := make([]string, 0, len(headers))
	for k := range headers {
		headersToSign = append(headersToSign, k)
	}

	return sign(credentials, httpMethod, path, headers, timestamp, expirationInSeconds, headersToSign)
}

// BceCredentials holds the access key and secret key
type BceCredentials struct {
	AccessKeyId     string
	SecretAccessKey string
}

// normalizeString performs URI encoding according to RFC 3986
func normalizeString(inStr string, encodingSlash bool) string {
	if inStr == "" {
		return ""
	}

	var result strings.Builder
	for _, ch := range []byte(inStr) {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '-' ||
			ch == '_' || ch == '~' || (!encodingSlash && ch == '/') {
			result.WriteByte(ch)
		} else {
			result.WriteString(fmt.Sprintf("%%%02X", ch))
		}
	}
	return result.String()
}

// getCanonicalTime generates a timestamp in UTC format
func getCanonicalTime(timestamp int64) string {
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}
	t := time.Unix(timestamp, 0).UTC()
	return t.Format("2006-01-02T15:04:05Z")
}

// getCanonicalUri generates a canonical URI
func getCanonicalUri(path string) string {
	return normalizeString(path, false)
}

// getCanonicalHeaders generates canonical headers
func getCanonicalHeaders(headers map[string]string, headersToSign []string) string {
	if len(headers) == 0 {
		return ""
	}

	// If headersToSign is not specified, use default headers
	if len(headersToSign) == 0 {
		headersToSign = []string{"host", "content-md5", "content-length", "content-type"}
	}

	// Convert headersToSign to a map for easier lookup
	headerMap := make(map[string]bool)
	for _, header := range headersToSign {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = true
	}

	// Create a slice to hold the canonical headers
	var canonicalHeaders []string
	for k, v := range headers {
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)

		// Add headers that start with x-bce- or are in headersToSign
		if strings.HasPrefix(k, bce_prefix) || headerMap[k] {
			canonicalHeaders = append(canonicalHeaders,
				fmt.Sprintf("%s:%s", normalizeString(k, true), normalizeString(v, true)))
		}
	}

	// Sort the canonical headers
	sort.Strings(canonicalHeaders)

	return strings.Join(canonicalHeaders, "\n")
}

// sign generates the authorization string
func sign(credentials BceCredentials, httpMethod, path string, headers map[string]string,
	timestamp int64, expirationInSeconds int,
	headersToSign []string) string {

	// Generate sign key
	signKeyInfo := fmt.Sprintf("bce-auth-v1/%s/%s/%d",
		credentials.AccessKeyId,
		getCanonicalTime(timestamp),
		expirationInSeconds)

	// Generate sign key using HMAC-SHA256
	h := hmac.New(sha256.New, []byte(credentials.SecretAccessKey))
	h.Write([]byte(signKeyInfo))
	signKey := hex.EncodeToString(h.Sum(nil))

	// Generate canonical URI
	canonicalUri := getCanonicalUri(path)

	// Generate canonical headers
	canonicalHeaders := getCanonicalHeaders(headers, headersToSign)

	// Generate string to sign
	stringToSign := strings.Join([]string{
		httpMethod,
		canonicalUri,
		"",
		canonicalHeaders,
	}, "\n")

	// Calculate final signature
	h = hmac.New(sha256.New, []byte(signKey))
	h.Write([]byte(stringToSign))
	signature := hex.EncodeToString(h.Sum(nil))

	// Generate final authorization string
	if len(headersToSign) > 0 {
		return fmt.Sprintf("%s/%s/%s", signKeyInfo, strings.Join(headersToSign, ";"), signature)
	}
	return fmt.Sprintf("%s//%s", signKeyInfo, signature)
}

// GetTickFunc Refresh apiToken (apiToken) periodically, the maximum apiToken expiration time is 24 hours
func (g *baiduProvider) GetTickFunc(log wrapper.Log) (tickPeriod int64, tickFunc func()) {
	vmID := generateVMID()

	return baiduApiTokenRefreshInterval * 1000, func() {
		// Only the Wasm VM that successfully acquires the lease will refresh the apiToken
		if g.config.tryAcquireOrRenewLease(vmID, log) {
			log.Debugf("Successfully acquired or renewed lease for baidu apiToken refresh task, vmID: %v", vmID)
			// Get the apiToken that is about to expire, will be removed after the new apiToken is obtained
			oldApiTokens, _, err := getApiTokens(g.config.failover.ctxApiTokens)
			if err != nil {
				log.Errorf("Get old apiToken failed: %v", err)
				return
			}
			log.Debugf("Old apiTokens: %v", oldApiTokens)

			for _, accessKeyAndSecret := range g.config.baiduAccessKeyAndSecret {
				authorizationString := generateAuthorizationString(accessKeyAndSecret, baiduAuthorizationStringExpirationSeconds)
				log.Debugf("Generate authorizationString: %v", authorizationString)
				g.generateNewApiToken(authorizationString, log)
			}

			// remove old old apiToken
			for _, token := range oldApiTokens {
				log.Debugf("Remove old apiToken: %v", token)
				removeApiToken(g.config.failover.ctxApiTokens, token, log)
			}
		}
	}
}

func (g *baiduProvider) generateNewApiToken(authorizationString string, log wrapper.Log) {
	client := wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: g.config.baiduApiTokenServiceName,
		Host: g.config.baiduApiTokenServiceHost,
		Port: g.config.baiduApiTokenServicePort,
	})

	headers := [][2]string{
		{"content-type", "application/json"},
		{"Authorization", authorizationString},
	}

	var apiToken string
	err := client.Get(baiduApiTokenPath, headers, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		if statusCode == 201 {
			var response map[string]interface{}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				log.Errorf("Unmarshal response failed: %v", err)
			} else {
				apiToken = response["token"].(string)
				addApiToken(g.config.failover.ctxApiTokens, apiToken, log)
			}
		} else {
			log.Errorf("Get apiToken failed, status code: %d, response body: %s", statusCode, string(responseBody))
		}
	}, 30000)

	if err != nil {
		log.Errorf("Get apiToken failed: %v", err)
	}
}
