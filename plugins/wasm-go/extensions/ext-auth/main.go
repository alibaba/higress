// Copyright (c) 2024 Alibaba Group Holding Ltd.
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
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"net/http"
)

func main() {
	wrapper.SetCtx(
		"ext-auth",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config ExtAuthConfig, log wrapper.Log) types.Action {
	if config.withRequestBody {
		// Disable the route re-calculation since the plugin may modify some headers related to the chosen route.
		ctx.DisableReroute()
		// The request has a body and requires delaying the header transmission until a cache miss occurs,
		// at which point the header should be sent.
		return types.HeaderStopIteration
	}
	ctx.DontReadRequestBody()
	return checkExtAuth(ctx, config, nil, log)
}

func onHttpRequestBody(ctx wrapper.HttpContext, config ExtAuthConfig, body []byte, log wrapper.Log) types.Action {
	if config.withRequestBody {
		return checkExtAuth(ctx, config, body, log)
	}
	return types.ActionContinue
}

const (
	HeaderAuthorization    string = "authorization"
	HeaderFailureModeAllow string = "x-envoy-auth-failure-mode-allowed"
)

func checkExtAuth(ctx wrapper.HttpContext, config ExtAuthConfig, body []byte, log wrapper.Log) types.Action {
	// build extAuth request headers
	extAuthReqHeaders := http.Header{}

	headers, _ := proxywasm.GetHttpRequestHeaders()
	if config.allowedHeaders != nil {
		for _, header := range headers {
			key := header[0]
			if config.allowedHeaders.Match(key) {
				extAuthReqHeaders.Set(key, header[1])
			}
		}
	}

	for key, value := range config.httpService.authorizationRequest.headersToAdd {
		extAuthReqHeaders.Set(key, value)
	}

	// add Authorization header
	authorization := extractFromHeader(headers, HeaderAuthorization)
	if authorization != "" {
		extAuthReqHeaders.Set(HeaderAuthorization, authorization)
	}

	// call ext auth server
	err := config.httpService.client.Do(ctx.Method(), config.httpService.path, reconvertHeaders(extAuthReqHeaders), body,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			defer proxywasm.ResumeHttpRequest()

			if statusCode != http.StatusOK {
				log.Warnf("failed to call ext auth server, status: %d", statusCode)
				callExtAuthServerErrorHandler(config, statusCode, responseHeaders)
				return
			}

			if config.httpService.authorizationResponse.allowedUpstreamHeaders != nil {
				for headK, headV := range responseHeaders {
					if config.httpService.authorizationResponse.allowedUpstreamHeaders.Match(headK) {
						_ = proxywasm.ReplaceHttpRequestHeader(headK, headV[0])
					}
				}
			}

		}, config.httpService.timeout)

	if err != nil {
		log.Warnf("failed to call ext auth server: %v", err)
		// Since the handling logic for call errors and HTTP status code 500 is the same, we directly use 500 here.
		callExtAuthServerErrorHandler(config, 500, nil)
		return types.ActionContinue
	}
	return types.ActionPause
}

func callExtAuthServerErrorHandler(config ExtAuthConfig, statusCode int, extAuthRespHeaders http.Header) {
	if statusCode >= 500 && config.failureModeAllow {
		if config.failureModeAllowHeaderAdd {
			_ = proxywasm.ReplaceHttpRequestHeader(HeaderFailureModeAllow, "true")
		}
		return
	}
	var respHeaders = extAuthRespHeaders
	if config.httpService.authorizationResponse.allowedClientHeaders != nil {
		respHeaders = http.Header{}
		for headK, headV := range extAuthRespHeaders {
			if config.httpService.authorizationResponse.allowedClientHeaders.Match(headK) {
				respHeaders.Set(headK, headV[0])
			}
		}
	}
	_ = sendResponse(config.statusOnError, respHeaders)
}
