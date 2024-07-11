package util

import (
	"encoding/json"
	"net/url"
	"strings"
	"user-gray/config"
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
			foundCookieValue = strings.TrimPrefix(cookie, curCookieName)
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

func ContainsRule(rules []*config.GrayRule, name string) *config.GrayRule {
	for _, rule := range rules {
		if rule.Name == name {
			return rule
		}
	}
	return nil
}

func GetUidFromSubKey(userInfoStr string, uidSubKey string) string {
	jsonStr, err := url.QueryUnescape(userInfoStr)
	if err != nil {
		return ""
	}
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return ""
	}
	// 从 map 中获取 userCode 的值
	real, ok := result[uidSubKey].(string)
	if !ok {
		return ""
	}
	return real
}
