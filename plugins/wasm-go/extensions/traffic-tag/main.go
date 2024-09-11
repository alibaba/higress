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
	"math/rand"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

const (
	PluginName      = "traffic-tag"
	ConditionGroups = "conditionGroups"
	WeightGroups    = "weightGroups"
	HeaderName      = "headerName"
	HeaderValue     = "headerValue"
	Conditions      = "conditions"
	MatchLogic      = "logic"
	CondKeyType     = "conditionType"
	CondKey         = "key"
	CondMatchType   = "operator"
	CondValue       = "value"
	Weight          = "weight"
)

const (
	DefaultTagKey  = "defaultTagKey"
	DefaultTagVal  = "defaultTagVal"
	Type_Content   = "content"
	Type_Weight    = "weight"
	Type_Header    = "header"
	Type_Cookie    = "cookie"
	Type_Parameter = "parameter"
	Op_Prefix      = "prefix"
	Op_Equal       = "equal"
	Op_NotEqual    = "not_equal"
	Op_Regex       = "regex"
	Op_In          = "in"
	Op_NotIn       = "not_in"
	Op_Percent     = "percentage"
	TotalWeight    = 100
)

type TrafficTagConfig struct {
	ConditionGroups []ConditionGroup `json:"conditionGroups,omitempty"`
	WeightGroups    []WeightGroup    `json:"weightGroups,omitempty"`
	DefaultTagKey   string           `json:"defaultTagKey,omitempty"`
	DefaultTagVal   string           `json:"defaultTagVal,omitempty"`
	randGen         *rand.Rand
}

type ConditionGroup struct {
	HeaderName  string          `json:"headerName"`
	HeaderValue string          `json:"headerValue"`
	Logic       string          `json:"logic"`
	Conditions  []ConditionRule `json:"conditions"`
}

type ConditionRule struct {
	ConditionType string   `json:"conditionType"`
	Key           string   `json:"key"`
	Operator      string   `json:"operator"`
	Value         []string `json:"value"`
}

type WeightGroup struct {
	HeaderName  string `json:"headerName"`
	HeaderValue string `json:"headerValue"`
	Weight      int64  `json:"weight"`
	Accumulate  int64
}

func main() {
	wrapper.SetCtx(
		PluginName,
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

func parseConfig(json gjson.Result, config *TrafficTagConfig, log wrapper.Log) error {
	if err := jsonValidate(json, log); err != nil {
		return err
	}

	err := parseContentConfig(json, config, log)
	if err != nil {
		return err
	}

	return parseWeightConfig(json, config, log)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config TrafficTagConfig, log wrapper.Log) types.Action {
	if add := (onContentRequestHeaders(config.ConditionGroups, log) || onWeightRequestHeaders(config.WeightGroups, config.randGen, log)); !add {
		setDefaultTag(config.DefaultTagKey, config.DefaultTagVal, log)
	}

	return types.ActionContinue
}
