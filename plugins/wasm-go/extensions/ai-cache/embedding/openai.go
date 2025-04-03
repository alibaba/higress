package embedding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
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
	apiKey string
}

func (c *openAIProviderInitializer) InitConfig(json gjson.Result) {
	openAIConfig.apiKey = json.Get("apiKey").String()
}

func (c *openAIProviderInitializer) ValidateConfig() error {
	if openAIConfig.apiKey == "" {
		return errors.New("[openAI] apiKey is required")
	}
	return nil
}

func (t *openAIProviderInitializer) CreateProvider(c ProviderConfig) (Provider, error) {
	if c.servicePort == 0 {
		c.servicePort = OPENAI_PORT
	}
	if c.serviceHost == "" {
		c.serviceHost = OPENAI_DOMAIN
	}
	if c.model == "" {
		c.model = OPENAI_DEFAULT_MODEL_NAME
	}
	return &OpenAIProvider{
		config: c,
		client: wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: c.serviceName,
			Host: c.serviceHost,
			Port: c.servicePort,
		}),
	}, nil
}

func (t *OpenAIProvider) GetProviderType() string {
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
	Embedding []float64 `json:"embedding"`
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
	config ProviderConfig
	client wrapper.HttpClient
}

func (t *OpenAIProvider) constructParameters(text string) (string, [][2]string, []byte, error) {
	if text == "" {
		err := errors.New("queryString text cannot be empty")
		return "", nil, nil, err
	}

	data := OpenAIEmbeddingRequest{
		Input: text,
		Model: t.config.model,
	}

	requestBody, err := json.Marshal(data)
	if err != nil {
		log.Errorf("failed to marshal request data: %v", err)
		return "", nil, nil, err
	}

	headers := [][2]string{
		{"Authorization", fmt.Sprintf("Bearer %s", openAIConfig.apiKey)},
		{"Content-Type", "application/json"},
	}

	return OPENAI_ENDPOINT, headers, requestBody, err
}

func (t *OpenAIProvider) parseTextEmbedding(responseBody []byte) (*OpenAIResponse, error) {
	var resp OpenAIResponse
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (t *OpenAIProvider) GetEmbedding(
	queryString string,
	ctx wrapper.HttpContext,
	callback func(emb []float64, err error)) error {
	embUrl, embHeaders, embRequestBody, err := t.constructParameters(queryString)
	if err != nil {
		log.Errorf("failed to construct parameters: %v", err)
		return err
	}

	var resp *OpenAIResponse
	err = t.client.Post(embUrl, embHeaders, embRequestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {

			if statusCode != http.StatusOK {
				err = fmt.Errorf("failed to get embedding due to status code: %d, resp: %s", statusCode, responseBody)
				callback(nil, err)
				return
			}

			resp, err = t.parseTextEmbedding(responseBody)
			if err != nil {
				err = fmt.Errorf("failed to parse response: %v", err)
				callback(nil, err)
				return
			}

			log.Debugf("get embedding response: %d, %s", statusCode, responseBody)

			if len(resp.Data) == 0 {
				err = errors.New("no embedding found in response")
				callback(nil, err)
				return
			}

			callback(resp.Data[0].Embedding, nil)

		}, t.config.timeout)
	return err
}
