package bing

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-search/engine"
)

type BingSearch struct {
	optionArgs         map[string]string
	apiKey             string
	start              int
	count              int
	timeoutMillisecond uint32
	client             wrapper.HttpClient
}

func NewBingSearch(config *gjson.Result) (*BingSearch, error) {
	engine := &BingSearch{}
	engine.apiKey = config.Get("apiKey").String()
	if engine.apiKey == "" {
		return nil, errors.New("apiKey not found")
	}
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
	engine.start = int(config.Get("start").Uint())
	engine.count = int(config.Get("count").Uint())
	if engine.count == 0 {
		engine.count = 10
	}
	engine.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if engine.timeoutMillisecond == 0 {
		engine.timeoutMillisecond = 5000
	}
	engine.optionArgs = map[string]string{}
	for key, value := range config.Get("optionArgs").Map() {
		valStr := value.String()
		if valStr != "" {
			engine.optionArgs[key] = value.String()
		}
	}
	return engine, nil
}

func (b BingSearch) NeedExectue(ctx engine.SearchContext) bool {
	return ctx.EngineType == "" || ctx.EngineType == "internet"
}

func (b BingSearch) Client() wrapper.HttpClient {
	return b.client
}

func (b BingSearch) CallArgs(ctx engine.SearchContext) engine.CallArgs {
	queryUrl := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&count=%d&offset=%d",
		url.QueryEscape(strings.Join(ctx.Querys, " ")), b.count, b.start)
	var extraArgs []string
	for key, value := range b.optionArgs {
		extraArgs = append(extraArgs, fmt.Sprintf("%s=%s", key, url.QueryEscape(value)))
	}
	if ctx.Language != "" {
		extraArgs = append(extraArgs, fmt.Sprintf("mkt=%s", ctx.Language))
	}
	if len(extraArgs) > 0 {
		queryUrl = fmt.Sprintf("%s&%s", queryUrl, strings.Join(extraArgs, "&"))
	}
	return engine.CallArgs{
		Method:             http.MethodGet,
		Url:                queryUrl,
		Headers:            [][2]string{{"Ocp-Apim-Subscription-Key", b.apiKey}},
		TimeoutMillisecond: b.timeoutMillisecond,
	}
}

func (b BingSearch) ParseResult(ctx engine.SearchContext, response []byte) []engine.SearchResult {
	jsonObj := gjson.ParseBytes(response)
	var results []engine.SearchResult
	webPages := jsonObj.Get("webPages.value")
	for _, page := range webPages.Array() {
		result := engine.SearchResult{
			Title:   page.Get("name").String(),
			Link:    page.Get("url").String(),
			Content: page.Get("snippet").String(),
		}
		if result.Valid() {
			results = append(results, result)
		}
		deepLinks := page.Get("deepLinks")
		for _, inner := range deepLinks.Array() {
			innerResult := engine.SearchResult{
				Title:   inner.Get("name").String(),
				Link:    inner.Get("url").String(),
				Content: inner.Get("snippet").String(),
			}
			if innerResult.Valid() {
				results = append(results, innerResult)
			}
		}
	}
	news := jsonObj.Get("news.value")
	for _, article := range news.Array() {
		result := engine.SearchResult{
			Title:   article.Get("name").String(),
			Link:    article.Get("url").String(),
			Content: article.Get("description").String(),
		}
		if result.Valid() {
			results = append(results, result)
		}
	}
	return results
}
