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
	DASHSCOPE_DOMAIN             = "dashscope.aliyuncs.com"
	DASHSCOPE_PORT               = 443
	DASHSCOPE_DEFAULT_MODEL_NAME = "text-embedding-v2"
	DASHSCOPE_ENDPOINT           = "/api/v1/services/embeddings/text-embedding/text-embedding"
)

var dashScopeConfig dashScopeProviderConfig

type dashScopeProviderInitializer struct {
}
type dashScopeProviderConfig struct {
	// @Title zh-CN 文本特征提取服务 API Key
	// @Description zh-CN 文本特征提取服务 API Key
	apiKey string
	model  string
}

func (c *dashScopeProviderInitializer) InitConfig(config config.EmbeddingConfig) {
	dashScopeConfig.apiKey = config.APIKey
	dashScopeConfig.model = config.Model
}

func (c *dashScopeProviderInitializer) ValidateConfig() error {
	if dashScopeConfig.apiKey == "" {
		return errors.New("[DashScope] apiKey is required")
	}
	return nil
}

func (c *dashScopeProviderInitializer) CreateProvider(config config.EmbeddingConfig) (Provider, error) {
	c.InitConfig(config)
	err := c.ValidateConfig()
	if err != nil {
		return nil, err
	}

	// 创建HTTP客户端
	headers := map[string]string{
		"Authorization": "Bearer " + config.APIKey,
		"Content-Type":  "application/json",
	}
	httpClient := common.NewHTTPClient(fmt.Sprintf("https://%s", DASHSCOPE_DOMAIN), headers)

	return &DashScopeProvider{
		config: dashScopeConfig,
		client: httpClient,
	}, nil
}

func (d *DashScopeProvider) GetProviderType() string {
	return PROVIDER_TYPE_DASHSCOPE
}

type Embedding struct {
	Embedding []float64 `json:"embedding"`
	TextIndex int       `json:"text_index"`
}

type Input struct {
	Texts []string `json:"texts"`
}

type Params struct {
	TextType string `json:"text_type"`
}

type Response struct {
	RequestID string `json:"request_id"`
	Output    Output `json:"output"`
	Usage     Usage  `json:"usage"`
}

type Output struct {
	Embeddings []Embedding `json:"embeddings"`
}

type Usage struct {
	TotalTokens int `json:"total_tokens"`
}

type EmbeddingRequest struct {
	Model      string `json:"model"`
	Input      Input  `json:"input"`
	Parameters Params `json:"parameters"`
}

type Document struct {
	Vector []float64         `json:"vector"`
	Fields map[string]string `json:"fields"`
}

type DashScopeProvider struct {
	config dashScopeProviderConfig
	client *common.HTTPClient
}

func (d *DashScopeProvider) constructRequestData(texts []string) (EmbeddingRequest, error) {
	model := d.config.model
	if model == "" {
		model = DASHSCOPE_DEFAULT_MODEL_NAME
	}

	if dashScopeConfig.apiKey == "" {
		return EmbeddingRequest{}, errors.New("dashScopeKey is empty")
	}

	data := EmbeddingRequest{
		Model: model,
		Input: Input{
			Texts: texts,
		},
		Parameters: Params{
			TextType: "query",
		},
	}

	return data, nil
}

type Result struct {
	ID     string                 `json:"id"`
	Vector []float64              `json:"vector,omitempty"`
	Fields map[string]interface{} `json:"fields"`
	Score  float64                `json:"score"`
}

func (d *DashScopeProvider) parseTextEmbedding(responseBody []byte) (*Response, error) {
	var resp Response
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (d *DashScopeProvider) GetEmbedding(
	ctx context.Context,
	queryString string) ([]float64, error) {
	// 构造请求数据
	requestData, err := d.constructRequestData([]string{queryString})
	if err != nil {
		return nil, fmt.Errorf("failed to construct request data: %v", err)
	}

	// 发送POST请求
	responseBody, err := d.client.Post(DASHSCOPE_ENDPOINT, requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// 解析响应
	embeddingResp, err := d.parseTextEmbedding(responseBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// 检查是否有embedding结果
	if len(embeddingResp.Output.Embeddings) == 0 {
		return nil, errors.New("no embedding found in response")
	}

	return embeddingResp.Output.Embeddings[0].Embedding, nil
}
