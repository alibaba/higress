package scheduling

// LLMRequest is a structured representation of the fields we parse out of the LLMRequest body.
type LLMRequest struct {
	Model    string
	Critical bool
}
