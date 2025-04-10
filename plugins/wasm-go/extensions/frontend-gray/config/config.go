package config

import (
	"strings"

	"github.com/tidwall/gjson"
)

const (
	XHigressTag     = "x-higress-tag"
	XUniqueClientId = "x-unique-client"
	XPreHigressTag  = "x-pre-higress-tag"
	IsPageRequest   = "is-page-request"
	IsNotFound      = "is-not-found"
	EnabledGray     = "enabled-gray"
	SecFetchMode    = "sec-fetch-mode"
)

type LogInfo func(format string, args ...interface{})

type GrayRule struct {
	Name         string
	GrayKeyValue []string
	GrayTagKey   string
	GrayTagValue []string
}

type Deployment struct {
	Name              string
	Enabled           bool
	Version           string
	BackendVersion    string
	Weight            int
	VersionPredicates map[string]string
}

type Rewrite struct {
	Host     string
	NotFound string
	Index    map[string]string
	File     map[string]string
}

type Injection struct {
	Head []string
	Body *BodyInjection
}

type BodyInjection struct {
	First []string
	Last  []string
}

type GrayConfig struct {
	UserStickyMaxAge    string
	TotalGrayWeight     int
	GrayKey             string
	LocalStorageGrayKey string
	GraySubKey          string
	Rules               []*GrayRule
	Rewrite             *Rewrite
	Html                string
	BaseDeployment      *Deployment
	GrayDeployments     []*Deployment
	BackendGrayTag      string
	Injection           *Injection
	SkippedPathPrefixes []string
	IncludePathPrefixes []string
	SkippedByHeaders    map[string]string
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
	grayConfig.LocalStorageGrayKey = json.Get("localStorageGrayKey").String()
	grayConfig.GrayKey = json.Get("grayKey").String()
	if grayConfig.LocalStorageGrayKey != "" {
		grayConfig.GrayKey = grayConfig.LocalStorageGrayKey
	}
	grayConfig.GraySubKey = json.Get("graySubKey").String()
	grayConfig.BackendGrayTag = json.Get("backendGrayTag").String()
	grayConfig.UserStickyMaxAge = json.Get("userStickyMaxAge").String()
	grayConfig.Html = json.Get("html").String()
	grayConfig.SkippedPathPrefixes = convertToStringList(json.Get("skippedPathPrefixes").Array())
	grayConfig.SkippedByHeaders = convertToStringMap(json.Get("skippedByHeaders"))
	grayConfig.IncludePathPrefixes = convertToStringList(json.Get("includePathPrefixes").Array())

	if grayConfig.UserStickyMaxAge == "" {
		// 默认值2天
		grayConfig.UserStickyMaxAge = "172800"
	}

	if grayConfig.BackendGrayTag == "" {
		grayConfig.BackendGrayTag = "x-mse-tag"
	}

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

	grayConfig.BaseDeployment = &Deployment{
		Name:              baseDeployment.Get("name").String(),
		Version:           strings.Trim(baseDeployment.Get("version").String(), " "),
		VersionPredicates: convertToStringMap(baseDeployment.Get("versionPredicates")),
	}
	for _, item := range grayDeployments {
		if !item.Get("enabled").Bool() {
			continue
		}
		grayWeight := int(item.Get("weight").Int())
		grayConfig.GrayDeployments = append(grayConfig.GrayDeployments, &Deployment{
			Name:              item.Get("name").String(),
			Enabled:           item.Get("enabled").Bool(),
			Version:           strings.Trim(item.Get("version").String(), " "),
			BackendVersion:    item.Get("backendVersion").String(),
			Weight:            grayWeight,
			VersionPredicates: convertToStringMap(item.Get("versionPredicates")),
		})
		grayConfig.TotalGrayWeight += grayWeight
	}

	grayConfig.Injection = &Injection{
		Head: convertToStringList(json.Get("injection.head").Array()),
		Body: &BodyInjection{
			First: convertToStringList(json.Get("injection.body.first").Array()),
			Last:  convertToStringList(json.Get("injection.body.last").Array()),
		},
	}
}
