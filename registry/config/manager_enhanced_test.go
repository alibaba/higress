package config

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/hashicorp/go-multierror"
	apiv1 "github.com/alibaba/higress/api/networking/v1"
)

// MockConfigProvider implements ConfigProvider for testing
type MockConfigProvider struct {
	mock.Mock
	isWatching bool
}

func (m *MockConfigProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConfigProvider) GetMCPConfig(ctx context.Context, configRef string) (*apiv1.MCPConfig, error) {
	args := m.Called(ctx, configRef)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apiv1.MCPConfig), args.Error(1)
}

func (m *MockConfigProvider) Watch(ctx context.Context, handler ConfigUpdateHandler) error {
	args := m.Called(ctx, handler)
	if args.Error(0) == nil {
		m.isWatching = true
	}
	return args.Error(0)
}

func (m *MockConfigProvider) Stop() error {
	args := m.Called()
	m.isWatching = false
	return args.Error(0)
}

func (m *MockConfigProvider) IsWatching() bool {
	return m.isWatching
}

// TestStartWatchingPartialFailure tests partial failure scenario
func TestStartWatchingPartialFailure(t *testing.T) {
	manager := NewManager(nil)
	
	// Create mock providers
	successProvider := &MockConfigProvider{}
	failingProvider := &MockConfigProvider{}
	
	// Setup expectations
	successProvider.On("Watch", mock.Anything, mock.Anything).Return(nil)
	failingProvider.On("Watch", mock.Anything, mock.Anything).Return(fmt.Errorf("connection failed"))
	
	// Register providers
	manager.RegisterProvider(ConfigSourceConfigMap, successProvider)
	manager.RegisterProvider(ConfigSourceSecret, failingProvider)
	
	// Test StartWatching
	ctx := context.Background()
	handler := func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error {
		return nil
	}
	
	err := manager.StartWatching(ctx, handler)
	
	// Verify partial failure behavior
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ConfigSourceSecret")
	assert.NotContains(t, err.Error(), "ConfigSourceConfigMap")
	
	// Verify successful provider is still running
	assert.True(t, successProvider.IsWatching())
	assert.False(t, failingProvider.IsWatching())
	
	// Verify all expectations were met
	successProvider.AssertExpectations(t)
	failingProvider.AssertExpectations(t)
}

// TestStartWatchingWithRetrySuccess tests retry mechanism success
func TestStartWatchingWithRetrySuccess(t *testing.T) {
	manager := NewManager(nil)
	
	provider := &MockConfigProvider{}
	
	// First call fails, second succeeds
	provider.On("Watch", mock.Anything, mock.Anything).Return(fmt.Errorf("temporary failure")).Once()
	provider.On("Watch", mock.Anything, mock.Anything).Return(nil).Once()
	
	manager.RegisterProvider(ConfigSourceConfigMap, provider)
	
	ctx := context.Background()
	handler := func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error {
		return nil
	}
	
	start := time.Now()
	err := manager.StartWatchingWithRetry(ctx, handler, 2)
	duration := time.Since(start)
	
	// Should succeed after retry
	assert.NoError(t, err)
	assert.True(t, provider.IsWatching())
	
	// Should have taken some time for retry delay
	assert.True(t, duration >= time.Second)
	
	provider.AssertExpectations(t)
}

// TestStartWatchingWithRetryExhausted tests retry exhaustion
func TestStartWatchingWithRetryExhausted(t *testing.T) {
	manager := NewManager(nil)
	
	provider := &MockConfigProvider{}
	
	// Always fail
	provider.On("Watch", mock.Anything, mock.Anything).Return(fmt.Errorf("persistent failure"))
	
	manager.RegisterProvider(ConfigSourceConfigMap, provider)
	
	ctx := context.Background()
	handler := func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error {
		return nil
	}
	
	maxRetries := 2
	err := manager.StartWatchingWithRetry(ctx, handler, maxRetries)
	
	// Should fail after exhausting retries
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after")
	assert.Contains(t, err.Error(), fmt.Sprintf("%d attempts", maxRetries+1))
	assert.False(t, provider.IsWatching())
	
	// Verify the right number of attempts
	provider.AssertNumberOfCalls(t, "Watch", maxRetries+1)
}

