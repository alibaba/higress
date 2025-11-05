package dashscope

// DashScope embedding service: Request
type Request struct {
	Model     string    `json:"model"`
	Input     Input     `json:"input"`
	Parameter Parameter `json:"parameters"`
}

type Input struct {
	Texts []string `json:"texts"`
}

type Parameter struct {
	TextType string `json:"text_type"`
}

// DashScope embedding service: Response
type Response struct {
	Output    Output `json:"output"`
	Usage     Usage  `json:"usage"`
	RequestID string `json:"request_id"`
}

type Output struct {
	Embeddings []Embedding `json:"embeddings"`
}

type Embedding struct {
	Embedding []float32 `json:"embedding"`
	TextIndex int32     `json:"text_index"`
}

type Usage struct {
	TotalTokens int32 `json:"total_tokens"`
}

// completion
type Completion struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int64     `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionResponse struct {
	Choices           []Choice        `json:"choices"`
	Object            string          `json:"object"`
	Usage             CompletionUsage `json:"usage"`
	Created           string          `json:"created"`
	SystemFingerprint string          `json:"system_fingerprint"`
	Model             string          `json:"model"`
	ID                string          `json:"id"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
}

type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
