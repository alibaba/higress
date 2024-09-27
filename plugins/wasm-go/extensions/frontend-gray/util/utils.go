package util

import (
	"fmt"
	"math/rand"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/frontend-gray/config"

	"github.com/tidwall/gjson"
)

func LogInfof(format string, args ...interface{}) {
	format = fmt.Sprintf("[%s] %s", "frontend-gray", format)
	proxywasm.LogInfof(format, args...)
}

func GetXPreHigressVersion(cookies string) (string, string) {
	xPreHigressVersion := ExtractCookieValueByKey(cookies, config.XPreHigressTag)
	preVersions := strings.Split(xPreHigressVersion, ",")
	if len(preVersions) == 0 {
		return "", ""
	}
	if len(preVersions) == 1 {
		return preVersions[0], ""
	}

	return strings.TrimSpace(preVersions[0]), strings.TrimSpace(preVersions[1])
}

// 从xff中获取真实的IP
func GetRealIpFromXff(xff string) string {
	if xff != "" {
		// 通常客户端的真实 IP 是 XFF 头中的第一个 IP
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	return ""
}

func IsGrayEnabled(grayConfig config.GrayConfig) bool {
	// 检查是否存在重写主机
	if grayConfig.Rewrite != nil && grayConfig.Rewrite.Host != "" {
		return true
	}

	// 检查是否存在灰度版本配置
	return len(grayConfig.GrayDeployments) > 0
}

// 是否启用后端的灰度（全链路灰度）
func IsBackendGrayEnabled(grayConfig config.GrayConfig) bool {
	for _, deployment := range grayConfig.GrayDeployments {
		if deployment.BackendVersion != "" {
			return true
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

func IsPageRequest(fetchMode string, myPath string) bool {
	if fetchMode == "cors" {
		return false
	}
	ext := path.Ext(myPath)
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

func GetVersion(grayConfig config.GrayConfig, deployment *config.Deployment, xPreHigressVersion string, isPageRequest bool) *config.Deployment {
	if isPageRequest {
		return deployment
	}
	// cookie 中为空，返回当前版本
	if xPreHigressVersion == "" {
		return deployment
	}

	// cookie 中和当前版本不相同，返回cookie中值
	if xPreHigressVersion != deployment.Version {
		deployments := append(grayConfig.GrayDeployments, grayConfig.BaseDeployment)
		for _, curDeployment := range deployments {
			if curDeployment.Version == xPreHigressVersion {
				return curDeployment
			}
		}
	}
	return deployment
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

func GetGrayKey(grayKeyValueByCookie string, grayKeyValueByHeader string, graySubKey string) string {
	grayKeyValue := grayKeyValueByCookie
	if grayKeyValueByCookie == "" {
		grayKeyValue = grayKeyValueByHeader
	}

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
func FilterGrayRule(grayConfig *config.GrayConfig, grayKeyValue string) *config.Deployment {
	for _, deployment := range grayConfig.GrayDeployments {
		grayRule := GetRule(grayConfig.Rules, deployment.Name)
		// 首先：先校验用户名单ID
		if grayRule.GrayKeyValue != nil && len(grayRule.GrayKeyValue) > 0 && grayKeyValue != "" {
			if ContainsValue(grayRule.GrayKeyValue, grayKeyValue) {
				return deployment
			}
		}
		//	第二：校验Cookie中的 GrayTagKey
		if grayRule.GrayTagKey != "" && grayRule.GrayTagValue != nil && len(grayRule.GrayTagValue) > 0 {
			cookieStr, _ := proxywasm.GetHttpRequestHeader("cookie")
			grayTagValue := ExtractCookieValueByKey(cookieStr, grayRule.GrayTagKey)
			if ContainsValue(grayRule.GrayTagValue, grayTagValue) {
				return deployment
			}
		}
	}
	return grayConfig.BaseDeployment
}

func FilterGrayWeight(grayConfig *config.GrayConfig, preVersion string, preUniqueClientId string, uniqueClientId string) *config.Deployment {
	// 如果没有灰度权重，直接返回基础版本
	if grayConfig.TotalGrayWeight == 0 {
		return grayConfig.BaseDeployment
	}

	deployments := append(grayConfig.GrayDeployments, grayConfig.BaseDeployment)
	LogInfof("preVersion: %s, preUniqueClientId: %s, uniqueClientId: %s", preVersion, preUniqueClientId, uniqueClientId)
	// 用户粘滞，确保每个用户每次访问的都是走同一版本
	if preVersion != "" && uniqueClientId == preUniqueClientId {
		for _, deployment := range deployments {
			if deployment.Version == preVersion {
				return deployment
			}
		}
	}

	totalWeight := 100
	// 如果总权重小于100，则将基础版本也加入到总版本列表中
	if grayConfig.TotalGrayWeight <= totalWeight {
		grayConfig.BaseDeployment.Weight = 100 - grayConfig.TotalGrayWeight
	} else {
		totalWeight = grayConfig.TotalGrayWeight
	}
	rand.Seed(time.Now().UnixNano())
	randWeight := rand.Intn(totalWeight)
	sumWeight := 0
	for _, deployment := range deployments {
		sumWeight += deployment.Weight
		if randWeight < sumWeight {
			return deployment
		}
	}
	return nil
}

// InjectContent 用于将内容注入到 HTML 文档的指定位置
func InjectContent(originalHtml string, injectionConfig *config.Injection) string {

	headInjection := strings.Join(injectionConfig.Head, "\n")
	bodyFirstInjection := strings.Join(injectionConfig.Body.First, "\n")
	bodyLastInjection := strings.Join(injectionConfig.Body.Last, "\n")

	// 使用 strings.Builder 来提高性能
	var sb strings.Builder
	// 预分配内存，避免多次内存分配
	sb.Grow(len(originalHtml) + len(headInjection) + len(bodyFirstInjection) + len(bodyLastInjection))
	sb.WriteString(originalHtml)

	modifiedHtml := sb.String()

	// 注入到头部
	modifiedHtml = strings.ReplaceAll(modifiedHtml, "</head>", headInjection+"\n</head>")
	// 注入到body头
	modifiedHtml = strings.ReplaceAll(modifiedHtml, "<body>", "<body>\n"+bodyFirstInjection)
	// 注入到body尾
	modifiedHtml = strings.ReplaceAll(modifiedHtml, "</body>", bodyLastInjection+"\n</body>")

	return modifiedHtml
}
