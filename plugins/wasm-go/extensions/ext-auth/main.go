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
	"strconv"
)

func main() {
	wrapper.SetCtx(
		"ext-auth",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

const (
	HeaderContentLength    string = "content-length"
	HeaderAuthorization    string = "authorization"
	HeaderFailureModeAllow string = "x-envoy-auth-failure-mode-allowed"
)

func onHttpRequestHeaders(ctx wrapper.HttpContext, config ExtAuthConfig, log wrapper.Log) types.Action {
	contentLengthStr, _ := proxywasm.GetHttpRequestHeader(HeaderContentLength)
	hasRequestBody := false
	if contentLengthStr != "" {
		contentLength, err := strconv.Atoi(contentLengthStr)
		hasRequestBody = err == nil && contentLength > 0
	}
	// If withRequestBody is true AND the HTTP request contains a request body,
	// it will be handled in the onHttpRequestBody phase.
	if config.httpService.authorizationRequest.withRequestBody && hasRequestBody {
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
	if config.httpService.authorizationRequest.withRequestBody {
		return checkExtAuth(ctx, config, body, log)
	}
	return types.ActionContinue
}

func checkExtAuth(ctx wrapper.HttpContext, config ExtAuthConfig, body []byte, log wrapper.Log) types.Action {
	// build extAuth request headers
	extAuthReqHeaders := http.Header{}

	httpServiceConfig := config.httpService
	requestConfig := httpServiceConfig.authorizationRequest
	reqHeaders, _ := proxywasm.GetHttpRequestHeaders()
	if requestConfig.allowedHeaders != nil {
		for _, header := range reqHeaders {
			headK := header[0]
			if requestConfig.allowedHeaders.Match(headK) {
				extAuthReqHeaders.Set(headK, header[1])
			}
		}
	}

	for key, value := range requestConfig.headersToAdd {
		extAuthReqHeaders.Set(key, value)
	}

	// add Authorization header
	authorization := extractFromHeader(reqHeaders, HeaderAuthorization)
	if authorization != "" {
		extAuthReqHeaders.Set(HeaderAuthorization, authorization)
	}

	// call ext auth server
	err := httpServiceConfig.client.Post(httpServiceConfig.path, reconvertHeaders(extAuthReqHeaders), body,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			defer proxywasm.ResumeHttpRequest()
			if statusCode != http.StatusOK {
				log.Errorf("failed to call ext auth server, status: %d", statusCode)
				callExtAuthServerErrorHandler(config, statusCode, responseHeaders)
				return
			}

			if httpServiceConfig.authorizationResponse.allowedUpstreamHeaders != nil {
				for headK, headV := range responseHeaders {
					if httpServiceConfig.authorizationResponse.allowedUpstreamHeaders.Match(headK) {
						_ = proxywasm.ReplaceHttpRequestHeader(headK, headV[0])
					}
				}
			}

		}, httpServiceConfig.timeout)

	if err != nil {
		log.Errorf("failed to call ext auth server: %v", err)
		// Since the handling logic for call errors and HTTP status code 500 is the same, we directly use 500 here.
		callExtAuthServerErrorHandler(config, http.StatusInternalServerError, nil)
		return types.ActionContinue
	}
	return types.ActionPause
}

func callExtAuthServerErrorHandler(config ExtAuthConfig, statusCode int, extAuthRespHeaders http.Header) {
	if statusCode >= http.StatusInternalServerError && config.failureModeAllow {
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

	// Rejects client requests with statusOnError on extAuth unavailability or 5xx.
	// Otherwise, uses the extAuth's returned status code to reject requests.
	statusToUse := statusCode
	if statusCode >= http.StatusInternalServerError {
		statusToUse = int(config.statusOnError)
	}
	_ = sendResponse(uint32(statusToUse), "ext-auth.unauthorized", respHeaders)
}
