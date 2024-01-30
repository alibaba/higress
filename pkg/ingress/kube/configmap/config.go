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

package configmap

import (
	"encoding/json"
)

type Result int32

const (
	ResultNothing Result = iota
	ResultReplace
	ResultDelete

	HigressConfigMapName = "higress-config"
	HigressConfigMapKey  = "higress"

	ModelUpdatedReason = "higress configmap updated"
)

type ItemEventHandler = func(name string)

type HigressConfig struct {
	Tracing              *Tracing    `json:"tracing,omitempty"`
	Gzip                 *Gzip       `json:"gzip,omitempty"`
	Downstream           *Downstream `json:"downstream,omitempty"`
	Upstream             *Upstream   `json:"upstream,omitempty"`
	DisableXEnvoyHeaders bool        `json:"disableXEnvoyHeaders,omitempty"`
	AddXRealIpHeader     bool        `json:"addXRealIpHeader,omitempty"`
}

func NewDefaultHigressConfig() *HigressConfig {
	globalOption := NewDefaultGlobalOption()
	higressConfig := &HigressConfig{
		Tracing:              NewDefaultTracing(),
		Gzip:                 NewDefaultGzip(),
		Downstream:           globalOption.Downstream,
		Upstream:             globalOption.Upstream,
		DisableXEnvoyHeaders: globalOption.DisableXEnvoyHeaders,
		AddXRealIpHeader:     globalOption.AddXRealIpHeader,
	}
	return higressConfig
}

func GetHigressConfigString(higressConfig *HigressConfig) string {
	bytes, _ := json.Marshal(higressConfig)
	return string(bytes)
}
