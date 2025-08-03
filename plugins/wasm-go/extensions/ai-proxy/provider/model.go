package provider

import (
	"fmt"
	"strings"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	streamEventIdItemKey        = "id:"
	streamEventNameItemKey      = "event:"
	streamBuiltInItemKey        = ":"
	streamHttpStatusValuePrefix = "HTTP_STATUS/"
	streamDataItemKey           = "data:"
	streamEndDataValue          = "[DONE]"

	eventResult = "result"

	httpStatus200 = "200"

	contentTypeText       = "text"
	contentTypeImageUrl   = "image_url"
	contentTypeInputAudio = "input_audio"
	contentTypeFile       = "file"

	reasoningStartTag = "<think>"
	reasoningEndTag   = "</think>"
)

type chatCompletionRequest struct {
	Messages            []chatMessage          `json:"messages"`
	Model               string                 `json:"model"`
	Store               bool                   `json:"store,omitempty"`
	ReasoningEffort     string                 `json:"reasoning_effort,omitempty"`
	Metadata            map[string]string      `json:"metadata,omitempty"`
	FrequencyPenalty    float64                `json:"frequency_penalty,omitempty"`
	LogitBias           map[string]int         `json:"logit_bias,omitempty"`
	Logprobs            bool                   `json:"logprobs,omitempty"`
	TopLogprobs         int                    `json:"top_logprobs,omitempty"`
	MaxTokens           int                    `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                    `json:"max_completion_tokens,omitempty"`
	N                   int                    `json:"n,omitempty"`
	Modalities          []string               `json:"modalities,omitempty"`
	Prediction          map[string]interface{} `json:"prediction,omitempty"`
	Audio               map[string]interface{} `json:"audio,omitempty"`
	PresencePenalty     float64                `json:"presence_penalty,omitempty"`
	ResponseFormat      map[string]interface{} `json:"response_format,omitempty"`
	Seed                int                    `json:"seed,omitempty"`
	ServiceTier         string                 `json:"service_tier,omitempty"`
	Stop                []string               `json:"stop,omitempty"`
	Stream              bool                   `json:"stream,omitempty"`
	StreamOptions       *streamOptions         `json:"stream_options,omitempty"`
	Temperature         float64                `json:"temperature,omitempty"`
	TopP                float64                `json:"top_p,omitempty"`
	Tools               []tool                 `json:"tools,omitempty"`
	ToolChoice          interface{}            `json:"tool_choice,omitempty"`
	ParallelToolCalls   bool                   `json:"parallel_tool_calls,omitempty"`
	User                string                 `json:"user,omitempty"`
}

func (c *chatCompletionRequest) getMaxTokens() int {
	if c.MaxCompletionTokens > 0 {
		return c.MaxCompletionTokens
	}
	return c.MaxTokens
}

func (c *chatCompletionRequest) getToolChoiceString() string {
	if c.ToolChoice == nil {
		return ""
	}

	if tc, ok := c.ToolChoice.(string); ok {
		return tc
	}
	return ""
}

func (c *chatCompletionRequest) getToolChoiceObject() *toolChoice {
	if c.ToolChoice == nil {
		return nil
	}

	if tc, ok := c.ToolChoice.(*toolChoice); ok {
		return tc
	}
	return nil
}

type CompletionRequest struct {
	Model            string         `json:"model"`
	Prompt           string         `json:"prompt"`
	BestOf           int            `json:"best_of,omitempty"`
	Echo             bool           `json:"echo,omitempty"`
	FrequencyPenalty float64        `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
	Logprobs         int            `json:"logprobs,omitempty"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	N                int            `json:"n,omitempty"`
	PresencePenalty  float64        `json:"presence_penalty,omitempty"`
	Seed             int            `json:"seed,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	Stream           bool           `json:"stream,omitempty"`
	StreamOptions    *streamOptions `json:"stream_options,omitempty"`
	Suffix           string         `json:"suffix,omitempty"`
	Temperature      float64        `json:"temperature,omitempty"`
	TopP             float64        `json:"top_p,omitempty"`
	User             string         `json:"user,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type tool struct {
	Type     string   `json:"type"`
	Function function `json:"function"`
}

type function struct {
	Description string                 `json:"description,omitempty"`
	Name        string                 `json:"name"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type toolChoice struct {
	Type     string   `json:"type"`
	Function function `json:"function"`
}

type chatCompletionResponse struct {
	Id                string                 `json:"id,omitempty"`
	Choices           []chatCompletionChoice `json:"choices"`
	Created           int64                  `json:"created,omitempty"`
	Model             string                 `json:"model,omitempty"`
	ServiceTier       string                 `json:"service_tier,omitempty"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Object            string                 `json:"object,omitempty"`
	Usage             *usage                 `json:"usage"`
}

type chatCompletionChoice struct {
	Index        int                    `json:"index"`
	Message      *chatMessage           `json:"message,omitempty"`
	Delta        *chatMessage           `json:"delta,omitempty"`
	FinishReason *string                `json:"finish_reason"`
	Logprobs     map[string]interface{} `json:"logprobs"`
}

type usage struct {
	PromptTokens            int                      `json:"prompt_tokens,omitempty"`
	CompletionTokens        int                      `json:"completion_tokens,omitempty"`
	TotalTokens             int                      `json:"total_tokens,omitempty"`
	CompletionTokensDetails *completionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type completionTokensDetails struct {
	ReasoningTokens          int `json:"reasoning_tokens,omitempty"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

type chatMessage struct {
	Id               string                 `json:"id,omitempty"`
	Audio            map[string]interface{} `json:"audio,omitempty"`
	Name             string                 `json:"name,omitempty"`
	Role             string                 `json:"role,omitempty"`
	Content          any                    `json:"content,omitempty"`
	ReasoningContent string                 `json:"reasoning_content,omitempty"`
	ToolCalls        []toolCall             `json:"tool_calls,omitempty"`
	Refusal          string                 `json:"refusal,omitempty"`
}

func (m *chatMessage) handleNonStreamingReasoningContent(reasoningContentMode string) {
	if m.ReasoningContent == "" {
		return
	}
	switch reasoningContentMode {
	case reasoningBehaviorIgnore:
		m.ReasoningContent = ""
		break
	case reasoningBehaviorConcat:
		m.Content = fmt.Sprintf("%s%v%s\n%v", reasoningStartTag, m.ReasoningContent, reasoningEndTag, m.Content)
		m.ReasoningContent = ""
		break
	case reasoningBehaviorPassThrough:
	default:
		break
	}
}

func (m *chatMessage) handleStreamingReasoningContent(ctx wrapper.HttpContext, reasoningContentMode string) {
	switch reasoningContentMode {
	case reasoningBehaviorIgnore:
		m.ReasoningContent = ""
		break
	case reasoningBehaviorConcat:
		contentPushed, _ := ctx.GetContext(ctxKeyContentPushed).(bool)
		reasoningContentPushed, _ := ctx.GetContext(ctxKeyReasoningContentPushed).(bool)

		if contentPushed {
			if m.ReasoningContent != "" {
				// This shouldn't happen, but if it does, we can add a log here.
				log.Warnf("[ai-proxy] Content already pushed, but reasoning content is not empty: %v", m)
			}
			return
		}

		if m.ReasoningContent != "" && !reasoningContentPushed {
			m.ReasoningContent = reasoningStartTag + m.ReasoningContent
			reasoningContentPushed = true
		}
		if m.Content != "" {
			if reasoningContentPushed && !contentPushed /* Keep the second part just to make it easy to understand*/ {
				m.ReasoningContent += reasoningEndTag
			}
			contentPushed = true
		}

		m.Content = fmt.Sprintf("%s\n%v", m.ReasoningContent, m.Content)
		m.ReasoningContent = ""

		ctx.SetContext(ctxKeyContentPushed, contentPushed)
		ctx.SetContext(ctxKeyReasoningContentPushed, reasoningContentPushed)
		break
	case reasoningBehaviorPassThrough:
	default:
		break
	}
}

type chatMessageContent struct {
	Type       string                      `json:"type,omitempty"`
	Text       string                      `json:"text"`
	ImageUrl   *chatMessageContentImageUrl `json:"image_url,omitempty"`
	File       *chatMessageContentFile     `json:"file,omitempty"`
	InputAudio *chatMessageContentAudio    `json:"input_audio,omitempty"`
}

type chatMessageContentAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type chatMessageContentFile struct {
	FileData string `json:"file_data,omitempty"`
	FileId   string `json:"file_id,omitempty"`
	FileName string `json:"file_name,omitempty"`
}

type chatMessageContentImageUrl struct {
	Url    string `json:"url,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func (m *chatMessage) IsEmpty() bool {
	if m.ReasoningContent != "" {
		return false
	}
	if m.IsStringContent() && m.Content != "" {
		return false
	}
	anyList, ok := m.Content.([]any)
	if ok && len(anyList) > 0 {
		return false
	}
	if len(m.ToolCalls) != 0 {
		nonEmpty := false
		for _, toolCall := range m.ToolCalls {
			if !toolCall.Function.IsEmpty() {
				nonEmpty = true
				break
			}
		}
		if nonEmpty {
			return false
		}
	}
	return true
}

func (m *chatMessage) IsStringContent() bool {
	_, ok := m.Content.(string)
	return ok
}

func (m *chatMessage) StringContent() string {
	content, ok := m.Content.(string)
	if ok {
		return content
	}
	contentList, ok := m.Content.([]any)
	if ok {
		var contentStr string
		for _, contentItem := range contentList {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			if contentMap["type"] == contentTypeText {
				if subStr, ok := contentMap[contentTypeText].(string); ok {
					contentStr += subStr + "\n"
				}
			}
		}
		return contentStr
	}
	return ""
}

func (m *chatMessage) ParseContent() []chatMessageContent {
	var contentList []chatMessageContent
	content, ok := m.Content.(string)
	if ok {
		contentList = append(contentList, chatMessageContent{
			Type: contentTypeText,
			Text: content,
		})
		return contentList
	}
	anyList, ok := m.Content.([]any)
	if ok {
		for _, contentItem := range anyList {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			switch contentMap["type"] {
			case contentTypeText:
				if subStr, ok := contentMap[contentTypeText].(string); ok {
					contentList = append(contentList, chatMessageContent{
						Type: contentTypeText,
						Text: subStr,
					})
				}
			case contentTypeImageUrl:
				if subObj, ok := contentMap[contentTypeImageUrl].(map[string]any); ok {
					msg := chatMessageContent{
						Type: contentTypeImageUrl,
						ImageUrl: &chatMessageContentImageUrl{
							Url: subObj["url"].(string),
						},
					}
					if detail, ok := subObj["detail"].(string); ok {
						msg.ImageUrl.Detail = detail
					}
					contentList = append(contentList, msg)
				}
			case contentTypeInputAudio:
				if subObj, ok := contentMap[contentTypeInputAudio].(map[string]any); ok {
					contentList = append(contentList, chatMessageContent{
						Type: contentTypeInputAudio,
						InputAudio: &chatMessageContentAudio{
							Data:   subObj["data"].(string),
							Format: subObj["format"].(string),
						},
					})
				}
			case contentTypeFile:
				if subObj, ok := contentMap[contentTypeFile].(map[string]any); ok {
					contentList = append(contentList, chatMessageContent{
						Type: contentTypeFile,
						File: &chatMessageContentFile{
							FileId: subObj["file_id"].(string),
							// FileName: subObj["file_name"].(string),
							// FileData: subObj["file_data"].(string),
						},
					})
				}
			}
		}
		return contentList
	}
	return nil
}

type toolCall struct {
	Index    int          `json:"index"`
	Id       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (m *functionCall) IsEmpty() bool {
	return m.Name == "" && m.Arguments == ""
}

type StreamEvent struct {
	Id         string `json:"id"`
	Event      string `json:"event"`
	Data       string `json:"data"`
	HttpStatus string `json:"http_status"`
}

func (e *StreamEvent) IsEndData() bool {
	return e.Data == streamEndDataValue
}

func (e *StreamEvent) SetValue(key, value string) {
	switch key {
	case streamEventIdItemKey:
		e.Id = value
	case streamEventNameItemKey:
		e.Event = value
	case streamDataItemKey:
		e.Data = value
	case streamBuiltInItemKey:
		if strings.HasPrefix(value, streamHttpStatusValuePrefix) {
			e.HttpStatus = value[len(streamHttpStatusValuePrefix):]
		}
	}
}

func (e *StreamEvent) ToHttpString() string {
	return fmt.Sprintf("%s %s\n\n", streamDataItemKey, e.Data)
}

// https://platform.openai.com/docs/guides/images
type imageGenerationRequest struct {
	Model             string `json:"model"`
	Prompt            string `json:"prompt"`
	Background        string `json:"background,omitempty"`
	Moderation        string `json:"moderation,omitempty"`
	OutputCompression int    `json:"output_compression,omitempty"`
	OutputFormat      string `json:"output_format,omitempty"`
	Quality           string `json:"quality,omitempty"`
	ResponseFormat    string `json:"response_format,omitempty"`
	Style             string `json:"style,omitempty"`
	N                 int    `json:"n,omitempty"`
	Size              string `json:"size,omitempty"`
}

type imageGenerationData struct {
	URL           string `json:"url,omitempty"`
	B64           string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type imageGenerationUsage struct {
	TotalTokens        int `json:"total_tokens"`
	InputTokens        int `json:"input_tokens"`
	OutputTokens       int `json:"output_tokens"`
	InputTokensDetails struct {
		TextTokens  int `json:"text_tokens"`
		ImageTokens int `json:"image_tokens"`
	} `json:"input_tokens_details"`
}

type imageGenerationResponse struct {
	Created int64                 `json:"created"`
	Data    []imageGenerationData `json:"data"`
	Usage   *imageGenerationUsage `json:"usage,omitempty"`
}

// https://platform.openai.com/docs/guides/speech-to-text
type audioSpeechRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Voice string `json:"voice"`
}

type embeddingsRequest struct {
	Input          interface{} `json:"input"`
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
	Dimensions     int         `json:"dimensions,omitempty"`
	User           string      `json:"user,omitempty"`
}

type embeddingsResponse struct {
	Object string      `json:"object"`
	Data   []embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  usage       `json:"usage"`
}

type embedding struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

func (r embeddingsRequest) ParseInput() []string {
	if r.Input == nil {
		return nil
	}
	var input []string
	switch r.Input.(type) {
	case string:
		input = []string{r.Input.(string)}
	case []any:
		input = make([]string, 0, len(r.Input.([]any)))
		for _, item := range r.Input.([]any) {
			if str, ok := item.(string); ok {
				input = append(input, str)
			}
		}
	}
	return input
}
