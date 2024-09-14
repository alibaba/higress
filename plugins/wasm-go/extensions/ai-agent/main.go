package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-agent/dashscope"
	prompttpl "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-agent/promptTpl"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

// 用于统计函数的递归调用次数
const ToolCallsCount = "ToolCallsCount"
const StreamContextKey = "Stream"

// react的正则规则
const ActionPattern = `Action:\s*(.*?)[.\n]`
const ActionInputPattern = `Action Input:\s*(.*)`
const FinalAnswerPattern = `Final Answer:(.*)`

func main() {
	wrapper.SetCtx(
		"ai-agent",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

func parseConfig(gjson gjson.Result, c *PluginConfig, log wrapper.Log) error {
	initResponsePromptTpl(gjson, c)

	err := initAPIs(gjson, c)
	if err != nil {
		return err
	}

	initReActPromptTpl(gjson, c)

	initLLMClient(gjson, c)

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	return types.ActionContinue
}

func firstReq(ctx wrapper.HttpContext, config PluginConfig, prompt string, rawRequest Request, log wrapper.Log) types.Action {
	log.Debugf("[onHttpRequestBody] firstreq:%s", prompt)

	var userMessage Message
	userMessage.Role = "user"
	userMessage.Content = prompt

	newMessages := []Message{userMessage}
	rawRequest.Messages = newMessages
	if rawRequest.Stream {
		ctx.SetContext(StreamContextKey, struct{}{})
		rawRequest.Stream = false
	}

	//replace old message and resume request qwen
	newbody, err := json.Marshal(rawRequest)
	if err != nil {
		return types.ActionContinue
	} else {
		log.Debugf("[onHttpRequestBody] newRequestBody: %s", string(newbody))
		err := proxywasm.ReplaceHttpRequestBody(newbody)
		if err != nil {
			log.Debug("替换失败")
			proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "application/json; charset=utf-8"}}, []byte(fmt.Sprintf(config.ReturnResponseTemplate, "替换失败"+err.Error())), -1)
		}
		log.Debug("[onHttpRequestBody] request替换成功")
		return types.ActionContinue
	}
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	log.Debug("onHttpRequestBody start")
	defer log.Debug("onHttpRequestBody end")

	//拿到请求
	var rawRequest Request
	err := json.Unmarshal(body, &rawRequest)
	if err != nil {
		log.Debugf("[onHttpRequestBody] body json umarshal err: %s", err.Error())
		return types.ActionContinue
	}
	log.Debugf("onHttpRequestBody rawRequest: %v", rawRequest)

	//获取用户query
	var query string
	var history string
	messageLength := len(rawRequest.Messages)
	log.Debugf("[onHttpRequestBody] messageLength: %s", messageLength)
	if messageLength > 0 {
		query = rawRequest.Messages[messageLength-1].Content
		log.Debugf("[onHttpRequestBody] query: %s", query)
		if messageLength >= 3 {
			for i := 0; i < messageLength-1; i += 2 {
				history += "human: " + rawRequest.Messages[i].Content + "\nAI: " + rawRequest.Messages[i+1].Content
			}
		} else {
			history = ""
		}
	} else {
		return types.ActionContinue
	}

	if query == "" {
		log.Debug("parse query from request body failed")
		return types.ActionContinue
	}

	//拼装agent prompt模板
	tool_desc := make([]string, 0)
	tool_names := make([]string, 0)
	for _, apisParam := range config.APIsParam {
		for _, tool_param := range apisParam.ToolsParam {
			tool_desc = append(tool_desc, fmt.Sprintf(prompttpl.TOOL_DESC, tool_param.ToolName, tool_param.Description, tool_param.Description, tool_param.Description, tool_param.Parameter), "\n")
			tool_names = append(tool_names, tool_param.ToolName)
		}
	}

	var prompt string
	if config.PromptTemplate.Language == "CH" {
		prompt = fmt.Sprintf(prompttpl.CH_Template,
			tool_desc,
			tool_names,
			config.PromptTemplate.CHTemplate.Question,
			config.PromptTemplate.CHTemplate.Thought1,
			config.PromptTemplate.CHTemplate.Observation,
			config.PromptTemplate.CHTemplate.Thought2,
			history,
			query)
	} else {
		prompt = fmt.Sprintf(prompttpl.EN_Template,
			tool_desc,
			tool_names,
			config.PromptTemplate.ENTemplate.Question,
			config.PromptTemplate.ENTemplate.Thought1,
			config.PromptTemplate.ENTemplate.Observation,
			config.PromptTemplate.ENTemplate.Thought2,
			history,
			query)
	}

	ctx.SetContext(ToolCallsCount, 0)

	//清理历史对话记录
	dashscope.MessageStore.Clear()

	//将请求加入到历史对话存储器中
	dashscope.MessageStore.AddForUser(prompt)

	//开始第一次请求
	ret := firstReq(ctx, config, prompt, rawRequest, log)

	return ret
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	log.Debug("onHttpResponseHeaders start")
	defer log.Debug("onHttpResponseHeaders end")

	return types.ActionContinue
}

