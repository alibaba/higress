package elasticsearch

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-search/engine"
)

type ElasticsearchSearch struct {
	client             wrapper.HttpClient
	index              string
	contentField       string
	semanticTextField  string
	linkField          string
	titleField         string
	start              int
	count              int
	timeoutMillisecond uint32
	username           string
	password           string
}

func NewElasticsearchSearch(config *gjson.Result) (*ElasticsearchSearch, error) {
	engine := &ElasticsearchSearch{}
	serviceName := config.Get("serviceName").String()
	if serviceName == "" {
		return nil, errors.New("serviceName not found")
	}
	servicePort := config.Get("servicePort").Int()
	if servicePort == 0 {
		return nil, errors.New("servicePort not found")
	}
	engine.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
	})
	engine.index = config.Get("index").String()
	if engine.index == "" {
		return nil, errors.New("index not found")
	}

	engine.contentField = config.Get("contentField").String()
	if engine.contentField == "" {
		return nil, errors.New("contentField not found")
	}
	engine.semanticTextField = config.Get("semanticTextField").String()
	if engine.semanticTextField == "" {
		return nil, errors.New("semanticTextField not found")
	}
	engine.linkField = config.Get("linkField").String()
	if engine.linkField == "" {
		return nil, errors.New("linkField not found")
	}
	engine.titleField = config.Get("titleField").String()
	if engine.titleField == "" {
		return nil, errors.New("titleField not found")
	}
	engine.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if engine.timeoutMillisecond == 0 {
		engine.timeoutMillisecond = 5000
	}
	engine.start = int(config.Get("start").Uint())
	engine.count = int(config.Get("count").Uint())
	if engine.count == 0 {
		engine.count = 10
	}

	engine.username = config.Get("username").String()
	engine.password = config.Get("password").String()

	return engine, nil
}

func (e ElasticsearchSearch) NeedExectue(ctx engine.SearchContext) bool {
	return ctx.EngineType == "private" || ctx.EngineType == ""
}

func (e ElasticsearchSearch) Client() wrapper.HttpClient {
	return e.client
}

func (e ElasticsearchSearch) generateAuthorizationHeader() string {
	return fmt.Sprintf(`Basic %s`, base64.StdEncoding.EncodeToString([]byte(e.username+":"+e.password)))
}

func (e ElasticsearchSearch) generateQueryBody(ctx engine.SearchContext) string {
	queryText := strings.Join(ctx.Querys, " ")
	return fmt.Sprintf(`{
		"retriever": {
			"rrf": {
				"retrievers": [
					{
						"standard": { 
							"query": {
								"match": {
									"%s": "%s" 
								}
							}
						}
					},
					{
						"standard": { 
							"query": {
								"semantic": {
									"field": "%s", 
									"query": "%s"
								}
							}
						}
					}
				]
			}
		}
	}`, e.contentField, queryText, e.semanticTextField, queryText)
}

func (e ElasticsearchSearch) CallArgs(ctx engine.SearchContext) engine.CallArgs {
	queryBody := e.generateQueryBody(ctx)
	return engine.CallArgs{
		Method: http.MethodPost,
		Url:    fmt.Sprintf("/%s/_search?from=%d&size=%d", e.index, e.start, e.count),
		Headers: [][2]string{
			{"Content-Type", "application/json"},
			{"Authorization", e.generateAuthorizationHeader()},
		},
		Body:               []byte(queryBody),
		TimeoutMillisecond: e.timeoutMillisecond,
	}
}

func (e ElasticsearchSearch) ParseResult(ctx engine.SearchContext, response []byte) []engine.SearchResult {
	jsonObj := gjson.ParseBytes(response)
	var results []engine.SearchResult
	for _, hit := range jsonObj.Get("hits.hits").Array() {
		source := hit.Get("_source")
		result := engine.SearchResult{
			Title:   source.Get(e.titleField).String(),
			Link:    source.Get(e.linkField).String(),
			Content: source.Get(e.contentField).String(),
		}
		if result.Valid() {
			results = append(results, result)
		}
	}
	return results
}
