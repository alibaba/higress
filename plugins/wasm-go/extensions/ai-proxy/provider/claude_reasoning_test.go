package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// B1 (issue #4107, Bug 1): when reasoning_effort="high" maps to a fixed
// budget_tokens of 16384 but the client also sends max_tokens=16384, Claude
// rejects the request with 400 "max_tokens must be greater than
// thinking.budget_tokens". The built request must keep budget_tokens
// strictly below max_tokens.
func TestClaudeProvider_ReasoningEffortHigh_ClampsBudgetBelowMaxTokens(t *testing.T) {
	provider := &claudeProvider{
		config: ProviderConfig{claudeCodeMode: false},
	}
	request := &chatCompletionRequest{
		Model:           "claude-sonnet-4-5-20250929",
		MaxTokens:       16384,
		ReasoningEffort: "high",
		Messages: []chatMessage{
			{Role: roleUser, Content: "Hello"},
		},
	}

	claudeReq, err := provider.buildClaudeTextGenRequest(request)
	require.NoError(t, err)

	require.NotNil(t, claudeReq.Thinking)
	assert.Equal(t, "enabled", claudeReq.Thinking.Type)
	// Claude requires max_tokens > thinking.budget_tokens (strict inequality).
	assert.Less(t, claudeReq.Thinking.BudgetTokens, claudeReq.MaxTokens,
		"thinking.budget_tokens must be strictly less than max_tokens, otherwise Claude returns 400")
}

// B2 (issue #4107, Bug 2): standard OpenAI SDKs (e.g. langchain) send the
// thinking budget via extra_body.thinking.budget_tokens (plural). It must be
// parsed (model.go wrongly tagged it singular "budget_token") and mapped onto
// the Claude thinking config, instead of being silently dropped and overwritten
// by reasoning_effort.
func TestClaudeProvider_StandardThinkingBudgetTokens_ParsedAndMapped(t *testing.T) {
	provider := &claudeProvider{
		config: ProviderConfig{claudeCodeMode: false},
	}
	// Standard OpenAI SDK payload: thinking.budget_tokens (plural).
	body := []byte(`{"model":"claude-sonnet-4-5-20250929","max_tokens":16384,"messages":[{"role":"user","content":"Hi"}],"thinking":{"type":"enabled","budget_tokens":8192}}`)
	request := &chatCompletionRequest{}
	require.NoError(t, json.Unmarshal(body, request))
	// The plural "budget_tokens" must be parsed into the thinking param.
	require.NotNil(t, request.Thinking)
	assert.Equal(t, 8192, request.Thinking.BudgetTokens,
		"standard OpenAI field thinking.budget_tokens must be parsed (plural)")

	claudeReq, err := provider.buildClaudeTextGenRequest(request)
	require.NoError(t, err)
	require.NotNil(t, claudeReq.Thinking)
	assert.Equal(t, "enabled", claudeReq.Thinking.Type)
	assert.Equal(t, 8192, claudeReq.Thinking.BudgetTokens,
		"explicit thinking.budget_tokens must be honored, not overwritten by reasoning_effort")
}

// B3 (issue #4107): an explicit thinking budget (via the standard OpenAI
// "thinking" field) that is >= max_tokens must also be clamped strictly
// below max_tokens, exactly like the reasoning_effort-derived path.
func TestClaudeProvider_ExplicitThinkingBudgetTokens_ClampedBelowMaxTokens(t *testing.T) {
	provider := &claudeProvider{
		config: ProviderConfig{claudeCodeMode: false},
	}
	// Explicit standard OpenAI thinking budget >= max_tokens.
	body := []byte(`{"model":"claude-sonnet-4-5-20250929","max_tokens":16384,"messages":[{"role":"user","content":"Hi"}],"thinking":{"type":"enabled","budget_tokens":16384}}`)
	request := &chatCompletionRequest{}
	require.NoError(t, json.Unmarshal(body, request))
	claudeReq, err := provider.buildClaudeTextGenRequest(request)
	require.NoError(t, err)
	require.NotNil(t, claudeReq.Thinking)
	assert.Equal(t, "enabled", claudeReq.Thinking.Type)
	// Claude requires max_tokens > thinking.budget_tokens (strict inequality).
	assert.Less(t, claudeReq.Thinking.BudgetTokens, claudeReq.MaxTokens,
		"explicit thinking.budget_tokens must be clamped below max_tokens")
}

