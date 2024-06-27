package main

import (
	"errors"
	"net/http"
	"net/url"
	"oidc/pkg/apis/options"
	"oidc/pkg/util"
	"oidc/pkg/validation"
	"oidc/providers"
	"strings"

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

type OidcConfig struct {
	Options     *options.Options
	OidcHandler *OAuthProxy
}

// 在控制台插件配置中填写的yaml配置会自动转换为json，此处直接从json这个参数里解析配置即可
func parseConfig(json gjson.Result, config *OidcConfig, log wrapper.Log) error {
	util.Logger = &log
	opts, err := options.LoadOptions(json)
	if err != nil {
		return err
	}

	if err = validation.Validate(opts); err != nil {
		return err
	}

	config.Options = opts
	validator := func(string) bool { return true }
	oauthproxy, err := NewOAuthProxy(opts, validator)
	if err != nil {
		return err
	}
	config.OidcHandler = oauthproxy

	wrapper.RegisteTickFunc(opts.VerifierInterval.Milliseconds(), func() {
		providers.NewVerifierFromConfig(config.Options.Providers[0], config.OidcHandler.provider.Data(), config.OidcHandler.client)
	})

	wrapper.RegisteTickFunc(opts.UpdateKeysInterval.Milliseconds(), func() {
		if *&config.OidcHandler.provider.Data().Verifier != nil {
			(*config.OidcHandler.provider.Data().Verifier.GetKeySet()).UpdateKeys(oauthproxy.client, config.Options.Providers[0].OIDCConfig.VerifierRequestTimeout)
		}
	})
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config OidcConfig, log wrapper.Log) types.Action {
	if err := validateVerifier(config.OidcHandler); err != nil {
		util.SendError(err.Error(), nil, http.StatusInternalServerError)
		return types.ActionContinue
	} else {
		config.OidcHandler.Ctx = ctx
		req := getHttpRequest()
		rw := util.NewRecorder()

		config.OidcHandler.serveMux.ServeHTTP(rw, req)
		if code := rw.GetStatus(); code != 0 {
			return types.ActionContinue
		}
	}
	return types.ActionPause
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config OidcConfig, log wrapper.Log) types.Action {
	value := ctx.GetContext(SetCookieHeader)
	if value != nil {
		proxywasm.AddHttpResponseHeader(SetCookieHeader, value.(string))
	}
	config.OidcHandler.Ctx = nil
	return types.ActionContinue
}

func validateVerifier(OidcHandler *OAuthProxy) error {
	if OidcHandler.provider.Data().Verifier == nil {
		return errors.New("Failed to obtain OpenID configuration. (There may be an error in the service configuration of the OIDC provider)")
	}
	return nil
}

func getHttpRequest() *http.Request {
	headers, _ := proxywasm.GetHttpRequestHeaders()

	var method, path string
	for _, header := range headers {
		switch header[0] {
		case ":method":
			method = header[1]
		case ":path":
			path = header[1]
		}
	}
	parsedURL, _ := url.Parse(path)

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
