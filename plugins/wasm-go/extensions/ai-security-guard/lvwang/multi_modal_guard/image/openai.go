package image

import (
	"encoding/json"
	"net/http"
	"time"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/common"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type ImageItemForOpenAI struct {
	Content string
	Type    string // URL or BASE64
}

func getOpenAIImageResults(body []byte) []ImageItemForOpenAI {
	// qwen api: https://bailian.console.aliyun.com/?tab=api#/api/?type=model&url=2975126
	result := []ImageItemForOpenAI{}
	for _, part := range gjson.GetBytes(body, "data").Array() {
		if url := part.Get("url").String(); url != "" {
			result = append(result, ImageItemForOpenAI{
				Content: url,
				Type:    "URL",
			})
		}
		if b64 := part.Get("b64_json").String(); b64 != "" {
			result = append(result, ImageItemForOpenAI{
				Content: b64,
				Type:    "BASE64",
			})
		}
	}
	return result
}

func HandleOpenAIImageGenerationResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	log.Debugf("checking response body...")
	checkImageService := config.GetResponseImageCheckService(consumer)
	startTime := time.Now().UnixMilli()
	imgResults := getOpenAIImageResults(body)
	if len(imgResults) == 0 {
		return types.ActionContinue
	}
	imageIndex := 0
	var singleCall func()
	callback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		imageIndex += 1
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			if imageIndex < len(imgResults) {
				singleCall()
			} else {
				proxywasm.ResumeHttpResponse()
			}
			return
		}
		var response cfg.Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Errorf("%+v", err)
			if imageIndex < len(imgResults) {
				singleCall()
			} else {
				proxywasm.ResumeHttpResponse()
			}
			return
		}
		endTime := time.Now().UnixMilli()
		if cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
			if imageIndex >= len(imgResults) {
				ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
				ctx.SetUserAttribute("safecheck_status", "request pass")
				ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
				proxywasm.ResumeHttpResponse()
			} else {
				singleCall()
			}
			return
		}
		proxywasm.SendHttpResponse(403, [][2]string{{"content-type", "application/json"}}, []byte("illegal image"), -1)
		config.IncrementCounter("ai_sec_request_deny", 1)
		ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
		ctx.SetUserAttribute("safecheck_status", "reqeust deny")
		ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
	}
	singleCall = func() {
		img := imgResults[imageIndex]
		imgUrl := ""
		imgBase64 := ""
		if img.Type == "BASE64" {
			imgBase64 = img.Content
		} else {
			imgUrl = img.Content
		}
		path, headers, body := common.GenerateRequestForImage(config, cfg.MultiModalGuardForBase64, checkImageService, imgUrl, imgBase64)
		err := config.Client.Post(path, headers, body, callback, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpResponse()
		}
	}
	singleCall()
	return types.ActionPause
}
