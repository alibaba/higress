package embedding

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

const (
	dashScopeDomain = "dashscope.aliyuncs.com"
	dashScopePort   = 443
)

type dashScopeProviderInitializer struct {
}

func (d *dashScopeProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if len(config.DashScopeKey) == 0 {
		return errors.New("DashScopeKey is required")
	}
	if len(config.ServiceName) == 0 {
		return errors.New("ServiceName is required")
	}
	return nil
}

func (d *dashScopeProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &DSProvider{
		config: config,
		client: wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: config.ServiceName,
			Port:        dashScopePort,
			Domain:      dashScopeDomain,
		}),
	}, nil
}

func (d *DSProvider) GetProviderType() string {
	return providerTypeDashScope
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

// EmbeddingRequest 定义请求的数据结构
type EmbeddingRequest struct {
	Model      string `json:"model"`
	Input      Input  `json:"input"`
	Parameters Params `json:"parameters"`
}

// Document 定义每个文档的结构
type Document struct {
	// ID     string            `json:"id"`
	Vector []float64         `json:"vector"`
	Fields map[string]string `json:"fields"`
}

type DSProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (d *DSProvider) constructParameters(texts []string, log wrapper.Log) (string, [][2]string, []byte, error) {
	const (
		endpoint    = "/api/v1/services/embeddings/text-embedding/text-embedding"
		modelName   = "text-embedding-v1"
		contentType = "application/json"
	)

	// 构造请求数据
	data := EmbeddingRequest{
		Model: modelName,
		Input: Input{
			Texts: texts,
		},
		Parameters: Params{
			TextType: "query",
		},
	}

	// 序列化请求体并处理错误
	requestBody, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Failed to marshal request data: %v", err)
		return "", nil, nil, err
	}

	// 检查 DashScopeKey 是否为空
	if d.config.DashScopeKey == "" {
		err := errors.New("DashScopeKey is empty")
		log.Errorf("Failed to construct headers: %v", err)
		return "", nil, nil, err
	}

	// 设置请求头
	headers := [][2]string{
		{"Authorization", "Bearer " + d.config.DashScopeKey},
		{"Content-Type", contentType},
	}

	return endpoint, headers, requestBody, err
}

// Result 定义查询结果的结构
type Result struct {
	ID     string                 `json:"id"`
	Vector []float64              `json:"vector,omitempty"` // omitempty 使得如果 vector 是空，它将不会被序列化
	Fields map[string]interface{} `json:"fields"`
	Score  float64                `json:"score"`
}

func (d *DSProvider) parseTextEmbedding(responseBody []byte) (*Response, error) {
	var resp Response
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (d *DSProvider) GetEmbedding(
	queryString string,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(emb []float64, statusCode int, responseHeaders http.Header, responseBody []byte)) error {

	// 构建参数并处理错误
	Emb_url, Emb_headers, Emb_requestBody, err := d.constructParameters([]string{queryString}, log)
	if err != nil {
		log.Errorf("Failed to construct parameters: %v", err)
		return err
	}

	// 发起 POST 请求
	d.client.Post(Emb_url, Emb_headers, Emb_requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			defer proxywasm.ResumeHttpRequest() // 确保 HTTP 请求被恢复

			// 日志记录响应
			log.Infof("Get embedding response: %d, %s", statusCode, responseBody)

			// 解析响应
			resp, err := d.parseTextEmbedding(responseBody)
			if err != nil {
				log.Errorf("Failed to parse response: %v", err)
				callback(nil, statusCode, responseHeaders, responseBody)
				return
			}

			// 检查是否存在嵌入结果
			if len(resp.Output.Embeddings) == 0 {
				log.Errorf("No embedding found in response")
				callback(nil, statusCode, responseHeaders, responseBody)
				return
			}

			// 调用回调函数
			callback(resp.Output.Embeddings[0].Embedding, statusCode, responseHeaders, responseBody)
		}, d.config.DashScopeTimeout)

	return nil
}
