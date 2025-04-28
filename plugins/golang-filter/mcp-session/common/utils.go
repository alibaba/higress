package common

import (
	"fmt"
	"net/url"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

type RequestURL struct {
	Method     string
	Scheme     string
	Host       string
	Path       string
	BaseURL    string
	ParsedURL  *url.URL
	InternalIP bool
}

func NewRequestURL(header api.RequestHeaderMap) *RequestURL {
	method, _ := header.Get(":method")
	scheme, _ := header.Get(":scheme")
	host, _ := header.Get(":authority")
	path, _ := header.Get(":path")
	internalIP, _ := header.Get("x-envoy-internal")
	baseURL := fmt.Sprintf("%s://%s", scheme, host)
	parsedURL, _ := url.Parse(path)
	api.LogDebugf("RequestURL: method=%s, scheme=%s, host=%s, path=%s", method, scheme, host, path)
	return &RequestURL{Method: method, Scheme: scheme, Host: host, Path: path, BaseURL: baseURL, ParsedURL: parsedURL, InternalIP: internalIP == "true"}
}
