package provider

import "strings"

const (
	streamEventIdItemKey        = "id:"
	streamEventNameItemKey      = "event:"
	streamBuiltInItemKey        = ":"
	streamHttpStatusValuePrefix = "HTTP_STATUS/"
	streamDataItemKey           = "data:"
	streamEndDataValue          = "[DONE]"

	eventResult = "result"

	httpStatus200 = "200"
)

type chatCompletionRequest struct {
	Model            string         `json:"model"`
	Messages         []chatMessage  `json:"messages"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	FrequencyPenalty float64        `json:"frequency_penalty,omitempty"`
	N                int            `json:"n,omitempty"`
	PresencePenalty  float64        `json:"presence_penalty,omitempty"`
	Seed             int            `json:"seed,omitempty"`
	Stream           bool           `json:"stream,omitempty"`
	StreamOptions    *streamOptions `json:"stream_options,omitempty"`
	Temperature      float64        `json:"temperature,omitempty"`
	TopP             float64        `json:"top_p,omitempty"`
	Tools            []tool         `json:"tools,omitempty"`
	ToolChoice       *toolChoice    `json:"tool_choice,omitempty"`
	User             string         `json:"user,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
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
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Object            string                 `json:"object,omitempty"`
	Usage             usage                  `json:"usage,omitempty"`
}

type chatCompletionChoice struct {
	Index        int          `json:"index"`
	Message      *chatMessage `json:"message,omitempty"`
	Delta        *chatMessage `json:"delta,omitempty"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type chatMessage struct {
	Name      string     `json:"name,omitempty"`
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

func (m *chatMessage) IsEmpty() bool {
	if m.Content != "" {
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

type streamEvent struct {
	Id         string `json:"id"`
	Event      string `json:"event"`
	Data       string `json:"data"`
	HttpStatus string `json:"http_status"`
}

func (e *streamEvent) setValue(key, value string) {
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
