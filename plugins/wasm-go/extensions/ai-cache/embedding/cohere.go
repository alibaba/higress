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
	COHERE_DOMAIN             = "api.cohere.com"
	COHERE_PORT               = 443
	COHERE_DEFAULT_MODEL_NAME = "embed-english-v2.0"
	COHERE_ENDPOINT           = "/v2/embed"
)

type cohereProviderInitializer struct {
}

var cohereConfig cohereProviderConfig

type cohereProviderConfig struct {
	// @Title zh-CN 文本特征提取服务 API Key
	// @Description zh-CN 文本特征提取服务 API Key
	apiKey string
}

func (c *cohereProviderInitializer) InitConfig(json gjson.Result) {
	cohereConfig.apiKey = json.Get("apiKey").String()
}
func (c *cohereProviderInitializer) ValidateConfig() error {
	if cohereConfig.apiKey == "" {
		return errors.New("[Cohere] apiKey is required")
	}
	return nil
}

func (t *cohereProviderInitializer) CreateProvider(c ProviderConfig) (Provider, error) {
	if c.servicePort == 0 {
		c.servicePort = COHERE_PORT
	}
	if c.serviceHost == "" {
		c.serviceHost = COHERE_DOMAIN
	}
	return &CohereProvider{
		config: c,
		client: wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: c.serviceName,
			Host: c.serviceHost,
			Port: int64(c.servicePort),
		}),
	}, nil
}

type cohereResponse struct {
	Embeddings cohereEmbeddings `json:"embeddings"`
}

type cohereEmbeddings struct {
	FloatTypeEebedding [][]float64 `json:"float"`
}

type cohereEmbeddingRequest struct {
	Texts          []string `json:"texts"`
	Model          string   `json:"model"`
	InputType      string   `json:"input_type"`
	EmbeddingTypes []string `json:"embedding_types"`
}

type CohereProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (t *CohereProvider) GetProviderType() string {
	return PROVIDER_TYPE_COHERE
}
func (t *CohereProvider) constructParameters(texts []string) (string, [][2]string, []byte, error) {
	model := t.config.model

	if model == "" {
		model = COHERE_DEFAULT_MODEL_NAME
	}
	data := cohereEmbeddingRequest{
		Texts:          texts,
		Model:          model,
		InputType:      "search_document",
		EmbeddingTypes: []string{"float"},
	}

	requestBody, err := json.Marshal(data)
	if err != nil {
		log.Errorf("failed to marshal request data: %v", err)
		return "", nil, nil, err
	}

	headers := [][2]string{
		{"Authorization", fmt.Sprintf("BEARER %s", cohereConfig.apiKey)},
		{"Content-Type", "application/json"},
	}

	return COHERE_ENDPOINT, headers, requestBody, nil
}

func (t *CohereProvider) parseTextEmbedding(responseBody []byte) (*cohereResponse, error) {
	var resp cohereResponse
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (t *CohereProvider) GetEmbedding(
	queryString string,
	ctx wrapper.HttpContext,
	callback func(emb []float64, err error)) error {
	embUrl, embHeaders, embRequestBody, err := t.constructParameters([]string{queryString})
	if err != nil {
		log.Errorf("failed to construct parameters: %v", err)
		return err
	}

	var resp *cohereResponse
	err = t.client.Post(embUrl, embHeaders, embRequestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {

			if statusCode != http.StatusOK {
				err = errors.New("failed to get embedding due to status code: " + strconv.Itoa(statusCode))
				callback(nil, err)
				return
			}

			log.Debugf("get embedding response: %d, %s", statusCode, responseBody)

			resp, err = t.parseTextEmbedding(responseBody)
			if err != nil {
				err = fmt.Errorf("failed to parse response: %v", err)
				callback(nil, err)
				return
			}

			if len(resp.Embeddings.FloatTypeEebedding) == 0 {
				err = errors.New("no embedding found in response")
				callback(nil, err)
				return
			}

			callback(resp.Embeddings.FloatTypeEebedding[0], nil)

		}, t.config.timeout)
	return err
}
