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

	"github.com/alibaba/higress/plugins/wasm-go/extensions/hello-world/templates"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	CacheKeyContextKey       = "cacheKey"
	CacheContentContextKey   = "cacheContent"
	ToolCallsContextKey      = "toolCalls"
	StreamContextKey         = "stream"
	PartialMessageContextKey = "partialMessage"
)

type PluginConfig struct {
	// @Title zh-CN: 自定义JsonSchema
	// @Description zh-CN: 自定义JsonSchema
	CustomJsonSchema map[string]interface{} `required:"false" yaml:"customJsonSchema" json:"customJsonSchema"`
	// @Title zh-CN: JsonSchema编译器
	// @Description zh-CN: JsonSchema编译器
	draft *jsonschema.Draft
	// @Title zh-CN: 支持服务的模型
	// @Description zh-CN: 支持服务的模型，用于传递到后端AI服务
	Model string `required:"false" yaml:"model" json:"model"`
	// @Title zh-CN: 是否启用Swagger
	// @Description zh-CN: 是否启用Swagger来验证上传的案例
	EnableSwagger bool `required:"false" yaml:"enable_swagger" json:"enable_swagger"`
	// @Title zh-CN: 是否启用OAS3
	// @Description zh-CN: 是否启用OAS3来验证上传的案例
	EnableOas3 bool `required:"false" yaml:"enable_oas3" json:"enable_oas3"`
}

type RequestInfom struct {
	Desc       string `json:"desc"`
	Case       string `json:"case"`
	Type       string `json:"type"`
	JsonSchema string `json:"jsonSchema"`
}

func main() {
	wrapper.SetCtx(
		"ai-struct-gen",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessStreamingResponseBodyBy(onHttpResponseBody),
	)
}

func askJson(log wrapper.Log, config PluginConfig, rinfo RequestInfom) chatCompletionRequest {
	var request chatCompletionRequest

	// Initialize content with the description from rinfo
	content := rinfo.Desc

	// Append example case if it's provided
	if rinfo.Case != "" {
		content += " Given an example case: " + rinfo.Case
	}

	// Append JSON schema if it's provided
	if rinfo.JsonSchema != "" {
		content += " Given a example JSON schema: " + rinfo.JsonSchema
	}
	messages := []chatMessage{
		{
			Role:    "system",
			Content: "I am writing a API document, please generate the a JSON for this API descripted later for me and provide descrption",
		},
		{
			Role:    "user",
			Content: content,
		},
	}
	request.Model = config.Model
	request.Messages = messages
	if request.ResponseFormat == nil {
		request.ResponseFormat = string2JsonObj(templates.AskJsonTemp, log)
	}
	return request

}

func askVerify(log wrapper.Log, config PluginConfig, rinfo RequestInfom) chatCompletionRequest {
	var request chatCompletionRequest
	messages := []chatMessage{
		{
			Role:    "system",
			Content: "I am validating a JSON case, please help me verify the JSON case based on the JSON schema",
		},
		{
			Role:    "user",
			Content: "Given the case" + rinfo.Case + " and the JSON schema" + rinfo.JsonSchema + ", they are not matched, please tell me the reason and how to fix it",
		},
	}
	request.Model = config.Model
	request.Messages = messages
	if request.ResponseFormat == nil {
		request.ResponseFormat = string2JsonObj(templates.AskVerifyTemp, log)
	}

	return request
}

func askJsonSchema(log wrapper.Log, config PluginConfig, rinfo RequestInfom) chatCompletionRequest {
	var request chatCompletionRequest
	// Initialize content with the description from rinfo
	content := rinfo.Desc

	// Append example case if it's provided
	if rinfo.Case != "" {
		content += " Given an example case: " + rinfo.Case
	}

	// Append JSON schema if it's provided
	if rinfo.JsonSchema != "" {
		content += " Given a example JSON schema: " + rinfo.JsonSchema
	}
	messages := []chatMessage{
		{
			Role:    "system",
			Content: "I am writing a API document, please generate the a JSON Schema for this API descripted later for me according to a JSON case",
		},
		{
			Role:    "user",
			Content: content,
		},
	}
	request.Model = config.Model
	request.Messages = messages
	if request.ResponseFormat == nil {
		request.ResponseFormat = string2JsonObj(templates.AskJsonSchemaTemp, log)
	}

	return request
}

