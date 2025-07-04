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
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"

	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"streaming-body-example",
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessStreamingRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpResponseBody),
	)
}

type Config struct {
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log log.Log) types.Action {
	proxywasm.RemoveHttpRequestHeader("content-length")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config Config, chunk []byte, isLastChunk bool, log log.Log) []byte {
	log.Infof("receive request body chunk:%s, isLastChunk:%v", chunk, isLastChunk)
	return []byte("test\n")
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config Config, log log.Log) types.Action {
	proxywasm.RemoveHttpResponseHeader("content-length")
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config Config, chunk []byte, isLastChunk bool, log log.Log) []byte {
	log.Infof("receive response body chunk:%s, isLastChunk:%v", chunk, isLastChunk)
	return []byte("test\n")
}
