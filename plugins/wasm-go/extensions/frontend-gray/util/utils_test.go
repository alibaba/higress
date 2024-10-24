package util

import (
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/frontend-gray/config"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestExtractCookieValueByKey(t *testing.T) {
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
			output := ExtractCookieValueByKey(test.cookie, test.cookieKey)
			assert.Equal(t, test.output, output)
		})
	}
}

// 测试首页Rewrite重写
func TestIndexRewrite(t *testing.T) {
	matchRules := map[string]string{
		"/app1": "/mfe/app1/{version}/index.html",
		"/":     "/mfe/app1/{version}/index.html",
	}

	var tests = []struct {
		path, output string
	}{
		{"/app1/", "/mfe/app1/v1.0.0/index.html"},
		{"/app123", "/mfe/app1/v1.0.0/index.html"},
		{"/app1/index.html", "/mfe/app1/v1.0.0/index.html"},
		{"/app1/index.jsp", "/mfe/app1/v1.0.0/index.html"},
		{"/app1/xxx", "/mfe/app1/v1.0.0/index.html"},
		{"/xxxx", "/mfe/app1/v1.0.0/index.html"},
	}
	for _, test := range tests {
		testName := test.path
		t.Run(testName, func(t *testing.T) {
			output := IndexRewrite(testName, "v1.0.0", matchRules)
			assert.Equal(t, test.output, output)
		})
	}
}

func TestIndexRewrite2(t *testing.T) {
	matchRules := map[string]string{
		"/":       "/{version}/index.html",
		"/sta":    "/sta/{version}/index.html",
		"/static": "/static/{version}/index.html",
	}

	var tests = []struct {
		path, output string
	}{
		{"/static123", "/static/v1.0.0/index.html"},
		{"/static", "/static/v1.0.0/index.html"},
		{"/sta", "/sta/v1.0.0/index.html"},
		{"/", "/v1.0.0/index.html"},
	}
	for _, test := range tests {
		testName := test.path
		t.Run(testName, func(t *testing.T) {
			output := IndexRewrite(testName, "v1.0.0", matchRules)
			assert.Equal(t, test.output, output)
		})
	}
}

func TestPrefixFileRewrite(t *testing.T) {
	matchRules := map[string]string{
		// 前缀匹配
		"/":             "/mfe/app1/{version}",
		"/app2/":        "/mfe/app1/{version}",
		"/app1/":        "/mfe/app1/{version}",
		"/app1/prefix2": "/mfe/app1/{version}",
		"/mfe/app1":     "/mfe/app1/{version}",
	}

	var tests = []struct {
		path, output string
	}{
		{"/js/a.js", "/mfe/app1/v1.0.0/js/a.js"},
		{"/app2/js/a.js", "/mfe/app1/v1.0.0/js/a.js"},
		{"/app1/js/a.js", "/mfe/app1/v1.0.0/js/a.js"},
		{"/app1/prefix2/js/a.js", "/mfe/app1/v1.0.0/js/a.js"},
		{"/app1/prefix2/js/a.js", "/mfe/app1/v1.0.0/js/a.js"},
		{"/mfe/app1/js/a.js", "/mfe/app1/v1.0.0/js/a.js"},
	}
	for _, test := range tests {
		testName := test.path
		t.Run(testName, func(t *testing.T) {
			output := PrefixFileRewrite(testName, "v1.0.0", matchRules)
			assert.Equal(t, test.output, output)
		})
	}
}

func TestIsPageRequest(t *testing.T) {
	var tests = []struct {
		p      string
		output bool
	}{
		{"/js/a.js", false},
		{"/js/a.js", false},
		{"/images/a.png", false},
		{"/index", true},
		{"/index.html", true},
		{"/demo.php", true},
	}
	for _, test := range tests {
		testPath := test.p
		t.Run(testPath, func(t *testing.T) {
			output := IsPageRequest(testPath)
			assert.Equal(t, test.output, output)
		})
	}
}

func TestFilterGrayWeight(t *testing.T) {
	var tests = []struct {
		name  string
		input string
	}{
		{"demo", `{"grayKey":"userId","rules":[{"name":"inner-user","grayKeyValue":["00000001","00000005"]},{"name":"beta-user","grayKeyValue":["noah","00000003"],"grayTagKey":"level","grayTagValue":["level3","level5"]}],"rewrite":{"host":"frontend-gray-cn-shanghai.oss-cn-shanghai-internal.aliyuncs.com","notFoundUri":"/mfe/app1/dev/404.html","indexRouting":{"/app1":"/mfe/app1/{version}/index.html","/":"/mfe/app1/{version}/index.html"},"fileRouting":{"/":"/mfe/app1/{version}","/app1":"/mfe/app1/{version}"}},"baseDeployment":{"version":"dev"},"grayDeployments":[{"name":"beta-user","version":"0.0.1","backendVersion":"beta","enabled":true,"weight":50}]}`},
	}
	for _, test := range tests {
		testName := test.name
		t.Run(testName, func(t *testing.T) {
			grayConfig := &config.GrayConfig{}
			config.JsonToGrayConfig(gjson.Parse(test.input), grayConfig)
			result := FilterGrayWeight(grayConfig, "base", "1.0.1", "192.168.1.1")
			t.Logf("result-----: %v", result)
		})
	}
}

func TestReplaceHtml(t *testing.T) {
	var tests = []struct {
		name  string
		input string
	}{
		{"demo", `{"injection":{"head":["<script>console.log('Head')</script>"],"body":{"first":["<script>console.log('BodyFirst')</script>"],"last":["<script>console.log('BodyLast')</script>"]},"last":["<script>console.log('BodyLast')</script>"]},"html": "<!DOCTYPE html>\n   <html lang=\"zh-CN\">\n<head>\n<title>app1</title>\n<meta charset=\"utf-8\" />\n</head>\n<body>\n\t测试替换html版本\n\t<br />\n\t版本: {version}\n\t<br />\n\t<script src=\"./{version}/a.js\"></script>\n</body>\n</html>"}`},
		{"demo-noBody", `{"injection":{"head":["<script>console.log('Head')</script>"],"body":{"first":["<script>console.log('BodyFirst')</script>"],"last":["<script>console.log('BodyLast')</script>"]},"last":["<script>console.log('BodyLast')</script>"]},"html": "<!DOCTYPE html>\n   <html lang=\"zh-CN\">\n<head>\n<title>app1</title>\n<meta charset=\"utf-8\" />\n</head>\n</html>"}`},
	}
	for _, test := range tests {
		testName := test.name
		t.Run(testName, func(t *testing.T) {
			grayConfig := &config.GrayConfig{}
			config.JsonToGrayConfig(gjson.Parse(test.input), grayConfig)
			result := InjectContent(grayConfig.Html, grayConfig.Injection)
			t.Logf("result-----: %v", result)
		})
	}
}
