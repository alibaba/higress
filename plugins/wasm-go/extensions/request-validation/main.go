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
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"request-validation",
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ParseConfigBy(parseConfig),
	)
}

// Config is the config for request validation.
type Config struct {
	// HeaderSchema is the schema for request header.
	//HeaderSchema gojsonschema.JSONLoader
	// BodySchema is the schema for request body.
	//BodySchema gojsonschema.JSONLoader
	// RejectedCode is the code for rejected request.
	RejectedCode uint32
	// RejectedMsg is the message for rejected request.
	RejectedMsg string
}

func parseConfig(result gjson.Result, config *Config, log wrapper.Log) error {
	headerSchema := result.Get("header_schema").String()
	bodySchema := result.Get("body_schema").String()

	log.Infof("header_schema: %s", headerSchema)

	if headerSchema == "" && bodySchema == "" {
		return nil
	}
	if headerSchema != "" {
		//config.HeaderSchema = gojsonschema.NewStringLoader(headerSchema)
	}
	if bodySchema != "" {
		//config.BodySchema = gojsonschema.NewStringLoader(bodySchema)
	}

	// check rejected_code is valid
	code := result.Get("rejected_code").Int()
	if code != 0 && code > 100 && code < 600 {
		config.RejectedCode = uint32(code)
	} else {
		config.RejectedCode = 403
	}
	config.RejectedMsg = result.Get("rejected_msg").String()
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	//if config.HeaderSchema == nil {
	//	log.Infof("header_schema is nil")
	//	return types.ActionContinue
	//}

	log.Infof("Config: ", config)

	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("get request headers failed: %v", err)
		return types.ActionContinue
	}
	// covert to schema
	schema := make(map[string]interface{})
	for _, header := range headers {
		schema[header[0]] = header[1]
	}

	// validate
	//requestSchema := gojsonschema.NewGoLoader(schema)
	////validate, err := gojsonschema.Validate(config.HeaderSchema, requestSchema)
	//if err != nil {
	//	log.Errorf("validate request headers failed: %v", err)
	//	return types.ActionContinue
	//}
	//if !validate.Valid() {
	//	log.Errorf("validate request headers failed: %v", validate.Errors())
	//	proxywasm.SendHttpResponse(config.RejectedCode, nil, []byte(config.RejectedMsg), -1)
	//	return types.ActionPause
	//}

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config Config, body []byte, log wrapper.Log) types.Action {
	//if config.BodySchema == nil {
	//	log.Infof("body_schema is nil")
	//	return types.ActionContinue
	//}

	// covert to schema
	schema := make(map[string]interface{})
	err := json.Unmarshal(body, &schema)
	if err != nil {
		log.Errorf("unmarshal request body failed: %v", err)
		return types.ActionContinue
	}

	// validate
	//requestSchema := gojsonschema.NewGoLoader(schema)
	//validate, err := gojsonschema.Validate(config.BodySchema, requestSchema)
	//if err != nil {
	//	log.Errorf("validate request body failed: %v", err)
	//	return types.ActionContinue
	//}
	//if !validate.Valid() {
	//	log.Errorf("validate request body failed: %v", validate.Errors())
	//	proxywasm.SendHttpResponse(config.RejectedCode, nil, []byte(config.RejectedMsg), -1)
	//	return types.ActionPause
	//}
	//
	//log.Errorf("passed request-validation")

	return types.ActionContinue
}
