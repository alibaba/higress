package vector

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	dashVectorPort = 443
	threshold      = 2000
)

type dashVectorProviderInitializer struct {
}

func (d *dashVectorProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if len(config.DashVectorKey) == 0 {
		return errors.New("DashVectorKey is required")
	}
	if len(config.DashVectorAuthApiEnd) == 0 {
		return errors.New("DashVectorEnd is required")
	}
	if len(config.DashVectorCollection) == 0 {
		return errors.New("DashVectorCollection is required")
	}
	if len(config.DashVectorServiceName) == 0 {
		return errors.New("DashVectorServiceName is required")
	}
	return nil
}

func (d *dashVectorProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &DvProvider{
		config: config,
		client: wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: config.DashVectorServiceName,
			Port:        dashVectorPort,
			Domain:      config.DashVectorAuthApiEnd,
		}),
	}, nil
}

type DvProvider struct {
	config ProviderConfig
	client wrapper.HttpClient
}

func (d *DvProvider) GetProviderType() string {
	return providerTypeDashVector
}

// type embeddingRequest struct {
// 	Model      string `json:"model"`
// 	Input      input  `json:"input"`
// 	Parameters params `json:"parameters"`
// }

// type params struct {
// 	TextType string `json:"text_type"`
// }

// type input struct {
// 	Texts []string `json:"texts"`
// }

// queryResponse 定义查询响应的结构
type queryResponse struct {
	Code      int      `json:"code"`
	RequestID string   `json:"request_id"`
	Message   string   `json:"message"`
	Output    []result `json:"output"`
}

// queryRequest 定义查询请求的结构
type queryRequest struct {
	Vector        []float64 `json:"vector"`
	TopK          int       `json:"topk"`
	IncludeVector bool      `json:"include_vector"`
}

// result 定义查询结果的结构
type result struct {
	ID     string                 `json:"id"`
	Vector []float64              `json:"vector,omitempty"` // omitempty 使得如果 vector 是空，它将不会被序列化
	Fields map[string]interface{} `json:"fields"`
	Score  float64                `json:"score"`
}

func (d *DvProvider) constructEmbeddingQueryParameters(vector []float64) (string, []byte, [][2]string, error) {
	url := fmt.Sprintf("/v1/collections/%s/query", d.config.DashVectorCollection)

	requestData := queryRequest{
		Vector:        vector,
		TopK:          d.config.DashVectorTopK,
		IncludeVector: false,
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return "", nil, nil, err
	}

	header := [][2]string{
		{"Content-Type", "application/json"},
		{"dashvector-auth-token", d.config.DashVectorKey},
	}

	return url, requestBody, header, nil
}

func (d *DvProvider) parseQueryResponse(responseBody []byte) (queryResponse, error) {
	var queryResp queryResponse
	err := json.Unmarshal(responseBody, &queryResp)
	if err != nil {
		return queryResponse{}, err
	}
	return queryResp, nil
}

func (d *DvProvider) GetThreshold() float64 {
	return threshold
}
func (d *DvProvider) QueryEmbedding(
	emb []float64,
	ctx wrapper.HttpContext,
	log wrapper.Log,
	callback func(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log)) {
	// 构造请求参数
	url, body, headers, err := d.constructEmbeddingQueryParameters(emb)
	if err != nil {
		log.Infof("Failed to construct embedding query parameters: %v", err)
	}

	d.client.Post(url, headers, body,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				log.Infof("Failed to query embedding: %d", statusCode)
				return
			}
			log.Infof("Query embedding response: %d, %s", statusCode, responseBody)
			callback(responseBody, ctx, log)
		},
		d.config.DashVectorTimeout)
	if err != nil {
		log.Infof("Failed to query embedding: %v", err)
	}
}

func (d *DvProvider) ParseQueryResponse(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log) (QueryEmbeddingResult, error) {
	resp, err := d.parseQueryResponse(responseBody)
	if err != nil {
		return QueryEmbeddingResult{}, err
	}
	if len(resp.Output) == 0 {
		return QueryEmbeddingResult{}, nil
	}
	return QueryEmbeddingResult{
		MostSimilarData: resp.Output[0].Fields["query"].(string),
		Score:           resp.Output[0].Score,
	}, nil
}

type document struct {
	Vector []float64         `json:"vector"`
	Fields map[string]string `json:"fields"`
}

type insertRequest struct {
	Docs []document `json:"docs"`
}

func (d *DvProvider) constructEmbeddingUploadParameters(emb []float64, query_string string) (string, []byte, [][2]string, error) {
	url := "/v1/collections/" + d.config.DashVectorCollection + "/docs"

	doc := document{
		Vector: emb,
		Fields: map[string]string{
			"query": query_string,
		},
	}

	requestBody, err := json.Marshal(insertRequest{Docs: []document{doc}})
	if err != nil {
		return "", nil, nil, err
	}

	header := [][2]string{
		{"Content-Type", "application/json"},
		{"dashvector-auth-token", d.config.DashVectorKey},
	}

	return url, requestBody, header, err
}

func (d *DvProvider) UploadEmbedding(query_emb []float64, queryString string, ctx wrapper.HttpContext, log wrapper.Log, callback func(ctx wrapper.HttpContext, log wrapper.Log)) {
	url, body, headers, _ := d.constructEmbeddingUploadParameters(query_emb, queryString)
	d.client.Post(
		url,
		headers,
		body,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("statusCode:%d, responseBody:%s", statusCode, string(responseBody))
			callback(ctx, log)
		},
		d.config.DashVectorTimeout)
}
