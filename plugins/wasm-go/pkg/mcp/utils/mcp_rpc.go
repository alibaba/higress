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

import "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"

func OnMCPResponseSuccess(ctx wrapper.HttpContext, result map[string]any) {
	OnJsonRpcResponseSuccess(ctx, result)
	// TODO: support pub to redis when use POST + SSE
}

func OnMCPResponseError(ctx wrapper.HttpContext, err error, code ...int) {
	OnJsonRpcResponseError(ctx, err, code...)
	// TODO: support pub to redis when use POST + SSE
}

func OnMCPToolCallSuccess(ctx wrapper.HttpContext, content []map[string]any) {
	OnMCPResponseSuccess(ctx, map[string]any{
		"content": content,
		"isError": false,
	})
}

func OnMCPToolCallError(ctx wrapper.HttpContext, err error) {
	OnMCPResponseSuccess(ctx, map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": err.Error(),
			},
		},
		"isError": true,
	})
}

func SendMCPToolTextResult(ctx wrapper.HttpContext, result string) {
	OnMCPToolCallSuccess(ctx, []map[string]any{
		{
			"type": "text",
			"text": result,
		},
	})
}
