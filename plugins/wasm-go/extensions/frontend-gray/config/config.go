package config

import (
	"github.com/tidwall/gjson"
)

const (
	XHigressTag    = "x-higress-tag"
	XPreHigressTag = "x-pre-higress-tag"
	XMseTag        = "x-mse-tag"
	IsHTML         = "is_html"
	IsIndex        = "is_index"
	NotFound       = "not_found"
)

type LogInfo func(format string, args ...interface{})

type GrayRule struct {
	Name         string
	GrayKeyValue []string
	GrayTagKey   string
	GrayTagValue []string
}

type BaseDeployment struct {
	Name    string
	Version string
}

type GrayDeployment struct {
	Name           string
	Enabled        bool
	Version        string
	BackendVersion string
}

type Rewrite struct {
	Host     string
	NotFound string
	Index    map[string]string
	File     map[string]string
}

type GrayConfig struct {
	GrayKey         string
	GraySubKey      string
	Rules           []*GrayRule
	Rewrite         *Rewrite
	BaseDeployment  *BaseDeployment
	GrayDeployments []*GrayDeployment
}

func convertToStringList(results []gjson.Result) []string {
	interfaces := make([]string, len(results)) // 预分配切片容量
	for i, result := range results {
		interfaces[i] = result.String() // 使用 String() 方法直接获取字符串
	}
	return interfaces
}

func convertToStringMap(result gjson.Result) map[string]string {
	m := make(map[string]string)
	result.ForEach(func(key, value gjson.Result) bool {
		m[key.String()] = value.String()
		return true // keep iterating
	})
	return m
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
			GrayKeyValue: convertToStringList(rule.Get("grayKeyValue").Array()),
			GrayTagKey:   rule.Get("grayTagKey").String(),
			GrayTagValue: convertToStringList(rule.Get("grayTagValue").Array()),
		}
		grayConfig.Rules = append(grayConfig.Rules, &grayRule)
	}
	grayConfig.Rewrite = &Rewrite{
		Host:     json.Get("rewrite.host").String(),
		NotFound: json.Get("rewrite.notFoundUri").String(),
		Index:    convertToStringMap(json.Get("rewrite.indexRouting")),
		File:     convertToStringMap(json.Get("rewrite.fileRouting")),
	}

	// 解析 deployment
	baseDeployment := json.Get("baseDeployment")
	grayDeployments := json.Get("grayDeployments").Array()

	grayConfig.BaseDeployment = &BaseDeployment{
		Name:    baseDeployment.Get("name").String(),
		Version: baseDeployment.Get("version").String(),
	}
	for _, item := range grayDeployments {
		grayConfig.GrayDeployments = append(grayConfig.GrayDeployments, &GrayDeployment{
			Name:           item.Get("name").String(),
			Enabled:        item.Get("enabled").Bool(),
			Version:        item.Get("version").String(),
			BackendVersion: item.Get("backendVersion").String(),
		})
	}
}