// TestCircuitBreakerIntegration tests circuit breaker functionality
func TestCircuitBreakerIntegration(t *testing.T) {
	manager := NewManager(nil)
	
	provider := &MockConfigProvider{}
	
	// Setup to fail multiple times to trigger circuit breaker
	provider.On("Watch", mock.Anything, mock.Anything).Return(fmt.Errorf("failure"))
	
	manager.RegisterProvider(ConfigSourceConfigMap, provider)
	
	ctx := context.Background()
	handler := func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error {
		return nil
	}
	
	// Trigger failures to open circuit breaker
	for i := 0; i < 3; i++ {
		manager.StartWatching(ctx, handler)
	}
	
	// Circuit breaker should now be open
	assert.True(t, manager.circuitBreaker.IsOpen("configmap"))
	
	// Next attempt should skip due to circuit breaker
	err := manager.StartWatching(ctx, handler)
	assert.NoError(t, err) // No error because circuit breaker prevents call
	
	provider.AssertExpectations(t)
}

// TestGetMCPConfigWithFallback tests stale cache fallback
func TestGetMCPConfigWithFallback(t *testing.T) {
	manager := NewManager(nil)
	
	provider := &MockConfigProvider{}
	
	// First call succeeds, second fails
	expectedConfig := &apiv1.MCPConfig{
		Instances: []*apiv1.MCPInstance{
			{Domain: "test.com", Port: 8080},
		},
	}
	
	provider.On("GetMCPConfig", mock.Anything, "test-config").Return(expectedConfig, nil).Once()
	provider.On("GetMCPConfig", mock.Anything, "test-config").Return(nil, fmt.Errorf("failure")).Once()
	
	manager.RegisterProvider(ConfigSourceConfigMap, provider)
	
	ctx := context.Background()
	
	// First call - should succeed and cache result
	config1, err1 := manager.GetMCPConfigWithFallback(ctx, ConfigSourceConfigMap, "test-config")
	assert.NoError(t, err1)
	assert.Equal(t, expectedConfig, config1)
	
	// Second call - should fail but return cached result
	config2, err2 := manager.GetMCPConfigWithFallback(ctx, ConfigSourceConfigMap, "test-config")
	assert.NoError(t, err2)
	assert.Equal(t, expectedConfig, config2)
	
	provider.AssertExpectations(t)
}

// TestStaleCache tests stale cache expiration
func TestStaleCache(t *testing.T) {
	cache := NewStaleCache(100 * time.Millisecond) // Very short expiry for testing
	
	config := &apiv1.MCPConfig{
		Instances: []*apiv1.MCPInstance{
			{Domain: "test.com", Port: 8080},
		},
	}
	
	// Store config
	cache.SetStale(ConfigSourceConfigMap, "test-config", config)
	
	// Should retrieve immediately
	retrieved := cache.GetStale(ConfigSourceConfigMap, "test-config")
	assert.Equal(t, config, retrieved)
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Should return nil after expiration
	expired := cache.GetStale(ConfigSourceConfigMap, "test-config")
	assert.Nil(t, expired)
}

// TestCircuitBreakerRecovery tests circuit breaker recovery
func TestCircuitBreakerRecovery(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond) // Low threshold and timeout for testing
	
	source := "test-source"
	
	// Trigger failures to open circuit
	cb.RecordFailure(source)
	cb.RecordFailure(source)
	
	assert.True(t, cb.IsOpen(source))
	
	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)
	
	// Should be in half-open state now
	assert.False(t, cb.IsOpen(source))
	
	// Record success to close circuit
	cb.RecordSuccess(source)
	assert.False(t, cb.IsOpen(source))
}

