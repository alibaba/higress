package embedding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/common"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

const (
	OPENAI_DOMAIN             = "api.openai.com"
	OPENAI_PORT               = 443
	OPENAI_DEFAULT_MODEL_NAME = "text-embedding-3-small"
	OPENAI_ENDPOINT           = "/v1/embeddings"
)

type openAIProviderInitializer struct {
}

var openAIConfig openAIProviderConfig

type openAIProviderConfig struct {
	// @Title zh-CN 文本特征提取服务 API Key
	// @Description zh-CN 文本特征提取服务 API Key
	baseUrl string
	apiKey  string
	model   string
}

func (c *openAIProviderInitializer) InitConfig(config config.EmbeddingConfig) {
	openAIConfig.apiKey = config.APIKey
	openAIConfig.model = config.Model
	openAIConfig.baseUrl = config.BaseURL
}

func (c *openAIProviderInitializer) ValidateConfig() error {
	if openAIConfig.apiKey == "" {
		return errors.New("[openAI] apiKey is required")
	}
	return nil
}

func (c *openAIProviderInitializer) CreateProvider(config config.EmbeddingConfig) (Provider, error) {
	c.InitConfig(config)
	err := c.ValidateConfig()
	if err != nil {
		return nil, err
	}

	if openAIConfig.model == "" {
		openAIConfig.model = OPENAI_DEFAULT_MODEL_NAME
	}

	if openAIConfig.baseUrl == "" {
		openAIConfig.baseUrl = fmt.Sprintf("https://%s", OPENAI_DOMAIN)
	}

	// 创建HTTP客户端
	headers := map[string]string{
		"Authorization": "Bearer " + config.APIKey,
		"Content-Type":  "application/json",
	}
	httpClient := common.NewHTTPClient(openAIConfig.baseUrl, headers)

	return &OpenAIProvider{
		config: openAIConfig,
		client: httpClient,
	}, nil
}

func (o *OpenAIProvider) GetProviderType() string {
	return PROVIDER_TYPE_OPENAI
}

type OpenAIResponse struct {
	Object string         `json:"object"`
	Data   []OpenAIResult `json:"data"`
	Model  string         `json:"model"`
	Error  *OpenAIError   `json:"error"`
}

type OpenAIResult struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type OpenAIError struct {
	Message string `json:"prompt_tokens"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Param   string `json:"param"`
}

type OpenAIEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type OpenAIProvider struct {
	config openAIProviderConfig
	client *common.HTTPClient
}

func (o *OpenAIProvider) constructRequestData(text string) (OpenAIEmbeddingRequest, error) {
	if text == "" {
		return OpenAIEmbeddingRequest{}, errors.New("queryString text cannot be empty")
	}

	if openAIConfig.apiKey == "" {
		return OpenAIEmbeddingRequest{}, errors.New("openAI apiKey is empty")
	}

	model := o.config.model
	if model == "" {
		model = OPENAI_DEFAULT_MODEL_NAME
	}

	data := OpenAIEmbeddingRequest{
		Input: text,
		Model: model,
	}

	return data, nil
}

func (o *OpenAIProvider) parseTextEmbedding(responseBody []byte) (*OpenAIResponse, error) {
	var resp OpenAIResponse
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (o *OpenAIProvider) GetEmbedding(ctx context.Context, queryString string) ([]float32, error) {
	// 构造请求数据
	requestData, err := o.constructRequestData(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request data: %v", err)
	}

	// 发送POST请求
	responseBody, err := o.client.Post(OPENAI_ENDPOINT, requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// 解析响应
	resp, err := o.parseTextEmbedding(responseBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// 检查API响应错误
	if resp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s - %s", resp.Error.Type, resp.Error.Message)
	}

	// 检查是否有embedding结果
	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding found in response")
	}

	return resp.Data[0].Embedding, nil
}
