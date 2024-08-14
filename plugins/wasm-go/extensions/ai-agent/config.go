package main

import (
	"encoding/json"
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v2"
)

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

func initResponsePromptTpl(gjson gjson.Result, c *PluginConfig) {
	//设置回复模板
	c.ReturnResponseTemplate = gjson.Get("returnResponseTemplate").String()
	if c.ReturnResponseTemplate == "" {
		c.ReturnResponseTemplate = `{"id":"from-cache","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	}
}

func initAPIs(gjson gjson.Result, c *PluginConfig) error {
	//从插件配置中获取apis信息
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
	return nil
}

func initReActPromptTpl(gjson gjson.Result, c *PluginConfig) {
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
}

func initLLMClient(gjson gjson.Result, c *PluginConfig) {
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
}
