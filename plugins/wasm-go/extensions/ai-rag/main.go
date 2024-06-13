package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"myplugin/dashscope"
	"myplugin/dashvector"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"ai-rag",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

type AIRagConfig struct {
	DashScopeClient      wrapper.HttpClient
	DashScopeAPIKey      string
	DashVectorClient     wrapper.HttpClient
	DashVectorAPIKey     string
	DashVectorCollection string
}

type Request struct {
	Model     string    `json:"model"`
	Input     Input     `json:"input"`
	Parameter Parameter `json:"parameters"`
}

type Input struct {
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Parameter struct {
	ResultFormat string `json:"result_format"`
}

func parseConfig(json gjson.Result, config *AIRagConfig, log wrapper.Log) error {
	config.DashScopeAPIKey = json.Get("dashscope.apiKey").String()

	config.DashScopeClient = wrapper.NewClusterClient(wrapper.DnsCluster{
		ServiceName: json.Get("dashscope.serviceName").String(),
		Port:        json.Get("dashscope.servicePort").Int(),
		Domain:      json.Get("dashscope.domain").String(),
	})
	config.DashVectorAPIKey = json.Get("dashvector.apiKey").String()
	config.DashVectorCollection = json.Get("dashvector.collection").String()
	config.DashVectorClient = wrapper.NewClusterClient(wrapper.DnsCluster{
		ServiceName: json.Get("dashvector.serviceName").String(),
		Port:        json.Get("dashvector.servicePort").Int(),
		Domain:      json.Get("dashvector.domain").String(),
	})
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIRagConfig, log wrapper.Log) types.Action {
	proxywasm.RemoveHttpRequestHeader("content-length")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIRagConfig, body []byte, log wrapper.Log) types.Action {
	var rawRequest Request
	_ = json.Unmarshal(body, &rawRequest)
	rawContent := rawRequest.Input.Messages[0].Content
	requestEmbedding := dashscope.Request{
		Model: "text-embedding-v1",
		Input: dashscope.Input{
			Texts: []string{rawContent},
		},
		Parameter: dashscope.Parameter{
			TextType: "query",
		},
	}
	headers := [][2]string{{"Content-Type", "application/json"}, {"Authorization", "Bearer " + config.DashScopeAPIKey}}
	reqEmbeddingSerialized, _ := json.Marshal(requestEmbedding)
	// log.Info(string(reqEmbeddingSerialized))
	config.DashScopeClient.Post(
		"/api/v1/services/embeddings/text-embedding/text-embedding",
		headers,
		reqEmbeddingSerialized,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			var responseEmbedding dashscope.Response
			_ = json.Unmarshal(responseBody, &responseEmbedding)
			requestQuery := dashvector.Request{
				TopK:         1,
				OutputFileds: []string{"raw"},
				Vector:       responseEmbedding.Output.Embeddings[0].Embedding,
			}
			requestQuerySerialized, _ := json.Marshal(requestQuery)
			config.DashVectorClient.Post(
				fmt.Sprintf("/v1/collections/%s/query", config.DashVectorCollection),
				[][2]string{{"Content-Type", "application/json"}, {"dashvector-auth-token", config.DashVectorAPIKey}},
				requestQuerySerialized,
				func(statusCode int, responseHeaders http.Header, responseBody []byte) {
					var response dashvector.Response
					_ = json.Unmarshal(responseBody, &response)
					doc := response.Output[0].Fields.Raw
					rawRequest.Input.Messages[0].Content = fmt.Sprintf("%s\n参考以上信息，对以下问题做出回答：%s", doc, rawContent)
					newBody, _ := json.Marshal(rawRequest)
					// log.Info(string(newBody))
					proxywasm.ReplaceHttpRequestBody(newBody)
					proxywasm.ResumeHttpRequest()
				},
			)
		},
		50000,
	)
	return types.ActionPause
}
