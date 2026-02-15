package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/stretchr/testify/require"
)

var (
	// defaultTestDomain is the default host name for the test host.
	defaultTestDomain = "default.test.com"
	// CommonVmCtx is init in wasm plugin by wrapper.SetCtx() once
	// wasmInitVMContext store the init CommonVmCtx for each go mode unit test
	wasmInitVMContext types.VMContext
	// testVMContext is the VM context for the each unit test.
	// testVMContext is wasmInitVMContext for go mode unit test
	// testVMContext is WasmVMContext wrap the wasm plugin for wasm mode unit test
	testVMContext types.VMContext
	// wasmInitMutex is the mutex for set the wasm init VM context.
	wasmInitMutex = &sync.Mutex{}
	// testMutex is the mutex for set and clear the test VM context.
	testMutex = &sync.Mutex{}
	// cachedWasmPath stores the compiled wasm file path for reuse across tests
	cachedWasmPath string
	// wasmCacheMutex protects the cached wasm path
	wasmCacheMutex = &sync.Mutex{}
)

// init sets up the test environment
func init() {
	// Enable debug-friendly panic handling in test environment
	// This allows panics to propagate instead of being recovered
	os.Setenv("WASM_DISABLE_PANIC_RECOVERY", "true")
}

// compileWasm compiles the current Go project to wasm binary with a fixed test filename
func compileWasm() (string, error) {
	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %v", err)
	}

	// Use fixed test filename that gets overwritten each time
	fileName := "wasm-unit-test.wasm"
	outputPath := filepath.Join(workDir, fileName)

	// Execute wasm compilation command
	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", outputPath, "./")

	// Filter out existing GOOS and GOARCH to avoid conflicts
	filteredEnv := []string{}
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "GOOS=") && !strings.HasPrefix(env, "GOARCH=") {
			filteredEnv = append(filteredEnv, env)
		}
	}
	cmd.Env = append(filteredEnv, "GOOS=wasip1", "GOARCH=wasm")
	cmd.Dir = workDir

	// Run the command and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wasm compilation failed: %v, output: %s", err, string(output))
	}

	fmt.Printf("[WASM_COMPILE] Successfully compiled wasm binary to: %s\n", outputPath)
	return outputPath, nil
}

// getDefaultWasmPath returns the default wasm file path, supporting both environment variable
// and automatic compilation with caching
func getDefaultWasmPath() string {
	// Priority 1: Environment variable
	if path := os.Getenv("WASM_FILE_PATH"); path != "" {
		fmt.Printf("[WASM_PATH] Using environment variable WASM_FILE_PATH: %s\n", path)
		return path
	}

	// Priority 2: Check cache first
	wasmCacheMutex.Lock()
	defer wasmCacheMutex.Unlock()

	if cachedWasmPath != "" {
		// Check if cached file still exists
		if _, err := os.Stat(cachedWasmPath); err == nil {
			fmt.Printf("[WASM_PATH] Using cached wasm file: %s\n", cachedWasmPath)
			return cachedWasmPath
		}
		// Cached file doesn't exist, clear cache
		cachedWasmPath = ""
	}

	// Priority 3: Automatic compilation
	fmt.Printf("[WASM_PATH] Auto-compiling wasm binary...\n")
	compiledPath, err := compileWasm()
	if err != nil {
		fmt.Printf("[WASM_PATH] Auto-compilation failed: %v\n", err)
		// Return empty string to indicate compilation failure
		// Test functions will handle this by skipping wasm mode tests
		return ""
	}

	// Cache the compiled path (always the same fixed filename)
	cachedWasmPath = compiledPath
	return compiledPath
}

// RunGoTest run the test in go mode, and the testVMContext will be set to the wasmInitVMContext.
// Run unit test in go mode using interface in abi_hostcalls_mock.go in proxy-wasm-go-sdk
func RunGoTest(t *testing.T, f func(*testing.T)) {
	t.Helper()
	t.Run("go", func(t *testing.T) {
		setTestVMContext(getWasmInitVMContext())
		defer clearTestVMContext()
		f(t)
	})
}

// RunWasmTestWithPath run the test in wasm mode with a specified wasm file path.
// This function allows callers to specify custom wasm file paths for testing.
func RunWasmTestWithPath(t *testing.T, wasmPath string, f func(*testing.T)) {
	t.Helper()
	t.Run("wasm", func(t *testing.T) {
		wasm, err := os.ReadFile(wasmPath)
		if err != nil {
			t.Skipf("wasm file not found at path: %s", wasmPath)
		}
		vm, err := proxytest.NewWasmVMContext(wasm)
		require.NoError(t, err)
		defer vm.Close()
		setTestVMContext(vm)
		defer clearTestVMContext()
		f(t)
	})
}

// RunWasmTest run the test in wasm mode, and the testVMContext will be set to the WasmVMContext.
// Run unit test with the compiled wasm binary helps to ensure that the plugin will run when actually compiled to wasm.
// This function automatically compiles the wasm binary if not specified via environment variable.
func RunWasmTest(t *testing.T, f func(*testing.T)) {
	RunWasmTestWithPath(t, getDefaultWasmPath(), f)
}

// RunTestWithPath run the test both in go and wasm mode with a specified wasm file path.
func RunTestWithPath(t *testing.T, wasmPath string, f func(*testing.T)) {
	t.Helper()

	t.Run("go", func(t *testing.T) {
		t.Log("go mode test start")
		setTestVMContext(getWasmInitVMContext())
		defer clearTestVMContext()
		f(t)
		t.Log("go mode test end")
	})

	t.Run("wasm", func(t *testing.T) {
		t.Log("wasm mode test start")
		wasm, err := os.ReadFile(wasmPath)
		if err != nil {
			t.Skipf("wasm file not found at path: %s", wasmPath)
		}
		vm, err := proxytest.NewWasmVMContext(wasm)
		require.NoError(t, err)
		defer vm.Close()
		setTestVMContext(vm)
		defer clearTestVMContext()
		f(t)
		t.Log("wasm mode test end")
	})
}

// RunTest runs unit tests both in go and wasm mode.
// The wasm binary is automatically compiled if not specified via environment variable.
func RunTest(t *testing.T, f func(*testing.T)) {
	RunTestWithPath(t, getDefaultWasmPath(), f)
}

// setWasmInitVMContext set the wasm init VM context.
func setWasmInitVMContext(vm types.VMContext) {
	wasmInitMutex.Lock()
	if wasmInitVMContext == nil {
		wasmInitVMContext = vm
	}
	wasmInitMutex.Unlock()
}

// getWasmInitVMContext get the wasm init VM context.
func getWasmInitVMContext() types.VMContext {
	return wasmInitVMContext
}

// setTestVMContext set the test VM context.
func setTestVMContext(vm types.VMContext) {
	testMutex.Lock()
	testVMContext = vm
}

// clearTestVMContext clear the test VM context.
func clearTestVMContext() {
	testVMContext = nil
	testMutex.Unlock()
}
