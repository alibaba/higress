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
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	name = "proxy-redirect"
)

func main() {
	wrapper.SetCtx(
		name,
		wrapper.ProcessRequestHeadersBy(func(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
			log.Info("try to GetProperty reqeust.xxx-yyy-zzz\n")
			property, err := proxywasm.GetProperty([]string{"request", "xxx-yyy-zzz"})
			if err != nil {
				log.Errorf("failure to GetProperty reqeust.xxx-yyy-zzz: %v\n", err)
			} else {
				log.Infof("success to GetProperty request.xxx-yyy-zzz: %s\n", string(property))
			}
			return types.ActionContinue
		}),
		wrapper.ProcessResponseHeadersBy(func(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
			property, err := proxywasm.GetProperty([]string{"request", "headers"})
			if err != nil {
				log.Errorf("failure to %s, err: %v\n", `proxywasm.GetProperty([]string{"request", "headers"})`, err)
			} else {
				log.Infof("success to proxywasm.GetProperty(request.headers): %s\n", string(property))
			}
			return types.ActionContinue
		}),
	)
}

type Config struct {
}
