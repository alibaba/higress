package main

import (
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	logs "github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		// 插件名称
		"ai-llm-router",
		// 为解析插件配置，设置自定义函数
		wrapper.ParseConfigBy(parseConfig),
		// 为处理请求头，设置自定义函数
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		// wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
		wrapper.ProcessResponseBody(onHttpResponseBody),
		// wrapper.ProcessStreamDone(onHttpStreamDone),
	)
}

// 自定义插件配置
type Config struct {
	mockEnable bool
}

// 在控制台插件配置中填写的yaml配置会自动转换为json，此处直接从json这个参数里解析配置即可
func parseConfig(json gjson.Result, config *Config, log logs.Log) error {
	// 解析出配置，更新到config中
	config.mockEnable = json.Get("mockEnable").Bool()
	return nil
}

// func onHttpRequestHeaders(ctx wrapper.HttpContext, config MyConfig, log logs.Log) types.Action {
// 	proxywasm.AddHttpRequestHeader("hello", "world")
// 	if config.mockEnable {
// 		proxywasm.SendHttpResponse(200, nil, []byte("hello world"), -1)
// 	}
// 	return types.HeaderContinue
// }

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log logs.Log) types.Action {
	ctx.DisableReroute()

	// 添加调试日志：检查wasm-go版本和host function支持情况
	log.Infof("=== 开始调试上游主机信息获取 ===")
	log.Infof("wasm-go版本信息准备检查...")

	// 获取上游主机信息
	log.Infof("尝试调用 proxywasm.GetUpstreamHosts()...")
	hostInfos, err := proxywasm.GetUpstreamHosts()
	if err != nil {
		return types.HeaderContinue
	}

	// 打印所有上游主机信息
	log.Infof("路由上游主机列表 (共 %d 个):", len(hostInfos))
	// 直接打印完整的主机信息数组
	log.Infof("上游主机详细信息: %+v", hostInfos)

	// 同时打印每个主机的详细信息
	for i, hostInfo := range hostInfos {
		log.Infof("  [%d] %+v", i+1, hostInfo)
	}

	return types.HeaderContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config Config, body []byte) types.Action {
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config Config) types.Action {
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config Config, body []byte) types.Action {
	return types.ActionContinue
}
