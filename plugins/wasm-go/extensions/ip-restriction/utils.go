package main

import (
	"fmt"
	"github.com/tidwall/gjson"
	"net"
	"strings"
)

// parseIPNets 解析Ip段配置
func parseIPNets(array []gjson.Result) ([]net.IPNet, error) {
	if len(array) == 0 {
		return []net.IPNet{}, nil
	} else {
		var ips []net.IPNet
		for _, result := range array {
			s := result.String()
			if strings.Contains(s, "/") {
				_, ipNet, err := net.ParseCIDR(s)
				if err != nil {
					return nil, err
				} else {
					ips = append(ips, *ipNet)
				}
			} else {
				ip := net.ParseIP(s)
				if ip == nil {
					return []net.IPNet{}, fmt.Errorf("invalid IP[%s]", s)
				} else {
					ips = append(ips, net.IPNet{
						IP:   ip,
						Mask: net.IPv4Mask(255, 255, 255, 255),
					})
				}
			}
		}
		return ips, nil
	}
}
