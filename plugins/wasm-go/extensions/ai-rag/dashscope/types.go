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
