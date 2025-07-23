package arxiv

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-search/engine"
)

type ArxivSearch struct {
	optionArgs         map[string]string
	start              int
	count              int
	timeoutMillisecond uint32
	client             wrapper.HttpClient
	arxivCategory      string
}

func NewArxivSearch(config *gjson.Result) (*ArxivSearch, error) {
	engine := &ArxivSearch{}
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
	engine.arxivCategory = config.Get("arxivCategory").String()
	return engine, nil
}

func (a ArxivSearch) NeedExectue(ctx engine.SearchContext) bool {
	return ctx.EngineType == "arxiv"
}

func (a ArxivSearch) Client() wrapper.HttpClient {
	return a.client
}

func (a ArxivSearch) CallArgs(ctx engine.SearchContext) engine.CallArgs {
	var searchQueryItems []string
	for _, q := range ctx.Querys {
		searchQueryItems = append(searchQueryItems, fmt.Sprintf("all:%s", url.QueryEscape(q)))
	}
	searchQuery := strings.Join(searchQueryItems, "+AND+")
	category := ctx.ArxivCategory
	if category == "" {
		category = a.arxivCategory
	}
	if category != "" {
		searchQuery = fmt.Sprintf("%s+AND+cat:%s", searchQuery, category)
	}
	queryUrl := fmt.Sprintf("https://export.arxiv.org/api/query?search_query=%s&max_results=%d&start=%d",
		searchQuery, a.count, a.start)
	var extraArgs []string
	for key, value := range a.optionArgs {
		extraArgs = append(extraArgs, fmt.Sprintf("%s=%s", key, url.QueryEscape(value)))
	}
	if len(extraArgs) > 0 {
		queryUrl = fmt.Sprintf("%s&%s", queryUrl, strings.Join(extraArgs, "&"))
	}
	return engine.CallArgs{
		Method:             http.MethodGet,
		Url:                queryUrl,
		Headers:            [][2]string{{"Accept", "application/atom+xml"}},
		TimeoutMillisecond: a.timeoutMillisecond,
	}
}

func (a ArxivSearch) ParseResult(ctx engine.SearchContext, response []byte) []engine.SearchResult {
	var results []engine.SearchResult
	doc, err := xmlquery.Parse(bytes.NewReader(response))
	if err != nil {
		return results
	}

	entries := xmlquery.Find(doc, "//entry")
	for _, entry := range entries {
		title := entry.SelectElement("title").InnerText()
		link := ""
		for _, l := range entry.SelectElements("link") {
			if l.SelectAttr("rel") == "alternate" && l.SelectAttr("type") == "text/html" {
				link = l.SelectAttr("href")
				break
			}
		}
		summary := entry.SelectElement("summary").InnerText()
		publishTime := entry.SelectElement("published").InnerText()
		authors := entry.SelectElements("author")
		var authorNames []string
		for _, author := range authors {
			authorNames = append(authorNames, author.SelectElement("name").InnerText())
		}
		content := fmt.Sprintf("%s\nAuthors: %s\nPublication time: %s", summary, strings.Join(authorNames, ", "), publishTime)
		result := engine.SearchResult{
			Title:   title,
			Link:    link,
			Content: content,
		}
		if result.Valid() {
			results = append(results, result)
		}
	}
	return results
}
