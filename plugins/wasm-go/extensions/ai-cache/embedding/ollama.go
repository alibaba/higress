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
	OLLAMA_DOMAIN             = "localhost"
	OLLAMA_PORT               = 11434
	OLLAMA_DEFAULT_MODEL_NAME = "llama3.2"
	OLLAMA_ENDPOINT           = "/api/embed"
)

type ollamaProviderInitializer struct {
}

func (c *ollamaProviderInitializer) InitConfig(json gjson.Result) {}

func (c *ollamaProviderInitializer) ValidateConfig() error {
	return nil
}

type ollamaProvider struct {
	config ProviderConfig
	client *wrapper.ClusterClient[wrapper.FQDNCluster]
}

func (t *ollamaProviderInitializer) CreateProvider(c ProviderConfig) (Provider, error) {
	if c.servicePort == 0 {
		c.servicePort = OLLAMA_PORT
	}
	if c.serviceHost == "" {
		c.serviceHost = OLLAMA_DOMAIN
	}
	if c.model == "" {
		c.model = OLLAMA_DEFAULT_MODEL_NAME
	}

	return &ollamaProvider{
		config: c,
		client: wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: c.serviceName,
			Host: c.serviceHost,
			Port: c.servicePort,
		}),
	}, nil
}

func (t *ollamaProvider) GetProviderType() string {
	return PROVIDER_TYPE_OLLAMA
}

type ollamaResponse struct {
	Model           string      `json:"model"`
	Embeddings      [][]float64 `json:"embeddings"`
	TotalDuration   int64       `json:"total_duration"`
	LoadDuration    int64       `json:"load_duration"`
	PromptEvalCount int64       `json:"prompt_eval_count"`
}

type ollamaEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

func (t *ollamaProvider) constructParameters(text string) (string, [][2]string, []byte, error) {
	if text == "" {
		err := errors.New("queryString text cannot be empty")
		return "", nil, nil, err
	}

	data := ollamaEmbeddingRequest{
		Input: text,
		Model: t.config.model,
	}

	requestBody, err := json.Marshal(data)
	if err != nil {
		log.Errorf("failed to marshal request data: %v", err)
		return "", nil, nil, err
	}

	headers := [][2]string{
		{"Content-Type", "application/json"},
	}
	log.Debugf("constructParameters: %s", string(requestBody))

	return OLLAMA_ENDPOINT, headers, requestBody, err
}

func (t *ollamaProvider) parseTextEmbedding(responseBody []byte) (*ollamaResponse, error) {
	var resp ollamaResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

func (t *ollamaProvider) GetEmbedding(
	queryString string,
	ctx wrapper.HttpContext,
	callback func(emb []float64, err error)) error {
	embUrl, embHeaders, embRequestBody, err := t.constructParameters(queryString)
	if err != nil {
		log.Errorf("failed to construct parameters: %v", err)
		return err
	}

	var resp *ollamaResponse

	defer func() {
		if err != nil {
			callback(nil, err)
		}
	}()
	err = t.client.Post(embUrl, embHeaders, embRequestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {

			if statusCode != http.StatusOK {
				err = errors.New("failed to get embedding due to status code: " + strconv.Itoa(statusCode))
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

			if len(resp.Embeddings) == 0 {
				err = errors.New("no embedding found in response")
				callback(nil, err)
				return
			}

			callback(resp.Embeddings[0], nil)

		}, t.config.timeout)
	return err
}
