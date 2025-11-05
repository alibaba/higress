package higress

// APIResponse represents the standard Higress API response format
type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
}
