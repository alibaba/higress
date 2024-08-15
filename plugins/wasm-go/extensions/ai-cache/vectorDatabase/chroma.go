package vectorDatabase

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

type chromaProviderInitializer struct{}

const chromaPort = 8001

func (c *chromaProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if len(config.ChromaCollectionID) == 0 {
		return errors.New("ChromaCollectionID is required")
	}
	if len(config.ChromaServiceName) == 0 {
		return errors.New("ChromaServiceName is required")
	}
	return nil
}

func (c *chromaProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &ChromaProvider{
		config: config,
		client: wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: config.ChromaServiceName,
			Port:        chromaPort,
			Domain:      config.ChromaServiceName,
		}),
	}, nil
}

type ChromaProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (c *ChromaProvider) GetProviderType() string {
	return providerTypeChroma
}

func (d *ChromaProvider) GetThreshold() float64 {
	return d.config.ChromaDistanceThreshold
}

func (d *ChromaProvider) QueryEmbedding(
	emb []float64,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log)) {
	// 最小需要填写的参数为 collection_id, embeddings 和 ids
	// 下面是一个例子
	// {
	// 	"where": {}, 用于 metadata 过滤，可选参数
	// 	"where_document": {}, 用于 document 过滤，可选参数
	// 	"query_embeddings": [
	// 	  [1.1, 2.3, 3.2]
	// 	],
	// 	"n_results": 5,
	// 	"include": [
	// 	  "metadatas",
	// 	  "distances"
	// 	]
	// }
	requestBody, err := json.Marshal(ChromaQueryRequest{
		QueryEmbeddings: []ChromaEmbedding{emb},
		NResults:        d.config.ChromaNResult,
		Include:         []string{"distances"},
	})

	if err != nil {
		log.Errorf("[Chroma] Failed to marshal query embedding request body: %v", err)
		return
	}

	d.client.Post(
		fmt.Sprintf("/api/v1/collections/%s/query", d.config.ChromaCollectionID),
		[][2]string{
			{"Content-Type", "application/json"},
		},
		requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("Query embedding response: %d, %s", statusCode, responseBody)
			callback(responseBody, ctx, log)
		},
		d.config.ChromaTimeout,
	)
}

func (d *ChromaProvider) UploadEmbedding(
	query_emb []float64,
	queryString string,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(ctx wrapper.HttpContext, log wrapper.Log)) {
	// 最小需要填写的参数为 collection_id, embeddings 和 ids
	// 下面是一个例子
	// {
	// 	"embeddings": [
	// 		  [1.1, 2.3, 3.2]
	// 	],
	// 	"ids": [
	// 	  "你吃了吗？"
	// 	]
	// }
	requestBody, err := json.Marshal(ChromaInsertRequest{
		Embeddings: []ChromaEmbedding{query_emb},
		IDs:        []string{queryString}, // queryString 指的是用户查询的问题
	})

	if err != nil {
		log.Errorf("[Chroma] Failed to marshal upload embedding request body: %v", err)
		return
	}

	d.client.Post(
		fmt.Sprintf("/api/v1/collections/%s/add", d.config.ChromaCollectionID),
		[][2]string{
			{"Content-Type", "application/json"},
		},
		requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("[Chroma] statusCode:%d, responseBody:%s", statusCode, string(responseBody))
			callback(ctx, log)
		},
		d.config.ChromaTimeout,
	)
}

// ChromaEmbedding represents the embedding vector for a data point.
type ChromaEmbedding []float64

// ChromaMetadataMap is a map from key to value for metadata.
type ChromaMetadataMap map[string]string

// Dataset represents the entire dataset containing multiple data points.
type ChromaInsertRequest struct {
	Embeddings []ChromaEmbedding   `json:"embeddings"`
	Metadatas  []ChromaMetadataMap `json:"metadatas,omitempty"` // Optional metadata map array
	Documents  []string            `json:"documents,omitempty"` // Optional document array
	IDs        []string            `json:"ids"`
}

// ChromaQueryRequest represents the query request structure.
type ChromaQueryRequest struct {
	Where           map[string]string `json:"where,omitempty"`          // Optional where filter
	WhereDocument   map[string]string `json:"where_document,omitempty"` // Optional where_document filter
	QueryEmbeddings []ChromaEmbedding `json:"query_embeddings"`
	NResults        int               `json:"n_results"`
	Include         []string          `json:"include"`
}

// ChromaQueryResponse represents the search result structure.
type ChromaQueryResponse struct {
	Ids        [][]string          `json:"ids"`                  // 每一个 embedding 相似的 key 可能会有多个，然后会有多个 embedding，所以是一个二维数组
	Distances  [][]float64         `json:"distances"`            // 与 Ids 一一对应
	Metadatas  []ChromaMetadataMap `json:"metadatas,omitempty"`  // Optional, can be null
	Embeddings []ChromaEmbedding   `json:"embeddings,omitempty"` // Optional, can be null
	Documents  []string            `json:"documents,omitempty"`  // Optional, can be null
	Uris       []string            `json:"uris,omitempty"`       // Optional, can be null
	Data       []interface{}       `json:"data,omitempty"`       // Optional, can be null
	Included   []string            `json:"included"`
}

func (d *ChromaProvider) ParseQueryResponse(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log) (QueryEmbeddingResult, error) {
	var queryResp ChromaQueryResponse
	err := json.Unmarshal(responseBody, &queryResp)
	if err != nil {
		return QueryEmbeddingResult{}, err
	}
	log.Infof("[Chroma] queryResp: %+v", queryResp)
	log.Infof("[Chroma] queryResp Ids len: %d", len(queryResp.Ids))
	if len(queryResp.Ids) == 1 && len(queryResp.Ids[0]) == 0 {
		return QueryEmbeddingResult{}, nil
	}
	return QueryEmbeddingResult{
		MostSimilarData: queryResp.Ids[0][0],
		Score:           queryResp.Distances[0][0],
	}, nil
}