// D1 (maintainer review): an explicit thinking.type="disabled" must be
// honored and must NOT be reopened by reasoning_effort. Explicit configuration
// takes priority over the derived one.
func TestClaudeProvider_ExplicitThinkingDisabled_NotReopenedByReasoningEffort(t *testing.T) {
	provider := &claudeProvider{
		config: ProviderConfig{claudeCodeMode: false},
	}
	// Explicitly disabled, yet a reasoning_effort is also present.
	body := []byte(`{"model":"claude-sonnet-4-5-20250929","max_tokens":16384,"messages":[{"role":"user","content":"Hi"}],"thinking":{"type":"disabled"},"reasoning_effort":"high"}`)
	request := &chatCompletionRequest{}
	require.NoError(t, json.Unmarshal(body, request))

	claudeReq, err := provider.buildClaudeTextGenRequest(request)
	require.NoError(t, err)
	require.NotNil(t, claudeReq.Thinking, "explicit disabled thinking must be preserved")
	assert.Equal(t, "disabled", claudeReq.Thinking.Type,
		"explicitly disabled thinking must not be reopened by reasoning_effort")
}

// D2 (maintainer review): in Claude Code (interleaved-thinking) mode Anthropic
// permits budget_tokens to exceed max_tokens (the budget is the total for the
// whole turn). The unified clamp must NOT silently shrink such a request.
func TestClaudeProvider_InterleavedThinking_NotClamped(t *testing.T) {
	provider := &claudeProvider{
		config: ProviderConfig{claudeCodeMode: true},
	}
	// This is the reported regression: Claude Code mode with tools and an
	// explicit Claude-native interleaved thinking budget above max_tokens.
	body := []byte(`{"model":"claude-sonnet-4-5-20250929","max_tokens":4096,"messages":[{"role":"user","content":"Hi"}],"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],"claude_thinking":{"type":"enabled","budget_tokens":16384}}`)
	request := &chatCompletionRequest{}
	require.NoError(t, json.Unmarshal(body, request))

	claudeReq, err := provider.buildClaudeTextGenRequest(request)
	require.NoError(t, err)
	require.NotNil(t, claudeReq.Thinking)
	assert.Equal(t, "enabled", claudeReq.Thinking.Type)
	assert.Equal(t, 16384, claudeReq.Thinking.BudgetTokens,
		"interleaved thinking budget_tokens must not be clamped below max_tokens")
	assert.Greater(t, claudeReq.Thinking.BudgetTokens, claudeReq.MaxTokens,
		"interleaved thinking legitimately allows budget_tokens > max_tokens")
	assert.Len(t, claudeReq.Tools, 1, "the interleaved-thinking exception is exercised with tools")
}

// D3 (maintainer review): when max_tokens is too small to enable thinking
// (max_tokens <= 1024), an explicitly requested thinking config has no valid
// budget in [1024, max_tokens-1] and must fail fast with a clear error instead
// of handing an invalid request to the upstream (which would return a cryptic
// HTTP 400). Note: the derived reasoning_effort path instead silently disables
// thinking in this case to stay consistent with the upstream baseline behavior.
func TestClaudeProvider_MaxTokensTooSmall_ReturnsError(t *testing.T) {
	provider := &claudeProvider{
		config: ProviderConfig{claudeCodeMode: false},
	}

	// Explicit thinking budget >= max_tokens with max_tokens=1024.
	body := []byte(`{"model":"claude-sonnet-4-5-20250929","max_tokens":1024,"messages":[{"role":"user","content":"Hi"}],"thinking":{"type":"enabled","budget_tokens":8192}}`)
	request := &chatCompletionRequest{}
	require.NoError(t, json.Unmarshal(body, request))
	_, err := provider.buildClaudeTextGenRequest(request)
	assert.Error(t, err, "explicit budget_tokens >= max_tokens=1024 must error locally")

	// Even a budget below max_tokens is invalid here because manual thinking
	// requires a minimum budget of 1024, leaving no valid interval.
	body = []byte(`{"model":"claude-sonnet-4-5-20250929","max_tokens":1024,"messages":[{"role":"user","content":"Hi"}],"thinking":{"type":"enabled","budget_tokens":512}}`)
	request2 := &chatCompletionRequest{}
	require.NoError(t, json.Unmarshal(body, request2))
	_, err = provider.buildClaudeTextGenRequest(request2)
	assert.Error(t, err, "all enabled-thinking requests with max_tokens=1024 must fail locally")
}

