package main

import (
	"encoding/json"
	"errors"
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
	"gopkg.in/yaml.v2"
)

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

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	FrequencyPenalty float64   `json:"frequency_penalty"`
	PresencePenalty  float64   `json:"presence_penalty"`
	Stream           bool      `json:"stream"`
	Temperature      float64   `json:"temperature"`
	Topp             int32     `json:"top_p"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Response struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Object  string   `json:"object"`
	Usage   Usage    `json:"usage"`
}

type Parameter struct {
	Name        string `yaml:"name"`
	In          string `yaml:"in"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Schema      struct {
		Type    string   `yaml:"type"`
		Default string   `yaml:"default"`
		Enum    []string `yaml:"enum"`
	} `yaml:"schema"`
}

type ParamReq struct {
	Parameter []Parameter `yaml:"parameters"`
}

type ToolsInfo struct {
	Title                 string `required:"true" yaml:"title" json:"title"`
	Name_for_model        string `required:"true" yaml:"name_for_model" json:"name_for_model"`
	Description_for_model string `required:"true" yaml:"description_for_model" json:"description_for_model"`
	Parameters            string `required:"true" yaml:"parameters" json:"parameters"`
	Url                   string `rearamquired:"true" yaml:"url" json:"url"`
	Method                string `required:"true" yaml:"method" json:"method"`
}

type ToolsClientInfo struct {
	// @Title zh-CN 服务名称
	// @Description zh-CN 访问外部API服务
	ServiceName string `required:"true" yaml:"serviceName" json:"serviceName"`
	// @Title zh-CN 服务端口
	// @Description zh-CN 访问外部API服务端口
	ServicePort int64 `required:"true" yaml:"servicePort" json:"servicePort"`
	// @Title zh-CN 服务域名
	// @Description zh-CN 访问外部API服务域名，例如 restapi.amap.com
	Domin string `required:"true" yaml:"domain" json:"domain"`
	// @Title zh-CN 访问外部API服务的key
	// @Description zh-CN 访问外部API服务的key
	APIKey string `required:"true" yaml:"apiKey" json:"apiKey"`
}

type DashScopeInfo struct {
	// @Title zh-CN 通义千问大模型服务名称
	// @Description zh-CN 通义千问大模型服务名称
	ServiceName string `required:"true" yaml:"serviceName" json:"serviceName"`
	// @Title zh-CN 通义千问大模型服务端口
	// @Description zh-CN 通义千问服务端口
	ServicePort int64 `required:"true" yaml:"servicePort" json:"servicePort"`
	// @Title zh-CN 通义千问大模型服务域名
	// @Description zh-CN 通义千问大模型服务域名，例如 dashscope.aliyuncs.com
	Domin string `required:"true" yaml:"domin" json:"domin"`
	// @Title zh-CN 通义千问大模型服务的key
	// @Description zh-CN 通义千问大模型服务的key
	APIKey string `required:"true" yaml:"apiKey" json:"apiKey"`
}

type PluginConfig struct {
	// @Title zh-CN 返回 HTTP 响应的模版
	// @Description zh-CN 用 %s 标记需要被 cache value 替换的部分
	ReturnResponseTemplate string `required:"true" yaml:"returnResponseTemplate" json:"returnResponseTemplate"`
	// @Title zh-CN ToolsInfo 工具信息
	// @Description zh-CN 用于存放工具信息
	ToolsInfo []ToolsInfo `required:"true" yaml:"tools" json:"tools"`
	// @Title zh-CN ToolsClient信息
	// @Description zh-CN 用于存储ToolsClient使用信息
	ToolsClientInfo []ToolsClientInfo    `required:"true" yaml:"toolsClientInfo" json:"toolsClientInfo"`
	ToolsClient     []wrapper.HttpClient `yaml:"-" json:"-"`
	// @Title zh-CN dashscope信息
	// @Description zh-CN 用于存储dashscope使用信息
	DashScopeInfo   DashScopeInfo      `required:"true" yaml:"dashscope" json:"dashscope"`
	DashScopeClient wrapper.HttpClient `yaml:"-" json:"-"`
}

