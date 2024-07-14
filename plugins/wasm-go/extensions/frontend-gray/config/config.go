package config

import (
	"strconv"

	"github.com/tidwall/gjson"
)

type GrayRule struct {
	Name         string
	GrayKeyValue []interface{}
	GrayTagKey   string
	GrayTagValue []interface{}
}

type BaseDeployment struct {
	Name    string
	Version string
}

type GrayDeployments struct {
	Name    string
	Version string
	Enabled bool
}

type GrayConfig struct {
	GrayKey         string
	GraySubKey      string
	Rules           []*GrayRule
	BaseDeployment  *BaseDeployment
	GrayDeployments []*GrayDeployments
}

func interfacesFromJSONResult(results []gjson.Result) []interface{} {
	var interfaces []interface{}
	for _, result := range results {
		switch v := result.Value().(type) {
		case float64:
			// 当 v 是 float64 时，将其转换为字符串
			interfaces = append(interfaces, strconv.FormatFloat(v, 'f', -1, 64))
		default:
			// 其它类型不改变，直接追加
			interfaces = append(interfaces, v)
		}
	}
	return interfaces
}

func JsonToGrayConfig(json gjson.Result, grayConfig *GrayConfig) {
	// 解析 GrayKey
	grayConfig.GrayKey = json.Get("grayKey").String()
	grayConfig.GraySubKey = json.Get("graySubKey").String()

	// 解析 Rules
	rules := json.Get("rules").Array()
	for _, rule := range rules {
		grayRule := GrayRule{
			Name:         rule.Get("name").String(),
			GrayKeyValue: interfacesFromJSONResult(rule.Get("grayKeyValue").Array()), // 使用辅助函数将 []gjson.Result 转换为 []interface{}
			GrayTagKey:   rule.Get("grayTagKey").String(),
			GrayTagValue: interfacesFromJSONResult(rule.Get("grayTagValue").Array()),
		}
		grayConfig.Rules = append(grayConfig.Rules, &grayRule)
	}

	// 解析 deploy
	baseDeployment := json.Get("baseDeployment")
	grayDeployments := json.Get("grayDeployments").Array()

	grayConfig.BaseDeployment = &BaseDeployment{
		Name:    baseDeployment.Get("name").String(),
		Version: baseDeployment.Get("version").String(),
	}
	for _, item := range grayDeployments {
		grayConfig.GrayDeployments = append(grayConfig.GrayDeployments, &GrayDeployments{
			Name:    item.Get("name").String(),
			Version: item.Get("version").String(),
			Enabled: item.Get("enabled").Bool(),
		})
	}
}
