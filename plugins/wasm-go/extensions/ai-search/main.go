// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	DEFAULT_MAX_BODY_BYTES uint32 = 100 * 1024 * 1024
)

func main() {
	wrapper.SetCtx(
		"ai-search",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onStreamingResponseBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

type searchResult struct {
	title   string
	link    string
	content string
}

func (result searchResult) valid() bool {
	return result.title != "" && result.link != "" && result.content != ""
}

type searchContext struct {
	query    string
	needNews bool
	language string
}

type callArgs struct {
	method             string
	url                string
	headers            [][2]string
	body               []byte
	timeoutMillisecond uint32
}

type searchEngine interface {
	Client() wrapper.HttpClient
	CallArgs(ctx searchContext) callArgs
	ParseResult(ctx searchContext, response []byte) []searchResult
}

type GoogleSearch struct {
	apiKey             string
	cx                 string
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
	engine.count = int(config.Get("count").Uint())
	if engine.count == 0 {
		engine.count = 10
	}
	engine.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if engine.timeoutMillisecond == 0 {
		engine.timeoutMillisecond = 5000
	}
	return engine, nil
}

func (engine GoogleSearch) Client() wrapper.HttpClient {
	return engine.client
}

func (engine GoogleSearch) CallArgs(ctx searchContext) callArgs {
	return callArgs{
		method: http.MethodGet,
		url: fmt.Sprintf("https://customsearch.googleapis.com/customsearch/v1?cx=%s&q=%s&num=%d&lr=lang_%s&key=%s",
			engine.cx, url.QueryEscape(ctx.query), engine.count, ctx.language, engine.apiKey),
		headers: [][2]string{
			{"Accept", "application/json"},
		},
		timeoutMillisecond: engine.timeoutMillisecond,
	}
}

func (engine GoogleSearch) ParseResult(ctx searchContext, response []byte) []searchResult {
	jsonObj := gjson.ParseBytes(response)
	var results []searchResult
	for _, item := range jsonObj.Get("items").Array() {
		result := searchResult{
			title:   item.Get("title").String(),
			link:    item.Get("link").String(),
			content: item.Get("snippet").String(),
		}
		if result.valid() {
			results = append(results, result)
		}
	}
	return results
}

type BeingSearch struct {
	apiKey             string
	count              int
	newsFirst          bool
	timeoutMillisecond uint32
	client             wrapper.HttpClient
}

func NewBeingSearch(config *gjson.Result) (*BeingSearch, error) {
	engine := &BeingSearch{}
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
	engine.count = int(config.Get("count").Uint())
	if engine.count == 0 {
		engine.count = 10
	}
	engine.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if engine.timeoutMillisecond == 0 {
		engine.timeoutMillisecond = 5000
	}
	engine.newsFirst = config.Get("newsFirst").Bool()
	return engine, nil
}

func (engine BeingSearch) Client() wrapper.HttpClient {
	return engine.client
}

func (engine BeingSearch) CallArgs(ctx searchContext) callArgs {
	filter := "webpage"
	var appendArgs []string
	if ctx.needNews || engine.newsFirst {
		filter += ",news"
		appendArgs = append(appendArgs, fmt.Sprintf("answerCount=%d&promote=news", engine.count))
	}
	mkt := ctx.language
	return callArgs{
		method: http.MethodGet,
		url: fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&responseFilter=%s&count=%d&mkt=%s&%s",
			url.QueryEscape(ctx.query), filter, engine.count, mkt, strings.Join(appendArgs, "&")),
		headers:            [][2]string{{"Ocp-Apim-Subscription-Key", engine.apiKey}},
		timeoutMillisecond: engine.timeoutMillisecond,
	}
}
func (engine BeingSearch) ParseResult(ctx searchContext, response []byte) []searchResult {
	jsonObj := gjson.ParseBytes(response)
	var results []searchResult
	if ctx.needNews || engine.newsFirst {
		news := jsonObj.Get("news.value")
		for _, article := range news.Array() {
			result := searchResult{
				title:   article.Get("name").String(),
				link:    article.Get("url").String(),
				content: article.Get("description").String(),
			}
			if result.valid() {
				results = append(results, result)
			}
		}
	}
	webPages := jsonObj.Get("webPages.value")
	for _, page := range webPages.Array() {
		result := searchResult{
			title:   page.Get("name").String(),
			link:    page.Get("url").String(),
			content: page.Get("snippet").String(),
		}
		if result.valid() {
			results = append(results, result)
		}
		deepLinks := page.Get("deepLinks")
		for _, inner := range deepLinks.Array() {
			innerResult := searchResult{
				title:   inner.Get("name").String(),
				link:    inner.Get("url").String(),
				content: inner.Get("snippet").String(),
			}
			if innerResult.valid() {
				results = append(results, innerResult)
			}
		}
	}
	return results
}

type Config struct {
	engine          []searchEngine
	promptTemplate  string
	referenceFormat string
	defaultLanguage string
	needReference   bool
}

func parseConfig(json gjson.Result, config *Config, log wrapper.Log) error {
	config.needReference = json.Get("needReference").Bool()
	if config.needReference {
		config.referenceFormat = json.Get("referenceFormat").String()
		if config.referenceFormat == "" {
			config.referenceFormat = "**References:**\n%s"
		} else if !strings.Contains(config.referenceFormat, "%s") {
			return fmt.Errorf("invalid referenceFormat:%s", config.referenceFormat)
		}
	}
	config.defaultLanguage = json.Get("defaultLang").String()
	if config.defaultLanguage == "" {
		config.defaultLanguage = "zh-CN"
	}

	config.promptTemplate = json.Get("promptTemplate").String()
	if config.promptTemplate == "" {
		if config.needReference {
			config.promptTemplate = `# 以下内容是基于用户发送的消息的搜索结果:
{search_results}
在我给你的搜索结果中，每个结果都是[webpage X begin]...[webpage X end]格式的，X代表每篇文章的搜索排序和数字索引。请在适当的情况下在句子末尾引用上下文。请按照引用编号[X]的格式在答案中对应部分引用上下文。如果一句话源自多个上下文，请列出所有相关的引用编号，例如[3][5]，切记不要将引用集中在最后返回引用编号，而是在答案对应部分列出。
在回答时，请注意以下几点：
- 今天是北京时间：{cur_date}。
- 并非搜索结果的所有内容都与用户的问题密切相关，你需要结合问题，对搜索结果进行甄别、筛选。
- 对于列举类的问题（如列举所有航班信息），尽量将答案控制在10个要点以内，并告诉用户可以查看搜索来源、获得完整信息。优先提供信息完整、最相关的列举项；如非必要，不要主动告诉用户搜索结果未提供的内容。
- 对于创作类的问题（如写论文），请务必在正文的段落中引用对应的参考编号，例如[3][5]，不能只在文章末尾引用。你需要解读并概括用户的题目要求，选择合适的格式，充分利用搜索结果并抽取重要信息，生成符合用户要求、极具思想深度、富有创造力与专业性的答案。你的创作篇幅需要尽可能延长，对于每一个要点的论述要推测用户的意图，给出尽可能多角度的回答要点，且务必信息量大、论述详尽。
- 如果回答很长，请尽量结构化、分段落总结。如果需要分点作答，尽量控制在5个点以内，并合并相关的内容。
- 对于客观类的问答，如果问题的答案非常简短，可以适当补充一到两句相关信息，以丰富内容。
- 你需要根据用户要求和回答内容选择合适、美观的回答格式，确保可读性强。
- 你的回答应该综合多个相关网页来回答，不能重复引用一个网页。
- 除非用户要求，否则你回答的语言需要和用户提问的语言保持一致。

# 用户消息为：
{question}`
		} else {
			config.promptTemplate = `# 以下内容是基于用户发送的消息的搜索结果:
{search_results}
在我给你的搜索结果中，每个结果都是[webpage X begin]...[webpage X end]格式的，X代表每篇文章的搜索排序。
在回答时，请注意以下几点：
- 今天是北京时间：{cur_date}。
- 并非搜索结果的所有内容都与用户的问题密切相关，你需要结合问题，对搜索结果进行甄别、筛选。
- 对于列举类的问题（如列举所有航班信息），尽量将答案控制在10个要点以内，并告诉用户可以查看搜索来源、获得完整信息。优先提供信息完整、最相关的列举项；如非必要，不要主动告诉用户搜索结果未提供的内容。
- 对于创作类的问题（如写论文），你需要解读并概括用户的题目要求，选择合适的格式，充分利用搜索结果并抽取重要信息，生成符合用户要求、极具思想深度、富有创造力与专业性的答案。你的创作篇幅需要尽可能延长，对于每一个要点的论述要推测用户的意图，给出尽可能多角度的回答要点，且务必信息量大、论述详尽。
- 如果回答很长，请尽量结构化、分段落总结。如果需要分点作答，尽量控制在5个点以内，并合并相关的内容。
- 对于客观类的问答，如果问题的答案非常简短，可以适当补充一到两句相关信息，以丰富内容。
- 你需要根据用户要求和回答内容选择合适、美观的回答格式，确保可读性强。
- 你的回答应该综合多个相关网页来回答，不能重复引用一个网页。
- 除非用户要求，否则你回答的语言需要和用户提问的语言保持一致。

# 用户消息为：
{question}`
		}
	}
	if !strings.Contains(config.promptTemplate, "{search_results}") ||
		!strings.Contains(config.promptTemplate, "{question}") {
		return fmt.Errorf("invalid promptTemplate, must contains {search_results} and {question}:%s", config.promptTemplate)
	}
	for _, engine := range json.Get("searchFrom").Array() {
		switch engine.Get("type").String() {
		case "being":
			searchEngine, err := NewBeingSearch(&engine)
			if err != nil {
				return fmt.Errorf("being search engine init failed:%s", err)
			}
			config.engine = append(config.engine, searchEngine)
		case "google":
			searchEngine, err := NewGoogleSearch(&engine)
			if err != nil {
				return fmt.Errorf("google search engine init failed:%s", err)
			}
			config.engine = append(config.engine, searchEngine)
		default:
			return fmt.Errorf("unkown search engine:%s", engine.Get("type").String())
		}
	}
	if len(config.engine) == 0 {
		return fmt.Errorf("no avaliable search engine found")
	}
	log.Debugf("ai search enabled, config: %#v", config)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	// The request does not have a body.
	if contentType == "" {
		return types.ActionContinue
	}
	if !strings.Contains(contentType, "application/json") {
		log.Warnf("content is not json, can't process: %s", contentType)
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	ctx.SetRequestBodyBufferLimit(DEFAULT_MAX_BODY_BYTES)
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config Config, body []byte, log wrapper.Log) types.Action {
	var queryIndex int
	var query string
	messages := gjson.GetBytes(body, "messages").Array()
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Get("role").String() == "user" {
			queryIndex = i
			query = messages[i].Get("content").String()
			break
		}
	}
	if query == "" {
		log.Errorf("not found user query in body:%s", body)
		return types.ActionContinue
	}
	var searchResultGroups [][]searchResult = make([][]searchResult, len(config.engine))
	var finished int
	var searching int
	for i := 0; i < len(config.engine); i++ {
		engine := config.engine[i]
		searchCtx := searchContext{query: query, language: config.defaultLanguage}
		args := engine.CallArgs(searchCtx)
		index := i
		err := engine.Client().Call(args.method, args.url, args.headers, args.body,
			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
				defer func() {
					finished++
					if finished == searching {
						// Merge and sort search results from all engines
						var mergedResults []searchResult
						maxResults := 0
						for _, results := range searchResultGroups {
							if len(results) > maxResults {
								maxResults = len(results)
							}
						}
						for j := 0; j < maxResults; j++ {
							for _, results := range searchResultGroups {
								if j < len(results) {
									mergedResults = append(mergedResults, results[j])
								}
							}
						}
						// Format search results for prompt template
						var formattedResults []string
						for j, result := range mergedResults {
							formattedResults = append(formattedResults, fmt.Sprintf("[webpage %d begin]\n%s\n[webpage %d end]",
								j+1, result.content, j+1))
						}
						// Prepare template variables
						curDate := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006年1月2日")
						searchResults := strings.Join(formattedResults, "\n")
						log.Debugf("searchResults: %s", searchResults)
						// Fill prompt template
						prompt := strings.Replace(config.promptTemplate, "{search_results}", searchResults, 1)
						prompt = strings.Replace(prompt, "{question}", query, 1)
						prompt = strings.Replace(prompt, "{cur_date}", curDate, 1)
						// Update request body with processed prompt
						modifiedBody, err := sjson.SetBytes(body, fmt.Sprintf("messages.%d.content", queryIndex), prompt)
						if err != nil {
							log.Errorf("modify request message content failed, err:%v, body:%s", err, body)
						} else {
							log.Debugf("modifeid body:%s", modifiedBody)
							proxywasm.ReplaceHttpRequestBody(modifiedBody)
						}
						proxywasm.ResumeHttpRequest()
					}
				}()
				if statusCode != http.StatusOK {
					log.Errorf("search call failed, status: %d, engine: %#v", statusCode, engine)
					return
				}
				searchResultGroups[index] = engine.ParseResult(searchCtx, responseBody)
			}, args.timeoutMillisecond)
		if err != nil {
			log.Errorf("serach call failed, engine: %#v", engine)
			continue
		}
		searching++
	}
	if searching > 0 {
		return types.ActionPause
	}
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	if !config.needReference {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	proxywasm.RemoveHttpResponseHeader("content-length")
	contentType, err := proxywasm.GetHttpResponseHeader("Content-Type")
	if err != nil || !strings.HasPrefix(contentType, "text/event-stream") {
		if err != nil {
			log.Errorf("unable to load content-type header from response: %v", err)
		}
		ctx.BufferResponseBody()
		ctx.SetResponseBodyBufferLimit(DEFAULT_MAX_BODY_BYTES)
	}
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config Config, body []byte, log wrapper.Log) types.Action {
	references := ctx.GetStringContext("References", "")
	if references == "" {
		return types.ActionContinue
	}
	content := gjson.GetBytes(body, "choices.0.message.content")
	modifiedContent := fmt.Sprintf("%s\n\n%s", fmt.Sprintf(config.referenceFormat, references), content)
	body, err := sjson.SetBytes(body, "choices.0.message.content", modifiedContent)
	if err != nil {
		log.Errorf("modify response message content failed, err:%v, body:%s", err, body)
		return types.ActionContinue
	}
	proxywasm.ReplaceHttpResponseBody(body)
	return types.ActionContinue
}

func onStreamingResponseBody(ctx wrapper.HttpContext, config Config, chunk []byte, isLastChunk bool, log wrapper.Log) []byte {
	if ctx.GetBoolContext("ReferenceAppended", false) {
		return chunk
	}
	references := ctx.GetStringContext("References", "")
	if references == "" {
		return chunk
	}
	modifiedChunk, responseReady := setReferencesToFirstMessage(ctx, chunk, fmt.Sprintf(config.referenceFormat, references), log)
	if responseReady {
		ctx.SetContext("ReferenceAppended", true)
		return modifiedChunk
	} else {
		return []byte("")
	}
}

const PARTIAL_MESSAGE_CONTEXT_KEY = "partialMessage"

func setReferencesToFirstMessage(ctx wrapper.HttpContext, chunk []byte, references string, log wrapper.Log) ([]byte, bool) {
	if len(chunk) == 0 {
		log.Debugf("chunk is empty")
		return nil, false
	}

	var partialMessage []byte
	partialMessageI := ctx.GetContext(PARTIAL_MESSAGE_CONTEXT_KEY)
	if partialMessageI != nil {
		if pMsg, ok := partialMessageI.([]byte); ok {
			partialMessage = append(pMsg, chunk...)
		} else {
			log.Warnf("invalid partial message type: %T", partialMessageI)
			partialMessage = chunk
		}
	} else {
		partialMessage = chunk
	}

	if len(partialMessage) == 0 {
		log.Debugf("partial message is empty")
		return nil, false
	}
	messages := strings.Split(string(partialMessage), "\n\n")
	if len(messages) > 1 {
		firstMessage := messages[0]
		log.Debugf("first message: %s", firstMessage)
		firstMessage = strings.TrimPrefix(firstMessage, "data: ")
		firstMessage = strings.TrimSuffix(firstMessage, "\n")
		deltaContent := gjson.Get(firstMessage, "choices.0.delta.content")
		modifiedMessage, err := sjson.Set(firstMessage, "choices.0.delta.content", fmt.Sprintf("%s\n\n%s", references, deltaContent))
		if err != nil {
			log.Errorf("modify response delta content failed, err:%v", err)
			return partialMessage, true
		}
		modifiedMessage = fmt.Sprintf("data: %s", modifiedMessage)
		log.Debugf("modified message: %s", firstMessage)
		messages[0] = string(modifiedMessage)
		return []byte(strings.Join(messages, "\n\n")), true
	}
	ctx.SetContext(PARTIAL_MESSAGE_CONTEXT_KEY, partialMessage)
	return nil, false
}
