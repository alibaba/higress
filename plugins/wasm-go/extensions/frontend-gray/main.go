package main

import (
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
	)
}

func parseConfig(json gjson.Result, grayConfig *config.GrayConfig, log wrapper.Log) error {
	// 解析json 为GrayConfig
	config.JsonToGrayConfig(json, grayConfig)
	return nil
}

// FilterGrayRule 过滤灰度规则
func FilterGrayRule(grayConfig *config.GrayConfig, grayKeyValue string, log wrapper.Log) *config.GrayDeployments {
	for _, grayDeployment := range grayConfig.GrayDeployments {
		if !grayDeployment.Enabled {
			// 跳过Enabled=false
			continue
		}
		grayRule := util.GetRule(grayConfig.Rules, grayDeployment.Name)
		// 首先：先校验用户名单ID
		if grayRule.GrayKeyValue != nil && len(grayRule.GrayKeyValue) > 0 && grayKeyValue != "" {
			if util.Contains(grayRule.GrayKeyValue, grayKeyValue) {
				log.Infof("x-mse-tag: %s, grayKeyValue: %s", grayDeployment.Version, grayKeyValue)
				return grayDeployment
			}
		}
		//	第二：校验Cookie中的 GrayTagKey
		if grayRule.GrayTagKey != "" && grayRule.GrayTagValue != nil && len(grayRule.GrayTagValue) > 0 {
			cookieStr, _ := proxywasm.GetHttpRequestHeader("cookie")
			grayTagValue := util.GetValueByCookie(cookieStr, grayRule.GrayTagKey)
			if util.Contains(grayRule.GrayTagValue, grayTagValue) {
				log.Infof("x-mse-tag: %s, grayTag: %s=%s", grayDeployment.Version, grayRule.GrayTagKey, grayTagValue)
				return grayDeployment
			}
		}
	}
	log.Infof("x-mse-tag: %s, grayKeyValue: %s", grayConfig.BaseDeployment.Version, grayKeyValue)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, grayConfig config.GrayConfig, log wrapper.Log) types.Action {
	// 优先从cookie中获取，如果拿不到再从header中获取
	cookieStr, _ := proxywasm.GetHttpRequestHeader("cookie")
	grayHeaderKey, _ := proxywasm.GetHttpRequestHeader(grayConfig.GrayKey)
	grayKeyValue := util.GetValueByCookie(cookieStr, grayConfig.GrayKey)
	proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	// 优先从Cookie中获取，否则从header中获取
	if grayKeyValue == "" {
		grayKeyValue = grayHeaderKey
	}
	// 如果有子key, 尝试从子key中获取值
	if grayConfig.GraySubKey != "" {
		subKeyValue := util.GetBySubKey(grayKeyValue, grayConfig.GraySubKey)
		if subKeyValue != "" {
			grayKeyValue = subKeyValue
		}
	}
	grayDeployment := FilterGrayRule(&grayConfig, grayKeyValue, log)
	if grayDeployment != nil {
		proxywasm.AddHttpRequestHeader("x-mse-tag", grayDeployment.Version)
	}
	return types.ActionContinue
}
