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

func firstReq(config PluginConfig, prompt string, rawRequest Request, log wrapper.Log) types.Action {
	log.Debugf("[onHttpRequestBody] firstreq:%s", prompt)

	var userMessage Message
	userMessage.Role = "user"
	userMessage.Content = prompt

	newMessages := []Message{userMessage}
	rawRequest.Messages = newMessages

	//replace old message and resume request qwen
	newbody, err := json.Marshal(rawRequest)
	if err != nil {
		return types.ActionContinue
	} else {
		log.Debugf("[onHttpRequestBody] newRequestBody: ", string(newbody))
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
		log.Debugf("[onHttpRequestBody] body json umarshal err: ", err.Error())
		return types.ActionContinue
	}
	log.Debugf("onHttpRequestBody rawRequest: %v", rawRequest)

	//获取用户query
	var query string
	messageLength := len(rawRequest.Messages)
	log.Debugf("[onHttpRequestBody] messageLength: %s\n", messageLength)
	if messageLength > 0 {
		query = rawRequest.Messages[messageLength-1].Content
		log.Debugf("[onHttpRequestBody] query: %s\n", query)
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
	for _, apiParam := range config.APIParam {
		for _, tool_param := range apiParam.Tool_Param {
			tool_desc = append(tool_desc, fmt.Sprintf(prompttpl.TOOL_DESC, tool_param.ToolName, tool_param.Description, tool_param.Description, tool_param.Description, tool_param.Parameter), "\n")
			tool_names = append(tool_names, tool_param.ToolName)
		}
	}

	var prompt string
	if config.PromptTemplate.Language == "CH" {
		prompt = fmt.Sprintf(prompttpl.CH_Template,
			tool_desc,
			config.PromptTemplate.CHTemplate.Question,
			config.PromptTemplate.CHTemplate.Thought1,
			tool_names,
			config.PromptTemplate.CHTemplate.ActionInput,
			config.PromptTemplate.CHTemplate.Observation,
			config.PromptTemplate.CHTemplate.Thought2,
			config.PromptTemplate.CHTemplate.FinalAnswer,
			config.PromptTemplate.CHTemplate.Begin,
			query)
	} else {
		prompt = fmt.Sprintf(prompttpl.EN_Template,
			tool_desc,
			config.PromptTemplate.ENTemplate.Question,
			config.PromptTemplate.ENTemplate.Thought1,
			tool_names,
			config.PromptTemplate.ENTemplate.ActionInput,
			config.PromptTemplate.ENTemplate.Observation,
			config.PromptTemplate.ENTemplate.Thought2,
			config.PromptTemplate.ENTemplate.FinalAnswer,
			config.PromptTemplate.ENTemplate.Begin,
			query)
	}

	ctx.SetContext(ToolCallsCount, 0)

	//清理历史对话记录
	dashscope.MessageStore.Clear()

	//将请求加入到历史对话存储器中
	dashscope.MessageStore.AddForUser(prompt)

	//开始第一次请求
	ret := firstReq(config, prompt, rawRequest, log)

	return ret
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	log.Debug("onHttpResponseHeaders start")
	defer log.Debug("onHttpResponseHeaders end")

	return types.ActionContinue
}

func toolsCallResult(ctx wrapper.HttpContext, config PluginConfig, content string, rawResponse Response, log wrapper.Log, statusCode int, responseBody []byte) {
	if statusCode != http.StatusOK {
		log.Debugf("statusCode: %d\n", statusCode)
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
			log.Infof("[toolsCall] content: %s\n", responseCompletion.Choices[0].Message.Content)

			if responseCompletion.Choices[0].Message.Content != "" {
				retType := toolsCall(ctx, config, responseCompletion.Choices[0].Message.Content, rawResponse, log)
				if retType == types.ActionContinue {
					//得到了Final Answer
					var assistantMessage Message
					assistantMessage.Role = "assistant"
					startIndex := strings.Index(responseCompletion.Choices[0].Message.Content, "Final Answer:")
					if startIndex != -1 {
						startIndex += len("Final Answer:") // 移动到"Final Answer:"之后的位置
						extractedText := responseCompletion.Choices[0].Message.Content[startIndex:]
						assistantMessage.Content = extractedText
					}

					rawResponse.Choices[0].Message = assistantMessage

					newbody, err := json.Marshal(rawResponse)
					if err != nil {
						proxywasm.ResumeHttpResponse()
						return
					} else {
						log.Infof("[onHttpResponseBody] newResponseBody: ", string(newbody))
						proxywasm.ReplaceHttpResponseBody(newbody)

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

func toolsCall(ctx wrapper.HttpContext, config PluginConfig, content string, rawResponse Response, log wrapper.Log) types.Action {
	dashscope.MessageStore.AddForAssistant(content)

	//得到最终答案
	regexPattern := regexp.MustCompile(FinalAnswerPattern)
	finalAnswer := regexPattern.FindStringSubmatch(content)
	if len(finalAnswer) > 1 {
		return types.ActionContinue
	}
	count := ctx.GetContext(ToolCallsCount).(int)
	count++
	log.Debugf("toolCallsCount:%d, config.LLMInfo.MaxIterations=%d\n", count, config.LLMInfo.MaxIterations)
	//函数递归调用次数，达到了预设的循环次数，强制结束
	if int64(count) > config.LLMInfo.MaxIterations {
		ctx.SetContext(ToolCallsCount, 0)
		return types.ActionContinue
	} else {
		ctx.SetContext(ToolCallsCount, count)
	}

	//没得到最终答案
	regexAction := regexp.MustCompile(ActionPattern)
	regexActionInput := regexp.MustCompile(ActionInputPattern)

	action := regexAction.FindStringSubmatch(content)
	actionInput := regexActionInput.FindStringSubmatch(content)

	if len(action) > 1 && len(actionInput) > 1 {
		var url string
		var headers [][2]string
		var apiClient wrapper.HttpClient
		var method string
		var reqBody []byte
		var key string

		for i, apiParam := range config.APIParam {
			for _, tool_param := range apiParam.Tool_Param {
				if action[1] == tool_param.ToolName {
					log.Infof("calls %s\n", tool_param.ToolName)
					log.Infof("actionInput[1]: %s", actionInput[1])

					//将大模型需要的参数反序列化
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(actionInput[1]), &data); err != nil {
						log.Debugf("Error: %s\n", err.Error())
						return types.ActionContinue
					}

					method = tool_param.Method

					//key or header组装
					if apiParam.APIKey.Name != "" {
						if apiParam.APIKey.In == "query" { //query类型的key要放到url中
							headers = nil
							key = "?" + apiParam.APIKey.Name + "=" + apiParam.APIKey.Value
						} else if apiParam.APIKey.In == "header" { //header类型的key放在header中
							headers = [][2]string{{"Content-Type", "application/json"}, {"Authorization", apiParam.APIKey.Name + " " + apiParam.APIKey.Value}}
						}
					}

					if method == "GET" {
						//query组装
						var args string
						for i, param := range tool_param.ParamName { //从参数列表中取出参数
							if i == 0 && apiParam.APIKey.In != "query" {
								args = "?" + param + "=%s"
								args = fmt.Sprintf(args, data[param])
							} else {
								args = args + "&" + param + "=%s"
								args = fmt.Sprintf(args, data[param])
							}
						}

						//url组装
						url = apiParam.URL + tool_param.Path + key + args
					} else if method == "POST" {
						reqBody = nil
						//json参数组装
						jsonData, err := json.Marshal(data)
						if err != nil {
							log.Debugf("Error: %s\n", err.Error())
							return types.ActionContinue
						}
						reqBody = jsonData

						//url组装
						url = apiParam.URL + tool_param.Path + key
					}

					log.Infof("url: %s\n", url)

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
				}, 50000)
			if err != nil {
				log.Debugf("tool calls error: %s\n", err.Error())
				proxywasm.ResumeHttpRequest()
			}
		} else {
			return types.ActionContinue
		}
	}
	return types.ActionPause
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
	log.Infof("first content: %s\n", rawResponse.Choices[0].Message.Content)
	//如果gpt返回的内容不是空的
	if rawResponse.Choices[0].Message.Content != "" {
		//进入agent的循环思考，工具调用的过程中
		return toolsCall(ctx, config, rawResponse.Choices[0].Message.Content, rawResponse, log)
	} else {
		return types.ActionContinue
	}
}
