package utils

const (
	OpenAIResponseFormat       = `{"id": "%s","object":"chat.completion","model":"from-security-guard","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	OpenAIStreamResponseChunk  = `data:{"id":"%s","object":"chat.completion.chunk","model":"from-security-guard","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]}`
	OpenAIStreamResponseEnd    = `data:{"id":"%s","object":"chat.completion.chunk","model":"from-security-guard","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	OpenAIStreamResponseFormat = OpenAIStreamResponseChunk + "\n\n" + OpenAIStreamResponseEnd + "\n\n" + `data: [DONE]`

	DefaultDenyCode    = 200
	DefaultDenyMessage = "很抱歉，我无法回答您的问题"
	DefaultTimeout     = 2000

	AliyunUserAgent = "CIPFrom/AIGateway"
	LengthLimit     = 1800
)
