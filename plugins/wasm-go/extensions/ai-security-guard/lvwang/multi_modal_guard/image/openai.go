package image

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/common"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/utils"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	OpenAIResponseBase64JsonPath = "data.b64_json"
	OpenAIResponseUrlJsonPath    = "data.url"
)

func HandleImageGenerationResponseHeader(ctx wrapper.HttpContext, config cfg.AISecurityConfig) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	ctx.SetContext("end_of_stream_received", false)
	ctx.SetContext("during_call", false)
	ctx.SetContext("risk_detected", false)
	sessionID, _ := utils.GenerateHexID(20)
	ctx.SetContext("sessionID", sessionID)
	if strings.Contains(contentType, "text/event-stream") {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	} else {
		return types.HeaderStopIteration
	}
}

func HandleOpenAIImageGenerationResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	log.Debugf("checking response body...")
	checkImageService := config.GetRequestImageCheckService(consumer)
	startTime := time.Now().UnixMilli()

	imgUrl := gjson.GetBytes(body, OpenAIResponseUrlJsonPath).String()
	imgBase64 := gjson.GetBytes(body, OpenAIResponseBase64JsonPath).String()
	if imgUrl != "" || imgBase64 != "" {
		path, headers, body := common.GenerateRequestForImage(config, cfg.MultiModalGuardForBase64, checkImageService, imgUrl, imgBase64)
		err := config.Client.Post(path, headers, body, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Info(string(responseBody))
			if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
				return
			}
			var response cfg.Response
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				log.Errorf("%+v", err)
				return
			}
			endTime := time.Now().UnixMilli()
			if cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
				ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
				ctx.SetUserAttribute("safecheck_status", "request pass")
				ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
				return
			}
			proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, []byte("illegal image"), -1)
			config.IncrementCounter("ai_sec_request_deny", 1)
			ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
			ctx.SetUserAttribute("safecheck_status", "reqeust deny")
			ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
		}, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpRequest()
		}
	}
	return types.ActionPause
}
