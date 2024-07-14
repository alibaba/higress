package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetValueByCookie(t *testing.T) {
	var tests = []struct {
		cookie, cookieKey, output string
	}{
		{"", "uid", ""},
		{`cna=pf_9be76347560439f3b87daede1b485e37; uid=111`, "uid", "111"},
		{`cna=pf_9be76347560439f3b87daede1b485e37; userid=222`, "userid", "222"},
		{`uid=333`, "uid", "333"},
		{`cna=pf_9be76347560439f3b87daede1b485e37;`, "uid", ""},
	}
	for _, test := range tests {
		testName := test.cookie
		t.Run(testName, func(t *testing.T) {
			output := GetValueByCookie(test.cookie, test.cookieKey)
			assert.Equal(t, test.output, output)
		})
	}
}

func TestDecodeJsonCookie(t *testing.T) {
	var tests = []struct {
		userInfoStr, grayJsonKey, output string
	}{
		{"{%22password%22:%22$2a$10$YAvYjA6783YeCi44/M395udIZ4Ll2iyKkQCzePaYx5NNG/aIWgICG%22%2C%22username%22:%22%E8%B0%A2%E6%99%AE%E8%80%80%22%2C%22authorities%22:[]%2C%22accountNonExpired%22:true%2C%22accountNonLocked%22:true%2C%22credentialsNonExpired%22:true%2C%22enabledd%22:true%2C%22id%22:838925798835720200%2C%22mobile%22:%22%22%2C%22userCode%22:%22noah%22%2C%22userName%22:%22%E8%B0%A2%E6%99%AE%E8%80%80%22%2C%22orgId%22:10%2C%22ocId%22:87%2C%22userType%22:%22OWN%22%2C%22firstLogin%22:false%2C%22ownOrgId%22:null%2C%22clientCode%22:%22%22%2C%22clientType%22:null%2C%22country%22:%22UAE%22%2C%22isGuide%22:null%2C%22acctId%22:null%2C%22userToken%22:null%2C%22deviceId%22:%223a47fec00a59d140%22%2C%22ocCode%22:%2299990002%22%2C%22secondType%22:%22dtl%22%2C%22vendorCode%22:%2210000001%22%2C%22status%22:%22ACTIVE%22%2C%22isDelete%22:false%2C%22email%22:%22%22%2C%22deleteStatus%22:null%2C%22deleteRequestDate%22:null%2C%22wechatId%22:null%2C%22userMfaInfoDTO%22:{%22checkMfa%22:false%2C%22checkSuccess%22:false%2C%22mobile%22:null%2C%22email%22:null%2C%22wechatId%22:null%2C%22totpSecret%22:null}}",
			"userCode", "noah"},
	}
	for _, test := range tests {
		testName := test.userInfoStr
		t.Run(testName, func(t *testing.T) {
			output := GetBySubKey(test.userInfoStr, test.grayJsonKey)
			assert.Equal(t, test.output, output)
		})
	}
}
