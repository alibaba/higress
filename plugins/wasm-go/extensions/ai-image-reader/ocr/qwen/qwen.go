package qwen

import (
	"ai-image-reader/ocr"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"net/http"
)

const (
	queryUrl  string = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	minPixels int    = 3136
	maxPixels int    = 1003520
)

type QwenOcrReq struct {
	Model    string        `json:"model,omitempty"`
	Messages []chatMessage `json:"messages,omitempty"`
}

type QwenOcrResp struct {
	Choices []chatCompletionChoice `json:"choices"`
}

type chatCompletionChoice struct {
	Message *chatMessageContent `json:"message,omitempty"`
}

type chatMessageContent struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type chatMessage struct {
	Role    string    `json:"role"`
	Content []content `json:"content"`
}

type imageURL struct {
	URL string `json:"url"`
}

type content struct {
	Type      string   `json:"type"`
	ImageUrl  imageURL `json:"image_url,omitempty"`
	MinPixels int      `json:"min_pixels,omitempty"`
	MaxPixels int      `json:"max_pixels,omitempty"`
	Text      string   `json:"text,omitempty"`
}

type QwenOcr struct {
	apiKey             string
	model              string
	timeoutMillisecond uint32
	client             wrapper.HttpClient
}

func NewQwenOcr(config *gjson.Result) (*QwenOcr, error) {
	ocr := &QwenOcr{}
	ocr.apiKey = config.Get("apiKey").String()
	if ocr.apiKey == "" {
		return nil, errors.New("apiKey not found")
	}
	serviceName := config.Get("serviceName").String()
	if serviceName == "" {
		return nil, errors.New("serviceName not found")
	}
	servicePort := config.Get("servicePort").Int()
	if servicePort == 0 {
		return nil, errors.New("servicePort not found")
	}
	ocr.model = config.Get("model").String()
	if ocr.model == "" {
		return nil, errors.New("model not found")
	}
	ocr.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
	})
	ocr.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if ocr.timeoutMillisecond == 0 {
		ocr.timeoutMillisecond = 30000
	}
	return ocr, nil
}

func (q QwenOcr) Client() wrapper.HttpClient {
	return q.client
}

func (q QwenOcr) CallArgs(imageUrl string) ocr.CallArgs {
	reqBody := QwenOcrReq{
		Model: q.model,
		Messages: []chatMessage{
			{
				Role: "user",
				Content: []content{
					{
						Type: "image_url",
						ImageUrl: imageURL{
							URL: imageUrl,
						},
						MinPixels: minPixels,
						MaxPixels: maxPixels,
					},
				},
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	return ocr.CallArgs{
		Method: http.MethodPost,
		Url:    queryUrl,
		Headers: [][2]string{
			{"Content-Type", "application/json"},
			{"Authorization", fmt.Sprintf("Bearer %s", q.apiKey)},
		},
		Body:               body,
		TimeoutMillisecond: q.timeoutMillisecond,
	}
}

func (q QwenOcr) ParseResult(response []byte) string {
	var resp QwenOcrResp
	_ = json.Unmarshal(response, &resp)
	return resp.Choices[0].Message.Content
}
