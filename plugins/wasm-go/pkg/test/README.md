# Test framework for wasm-go

The `pkg/test` directory provides a unit testing framework for the wasm-go project, helping plugin developers write and run high-quality unit tests. The framework supports running tests in both Go mode and Wasm mode. 

![Test Framework Architecture](https://img.alicdn.com/imgextra/i1/O1CN01wJ1Vgk1v9axtimgJT_!!6000000006130-2-tps-1859-547.png)
> Running test with the compiled wasm binary helps to ensure that the plugin will run when actually compiled to wasm, however stack traces and other debug features will be much worse. It is recommended to run unit tests both with Go and with wasm. Tests will run much faster under Go mode for quicker development cycles, and the wasm mode test can confirm the behavior matches when actually compiled.


## Framework Structure

- **`host.go`** - Provides `TestHost` interface to simulate host(envoy) behavior
- **`redis.go`** - Provides Redis response building utility functions
- **`test.go`** - Provides test runners supporting both Go mode and Wasm mode
- **`utils.go`** - Provides utility functions for header testing

## Core Features

### 1. Test Runners (`test.go`)

#### `RunTest(t *testing.T, f func(*testing.T))`
Runs tests in both Go mode and Wasm mode simultaneously, ensuring the plugin works correctly in both environments.

```go
func TestMyPlugin(t *testing.T) {
    test.RunTest(t, func(t *testing.T) {
        // Your test code
    })
}
```

#### `RunTestWithPath(t *testing.T, wasmPath string, f func(*testing.T))`
Runs tests in both Go mode and Wasm mode with a specified wasm file path. This function allows callers to specify custom wasm file paths for testing.

#### `RunGoTest(t *testing.T, f func(*testing.T))`
Runs tests only in Go mode using ABI host call mock interfaces.

#### `RunWasmTest(t *testing.T, f func(*testing.T))`
Runs tests only in Wasm mode using intelligent wasm file path detection in wazero runtime.

#### `RunWasmTestWithPath(t *testing.T, wasmPath string, f func(*testing.T))`
Runs tests only in Wasm mode with a specified wasm file path. This function allows callers to specify custom wasm file paths for testing.

**Note**: The framework automatically compiles wasm binaries when needed. The framework supports multiple ways to specify the wasm file path:

1. **Environment Variable**: Set `WASM_FILE_PATH` environment variable
2. **Custom Path**: Use `RunWasmTestWithPath()` or `RunTestWithPath()` functions
3. **Auto-compilation**: The framework automatically compiles wasm binariy with a fixed filename (`wasm-unit-test.wasm`)
4. **Debug-Friendly**: Panics are preserved in test environment for better debugging, while still recovered in production

#### Common Wasm file Path

The framework searches for wasm files in the following locations (in order of priority):

- `main.wasm` - Default wasm file name in project root
- `plugin.wasm` - Alternative wasm file name in project root

**Example project structure:**
```
my-wasm-plugin/
├── main.go
├── main.wasm         
├── go.mod
└── pkg/
```

**Note**: The auto-detection only searches in the current working directory. For more complex project structures, use environment variables or explicit path functions.

```bash
# Compile wasm binary
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./

# Or specify custom path via environment variable
export WASM_FILE_PATH="build/plugin.wasm"
go test ./...
```

### 2. Test Host Simulation (`host.go`)

#### `NewTestHost(config json.RawMessage) (TestHost, types.OnPluginStartStatus)`
Creates a test host instance to simulate the Envoy proxy environment. The `config` parameter represents the configuration for the wasm plugin.

```go
config := json.RawMessage(`{"key": "value"}`)
host, status := test.NewTestHost(config)
require.Equal(t, types.OnPluginStartStatusOK, status)
defer host.Reset()
```

#### Main Methods

##### HTTP Request
- `CallOnHttpRequestHeaders(headers [][2]string) types.Action` - Call request header processing
- `CallOnHttpRequestBody(body []byte) types.Action` - Call request body processing
- `CallOnHttpStreamingRequestBody(body []byte, endOfStream bool) types.Action` - Call streaming request body processing

##### HTTP Response
- `CallOnHttpResponseHeaders(headers [][2]string) types.Action` - Call response header processing
- `CallOnHttpResponseBody(body []byte) types.Action` - Call response body processing
- `CallOnHttpStreamingResponseBody(body []byte, endOfStream bool) types.Action` - Call streaming response body processing

##### External Call
- `CallOnHttpCall(headers [][2]string, body []byte)` - Simulate HTTP call response
- `CallOnRedisCall(status int32, response []byte)` - Simulate Redis call response
- `GetHttpCalloutAttributes() []proxytest.HttpCalloutAttribute` - Get HTTP callout attributes (outbound http calls made by the plugin)
- `GetRedisCalloutAttributes() []proxytest.RedisCalloutAttribute` - Get Redis callout attributes (outbound redis calls made by the plugin)

##### Plugin Configuration
- `GetMatchConfig() (any, error)` - Get match configuration

##### Property
- `SetRouteName(routeName string) error` - Set route name
- `SetClusterName(clusterName string) error` - Set cluster name
- `SetRequestId(requestId string) error` - Set request ID
- `GetProperty(path []string) ([]byte, error)` - Get property data from the host for a given path
- `SetProperty(path []string, data []byte) error` - Set property data on the host for a given path

##### Result Retrieval
- `GetHttpStreamAction() types.Action` - Get HTTP stream action
- `GetRequestHeaders() [][2]string` - Get request headers
- `GetResponseHeaders() [][2]string` - Get response headers
- `GetRequestBody() []byte` - Get request body
- `GetResponseBody() []byte` - Get response body
- `GetLocalResponse() *proxytest.LocalHttpResponse` - Get local response

##### Metrics
- `GetCounterMetric(name string) (uint64, error)` - Get the value for the counter metric in the host
- `GetGaugeMetric(name string) (uint64, error)` - Get the value for the gauge metric in the host
- `GetHistogramMetric(name string) (uint64, error)` - Get the value for the histogram metric in the host

##### Logs
- `GetTraceLogs() []string` - Get the trace logs that have been collected in the host
- `GetDebugLogs() []string` - Get the debug logs that have been collected in the host
- `GetInfoLogs() []string` - Get the info logs that have been collected in the host
- `GetWarnLogs() []string` - Get the warn logs that have been collected in the host
- `GetErrorLogs() []string` - Get the error logs that have been collected in the host
- `GetCriticalLogs() []string` - Get the critical logs that have been collected in the host

##### Context Management
- `CompleteHttp()` - Complete HTTP request
- `Reset()` - Reset test host state

##### Tick
- `GetTickPeriod() uint32` - Get the current tick period in the host
- `Tick()` - Execute types.PluginContext.OnTick in the plugin
### 3. Redis Response Building (`redis.go`)

#### General Function
- `CreateRedisResp(value interface{}) []byte` - Create Redis response for any type
- `CreateRedisRespArray(values []interface{}) []byte` - Create array response for any type
  
#### Specific Functions
- `CreateRedisRespString(value string) []byte` - Create string response
- `CreateRedisRespInt(value int) []byte` - Create integer response
- `CreateRedisRespBool(value bool) []byte` - Create boolean response
- `CreateRedisRespFloat(value float64) []byte` - Create float response
- `CreateRedisRespNull() []byte` - Create null response
- `CreateRedisRespError(message string) []byte` - Create error response

### 4. Header Utility Functions (`utils.go`)

#### Header Checking Functions
- `HasHeader(headers [][2]string, headerName string) bool` - Check if headers contain a header with the specified name (case-insensitive)
- `GetHeaderValue(headers [][2]string, headerName string) (string, bool)` - Get the value of the specified header, returns value and found status (case-insensitive)
- `HasHeaderWithValue(headers [][2]string, headerName, expectedValue string) bool` - Check if headers contain a header with the specified name and value (case-insensitive)

These utility functions are particularly useful for testing HTTP header processing in your wasm plugins. They provide case-insensitive header matching, which is important for HTTP compliance.

## Usage Examples

### Basic Test Example

```go
func TestMyPlugin(t *testing.T) {
    test.RunTest(t, func(t *testing.T) {
        // 1. Create test host
        config := json.RawMessage(`{"key": "value"}`)
        host, status := test.NewTestHost(config)
        require.Equal(t, types.OnPluginStartStatusOK, status)
        defer host.Reset()

        // 2. Set request headers
        headers := [][2]string{
            {":method", "GET"},
            {":path", "/test"},
            {":authority", "test.com"},
        }

        // 3. Call plugin methods
        action := host.CallOnHttpRequestHeaders(headers)
        require.Equal(t, types.ActionPause, action)

        // 4. Verify outbound calls made by the plugin (if any)
        // httpCallouts := host.GetHttpCalloutAttributes()
        // require.Len(t, httpCallouts, 1)
        // assert.Equal(t, "httpbin.org", httpCallouts[0].Upstream)
        // assert.Equal(t, "GET", test.GetHeaderValue(httpCallouts[0].Headers, ":method"))

        // 5. Simulate external call responses (if needed)

        // host.CallOnRedisCall(0, test.CreateRedisRespString("OK"))

        // host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(`{"result": "success"}`))

        // 6. Complete request
        host.CompleteHttp()

        // 7. Verify results
        localResponse := host.GetLocalResponse()
        require.NotNil(t, localResponse)
        assert.Equal(t, uint32(200), localResponse.StatusCode)
    })
}
```

### Streaming Request Test Example

```go
func TestStreamingRequest(t *testing.T) {
    test.RunTest(t, func(t *testing.T) {
        host, status := test.NewTestHost(testConfig)
        require.Equal(t, types.OnPluginStartStatusOK, status)
        defer host.Reset()

        // Set request headers
        headers := [][2]string{
            {":method", "GET"},
            {":path", "/stream"},
            {":authority", "test.com"},
        }

        // Call request header processing
        action := host.CallOnHttpRequestHeaders(headers)
        require.Equal(t, types.ActionPause, action)

        // Simulate streaming response body
        action = host.CallOnHttpStreamingRequestBody([]byte("chunk1"), false)
        assert.Equal(t, types.ActionContinue, action)

        action = host.CallOnHttpStreamingRequestBody([]byte("chunk2"), true)
        assert.Equal(t, types.ActionContinue, action)

        // Complete request
        host.CompleteHttp()
    })
}
```

### Plugin Configuration Test Example

```go
var testConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		// Global config - applies to all requests when no specific rule matches
		"name": "john",
		// Rules for specific route matching
		"_rules_": []map[string]interface{}{
			{
				"_match_route_": []string{"foo"}, // route level config
				"name": "foo",
			},
            {
                "_match_domain_": []string{"foo.bar.com"}, // domain level config
                "name": "foo.bar.com",
            }
		},
	})
	return data
}()

func TestParseConfig(t *testing.T) {
    test.RunGoTest(t, func(t *testing.T) {
        host, status := test.NewTestHost(testConfig)
        require.Equal(t, types.OnPluginStartStatusOK, status)
        defer host.Reset()

        // Get global plugin configuration
        config, err := host.GetMatchConfig()
		// Get plugin configuration with match route
		host.SetRouteName("foo")
		config, err = host.GetMatchConfig()
		// Get plugin configuration with match cluster
		host.SetClusterName("service")
		config, err = host.GetMatchConfig()
		// Get plugin configuration with match domain
		host.SetDomainName("foo.bar.com")
		config, err = host.GetMatchConfig()

        // Verify configuration content
        // ... Your configuration validation logic
    })
}
```
**Note**: `GetMatchConfig()` can only be used in `RunGoTest()` mode because they are not the proxy-wasm ABI interface. These functions use Go reflection to expose internal plugin configuration for testing.

## Best Practices

### 1. Test Mode Selection
- Use `RunTest()` to ensure the plugin works in both modes
- Use `RunGoTest()` for rapid iteration during development
- Always use `RunWasmTest()` before release to verify compiled behavior
- Use `RunTestWithPath()` or `RunWasmTestWithPath()` when you need to specify custom wasm file paths

### 2. Resource Management
- Always use `defer host.Reset()` to clean up test state
- Create new test host instances at the beginning of each test function

### 3. Assertion Usage
- Use `require` for precondition checks
- Use `assert` for result verification
- Provide clear error messages

### 4. Test Data
- Use meaningful test data
- Test boundary conditions and error cases
- Simulate real network environments

### 5. Outbound Call Testing
- Use `GetHttpCalloutAttributes()` and `GetRedisCalloutAttributes()` to verify external service calls
- **Test order**: Verify outbound calls before simulating external responses
- Check that the plugin makes the expected outbound calls with correct parameters
- Verify upstream service names, headers, and request bodies
- Test both successful and failed external call scenarios

### 6. Wasm File Path Management
- Use environment variable `WASM_FILE_PATH` for consistent configuration across different environments
- Leverage intelligent path detection for common project structures
- Use `RunTestWithPath()` or `RunWasmTestWithPath()` for project-specific wasm file locations

## Important Notes

1. **Test Isolation**: Each test case should use an independent test host instance
2. **State Cleanup**: Use `defer host.Reset()` to ensure test state is properly cleaned up
3. **Error Handling**: Tests should verify the plugin's error handling logic
4. **Performance Considerations**: Avoid creating too many objects or performing time-consuming operations in tests
5. **HTTP Request Lifecycle**: If plugin implementing custom `onHttp*` methods, follow the proper request lifecycle in test. Do not skip intermediate steps - if you implement `onHttpRequestHeader`, do not directly call `onHttpRequestBody`.
6. **Wasm File Path**: The framework automatically detects wasm files in common locations. For custom paths, use environment variables or explicit path functions to ensure consistent test execution across different environments.

## Related Resources

- [proxy-wasm-go-sdk](https://github.com/higress-group/proxy-wasm-go-sdk) - Underlying SDK
- [examples/](https://github.com/higress-group/wasm-go/tree/main/examples) - More test examples
- [proxy-wasm specification](https://github.com/proxy-wasm/spec) - WebAssembly proxy specification

---

By using this testing framework, you can ensure your wasm-go plugin works correctly in various environments, improving code quality and reliability.
