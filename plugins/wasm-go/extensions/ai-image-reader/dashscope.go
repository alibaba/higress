package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	DashscopeDomain           = "dashscope.aliyuncs.com"
	DashscopePort             = 443
	DashscopeDefaultModelName = "qwen-vl-ocr"
	DashscopeEndpoint         = "/compatible-mode/v1/chat/completions"
	MinPixels                 = 3136
	MaxPixels                 = 1003520
)

type OcrReq struct {
	Model    string        `json:"model,omitempty"`
	Messages []chatMessage `json:"messages,omitempty"`
}

type OcrResp struct {
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

var dashScopeConfig dashScopeProviderConfig

type dashScopeProviderInitializer struct {
}

func (d *dashScopeProviderInitializer) InitConfig(json gjson.Result) {
	dashScopeConfig.apiKey = json.Get("apiKey").String()
}

func (d *dashScopeProviderInitializer) ValidateConfig() error {
	if dashScopeConfig.apiKey == "" {
		return errors.New("[DashScope] apiKey is required")
	}
	return nil
}

func (d *dashScopeProviderInitializer) CreateProvider(c ProviderConfig) (Provider, error) {
	if c.servicePort == 0 {
		c.servicePort = DashscopePort
	}
	if c.serviceHost == "" {
		c.serviceHost = DashscopeDomain
	}
	return &DSProvider{
		config: c,
		client: wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: c.serviceName,
			Host: c.serviceHost,
			Port: int64(c.servicePort),
		}),
	}, nil
}

type dashScopeProviderConfig struct {
	// @Title zh-CN 文字识别服务 API Key
	// @Description zh-CN 文字识别服务 API Key
	apiKey string
}

type DSProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (d *DSProvider) GetProviderType() string {
	return ProviderTypeDashscope
}

func (d *DSProvider) CallArgs(imageUrl string) CallArgs {
	model := d.config.model
	if model == "" {
		model = DashscopeDefaultModelName
	}
	reqBody := OcrReq{
		Model: model,
		Messages: []chatMessage{
			{
				Role: "user",
				Content: []content{
					{
						Type: "image_url",
						ImageUrl: imageURL{
							URL: imageUrl,
						},
						MinPixels: MinPixels,
						MaxPixels: MaxPixels,
					},
				},
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	return CallArgs{
		Method: http.MethodPost,
		Url:    DashscopeEndpoint,
		Headers: [][2]string{
			{"Content-Type", "application/json"},
			{"Authorization", fmt.Sprintf("Bearer %s", dashScopeConfig.apiKey)},
		},
		Body:               body,
		TimeoutMillisecond: d.config.timeout,
	}
}

func (d *DSProvider) parseOcrResponse(responseBody []byte) (*OcrResp, error) {
	var resp OcrResp
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (d *DSProvider) DoOCR(
	imageUrl string,
	callback func(imageContent string, err error)) error {
	args := d.CallArgs(imageUrl)
	err := d.client.Call(args.Method, args.Url, args.Headers, args.Body,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				err := errors.New("failed to do ocr due to status code: " + strconv.Itoa(statusCode))
				callback("", err)
				return
			}
			log.Debugf("do ocr response: %d, %s", statusCode, responseBody)
			resp, err := d.parseOcrResponse(responseBody)
			if err != nil {
				err = fmt.Errorf("failed to parse response: %v", err)
				callback("", err)
				return
			}
			if len(resp.Choices) == 0 {
				err = errors.New("no ocr response found")
				callback("", err)
				return
			}
			callback(resp.Choices[0].Message.Content, nil)
		}, args.TimeoutMillisecond)
	return err
}
