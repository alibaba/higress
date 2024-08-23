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
	"errors"
	"net/http"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/santhosh-tekuri/jsonschema"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	DefaultSchema                = "defaultSchema"
	JsonSchemaNotValidCode       = 1001
	JsonSchemaCompileFailedCode  = 1002
	CannotFindJsonInResponseCode = 1003
	ReturnContentisEmpytCode     = 1004
	ReturnJsonMisMatchSchemaCode = 1005
	ReachMeaxRetryCountCode      = 1006
	ServiceUnavailableCode       = 1007
	ServiceConfiginValidCode     = 1008
)

type RejStruct struct {
	RejCode uint32 `json:"Code"`
	RejMsg  string `json:"Msg"`
}

func (r RejStruct) Byte() []byte {
	jsonData, _ := json.Marshal(r)
	return jsonData
}

func (r RejStruct) GetShortMsg() string {
	return "ai-json-resp." + strings.Split(r.RejMsg, ":")[0]
}

type PluginConfig struct {
	// @Title zh-CN 服务名称
	// @Description zh-CN 用以请求服务的名称(网关或其他AI服务)
	serviceName string `required:"true" json:"serviceName" yaml:"serviceName"`
	// @Title zh-CN 服务域名
	// @Description zh-CN 用以请求服务的域名
	serviceDomain string `required:"false" json:"serviceDomain" yaml:"serviceDomain"`
	// @Title zh-CN 服务端口
	// @Description zh-CN 用以请求服务的端口
	servicePort int `required:"false" json:"servicePort" yaml:"servicePort"`
	// @Title zh-CN 服务URL
	// @Description zh-CN 用以请求服务的URL，若提供则会覆盖serviceDomain和servicePort
	serviceUrl string `required:"false" json:"serviceUrl" yaml:"serviceUrl"`
	// @Title zh-CN API Key
	// @Description zh-CN 若使用AI服务，需要填写请求服务的API Key
	apiKey string `required:"false" json: "apiKey" yaml:"apiKey"`
	// @Title zh-CN 请求端点
	// @Description zh-CN 用以请求服务的端点, 默认为"/v1/chat/completions"
	servicePath string `required:"false" json: "servicePath" yaml:"servicePath"`
	// @Title zh-CN 服务超时时间
	// @Description zh-CN 用以请求服务的超时时间
	serviceTimeout int `required:"false" json:"serviceTimeout" yaml:"serviceTimeout"`
	// @Title zh-CN 最大重试次数
	// @Description zh-CN 用以请求服务的最大重试次数
	maxRetry int `required:"false" json:"maxRetry" yaml:"maxRetry"`
	// @Title zh-CN 内容路径
	// @Description zh-CN 从AI服务返回的响应中提取json的gpath路径
	contentPath string `required:"false" json:"contentPath" yaml:"contentPath"`
	// @Title zh-CN Json Schema
	// @Description zh-CN 用以验证响应json的Json Schema, 为空则只验证返回的响应是否为合法json
	jsonSchema map[string]interface{} `required:"false" json:"jsonSchema" yaml:"jsonSchema"`
	// @Title zh-CN 是否启用swagger
	// @Description zh-CN 是否启用swagger进行Json Schema验证
	enableSwagger bool `required:"false" json:"enableSwagger" yaml:"enableSwagger"`
	// @Title zh-CN 是否启用oas3
	// @Description zh-CN 是否启用oas3进行Json Schema验证
	enableOas3    bool `required:"false" json:"enableOas3" yaml:"enableOas3"`
	serviceClient wrapper.HttpClient
	draft         *jsonschema.Draft
	compiler      *jsonschema.Compiler
	compile       *jsonschema.Schema
	rejStruct     RejStruct
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
	HisMsg     []chatMessage
}

func parseUrl(url string) (string, string) {
	if url == "" {
		return "", ""
	}
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	index := strings.Index(url, "/")
	if index == -1 {
		return url, ""
	}
	return url[:index], url[index:]
}

