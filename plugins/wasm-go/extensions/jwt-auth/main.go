// Copyright (c) 2023 Alibaba Group Holding Ltd.
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
	"github.com/alibaba/higress/plugins/wasm-go/extensions/jwt-auth/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/jwt-auth/handler"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

// @Name jwt-proxy
// @Category auth
// @Phase UNSPECIFIED_PHASE
// @Priority 0
// @Title zh-CN jwt验证
// @Description zh-CN 通过jwt进行验证
// @Version 0.1.0
//
// @Contact.name Ink33
// @Contact.url https://github.com/Ink-33
// @Contact.email ink33@smlk.org
//
// @Example
//{}
// @End

func main() {
	wrapper.SetCtx(
		// 插件名称
		"jwt-auth",
		// 为解析插件配置，设置自定义函数
		wrapper.ParseConfigBy(config.ParseGlobalConfig),
		wrapper.ParseOverrideConfigBy(config.ParseGlobalConfig, config.ParseRuleConfig),
		// 为处理请求头，设置自定义函数
		wrapper.ProcessRequestHeadersBy(handler.OnHTTPRequestHeaders),
	)
}
