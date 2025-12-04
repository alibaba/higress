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

func getQwenImageUrls(body []byte) []string {
	// qwen api: https://bailian.console.aliyun.com/?tab=api#/api/?type=model&url=2975126
	result := []string{}
	// 文生图/文生图v1/文生图v2/通用图像编辑2.5/通用图像编辑2.1/涂鸦作画/图像局部重绘/人像风格重绘
	// 虚拟模特/图像背景生成/人物写真FaceChain/文生图StableDiffusion/文生图FLUX/文字纹理生成API
	for _, part := range gjson.GetBytes(body, "output.results").Array() {
		if url := part.Get("url").String(); url != "" {
			result = append(result, url)
		}
	}
	// 图像编辑
	for _, part := range gjson.GetBytes(body, "output.choices.0.message.content").Array() {
		if url := part.Get("image").String(); url != "" {
			result = append(result, url)
		}
	}
	// 图像翻译/AI试衣OutfitAnyone
	if url := gjson.GetBytes(body, "output.image_url").String(); url != "" {
		result = append(result, url)
	}
	// 图像画面扩展/(part of)人物实例分割/图像擦除补全
	if url := gjson.GetBytes(body, "output.output_image_url").String(); url != "" {
		result = append(result, url)
	}
	// 鞋靴模特
	if url := gjson.GetBytes(body, "output.result_url").String(); url != "" {
		result = append(result, url)
	}
	// 创意海报生成
	for _, part := range gjson.GetBytes(body, "output.render_urls").Array() {
		if url := part.String(); url != "" {
			result = append(result, url)
		}
	}
	for _, part := range gjson.GetBytes(body, "output.bg_urls").Array() {
		if url := part.String(); url != "" {
			result = append(result, url)
		}
	}
	// 人物实例分割
	if url := gjson.GetBytes(body, "output.output_vis_image_url").String(); url != "" {
		result = append(result, url)
	}
	// 文字变形API
	for _, part := range gjson.GetBytes(body, "output.results").Array() {
		if url := part.Get("png_url").String(); url != "" {
			result = append(result, url)
		}
		if url := part.Get("svg_url").String(); url != "" {
			result = append(result, url)
		}
	}
	return result
}

func HandleQwenImageGenerationResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	log.Debugf("checking response body...")
	checkImageService := config.GetResponseImageCheckService(consumer)
	startTime := time.Now().UnixMilli()
	imgUrls := getQwenImageUrls(body)
	if len(imgUrls) == 0 {
		return types.ActionContinue
	}
	imageIndex := 0
	var singleCall func()
	callback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		imageIndex += 1
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			if imageIndex < len(imgUrls) {
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
			if imageIndex < len(imgUrls) {
				singleCall()
			} else {
				proxywasm.ResumeHttpResponse()
			}
			return
		}
		endTime := time.Now().UnixMilli()
		if cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
			if imageIndex >= len(imgUrls) {
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
		imgUrl := imgUrls[imageIndex]
		path, headers, body := common.GenerateRequestForImage(config, cfg.MultiModalGuardForBase64, checkImageService, imgUrl, "")
		err := config.Client.Post(path, headers, body, callback, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpResponse()
		}
	}
	singleCall()
	return types.ActionPause
}