// TestConcurrentAccess tests concurrent access to manager
func TestConcurrentAccess(t *testing.T) {
	manager := NewManager(nil)
	
	provider := &MockConfigProvider{}
	provider.On("GetMCPConfig", mock.Anything, mock.Anything).Return(&apiv1.MCPConfig{}, nil)
	
	manager.RegisterProvider(ConfigSourceConfigMap, provider)
	
	ctx := context.Background()
	
	// Run concurrent operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			configRef := fmt.Sprintf("config-%d", id)
			_, err := manager.GetMCPConfig(ctx, ConfigSourceConfigMap, configRef)
			assert.NoError(t, err)
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Should have been called 10 times
	provider.AssertNumberOfCalls(t, "GetMCPConfig", 10)
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	manager := NewManager(nil)
	
	provider := &MockConfigProvider{}
	provider.On("Watch", mock.Anything, mock.Anything).Return(fmt.Errorf("failure"))
	
	manager.RegisterProvider(ConfigSourceConfigMap, provider)
	
	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	handler := func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error {
		return nil
	}
	
	// Cancel context immediately
	cancel()
	
	// Should return context error when retrying
	err := manager.StartWatchingWithRetry(ctx, handler, 2)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestMultipleErrorAggregation tests multierror functionality
func TestMultipleErrorAggregation(t *testing.T) {
	manager := NewManager(nil)
	
	// Create multiple failing providers
	provider1 := &MockConfigProvider{}
	provider2 := &MockConfigProvider{}
	provider3 := &MockConfigProvider{}
	
	// Setup different failure types
	provider1.On("Watch", mock.Anything, mock.Anything).Return(fmt.Errorf("connection timeout"))
	provider2.On("Watch", mock.Anything, mock.Anything).Return(fmt.Errorf("permission denied"))
	provider3.On("Watch", mock.Anything, mock.Anything).Return(fmt.Errorf("service unavailable"))
	
	// Register all providers
	manager.RegisterProvider(ConfigSourceConfigMap, provider1)
	manager.RegisterProvider(ConfigSourceSecret, provider2)
	manager.RegisterProvider(ConfigSourceConsul, provider3)
	
	ctx := context.Background()
	handler := func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error {
		return nil
	}
	
	// Test StartWatching with multiple failures
	err := manager.StartWatching(ctx, handler)
	
	// Should be a multierror.Error
	assert.Error(t, err)
	
	// Check if it's a multierror and contains all expected errors
	if multiErr, ok := err.(*multierror.Error); ok {
		assert.Len(t, multiErr.Errors, 3)
		
		errorStrs := make([]string, len(multiErr.Errors))
		for i, e := range multiErr.Errors {
			errorStrs[i] = e.Error()
		}
		
		// Verify all error sources are present
		assert.True(t, containsError(errorStrs, "configmap"))
		assert.True(t, containsError(errorStrs, "secret"))
		assert.True(t, containsError(errorStrs, "consul"))
		
		// Verify specific error messages are preserved
		assert.True(t, containsError(errorStrs, "connection timeout"))
		assert.True(t, containsError(errorStrs, "permission denied"))
		assert.True(t, containsError(errorStrs, "service unavailable"))
	} else {
		t.Errorf("Expected multierror.Error, got %T", err)
	}
	
	// Verify all providers were called
	provider1.AssertExpectations(t)
	provider2.AssertExpectations(t)
	provider3.AssertExpectations(t)
}

// TestStopMultipleErrors tests Stop method with multiple failures
func TestStopMultipleErrors(t *testing.T) {
	manager := NewManager(nil)
	
	provider1 := &MockConfigProvider{}
	provider2 := &MockConfigProvider{}
	provider3 := &MockConfigProvider{}
	
	// Setup mixed success/failure scenarios
	provider1.On("Stop").Return(nil) // Success
	provider2.On("Stop").Return(fmt.Errorf("cleanup failed"))
	provider3.On("Stop").Return(fmt.Errorf("resource locked"))
	
	manager.RegisterProvider(ConfigSourceConfigMap, provider1)
	manager.RegisterProvider(ConfigSourceSecret, provider2)
	manager.RegisterProvider(ConfigSourceConsul, provider3)
	
	// Test Stop with multiple failures
	err := manager.Stop()
	
	// Should be a multierror.Error with 2 failures
	assert.Error(t, err)
	
	if multiErr, ok := err.(*multierror.Error); ok {
		assert.Len(t, multiErr.Errors, 2)
		
		errorStrs := make([]string, len(multiErr.Errors))
		for i, e := range multiErr.Errors {
			errorStrs[i] = e.Error()
		}
		
		// Verify only failing providers are in the error
		assert.False(t, containsError(errorStrs, "configmap")) // This one succeeded
		assert.True(t, containsError(errorStrs, "secret"))
		assert.True(t, containsError(errorStrs, "consul"))
		
		// Verify specific error messages
		assert.True(t, containsError(errorStrs, "cleanup failed"))
		assert.True(t, containsError(errorStrs, "resource locked"))
	} else {
		t.Errorf("Expected multierror.Error, got %T", err)
	}
	
	// Verify all providers were called
	provider1.AssertExpectations(t)
	provider2.AssertExpectations(t)
	provider3.AssertExpectations(t)
}

// TestErrorWrapping tests that original errors are properly wrapped
func TestErrorWrapping(t *testing.T) {
	manager := NewManager(nil)
	
	originalErr := fmt.Errorf("original database error")
	provider := &MockConfigProvider{}
	provider.On("Watch", mock.Anything, mock.Anything).Return(originalErr)
	
	manager.RegisterProvider(ConfigSourceConfigMap, provider)
	
	ctx := context.Background()
	handler := func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error {
		return nil
	}
	
	err := manager.StartWatching(ctx, handler)
	
	// Check error unwrapping works
	assert.Error(t, err)
	assert.ErrorIs(t, err, originalErr)
	
	provider.AssertExpectations(t)
}

// TestMultiErrorFormatting tests multierror string representation
func TestMultiErrorFormatting(t *testing.T) {
	manager := NewManager(nil)
	
	provider1 := &MockConfigProvider{}
	provider2 := &MockConfigProvider{}
	
	provider1.On("Stop").Return(fmt.Errorf("first error"))
	provider2.On("Stop").Return(fmt.Errorf("second error"))
	
	manager.RegisterProvider(ConfigSourceConfigMap, provider1)
	manager.RegisterProvider(ConfigSourceSecret, provider2)
	
	err := manager.Stop()
	
	assert.Error(t, err)
	
	// Check formatted error message
	errStr := err.Error()
	assert.Contains(t, errStr, "first error")
	assert.Contains(t, errStr, "second error")
	assert.Contains(t, errStr, "configmap")
	assert.Contains(t, errStr, "secret")
	
	// Verify multierror format (should contain multiple lines)
	lines := strings.Split(errStr, "\n")
	assert.True(t, len(lines) > 1, "Multierror should format as multiple lines")
	
	provider1.AssertExpectations(t)
	provider2.AssertExpectations(t)
}

// TestEmptyProviderList tests behavior with no providers
func TestEmptyProviderList(t *testing.T) {
	manager := NewManager(nil)
	
	ctx := context.Background()
	handler := func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error {
		return nil
	}
	
	// Should succeed with no providers
	err := manager.StartWatching(ctx, handler)
	assert.NoError(t, err)
	
	// Stop should also succeed
	err = manager.Stop()
	assert.NoError(t, err)
}

// TestMixedSuccessFailure tests partial success scenarios
func TestMixedSuccessFailure(t *testing.T) {
	manager := NewManager(nil)
	
	successProvider := &MockConfigProvider{}
	failingProvider := &MockConfigProvider{}
	
	successProvider.On("Watch", mock.Anything, mock.Anything).Return(nil)
	failingProvider.On("Watch", mock.Anything, mock.Anything).Return(fmt.Errorf("network error"))
	
	manager.RegisterProvider(ConfigSourceConfigMap, successProvider)
	manager.RegisterProvider(ConfigSourceSecret, failingProvider)
	
	ctx := context.Background()
	handler := func(configRef string, config *apiv1.MCPConfig, eventType ConfigEventType) error {
		return nil
	}
	
	err := manager.StartWatching(ctx, handler)
	
	// Should report error for failing provider only
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret")
	assert.NotContains(t, err.Error(), "configmap")
	
	// Success provider should still be watching
	assert.True(t, successProvider.IsWatching())
	assert.False(t, failingProvider.IsWatching())
	
	successProvider.AssertExpectations(t)
	failingProvider.AssertExpectations(t)
}

// Helper function to check if error list contains specific text
func containsError(errors []string, text string) bool {
	for _, err := range errors {
		if strings.Contains(strings.ToLower(err), strings.ToLower(text)) {
			return true
		}
	}
	return false
}

// TestUnimplementedProviderErrors tests error messages for unimplemented providers
func TestUnimplementedProviderErrors(t *testing.T) {
	factory := NewDefaultProviderFactory(nil)
	
	testCases := []struct {
		name           string
		source         ConfigSource
		expectedText   []string
		shouldContain  []string
	}{
		{
			name:   "Secret Provider",
			source: ConfigSourceSecret,
			expectedText: []string{
				"secret provider is not yet implemented",
				"please use ConfigMap provider instead",
				"supported sources:",
			},
			shouldContain: []string{"configmap"},
		},
		{
			name:   "Etcd Provider", 
			source: ConfigSourceEtcd,
			expectedText: []string{
				"etcd provider is not yet implemented",
				"please use ConfigMap provider instead",
				"supported sources:",
			},
			shouldContain: []string{"configmap"},
		},
		{
			name:   "Consul Provider",
			source: ConfigSourceConsul,
			expectedText: []string{
				"consul provider is not yet implemented", 
				"please use ConfigMap provider instead",
				"supported sources:",
			},
			shouldContain: []string{"configmap"},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &ProviderConfig{
				Source:    tc.source,
				Namespace: "test",
			}
			
			provider, err := factory.CreateProvider(config)
			
			// Should return error
			assert.Error(t, err)
			assert.Nil(t, provider)
			
			errMsg := err.Error()
			
			// Check for expected text
			for _, text := range tc.expectedText {
				assert.Contains(t, errMsg, text, "Error message should contain: %s", text)
			}
			
			// Check for supportive content
			for _, text := range tc.shouldContain {
				assert.Contains(t, strings.ToLower(errMsg), text, "Error message should mention supported alternative: %s", text)
			}
		})
	}
}

