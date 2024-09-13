package main

import (
	"fmt"
	"net/http"
	"runtime"

	. "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
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
	log.Infof("alloc success, point address: %p", b)
	memstats := fmt.Sprintf(`{"Sys": %d,"HeapSys": %d,"HeapIdle": %d,"HeapInuse": %d,"HeapReleased": %d}`, m.Sys, m.HeapSys, m.HeapIdle, m.HeapInuse, m.HeapReleased)
	log.Info(memstats)
	_ = proxywasm.SendHttpResponseWithDetail(http.StatusOK, "gc-test", [][2]string{{"Content-Type", "application/json"}}, []byte(memstats), -1)
	return types.ActionContinue
}
