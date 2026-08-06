package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLongcatProviderInitializer_ValidateConfig(t *testing.T) {
	initializer := &longcatProviderInitializer{}

	t.Run("valid_config_with_api_tokens", func(t *testing.T) {
		config := &ProviderConfig{
			apiTokens: []string{"test-token"},
		}
		err := initializer.ValidateConfig(config)
		assert.NoError(t, err)
	})

	t.Run("invalid_config_without_api_tokens", func(t *testing.T) {
		config := &ProviderConfig{
			apiTokens: nil,
		}
		err := initializer.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no apiToken found in provider config")
	})

	t.Run("invalid_config_with_empty_api_tokens", func(t *testing.T) {
		config := &ProviderConfig{
			apiTokens: []string{},
		}
		err := initializer.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no apiToken found in provider config")
	})
}

func TestLongcatProviderInitializer_DefaultCapabilities(t *testing.T) {
	initializer := &longcatProviderInitializer{}

	capabilities := initializer.DefaultCapabilities()
	expected := map[string]string{
		string(ApiNameChatCompletion): PathOpenAIChatCompletions,
		string(ApiNameEmbeddings):     PathOpenAIEmbeddings,
		string(ApiNameModels):         PathOpenAIModels,
	}

	assert.Equal(t, expected, capabilities)
}

func TestLongcatProviderInitializer_CreateProvider(t *testing.T) {
	initializer := &longcatProviderInitializer{}

	config := ProviderConfig{
		apiTokens: []string{"test-token"},
	}

	provider, err := initializer.CreateProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	assert.Equal(t, providerTypeLongcat, provider.GetProviderType())

	longcatProvider, ok := provider.(*longcatProvider)
	require.True(t, ok)
	assert.NotNil(t, longcatProvider.config.apiTokens)
	assert.Equal(t, []string{"test-token"}, longcatProvider.config.apiTokens)
}

func TestLongcatProvider_GetProviderType(t *testing.T) {
	provider := &longcatProvider{
		config: ProviderConfig{
			apiTokens: []string{"test-token"},
		},
		contextCache: createContextCache(&ProviderConfig{}),
	}

	assert.Equal(t, providerTypeLongcat, provider.GetProviderType())
}

func TestLongcatProvider_IsSupportedAPI(t *testing.T) {
	provider := &longcatProvider{
		config: ProviderConfig{
			capabilities: map[string]string{
				string(ApiNameChatCompletion): PathOpenAIChatCompletions,
				string(ApiNameEmbeddings):     PathOpenAIEmbeddings,
			},
		},
	}

	t.Run("supported_api", func(t *testing.T) {
		assert.True(t, provider.config.isSupportedAPI(ApiNameChatCompletion))
		assert.True(t, provider.config.isSupportedAPI(ApiNameEmbeddings))
	})

	t.Run("unsupported_api", func(t *testing.T) {
		assert.False(t, provider.config.isSupportedAPI(ApiName("unsupported")))
		assert.False(t, provider.config.isSupportedAPI(ApiNameModels))
	})
}

func TestLongcatProvider_TransformRequestBody(t *testing.T) {
	t.Run("with_response_schema", func(t *testing.T) {
		provider := &longcatProvider{
			config: ProviderConfig{
				apiTokens: []string{"test-token"},
				responseJsonSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"answer": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		}

		requestBody := `{"model":"test","messages":[{"role":"user","content":"Hello"}]}`

		result, err := provider.TransformRequestBody(nil, ApiNameChatCompletion, []byte(requestBody))
		require.NoError(t, err)

		var transformedRequest chatCompletionRequest
		err = json.Unmarshal(result, &transformedRequest)
		require.NoError(t, err)

		assert.Equal(t, provider.config.responseJsonSchema, transformedRequest.ResponseFormat)
	})

	t.Run("invalid_json_request", func(t *testing.T) {
		provider := &longcatProvider{
			config: ProviderConfig{
				responseJsonSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		requestBody := `invalid json`

		_, err := provider.TransformRequestBody(nil, ApiNameChatCompletion, []byte(requestBody))
		assert.Error(t, err)
	})

	t.Run("without_response_schema", func(t *testing.T) {
		provider := &longcatProvider{
			config: ProviderConfig{
				apiTokens: []string{"test-token"},
			},
		}

		requestBody := `{"model":"test","messages":[{"role":"user","content":"Hello"}]}`

		result, err := provider.TransformRequestBody(nil, ApiNameChatCompletion, []byte(requestBody))
		assert.NoError(t, err)

		var transformedRequest chatCompletionRequest
		err = json.Unmarshal(result, &transformedRequest)
		require.NoError(t, err)

		// Without response schema, the request should remain unchanged
		assert.Nil(t, transformedRequest.ResponseFormat)
	})
}

func TestLongcatProvider_Integration(t *testing.T) {
	// Test the complete flow from initialization to basic functionality
	initializer := &longcatProviderInitializer{}

	config := ProviderConfig{
		apiTokens: []string{"test-token-123"},
	}

	provider, err := initializer.CreateProvider(config)
	require.NoError(t, err)

	// Test provider type
	assert.Equal(t, providerTypeLongcat, provider.GetProviderType())

	// Test capabilities are set correctly
	longcatProvider, ok := provider.(*longcatProvider)
	require.True(t, ok)

	expectedCapabilities := map[string]string{
		string(ApiNameChatCompletion): PathOpenAIChatCompletions,
		string(ApiNameEmbeddings):     PathOpenAIEmbeddings,
		string(ApiNameModels):         PathOpenAIModels,
	}
	assert.Equal(t, expectedCapabilities, longcatProvider.config.capabilities)

	// Test API support
	assert.True(t, longcatProvider.config.isSupportedAPI(ApiNameChatCompletion))
	assert.True(t, longcatProvider.config.isSupportedAPI(ApiNameEmbeddings))
	assert.True(t, longcatProvider.config.isSupportedAPI(ApiNameModels))
	assert.False(t, longcatProvider.config.isSupportedAPI(ApiName("unsupported")))
}

// Test constants
func TestLongcatConstants(t *testing.T) {
	assert.Equal(t, "api.longcat.chat", longcatDomain)
	assert.Equal(t, "longcat", providerTypeLongcat)
}
