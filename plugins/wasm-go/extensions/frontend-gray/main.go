package main

import (
	"frontend-gray/config"
	"frontend-gray/util"

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

// OmitGrayRule 过滤灰度规则
func OmitGrayRule(grayConfig *config.GrayConfig, grayKey string) *config.DeployItem {
	for _, grayItem := range grayConfig.Deploy.Gray {
		if !grayItem.Enable {
			// 跳过Enable=false
			continue
		}
		grayRule := util.ContainsRule(grayConfig.Rules, grayItem.Name)
		// 首先：先校验用户名单ID
		if grayRule.GrayKeyValue != nil && len(grayRule.GrayKeyValue) > 0 && grayKey != "" {
			if util.Contains(grayRule.GrayKeyValue, grayKey) {
				return grayItem
			}
		}
		//	第二：校验Cookie中的 GrayTagKey
		if grayRule.GrayTagKey != "" && grayRule.GrayTagValue != nil && len(grayRule.GrayTagValue) > 0 {
			cookieStr, _ := proxywasm.GetHttpRequestHeader("cookie")
			grayTagValue := util.GetValueByCookie(cookieStr, grayRule.GrayTagKey)
			if util.Contains(grayRule.GrayTagValue, grayTagValue) {
				return grayItem
			}
		}
	}
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
		subKeyValue := util.GetFromSubKey(grayKeyValue, grayConfig.GraySubKey)
		if subKeyValue != "" {
			grayKeyValue = subKeyValue
		}
	}
	grayDeployItem := OmitGrayRule(&grayConfig, grayKeyValue)
	if grayDeployItem != nil {
		log.Infof("x-mse-tag: %s, grayKey: %s", grayDeployItem.Version, grayKeyValue)
		proxywasm.AddHttpRequestHeader("x-mse-tag", grayDeployItem.Version)
	} else {
		log.Infof("x-mse-tag: %s, grayKey: %s", grayConfig.Deploy.Base.Version, grayKeyValue)
	}

	return types.ActionContinue
}
