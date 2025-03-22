package main

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/frontend-gray/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/frontend-gray/util"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
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
	)
}

func parseConfig(json gjson.Result, grayConfig *config.GrayConfig, log log.Log) error {
	// 解析json 为GrayConfig
	config.JsonToGrayConfig(json, grayConfig)
	log.Infof("Rewrite: %v, GrayDeployments: %v", json.Get("rewrite"), json.Get("grayDeployments"))
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, grayConfig config.GrayConfig, log log.Log) types.Action {
	requestPath, _ := proxywasm.GetHttpRequestHeader(":path")
	requestPath = path.Clean(requestPath)
	parsedURL, err := url.Parse(requestPath)
	if err == nil {
		requestPath = parsedURL.Path
	} else {
		log.Errorf("parse request path %s failed: %v", requestPath, err)
	}
	enabledGray := util.IsGrayEnabled(grayConfig, requestPath)
	ctx.SetContext(config.EnabledGray, enabledGray)
	secFetchMode, _ := proxywasm.GetHttpRequestHeader("sec-fetch-mode")
	ctx.SetContext(config.SecFetchMode, secFetchMode)

	if !enabledGray {
		log.Infof("gray not enabled")
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	cookies, _ := proxywasm.GetHttpRequestHeader("cookie")
	isPageRequest := util.IsPageRequest(requestPath)
	hasRewrite := len(grayConfig.Rewrite.File) > 0 || len(grayConfig.Rewrite.Index) > 0
	grayKeyValueByCookie := util.ExtractCookieValueByKey(cookies, grayConfig.GrayKey)
	grayKeyValueByHeader, _ := proxywasm.GetHttpRequestHeader(grayConfig.GrayKey)
	// 优先从cookie中获取，否则从header中获取
	grayKeyValue := util.GetGrayKey(grayKeyValueByCookie, grayKeyValueByHeader, grayConfig.GraySubKey)
	// 如果有重写的配置，则进行重写
	if hasRewrite {
		// 禁止重新路由，要在更改Header之前操作，否则会失效
		ctx.DisableReroute()
	}

	// 删除Accept-Encoding，避免压缩， 如果是压缩的内容，后续插件就没法处理了
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	deployment := &config.Deployment{}

	preVersion, preUniqueClientId := util.GetXPreHigressVersion(cookies)
	// 客户端唯一ID，用于在按照比率灰度时候 客户访问黏贴
	uniqueClientId := grayKeyValue
	if uniqueClientId == "" {
		xForwardedFor, _ := proxywasm.GetHttpRequestHeader("X-Forwarded-For")
		uniqueClientId = util.GetRealIpFromXff(xForwardedFor)
	}

	// 如果没有配置比例，则进行灰度规则匹配
	if util.IsSupportMultiVersion(grayConfig) {
		deployment = util.FilterMultiVersionGrayRule(&grayConfig, grayKeyValue, requestPath)
		log.Infof("multi version %v", deployment)
	} else {
		if isPageRequest {
			if grayConfig.TotalGrayWeight > 0 {
				log.Infof("grayConfig.TotalGrayWeight: %v", grayConfig.TotalGrayWeight)
				deployment = util.FilterGrayWeight(&grayConfig, preVersion, preUniqueClientId, uniqueClientId)
			} else {
				deployment = util.FilterGrayRule(&grayConfig, grayKeyValue)
			}
			log.Infof("index deployment: %v, path: %v, backend: %v, xPreHigressVersion: %s,%s", deployment, requestPath, deployment.BackendVersion, preVersion, preUniqueClientId)
		} else {
			grayDeployment := util.FilterGrayRule(&grayConfig, grayKeyValue)
			deployment = util.GetVersion(grayConfig, grayDeployment, preVersion, isPageRequest)
		}
		ctx.SetContext(config.XPreHigressTag, deployment.Version)
		ctx.SetContext(grayConfig.BackendGrayTag, deployment.BackendVersion)
	}

	proxywasm.AddHttpRequestHeader(config.XHigressTag, deployment.Version)

	ctx.SetContext(config.IsPageRequest, isPageRequest)
	ctx.SetContext(config.XUniqueClientId, uniqueClientId)

	rewrite := grayConfig.Rewrite
	if rewrite.Host != "" {
		err := proxywasm.ReplaceHttpRequestHeader(":authority", rewrite.Host)
		if err != nil {
			log.Errorf("host rewrite failed: %v", err)
		}
	}

	if hasRewrite {
		rewritePath := requestPath
		if isPageRequest {
			rewritePath = util.IndexRewrite(requestPath, deployment.Version, grayConfig.Rewrite.Index)
		} else {
			rewritePath = util.PrefixFileRewrite(requestPath, deployment.Version, grayConfig.Rewrite.File)
		}
		if requestPath != rewritePath {
			log.Infof("rewrite path:%s, rewritePath:%s, Version:%v", requestPath, rewritePath, deployment.Version)
			proxywasm.ReplaceHttpRequestHeader(":path", rewritePath)
		}
	}
	log.Infof("request path:%s, has rewrited:%v, rewrite config:%+v", requestPath, hasRewrite, rewrite)
	return types.ActionContinue
}

func onHttpResponseHeader(ctx wrapper.HttpContext, grayConfig config.GrayConfig, log log.Log) types.Action {
	enabledGray, _ := ctx.GetContext(config.EnabledGray).(bool)
	if !enabledGray {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	secFetchMode, isSecFetchModeOk := ctx.GetContext(config.SecFetchMode).(string)
	if isSecFetchModeOk && secFetchMode == "cors" {
		proxywasm.ReplaceHttpResponseHeader("cache-control", "no-cache, no-store, max-age=0, must-revalidate")
	}
	isPageRequest, ok := ctx.GetContext(config.IsPageRequest).(bool)
	if !ok {
		isPageRequest = false // 默认值
	}
	// response 不处理非首页的请求
	if !isPageRequest {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	} else {
		// 不会进去Streaming 的Body处理
		ctx.BufferResponseBody()
	}

	status, err := proxywasm.GetHttpResponseHeader(":status")
	if grayConfig.Rewrite != nil && grayConfig.Rewrite.Host != "" {
		// 删除Content-Disposition，避免自动下载文件
		proxywasm.RemoveHttpResponseHeader("Content-Disposition")
	}

	// 删除content-length，可能要修改Response返回值
	proxywasm.RemoveHttpResponseHeader("Content-Length")

	// 处理code为 200的情况
	if err != nil || status != "200" {
		if status == "404" {
			if grayConfig.Rewrite.NotFound != "" && isPageRequest {
				ctx.SetContext(config.IsNotFound, true)
				responseHeaders, _ := proxywasm.GetHttpResponseHeaders()
				headersMap := util.ConvertHeaders(responseHeaders)
				if _, ok := headersMap[":status"]; !ok {
					headersMap[":status"] = []string{"200"} // 如果没有初始化，设定默认值
				} else {
					headersMap[":status"][0] = "200" // 修改现有值
				}
				if _, ok := headersMap["content-type"]; !ok {
					headersMap["content-type"] = []string{"text/html"} // 如果没有初始化，设定默认值
				} else {
					headersMap["content-type"][0] = "text/html" // 修改现有值
				}
				// 删除 content-length 键
				delete(headersMap, "content-length")
				proxywasm.ReplaceHttpResponseHeaders(util.ReconvertHeaders(headersMap))
				ctx.BufferResponseBody()
				return types.ActionContinue
			} else {
				// 直接返回400
				ctx.DontReadResponseBody()
			}
		}
		log.Errorf("error status: %s, error message: %v", status, err)
		return types.ActionContinue
	}
	cacheControl, _ := proxywasm.GetHttpResponseHeader("cache-control")
	if !strings.Contains(cacheControl, "no-cache") {
		proxywasm.ReplaceHttpResponseHeader("cache-control", "no-cache, no-store, max-age=0, must-revalidate")
	}

	frontendVersion, isFeVersionOk := ctx.GetContext(config.XPreHigressTag).(string)
	xUniqueClient, isUniqClientOk := ctx.GetContext(config.XUniqueClientId).(string)

	// 设置前端的版本
	if isFeVersionOk && isUniqClientOk && frontendVersion != "" {
		proxywasm.AddHttpResponseHeader("Set-Cookie", fmt.Sprintf("%s=%s,%s; Max-Age=%s; Path=/;", config.XPreHigressTag, frontendVersion, xUniqueClient, grayConfig.UserStickyMaxAge))
	}
	// 设置后端的版本
	if util.IsBackendGrayEnabled(grayConfig) {
		backendVersion, isBackVersionOk := ctx.GetContext(grayConfig.BackendGrayTag).(string)
		if isBackVersionOk && backendVersion != "" {
			proxywasm.AddHttpResponseHeader("Set-Cookie", fmt.Sprintf("%s=%s; Max-Age=%s; Path=/;", grayConfig.BackendGrayTag, backendVersion, grayConfig.UserStickyMaxAge))
		}
	}
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, grayConfig config.GrayConfig, body []byte, log log.Log) types.Action {
	enabledGray, _ := ctx.GetContext(config.EnabledGray).(bool)
	if !enabledGray {
		return types.ActionContinue
	}
	isPageRequest, isPageRequestOk := ctx.GetContext(config.IsPageRequest).(bool)
	frontendVersion, isFeVersionOk := ctx.GetContext(config.XPreHigressTag).(string)
	// 只处理首页相关请求
	if !isFeVersionOk || !isPageRequestOk || !isPageRequest {
		return types.ActionContinue
	}

	isNotFound, ok := ctx.GetContext(config.IsNotFound).(bool)
	if !ok {
		isNotFound = false // 默认值
	}

	// 检查是否存在自定义 HTML， 如有则省略 rewrite.indexRouting 的内容
	if grayConfig.Html != "" {
		log.Debugf("Returning custom HTML from config.")
		// 替换响应体为 config.Html 内容
		if err := proxywasm.ReplaceHttpResponseBody([]byte(grayConfig.Html)); err != nil {
			log.Errorf("Error replacing response body: %v", err)
			return types.ActionContinue
		}

		newHtml := util.InjectContent(grayConfig.Html, grayConfig.Injection)
		// 替换当前html加载的动态文件版本
		newHtml = strings.ReplaceAll(newHtml, "{version}", frontendVersion)

		// 最终替换响应体
		if err := proxywasm.ReplaceHttpResponseBody([]byte(newHtml)); err != nil {
			log.Errorf("Error replacing injected response body: %v", err)
			return types.ActionContinue
		}

		return types.ActionContinue
	}

	// 针对404页面处理
	if isNotFound && grayConfig.Rewrite.Host != "" && grayConfig.Rewrite.NotFound != "" {
		client := wrapper.NewClusterClient(wrapper.RouteCluster{Host: grayConfig.Rewrite.Host})

		client.Get(strings.Replace(grayConfig.Rewrite.NotFound, "{version}", frontendVersion, -1), nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			proxywasm.ReplaceHttpResponseBody(responseBody)
			proxywasm.ResumeHttpResponse()
		}, 1500)
		return types.ActionPause
	}

	// 处理响应体HTML
	newBody := string(body)
	newBody = util.InjectContent(newBody, grayConfig.Injection)
	if grayConfig.LocalStorageGrayKey != "" {
		localStr := strings.ReplaceAll(`<script>
		!function(){var o,e,n="@@X_GRAY_KEY",t=document.cookie.split("; ").filter(function(o){return 0===o.indexOf(n+"=")});try{"undefined"!=typeof localStorage&&null!==localStorage&&(o=localStorage.getItem(n),e=0<t.length?decodeURIComponent(t[0].split("=")[1]):null,o)&&o.indexOf("=")<0&&e&&e!==o&&(document.cookie=n+"="+encodeURIComponent(o)+"; path=/;",window.location.reload())}catch(o){}}();
		</script>
		`, "@@X_GRAY_KEY", grayConfig.LocalStorageGrayKey)
		newBody = strings.ReplaceAll(newBody, "<body>", "<body>\n"+localStr)
	}
	if err := proxywasm.ReplaceHttpResponseBody([]byte(newBody)); err != nil {
		return types.ActionContinue
	}
	return types.ActionContinue
}
