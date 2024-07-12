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

type DeployItem struct {
	Name    string
	Version string
	Enable  bool
}

type Deploy struct {
	Base *DeployItem
	Gray []*DeployItem
}

type GrayConfig struct {
	GrayKey    string
	GraySubKey string
	Rules      []*GrayRule
	Deploy     *Deploy
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
	grayConfig.GrayKey = json.Get("gray-key").String()
	grayConfig.GraySubKey = json.Get("gray-sub-key").String()

	// 解析 Rules
	rules := json.Get("rules").Array()
	for _, rule := range rules {
		grayRule := GrayRule{
			Name:         rule.Get("name").String(),
			GrayKeyValue: interfacesFromJSONResult(rule.Get("gray-key-value").Array()), // 使用辅助函数将 []gjson.Result 转换为 []interface{}
			GrayTagKey:   rule.Get("gray-tag-key").String(),
			GrayTagValue: interfacesFromJSONResult(rule.Get("gray-tag-value").Array()),
		}
		grayConfig.Rules = append(grayConfig.Rules, &grayRule)
	}

	// 解析 deploy
	deployJSON := json.Get("deploy")
	baseItem := deployJSON.Get("base")
	grayItems := deployJSON.Get("gray").Array()

	// 分配内存给 release 对象
	grayConfig.Deploy = &Deploy{
		Base: &DeployItem{
			Name:    baseItem.Get("name").String(),
			Version: baseItem.Get("version").String(),
			Enable:  baseItem.Get("enable").Bool(),
		},
		Gray: []*DeployItem{},
	}

	// 解析 Gray 列表
	for _, item := range grayItems {
		DeployItem := &DeployItem{
			Name:    item.Get("name").String(),
			Version: item.Get("version").String(),
			Enable:  item.Get("enable").Bool(),
		}
		grayConfig.Deploy.Gray = append(grayConfig.Deploy.Gray, DeployItem)
	}
}
