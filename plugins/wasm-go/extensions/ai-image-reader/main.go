package main

import (
	"encoding/json"
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
	model                  string = "qwen-vl-ocr"
	queryUrl               string = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	timeoutMillisecond     uint32 = 30000
	minPixels              int    = 3136
	maxPixels              int    = 1003520
	DEFAULT_MAX_BODY_BYTES uint32 = 100 * 1024 * 1024
)

type Config struct {
	apiKey         string
	serviceName    string
	servicePort    int64
	promptTemplate string
}

type QwenOcrReq struct {
	Model    string        `json:"model,omitempty"`
	Messages []chatMessage `json:"messages,omitempty"`
}

type QwenOcrResp struct {
	Choices []chatCompletionChoice `json:"choices"`
}

type chatCompletionChoice struct {
	Message *chatMessageContent `json:"message,omitempty"`
}

type chatMessageContent struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type chatMessage struct {
	Role    string    `json:"role"`
	Content []content `json:"content"`
}

type imageURL struct {
	URL string `json:"url"`
}

type content struct {
	Type      string   `json:"type"`
	ImageUrl  imageURL `json:"image_url,omitempty"`
	MinPixels int      `json:"min_pixels,omitempty"`
	MaxPixels int      `json:"max_pixels,omitempty"`
	Text      string   `json:"text,omitempty"`
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
	config.apiKey = json.Get("apiKey").String()
	config.serviceName = json.Get("serviceName").String()
	config.servicePort = json.Get("servicePort").Int()
	config.promptTemplate = `# 用户发送的图片解析得到的文字内容如下:
{image_content}
在回答时，请注意以下几点：
- 请你回答问题时结合用户图片的文字内容回答。
- 除非用户要求，否则你回答的语言需要和用户提问的语言保持一致。

# 用户消息为：
{question}`
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	// The request does not have a body.
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
					query = content[j].Get("text.text").String()
				}
			}
			break
		}
	}
	if len(imageUrls) == 0 {
		return types.ActionContinue
	}
	return executeReadImage(imageUrls, config, query, queryIndex, log)
}

func executeReadImage(imageUrls []string, config Config, query string, queryIndex int, log wrapper.Log) types.Action {
	var imageContents []string
	client := wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: config.serviceName,
		Port: config.servicePort,
	})
	var totalImages int
	var finished int
	for _, imageUrl := range imageUrls {
		reqBody := QwenOcrReq{
			Model: model,
			Messages: []chatMessage{
				{
					Role: "user",
					Content: []content{
						{
							Type: "image_url",
							ImageUrl: imageURL{
								URL: imageUrl,
							},
							MinPixels: minPixels,
							MaxPixels: maxPixels,
						},
					},
				},
			},
		}
		body, err := json.Marshal(reqBody)
		if err != nil {
			log.Errorf("Failed to marshal request: %v", err)
		}
		var resp QwenOcrResp
		err = client.Post(queryUrl,
			[][2]string{
				{"Content-Type", "application/json"},
				{"Authorization", fmt.Sprintf("Bearer %s", config.apiKey)},
			}, body,
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
				if err := json.Unmarshal(responseBody, &resp); err != nil {
					log.Errorf("unable to unmarshal ocr response: %v", err)
					proxywasm.ResumeHttpRequest()
					return
				}
				imageContents = append(imageContents, resp.Choices[0].Message.Content)
			}, timeoutMillisecond)
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
