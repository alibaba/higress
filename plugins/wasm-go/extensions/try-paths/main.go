package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

// 默认超时时间为60s
var defaultTimeout uint32 = 60000

const VARSTR = "$uri"

func main() {
	wrapper.SetCtx(
		"try-paths",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

// 自定义插件配置
type TryPathsConfig struct {
	tryPaths []string
	code     []int // 支持多个返回码
	client   wrapper.HttpClient
}

func Client(json gjson.Result) (wrapper.HttpClient, error) {
	serviceSource := json.Get("serviceSource").String()
	serviceName := json.Get("serviceName").String()
	host := json.Get("host").String()
	servicePort := json.Get("servicePort").Int()
	if serviceName == "" || servicePort == 0 {
		return nil, errors.New("invalid service config")
	}
	switch serviceSource {
	case "k8s":
		namespace := json.Get("namespace").String()
		return wrapper.NewClusterClient(wrapper.K8sCluster{
			ServiceName: serviceName,
			Namespace:   namespace,
			Port:        servicePort,
			Host:        host,
		}), nil
	case "nacos":
		namespace := json.Get("namespace").String()
		return wrapper.NewClusterClient(wrapper.NacosCluster{
			ServiceName: serviceName,
			NamespaceID: namespace,
			Port:        servicePort,
			Host:        host,
		}), nil
	case "ip":
		return wrapper.NewClusterClient(wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Host:        host,
		}), nil
	case "dns":
		domain := json.Get("domain").String()
		return wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      domain,
		}), nil
	default:
		return nil, errors.New("unknown service source: " + serviceSource)
	}
}

func parseConfig(json gjson.Result, config *TryPathsConfig, log wrapper.Log) error {
	// 解析出配置，更新到config中
	for _, result := range json.Get("tryPaths").Array() {
		config.tryPaths = append(config.tryPaths, result.String())
	}

	// code默认值为["404", "403"]
	if json.Get("code").String() == "" {
		config.code = []int{http.StatusNotFound, http.StatusForbidden}
	} else {
		for _, result := range json.Get("code").Array() {
			config.code = append(config.code, int(result.Int()))
		}
	}
	client, err := Client(json)
	if err != nil {
		return err
	}
	config.client = client
	return nil
}

func convHttpHeadersToStruct(responseHeaders http.Header) [][2]string {
	headerStruct := make([][2]string, len(responseHeaders))
	i := 0
	for key, values := range responseHeaders {
		headerStruct[i][0] = key
		headerStruct[i][1] = values[0]
		i++
	}
	return headerStruct
}

func included(array []int, value int) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}

func tryHttpCallback(ctx wrapper.HttpContext, config TryPathsConfig, length int, index int, path string, log wrapper.Log) {
	if length == index {
		proxywasm.ResumeHttpRequest()
		return
	}
	requestPath := strings.Replace(config.tryPaths[index], VARSTR, path, -1)
	config.client.Get(requestPath, nil,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if !included(config.code, statusCode) {
				proxywasm.SendHttpResponse(uint32(statusCode), convHttpHeadersToStruct(responseHeaders), responseBody, -1)
				return
			}
			tryHttpCallback(ctx, config, length, index+1, path, log)
		}, defaultTimeout)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config TryPathsConfig, log wrapper.Log) types.Action {
	path := ctx.Path()
	if len(config.tryPaths) == 0 {
		return types.ActionContinue
	}

	tryHttpCallback(ctx, config, len(config.tryPaths), 0, path, log)
	// 需要等待异步回调完成，返回Pause状态，可以被ResumeHttpRequest恢复
	return types.ActionPause
}
