package provider

import (
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
	"net/http"
)

const (
	ctxRequestBody = "requestBody"
	ctxRetryCount  = "retryCount"
)

type retryOnFailure struct {
	// @Title zh-CN 是否启用请求重试
	enabled bool `required:"false" yaml:"enabled" json:"enabled"`
	// @Title zh-CN 重试次数
	maxRetries int64 `required:"false" yaml:"maxRetries" json:"maxRetries"`
	// @Title zh-CN 重试超时时间
	retryTimeout int64 `required:"false" yaml:"retryTimeout" json:"retryTimeout"`
}

func (r *retryOnFailure) FromJson(json gjson.Result) {
	r.enabled = json.Get("enabled").Bool()
	r.maxRetries = json.Get("maxRetries").Int()
	if r.maxRetries == 0 {
		r.maxRetries = 1
	}
	r.retryTimeout = json.Get("retryTimeout").Int()
	if r.retryTimeout == 0 {
		r.retryTimeout = 5000
	}
}

func (c *ProviderConfig) isRetryOnFailureEnabled() bool {
	return c.retryOnFailure.enabled
}

func (c *ProviderConfig) retryFailedRequest(activeProvider Provider, ctx wrapper.HttpContext, log wrapper.Log) {
	log.Debugf("Retry failed request: provider=%s", activeProvider.GetProviderType())
	retryClient := createRetryClient(ctx)
	apiName, _ := ctx.GetContext(CtxKeyApiName).(ApiName)
	ctx.SetContext(ctxRetryCount, 1)
	c.sendRetryRequest(ctx, apiName, activeProvider, retryClient, log)
}

func (c *ProviderConfig) transformResponseHeadersAndBody(ctx wrapper.HttpContext, activeProvider Provider, apiName ApiName, headers http.Header, body []byte, log wrapper.Log) ([][2]string, []byte) {
	if handler, ok := activeProvider.(TransformResponseHeadersHandler); ok {
		handler.TransformResponseHeaders(ctx, apiName, headers, log)
	} else {
		c.DefaultTransformResponseHeaders(ctx, headers)
	}

	if handler, ok := activeProvider.(TransformResponseBodyHandler); ok {
		var err error
		body, err = handler.TransformResponseBody(ctx, apiName, body, log)
		if err != nil {
			log.Errorf("Failed to transform response body: %v", err)
		}
	}

	return util.HeaderToSlice(headers), body
}

func (c *ProviderConfig) retryCall(
	ctx wrapper.HttpContext, log wrapper.Log, activeProvider Provider,
	apiName ApiName, statusCode int, responseHeaders http.Header, responseBody []byte,
	retryClient *wrapper.ClusterClient[wrapper.RouteCluster]) {

	retryCount := ctx.GetContext(ctxRetryCount).(int)
	log.Debugf("Sended retry request: %d/%d", retryCount, c.retryOnFailure.maxRetries)

	if statusCode == 200 {
		log.Debugf("Retry request succeeded")
		headers, body := c.transformResponseHeadersAndBody(ctx, activeProvider, apiName, responseHeaders, responseBody, log)
		proxywasm.SendHttpResponse(200, headers, body, -1)
	} else {
		log.Debugf("The retry request still failed, status: %d, responseHeaders: %v, responseBody: %s", statusCode, responseHeaders, string(responseBody))
	}

	retryCount++
	if retryCount <= int(c.retryOnFailure.maxRetries) {
		ctx.SetContext(ctxRetryCount, retryCount)
		c.sendRetryRequest(ctx, apiName, activeProvider, retryClient, log)
	} else {
		log.Debugf("Reached the maximum retry count: %d", c.retryOnFailure.maxRetries)
		proxywasm.ResumeHttpResponse()
	}
}

func (c *ProviderConfig) sendRetryRequest(
	ctx wrapper.HttpContext, apiName ApiName, activeProvider Provider,
	retryClient *wrapper.ClusterClient[wrapper.RouteCluster], log wrapper.Log) {

	requestHeaders, requestBody := c.getRetryRequestHeadersAndBody(ctx, activeProvider, apiName, log)
	path := getRetryPath(ctx)

	err := retryClient.Post(path, util.HeaderToSlice(requestHeaders), requestBody, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		c.retryCall(ctx, log, activeProvider, apiName, statusCode, responseHeaders, responseBody, retryClient)
	}, uint32(c.retryOnFailure.retryTimeout))
	if err != nil {
		log.Errorf("Failed to send retry request: %v", err)
		proxywasm.ResumeHttpResponse()
	}
}

func createRetryClient(ctx wrapper.HttpContext) *wrapper.ClusterClient[wrapper.RouteCluster] {
	host := wrapper.GetRequestHost()
	if host == "" {
		host = ctx.GetContext(ctxRequestHost).(string)
	}
	retryClient := wrapper.NewClusterClient(wrapper.RouteCluster{
		Host: host,
	})
	return retryClient
}

func getRetryPath(ctx wrapper.HttpContext) string {
	path := wrapper.GetRequestPath()
	if path == "" {
		path = ctx.GetContext(ctxRequestPath).(string)
	}
	return path
}

func (c *ProviderConfig) getRetryRequestHeadersAndBody(ctx wrapper.HttpContext, activeProvider Provider, apiName ApiName, log wrapper.Log) (http.Header, []byte) {
	// The retry request may be sent with different apiToken, so the header needs to be regenerated
	c.SetApiTokenInUse(ctx, log)

	requestHeaders := http.Header{
		"Content-Type": []string{"application/json"},
	}
	if handler, ok := activeProvider.(TransformRequestHeadersHandler); ok {
		handler.TransformRequestHeaders(ctx, apiName, requestHeaders, log)
	}
	requestBody := ctx.GetContext(ctxRequestBody).([]byte)

	return requestHeaders, requestBody
}
