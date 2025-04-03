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
	"fmt"
	"strconv"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"custom-response",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
	)
}

type CustomResponseConfig struct {
	rules                 []CustomResponseRule
	enableOnStatusRuleMap map[uint32]*CustomResponseRule
}

type CustomResponseRule struct {
	statusCode     uint32
	headers        [][2]string
	body           string
	enableOnStatus []uint32
	contentType    string
}

func parseConfig(gjson gjson.Result, config *CustomResponseConfig) error {
	rules := gjson.Get("rules")
	rulesVersion := rules.Exists() && rules.IsArray()
	if rulesVersion {
		for _, cf := range gjson.Get("rules").Array() {
			item := new(CustomResponseRule)
			if err := parseRuleItem(cf, item); err != nil {
				return err
			}
			config.rules = append(config.rules, *item)
		}

	} else {
		rule := new(CustomResponseRule)
		if err := parseRuleItem(gjson, rule); err != nil {
			return err
		}
		config.rules = append(config.rules, *rule)
	}
	config.enableOnStatusRuleMap = make(map[uint32]*CustomResponseRule)
	for i, configItem := range config.rules {
		for _, statusCode := range configItem.enableOnStatus {
			if v, ok := config.enableOnStatusRuleMap[statusCode]; ok {
				log.Errorf("enable_on_status code used in %v, want to add %v", v, configItem.statusCode)
				return errors.New("enableOnStatus can only use once")
			}
			config.enableOnStatusRuleMap[statusCode] = &config.rules[i]
		}
	}
	if rulesVersion && len(config.enableOnStatusRuleMap) == 0 {
		return errors.New("enableOnStatus is required")
	}
	return nil
}

func parseRuleItem(gjson gjson.Result, rule *CustomResponseRule) error {
	headersArray := gjson.Get("headers").Array()
	rule.headers = make([][2]string, 0, len(headersArray))
	for _, v := range headersArray {
		kv := strings.SplitN(v.String(), "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			if strings.EqualFold(key, "content-type") {
				rule.contentType = value
			} else if strings.EqualFold(key, "content-length") {
				continue
			} else {
				rule.headers = append(rule.headers, [2]string{key, value})
			}
		} else {
			return fmt.Errorf("invalid header pair format: %s", v.String())
		}
	}

	rule.body = gjson.Get("body").String()
	if rule.contentType == "" && rule.body != "" {
		if json.Valid([]byte(rule.body)) {
			rule.contentType = "application/json; charset=utf-8"
		} else {
			rule.contentType = "text/plain; charset=utf-8"
		}
	}
	rule.headers = append(rule.headers, [2]string{"content-type", rule.contentType})

	rule.statusCode = 200
	if gjson.Get("status_code").Exists() {
		statusCode := gjson.Get("status_code")
		parsedStatusCode, err := strconv.Atoi(statusCode.String())
		if err != nil {
			return fmt.Errorf("invalid status code value: %s", statusCode.String())
		}
		rule.statusCode = uint32(parsedStatusCode)
	}

	enableOnStatusArray := gjson.Get("enable_on_status").Array()
	rule.enableOnStatus = make([]uint32, 0, len(enableOnStatusArray))
	for _, v := range enableOnStatusArray {
		parsedEnableOnStatus, err := strconv.Atoi(v.String())
		if err != nil {
			return fmt.Errorf("invalid enable_on_status value: %s", v.String())
		}
		rule.enableOnStatus = append(rule.enableOnStatus, uint32(parsedEnableOnStatus))
	}
	return nil
}

func onHttpRequestHeaders(_ wrapper.HttpContext, config CustomResponseConfig) types.Action {
	if len(config.enableOnStatusRuleMap) != 0 {
		return types.ActionContinue
	}
	err := proxywasm.SendHttpResponseWithDetail(config.rules[0].statusCode, "custom-response", config.rules[0].headers, []byte(config.rules[0].body), -1)
	if err != nil {
		log.Errorf("send http response failed: %v", err)
	}

	return types.ActionPause
}

func onHttpResponseHeaders(_ wrapper.HttpContext, config CustomResponseConfig) types.Action {
	// enableOnStatusRuleMap is not empty, compare the status code.
	// if match the status code, mock the response.
	statusCodeStr, err := proxywasm.GetHttpResponseHeader(":status")
	if err != nil {
		log.Errorf("get http response status code failed: %v", err)
		return types.ActionContinue
	}
	statusCode, err := strconv.ParseUint(statusCodeStr, 10, 32)
	if err != nil {
		log.Errorf("parse http response status code failed: %v", err)
		return types.ActionContinue
	}
	if rule, ok := config.enableOnStatusRuleMap[uint32(statusCode)]; ok {
		err = proxywasm.SendHttpResponseWithDetail(rule.statusCode, "custom-response", rule.headers, []byte(rule.body), -1)
		if err != nil {
			log.Errorf("send http response failed: %v", err)
		}
		return types.ActionContinue
	}
	return types.ActionContinue
}
