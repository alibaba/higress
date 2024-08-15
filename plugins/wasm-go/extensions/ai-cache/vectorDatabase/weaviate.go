// package vectorDatabase

// import (
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"net/http"

// 	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
// )

// const (
// 	dashVectorPort = 443
// )

// type dashVectorProviderInitializer struct {
// }

// func (d *dashVectorProviderInitializer) ValidateConfig(config ProviderConfig) error {
// 	if len(config.DashVectorKey) == 0 {
// 		return errors.New("DashVectorKey is required")
// 	}
// 	if len(config.DashVectorAuthApiEnd) == 0 {
// 		return errors.New("DashVectorEnd is required")
// 	}
// 	if len(config.DashVectorCollection) == 0 {
// 		return errors.New("DashVectorCollection is required")
// 	}
// 	if len(config.DashVectorServiceName) == 0 {
// 		return errors.New("DashVectorServiceName is required")
// 	}
// 	return nil
// }

// func (d *dashVectorProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
// 	return &DvProvider{
// 		config: config,
// 		client: wrapper.NewClusterClient(wrapper.DnsCluster{
// 			ServiceName: config.DashVectorServiceName,
// 			Port:        dashVectorPort,
// 			Domain:      config.DashVectorAuthApiEnd,
// 		}),
// 	}, nil
// }

// type DvProvider struct {
// 	config ProviderConfig
// 	client wrapper.HttpClient
// }

// func (d *DvProvider) GetProviderType() string {
// 	return providerTypeDashVector
// }

// type EmbeddingRequest struct {
// 	Model      string `json:"model"`
// 	Input      Input  `json:"input"`
// 	Parameters Params `json:"parameters"`
// }

// type Params struct {
// 	TextType string `json:"text_type"`
// }

// type Input struct {
// 	Texts []string `json:"texts"`
// }

// func (d *DvProvider) ConstructEmbeddingQueryParameters(vector []float64) (string, []byte, [][2]string, error) {
// 	url := fmt.Sprintf("/v1/collections/%s/query", d.config.DashVectorCollection)

// 	requestData := QueryRequest{
// 		Vector:        vector,
// 		TopK:          d.config.DashVectorTopK,
// 		IncludeVector: false,
// 	}

// 	requestBody, err := json.Marshal(requestData)
// 	if err != nil {
// 		return "", nil, nil, err
// 	}

// 	header := [][2]string{
// 		{"Content-Type", "application/json"},
// 		{"dashvector-auth-token", d.config.DashVectorKey},
// 	}

// 	return url, requestBody, header, nil
// }

// func (d *DvProvider) ParseQueryResponse(responseBody []byte) (QueryResponse, error) {
// 	var queryResp QueryResponse
// 	err := json.Unmarshal(responseBody, &queryResp)
// 	if err != nil {
// 		return QueryResponse{}, err
// 	}
// 	return queryResp, nil
// }

// func (d *DvProvider) QueryEmbedding(
// 	queryEmb []float64,
// 	ctx wrapper.HttpContext,
// 	log wrapper.Log,
// 	callback func(query_resp QueryResponse, ctx wrapper.HttpContext, log wrapper.Log)) {

// 	// 构造请求参数
// 	url, body, headers, err := d.ConstructEmbeddingQueryParameters(queryEmb)
// 	if err != nil {
// 		log.Infof("Failed to construct embedding query parameters: %v", err)
// 	}

// 	err = d.client.Post(url, headers, body,
// 		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
// 			log.Infof("Query embedding response: %d, %s", statusCode, responseBody)
// 			query_resp, err_query := d.ParseQueryResponse(responseBody)
// 			if err_query != nil {
// 				log.Infof("Failed to parse response: %v", err_query)
// 			}
// 			callback(query_resp, ctx, log)
// 		},
// 		d.config.DashVectorTimeout)
// 	if err != nil {
// 		log.Infof("Failed to query embedding: %v", err)
// 	}

// }

// type Document struct {
// 	Vector []float64         `json:"vector"`
// 	Fields map[string]string `json:"fields"`
// }

// type InsertRequest struct {
// 	Docs []Document `json:"docs"`
// }

// func (d *DvProvider) ConstructEmbeddingUploadParameters(emb []float64, query_string string) (string, []byte, [][2]string, error) {
// 	url := "/v1/collections/" + d.config.DashVectorCollection + "/docs"

// 	doc := Document{
// 		Vector: emb,
// 		Fields: map[string]string{
// 			"query": query_string,
// 		},
// 	}

// 	requestBody, err := json.Marshal(InsertRequest{Docs: []Document{doc}})
// 	if err != nil {
// 		return "", nil, nil, err
// 	}

// 	header := [][2]string{
// 		{"Content-Type", "application/json"},
// 		{"dashvector-auth-token", d.config.DashVectorKey},
// 	}

// 	return url, requestBody, header, err
// }

//	func (d *DvProvider) UploadEmbedding(query_emb []float64, queryString string, ctx wrapper.HttpContext, log wrapper.Log, callback func(ctx wrapper.HttpContext, log wrapper.Log)) {
//		url, body, headers, _ := d.ConstructEmbeddingUploadParameters(query_emb, queryString)
//		d.client.Post(
//			url,
//			headers,
//			body,
//			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
//				log.Infof("statusCode:%d, responseBody:%s", statusCode, string(responseBody))
//				callback(ctx, log)
//			},
//			d.config.DashVectorTimeout)
//	}
package vectorDatabase
