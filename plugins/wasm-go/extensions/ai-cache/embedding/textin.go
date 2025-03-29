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
	TEXTIN_DOMAIN             = "api.textin.com"
	TEXTIN_PORT               = 443
	TEXTIN_DEFAULT_MODEL_NAME = "acge-text-embedding"
	TEXTIN_ENDPOINT           = "/ai/service/v1/acge_embedding"
)

type textInProviderInitializer struct {
}

var textInConfig textInProviderConfig

type textInProviderConfig struct {
	//@Title zh-CN TextIn x-ti-app-id
	// @Description zh-CN 仅适用于 TextIn 服务。参考 https://www.textin.com/document/acge_text_embedding
	textinAppId string
	//@Title zh-CN TextIn x-ti-secret-code
	// @Description zh-CN 仅适用于 TextIn 服务。参考 https://www.textin.com/document/acge_text_embedding
	textinSecretCode string
	//@Title zh-CN TextIn request matryoshka_dim
	// @Description zh-CN 仅适用于 TextIn 服务, 指定返回的向量维度。参考 https://www.textin.com/document/acge_text_embedding
	textinMatryoshkaDim int
}

func (c *textInProviderInitializer) InitConfig(json gjson.Result) {
	textInConfig.textinAppId = json.Get("textinAppId").String()
	textInConfig.textinSecretCode = json.Get("textinSecretCode").String()
	textInConfig.textinMatryoshkaDim = int(json.Get("textinMatryoshkaDim").Int())
}

func (c *textInProviderInitializer) ValidateConfig() error {
	if textInConfig.textinAppId == "" {
		return errors.New("textinAppId is required")
	}
	if textInConfig.textinSecretCode == "" {
		return errors.New("textinSecretCode is required")
	}
	if textInConfig.textinMatryoshkaDim == 0 {
		return errors.New("embedding service TextIn Matryoshka Dim is required")
	}
	return nil
}

func (t *textInProviderInitializer) CreateProvider(c ProviderConfig) (Provider, error) {
	if c.servicePort == 0 {
		c.servicePort = TEXTIN_PORT
	}
	if c.serviceHost == "" {
		c.serviceHost = TEXTIN_DOMAIN
	}
	return &TIProvider{
		config: c,
		client: wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: c.serviceName,
			Host: c.serviceHost,
			Port: int64(c.servicePort),
		}),
	}, nil
}

func (t *TIProvider) GetProviderType() string {
	return PROVIDER_TYPE_TEXTIN
}

type TextInResponse struct {
	Code     int          `json:"code"`
	Message  string       `json:"message"`
	Duration float64      `json:"duration"`
	Result   TextInResult `json:"result"`
}

type TextInResult struct {
	Embeddings    [][]float64 `json:"embedding"`
	MatryoshkaDim int         `json:"matryoshka_dim"`
}

type TextInEmbeddingRequest struct {
	Input         []string `json:"input"`
	MatryoshkaDim int      `json:"matryoshka_dim"`
}

type TIProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (t *TIProvider) constructParameters(texts []string) (string, [][2]string, []byte, error) {

	data := TextInEmbeddingRequest{
		Input:         texts,
		MatryoshkaDim: textInConfig.textinMatryoshkaDim,
	}

	requestBody, err := json.Marshal(data)
	if err != nil {
		log.Errorf("failed to marshal request data: %v", err)
		return "", nil, nil, err
	}

	if textInConfig.textinAppId == "" {
		err := errors.New("textinAppId is empty")
		log.Errorf("failed to construct headers: %v", err)
		return "", nil, nil, err
	}
	if textInConfig.textinSecretCode == "" {
		err := errors.New("textinSecretCode is empty")
		log.Errorf("failed to construct headers: %v", err)
		return "", nil, nil, err
	}

	headers := [][2]string{
		{"x-ti-app-id", textInConfig.textinAppId},
		{"x-ti-secret-code", textInConfig.textinSecretCode},
		{"Content-Type", "application/json"},
	}

	return TEXTIN_ENDPOINT, headers, requestBody, err
}

func (t *TIProvider) parseTextEmbedding(responseBody []byte) (*TextInResponse, error) {
	var resp TextInResponse
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (t *TIProvider) GetEmbedding(
	queryString string,
	ctx wrapper.HttpContext,
	callback func(emb []float64, err error)) error {
	embUrl, embHeaders, embRequestBody, err := t.constructParameters([]string{queryString})
	if err != nil {
		log.Errorf("failed to construct parameters: %v", err)
		return err
	}

	var resp *TextInResponse
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

			if len(resp.Result.Embeddings) == 0 {
				err = errors.New("no embedding found in response")
				callback(nil, err)
				return
			}

			callback(resp.Result.Embeddings[0], nil)

		}, t.config.timeout)
	return err
}
