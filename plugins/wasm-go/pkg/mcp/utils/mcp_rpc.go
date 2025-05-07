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

package utils

import (
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

func OnMCPResponseSuccess(sendDirectly bool, ctx wrapper.HttpContext, result map[string]any, debugInfo string) {
	OnJsonRpcResponseSuccess(sendDirectly, ctx, result, debugInfo)
	// TODO: support pub to redis when use POST + SSE
}

func OnMCPResponseError(sendDirectly bool, ctx wrapper.HttpContext, err error, code int, debugInfo string) {
	OnJsonRpcResponseError(sendDirectly, ctx, err, code, debugInfo)
	// TODO: support pub to redis when use POST + SSE
}

func OnMCPToolCallSuccess(sendDirectly bool, ctx wrapper.HttpContext, content []map[string]any, debugInfo string) {
	OnMCPResponseSuccess(sendDirectly, ctx, map[string]any{
		"content": content,
		"isError": false,
	}, debugInfo)
}

func OnMCPToolCallError(sendDirectly bool, ctx wrapper.HttpContext, err error, debugInfo ...string) {
	responseDebugInfo := fmt.Sprintf("mcp:tools/call:error(%s)", err)
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}
	OnMCPResponseSuccess(sendDirectly, ctx, map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": err.Error(),
			},
		},
		"isError": true,
	}, responseDebugInfo)
}

func SendMCPToolTextResult(sendDirectly bool, ctx wrapper.HttpContext, result string, debugInfo ...string) {
	responseDebugInfo := "mcp:tools/call::result"
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}
	OnMCPToolCallSuccess(sendDirectly, ctx, []map[string]any{
		{
			"type": "text",
			"text": result,
		},
	}, responseDebugInfo)
}