func toolsCallResult(ctx wrapper.HttpContext, config PluginConfig, content string, rawResponse Response, log wrapper.Log, statusCode int, responseBody []byte) {
	if statusCode != http.StatusOK {
		log.Debugf("statusCode: %d", statusCode)
	}
	log.Info("========函数返回结果========")
	log.Infof(string(responseBody))

	observation := "Observation: " + string(responseBody)

	dashscope.MessageStore.AddForUser(observation)

	completion := dashscope.Completion{
		Model:     config.LLMInfo.Model,
		Messages:  dashscope.MessageStore,
		MaxTokens: config.LLMInfo.MaxTokens,
	}

	headers := [][2]string{{"Content-Type", "application/json"}, {"Authorization", "Bearer " + config.LLMInfo.APIKey}}
	completionSerialized, _ := json.Marshal(completion)
	err := config.LLMClient.Post(
		config.LLMInfo.Path,
		headers,
		completionSerialized,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			//得到gpt的返回结果
			var responseCompletion dashscope.CompletionResponse
			_ = json.Unmarshal(responseBody, &responseCompletion)
			log.Infof("[toolsCall] content: %s", responseCompletion.Choices[0].Message.Content)

			if responseCompletion.Choices[0].Message.Content != "" {
				retType, actionInput := toolsCall(ctx, config, responseCompletion.Choices[0].Message.Content, rawResponse, log)
				if retType == types.ActionContinue {
					//得到了Final Answer
					var assistantMessage Message
					if ctx.GetContext(StreamContextKey) == nil {
						assistantMessage.Role = "assistant"
						assistantMessage.Content = actionInput
						rawResponse.Choices[0].Message = assistantMessage
						newbody, err := json.Marshal(rawResponse)
						if err != nil {
							proxywasm.ResumeHttpResponse()
							return
						} else {
							proxywasm.ReplaceHttpResponseBody(newbody)

							log.Debug("[onHttpResponseBody] response替换成功")
							proxywasm.ResumeHttpResponse()
						}
					} else {
						headers := [][2]string{{"content-type", "text/event-stream; charset=utf-8"}}
						proxywasm.ReplaceHttpResponseHeaders(headers)
						// Remove quotes from actionInput
						actionInput = strings.Trim(actionInput, "\"")
						returnStreamResponseTemplate := `data:{"id":"%s","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"%s","object":"chat.completion","usage":{"prompt_tokens":%d,"completion_tokens":%d,"total_tokens":%d}}` + "\n\ndata:[DONE]\n\n"
						newbody := fmt.Sprintf(returnStreamResponseTemplate, rawResponse.ID, actionInput, rawResponse.Model, rawResponse.Usage.PromptTokens, rawResponse.Usage.CompletionTokens, rawResponse.Usage.TotalTokens)
						log.Infof("[onHttpResponseBody] newResponseBody: ", newbody)
						proxywasm.ReplaceHttpResponseBody([]byte(newbody))

						log.Debug("[onHttpResponseBody] response替换成功")
						proxywasm.ResumeHttpResponse()
					}
				}
			} else {
				proxywasm.ResumeHttpRequest()
			}
		}, uint32(config.LLMInfo.MaxExecutionTime))
	if err != nil {
		log.Debugf("[onHttpRequestBody] completion err: %s", err.Error())
		proxywasm.ResumeHttpRequest()
	}
}

func outputParser(response string, log wrapper.Log) (string, string) {
	log.Debugf("Raw response:%s", response)

	start := strings.Index(response, "```")
	end := strings.LastIndex(response, "```")

	var jsonStr string
	if start != -1 && end != -1 {
		jsonStr = strings.TrimSpace(response[start+3 : end])
	} else {
		jsonStr = response
	}

	log.Debugf("Extracted JSON string:%s", jsonStr)

	var action map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &action)
	if err == nil {
		var actionName, actionInput string
		for key, value := range action {
			if strings.Contains(strings.ToLower(key), "input") {
				actionInput = fmt.Sprintf("%v", value)
			} else {
				actionName = fmt.Sprintf("%v", value)
			}
		}
		if actionName != "" && actionInput != "" {
			return actionName, actionInput
		}
	}
	log.Debugf("json parse err: %s", err.Error())
	// Fallback to regex parsing if JSON unmarshaling fails
	pattern := `\{\s*"action":\s*"([^"]+)",\s*"action_input":\s*((?:\{[^}]+\})|"[^"]+")\s*\}`
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(jsonStr)

	if len(match) == 3 {
		action := match[1]
		actionInput := match[2]
		log.Debugf("Parsed action:%s, action_input:%s", action, actionInput)
		return action, actionInput
	}

	log.Debug("No valid action and action_input found in the response")
	return "", ""
}

