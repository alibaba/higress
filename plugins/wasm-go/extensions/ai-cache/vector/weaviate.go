package vector

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type weaviateProviderInitializer struct{}

const weaviatePort = 8081

func (c *weaviateProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if len(config.WeaviateCollection) == 0 {
		return errors.New("WeaviateCollection is required")
	}
	if len(config.WeaviateServiceName) == 0 {
		return errors.New("WeaviateServiceName is required")
	}
	return nil
}

func (c *weaviateProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &WeaviateProvider{
		config: config,
		client: wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: config.WeaviateServiceName,
			Port:        weaviatePort,
			Domain:      config.WeaviateServiceName,
		}),
	}, nil
}

type WeaviateProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (c *WeaviateProvider) GetProviderType() string {
	return providerTypeWeaviate
}

func (d *WeaviateProvider) GetThreshold() float64 {
	return d.config.WeaviateThreshold
}

func (d *WeaviateProvider) QueryEmbedding(
	emb []float64,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log)) {
	// 最少需要填写的参数为 class, vector
	// 下面是一个例子
	// {"query": "{ Get { Higress ( limit: 2 nearVector: { vector: [0.1, 0.2, 0.3] } ) { question _additional { distance } } } }"}
	embString, err := json.Marshal(emb)
	if err != nil {
		log.Errorf("[Weaviate] Failed to marshal query embedding: %v", err)
		return
	}
	// 这里默认按照 distance 进行升序，所以不用再次排序
	graphql := fmt.Sprintf(`
	{
	  Get {
	    %s (
	      limit: %d
	      nearVector: {
	        vector: %s
	      }
	    ) {
		  question
	      _additional {
	        distance
	      }
	    }
	  }
	}
	`, d.config.WeaviateCollection, d.config.WeaviateNResult, embString)

	requestBody, err := json.Marshal(WeaviateQueryRequest{
		Query: graphql,
	})

	if err != nil {
		log.Errorf("[Weaviate] Failed to marshal query embedding request body: %v", err)
		return
	}

	d.client.Post(
		"/v1/graphql",
		[][2]string{
			{"Content-Type", "application/json"},
		},
		requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("Query embedding response: %d, %s", statusCode, responseBody)
			callback(responseBody, ctx, log)
		},
		d.config.WeaviateTimeout,
	)
}

func (d *WeaviateProvider) UploadEmbedding(
	queryEmb []float64,
	queryString string,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(ctx wrapper.HttpContext, log wrapper.Log)) {
	// 最少需要填写的参数为 class, vector 和 question
	// 下面是一个例子
	// {"class": "Higress", "vector": [0.1, 0.2, 0.3], "properties": {"question": "这里是问题"}}
	requestBody, err := json.Marshal(WeaviateInsertRequest{
		Class:      d.config.WeaviateCollection,
		Vector:     queryEmb,
		Properties: WeaviateProperties{Question: queryString}, // queryString 指的是用户查询的问题
	})

	if err != nil {
		log.Errorf("[Weaviate] Failed to marshal upload embedding request body: %v", err)
		return
	}

	d.client.Post(
		"/v1/objects",
		[][2]string{
			{"Content-Type", "application/json"},
		},
		requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("[Weaviate] statusCode:%d, responseBody:%s", statusCode, string(responseBody))
			callback(ctx, log)
		},
		d.config.WeaviateTimeout,
	)
}

type WeaviateProperties struct {
	Question string `json:"question"`
}

type WeaviateInsertRequest struct {
	Class      string             `json:"class"`
	Vector     []float64          `json:"vector"`
	Properties WeaviateProperties `json:"properties"`
}

type WeaviateQueryRequest struct {
	Query string `json:"query"`
}

func (d *WeaviateProvider) ParseQueryResponse(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log) (QueryEmbeddingResult, error) {
	if !gjson.GetBytes(responseBody, fmt.Sprintf("data.Get.%s.0._additional.distance", d.config.WeaviateCollection)).Exists() {
		log.Errorf("[Weaviate] No distance found in response body: %s", responseBody)
		return QueryEmbeddingResult{}, nil
	}

	if !gjson.GetBytes(responseBody, fmt.Sprintf("data.Get.%s.0.question", d.config.WeaviateCollection)).Exists() {
		log.Errorf("[Weaviate] No question found in response body: %s", responseBody)
		return QueryEmbeddingResult{}, nil
	}

	return QueryEmbeddingResult{
		MostSimilarData: gjson.GetBytes(responseBody, fmt.Sprintf("data.Get.%s.0.question", d.config.WeaviateCollection)).String(),
		Score:           gjson.GetBytes(responseBody, fmt.Sprintf("data.Get.%s.0._additional.distance", d.config.WeaviateCollection)).Float(),
	}, nil
}
