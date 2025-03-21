package config

import (
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	XHigressTag       = "x-higress-tag"
	PreHigressVersion = "pre-higress-version"
	HigressUniqueId   = "higress-unique-id"
	IsHtmlRequest     = "is-html-request"
	IsIndexRequest    = "is-index-request"
	EnabledGray       = "enabled-gray"
)

type LogInfo func(format string, args ...interface{})

type HigressTagCookie struct {
	FrontendVersion string
	UniqueId        string
}

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
	Host  string
	Index map[string]string
	File  map[string]string
}

type Injection struct {
	GlobalConfig *GlobalConfig
	Head         []string
	Body         *BodyInjection
}

type GlobalConfig struct {
	Key        string
	FeatureKey string
	Value      string
	Enabled    bool
}

type BodyInjection struct {
	First []string
	Last  []string
}

type GrayConfig struct {
	UserStickyMaxAge    string
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
	SkippedPaths        []string
	SkippedByHeaders    map[string]string
	IndexPaths          []string
	GrayWeight          int
}

func GetWithDefault(json gjson.Result, path, defaultValue string) string {
	res := json.Get(path)
	if res.Exists() {
		return res.String()
	}
	return defaultValue
}

func convertToStringList(results []gjson.Result) []string {
	interfaces := make([]string, len(results)) // 预分配切片容量
	for i, result := range results {
		interfaces[i] = result.String() // 使用 String() 方法直接获取字符串
	}
	return interfaces
}

func compatibleConvertToStringList(results []gjson.Result, compatibleResults []gjson.Result) []string {
	// 优先使用兼容模式的数据
	if len(compatibleResults) == 0 {
		interfaces := make([]string, len(results)) // 预分配切片容量
		for i, result := range results {
			interfaces[i] = result.String() // 使用 String() 方法直接获取字符串
		}
		return interfaces
	}
	compatibleInterfaces := make([]string, len(compatibleResults)) // 预分配切片容量
	for i, result := range compatibleResults {
		compatibleInterfaces[i] = filepath.Join(result.String(), "**")
	}
	return compatibleInterfaces
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
	grayConfig.SkippedPaths = compatibleConvertToStringList(json.Get("skippedPaths").Array(), json.Get("skippedPathPrefixes").Array())
	grayConfig.IndexPaths = compatibleConvertToStringList(json.Get("indexPaths").Array(), json.Get("includePathPrefixes").Array())
	grayConfig.SkippedByHeaders = convertToStringMap(json.Get("skippedByHeaders"))

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
		Host:  json.Get("rewrite.host").String(),
		Index: convertToStringMap(json.Get("rewrite.indexRouting")),
		File:  convertToStringMap(json.Get("rewrite.fileRouting")),
	}

	// 解析 deployment
	baseDeployment := json.Get("baseDeployment")
	grayDeployments := json.Get("grayDeployments").Array()

	grayConfig.BaseDeployment = &Deployment{
		Name:              baseDeployment.Get("name").String(),
		BackendVersion:    baseDeployment.Get("backendVersion").String(),
		Version:           strings.Trim(baseDeployment.Get("version").String(), " "),
		VersionPredicates: convertToStringMap(baseDeployment.Get("versionPredicates")),
	}
	for _, item := range grayDeployments {
		if !item.Get("enabled").Bool() {
			continue
		}
		weight := int(item.Get("weight").Int())
		grayConfig.GrayDeployments = append(grayConfig.GrayDeployments, &Deployment{
			Name:              item.Get("name").String(),
			Enabled:           item.Get("enabled").Bool(),
			Version:           strings.Trim(item.Get("version").String(), " "),
			BackendVersion:    item.Get("backendVersion").String(),
			Weight:            weight,
			VersionPredicates: convertToStringMap(item.Get("versionPredicates")),
		})
		if weight > 0 {
			grayConfig.GrayWeight = weight
			break
		}
	}

	grayConfig.Injection = &Injection{
		Head: convertToStringList(json.Get("injection.head").Array()),
		Body: &BodyInjection{
			First: convertToStringList(json.Get("injection.body.first").Array()),
			Last:  convertToStringList(json.Get("injection.body.last").Array()),
		},
		GlobalConfig: &GlobalConfig{
			FeatureKey: GetWithDefault(json, "injection.globalConfig.featureKey", "FEATURE_STATUS"),
			Key:        GetWithDefault(json, "injection.globalConfig.key", "HIGRESS_CONSOLE_CONFIG"),
			Value:      json.Get("injection.globalConfig.value").String(),
			Enabled:    json.Get("injection.globalConfig.enabled").Bool(),
		},
	}
}
