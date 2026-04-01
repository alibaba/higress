package text

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
	"github.com/tidwall/resp"
)

type ImageItem struct {
	Content string
	Type    string // URL or BASE64
}

func parseContent(json gjson.Result) (text string, images []ImageItem) {
	images = []ImageItem{}
	if json.IsArray() {
		for _, item := range json.Array() {
			switch item.Get("type").String() {
			case "text":
				text += item.Get("text").String()
			case "image_url":
				imgContent := item.Get("image_url.url").String()
				if strings.HasPrefix(imgContent, "data:image") {
					images = append(images, ImageItem{
						Content: imgContent,
						Type:    "BASE64",
					})
				} else {
					images = append(images, ImageItem{
						Content: imgContent,
						Type:    "URL",
					})
				}
			}
		}
	} else {
		text = json.String()
	}
	return text, images
}

func HandleTextGenerationRequestBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	if config.CheckAllMessages && config.RedisClient != nil {
		return handleRequestWithDedup(ctx, config, body)
	}
	return handleDefaultRequest(ctx, config, body)
}

func handleDefaultRequest(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	checkService := config.GetRequestCheckService(consumer)
	checkImageService := config.GetRequestImageCheckService(consumer)
	startTime := time.Now().UnixMilli()
	content, images := parseContent(gjson.GetBytes(body, config.RequestContentJsonPath))
	log.Debugf("Raw request content is: %s", content)
	if len(content) == 0 && len(images) == 0 {
		log.Info("request content is empty. skip")
		return types.ActionContinue
	}

	imageIndex := 0
	var singleCallForImage func()
	callbackForImage := buildImageCallback(ctx, config, body, images, startTime, consumer, &imageIndex, &singleCallForImage, func() {
		proxywasm.ResumeHttpRequest()
	})
	singleCallForImage = buildImageCaller(config, images, checkImageService, &imageIndex, callbackForImage)

	textCheckFn := func(contentPiece, sessionID string) (string, [][2]string, []byte) {
		return common.GenerateRequestForText(config, cfg.MultiModalGuard, checkService, contentPiece, sessionID)
	}
	if len(content) > 0 {
		common.RunChunkedTextCheck(ctx, config, body, content, startTime, consumer, textCheckFn, func() {
			if len(images) > 0 && config.CheckRequestImage {
				singleCallForImage()
			} else {
				proxywasm.ResumeHttpRequest()
			}
		})
	} else {
		singleCallForImage()
	}
	return types.ActionPause
}

func handleRequestWithDedup(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	checkService := config.GetRequestCheckService(consumer)
	checkImageService := config.GetRequestImageCheckService(consumer)
	policyFingerprint := config.BuildPolicyFingerprint(consumer)
	startTime := time.Now().UnixMilli()

	allMessages := utils.ParseAllMessages(body)
	messages := utils.FilterByRole(allMessages, "system", "user")
	log.Infof("[dedup] %d messages after role filter (system/user only), %d total", len(messages), len(allMessages))
	if len(messages) == 0 {
		log.Info("no messages to check after role filter, skip")
		return types.ActionContinue
	}

	keys := utils.BuildRedisKeys(messages, consumer, policyFingerprint)
	err := config.RedisClient.MGet(keys, func(redisResponse resp.Value) {
		unchecked := utils.FilterUnchecked(messages, redisResponse)
		log.Infof("[dedup] total=%d, unchecked=%d, cached=%d", len(messages), len(unchecked), len(messages)-len(unchecked))
		if len(unchecked) == 0 {
			log.Info("all messages already checked, skip security check")
			proxywasm.ResumeHttpRequest()
			return
		}

		content := utils.ConcatTextContent(unchecked)
		var images []ImageItem
		lastMsg := messages[len(messages)-1]
		lastMsgUnchecked := false
		for _, u := range unchecked {
			if u.Index == lastMsg.Index {
				lastMsgUnchecked = true
				break
			}
		}
		if lastMsgUnchecked && config.CheckRequestImage {
			_, images = parseContent(gjson.GetBytes(body, config.RequestContentJsonPath))
		}

		if len(content) == 0 && len(images) == 0 {
			log.Info("no content to check in unchecked messages, marking as checked")
			utils.MarkChecked(config.RedisClient, unchecked, consumer, policyFingerprint, config.CheckRecordTTL, func() {
				proxywasm.ResumeHttpRequest()
			})
			return
		}

		markCheckedAndResume := func() {
			utils.MarkChecked(config.RedisClient, unchecked, consumer, policyFingerprint, config.CheckRecordTTL, func() {
				proxywasm.ResumeHttpRequest()
			})
		}

		imageIndex := 0
		var singleCallForImage func()
		callbackForImage := buildImageCallback(ctx, config, body, images, startTime, consumer, &imageIndex, &singleCallForImage, markCheckedAndResume)
		singleCallForImage = buildImageCaller(config, images, checkImageService, &imageIndex, callbackForImage)

		textCheckFn := func(contentPiece, sessionID string) (string, [][2]string, []byte) {
			return common.GenerateRequestForText(config, cfg.MultiModalGuard, checkService, contentPiece, sessionID)
		}
		if len(content) > 0 {
			common.RunChunkedTextCheck(ctx, config, body, content, startTime, consumer, textCheckFn, func() {
				if len(images) > 0 {
					singleCallForImage()
				} else {
					markCheckedAndResume()
				}
			})
		} else if len(images) > 0 {
			singleCallForImage()
		}
	})
	if err != nil {
		log.Warnf("redis MGet failed: %v, fallback to default check", err)
		return handleDefaultRequest(ctx, config, body)
	}
	return types.ActionPause
}

// buildImageCallback creates the callback for sequential image checking.
// onAllPassed is called when all images pass (e.g., ResumeHttpRequest or MarkChecked+Resume).
func buildImageCallback(
	ctx wrapper.HttpContext,
	config cfg.AISecurityConfig,
	body []byte,
	images []ImageItem,
	startTime int64,
	consumer string,
	imageIndex *int,
	singleCallForImage *func(),
	onAllPassed func(),
) func(int, http.Header, []byte) {
	return func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		*imageIndex += 1
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			if *imageIndex < len(images) {
				(*singleCallForImage)()
			} else {
				proxywasm.ResumeHttpRequest()
			}
			return
		}
		var response cfg.Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Errorf("%+v", err)
			if *imageIndex < len(images) {
				(*singleCallForImage)()
			} else {
				proxywasm.ResumeHttpRequest()
			}
			return
		}
		if cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
			if *imageIndex >= len(images) {
				common.SetRequestPassAttributes(ctx, startTime)
				onAllPassed()
			} else {
				(*singleCallForImage)()
			}
			return
		}
		common.SendDenyResponse(ctx, config, body, response, startTime)
	}
}

// buildImageCaller creates the function that dispatches image check requests sequentially.
func buildImageCaller(
	config cfg.AISecurityConfig,
	images []ImageItem,
	checkImageService string,
	imageIndex *int,
	callbackForImage func(int, http.Header, []byte),
) func() {
	return func() {
		img := images[*imageIndex]
		imgUrl := ""
		imgBase64 := ""
		if img.Type == "BASE64" {
			imgBase64 = img.Content
		} else {
			imgUrl = img.Content
		}
		path, headers, reqBody := common.GenerateRequestForImage(config, cfg.MultiModalGuardForBase64, checkImageService, imgUrl, imgBase64)
		err := config.Client.Post(path, headers, reqBody, callbackForImage, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpRequest()
		}
	}
}
