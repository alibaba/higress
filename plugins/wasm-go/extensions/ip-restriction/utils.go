package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/asergeyev/nradix"
	"github.com/tidwall/gjson"
	"github.com/zmap/go-iptree/iptree"

	"github.com/higress-group/wasm-go/pkg/log"
)

// parseIPNets Parse CIDR or IP range or subnet configuration
func parseIPNets(array []gjson.Result) (*iptree.IPTree, error) {
	if len(array) == 0 {
		return nil, nil
	} else {
		tree := iptree.New()
		for _, result := range array {
			err := tree.AddByString(result.String(), 0)
			if err != nil {
				if errors.Is(err, nradix.ErrNodeBusy) {
					// ErrNodeBusy means the IP already exists in the tree
					log.Warnf("ignore duplicate IP [%s]", result.String())
				} else {
					return nil, fmt.Errorf("add IP [%s] into tree failed: %v", result.String(), err)
				}
			}
		}
		return tree, nil
	}
}

// parseIP Parse IP
func parseIP(source string, fromHeader bool) string {

	if fromHeader {
		source = strings.Split(source, ",")[0]
	}
	source = strings.Trim(source, " ")
	if strings.Contains(source, ".") {
		// parse ipv4
		return strings.Split(source, ":")[0]
	}
	//parse ipv6
	if strings.Contains(source, "]") {
		return strings.Split(source, "]")[0][1:]
	}
	return source
}
