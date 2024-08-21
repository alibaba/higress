package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestIsIndexRequest(t *testing.T) {
	var tests = []struct {
		fetchMode string
		p         string
		output    bool
	}{
		{"cors", "/js/a.js", false},
		{"no-cors", "/js/a.js", false},
		{"no-cors", "/images/a.png", false},
		{"no-cors", "/index", true},
		{"cors", "/inde", false},
		{"no-cors", "/index.html", true},
		{"no-cors", "/demo.php", true},
	}
	for _, test := range tests {
		testPath := test.p
		t.Run(testPath, func(t *testing.T) {
			output := IsIndexRequest(test.fetchMode, testPath)
			assert.Equal(t, test.output, output)
		})
	}
}
