package dashvector

// DashVecotor document search: Request
type Request struct {
	TopK         int32     `json:"topk"`
	OutputFileds []string  `json:"output_fileds"`
	Vector       []float32 `json:"vector"`
}

// DashVecotor document search: Response
type Response struct {
	Code      int32          `json:"code"`
	RequestID string         `json:"request_id"`
	Message   string         `json:"message"`
	Output    []OutputObject `json:"output"`
}

type OutputObject struct {
	ID     string      `json:"id"`
	Fields FieldObject `json:"fields"`
	Score  float32     `json:"score"`
}

type FieldObject struct {
	Raw string `json:"raw"`
}
