// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tokenusage

import (
	"bytes"
	"slices"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	CtxKeyInputToken         = "input_token"
	CtxKeyInputTokenDetails  = "input_token_details"
	CtxKeyOutputToken        = "output_token"
	CtxKeyOutputTokenDetails = "output_token_details"
	CtxKeyTotalToken         = "total_token"
	CtxKeyModel              = "model"
	CtxKeyRequestModel       = "request_model"
	CtxKeyChatId             = "chat_id"

	ModelEmpty   = ""
	ModelUnknown = "unknown"

	ChatIdPathOpenAIChatCompletions = "id"
	ChatIdPathOpenAIResponses       = "response.id"
	ChatIdPathGemini                = "responseId"
	ChatIdPathAnthropicMessages     = "message.id"

	ModelPathOpenAIChatCompletions = "model"
	ModelPathOpenAIBatches         = "body.model"
	ModelPathOpenAIResponses       = "response.model"
	ModelPathAnthropicMessages     = "message.model"
	ModelPathGeminiGenerateContent = "modelVersion"

	UsageInputTokensPathOpenAIChatCompletions = "usage.prompt_tokens"
	UsageInputTokensPathOpenAIImages          = "usage.input_tokens"
	UsageInputTokensPathOpenAIResponses       = "response.usage.input_tokens"
	UsageInputTokensPathGemini                = "usageMetadata.promptTokenCount"
	UsageInputTokensPathAnthropicMessages     = "message.usage.input_tokens"

	UsageCacheCreationInputTokensPathAnthropicMessages = "usage.cache_creation_input_tokens"
	UsageCacheReadInputTokensPathAnthropicMessages     = "usage.cache_read_input_tokens"

	UsageInputTokensDetailsPathOpenAIChatCompletions = "usage.prompt_tokens_details"
	UsageInputTokensDetailsPathOpenAIResponses       = "response.usage.input_tokens_details"
	UsageInputTokensDetailsPathDoubao                = "usage.input_tokens_details"
	UsageInputTokensDetailsPathGemini                = "usageMetadata.promptTokensDetails"

	UsageOutputTokensPathOpenAIChatCompletions = "usage.completion_tokens"
	UsageOutputTokensPathOpenAIImages          = "usage.output_tokens"
	UsageOutputTokensPathOpenAIResponses       = "response.usage.output_tokens"
	UsageOutputTokensPathGemini                = "usageMetadata.candidatesTokenCount"
	UsageOutputTokensPathAnthropicMessages     = "message.usage.output_tokens"

	UsageMetadataThoughtsTokenCountPathGemini      = "usageMetadata.thoughtsTokenCount"
	UsageMetadataCachedContentTokenCountPathGemini = "usageMetadata.cachedContentTokenCount"
	UsageMetadataToolUsePromptTokenCountPathGemini = "usageMetadata.toolUsePromptTokenCount"
	UsageGeneratedImagesPathDoubao                 = "usage.generated_images"

	UsageOutputTokensDetailsPathOpenAIChatCompletions = "usage.completion_tokens_details"
	UsageOutputTokensDetailsPathOpenAIResponses       = "response.usage.output_tokens_details"
	UsageOutputTokensDetailsPathDoubao                = "usage.output_tokens_details"
	UsageOutputTokensDetailsPathGemini                = "usageMetadata.candidatesTokensDetails"

	UsageTotalTokensPathOpenAIChatCompletions = "usage.total_tokens"
	UsageTotalTokensPathOpenAIResponses       = "response.usage.total_tokens"
	UsageTotalTokensPathGemini                = "usageMetadata.totalTokenCount"

	InputTokenDetailsKeyAnthropicMessagesUsageCacheCreationInputTokens = "cache_creation_input_tokens"
	InputTokenDetailsKeyAnthropicMessagesUsageCacheReadInputTokens     = "cache_read_input_tokens"
	InputTokenDetailsKeyGeminiCachedContentTokenCount                  = "cached_content_token_count"
	InputTokenDetailsKeyGeminiToolUsePromptTokenCount                  = "tool_use_prompt_token_count"

	OutputTokenDetailsKeyDoubaoGeneratedImages    = "generated_images"
	OutputTokenDetailsKeyGeminiThoughtsTokenCount = "thoughts_token_count"

	ctxKeyDeltaSSEMessage = "delta_sse_message"
	ctxKeyDeltaBeginning  = "delta_beginning"
)

type TokenUsage struct {
	InputToken         int64
	InputTokenDetails  map[string]int64
	OutputTokenDetails map[string]int64
	OutputToken        int64
	TotalToken         int64
	Model              string

	// Anthropic Messages
	AnthropicCacheCreationInputToken int64
	AnthropicCacheReadInputToken     int64
}