func parseConfig(json gjson.Result, config *PluginConfig, log wrapper.Log) error {
	config.Model = json.Get("model").String()
	if config.Model == "" {
		config.Model = "gpt-4o-2024-08-06"
	}
	if schemaValue, ok := json.Get("responseJsonSchema").Value().(map[string]interface{}); ok {
		config.CustomJsonSchema = schemaValue
	} else {
		config.CustomJsonSchema = nil
	}

	config.EnableSwagger = json.Get("enable_swagger").Bool()
	config.EnableOas3 = json.Get("enable_oas3").Bool()

	// set draft version ref: request-validation/main.go
	if config.EnableSwagger {
		config.draft = jsonschema.Draft4
	}
	if config.EnableOas3 {
		config.draft = jsonschema.Draft7
	}
	if !config.EnableOas3 && !config.EnableSwagger {
		config.draft = jsonschema.Draft7
	}
	return nil
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	var adjustBody chatCompletionRequest
	adjustBody.Stream = false

	rinfo := RequestInfom{}
	err := json.Unmarshal(body, &rinfo)
	if err != nil {
		proxywasm.SendHttpResponse(http.StatusBadRequest, nil, []byte("{\"reason\": \"failed to unmarshal request body\"}"), -1)
	}

	if rinfo.Type != "val" {
		// default to gen json/jsonSchema
		if rinfo.Case != "" {
			adjustBody = askJsonSchema(log, config, rinfo)
		} else {
			adjustBody = askJson(log, config, rinfo)
		}
	} else {
		// Check if both Case and JsonSchema are provided
		if rinfo.Case == "" || rinfo.JsonSchema == "" {
			proxywasm.SendHttpResponse(http.StatusBadRequest, nil, []byte("{\"reason\": \"case and jsonSchema are required for validation\"}"), -1)
			return types.ActionContinue
		}
		// Compile the JSON Schema
		comiler := jsonschema.NewCompiler()
		comiler.Draft = config.draft
		err := comiler.AddResource("customJsonSchema", strings.NewReader(rinfo.JsonSchema))
		if err != nil {
			proxywasm.SendHttpResponse(http.StatusBadRequest, nil, []byte("{\"reason\": \"failed to compile json schema, please check the json schema you provided\"}"), -1)
			return types.ActionContinue
		}

		// Validate the Case against the JSON Schema
		comile, err := comiler.Compile("customJsonSchema")
		if err != nil {
			proxywasm.SendHttpResponse(http.StatusBadRequest, nil, []byte("{\"reason\": \"failed to compile json schema, please check the json schema you provided\"}"), -1)
			return types.ActionContinue
		}
		err = comile.Validate(strings.NewReader(rinfo.Case))
		if err == nil {
			proxywasm.SendHttpResponse(http.StatusOK, nil, []byte("{\"reason\": \"case is valid\"}"), -1)
			return types.ActionContinue
		}
		adjustBody = askVerify(log, config, rinfo)
	}

	replaceJsonRequestBody(adjustBody, log)
	proxywasm.ResumeHttpRequest()
	return types.ActionPause
}

func onHttpResponseBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, isLastChunk bool, log wrapper.Log) []byte {
	// TODO: support streaming response body
	if len(body) == 0 {
		log.Infof("Received empty chunk")
		return body
	}

	// Attempt to parse JSON and extract the content
	content := gjson.Get(string(body), "choices.0.message.content").String()

	if content == "" {
		log.Infof("Failed to extract content from response chunk: %s", string(body))
		return body
	}

	// Return the extracted content
	return []byte(content)
}
