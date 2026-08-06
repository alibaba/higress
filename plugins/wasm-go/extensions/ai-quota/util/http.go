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
	if len(kvs)%2 != 0 {
		kvs = kvs[:len(kvs)-1]
	}
	headers := make([][2]string, 0, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		headers = append(headers, [2]string{kvs[i], kvs[i+1]})
	}
	return headers
}
