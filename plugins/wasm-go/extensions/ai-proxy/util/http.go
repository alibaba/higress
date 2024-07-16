package util

import "github.com/higress-group/proxy-wasm-go-sdk/proxywasm"

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

func OverwriteRequestHost(host string) error {
	if originHost, err := proxywasm.GetHttpRequestHeader(":authority"); err == nil {
		_ = proxywasm.ReplaceHttpRequestHeader("X-ENVOY-ORIGINAL-HOST", originHost)
	}
	return proxywasm.ReplaceHttpRequestHeader(":authority", host)
}

func OverwriteRequestPath(path string) error {
	if originPath, err := proxywasm.GetHttpRequestHeader(":path"); err == nil {
		_ = proxywasm.ReplaceHttpRequestHeader("X-ENVOY-ORIGINAL-PATH", originPath)
	}
	return proxywasm.ReplaceHttpRequestHeader(":path", path)
}
