package main

import (
	"user-gray/config"
	"user-gray/util"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"user-gray",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

func parseConfig(json gjson.Result, userGrayConfig *config.UserGrayConfig, log wrapper.Log) error {
	// 解析json 为UserGrayConfig
	config.JsonToUserGrayConfig(json, userGrayConfig)
	return nil
}

// OmitGrayRule 过滤灰度规则
func OmitGrayRule(userGrayConfig *config.UserGrayConfig, userId string) *config.DeployItem {
	for _, grayItem := range userGrayConfig.Deploy.Gray {
		if !grayItem.Enable {
			// 跳过Enable=false
			continue
		}
		grayRule := util.ContainsRule(userGrayConfig.Rules, grayItem.Name)
		// 首先：先校验用户名单ID
		if grayRule.UidValue != nil && len(grayRule.UidValue) > 0 && userId != "" {
			if util.Contains(grayRule.UidValue, userId) {
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

func onHttpRequestHeaders(ctx wrapper.HttpContext, grayConfig config.UserGrayConfig, log wrapper.Log) types.Action {
	// 优先从cookie中获取，如果拿不到再从header中获取
	cookieStr, _ := proxywasm.GetHttpRequestHeader("cookie")
	uidHeaderKey, _ := proxywasm.GetHttpRequestHeader(grayConfig.UidKey)
	uidKeyValue := util.GetValueByCookie(cookieStr, grayConfig.UidKey)
	proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	// 优先从Cookie中获取，否则从header中获取
	if uidKeyValue == "" {
		uidKeyValue = uidHeaderKey
	}
	// 如果有子key, 尝试从子key中获取值
	if grayConfig.UidSubKey != "" {
		uidSubKeyValue := util.GetUidFromSubKey(uidKeyValue, grayConfig.UidSubKey)
		if uidSubKeyValue != "" {
			uidKeyValue = uidSubKeyValue
		}
	}
	grayDeployItem := OmitGrayRule(&grayConfig, uidKeyValue)
	if grayDeployItem != nil {
		log.Infof("x-mse-tag: %s, userCode: %s", grayDeployItem.Version, uidKeyValue)
		proxywasm.AddHttpRequestHeader("x-mse-tag", grayDeployItem.Version)
	} else {
		log.Infof("x-mse-tag: %s, userCode: %s", grayConfig.Deploy.Base.Version, uidKeyValue)
	}

	return types.ActionContinue
}