func GetTokenUsage(ctx wrapper.HttpContext, body []byte) TokenUsage {
	chunks := bytes.SplitSeq(wrapper.UnifySSEChunk(body), []byte("\n\n"))
	u := TokenUsage{
		InputTokenDetails:  make(map[string]int64),
		OutputTokenDetails: make(map[string]int64),
	}
	for chunk := range chunks {
		// the feature strings are used to identify the usage data, like:
		// {"model":"gpt2","usage":{"prompt_tokens":1,"completion_tokens":1}}

		// openai/v1/responses
		chunk = mergeLargeResponseAPIChunks(ctx, chunk)

		if !bytes.Contains(chunk, []byte(`"usage"`)) && !bytes.Contains(chunk, []byte(`"usageMetadata"`)) {
			continue
		}

		ExtractModel(ctx, chunk, &u)
		ExtractInputTokens(ctx, chunk, &u)
		ExtractOutputTokens(ctx, chunk, &u)
		ExtractInputTokenDetails(ctx, chunk, &u)
		ExtractOutputTokenDetails(ctx, chunk, &u)
		ExtractTotalTokens(ctx, chunk, &u)
	}
	return u
}

func mergeLargeResponseAPIChunks(ctx wrapper.HttpContext, chunk []byte) []byte {
	if bytes.Contains(chunk, []byte(`"response.completed"`)) && !bytes.Contains(chunk, []byte(`"usage"`)) {
		ctx.SetContext(ctxKeyDeltaBeginning, true)
	}

	if ctx.GetBoolContext(ctxKeyDeltaBeginning, false) {
		// end of streaming
		if len(bytes.TrimSpace(chunk)) == 0 {
			ctx.SetContext(ctxKeyDeltaBeginning, false)
			chunk = ctx.GetByteSliceContext(ctxKeyDeltaSSEMessage, chunk)
			ctx.SetContext(ctxKeyDeltaSSEMessage, nil)
		} else {
			deltaMessage := ctx.GetByteSliceContext(ctxKeyDeltaSSEMessage, []byte{})
			deltaMessage = append(deltaMessage, chunk...)
			ctx.SetContext(ctxKeyDeltaSSEMessage, deltaMessage)
		}
	}

	return chunk
}

func ExtractModel(ctx wrapper.HttpContext, body []byte, u *TokenUsage) {
	if model := wrapper.GetValueFromBody(body, []string{
		ModelPathOpenAIChatCompletions,
		ModelPathOpenAIBatches,         // batches
		ModelPathOpenAIResponses,       // responses
		ModelPathAnthropicMessages,     // anthropic messages
		ModelPathGeminiGenerateContent, // Gemini GenerateContent
	}); model != nil {
		u.Model = model.String()
	} else if model, ok := ctx.GetUserAttribute(CtxKeyModel).(string); ok && !slices.Contains([]string{ModelEmpty, ModelUnknown}, model) { // anthropic messages
		u.Model = model
	} else if model := ctx.GetStringContext(CtxKeyRequestModel, ModelEmpty); model != ModelEmpty { // Openai Image Generate
		u.Model = model
	} else {
		u.Model = ModelUnknown
	}
	ctx.SetUserAttribute(CtxKeyModel, u.Model)
}

func ExtractInputTokens(ctx wrapper.HttpContext, body []byte, u *TokenUsage) {
	if inputToken := wrapper.GetValueFromBody(body, []string{
		UsageInputTokensPathOpenAIChatCompletions, // completions , chatcompleations
		UsageInputTokensPathOpenAIImages,          // images, audio
		UsageInputTokensPathOpenAIResponses,       // responses
		UsageInputTokensPathGemini,                // Gemini GenerateContent
		UsageInputTokensPathAnthropicMessages,     // Anthrophic messages
	}); inputToken != nil {
		u.InputToken = inputToken.Int()
	} else {
		inputToken, ok := ctx.GetUserAttribute(CtxKeyInputToken).(int64) // anthropic messages
		if ok && inputToken > 0 {
			u.InputToken = inputToken
		}
	}
	ctx.SetUserAttribute(CtxKeyInputToken, u.InputToken)
}

func ExtractOutputTokens(ctx wrapper.HttpContext, body []byte, u *TokenUsage) {
	if outputToken := wrapper.GetValueFromBody(body, []string{
		UsageOutputTokensPathOpenAIChatCompletions, // completions , chatcompleations
		UsageOutputTokensPathOpenAIImages,          // images, audio
		UsageOutputTokensPathOpenAIResponses,       // responses
		UsageOutputTokensPathGemini,                // Gemini GeneratenContent
		UsageOutputTokensPathAnthropicMessages,     // Anthropic messages
	}); outputToken != nil {
		u.OutputToken = outputToken.Int()
	} else {
		outputToken, ok := ctx.GetUserAttribute(CtxKeyOutputToken).(int64)
		if ok && outputToken > 0 {
			u.OutputToken = outputToken
		}
	}
	ctx.SetUserAttribute(CtxKeyOutputToken, u.OutputToken)
}

