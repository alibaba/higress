package embedding

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	domain    = "dashscope.aliyuncs.com"
	port      = 443
	modelName = "text-embedding-v1"
	endpoint  = "/api/v1/services/embeddings/text-embedding/text-embedding"
)

type dashScopeProviderInitializer struct {
}

func (d *dashScopeProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiKey == "" {
		return errors.New("DashScopeKey is required")
	}
	return nil
}

func (d *dashScopeProviderInitializer) CreateProvider(c ProviderConfig) (Provider, error) {
	if c.servicePort == 0 {
		c.servicePort = port
	}
	if c.serviceHost == "" {
		c.serviceHost = domain
	}
	return &DSProvider{
		config: c,
		client: wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: c.serviceName,
			Port:        c.servicePort,
			Domain:      c.serviceHost,
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

type EmbeddingRequest struct {
	Model      string `json:"model"`
	Input      Input  `json:"input"`
	Parameters Params `json:"parameters"`
}

type Document struct {
	Vector []float64         `json:"vector"`
	Fields map[string]string `json:"fields"`
}

type DSProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (d *DSProvider) constructParameters(texts []string, log wrapper.Log) (string, [][2]string, []byte, error) {

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
	if d.config.apiKey == "" {
		err := errors.New("DashScopeKey is empty")
		log.Errorf("Failed to construct headers: %v", err)
		return "", nil, nil, err
	}

	// 设置请求头
	headers := [][2]string{
		{"Authorization", "Bearer " + d.config.apiKey},
		{"Content-Type", "application/json"},
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

// 返回指针防止拷贝 Embedding
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
	Emb_url, Emb_headers, Emb_requestBody, err := d.constructParameters([]string{queryString}, log)
	if err != nil {
		log.Errorf("Failed to construct parameters: %v", err)
		return err
	}

	var resp *Response
	d.client.Post(Emb_url, Emb_headers, Emb_requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				log.Errorf("Failed to fetch embeddings, statusCode: %d, responseBody: %s", statusCode, string(responseBody))
				err = errors.New("failed to get embedding")
				callback(nil, statusCode, responseHeaders, responseBody)
				return
			}

			log.Infof("Get embedding response: %d, %s", statusCode, responseBody)

			resp, err = d.parseTextEmbedding(responseBody)
			if err != nil {
				log.Errorf("Failed to parse response: %v", err)
				callback(nil, statusCode, responseHeaders, responseBody)
				return
			}

			if len(resp.Output.Embeddings) == 0 {
				log.Errorf("No embedding found in response")
				err = errors.New("no embedding found in response")
				callback(nil, statusCode, responseHeaders, responseBody)
				return
			}

			callback(resp.Output.Embeddings[0].Embedding, statusCode, responseHeaders, responseBody)

		}, d.config.timeout)
	return nil
}
