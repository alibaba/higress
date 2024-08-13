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

// 用于存放拆解出来的工具相关信息
type Tool_Param struct {
	ToolName   string   `yaml:"toolName"`
	Path       string   `yaml:"path"`
	Method     string   `yaml:"method"`
	ParamName  []string `yaml:"paramName"`
	Parameter  string   `yaml:"parameter"`
	Desciption string   `yaml:"description"`
}

// 用于存放拆解出来的api相关信息
type API_Param struct {
	APIKey     APIKey       `yaml:"apiKey"`
	URL        string       `yaml:"url"`
	Tool_Param []Tool_Param `yaml:"tool_Param"`
}

type Info struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

type Server struct {
	URL string `yaml:"url"`
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

type PathItem struct {
	Description string      `yaml:"description"`
	OperationID string      `yaml:"operationId"`
	Parameters  []Parameter `yaml:"parameters"`
	Deprecated  bool        `yaml:"deprecated"`
}

type Paths map[string]map[string]PathItem

type Components struct {
	Schemas map[string]interface{} `yaml:"schemas"`
}

type API struct {
	OpenAPI    string     `yaml:"openapi"`
	Info       Info       `yaml:"info"`
	Servers    []Server   `yaml:"servers"`
	Paths      Paths      `yaml:"paths"`
	Components Components `yaml:"components"`
}

type APIKey struct {
	In    string `yaml:"in" json:"in"`
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

type APIProvider struct {
	// @Title zh-CN 服务名称
	// @Description zh-CN 带服务类型的完整 FQDN 名称，例如 my-redis.dns、redis.my-ns.svc.cluster.local
	ServiceName string `required:"true" yaml:"serviceName" json:"serviceName"`
	// @Title zh-CN 服务端口
	// @Description zh-CN 服务端口
	ServicePort int64 `required:"true" yaml:"servicePort" json:"servicePort"`
	// @Title zh-CN 服务域名
	// @Description zh-CN 服务域名，例如 restapi.amap.com
	Domin string `required:"true" yaml:"domain" json:"domain"`
	// @Title zh-CN 通义千问大模型服务的key
	// @Description zh-CN 通义千问大模型服务的key
	APIKey APIKey `required:"true" yaml:"apiKey" json:"apiKey"`
}

type APIs struct {
	APIProvider APIProvider `required:"true" yaml:"apiProvider" json:"apiProvider"`
	API         string      `required:"true" yaml:"api" json:"api"`
}

type Template struct {
	Question    string `yaml:"question" json:"question"`
	Thought1    string `yaml:"thought1" json:"thought1"`
	ActionInput string `yaml:"actionInput" json:"actionInput"`
	Observation string `yaml:"observation" json:"observation"`
	Thought2    string `yaml:"thought2" json:"thought2"`
	FinalAnswer string `yaml:"finalAnswer" json:"finalAnswer"`
	Begin       string `yaml:"begin" json:"begin"`
}

type PromptTemplate struct {
	Language   string   `required:"true" yaml:"language" json:"language"`
	CHTemplate Template `yaml:"chTemplate" json:"chTemplate"`
	ENTemplate Template `yaml:"enTemplate" json:"enTemplate"`
}

type LLMInfo struct {
	// @Title zh-CN 大模型服务名称
	// @Description zh-CN 带服务类型的完整 FQDN 名称
	ServiceName string `required:"true" yaml:"serviceName" json:"serviceName"`
	// @Title zh-CN 大模型服务端口
	// @Description zh-CN 服务端口
	ServicePort int64 `required:"true" yaml:"servicePort" json:"servicePort"`
	// @Title zh-CN 大模型服务域名
	// @Description zh-CN 大模型服务域名，例如 dashscope.aliyuncs.com
	Domin string `required:"true" yaml:"domin" json:"domin"`
	// @Title zh-CN 大模型服务的key
	// @Description zh-CN 大模型服务的key
	APIKey string `required:"true" yaml:"apiKey" json:"apiKey"`
	// @Title zh-CN 大模型服务的请求路径
	// @Description zh-CN 大模型服务的请求路径，如"/compatible-mode/v1/chat/completions"
	Path string `required:"true" yaml:"path" json:"path"`
	// @Title zh-CN 大模型服务的模型名称
	// @Description zh-CN 大模型服务的模型名称，如"qwen-max-0403"
	Model string `required:"true" yaml:"model" json:"model"`
}

type PluginConfig struct {
	// @Title zh-CN 返回 HTTP 响应的模版
	// @Description zh-CN 用 %s 标记需要被 cache value 替换的部分
	ReturnResponseTemplate string `required:"true" yaml:"returnResponseTemplate" json:"returnResponseTemplate"`
	// @Title zh-CN 工具服务商以及工具信息
	// @Description zh-CN 用于存储工具服务商以及工具信息
	APIs      []APIs               `required:"true" yaml:"apis" json:"apis"`
	APIClient []wrapper.HttpClient `yaml:"-" json:"-"`
	// @Title zh-CN llm信息
	// @Description zh-CN 用于存储llm使用信息
	LLMInfo        LLMInfo            `required:"true" yaml:"llm" json:"llm"`
	LLMClient      wrapper.HttpClient `yaml:"-" json:"-"`
	API_Param      []API_Param        `yaml:"-" json:"-"`
	PromptTemplate PromptTemplate     `yaml:"promptTemplate" json:"promptTemplate"`
}

func parseConfig(gjson gjson.Result, c *PluginConfig, log wrapper.Log) error {
	//设置回复模板
	c.ReturnResponseTemplate = gjson.Get("returnResponseTemplate").String()
	if c.ReturnResponseTemplate == "" {
		c.ReturnResponseTemplate = `{"id":"from-cache","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	}

	//从插件配置中获取toolsClientInfo信息
	apis := gjson.Get("apis")
	if !apis.Exists() {
		return errors.New("apis is required")
	}
	if len(apis.Array()) == 0 {
		return errors.New("apis cannot be empty")
	}

	for _, item := range apis.Array() {
		serviceName := item.Get("apiProvider.serviceName")
		if !serviceName.Exists() || serviceName.String() == "" {
			return errors.New("apiProvider serviceName is required")
		}

		servicePort := item.Get("apiProvider.servicePort")
		if !servicePort.Exists() || servicePort.Int() == 0 {
			return errors.New("apiProvider servicePort is required")
		}

		domain := item.Get("apiProvider.domain")
		if !domain.Exists() || domain.String() == "" {
			return errors.New("apiProvider domain is required")
		}

		apiKeyIn := item.Get("apiProvider.apiKey.in").String()
		if apiKeyIn != "query" {
			apiKeyIn = "header"
		}

		apiKeyName := item.Get("apiProvider.apiKey.name")

		apiKeyValue := item.Get("apiProvider.apiKey.value")

		//根据多个toolsClientInfo的信息，分别初始化toolsClient
		apiClient := wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: serviceName.String(),
			Port: servicePort.Int(),
			Host: domain.String(),
		})

		c.APIClient = append(c.APIClient, apiClient)

		api := item.Get("api")
		if !api.Exists() || api.String() == "" {
			return errors.New("api is required")
		}

		var apiStrcut API
		err := yaml.Unmarshal([]byte(api.String()), &apiStrcut)
		if err != nil {
			return err
		}

		var allTool_param []Tool_Param
		//拆除服务下面的每个api的path
		for path, pathmap := range apiStrcut.Paths {
			//拆解出每个api对应的参数
			for method, submap := range pathmap {
				//把参数列表存起来
				var param Tool_Param
				param.Path = path
				param.Method = method
				param.ToolName = submap.OperationID
				paramName := make([]string, 0)
				for _, parammeter := range submap.Parameters {
					paramName = append(paramName, parammeter.Name)
				}
				param.ParamName = paramName
				out, _ := json.Marshal(submap.Parameters)
				param.Parameter = string(out)
				param.Desciption = submap.Description
				allTool_param = append(allTool_param, param)
			}
		}
		api_param := API_Param{
			APIKey:     APIKey{In: apiKeyIn, Name: apiKeyName.String(), Value: apiKeyValue.String()},
			URL:        apiStrcut.Servers[0].URL,
			Tool_Param: allTool_param,
		}

		c.API_Param = append(c.API_Param, api_param)
	}

	c.PromptTemplate.Language = gjson.Get("promptTemplate.language").String()
	if c.PromptTemplate.Language != "EN" && c.PromptTemplate.Language != "CH" {
		c.PromptTemplate.Language = "EN"
	}
	if c.PromptTemplate.Language == "EN" {
		c.PromptTemplate.ENTemplate.Question = gjson.Get("promptTemplate.enTemplate.question").String()
		if c.PromptTemplate.ENTemplate.Question == "" {
			c.PromptTemplate.ENTemplate.Question = "the input question you must answer"
		}
		c.PromptTemplate.ENTemplate.Thought1 = gjson.Get("promptTemplate.enTemplate.thought1").String()
		if c.PromptTemplate.ENTemplate.Thought1 == "" {
			c.PromptTemplate.ENTemplate.Thought1 = "you should always think about what to do"
		}
		c.PromptTemplate.ENTemplate.ActionInput = gjson.Get("promptTemplate.enTemplate.actionInput").String()
		if c.PromptTemplate.ENTemplate.ActionInput == "" {
			c.PromptTemplate.ENTemplate.ActionInput = "the input to the action"
		}
		c.PromptTemplate.ENTemplate.Observation = gjson.Get("promptTemplate.enTemplate.observation").String()
		if c.PromptTemplate.ENTemplate.Observation == "" {
			c.PromptTemplate.ENTemplate.Observation = "the result of the action"
		}
		c.PromptTemplate.ENTemplate.Thought1 = gjson.Get("promptTemplate.enTemplate.thought2").String()
		if c.PromptTemplate.ENTemplate.Thought1 == "" {
			c.PromptTemplate.ENTemplate.Thought1 = "I now know the final answer"
		}
		c.PromptTemplate.ENTemplate.FinalAnswer = gjson.Get("promptTemplate.enTemplate.finalAnswer").String()
		if c.PromptTemplate.ENTemplate.FinalAnswer == "" {
			c.PromptTemplate.ENTemplate.FinalAnswer = "the final answer to the original input question, please give the most direct answer directly in Chinese, not English, and do not add extra content."
		}
		c.PromptTemplate.ENTemplate.Begin = gjson.Get("promptTemplate.enTemplate.begin").String()
		if c.PromptTemplate.ENTemplate.Begin == "" {
			c.PromptTemplate.ENTemplate.Begin = "Begin! Remember to speak as a pirate when giving your final answer. Use lots of \"Arg\"s"
		}
	} else if c.PromptTemplate.Language == "CH" {
		c.PromptTemplate.CHTemplate.Question = gjson.Get("promptTemplate.chTemplate.question").String()
		if c.PromptTemplate.CHTemplate.Question == "" {
			c.PromptTemplate.CHTemplate.Question = "你需要回答的输入问题"
		}
		c.PromptTemplate.CHTemplate.Thought1 = gjson.Get("promptTemplate.chTemplate.thought1").String()
		if c.PromptTemplate.CHTemplate.Thought1 == "" {
			c.PromptTemplate.CHTemplate.Thought1 = "你应该总是思考该做什么"
		}
		c.PromptTemplate.CHTemplate.ActionInput = gjson.Get("promptTemplate.chTemplate.actionInput").String()
		if c.PromptTemplate.CHTemplate.ActionInput == "" {
			c.PromptTemplate.CHTemplate.ActionInput = "行动的输入，必须出现在Action后"
		}
		c.PromptTemplate.CHTemplate.Observation = gjson.Get("promptTemplate.chTemplate.observation").String()
		if c.PromptTemplate.CHTemplate.Observation == "" {
			c.PromptTemplate.CHTemplate.Observation = "行动的结果"
		}
		c.PromptTemplate.CHTemplate.Thought1 = gjson.Get("promptTemplate.chTemplate.thought2").String()
		if c.PromptTemplate.CHTemplate.Thought1 == "" {
			c.PromptTemplate.CHTemplate.Thought1 = "我现在知道最终答案"
		}
		c.PromptTemplate.CHTemplate.FinalAnswer = gjson.Get("promptTemplate.chTemplate.finalAnswer").String()
		if c.PromptTemplate.CHTemplate.FinalAnswer == "" {
			c.PromptTemplate.CHTemplate.FinalAnswer = "对原始输入问题的最终答案"
		}
		c.PromptTemplate.CHTemplate.Begin = gjson.Get("promptTemplate.chTemplate.begin").String()
		if c.PromptTemplate.CHTemplate.Begin == "" {
			c.PromptTemplate.CHTemplate.Begin = "再次重申，不要修改以上模板的字段名称，开始吧！"
		}
	}

	c.LLMInfo.APIKey = gjson.Get("llm.apiKey").String()
	c.LLMInfo.ServiceName = gjson.Get("llm.serviceName").String()
	c.LLMInfo.ServicePort = gjson.Get("llm.servicePort").Int()
	c.LLMInfo.Domin = gjson.Get("llm.domain").String()
	c.LLMInfo.Path = gjson.Get("llm.path").String()
	c.LLMInfo.Model = gjson.Get("llm.model").String()

	c.LLMClient = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: c.LLMInfo.ServiceName,
		Port: c.LLMInfo.ServicePort,
		Host: c.LLMInfo.Domin,
	})

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	log.Debug("onHttpRequestHeaders start")
	defer log.Debug("onHttpRequestHeaders end")
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
	for _, api_param := range config.API_Param {
		for _, tool_param := range api_param.Tool_Param {
			tool_desc = append(tool_desc, fmt.Sprintf(prompttpl.TOOL_DESC, tool_param.ToolName, tool_param.Desciption, tool_param.Desciption, tool_param.Desciption, tool_param.Parameter), "\n")
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
			config.PromptTemplate.ENTemplate.FinalAnswer,
			config.PromptTemplate.ENTemplate.Begin,
			query)
	}

	//将请求加入到历史对话存储器中
	dashscope.MessageStore.AddForUser(prompt)

	//开始第一次请求
	ret := firstreq(config, prompt, rawRequest, log)

	return ret
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	log.Debug("onHttpResponseHeaders start")
	defer log.Debug("onHttpResponseHeaders end")

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
		var headers [][2]string
		var apiClient wrapper.HttpClient
		var method string

		for i, api_param := range config.API_Param {
			for _, tool_param := range api_param.Tool_Param {
				if action[1] == tool_param.ToolName {
					log.Infof("calls %s\n", tool_param.ToolName)
					log.Infof("actionInput[1]: %s", actionInput[1])

					//将大模型需要的参数反序列化
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(actionInput[1]), &data); err != nil {
						log.Debugf("Error: %s\n", err.Error())
						return types.ActionContinue
					}

					var args string
					for i, param := range tool_param.ParamName { //从参数列表中取出参数
						if i == 0 {
							args = "?" + param + "=%s"
							args = fmt.Sprintf(args, data[param])
						} else {
							args = args + "&" + param + "=%s"
							args = fmt.Sprintf(args, data[param])
						}
					}

					url = api_param.URL + tool_param.Path + args

					if api_param.APIKey.Name != "" {
						if api_param.APIKey.In == "query" {
							headers = nil
							key := "&" + api_param.APIKey.Name + "=" + api_param.APIKey.Value
							url += key
						} else if api_param.APIKey.In == "header" {
							headers = [][2]string{{"Content-Type", "application/json"}, {"Authorization", api_param.APIKey.Name + " " + api_param.APIKey.Value}}
						}
					}

					log.Infof("url: %s\n", url)

					method = tool_param.Method

					apiClient = config.APIClient[i]
					break
				}
			}
		}

		if method == "get" {
			//调用工具
			err := apiClient.Get(
				url,
				headers,
				func(statusCode int, responseHeaders http.Header, responseBody []byte) {
					if statusCode != http.StatusOK {
						log.Debugf("statusCode: %d\n", statusCode)
					}
					log.Info("========函数返回结果========")
					log.Infof(string(responseBody))

					Observation := "Observation: " + string(responseBody)

					dashscope.MessageStore.AddForUser(Observation)

					// for _, v := range dashscope.MessageStore {
					// 	log.Infof("role: %s\n", v.Role)
					// 	log.Infof("Content: %s\n", v.Content)
					// }

					completion := dashscope.Completion{
						Model:    config.LLMInfo.Model,
						Messages: dashscope.MessageStore,
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

										log.Debug("[onHttpResponseBody] response替换成功")
										proxywasm.ResumeHttpResponse()
									}
								}
							} else {
								proxywasm.ResumeHttpRequest()
							}
						}, 50000)
					if err != nil {
						log.Debugf("[onHttpRequestBody] completion err: %s", err.Error())
						proxywasm.ResumeHttpRequest()
					}
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
		return toolsCall(config, rawResponse.Choices[0].Message.Content, rawResponse, log)
	} else {
		return types.ActionContinue
	}
}