// TestUnsupportedProviderError tests error message for completely unsupported providers
func TestUnsupportedProviderError(t *testing.T) {
	factory := NewDefaultProviderFactory(nil)
	
	config := &ProviderConfig{
		Source:    "invalid-source",
		Namespace: "test",
	}
	
	provider, err := factory.CreateProvider(config)
	
	assert.Error(t, err)
	assert.Nil(t, provider)
	
	errMsg := err.Error()
	assert.Contains(t, errMsg, "unsupported configuration source: invalid-source")
	assert.Contains(t, errMsg, "supported sources:")
	assert.Contains(t, strings.ToLower(errMsg), "configmap")
}

// TestSupportedSourcesList tests that SupportedSources returns correct list
func TestSupportedSourcesList(t *testing.T) {
	factory := NewDefaultProviderFactory(nil)
	
	sources := factory.SupportedSources()
	
	// Should contain ConfigMap
	assert.Contains(t, sources, ConfigSourceConfigMap)
	
	// Should not contain unimplemented sources
	assert.NotContains(t, sources, ConfigSourceSecret)
	assert.NotContains(t, sources, ConfigSourceEtcd) 
	assert.NotContains(t, sources, ConfigSourceConsul)
}

// TestErrorMessageConsistency tests that all error messages follow consistent format
func TestErrorMessageConsistency(t *testing.T) {
	factory := NewDefaultProviderFactory(nil)
	
	unimplementedSources := []ConfigSource{
		ConfigSourceSecret,
		ConfigSourceEtcd,
		ConfigSourceConsul,
	}
	
	for _, source := range unimplementedSources {
		config := &ProviderConfig{
			Source:    source,
			Namespace: "test",
		}
		
		_, err := factory.CreateProvider(config)
		assert.Error(t, err)
		
		errMsg := err.Error()
		
		// All unimplemented providers should follow same format
		assert.Contains(t, errMsg, "is not yet implemented")
		assert.Contains(t, errMsg, "please use ConfigMap provider instead")
		assert.Contains(t, errMsg, "supported sources:")
		
		// Should be user-friendly (no technical jargon)
		assert.NotContains(t, errMsg, "panic")
		assert.NotContains(t, errMsg, "nil pointer")
		assert.NotContains(t, errMsg, "undefined")
	}
}

// TestProviderCreationSuccess tests successful provider creation
func TestProviderCreationSuccess(t *testing.T) {
	factory := NewDefaultProviderFactory(nil)
	
	config := &ProviderConfig{
		Source:    ConfigSourceConfigMap,
		Namespace: "test",
	}
	
	provider, err := factory.CreateProvider(config)
	
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "configmap", provider.Name())
}