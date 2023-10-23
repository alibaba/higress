package main

import (
	"fmt"
	"runtime"

	. "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	SetCtx(
		"gc-test",
		ParseConfigBy(parseConfig),
		ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type MyConfig struct {
	bytes uint64
}

func parseConfig(json gjson.Result, config *MyConfig, log Log) error {
	config.bytes = json.Get("bytes").Uint()
	return nil
}

func onHttpRequestHeaders(ctx HttpContext, config MyConfig, log Log) types.Action {
	b := make([]byte, int(config.bytes))
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Infof("MemStats Sys:%d, HeapSys:%d, HeapIdle:%d, HeapInuse:%d, HeapReleased:%d", m.Sys, m.HeapSys, m.HeapIdle, m.HeapInuse, m.HeapReleased)
	info := fmt.Sprintf("alloc success, point address: %p", b)
	proxywasm.SendHttpResponse(200, nil, []byte(info), -1)
	return types.ActionContinue
}
