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
	"encoding/json"
	"net/http"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/santhosh-tekuri/jsonschema"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	BufferBody    = "bufferBody"
	DefaultSchema = "defaultSchema"
)

type PluginConfig struct {
	serviceName    string                 `required:"true" json:"serviceName" yaml:"serviceName"`
	serviceDomain  string                 `required:"true" json:"serviceDomain" yaml:"serviceDomain"`
	servicePort    int                    `required:"true" json:"servicePort" yaml:"servicePort"`
	serviceTimeout int                    `required:"false" json:"serviceTimeout" yaml:"serviceTimeout"`
	maxRetry       int                    `required:"false" json:"maxRetry" yaml:"maxRetry"`
	contentPath    string                 `required:"false" json:"contentPath" yaml:"contentPath"`
	jsonSchema     map[string]interface{} `required:"false" json:"jsonSchema" yaml:"jsonSchema"`
	enableSwagger  bool                   `required:"false" json:"enableSwagger" yaml:"enableSwagger"`
	enableOas3     bool                   `required:"false" json:"enableOas3" yaml:"enableOas3"`
	serviceClient  wrapper.HttpClient
	draft          *jsonschema.Draft
	compiler       *jsonschema.Compiler
}

func main() {
	wrapper.SetCtx(
		"ai-json-resp",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

type ReplayBuffer struct {
	url        string
	ReqHeader  [][2]string
	ReqBody    []byte
	RespHeader [][2]string
	RespBody   []byte
}

func parseConfig(result gjson.Result, config *PluginConfig, log wrapper.Log) error {
	config.serviceName = result.Get("serviceName").String()
	log.Debugf("Service name: %s", config.serviceName)
	config.serviceDomain = result.Get("serviceDomain").String()
	log.Debugf("Service domain: %s", config.serviceDomain)
	config.servicePort = int(result.Get("servicePort").Int())
	log.Debugf("Service port: %d", config.servicePort)
	config.serviceTimeout = int(result.Get("serviceTimeout").Int())
	if config.serviceTimeout == 0 {
		config.serviceTimeout = 50000
	}
	config.maxRetry = int(result.Get("maxRetry").Int())
	if config.maxRetry == 0 {
		config.maxRetry = 3
	}
	config.contentPath = result.Get("contentPath").String()
	if config.contentPath == "" {
		config.contentPath = "choices.0.message.content"
	}

	if schemaValue, ok := result.Get("jsonSchema").Value().(map[string]interface{}); ok {
		config.jsonSchema = schemaValue
	} else {
		config.jsonSchema = ApiTemp
	}

	config.serviceClient = wrapper.NewClusterClient(wrapper.DnsCluster{
		ServiceName: config.serviceName,
		Port:        int64(config.servicePort),
		Domain:      config.serviceDomain,
	})

	enableSwagger := result.Get("enableSwagger").Bool()
	enableOas3 := result.Get("enableOas3").Bool()

	// set draft version
	if enableSwagger {
		config.draft = jsonschema.Draft4
	}
	if enableOas3 {
		config.draft = jsonschema.Draft7
	}
	if !enableSwagger && !enableOas3 {
		config.draft = jsonschema.Draft7
	}

	// create compiler
	compiler := jsonschema.NewCompiler()
	compiler.Draft = config.draft
	config.compiler = compiler

	jsonSchemaBytes, _ := json.Marshal(config.jsonSchema)
	jsonSchemaStr := string(jsonSchemaBytes)
	config.compiler.AddResource(DefaultSchema, strings.NewReader(jsonSchemaStr))

	return nil
}

func (r *ReplayBuffer) assembleReqBody(config PluginConfig) []byte {
	var reqBodystrut chatCompletionRequest
	json.Unmarshal(r.ReqBody, &reqBodystrut)
	// 从响应体中提取所需的内容
	content := gjson.ParseBytes(r.RespBody).Get(config.contentPath).String()

	// 将jsonSchema转换为字符串
	jsonSchemaBytes, _ := json.Marshal(config.jsonSchema)
	jsonSchemaStr := string(jsonSchemaBytes)

	// 组合成一个问题
	askQuestion := "Given the json schema: " + jsonSchemaStr + ", please help me construct the following content to a pure json: " + content
	askQuestion += "\n Do not response other content except the pure json!!!!"

	reqBodystrut.Messages = []chatMessage{
		{
			Role:    "user",
			Content: askQuestion,
		},
	}

	// 将结构体转换为byte
	reqBody, _ := json.Marshal(reqBodystrut)
	return reqBody
}

func (c PluginConfig) ValidateJson(body []byte, log wrapper.Log) string {
	content := gjson.ParseBytes(body).Get(c.contentPath).String()
	// first extract json from response body
	if content == "" {
		return ""
	}
	jsonStr := c.ExtractJson(content)

	if jsonStr == "" {
		return ""
	}
	// second use json schema to validate the json
	compile, err := c.compiler.Compile(DefaultSchema)
	if err != nil {
		return ""
	}

	// validate the json
	err = compile.Validate(strings.NewReader(jsonStr))
	if err != nil {
		log.Debugf("json schema validate failed: %v", err)
		return ""
	}
	return jsonStr
}

func (c PluginConfig) ExtractJson(bodyStr string) string {
	// simply extract json from response body string
	startIndex := strings.Index(bodyStr, "{")
	endIndex := strings.LastIndex(bodyStr, "}") + 1

	// if not found
	if startIndex == -1 || endIndex == -1 || startIndex >= endIndex {
		return ""
	}

	jsonStr := bodyStr[startIndex:endIndex]

	// attempt to parse the JSON
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return ""
	}
	return jsonStr
}

func sendResponse(ctx wrapper.HttpContext, log wrapper.Log, body []byte, bodyStr string) {
	log.Debugf("final send response: %s", body)
	if body == nil && bodyStr != "" {
		body = []byte(bodyStr)
	}
	header := [][2]string{
		{"Content-Type", "application/json"},
	}
	if body != nil {
		header = append(header, [2]string{"Content-Disposition", "attachment; filename=\"response.json\""})
	}

	proxywasm.SendHttpResponse(uint32(200), header, body, -1)
}

func recursiveRefineJson(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log, retryCount int, bufferRB *ReplayBuffer) {
	// if retry count exceed max retry count, return the response
	if retryCount >= config.maxRetry {
		log.Debugf("retry count exceed max retry count")
		// TODO: should we return some meaningful message?
		sendResponse(ctx, log, nil, "")
		return
	}

	// recursively refine json
	config.serviceClient.Post(bufferRB.url, bufferRB.ReqHeader, bufferRB.assembleReqBody(config),
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			retryCount++
			log.Debugf("[retry request %d/%d] resp code: %d, resp headers: %v, resp body: %s", retryCount, config.maxRetry, statusCode, responseHeaders, responseBody)
			bufferRB.RespBody = responseBody
			validateJson := config.ValidateJson(responseBody, log)
			if validateJson != "" {
				sendResponse(ctx, log, nil, validateJson)
			} else {
				recursiveRefineJson(ctx, config, log, retryCount, bufferRB)
			}
		}, uint32(config.serviceTimeout))
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {

	isBuffer, err := proxywasm.GetHttpRequestHeader("isBuffer")
	if err != nil {
		ctx.SetContext("isBuffer", "false")
	}

	if isBuffer == "true" {
		ctx.SetContext("isBuffer", "true")
		proxywasm.ResumeHttpRequest()
		return types.ActionContinue
	}
	url, _ := proxywasm.GetHttpRequestHeader(":path")
	ctx.SetContext("url", url)

	header, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Infof("get request header failed: %v", err)
	}
	ctx.SetContext("headers", header)

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	if ctx.GetContext("isBuffer").(string) == "true" {
		log.Debugf("detect buffer_request, skip sending request to AI service")
		return types.ActionContinue
	}

	header := ctx.GetContext("headers").([][2]string)
	if header == nil {
		header = [][2]string{
			{"Content-Type", "application/json"},
		}
	}
	url := ctx.GetContext("url").(string)
	if url == "" {
		log.Debugf("get request url failed")
		url = "/v1/chat/completions"
	}

	header = append(header, [2]string{"isBuffer", "true"})
	bufferRB := &ReplayBuffer{
		url:       url,
		ReqHeader: header,
		ReqBody:   body,
	}

	config.serviceClient.Post(bufferRB.url, bufferRB.ReqHeader, bufferRB.ReqBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			bufferRB.RespBody = responseBody
			log.Debugf("[first request] resp code: %d, resp headers: %v, resp body: %s", statusCode, responseHeaders, responseBody)
			validateJson := config.ValidateJson(responseBody, log)
			if validateJson != "" {
				sendResponse(ctx, log, nil, validateJson)
				return
			} else {
				retryCount := 0
				recursiveRefineJson(ctx, config, log, retryCount, bufferRB)
			}
		}, uint32(config.serviceTimeout))

	return types.ActionPause
}
