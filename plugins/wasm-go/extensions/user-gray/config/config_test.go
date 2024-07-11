package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestJsonToUserGreyConfig(t *testing.T) {
	allConfigData := `{"uid-key":"userid","rules":[{"name":"inner-user","uid-value":["00000001","00000005"]},{"name":"beta-user","uid-value":["00000002","00000003"],"gray-tag-key":"level","gray-tag-value":["level3","level5"]}],"deploy":{"base":{"version":"base"},"gray":[{"name":"beta-user","version":"gray","enable":true}]}}`
	var tests = []struct {
		testName string
		uidKey   string
		json     string
	}{
		{"完整的数据", "userid", allConfigData},
	}
	for _, test := range tests {
		testName := test.testName
		t.Run(testName, func(t *testing.T) {
			var userGrayConfig = &UserGrayConfig{}
			JsonToUserGrayConfig(gjson.Parse(test.json), userGrayConfig)
			assert.Equal(t, test.uidKey, userGrayConfig.UidKey)
		})
	}
}
