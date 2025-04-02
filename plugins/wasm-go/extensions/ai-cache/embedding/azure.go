package embedding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	AZURE_PORT               = 443
	AZURE_DEFAULT_MODEL_NAME = "text-embedding-ada-002"
	AZURE_ENDPOINT           = "/openai/deployments/{model}/embeddings"
)

type azureProviderInitializer struct {
}

var azureConfig azureProviderConfig

type azureProviderConfig struct {
	// @Title zh-CN 文本特征提取服务 API Key
	// @Description zh-CN 文本特征提取服务 API Key
	apiKey string
	// @Title zh-CN 文本特征提取 api-version
	// @Description zh-CN 文本特征提取服务 api-version
	apiVersion string
}

func (c *azureProviderInitializer) InitConfig(json gjson.Result) {
	azureConfig.apiKey = json.Get("apiKey").String()
	azureConfig.apiVersion = json.Get("apiVersion").String()
}

func (c *azureProviderInitializer) ValidateConfig() error {
	if azureConfig.apiKey == "" {
		return errors.New("[Azure] apiKey is required")
	}
	if azureConfig.apiVersion == "" {
		return errors.New("[Azure] apiVersion is required")
	}
	return nil
}

func (t *azureProviderInitializer) CreateProvider(c ProviderConfig) (Provider, error) {
	if c.servicePort == 0 {
		c.servicePort = AZURE_PORT
	}

	if c.model == "" {
		c.model = AZURE_DEFAULT_MODEL_NAME
	}

	return &AzureProvider{
		config: c,
		client: wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: c.serviceName,
			Host: c.serviceHost,
			Port: c.servicePort,
		}),
	}, nil
}

func (t *AzureProvider) GetProviderType() string {
	return PROVIDER_TYPE_AZURE
}

type AzureProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

type AzureEmbeddingRequest struct {
	Input string `json:"input"`
}

func (t *AzureProvider) constructParameters(text string) (string, [][2]string, []byte, error) {
	if text == "" {
		err := errors.New("queryString text cannot be empty")
		return "", nil, nil, err
	}

	data := AzureEmbeddingRequest{
		Input: text,
	}

	requestBody, err := json.Marshal(data)
	if err != nil {
		log.Errorf("failed to marshal request data: %v", err)
		return "", nil, nil, err
	}

	model := t.config.model
	if model == "" {
		model = AZURE_DEFAULT_MODEL_NAME
	}

	// 拼接 endpoint
	endpoint := strings.Replace(AZURE_ENDPOINT, "{model}", model, 1)
	endpoint = endpoint + "?" + "api-version=" + azureConfig.apiVersion

	headers := [][2]string{
		{"api-key", azureConfig.apiKey},
		{"Content-Type", "application/json"},
	}

	return endpoint, headers, requestBody, err
}

type AzureEmbeddingResponse struct {
	Object string `json:"object"`
	Model  string `json:"model"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func (t *AzureProvider) parseTextEmbedding(responseBody []byte) (*AzureEmbeddingResponse, error) {
	var resp AzureEmbeddingResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

func (t *AzureProvider) GetEmbedding(
	queryString string,
	ctx wrapper.HttpContext,
	callback func(emb []float64, err error)) error {
	embUrl, embHeaders, embRequestBody, err := t.constructParameters(queryString)
	if err != nil {
		log.Errorf("failed to construct parameters: %v", err)
		return err
	}

	var resp *AzureEmbeddingResponse
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