func parseConfig(json gjson.Result, c *PluginConfig, log wrapper.Log) error {
	//设置回复模板
	c.ReturnResponseTemplate = json.Get("returnResponseTemplate").String()
	if c.ReturnResponseTemplate == "" {
		c.ReturnResponseTemplate = `{"id":"from-cache","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	}

	//从插件配置中获取toolsClientInfo信息
	toolsClientInfo := json.Get("toolsClientInfo")
	if !toolsClientInfo.Exists() {
		return errors.New("toolsClientInfo is required")
	}
	if len(toolsClientInfo.Array()) == 0 {
		return errors.New("toolsClientInfo cannot be empty")
	}

	for _, item := range toolsClientInfo.Array() {
		serviceName := item.Get("serviceName")
		if !serviceName.Exists() || serviceName.String() == "" {
			return errors.New("toolsClientInfo serviceName is required")
		}

		servicePort := item.Get("servicePort")
		if !servicePort.Exists() || servicePort.Int() == 0 {
			return errors.New("toolsClientInfo servicePort is required")
		}

		domain := item.Get("domain")
		if !domain.Exists() || domain.String() == "" {
			return errors.New("toolsClientInfo domain is required")
		}

		apiKey := item.Get("apiKey")
		if !apiKey.Exists() || apiKey.String() == "" {
			return errors.New("toolsClientInfo apiKey is required")
		}

		toolsClientInfo := ToolsClientInfo{
			ServiceName: serviceName.String(),
			ServicePort: servicePort.Int(),
			Domin:       domain.String(),
			APIKey:      apiKey.String(),
		}

		c.ToolsClientInfo = append(c.ToolsClientInfo, toolsClientInfo)

		//根据多个toolsClientInfo的信息，分别初始化toolsClient
		toolsClient := wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: toolsClientInfo.ServiceName,
			Port:        toolsClientInfo.ServicePort,
			Domain:      toolsClientInfo.Domin,
		})

		c.ToolsClient = append(c.ToolsClient, toolsClient)
	}

	//从插件配置中获取tools信息
	tools := json.Get("tools")
	if !tools.Exists() {
		return errors.New("tools is required")
	}
	if len(tools.Array()) == 0 {
		return errors.New("tools cannot be empty")
	}

	for _, item := range tools.Array() {
		title := item.Get("title")
		if !title.Exists() || title.String() == "" {
			return errors.New("tools title is required")
		}

		name_for_model := item.Get("name_for_model")
		if !name_for_model.Exists() || name_for_model.String() == "" {
			return errors.New("tools name_for_model is required")
		}

		description_for_model := item.Get("description_for_model")
		if !description_for_model.Exists() || description_for_model.String() == "" {
			return errors.New("tools description_for_model is required")
		}

		parameters := item.Get("parameters")
		if !parameters.Exists() || parameters.String() == "" {
			return errors.New("tools parameters is required")
		}

		url := item.Get("url")
		if !url.Exists() || url.String() == "" {
			return errors.New("tools url is required")
		}

		method := item.Get("method")
		if !method.Exists() || method.String() == "" {
			return errors.New("tools method is required")
		}

		tool := ToolsInfo{
			Title:                 title.String(),
			Name_for_model:        name_for_model.String(),
			Description_for_model: description_for_model.String(),
			Parameters:            parameters.String(),
			Url:                   url.String(),
			Method:                method.String(),
		}

		c.ToolsInfo = append(c.ToolsInfo, tool)
	}

	c.DashScopeInfo.APIKey = json.Get("dashscope.apiKey").String()
	c.DashScopeInfo.ServiceName = json.Get("dashscope.serviceName").String()
	c.DashScopeInfo.ServicePort = json.Get("dashscope.servicePort").Int()
	c.DashScopeInfo.Domin = json.Get("dashscope.domain").String()

	c.DashScopeClient = wrapper.NewClusterClient(wrapper.DnsCluster{
		ServiceName: c.DashScopeInfo.ServiceName,
		Port:        c.DashScopeInfo.ServicePort,
		Domain:      c.DashScopeInfo.Domin,
	})

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	log.Info("onHttpRequestHeaders start")
	defer log.Info("onHttpRequestHeaders end")
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	// The request does not have a body.
	if contentType == "" {
		return types.ActionContinue
	}
	if !strings.Contains(contentType, "application/json") {
		log.Warnf("content is not json, can't process:%s", contentType)
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	// The request has a body and requires delaying the header transmission until a cache miss occurs,
	// at which point the header should be sent.
	return types.HeaderStopIteration
}

func firstreq(config PluginConfig, prompt string, rawRequest Request, log wrapper.Log) types.Action {
	log.Infof("[onHttpRequestBody] firstreq:%s", prompt)

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
		log.Infof("[onHttpRequestBody] newRequestBody: ", string(newbody))
		err := proxywasm.ReplaceHttpRequestBody(newbody)
		if err != nil {
			log.Info("替换失败")
			proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "application/json; charset=utf-8"}}, []byte(fmt.Sprintf(config.ReturnResponseTemplate, "替换失败"+err.Error())), -1)
		}
		log.Info("[onHttpRequestBody] request替换成功")
		return types.ActionContinue
	}
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	log.Info("onHttpRequestBody start")
	defer log.Info("onHttpRequestBody end")

	//拿到请求
	var rawRequest Request
	err := json.Unmarshal(body, &rawRequest)
	if err != nil {
		log.Infof("[onHttpRequestBody] body json umarshal err: ", err.Error())
		return types.ActionContinue
	}
	log.Infof("onHttpRequestBody rawRequest: %v", rawRequest)

	//获取用户query
	var query string
	messageLength := len(rawRequest.Messages)
	log.Infof("[onHttpRequestBody] messageLength: ", messageLength)
	if messageLength > 0 {
		query = rawRequest.Messages[messageLength-1].Content
		log.Infof("[onHttpRequestBody] query: ", query)
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
	for _, tool := range config.ToolsInfo {
		tool_desc = append(tool_desc, fmt.Sprintf(prompttpl.TOOL_DESC, tool.Name_for_model, tool.Description_for_model, tool.Description_for_model, tool.Description_for_model, tool.Parameters), "\n")
		tool_names = append(tool_names, tool.Name_for_model)
	}

	prompt := fmt.Sprintf(prompttpl.Template, tool_desc, tool_names, query)

	//将请求加入到历史对话存储器中
	dashscope.MessageStore.AddForUser(prompt)

	//开始第一次请求
	ret := firstreq(config, prompt, rawRequest, log)

	return ret
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	log.Info("onHttpResponseHeaders start")
	defer log.Info("onHttpResponseHeaders end")

	return types.ActionContinue
}

func toolsCall(config PluginConfig, content string, rawResponse Response, log wrapper.Log) types.Action {
	dashscope.MessageStore.AddForAssistant(content)

	//得到最终答案
	regexPattern := regexp.MustCompile(`Final Answer:(.*)`)
	finalAnswer := regexPattern.FindStringSubmatch(content)
	if len(finalAnswer) > 1 {
		return types.ActionContinue
	}

	//没得到最终答案
	regexAction := regexp.MustCompile(`Action:\s*(.*?)[.\n]`)
	regexActionInput := regexp.MustCompile(`Action Input:\s*(.*)`)

	action := regexAction.FindStringSubmatch(content)
	actionInput := regexActionInput.FindStringSubmatch(content)

	if len(action) > 1 && len(actionInput) > 1 {
		var url string
		var toolsClient wrapper.HttpClient
		var toolsInfo ToolsInfo
		var method string

		for _, tool := range config.ToolsInfo {
			if action[1] == tool.Name_for_model {
				log.Infof("calls %s\n", tool.Name_for_model)
				log.Infof("actionInput[1]: %s", actionInput[1])
				toolsInfo = tool
				break
			}
		}

		//取出工具的http request方法
		method = toolsInfo.Method

		//根据大模型要求的API所对应的title，取出对应的client和apikey
		var apiKey string
		for i, toolsClientInfo := range config.ToolsClientInfo {
			if toolsClientInfo.ServiceName == toolsInfo.Title {
				toolsClient = config.ToolsClient[i]
				apiKey = toolsClientInfo.APIKey
				break
			}
		}

		var paramReq ParamReq
		if err := yaml.Unmarshal([]byte(toolsInfo.Parameters), &paramReq); err != nil {
			log.Infof("Error: %s\n", err.Error())
			return types.ActionContinue
		}

		log.Infof("paramReq: %s %s\n", paramReq.Parameter[0].Name, paramReq.Parameter[0].In)

		//将大模型需要的参数反序列化
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(actionInput[1]), &data); err != nil {
			log.Infof("Error: %s\n", err.Error())
			return types.ActionContinue
		}

		if method == "get" {
			//按照参数列表中的参数排列顺序，从data中取出参数
			var args []interface{}
			for _, param := range paramReq.Parameter {
				if param.Name == "apiKey" {
					data[param.Name] = apiKey
				}
				if param.In == "query" { //query表示是组装到url中的参数
					arg := data[param.Name] //在反序列化后的Action Input中把参数的值取出来
					args = append(args, arg)
				}
			}

			//组装Url
			url = fmt.Sprintf(toolsInfo.Url, args...)
			log.Infof("url: %s", url)

			//调用工具
			err := toolsClient.Get(
				url,
				nil,
				func(statusCode int, responseHeaders http.Header, responseBody []byte) {
					if statusCode != http.StatusOK {
						log.Infof("statusCode: %d\n", statusCode)
					}
					log.Info("========函数返回结果========")
					log.Infof(string(responseBody))

					Observation := "Observation: " + string(responseBody)
					prompt := content + Observation

					dashscope.MessageStore.AddForUser(prompt)

					for _, v := range dashscope.MessageStore {
						log.Infof("role: %s\n", v.Role)
						log.Infof("Content: %s\n", v.Content)
					}

					completion := dashscope.Completion{
						Model:    "qwen-long",
						Messages: dashscope.MessageStore,
					}

					headers := [][2]string{{"Content-Type", "application/json"}, {"Authorization", "Bearer " + config.DashScopeInfo.APIKey}}
					completionSerialized, _ := json.Marshal(completion)
					err := config.DashScopeClient.Post(
						"/compatible-mode/v1/chat/completions",
						headers,
						completionSerialized,
						func(statusCode int, responseHeaders http.Header, responseBody []byte) {
							//得到gpt的返回结果
							var responseCompletion dashscope.CompletionResponse
							_ = json.Unmarshal(responseBody, &responseCompletion)
							log.Infof("[toolsCall] content: ", responseCompletion.Choices[0].Message.Content)

							if responseCompletion.Choices[0].Message.Content != "" {
								retType := toolsCall(config, responseCompletion.Choices[0].Message.Content, rawResponse, log)
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
									//assistantMessage.Content = responseCompletion.Choices[0].Message.Content
									rawResponse.Choices[0].Message = assistantMessage

									newbody, err := json.Marshal(rawResponse)
									if err != nil {
										proxywasm.ResumeHttpResponse()
										return
									} else {
										log.Infof("[onHttpResponseBody] newResponseBody: ", string(newbody))
										proxywasm.ReplaceHttpResponseBody(newbody)

										log.Info("[onHttpResponseBody] response替换成功")
										proxywasm.ResumeHttpResponse()
									}
								}
							} else {
								proxywasm.ResumeHttpRequest()
							}
						}, 50000)
					if err != nil {
						log.Infof("[onHttpRequestBody] completion err: %s", err.Error())
						proxywasm.ResumeHttpRequest()
					}
				}, 50000)
			if err != nil {
				log.Infof("tool calls error: %s\n", err.Error())
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
	log.Info("onHttpResponseBody start")
	defer log.Info("onHttpResponseBody end")

	//初始化接收gpt返回内容的结构体
	var rawResponse Response
	err := json.Unmarshal(body, &rawResponse)
	if err != nil {
		log.Infof("[onHttpResponseBody] body to json err: %s", err.Error())
		return types.ActionContinue
	}

	//如果gpt返回的内容不是空的
	if rawResponse.Choices[0].Message.Content != "" {
		//进入agent的循环思考，工具调用的过程中
		return toolsCall(config, rawResponse.Choices[0].Message.Content, rawResponse, log)
	} else {
		return types.ActionContinue
	}
}