func parseConfig(result gjson.Result, config *PluginConfig, log wrapper.Log) error {
	config.serviceName = result.Get("serviceName").String()
	config.serviceUrl = result.Get("serviceUrl").String()
	config.serviceDomain = result.Get("serviceDomain").String()
	config.servicePath = result.Get("servicePath").String()
	config.servicePort = int(result.Get("servicePort").Int())
	if config.serviceUrl != "" {
		domain, url := parseUrl(config.serviceUrl)
		log.Debugf("serviceUrl: %s, the parsed domain: %s, the parsed url: %s", config.serviceUrl, domain, url)
		if config.serviceDomain == "" {
			config.serviceDomain = domain
		}
		if config.servicePath == "" {
			config.servicePath = url
		}
	}
	if config.servicePort == 0 {
		config.servicePort = 443
	}
	config.serviceTimeout = int(result.Get("serviceTimeout").Int())
	config.apiKey = result.Get("apiKey").String()
	config.rejStruct = RejStruct{uint32(200), ""}
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

	if jsonSchemaValue := result.Get("jsonSchema"); jsonSchemaValue.Exists() {
		if schemaValue, ok := jsonSchemaValue.Value().(map[string]interface{}); ok {
			config.jsonSchema = schemaValue

		} else {
			config.rejStruct = RejStruct{JsonSchemaNotValidCode, "Json Schema is not valid"}
		}
	} else {
		config.jsonSchema = nil
	}

	if config.serviceDomain == "" {
		config.rejStruct = RejStruct{ServiceConfiginValidCode, "service domain is empty"}
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

	jsonSchemaBytes, err := json.Marshal(config.jsonSchema)
	if err != nil {
		config.rejStruct = RejStruct{JsonSchemaNotValidCode, "Json Schema marshal failed"}
		return err
	}
	jsonSchemaStr := string(jsonSchemaBytes)
	config.compiler.AddResource(DefaultSchema, strings.NewReader(jsonSchemaStr))
	// Test if the Json Schema is valid
	compile, err := config.compiler.Compile(DefaultSchema)
	if err != nil {
		log.Infof("Json Schema compile failed: %v", err)
		config.rejStruct = RejStruct{JsonSchemaCompileFailedCode, "Json Schema compile failed: " + err.Error()}
		config.compile = nil
	} else {
		config.compile = compile
	}

	return nil
}

func (r *ReplayBuffer) assembleReqBody(config PluginConfig) []byte {
	var reqBodystrut chatCompletionRequest
	json.Unmarshal(r.ReqBody, &reqBodystrut)
	content := gjson.ParseBytes(r.RespBody).Get(config.contentPath).String()
	jsonSchemaBytes, _ := json.Marshal(config.jsonSchema)
	jsonSchemaStr := string(jsonSchemaBytes)

	askQuestion := "Given the Json Schema: " + jsonSchemaStr + ", please help me construct the following content to a pure json: " + content
	askQuestion += "\n Do not response other content except the pure json!!!!"

	reqBodystrut.Messages = append(r.HisMsg, []chatMessage{
		{
			Role:    "user",
			Content: askQuestion,
		},
	}...)

	reqBody, _ := json.Marshal(reqBodystrut)
	return reqBody
}

func (r *ReplayBuffer) SaveHisBody(log wrapper.Log, reqBody []byte, respBody []byte) {
	r.RespBody = respBody
	lastUserMessage := ""
	lastSystemMessage := ""

	var reqBodystrut chatCompletionRequest
	err := json.Unmarshal(reqBody, &reqBodystrut)
	if err != nil {
		log.Debugf("unmarshal reqBody failed: %v", err)
	} else {
		if len(reqBodystrut.Messages) != 0 {
			lastUserMessage = reqBodystrut.Messages[len(reqBodystrut.Messages)-1].Content
		}
	}

	var respBodystrut chatCompletionResponse
	err = json.Unmarshal(respBody, &respBodystrut)
	if err != nil {
		log.Debugf("unmarshal respBody failed: %v", err)
	} else {
		if len(respBodystrut.Choices) != 0 {
			lastSystemMessage = respBodystrut.Choices[len(respBodystrut.Choices)-1].Message.Content
		}
	}

	if lastUserMessage != "" {
		r.HisMsg = append(r.HisMsg, chatMessage{
			Role:    "user",
			Content: lastUserMessage,
		})
	}

	if lastSystemMessage != "" {
		r.HisMsg = append(r.HisMsg, chatMessage{
			Role:    "system",
			Content: lastSystemMessage,
		})
	}
}

func (r *ReplayBuffer) SaveHisStr(log wrapper.Log, errMsg string) {
	r.HisMsg = append(r.HisMsg, chatMessage{
		Role:    "system",
		Content: errMsg,
	})
}

func (c *PluginConfig) ValidateBody(body []byte) error {
	var respJsonStrct chatCompletionResponse
	err := json.Unmarshal(body, &respJsonStrct)
	if err != nil {
		c.rejStruct = RejStruct{ServiceUnavailableCode, "service unavailable: " + string(body)}
		return errors.New(c.rejStruct.RejMsg)
	}
	content := gjson.ParseBytes(body).Get(c.contentPath)
	if !content.Exists() {
		c.rejStruct = RejStruct{ServiceUnavailableCode, "response body does not contain the content:" + string(body)}
		return errors.New(c.rejStruct.RejMsg)
	}
	return nil
}

func (c *PluginConfig) ValidateJson(body []byte, log wrapper.Log) (string, error) {
	content := gjson.ParseBytes(body).Get(c.contentPath).String()
	// first extract json from response body
	if content == "" {
		log.Infof("response body does not contain the content")
		c.rejStruct = RejStruct{ReturnContentisEmpytCode, "response body does not contain the content"}
		return "", errors.New(c.rejStruct.RejMsg)
	}
	jsonStr, err := c.ExtractJson(content)

	if err != nil {
		log.Infof("response body does not contain the valid json: %v", err)
		c.rejStruct = RejStruct{CannotFindJsonInResponseCode, "response body does not contain the valid json: " + err.Error()}
		return "", errors.New(c.rejStruct.RejMsg)
	}

	if c.jsonSchema != nil {

		// validate the json
		err = c.compile.Validate(strings.NewReader(jsonStr))
		if err != nil {
			log.Infof("response body does not match the Json Schema: %v", err)
			c.rejStruct = RejStruct{ReturnJsonMisMatchSchemaCode, "response body does not match the Json Schema" + err.Error()}
			return "", errors.New(c.rejStruct.RejMsg)
		}
	}
	c.rejStruct = RejStruct{uint32(200), ""}
	return jsonStr, nil
}

func (c PluginConfig) ExtractJson(bodyStr string) (string, error) {
	// simply extract json from response body string
	startIndex := strings.Index(bodyStr, "{")
	endIndex := strings.LastIndex(bodyStr, "}") + 1

	// if not found
	if startIndex == -1 || endIndex == -1 || startIndex >= endIndex {
		return "", errors.New("cannot find json in the response body")
	}

	jsonStr := bodyStr[startIndex:endIndex]

	// attempt to parse the JSON
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func sendResponse(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log, body []byte) {
	log.Infof("Final send: Code %d, Message %s, Body: %s %s", config.rejStruct.RejCode, config.rejStruct.RejMsg, string(body))
	header := [][2]string{
		{"Content-Type", "application/json"},
	}
	if body != nil {
		header = append(header, [2]string{"Content-Disposition", "attachment; filename=\"response.json\""})
	}
	if config.rejStruct.RejCode != uint32(200) {
		proxywasm.SendHttpResponseWithDetail(uint32(500), config.rejStruct.GetShortMsg(), nil, config.rejStruct.Byte(), -1)
	} else {
		proxywasm.SendHttpResponse(uint32(200), header, body, -1)
	}
}

func recursiveRefineJson(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log, retryCount int, bufferRB *ReplayBuffer) {
	// if retry count exceed max retry count, return the response
	if retryCount >= config.maxRetry {
		log.Debugf("retry count exceed max retry count")
		// report more useful error by appending the last of previous error message
		config.rejStruct = RejStruct{ReachMeaxRetryCountCode, "retry count exceed max retry count:" + config.rejStruct.RejMsg}
		sendResponse(ctx, config, log, nil)
		return
	}

	// recursively refine json
	config.serviceClient.Post(bufferRB.url, bufferRB.ReqHeader, bufferRB.assembleReqBody(config),
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			err := config.ValidateBody(responseBody)
			if err != nil {
				sendResponse(ctx, config, log, nil)
				return
			}
			retryCount++
			bufferRB.SaveHisBody(log, bufferRB.assembleReqBody(config), responseBody)
			log.Debugf("[retry request %d/%d] resp code: %d", retryCount, config.maxRetry, statusCode)
			validateJson, err := config.ValidateJson(responseBody, log)
			if err == nil {
				sendResponse(ctx, config, log, []byte(validateJson))
			} else {
				bufferRB.SaveHisStr(log, err.Error())
				recursiveRefineJson(ctx, config, log, retryCount, bufferRB)
			}
		}, uint32(config.serviceTimeout))
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	if config.rejStruct.RejCode != uint32(200) {
		sendResponse(ctx, config, log, nil)
		return types.ActionPause
	}

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

	apiKey, _ := proxywasm.GetHttpRequestHeader("Authorization")
	if apiKey != "" {
		// remove the Authorization header
		proxywasm.RemoveHttpRequestHeader("Authorization")
	}
	if config.apiKey != "" {
		log.Debugf("add Authorization header %s", "Bearer "+config.apiKey)
		header = append(header, [2]string{"Authorization", "Bearer " + config.apiKey})
	}
	ctx.SetContext("headers", header)

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	if ctx.GetContext("isBuffer").(string) == "true" {
		log.Debugf("detect buffer_request, sending request to AI service")
		return types.ActionContinue
	}

	// if there is any error in the config, return the response directly
	if config.rejStruct.RejCode != uint32(200) {
		sendResponse(ctx, config, log, nil)
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
		url = "/v1/chat/completions"
	}

	if config.servicePath != "" {
		log.Debugf("use base url: %s", config.servicePath)
		url = config.servicePath
	}

	header = append(header, [2]string{"isBuffer", "true"})
	bufferRB := &ReplayBuffer{
		url:       url,
		ReqHeader: header,
		ReqBody:   body,
	}

	config.serviceClient.Post(bufferRB.url, bufferRB.ReqHeader, bufferRB.ReqBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			err := config.ValidateBody(responseBody)
			if err != nil {
				sendResponse(ctx, config, log, nil)
				return
			}
			bufferRB.SaveHisBody(log, body, responseBody)
			log.Debugf("[first request] resp code: %d", statusCode)
			validateJson, err := config.ValidateJson(responseBody, log)
			if err == nil {
				sendResponse(ctx, config, log, []byte(validateJson))
				return
			} else {
				retryCount := 0
				bufferRB.SaveHisStr(log, err.Error())
				recursiveRefineJson(ctx, config, log, retryCount, bufferRB)
			}
		}, uint32(config.serviceTimeout))

	return types.ActionPause
}
