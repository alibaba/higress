package main

import (
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/zmap/go-iptree/iptree"
)

// parseIPNets 解析Ip段配置
func parseIPNets(array []gjson.Result) (*iptree.IPTree, error) {
	if len(array) == 0 {
		return nil, nil
	} else {
		tree := iptree.New()
		for _, result := range array {
			err := tree.AddByString(result.String(), 0)
			if err != nil {
				return nil, fmt.Errorf("invalid IP[%s]", result.String())
			}
		}
		return tree, nil
	}
}
