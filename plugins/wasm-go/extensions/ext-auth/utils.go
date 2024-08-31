package main

import (
	"net/http"
	"sort"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

func sendResponse(statusCode uint32, statusCodeDetailData string, headers http.Header) error {
	return proxywasm.SendHttpResponseWithDetail(statusCode, statusCodeDetailData, reconvertHeaders(headers), nil, -1)
}

func reconvertHeaders(headers http.Header) [][2]string {
	var ret [][2]string
	if headers == nil {
		return ret
	}
	for k, vs := range headers {
		for _, v := range vs {
			ret = append(ret, [2]string{k, v})
		}
	}
	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i][0] < ret[j][0]
	})
	return ret
}

func extractFromHeader(headers [][2]string, headerKey string) string {
	for _, header := range headers {
		key := header[0]
		if strings.ToLower(key) == headerKey {
			return strings.TrimSpace(header[1])
		}
	}
	return ""
}
