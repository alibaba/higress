package vector

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

type pineconeProviderInitializer struct{}

const pineconePort = 443

func (c *pineconeProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if len(config.PineconeApiEndpoint) == 0 {
		return errors.New("PineconeApiEndpoint is required")
	}
	if len(config.PineconeServiceName) == 0 {
		return errors.New("PineconeServiceName is required")
	}
	if len(config.PineconeApiKey) == 0 {
		return errors.New("PineconeApiKey is required")
	}
	return nil
}

func (c *pineconeProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &pineconeProvider{
		config: config,
		client: wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: config.PineconeServiceName,
			Port:        pineconePort,
			Domain:      config.PineconeApiEndpoint,
		}),
	}, nil
}

type pineconeProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (c *pineconeProvider) GetProviderType() string {
	return providerTypePinecone
}

func (d *pineconeProvider) GetThreshold() float64 {
	return d.config.PineconeThreshold
}

type pineconeMetadata struct {
	Question string `json:"question"`
}

type pineconeVector struct {
	ID         string           `json:"id"`
	Values     []float64        `json:"values"`
	Properties pineconeMetadata `json:"metadata"`
}

type pineconeInsertRequest struct {
	Vectors   []pineconeVector `json:"vectors"`
	Namespace string           `json:"namespace"`
}

func (d *pineconeProvider) UploadEmbedding(
	queryEmb []float64,
	queryString string,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(ctx wrapper.HttpContext, log wrapper.Log)) {
	// 最少需要填写的参数为 vector 和 question
	// 下面是一个例子
	// {
	// 	"vectors": [
	// 	  {
	// 		"id": "A",
	// 		"values": [0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1],
	// 		"metadata": {"question": "你好"}
	// 	  }
	// 	]
	// }
	requestBody, err := json.Marshal(pineconeInsertRequest{
		Vectors: []pineconeVector{
			{
				ID:         uuid.New().String(),
				Values:     queryEmb,
				Properties: pineconeMetadata{Question: queryString},
			},
		},
		Namespace: d.config.PineconeNamespace,
	})

	if err != nil {
		log.Errorf("[Pinecone] Failed to marshal upload embedding request body: %v", err)
		return
	}

	d.client.Post(
		"/vectors/upsert",
		[][2]string{
			{"Content-Type", "application/json"},
			{"Api-Key", d.config.PineconeApiKey},
		},
		requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("[Pinecone] statusCode:%d, responseBody:%s", statusCode, string(responseBody))
			callback(ctx, log)
		},
		d.config.PineconeTimeout,
	)
}

type pineconeQueryRequest struct {
	Namespace       string    `json:"namespace"`
	Vector          []float64 `json:"vector"`
	TopK            int       `json:"topK"`
	IncludeMetadata bool      `json:"includeMetadata"`
	IncludeValues   bool      `json:"includeValues"`
}

func (d *pineconeProvider) QueryEmbedding(
	emb []float64,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log)) {
	// 最少需要填写的参数为 vector
	// 下面是一个例子
	// {
	// 	"namespace": "higress",
	// 	"vector": [0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1],
	// 	"topK": 1,
	// 	"includeMetadata": false
	// }
	requestBody, err := json.Marshal(pineconeQueryRequest{
		Namespace:       d.config.PineconeNamespace,
		Vector:          emb,
		TopK:            d.config.PineconeTopK,
		IncludeMetadata: true,
		IncludeValues:   false,
	})
	if err != nil {
		log.Errorf("[Pinecone] Failed to marshal query embedding: %v", err)
		return
	}

	d.client.Post(
		"/query",
		[][2]string{
			{"Content-Type", "application/json"},
			{"Api-Key", d.config.PineconeApiKey},
		},
		requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("Query embedding response: %d, %s", statusCode, responseBody)
			callback(responseBody, ctx, log)
		},
		d.config.PineconeTimeout,
	)
}

func (d *pineconeProvider) ParseQueryResponse(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log) (QueryEmbeddingResult, error) {
	if !gjson.GetBytes(responseBody, "matches.0.score").Exists() {
		log.Errorf("[Pinecone] No distance found in response body: %s", responseBody)
		return QueryEmbeddingResult{}, nil
	}

	if !gjson.GetBytes(responseBody, "matches.0.metadata.question").Exists() {
		log.Errorf("[Pinecone] No question found in response body: %s", responseBody)
		return QueryEmbeddingResult{}, nil
	}

	return QueryEmbeddingResult{
		MostSimilarData: gjson.GetBytes(responseBody, "matches.0.metadata.question").String(),
		Score:           gjson.GetBytes(responseBody, "matches.0.score").Float(),
	}, nil
}