// D4 (maintainer review): the 1024/1025 boundary. At max_tokens=1025 the clamp
// can produce a valid budget (1024), so the request must succeed and be clamped
// strictly below max_tokens.
func TestClaudeProvider_MaxTokensBoundary_1025Clamps(t *testing.T) {
	provider := &claudeProvider{
		config: ProviderConfig{claudeCodeMode: false},
	}
	request := &chatCompletionRequest{
		Model:           "claude-sonnet-4-5-20250929",
		MaxTokens:       1025,
		ReasoningEffort: "high", // budget 16384
		Messages:        []chatMessage{{Role: roleUser, Content: "Hi"}},
	}
	claudeReq, err := provider.buildClaudeTextGenRequest(request)
	require.NoError(t, err)
	require.NotNil(t, claudeReq.Thinking)
	assert.Equal(t, "enabled", claudeReq.Thinking.Type)
	assert.Equal(t, 1024, claudeReq.Thinking.BudgetTokens,
		"budget must be clamped to max_tokens-1 = 1024")
	assert.Less(t, claudeReq.Thinking.BudgetTokens, claudeReq.MaxTokens)
}

// D5 (maintainer review): when both the singular "budget_token" and the
// standard plural "budget_tokens" are present, the plural (standard) form must
// win — the two must NOT be summed.
func TestClaudeProvider_DualBudgetAliases_PrefersPlural(t *testing.T) {
	provider := &claudeProvider{
		config: ProviderConfig{claudeCodeMode: false},
	}
	body := []byte(`{"model":"claude-sonnet-4-5-20250929","max_tokens":16384,"messages":[{"role":"user","content":"Hi"}],"thinking":{"type":"enabled","budget_token":4096,"budget_tokens":8192}}`)
	request := &chatCompletionRequest{}
	require.NoError(t, json.Unmarshal(body, request))
	require.NotNil(t, request.Thinking)

	claudeReq, err := provider.buildClaudeTextGenRequest(request)
	require.NoError(t, err)
	require.NotNil(t, claudeReq.Thinking)
	assert.Equal(t, 8192, claudeReq.Thinking.BudgetTokens,
		"plural budget_tokens must win over singular budget_token; must not be summed (4096+8192)")
}

// D6 (maintainer review): exercise the real TransformRequestBody path and
// verify the wire JSON sent upstream.
func TestClaudeProvider_TransformRequestBody_ThinkingWireJSON(t *testing.T) {
	provider := &claudeProvider{
		config: ProviderConfig{claudeCodeMode: false},
	}

	// Enabled thinking via standard budget_tokens.
	enabledBody := []byte(`{"model":"claude-sonnet-4-5-20250929","max_tokens":16384,"messages":[{"role":"user","content":"Hi"}],"thinking":{"type":"enabled","budget_tokens":8192}}`)
	enabledOut, err := provider.TransformRequestBody(newMockMultipartHttpContext(), ApiNameChatCompletion, enabledBody)
	require.NoError(t, err)
	var enabledWire struct {
		Thinking *struct {
			Type         string `json:"type"`
			BudgetTokens int    `json:"budget_tokens,omitempty"`
		} `json:"thinking,omitempty"`
	}
	require.NoError(t, json.Unmarshal(enabledOut, &enabledWire))
	require.NotNil(t, enabledWire.Thinking)
	assert.Equal(t, "enabled", enabledWire.Thinking.Type)
	assert.Equal(t, 8192, enabledWire.Thinking.BudgetTokens)

	// Disabled thinking must round-trip as type=disabled and must not be
	// reopened by a concurrent reasoning_effort.
	disabledBody := []byte(`{"model":"claude-sonnet-4-5-20250929","max_tokens":16384,"messages":[{"role":"user","content":"Hi"}],"thinking":{"type":"disabled"},"reasoning_effort":"high"}`)
	disabledOut, err := provider.TransformRequestBody(newMockMultipartHttpContext(), ApiNameChatCompletion, disabledBody)
	require.NoError(t, err)
	var disabledWire struct {
		Thinking *struct {
			Type string `json:"type"`
		} `json:"thinking,omitempty"`
	}
	require.NoError(t, json.Unmarshal(disabledOut, &disabledWire))
	require.NotNil(t, disabledWire.Thinking)
	assert.Equal(t, "disabled", disabledWire.Thinking.Type)
}
