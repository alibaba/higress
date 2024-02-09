package main

import (
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"body-transformer",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

type BodyTransformerConfig struct {
	reqTemplate string
	rspTemplate string
}

func parseConfig(json gjson.Result, config *BodyTransformerConfig, log wrapper.Log) error {
	return nil
}

func onHttpResponseBody(ctx wrapper.HttpContext, config BodyTransformerConfig, body []byte, log wrapper.Log) types.Action {
	decoders(formatParse(ctx))
	return types.ActionContinue
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config BodyTransformerConfig, log wrapper.Log) types.Action {
	decoders(formatParse(ctx))
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config BodyTransformerConfig, body []byte, log wrapper.Log) types.Action {
	decoders(formatParse(ctx))
	return types.ActionContinue
}

func formatParse(ctx wrapper.HttpContext) string {
	var f string
	if ctx.Method() == http.MethodGet {
		f = "args"
	}

	cType, _ := proxywasm.GetHttpRequestHeader("Content-Type")
	if len(f) == 0 && len(cType) != 0 {
		if cType == "text/xml" {
			f = "xml"
		} else if cType == "application/json" {
			f = "json"
		} else if cType == "application/x-www-form-urlencoded" {
			f = "encoded"
		}
	}
	return f
}

func decoders(format string) {
	switch format {
	case "json":
	case "xml":
	case "encoded":
	case "args":
	}
}
