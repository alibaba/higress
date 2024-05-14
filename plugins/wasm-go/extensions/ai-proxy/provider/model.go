package provider

type chatCompletionRequest struct {
	Model            string        `json:"model"`
	Messages         []chatMessage `json:"messages"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
	FrequencyPenalty float64       `json:"frequency_penalty,omitempty"`
	N                int           `json:"n,omitempty"`
	PresencePenalty  float64       `json:"presence_penalty,omitempty"`
	Seed             int           `json:"seed,omitempty"`
	Stream           bool          `json:"stream,omitempty"`
	Temperature      float64       `json:"temperature,omitempty"`
	TopP             float64       `json:"top_p,omitempty"`
	User             string        `json:"user,omitempty"`
}

type chatCompletionResponse struct {
	Id                string                 `json:"id,omitempty"`
	Choices           []chatCompletionChoice `json:"choices,omitempty"`
	Created           int64                  `json:"created,omitempty"`
	Model             string                 `json:"model,omitempty"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Object            string                 `json:"object,omitempty"`
	Usage             chatCompletionUsage    `json:"usage,omitempty"`
}

type chatCompletionChoice struct {
	Index        int          `json:"index"`
	Message      *chatMessage `json:"message,omitempty"`
	Delta        *chatMessage `json:"delta,omitempty"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

type chatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type chatMessage struct {
	Name    string `json:"name,omitempty"`
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}
