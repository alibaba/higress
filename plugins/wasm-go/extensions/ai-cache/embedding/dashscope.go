package embedding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
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
}

func (c *dashScopeProviderInitializer) InitConfig(json gjson.Result) {
	dashScopeConfig.apiKey = json.Get("apiKey").String()
}

func (c *dashScopeProviderInitializer) ValidateConfig() error {
	if dashScopeConfig.apiKey == "" {
		return errors.New("[DashScope] apiKey is required")
	}
	return nil
}

func (d *dashScopeProviderInitializer) CreateProvider(c ProviderConfig) (Provider, error) {
	if c.servicePort == 0 {
		c.servicePort = DASHSCOPE_PORT
	}
	if c.serviceHost == "" {
		c.serviceHost = DASHSCOPE_DOMAIN
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

func (d *DSProvider) GetProviderType() string {
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

type DSProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (d *DSProvider) constructParameters(texts []string) (string, [][2]string, []byte, error) {

	model := d.config.model

	if model == "" {
		model = DASHSCOPE_DEFAULT_MODEL_NAME
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

	requestBody, err := json.Marshal(data)
	if err != nil {
		log.Errorf("failed to marshal request data: %v", err)
		return "", nil, nil, err
	}

	if dashScopeConfig.apiKey == "" {
		err := errors.New("dashScopeKey is empty")
		log.Errorf("failed to construct headers: %v", err)
		return "", nil, nil, err
	}

	headers := [][2]string{
		{"Authorization", "Bearer " + dashScopeConfig.apiKey},
		{"Content-Type", "application/json"},
	}

	return DASHSCOPE_ENDPOINT, headers, requestBody, err
}

type Result struct {
	ID     string                 `json:"id"`
	Vector []float64              `json:"vector,omitempty"`
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
	callback func(emb []float64, err error)) error {
	embUrl, embHeaders, embRequestBody, err := d.constructParameters([]string{queryString})
	if err != nil {
		log.Errorf("failed to construct parameters: %v", err)
		return err
	}

	var resp *Response
	err = d.client.Post(embUrl, embHeaders, embRequestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {

			if statusCode != http.StatusOK {
				err = errors.New("failed to get embedding due to status code: " + strconv.Itoa(statusCode))
				callback(nil, err)
				return
			}

			log.Debugf("get embedding response: %d, %s", statusCode, responseBody)

			resp, err = d.parseTextEmbedding(responseBody)
			if err != nil {
				err = fmt.Errorf("failed to parse response: %v", err)
				callback(nil, err)
				return
			}

			if len(resp.Output.Embeddings) == 0 {
				err = errors.New("no embedding found in response")
				callback(nil, err)
				return
			}

			callback(resp.Output.Embeddings[0].Embedding, nil)

		}, d.config.timeout)
	return err
}
