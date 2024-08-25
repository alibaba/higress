package vector

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

type esProviderInitializer struct{}

const esPort = 9200

func (c *esProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if len(config.ESIndex) == 0 {
		return errors.New("ESIndex is required")
	}
	if len(config.ESServiceName) == 0 {
		return errors.New("ESServiceName is required")
	}
	return nil
}

func (c *esProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &ESProvider{
		config: config,
		client: wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: config.ESServiceName,
			Port:        esPort,
			Domain:      config.ESServiceName,
		}),
	}, nil
}

type ESProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (c *ESProvider) GetProviderType() string {
	return providerTypeES
}

func (d *ESProvider) GetThreshold() float64 {
	return d.config.ESThreshold
}

func (d *ESProvider) QueryEmbedding(
	emb []float64,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log)) {
	// 最少需要填写的参数为 index, embeddings 和 ids
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
	requestBody, err := json.Marshal(esQueryRequest{
		Source: Source{Excludes: []string{"embedding"}},
		Knn: knn{
			Field:       "embedding",
			QueryVector: emb,
			K:           d.config.ESNResult,
		},
		Size: d.config.ESNResult,
	})

	if err != nil {
		log.Errorf("[es] Failed to marshal query embedding request body: %v", err)
		return
	}

	d.client.Post(
		fmt.Sprintf("/%s/_search", d.config.ESIndex),
		[][2]string{
			{"Content-Type", "application/json"},
			{"Authorization", d.getCredentials()},
		},
		requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("Query embedding response: %d, %s", statusCode, responseBody)
			callback(responseBody, ctx, log)
		},
		d.config.ESTimeout,
	)
}

// 编码 ES 身份认证字符串
func (d *ESProvider) getCredentials() string {
	credentials := fmt.Sprintf("%s:%s", d.config.ESUsername, d.config.ESPassword)
	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(credentials))
	return fmt.Sprintf("Basic %s", encodedCredentials)
}

func (d *ESProvider) UploadEmbedding(
	query_emb []float64,
	queryString string,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(ctx wrapper.HttpContext, log wrapper.Log)) {
	// 最少需要填写的参数为 index, embeddings 和 question
	// 下面是一个例子
	// POST /<index>/_doc
	// {
	// 	"embedding": [
	// 		  [1.1, 2.3, 3.2]
	// 	],
	// 	"question": [
	// 	  "你吃了吗？"
	// 	]
	// }
	requestBody, err := json.Marshal(esInsertRequest{
		Embedding: query_emb,
		Question:  queryString,
	})
	if err != nil {
		log.Errorf("[ES] Failed to marshal upload embedding request body: %v", err)
		return
	}

	d.client.Post(
		fmt.Sprintf("/%s/_doc", d.config.ESIndex),
		[][2]string{
			{"Content-Type", "application/json"},
			{"Authorization", d.getCredentials()},
		},
		requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("[ES] statusCode:%d, responseBody:%s", statusCode, string(responseBody))
			callback(ctx, log)
		},
		d.config.ESTimeout,
	)
}

type esInsertRequest struct {
	Embedding []float64 `json:"embedding"`
	Question  string    `json:"question"`
}

type knn struct {
	Field       string    `json:"field"`
	QueryVector []float64 `json:"query_vector"`
	K           int       `json:"k"`
}

type Source struct {
	Excludes []string `json:"excludes"`
}

type esQueryRequest struct {
	Source Source `json:"_source"`
	Knn    knn    `json:"knn"`
	Size   int    `json:"size"`
}

// esQueryResponse represents the search result structure.
type esQueryResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		Hits []struct {
			Index  string                 `json:"_index"`
			ID     string                 `json:"_id"`
			Score  float64                `json:"_score"`
			Source map[string]interface{} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (d *ESProvider) ParseQueryResponse(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log) (QueryEmbeddingResult, error) {
	var queryResp esQueryResponse
	err := json.Unmarshal(responseBody, &queryResp)
	if err != nil {
		return QueryEmbeddingResult{}, err
	}
	log.Infof("[ES] queryResp: %+v", queryResp)
	log.Infof("[ES] queryResp Hits len: %d", len(queryResp.Hits.Hits))
	if len(queryResp.Hits.Hits) == 0 {
		return QueryEmbeddingResult{}, nil
	}
	return QueryEmbeddingResult{
		MostSimilarData: queryResp.Hits.Hits[0].Source["question"].(string),
		Score:           queryResp.Hits.Hits[0].Score,
	}, nil
}
