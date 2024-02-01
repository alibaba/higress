package main

import (
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	wrapper.SetCtx(
		"cache-control",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
	)
}

type CacheControlConfig struct {
	suffix []string
	maxAge int64
}

func parseConfig(json gjson.Result, config *CacheControlConfig, log wrapper.Log) error {
	suffix := json.Get("suffix").String()
	parts := strings.Split(suffix, "|")
	config.suffix = parts

	config.maxAge = json.Get("maxAge").Int()

	log.Infof("suffix: %q, maxAge: %v", config.suffix, config.maxAge)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config CacheControlConfig, log wrapper.Log) types.Action {
	path := ctx.Path()
	if strings.Contains(path, "?") {
		path = strings.Split(path, "?")[0]
	}
	ctx.SetContext("path", path)
	log.Debugf("path: %s", path)

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config CacheControlConfig, log wrapper.Log) types.Action {
	if len(config.suffix) > 0 {
		path, ok := ctx.GetContext("path").(string)
		if !ok {
			log.Error("failed to get request path")
			return types.ActionContinue
		}

		hit := false
		for _, part := range config.suffix {
			if strings.HasSuffix(path, "."+part) {
				hit = true
				break
			}
		}

		if hit {
			currentTime := time.Now()
			expireTime := currentTime.Add(time.Duration(config.maxAge) * time.Second)
			proxywasm.AddHttpResponseHeader("Expires", expireTime.UTC().Format(http.TimeFormat))
			proxywasm.AddHttpResponseHeader("Cache-Control", "maxAge="+strconv.FormatInt(config.maxAge, 10))
		}
	}
	return types.ActionContinue
}