func toolsCall(ctx wrapper.HttpContext, config PluginConfig, content string, rawResponse Response, log wrapper.Log) (types.Action, string) {
	dashscope.MessageStore.AddForAssistant(content)

	action, actionInput := outputParser(content, log)

	//得到最终答案
	if action == "Final Answer" {
		return types.ActionContinue, actionInput
	}
	count := ctx.GetContext(ToolCallsCount).(int)
	count++
	log.Debugf("toolCallsCount:%d, config.LLMInfo.MaxIterations=%d", count, config.LLMInfo.MaxIterations)
	//函数递归调用次数，达到了预设的循环次数，强制结束
	if int64(count) > config.LLMInfo.MaxIterations {
		ctx.SetContext(ToolCallsCount, 0)
		return types.ActionContinue, ""
	} else {
		ctx.SetContext(ToolCallsCount, count)
	}

	//没得到最终答案

	var url string
	var headers [][2]string
	var apiClient wrapper.HttpClient
	var method string
	var reqBody []byte
	var key string
	var maxExecutionTime int64

	for i, apisParam := range config.APIsParam {
		maxExecutionTime = apisParam.MaxExecutionTime
		for _, tools_param := range apisParam.ToolsParam {
			if action == tools_param.ToolName {
				log.Infof("calls %s", tools_param.ToolName)
				log.Infof("actionInput: %s", actionInput)

				//将大模型需要的参数反序列化
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(actionInput), &data); err != nil {
					log.Debugf("Error: %s", err.Error())
					return types.ActionContinue, ""
				}

				method = tools_param.Method

				// 组装 headers 和 key
				headers = [][2]string{{"Content-Type", "application/json"}}
				if apisParam.APIKey.Name != "" {
					if apisParam.APIKey.In == "query" {
						key = "?" + apisParam.APIKey.Name + "=" + apisParam.APIKey.Value
					} else if apisParam.APIKey.In == "header" {
						headers = append(headers, [2]string{"Authorization", apisParam.APIKey.Name + " " + apisParam.APIKey.Value})
					}
				}

				// 组装 URL 和请求体
				url = apisParam.URL + tools_param.Path + key
				if method == "GET" {
					queryParams := make([]string, 0, len(tools_param.ParamName))
					for _, param := range tools_param.ParamName {
						if value, ok := data[param]; ok {
							queryParams = append(queryParams, fmt.Sprintf("%s=%v", param, value))
						}
					}
					if len(queryParams) > 0 {
						url += "&" + strings.Join(queryParams, "&")
					}
				} else if method == "POST" {
					var err error
					reqBody, err = json.Marshal(data)
					if err != nil {
						log.Debugf("Error marshaling JSON: %s", err.Error())
						return types.ActionContinue, ""
					}
				}

				log.Infof("url: %s", url)

				apiClient = config.APIClient[i]
				break
			}
		}
	}

	if apiClient != nil {
		err := apiClient.Call(
			method,
			url,
			headers,
			reqBody,
			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
				toolsCallResult(ctx, config, content, rawResponse, log, statusCode, responseBody)
			}, uint32(maxExecutionTime))
		if err != nil {
			log.Debugf("tool calls error: %s", err.Error())
			proxywasm.ResumeHttpRequest()
		}
	} else {
		return types.ActionContinue, ""
	}

	return types.ActionPause, ""
}

// 从response接收到firstreq的大模型返回
func onHttpResponseBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	log.Debugf("onHttpResponseBody start")
	defer log.Debugf("onHttpResponseBody end")

	//初始化接收gpt返回内容的结构体
	var rawResponse Response
	err := json.Unmarshal(body, &rawResponse)
	if err != nil {
		log.Debugf("[onHttpResponseBody] body to json err: %s", err.Error())
		return types.ActionContinue
	}
	log.Infof("first content: %s", rawResponse.Choices[0].Message.Content)
	//如果gpt返回的内容不是空的
	if rawResponse.Choices[0].Message.Content != "" {
		//进入agent的循环思考，工具调用的过程中
		retType, _ := toolsCall(ctx, config, rawResponse.Choices[0].Message.Content, rawResponse, log)
		return retType
	} else {
		return types.ActionContinue
	}
}
