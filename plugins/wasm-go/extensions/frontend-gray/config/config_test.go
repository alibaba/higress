package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestJsonToGrayConfig(t *testing.T) {
	allConfigData := `{"grayKey":"userid","rules":[{"name":"inner-user","grayKeyValue":["00000001","00000005"]},{"name":"beta-user","grayKeyValue":["00000002","00000003"],"grayTagKey":"level","grayTagValue":["level3","level5"]}],"deploy":{"base":{"version":"base"},"gray":[{"name":"beta-user","version":"gray","enabled":true}]}}`
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
