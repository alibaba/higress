package google

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

type GoogleSearch struct {
	optionArgs         map[string]string
	apiKey             string
	cx                 string
	start              int
	count              int
	timeoutMillisecond uint32
	client             wrapper.HttpClient
}

func NewGoogleSearch(config *gjson.Result) (*GoogleSearch, error) {
	engine := &GoogleSearch{}
	engine.apiKey = config.Get("apiKey").String()
	if engine.apiKey == "" {
		return nil, errors.New("apiKey not found")
	}
	engine.cx = config.Get("cx").String()
	if engine.cx == "" {
		return nil, errors.New("cx not found")
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
	if engine.count > 10 || engine.start+engine.count > 100 {
		return nil, errors.New("count must be less than 10, and start + count must be less than or equal to 100.")
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

func (g GoogleSearch) NeedExectue(ctx engine.SearchContext) bool {
	return ctx.EngineType == "" || ctx.EngineType == "internet"
}

func (g GoogleSearch) Client() wrapper.HttpClient {
	return g.client
}

func (g GoogleSearch) CallArgs(ctx engine.SearchContext) engine.CallArgs {
	queryUrl := fmt.Sprintf("https://customsearch.googleapis.com/customsearch/v1?cx=%s&q=%s&num=%d&key=%s&start=%d",
		g.cx, url.QueryEscape(strings.Join(ctx.Querys, " ")), g.count, g.apiKey, g.start+1)
	var extraArgs []string
	for key, value := range g.optionArgs {
		extraArgs = append(extraArgs, fmt.Sprintf("%s=%s", key, url.QueryEscape(value)))
	}
	if ctx.Language != "" {
		extraArgs = append(extraArgs, fmt.Sprintf("lr=lang_%s", ctx.Language))
	}
	if len(extraArgs) > 0 {
		queryUrl = fmt.Sprintf("%s&%s", queryUrl, strings.Join(extraArgs, "&"))
	}
	return engine.CallArgs{
		Method: http.MethodGet,
		Url:    queryUrl,
		Headers: [][2]string{
			{"Accept", "application/json"},
		},
		TimeoutMillisecond: g.timeoutMillisecond,
	}
}

func (g GoogleSearch) ParseResult(ctx engine.SearchContext, response []byte) []engine.SearchResult {
	jsonObj := gjson.ParseBytes(response)
	var results []engine.SearchResult
	for _, item := range jsonObj.Get("items").Array() {
		content := item.Get("snippet").String()
		metaDescription := item.Get("pagemap.metatags.0.og:description").String()
		if metaDescription != "" {
			content = fmt.Sprintf("%s\n...\n%s", content, metaDescription)
		}
		result := engine.SearchResult{
			Title:   item.Get("title").String(),
			Link:    item.Get("link").String(),
			Content: content,
		}
		if result.Valid() {
			results = append(results, result)
		}
	}
	return results
}