func ExtractInputTokenDetails(ctx wrapper.HttpContext, body []byte, u *TokenUsage) {
	if inputTokenDetails := wrapper.GetValueFromBody(body, []string{
		UsageInputTokensDetailsPathOpenAIChatCompletions, // chatcompletions
		UsageInputTokensDetailsPathOpenAIResponses,       // responses
		UsageInputTokensDetailsPathDoubao,                // Doubao
		UsageInputTokensDetailsPathGemini,                // Gemini GenerateContent
	}); inputTokenDetails != nil && inputTokenDetails.IsObject() {
		for key, value := range inputTokenDetails.Map() {
			u.InputTokenDetails[key] = value.Int()
		}
	}

	// Gemini GenerateContent
	if geminiCachedContentTokenCount := wrapper.GetValueFromBody(body, []string{
		UsageMetadataCachedContentTokenCountPathGemini,
	}); geminiCachedContentTokenCount != nil {
		u.InputTokenDetails[InputTokenDetailsKeyGeminiCachedContentTokenCount] = geminiCachedContentTokenCount.Int()
	}
	if geminiToolUsePromptTokenCount := wrapper.GetValueFromBody(body, []string{
		UsageMetadataToolUsePromptTokenCountPathGemini,
	}); geminiToolUsePromptTokenCount != nil {
		u.InputTokenDetails[InputTokenDetailsKeyGeminiToolUsePromptTokenCount] = geminiToolUsePromptTokenCount.Int()
	}

	// Anthropic Messages
	if cacheCreationInputToken := wrapper.GetValueFromBody(body, []string{
		UsageCacheCreationInputTokensPathAnthropicMessages,
	}); cacheCreationInputToken != nil {
		u.AnthropicCacheCreationInputToken = cacheCreationInputToken.Int()
		u.InputTokenDetails[InputTokenDetailsKeyAnthropicMessagesUsageCacheCreationInputTokens] = cacheCreationInputToken.Int()
	}
	if cacheReadInputToken := wrapper.GetValueFromBody(body, []string{
		UsageCacheReadInputTokensPathAnthropicMessages,
	}); cacheReadInputToken != nil {
		u.AnthropicCacheReadInputToken = cacheReadInputToken.Int()
		u.InputTokenDetails[InputTokenDetailsKeyAnthropicMessagesUsageCacheReadInputTokens] = cacheReadInputToken.Int()
	}
	ctx.SetUserAttribute(CtxKeyInputTokenDetails, u.InputTokenDetails)
}

func ExtractOutputTokenDetails(ctx wrapper.HttpContext, body []byte, u *TokenUsage) {
	if outputTokensDetails := wrapper.GetValueFromBody(body, []string{
		UsageOutputTokensDetailsPathOpenAIChatCompletions, // completions , chatcompleations
		UsageOutputTokensDetailsPathOpenAIResponses,       // responses
		UsageOutputTokensDetailsPathDoubao,                // doubao
		UsageOutputTokensDetailsPathGemini,                // Gemini GenerateContent
	}); outputTokensDetails != nil && outputTokensDetails.IsObject() {
		for key, val := range outputTokensDetails.Map() {
			u.OutputTokenDetails[key] = val.Int()
		}
	}
	// Gemini GenerateContent
	if geminiThoughtsTokenCount := wrapper.GetValueFromBody(body, []string{
		UsageMetadataThoughtsTokenCountPathGemini,
	}); geminiThoughtsTokenCount != nil {
		u.OutputTokenDetails[OutputTokenDetailsKeyGeminiThoughtsTokenCount] = geminiThoughtsTokenCount.Int()
	}
	// Doubao Image Generate
	if doubaoGeneratedImages := wrapper.GetValueFromBody(body, []string{
		UsageGeneratedImagesPathDoubao,
	}); doubaoGeneratedImages != nil {
		u.OutputTokenDetails[OutputTokenDetailsKeyDoubaoGeneratedImages] = doubaoGeneratedImages.Int()
	}
	ctx.SetUserAttribute(CtxKeyOutputTokenDetails, u.OutputTokenDetails)
}

func ExtractTotalTokens(ctx wrapper.HttpContext, body []byte, u *TokenUsage) {
	if totalToken := wrapper.GetValueFromBody(body, []string{
		UsageTotalTokensPathOpenAIChatCompletions, // completions , chatcompleations, images, audio, responses
		UsageTotalTokensPathOpenAIResponses,       // responses
		UsageTotalTokensPathGemini,                // Gemini GenerationContent
	}); totalToken != nil {
		u.TotalToken = totalToken.Int()
	} else {
		u.TotalToken = u.InputToken + u.OutputToken + u.AnthropicCacheCreationInputToken + u.AnthropicCacheReadInputToken
	}
	ctx.SetUserAttribute(CtxKeyTotalToken, u.TotalToken)
}

func ExtractChatId(ctx wrapper.HttpContext, body []byte) {
	if chatID := wrapper.GetValueFromBody(body, []string{
		ChatIdPathOpenAIChatCompletions,
		ChatIdPathOpenAIResponses,
		ChatIdPathGemini,            // Gemini generateContent
		ChatIdPathAnthropicMessages, // anthropic messages
	}); chatID != nil {
		ctx.SetUserAttribute(CtxKeyChatId, chatID.String())
	}
}
