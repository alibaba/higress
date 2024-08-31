package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/frontend-gray/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/frontend-gray/util"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"frontend-gray",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeader),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
		wrapper.ProcessStreamingResponseBodyBy(onStreamingResponseBody),
	)
}

func parseConfig(json gjson.Result, grayConfig *config.GrayConfig, log wrapper.Log) error {
	// 解析json 为GrayConfig
	config.JsonToGrayConfig(json, grayConfig)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, grayConfig config.GrayConfig, log wrapper.Log) types.Action {
	if !util.IsGrayEnabled(grayConfig) {
		return types.ActionContinue
	}

	cookies, _ := proxywasm.GetHttpRequestHeader("cookie")
	path, _ := proxywasm.GetHttpRequestHeader(":path")
	fetchMode, _ := proxywasm.GetHttpRequestHeader("sec-fetch-mode")

	isIndex := util.IsIndexRequest(fetchMode, path)
	hasRewrite := len(grayConfig.Rewrite.File) > 0 || len(grayConfig.Rewrite.Index) > 0
	grayKeyValue := util.GetGrayKey(util.ExtractCookieValueByKey(cookies, grayConfig.GrayKey), grayConfig.GraySubKey)

	// 如果有重写的配置，则进行重写
	if hasRewrite {
		// 禁止重新路由，要在更改Header之前操作，否则会失效
		ctx.DisableReroute()
	}

	// 删除Accept-Encoding，避免压缩， 如果是压缩的内容，后续插件就没法处理了
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	grayDeployment := util.FilterGrayRule(&grayConfig, grayKeyValue, log.Infof)
	frontendVersion := util.GetVersion(grayConfig.BaseDeployment.Version, cookies, isIndex)
	backendVersion := ""

	// 命中灰度规则
	if grayDeployment != nil {
		frontendVersion = util.GetVersion(grayDeployment.Version, cookies, isIndex)
		backendVersion = grayDeployment.BackendVersion
	}

	proxywasm.AddHttpRequestHeader(config.XHigressTag, frontendVersion)

	ctx.SetContext(config.XPreHigressTag, frontendVersion)
	ctx.SetContext(config.XMseTag, backendVersion)
	ctx.SetContext(config.IsIndex, isIndex)

	rewrite := grayConfig.Rewrite
	if rewrite.Host != "" {
		proxywasm.ReplaceHttpRequestHeader("HOST", rewrite.Host)
	}

	if hasRewrite {
		rewritePath := path
		if isIndex {
			rewritePath = util.IndexRewrite(path, frontendVersion, grayConfig.Rewrite.Index)
		} else {
			rewritePath = util.PrefixFileRewrite(path, frontendVersion, grayConfig.Rewrite.File)
		}
		log.Infof("rewrite path: %s %s %v", path, frontendVersion, rewritePath)
		proxywasm.ReplaceHttpRequestHeader(":path", rewritePath)
	}

	return types.ActionContinue
}

