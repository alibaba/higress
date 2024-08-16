package main

import (
	"errors"
	"testing"

	//"strconv"
	_ "embed"

	"github.com/stretchr/testify/assert"
)

func TestRange2CidrList(t *testing.T) {
	tests := []struct {
		name    string
		startIp string
		endIp   string
		want    map[string]int
	}{
		{
			"test start ip with 0.0.0.0",
			"0.0.0.0",
			"1.0.0.255",
			map[string]int{
				"0.0.0.0/8":  1,
				"1.0.0.0/24": 1,
			},
		},
		{
			"test the same network segment",
			"1.0.1.0",
			"1.0.1.255",
			map[string]int{"1.0.1.0/24": 1},
		},
		{
			"test cross network segment",
			"1.0.1.0",
			"2.0.1.112",
			map[string]int{
				"1.0.1.0/24":   1,
				"1.0.2.0/23":   1,
				"1.0.4.0/22":   1,
				"1.0.8.0/21":   1,
				"1.0.16.0/20":  1,
				"1.0.32.0/19":  1,
				"1.0.64.0/18":  1,
				"1.0.128.0/17": 1,
				"1.1.0.0/16":   1,
				"1.2.0.0/15":   1,
				"1.4.0.0/14":   1,
				"1.8.0.0/13":   1,
				"1.16.0.0/12":  1,
				"1.32.0.0/11":  1,
				"1.64.0.0/10":  1,
				"1.128.0.0/9":  1,
				"2.0.0.0/24":   1,
				"2.0.1.0/26":   1,
				"2.0.1.64/27":  1,
				"2.0.1.96/28":  1,
				"2.0.1.112/32": 1,
			},
		},
		{
			"test end ip with 255.255.255.255",
			"224.0.0.0",
			"255.255.255.255",
			map[string]int{"224.0.0.0/3": 1},
		},
		{
			"test start ip is greater than end ip",
			"1.0.0.255",
			"1.0.0.0",
			map[string]int{},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			actual := range2cidrList(test.startIp, test.endIp)
			assert.Equal(t, len(test.want), len(actual), "")
			for _, v := range actual {
				if _, ok := test.want[v]; !ok {
					assert.Error(t, errors.New("not match"), "")
				}
			}

		})
	}
}
