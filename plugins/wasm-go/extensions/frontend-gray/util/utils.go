package util

import (
	"net/url"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/frontend-gray/config"

	"github.com/tidwall/gjson"
)

// GetValueByCookie 根据 cookieStr 和 cookieName 获取 cookie 值
func GetValueByCookie(cookieStr string, cookieName string) string {
	if cookieStr == "" {
		return ""
	}
	cookies := strings.Split(cookieStr, ";")
	curCookieName := cookieName + "="
	var foundCookieValue string
	var found bool
	// 遍历找到 cookie 对并处理
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie) // 清理空白符
		if strings.HasPrefix(cookie, curCookieName) {
			foundCookieValue = cookie[len(curCookieName):]
			found = true
			break
		}
	}
	if !found {
		return ""
	}
	return foundCookieValue
}

// contains 检查切片 slice 中是否含有元素 value。
func Contains(slice []interface{}, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func GetRule(rules []*config.GrayRule, name string) *config.GrayRule {
	for _, rule := range rules {
		if rule.Name == name {
			return rule
		}
	}
	return nil
}

func GetBySubKey(grayInfoStr string, graySubKey string) string {
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
