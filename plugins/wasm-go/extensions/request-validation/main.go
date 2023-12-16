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
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/request-validation/validation"
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
	HeaderSchema Schema
	// BodySchema is the schema for request body.
	BodySchema Schema
	// RejectedCode is the code for rejected request.
	RejectedCode uint32
	// RejectedMsg is the message for rejected request.
	RejectedMsg string
}

// Schema is the schema for request header or body.
type Schema struct {
	// Traffic is the traffic for request header or body.
	Traffic map[string]interface{}
}

// Traffic is the traffic for request header or body.
type Traffic interface {
	// Validation is the validation for request header or body.
	Validation(schema map[string]interface{}, paramName string) error
}

func (s *Schema) GetTraffics() (map[string]Traffic, error) {
	trafficMap := make(map[string]Traffic)
	for paramName, traffic := range s.Traffic {
		switch traffic.(type) {
		case validation.EnumValidation:
			trafficMap[paramName] = traffic.(validation.EnumValidation)
		case validation.IntRangeValidation:
			trafficMap[paramName] = traffic.(validation.IntRangeValidation)
		case validation.StringLengthValidation:
			trafficMap[paramName] = traffic.(validation.StringLengthValidation)
		case validation.RegexValidation:
			trafficMap[paramName] = traffic.(validation.RegexValidation)
		case validation.ArrayValidation:
			trafficMap[paramName] = traffic.(validation.ArrayValidation)
		default:
			return nil, fmt.Errorf("unknown traffic type %s", paramName)
		}
	}
	return trafficMap, nil
}

func parseConfig(result gjson.Result, config *Config, log wrapper.Log) error {
	headerSchema := result.Get("header_schema").String()
	bodySchema := result.Get("body_schema").String()
	if headerSchema == "" && bodySchema == "" {
		return nil
	}
	if headerSchema != "" {
		headerSchemaMap := make(map[string]interface{})
		err := json.Unmarshal([]byte(headerSchema), &headerSchemaMap)
		if err != nil {
			return err
		}
		config.HeaderSchema = Schema{Traffic: headerSchemaMap}
	}
	if bodySchema != "" {
		bodySchemaMap := make(map[string]interface{})
		err := json.Unmarshal([]byte(bodySchema), &bodySchemaMap)
		if err != nil {
			return err
		}
		config.BodySchema = Schema{Traffic: bodySchemaMap}
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
	traffics, err := config.HeaderSchema.GetTraffics()
	if err != nil {
		log.Errorf("get header traffics failed: %v", err)
		return types.ActionContinue
	}
	for paramName, traffic := range traffics {
		err := traffic.Validation(schema, paramName)
		if err != nil {
			log.Errorf("validate header %s failed: %v", paramName, err)
			proxywasm.SendHttpResponse(config.RejectedCode, nil, []byte(config.RejectedMsg), -1)
			return types.ActionPause
		}
	}
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config Config, body []byte, log wrapper.Log) types.Action {
	// covert to schema
	schema := make(map[string]interface{})
	err := json.Unmarshal(body, &schema)
	if err != nil {
		log.Errorf("unmarshal request body failed: %v", err)
		return types.ActionContinue
	}
	// validate
	traffics, err := config.BodySchema.GetTraffics()
	if err != nil {
		log.Errorf("get body traffics failed: %v", err)
		return types.ActionContinue
	}
	for paramName, traffic := range traffics {
		err := traffic.Validation(schema, paramName)
		if err != nil {
			log.Errorf("validate body %s failed: %v", paramName, err)
			proxywasm.SendHttpResponse(config.RejectedCode, nil, []byte(config.RejectedMsg), -1)
			return types.ActionPause
		}
	}
	return types.ActionContinue
}