func onHttpResponseHeader(ctx wrapper.HttpContext, grayConfig config.GrayConfig, log wrapper.Log) types.Action {
	if !util.IsGrayEnabled(grayConfig) {
		return types.ActionContinue
	}
	status, err := proxywasm.GetHttpResponseHeader(":status")
	contentType, _ := proxywasm.GetHttpResponseHeader("Content-Type")
	if err != nil || status != "200" {
		isIndex := ctx.GetContext(config.IsIndex)
		if status == "404" {
			if grayConfig.Rewrite.NotFound != "" && isIndex != nil && isIndex.(bool) {
				ctx.SetContext(config.NotFound, true)
				responseHeaders, _ := proxywasm.GetHttpResponseHeaders()
				headersMap := util.ConvertHeaders(responseHeaders)
				headersMap[":status"][0] = "200"
				headersMap["content-type"][0] = "text/html"
				delete(headersMap, "content-length")
				proxywasm.ReplaceHttpResponseHeaders(util.ReconvertHeaders(headersMap))
				ctx.BufferResponseBody()
				return types.ActionContinue
			} else {
				ctx.DontReadResponseBody()
			}
		}
		log.Errorf("error status: %s, error message: %v", status, err)
		return types.ActionContinue
	}

	// 删除content-length，可能要修改Response返回值
	proxywasm.RemoveHttpResponseHeader("Content-Length")

	// 删除Content-Disposition，避免自动下载文件
	proxywasm.RemoveHttpResponseHeader("Content-Disposition")

	if strings.HasPrefix(contentType, "text/html") {
		ctx.SetContext(config.IsHTML, true)
		// 不会进去Streaming 的Body处理
		ctx.BufferResponseBody()

		// 添加Cache-Control 头部，禁止缓存
		proxywasm.ReplaceHttpRequestHeader("Cache-Control", "no-cache, no-store")

		frontendVersion := ctx.GetContext(config.XPreHigressTag).(string)
		backendVersion := ctx.GetContext(config.XMseTag).(string)

		// 设置当前的前端版本
		proxywasm.AddHttpResponseHeader("Set-Cookie", fmt.Sprintf("%s=%s; Path=/;", config.XPreHigressTag, frontendVersion))
		// 设置后端的前端版本
		proxywasm.AddHttpResponseHeader("Set-Cookie", fmt.Sprintf("%s=%s; Path=/;", config.XMseTag, backendVersion))
	}
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, grayConfig config.GrayConfig, body []byte, log wrapper.Log) types.Action {
	if !util.IsGrayEnabled(grayConfig) {
		return types.ActionContinue
	}
	backendVersion := ctx.GetContext(config.XMseTag)
	isHtml := ctx.GetContext(config.IsHTML)
	isIndex := ctx.GetContext(config.IsIndex)
	notFoundUri := ctx.GetContext(config.NotFound)
	if isIndex != nil && isIndex.(bool) && notFoundUri != nil && notFoundUri.(bool) && grayConfig.Rewrite.Host != "" && grayConfig.Rewrite.NotFound != "" {
		client := wrapper.NewClusterClient(wrapper.RouteCluster{Host: grayConfig.Rewrite.Host})
		client.Get(grayConfig.Rewrite.NotFound, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			proxywasm.ReplaceHttpResponseBody(responseBody)
			proxywasm.ResumeHttpResponse()
		}, 1500)
		return types.ActionPause
	}

	// 以text/html 开头，将 cookie转到cookie
	if isHtml != nil && isHtml.(bool) && backendVersion != nil && backendVersion.(string) != "" {
		newText := strings.ReplaceAll(string(body), "</head>", `<script>
				!function(e,t){function n(e){var n="; "+t.cookie,r=n.split("; "+e+"=");return 2===r.length?r.pop().split(";").shift():null}var r=n("x-mse-tag");if(!r)return null;var s=XMLHttpRequest.prototype.open;XMLHttpRequest.prototype.open=function(e,t,n,a,i){return this._XHR=!0,this.addEventListener("readystatechange",function(){1===this.readyState&&r&&this.setRequestHeader("x-mse-tag",r)}),s.apply(this,arguments)};var a=e.fetch;e.fetch=function(e,t){return"undefined"==typeof t&&(t={}),"undefined"==typeof t.headers&&(t.headers={}),r&&(t.headers["x-mse-tag"]=r),a.apply(this,[e,t])}}(window,document);
			</script>
		</head>`)
		if err := proxywasm.ReplaceHttpResponseBody([]byte(newText)); err != nil {
			return types.ActionContinue
		}
	}
	return types.ActionContinue
}

func onStreamingResponseBody(ctx wrapper.HttpContext, pluginConfig config.GrayConfig, chunk []byte, isLastChunk bool, log wrapper.Log) []byte {
	return chunk
}
