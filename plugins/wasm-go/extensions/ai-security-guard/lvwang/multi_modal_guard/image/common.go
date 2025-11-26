package image

import (
	"strings"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func HandleImageGenerationResponseHeader(ctx wrapper.HttpContext, config cfg.AISecurityConfig) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	ctx.SetContext("risk_detected", false)
	if strings.Contains(contentType, "text/event-stream") {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	} else {
		ctx.BufferResponseBody()
		return types.HeaderStopIteration
	}
}
