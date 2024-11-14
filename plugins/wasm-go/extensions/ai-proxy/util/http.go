package util

import (
	"net/http"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

const (
	HeaderContentType = "Content-Type"

	MimeTypeTextPlain       = "text/plain"
	MimeTypeApplicationJson = "application/json"
)

func SendResponse(statusCode uint32, statusCodeDetails string, contentType, body string) error {
	return proxywasm.SendHttpResponseWithDetail(statusCode, statusCodeDetails, CreateHeaders(HeaderContentType, contentType), []byte(body), -1)
}

func CreateHeaders(kvs ...string) [][2]string {
	headers := make([][2]string, 0, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		headers = append(headers, [2]string{kvs[i], kvs[i+1]})
	}
	return headers
}

func OverwriteRequestPath(path string) error {
	if originPath, err := proxywasm.GetHttpRequestHeader(":path"); err == nil {
		_ = proxywasm.ReplaceHttpRequestHeader("X-ENVOY-ORIGINAL-PATH", originPath)
	}
	return proxywasm.ReplaceHttpRequestHeader(":path", path)
}

func OverwriteRequestAuthorization(credential string) error {
	if exist, _ := proxywasm.GetHttpRequestHeader("X-HI-ORIGINAL-AUTH"); exist == "" {
		if originAuth, err := proxywasm.GetHttpRequestHeader("Authorization"); err == nil {
			_ = proxywasm.AddHttpRequestHeader("X-HI-ORIGINAL-AUTH", originAuth)
		}
	}
	return proxywasm.ReplaceHttpRequestHeader("Authorization", credential)
}

func OverwriteRequestHostHeader(headers http.Header, host string) {
	if originHost, err := proxywasm.GetHttpRequestHeader(":authority"); err == nil {
		headers.Set("X-ENVOY-ORIGINAL-HOST", originHost)
	}
	headers.Set(":authority", host)
}

func OverwriteRequestPathHeader(headers http.Header, path string) {
	if originPath, err := proxywasm.GetHttpRequestHeader(":path"); err == nil {
		headers.Set("X-ENVOY-ORIGINAL-PATH", originPath)
	}
	headers.Set(":path", path)
}

func OverwriteRequestAuthorizationHeader(headers http.Header, credential string) {
	if exist := headers.Get("X-HI-ORIGINAL-AUTH"); exist == "" {
		if originAuth := headers.Get("Authorization"); originAuth != "" {
			headers.Set("X-HI-ORIGINAL-AUTH", originAuth)
		}
	}
	headers.Set("Authorization", credential)
}

func HeaderToSlice(header http.Header) [][2]string {
	slice := make([][2]string, 0, len(header))
	for key, values := range header {
		for _, value := range values {
			slice = append(slice, [2]string{key, value})
		}
	}
	return slice
}

func SliceToHeader(slice [][2]string) http.Header {
	header := make(http.Header)
	for _, pair := range slice {
		key := pair[0]
		value := pair[1]
		header.Add(key, value)
	}
	return header
}

func GetOriginalHttpHeaders() http.Header {
	originalHeaders, _ := proxywasm.GetHttpRequestHeaders()
	return SliceToHeader(originalHeaders)
}

func ReplaceOriginalHttpHeaders(headers http.Header) {
	modifiedHeaders := HeaderToSlice(headers)
	_ = proxywasm.ReplaceHttpRequestHeaders(modifiedHeaders)
}
