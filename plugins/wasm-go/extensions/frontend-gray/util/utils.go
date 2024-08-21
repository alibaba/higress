package util

import (
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/frontend-gray/config"

	"github.com/tidwall/gjson"
)

func IsGrayEnabled(grayConfig config.GrayConfig) bool {
	// 检查是否存在重写主机
	if grayConfig.Rewrite != nil && grayConfig.Rewrite.Host != "" {
		return true
	}

	// 检查灰度部署是否为 nil 或空
	grayDeployments := grayConfig.GrayDeployments
	if grayDeployments != nil && len(grayDeployments) > 0 {
		for _, grayDeployment := range grayDeployments {
			if grayDeployment.Enabled {
				return true
			}
		}
	}

	return false
}

// ExtractCookieValueByKey 根据 cookie 和 key 获取 cookie 值
func ExtractCookieValueByKey(cookie string, key string) string {
	if cookie == "" {
		return ""
	}
	value := ""
	pairs := strings.Split(cookie, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		kv := strings.Split(pair, "=")
		if kv[0] == key {
			value = kv[1]
			break
		}
	}
	return value
}

func ContainsValue(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// headers: [][2]string -> map[string][]string
func ConvertHeaders(hs [][2]string) map[string][]string {
	ret := make(map[string][]string)
	for _, h := range hs {
		k, v := strings.ToLower(h[0]), h[1]
		ret[k] = append(ret[k], v)
	}
	return ret
}

// headers: map[string][]string -> [][2]string
func ReconvertHeaders(hs map[string][]string) [][2]string {
	var ret [][2]string
	for k, vs := range hs {
		for _, v := range vs {
			ret = append(ret, [2]string{k, v})
		}
	}
	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i][0] < ret[j][0]
	})
	return ret
}

func GetRule(rules []*config.GrayRule, name string) *config.GrayRule {
	for _, rule := range rules {
		if rule.Name == name {
			return rule
		}
	}
	return nil
}

// 检查是否是页面
var indexSuffixes = []string{
	".html", ".htm", ".jsp", ".php", ".asp", ".aspx", ".erb", ".ejs", ".twig",
}

// IsIndexRequest determines if the request is an index request
func IsIndexRequest(fetchMode string, p string) bool {
	if fetchMode == "cors" {
		return false
	}
	ext := path.Ext(p)
	return ext == "" || ContainsValue(indexSuffixes, ext)
}

// 首页Rewrite
func IndexRewrite(path, version string, matchRules map[string]string) string {
	for prefix, rewrite := range matchRules {
		if strings.HasPrefix(path, prefix) {
			newPath := strings.Replace(rewrite, "{version}", version, -1)
			return newPath
		}
	}
	return path
}

func PrefixFileRewrite(path, version string, matchRules map[string]string) string {
	var matchedPrefix, replacement string
	for prefix, template := range matchRules {
		if strings.HasPrefix(path, prefix) {
			if len(prefix) > len(matchedPrefix) { // 找到更长的前缀
				matchedPrefix = prefix
				replacement = strings.Replace(template, "{version}", version, 1)
			}
		}
	}
	// 将path 中的前缀部分用 replacement 替换掉
	newPath := strings.Replace(path, matchedPrefix, replacement+"/", 1)
	return filepath.Clean(newPath)
}

func GetVersion(version string, cookies string, isIndex bool) string {
	if isIndex {
		return version
	}
	// 来自Cookie中的版本
	cookieVersion := ExtractCookieValueByKey(cookies, config.XPreHigressTag)
	// cookie 中为空，返回当前版本
	if cookieVersion == "" {
		return version
	}

	// cookie 中和当前版本不相同，返回cookie中值
	if cookieVersion != version {
		return cookieVersion
	}
	return version
}

// 从cookie中解析出灰度信息
func getBySubKey(grayInfoStr string, graySubKey string) string {
	// 首先对 URL 编码的字符串进行解码
	jsonStr, err := url.QueryUnescape(grayInfoStr)
	if err != nil {
		return ""
	}
	// 使用 gjson 从 JSON 字符串中提取 graySubKey 对应的值
	value := gjson.Get(jsonStr, graySubKey)

	// 检查所提取的值是否存在
	if !value.Exists() {
		return ""
	}
	// 返回字符串形式的值
	return value.String()
}

func GetGrayKey(grayKeyValue string, graySubKey string) string {
	// 如果有子key, 尝试从子key中获取值
	if graySubKey != "" {
		subKeyValue := getBySubKey(grayKeyValue, graySubKey)
		if subKeyValue != "" {
			grayKeyValue = subKeyValue
		}
	}
	return grayKeyValue
}

// FilterGrayRule 过滤灰度规则
func FilterGrayRule(grayConfig *config.GrayConfig, grayKeyValue string, logInfof func(format string, args ...interface{})) *config.GrayDeployment {
	for _, grayDeployment := range grayConfig.GrayDeployments {
		if !grayDeployment.Enabled {
			// 跳过Enabled=false
			continue
		}
		grayRule := GetRule(grayConfig.Rules, grayDeployment.Name)
		// 首先：先校验用户名单ID
		if grayRule.GrayKeyValue != nil && len(grayRule.GrayKeyValue) > 0 && grayKeyValue != "" {
			if ContainsValue(grayRule.GrayKeyValue, grayKeyValue) {
				logInfof("frontendVersion: %s, grayKeyValue: %s", grayDeployment.Version, grayKeyValue)
				return grayDeployment
			}
		}
		//	第二：校验Cookie中的 GrayTagKey
		if grayRule.GrayTagKey != "" && grayRule.GrayTagValue != nil && len(grayRule.GrayTagValue) > 0 {
			cookieStr, _ := proxywasm.GetHttpRequestHeader("cookie")
			grayTagValue := ExtractCookieValueByKey(cookieStr, grayRule.GrayTagKey)
			if ContainsValue(grayRule.GrayTagValue, grayTagValue) {
				logInfof("frontendVersion: %s, grayTag: %s=%s", grayDeployment.Version, grayRule.GrayTagKey, grayTagValue)
				return grayDeployment
			}
		}
	}
	logInfof("frontendVersion: %s, grayKeyValue: %s", grayConfig.BaseDeployment.Version, grayKeyValue)
	return nil
}
