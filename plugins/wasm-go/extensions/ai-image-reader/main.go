package main

import (
	"ai-image-reader/ocr"
	"ai-image-reader/ocr/qwen"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
	"strings"
)

const (
	DEFAULT_MAX_BODY_BYTES uint32 = 100 * 1024 * 1024
)

type Config struct {
	promptTemplate string
	ocrClient      ocr.OcrClient
}

func main() {
	wrapper.SetCtx(
		"ai-image-reader",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

func parseConfig(json gjson.Result, config *Config, log wrapper.Log) error {
	config.promptTemplate = `# 用户发送的图片解析得到的文字内容如下:
{image_content}
在回答时，请注意以下几点：
- 请你回答问题时结合用户图片的文字内容回答。
- 除非用户要求，否则你回答的语言需要和用户提问的语言保持一致。

# 用户消息为：
{question}`
	provider := json.Get("provider").String()
	switch provider {
	case "qwen":
		ocrClient, err := qwen.NewQwenOcr(&json)
		if err != nil {
			return fmt.Errorf("qwen ocr client init failed:%s", err)
		}
		config.ocrClient = ocrClient
		return nil
	default:
		return fmt.Errorf("unkown ocr provider:%s", provider)
	}
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	if contentType == "" {
		return types.ActionContinue
	}
	if !strings.Contains(contentType, "application/json") {
		log.Warnf("content is not json, can't process: %s", contentType)
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	ctx.SetRequestBodyBufferLimit(DEFAULT_MAX_BODY_BYTES)
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config Config, body []byte, log wrapper.Log) types.Action {
	var queryIndex int
	var query string
	messages := gjson.GetBytes(body, "messages").Array()
	var imageUrls []string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Get("role").String() == "user" {
			queryIndex = i
			content := messages[i].Get("content").Array()
			for j := len(content) - 1; j >= 0; j-- {
				if content[j].Get("type").String() == "image_url" {
					imageUrls = append(imageUrls, content[j].Get("image_url.url").String())
				} else if content[j].Get("type").String() == "text" {
					query = content[j].Get("text").String()
				}
			}
			break
		}
	}
	if len(imageUrls) == 0 {
		return types.ActionContinue
	}
	return executeReadImage(imageUrls, config, query, queryIndex, body, log)
}

func executeReadImage(imageUrls []string, config Config, query string, queryIndex int, body []byte, log wrapper.Log) types.Action {
	var imageContents []string
	var totalImages int
	var finished int
	for _, imageUrl := range imageUrls {
		args := config.ocrClient.CallArgs(imageUrl)
		err := config.ocrClient.Client().Call(args.Method, args.Url, args.Headers, args.Body,
			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
				defer func() {
					finished++
					if totalImages == finished {
						var processedContents []string
						for idx := len(imageContents) - 1; idx >= 0; idx-- {
							processedContents = append(processedContents, fmt.Sprintf("第%d张图片内容为 %s", totalImages-idx, imageContents[idx]))
						}
						imageSummary := fmt.Sprintf("总共有 %d 张图片。\n", totalImages)
						prompt := strings.Replace(config.promptTemplate, "{image_content}", imageSummary+strings.Join(processedContents, "\n"), 1)
						prompt = strings.Replace(prompt, "{question}", query, 1)
						modifiedBody, err := sjson.SetBytes(body, fmt.Sprintf("messages.%d.content", queryIndex), prompt)
						if err != nil {
							log.Errorf("modify request message content failed, err:%v, body:%s", err, body)
						} else {
							log.Debugf("modifeid body:%s", modifiedBody)
							proxywasm.ReplaceHttpRequestBody(modifiedBody)
						}
						proxywasm.ResumeHttpRequest()
					}
				}()
				if statusCode != http.StatusOK {
					log.Errorf("ocr call failed, status: %d", statusCode)
					return
				}
				imageContents = append(imageContents, config.ocrClient.ParseResult(responseBody))
			}, args.TimeoutMillisecond)
		if err != nil {
			log.Infof("ocr call failed, err:%v", err)
			continue
		}
		totalImages++
	}
	if totalImages > 0 {
		return types.ActionPause
	}
	return types.ActionContinue
}
