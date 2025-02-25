package quark

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-search/engine"
)

type QuarkSearch struct {
	apiKey             string
	secretKey          string
	timeoutMillisecond uint32
	client             wrapper.HttpClient
	count              uint32
	endpoint           string
}

const (
	Path               = "/linked-retrieval/linked-retrieval-entry/v2/linkedRetrieval/commands/genericSearch"
	ContentSha256      = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // for empty body
	Action             = "GenericSearch"
	Version            = "2024-11-11"
	SignatureAlgorithm = "ACS3-HMAC-SHA256"
	SignedHeaders      = "host;x-acs-action;x-acs-content-sha256;x-acs-date;x-acs-signature-nonce;x-acs-version"
)

func urlEncoding(rawStr string) string {
	encodedStr := url.PathEscape(rawStr)
	encodedStr = strings.ReplaceAll(encodedStr, "+", "%2B")
	encodedStr = strings.ReplaceAll(encodedStr, ":", "%3A")
	encodedStr = strings.ReplaceAll(encodedStr, "=", "%3D")
	encodedStr = strings.ReplaceAll(encodedStr, "&", "%26")
	encodedStr = strings.ReplaceAll(encodedStr, "$", "%24")
	encodedStr = strings.ReplaceAll(encodedStr, "@", "%40")
	// encodedStr := url.QueryEscape(rawStr)
	return encodedStr
}

func getSignature(stringToSign, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

func getCanonicalHeaders(params map[string]string) string {
	paramArray := []string{}
	for k, v := range params {
		paramArray = append(paramArray, k+":"+v)
	}
	sort.Slice(paramArray, func(i, j int) bool {
		return paramArray[i] <= paramArray[j]
	})
	return strings.Join(paramArray, "\n") + "\n"
}

func getHasedString(input string) string {
	hash := sha256.Sum256([]byte(input))
	hashHex := hex.EncodeToString(hash[:])
	return hashHex
}

func generateHexID(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func NewQuarkSearch(config *gjson.Result) (*QuarkSearch, error) {
	engine := &QuarkSearch{}
	engine.apiKey = config.Get("apiKey").String()
	if engine.apiKey == "" {
		return nil, errors.New("apiKey not found")
	}
	engine.secretKey = config.Get("optionArgs.secretKey").String()
	if engine.secretKey == "" {
		return nil, errors.New("secretKey not found")
	}
	serviceName := config.Get("serviceName").String()
	if serviceName == "" {
		return nil, errors.New("serviceName not found")
	}
	servicePort := config.Get("servicePort").Int()
	if servicePort == 0 {
		return nil, errors.New("servicePort not found")
	}
	engine.endpoint = config.Get("optionArgs.endpoint").String()
	if engine.endpoint == "" {
		return nil, errors.New("endpoint not found")
	}
	engine.count = uint32(config.Get("count").Int())
	if engine.count == 0 {
		engine.count = 10
	}
	engine.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
	})
	engine.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if engine.timeoutMillisecond == 0 {
		engine.timeoutMillisecond = 5000
	}
	return engine, nil
}

func (g QuarkSearch) NeedExectue(ctx engine.SearchContext) bool {
	return ctx.EngineType == "" || ctx.EngineType == "internet"
}

func (g QuarkSearch) Client() wrapper.HttpClient {
	return g.client
}

func (g QuarkSearch) CallArgs(ctx engine.SearchContext) engine.CallArgs {
	query := strings.Join(ctx.Querys, " ")
	canonicalURI := Path
	queryParams := map[string]string{
		"query":     query,
		"timeRange": "NoLimit",
	}
	queryParamsStr := []string{}
	for k, v := range queryParams {
		queryParamsStr = append(queryParamsStr, k+"="+urlEncoding(v))
	}
	canonicalQueryString := strings.Join(queryParamsStr, "&")
	timeStamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	randomID, _ := generateHexID(32)
	params := map[string]string{
		"host":                  g.endpoint,
		"x-acs-action":          Action,
		"x-acs-content-sha256":  ContentSha256,
		"x-acs-date":            timeStamp,
		"x-acs-signature-nonce": randomID,
		"x-acs-version":         Version,
	}
	canonicalHeaders := getCanonicalHeaders(params)
	canonicalRequest := http.MethodGet + "\n" + canonicalURI + "\n" + canonicalQueryString + "\n" + canonicalHeaders + "\n" + SignedHeaders + "\n" + ContentSha256
	stringToSign := SignatureAlgorithm + "\n" + getHasedString(canonicalRequest)

	authHeaderFmt := "%s Credential=%s,SignedHeaders=%s,Signature=%s"
	authHeader := fmt.Sprintf(authHeaderFmt, SignatureAlgorithm, g.apiKey, SignedHeaders, getSignature(stringToSign, g.secretKey))

	reqParams := url.Values{}
	for k, v := range queryParams {
		reqParams.Add(k, v)
	}
	requestURL := fmt.Sprintf("https://%s%s?%s", g.endpoint, Path, reqParams.Encode())

	return engine.CallArgs{
		Method: http.MethodGet,
		Url:    requestURL,
		Headers: [][2]string{
			{"x-acs-date", timeStamp},
			{"x-acs-signature-nonce", randomID},
			{"x-acs-content-sha256", ContentSha256},
			{"x-acs-version", Version},
			{"x-acs-action", Action},
			{"Authorization", authHeader},
		},
		Body:               nil,
		TimeoutMillisecond: g.timeoutMillisecond,
	}
}

func (g QuarkSearch) ParseResult(ctx engine.SearchContext, response []byte) []engine.SearchResult {
	jsonObj := gjson.ParseBytes(response)
	var results []engine.SearchResult
	for index, item := range jsonObj.Get("pageItems").Array() {
		result := engine.SearchResult{
			Title:   item.Get("title").String(),
			Link:    item.Get("link").String(),
			Content: item.Get("mainText").String(),
		}
		if result.Valid() && index < int(g.count) {
			results = append(results, result)
		}
	}
	return results
}
