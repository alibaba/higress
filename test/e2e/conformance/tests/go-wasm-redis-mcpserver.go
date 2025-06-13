package tests

import (
	"testing"

	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
	Register(WasmPluginsRedisMCPServer)
}

var WasmPluginsRedisMCPServer = suite.ConformanceTest{
	ShortName:   "WasmPluginRedisMCPServer",
	Description: "The Ingress in the higress-conformance-redis-mcpserver namespace tests the Redis MCPServer.",
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Manifests:   []string{"tests/go-wasm-redis-mcpserver.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{{Meta: http.AssertionMeta{
			TestCaseName:  "Redis MCPServer Set Operation",
			CompareTarget: http.CompareTargetResponse}, Request: http.AssertionRequest{ActualRequest: http.Request{
			Host:        "redis-mcpserver.example.com",
			Path:        "/mcp/tools/call",
			Method:      "POST",
			ContentType: http.ContentTypeApplicationJson,
			Body: []byte(`{  
                            "jsonrpc": "2.0",   
                            "id": 1,   
                            "method": "tools/call",   
                            "params": {  
                                "name": "set",   
                                "arguments": {  
                                    "key": "test_key",   
                                    "value": "test_value",   
                                    "expiration": 300  
                                }  
                            }  
                        }`),
		}}, Response: http.AssertionResponse{ExpectedResponse: http.Response{
			StatusCode:  200,
			ContentType: http.ContentTypeApplicationJson,
			Body:        []byte(`{"jsonrpc":"2.0","id":1,"result":{"output":"OK"}}`),
		}}}, {Meta: http.AssertionMeta{
			TestCaseName:  "Redis MCPServer Get Operation",
			CompareTarget: http.CompareTargetResponse}, Request: http.AssertionRequest{ActualRequest: http.Request{
			Host:        "redis-mcpserver.example.com",
			Path:        "/mcp/tools/call",
			Method:      "POST",
			ContentType: http.ContentTypeApplicationJson,
			Body: []byte(`{  
                            "jsonrpc": "2.0",   
                            "id": 2,   
                            "method": "tools/call",   
                            "params": {  
                                "name": "get",   
                                "arguments": {  
                                    "key": "test_key"  
                                }  
                            }  
                        }`),
		}}, Response: http.AssertionResponse{ExpectedResponse: http.Response{
			StatusCode:  200,
			ContentType: http.ContentTypeApplicationJson,
			Body:        []byte(`{"jsonrpc":"2.0","id":2,"result":{"output":"test_value"}}`),
		}}}, {Meta: http.AssertionMeta{
			TestCaseName:  "Redis MCPServer TTL Operation",
			CompareTarget: http.CompareTargetResponse}, Request: http.AssertionRequest{ActualRequest: http.Request{
			Host:        "redis-mcpserver.example.com",
			Path:        "/mcp/tools/call",
			Method:      "POST",
			ContentType: http.ContentTypeApplicationJson,
			Body: []byte(`{  
                            "jsonrpc": "2.0",   
                            "id": 3,   
                            "method": "tools/call",   
                            "params": {  
                                "name": "ttl",   
                                "arguments": {  
                                    "key": "test_key"  
                                }  
                            }  
                        }`),
		}}, Response: http.AssertionResponse{ExpectedResponse: http.Response{
			StatusCode:  200,
			ContentType: http.ContentTypeApplicationJson,
			JsonBodyMatcher: func(t *testing.T, body []byte) bool {
				// TTL值会不断变化，所以我们只检查是否能获取到TTL，不检查具体值
				return len(body) > 0
			}}}}, {Meta: http.AssertionMeta{
			TestCaseName:  "Redis MCPServer Del Operation",
			CompareTarget: http.CompareTargetResponse}, Request: http.AssertionRequest{ActualRequest: http.Request{
			Host:        "redis-mcpserver.example.com",
			Path:        "/mcp/tools/call",
			Method:      "POST",
			ContentType: http.ContentTypeApplicationJson,
			Body: []byte(`{  
                            "jsonrpc": "2.0",   
                            "id": 4,   
                            "method": "tools/call",   
                            "params": {  
                                "name": "del",   
                                "arguments": {  
                                    "keys": ["test_key"]  
                                }  
                            }  
                        }`),
		}}, Response: http.AssertionResponse{ExpectedResponse: http.Response{
			StatusCode:  200,
			ContentType: http.ContentTypeApplicationJson,
			Body:        []byte(`{"jsonrpc":"2.0","id":4,"result":{"output":"Deleted 1 keys"}}`),
		}}}}
		t.Run("Redis MCPServer Operations", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)

			}

		})

	},
}
