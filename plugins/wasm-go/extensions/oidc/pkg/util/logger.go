package util

import (
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

var Logger *wrapper.Log

func SendError(errMsg string, rw http.ResponseWriter, status int) {
	Logger.Errorf(errMsg)
	if rw != nil {
		rw.WriteHeader(status)
	}
	proxywasm.SendHttpResponse(uint32(status), nil, []byte(errMsg), -1)
}
