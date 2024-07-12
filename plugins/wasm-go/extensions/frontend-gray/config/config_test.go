package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestJsonToGreyConfig(t *testing.T) {
	allConfigData := `{"gray-key":"userid","rules":[{"name":"inner-user","gray-key-value":["00000001","00000005"]},{"name":"beta-user","gray-key-value":["00000002","00000003"],"gray-tag-key":"level","gray-tag-value":["level3","level5"]}],"deploy":{"base":{"version":"base"},"gray":[{"name":"beta-user","version":"gray","enable":true}]}}`
	var tests = []struct {
		testName string
		grayKey  string
		json     string
	}{
		{"完整的数据", "userid", allConfigData},
	}
	for _, test := range tests {
		testName := test.testName
		t.Run(testName, func(t *testing.T) {
			var grayConfig = &GrayConfig{}
			JsonToGrayConfig(gjson.Parse(test.json), grayConfig)
			assert.Equal(t, test.grayKey, grayConfig.GrayKey)
		})
	}
}
