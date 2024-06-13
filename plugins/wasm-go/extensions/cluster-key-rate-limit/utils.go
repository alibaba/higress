package main

import (
	"fmt"
	"github.com/zmap/go-iptree/iptree"
	"sort"
	"strings"
)

// parseIPNet 解析Ip段配置
func parseIPNet(key string) (*iptree.IPTree, error) {
	tree := iptree.New()
	err := tree.AddByString(key, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid IP[%s]", key)
	}
	return tree, nil
}

// parseIP 解析IP
func parseIP(source string) string {
	if strings.Contains(source, ".") {
		// parse ipv4
		return strings.Split(source, ":")[0]
	}
	// parse ipv6
	if strings.Contains(source, "]") {
		return strings.Split(source, "]")[0][1:]
	}
	return source
}

// reconvertHeaders headers: map[string][]string -> [][2]string
func reconvertHeaders(hs map[string][]string) [][2]string {
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

// extractCookieValueByKey 从cookie中提取key对应的value
func extractCookieValueByKey(cookie string, key string) (value string) {
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
