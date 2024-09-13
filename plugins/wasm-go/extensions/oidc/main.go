package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	oidc "github.com/higress-group/oauth2-proxy"
	"github.com/higress-group/oauth2-proxy/pkg/apis/options"
	"github.com/higress-group/oauth2-proxy/pkg/util"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		// 插件名称
		"oidc",
		// 为解析插件配置，设置自定义函数
		wrapper.ParseConfigBy(parseConfig),
		// 为处理请求头，设置自定义函数
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		// 为处理响应头，设置自定义函数
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
	)
}

var oidcHandler *oidc.OAuthProxy

type PluginConfig struct {
	options *options.Options
}

// 在控制台插件配置中填写的yaml配置会自动转换为json，此处直接从json这个参数里解析配置即可
func parseConfig(json gjson.Result, config *PluginConfig, log wrapper.Log) error {
	oidc.SetLogger(log)
	opts, err := oidc.LoadOptions(json)
	if err != nil {
		return err
	}
	opts.Providers[0].Scope = strings.Replace(opts.Providers[0].Scope, ";", " ", -1)
	config.options = opts

	oidcHandler, err = oidc.NewOAuthProxy(opts)
	if err != nil {
		return err
	}

	wrapper.RegisteTickFunc(opts.VerifierInterval.Milliseconds(), func() {
		oidcHandler.SetVerifier(opts)
	})
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	oidcHandler.SetContext(ctx)
	req := getHttpRequest()
	rw := util.NewRecorder()
	if options.IsAllowedByMode(req.URL.Host, req.URL.Path, config.options.MatchRules, config.options.ProxyPrefix) {
		log.Infof("request is allowed by mode %s", config.options.MatchRules.Mode)
		return types.ActionContinue
	}

	// TODO: remove this verifier after envoy support send request during parseConfig
	if err := oidcHandler.ValidateVerifier(); err != nil {
		log.Critical(err.Error())
		return types.ActionContinue
	}

	oidcHandler.ServeHTTP(rw, req)
	if code := rw.GetStatus(); code != 0 {
		return types.ActionContinue
	}
	return types.ActionPause
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	value := ctx.GetContext(oidc.SetCookieHeader)
	if value != nil {
		proxywasm.AddHttpResponseHeader(oidc.SetCookieHeader, value.(string))
	}
	oidcHandler.SetContext(nil)
	return types.ActionContinue
}

func getHttpRequest() *http.Request {
	headers, _ := proxywasm.GetHttpRequestHeaders()
	var method, path, authority, scheme string
	for _, header := range headers {
		switch header[0] {
		case ":method":
			method = header[1]
		case ":path":
			path = header[1]
		case ":authority":
			authority = header[1]
		case ":scheme":
			scheme = header[1]
		}
	}
	rawURL := fmt.Sprintf("%s://%s%s", scheme, authority, path)
	parsedURL, _ := url.Parse(rawURL)

	req := &http.Request{
		Method: method,
		URL:    parsedURL,
		Header: make(http.Header),
		Body:   nil,
	}
	req.Form, _ = url.ParseQuery(parsedURL.RawQuery)

	for _, header := range headers {
		if !strings.HasPrefix(header[0], ":") {
			req.Header.Add(header[0], header[1])
		}
	}
	return req
}
