package workflow

import (
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-workflow/utils"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"strings"
)

const (
	TaskTypeModel         string = "model"
	TaskTypeTool          string = "tool"
	TaskStart             string = "start"
	TaskEnd               string = "end"
	TaskContinue          string = "continue"
	ToolServiceTypeStatic string = "static"
	ToolServiceTypeDomain string = "domain"
	ModelTypeLLM          string = "llm"
	ModelTypeEmbeddings   string = "embeddings"
	ModelTypeRerank       string = "rerank"
	ModelTypeImage        string = "image"
	ModelTypeAudio        string = "audio"
	UseContextFlag        string = "||"
	AllFlag               string = "@all"
)

type PluginConfig struct {
	// @Title zh-CN 工具集
	// @Description zh-CN 工作流里可用的工具
	Tools map[string]Tool `json:"tools" yaml:"tools"`
	// @Title zh-CN 工作流
	// @Description zh-CN 工作流的具体描述
	DSL DSL `json:"dsl" yaml:"dsl"`
}

type DSL struct {
	// @Title zh-CN 工作列表
	// @Description zh-CN 工作列表
	WorkFlow []WorkFlow `json:"workflow" yaml:"workflow"`
}

type WorkFlow struct {
	// @Title zh-CN 上一步的操作
	// @Description zh-CN 上一步的操作，必须是定义的model或者tool的name，或者初始化工作流的start
	Source string `json:"source" yaml:"source"`
	// @Title zh-CN 当前的操作
	// @Description zh-CN 当前的操作，必须是定义的model或者tool的name，或者结束工作流的关键字 end continue
	Target string `json:"target" yaml:"target"`
	// @Title zh-CN 执行单元
	// @Description zh-CN 执行单元，里面实时封装需要的数据
	Task *Task
	// @Title zh-CN 进入的本轮操作数据的过滤方式
	// @Description zh-CN 进入的本轮操作数据过滤方式，为空就不过滤。使用标识表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	Input string `json:"input" yaml:"input"`
	// @Title zh-CN 流出的本轮操作数据的过滤方式
	// @Description zh-CN 流出的本轮操作数据过滤方式，为空就不过滤。使用标识表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	Output string `json:"output" yaml:"output"`
	// @Title zh-CN 判断表达式
	// @Description zh-CN 流出的本轮操作数据的过滤方式这一步是否执行的判断条件
	Conditional string `json:"conditional" yaml:"conditional"`
}
type Task struct {
	Cluster     wrapper.Cluster `json:"-" yaml:"-"`
	ServicePath string          `json:"service_path" yaml:"service_path"`
	ServicePort int64           `json:"service_port" yaml:"service_port"`
	ServiceKey  string          `json:"service_key" yaml:"service_key"`
	Body        []byte          `json:"-" yaml:"-"`
	Headers     [][2]string     `json:"headers" yaml:"headers"`
	Method      string          `json:"method" yaml:"method"`
	TaskType    string          `json:"task_type" yaml:"task_type"`
}

type Tool struct {
	Name string `json:"name" yaml:"name"`
	// @Title zh-CN 服务类型
	// @Description zh-CN 支持两个值 static domain 对于固定ip地址和域名
	ServiceType string `json:"service_type" yaml:"service_type"`
	// @Title zh-CN 服务名称
	// @Description zh-CN 带服务类型的完整名称，例如 my.dns or foo.static
	ServiceName string `json:"service_name" yaml:"service_name"`
	// @Title zh-CN 服务端口
	// @Description zh-CN static类型默认是80
	ServicePort int64 `json:"service_port" yaml:"service_port"`
	// @Title zh-CN 服务域名
	// @Description zh-CN 服务域名，例如 dashscope.aliyuncs.com
	ServiceDomain string `json:"service_domain" yaml:"service_domain"`
	// @Title zh-CN http访问路径
	// @Description zh-CN http访问路径，默认是 /
	ServicePath string `json:"service_path" yaml:"service_path"`
	// @Title zh-CN http 方法
	// @Description zh-CN http方法，支持所有可用方法 GET，
	ServiceMethod string `json:"service_method" yaml:"service_method"`
	// @Title zh-CN http 请求头文件
	// @Description zh-CN 请求头文件
	ServiceHeaders [][2]string `json:"service_headers" yaml:"service_headers"`
	// @Title zh-CN http 请求body模板
	// @Description zh-CN 请求body模板，用来构造请求
	ServiceBodyTmpl string `json:"service_body_tmpl" yaml:"service_body_tmpl"`
	// @Title zh-CN http 请求body模板替换键值对
	// @Description zh-CN 请求body模板替换键值对，用来构造请求。前面一个表示填充的位置，后面一个标识数据从哪里，
	//标识表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	ServiceBodyReplaceKeys [][2]string `json:"service_body_replace_keys" yaml:"service_body_replace_keys"`
}

func (w *WorkFlow) IsEnd() bool {
	if w.Target == TaskEnd {
		return true
	}
	return false
}
func (w *WorkFlow) IsContinue() bool {
	if w.Target == TaskContinue {
		return true
	}
	return false
}

func (w *WorkFlow) WrapperTask(config PluginConfig, ctx wrapper.HttpContext) error {

	//判断 tool 是否存在
	tool, isTool := config.Tools[w.Target]

	if isTool {
		w.Task.TaskType = TaskTypeTool
	} else {
		return fmt.Errorf("do not find target :%s", w.Target)
	}

	switch w.Task.TaskType {
	default:
		return fmt.Errorf("unknown task type :%s", w.Task.TaskType)
	case TaskTypeTool:
		err := w.wrapperToolTask(tool, ctx)
		if err != nil {
			return err
		}
	case TaskTypeModel:
		//todo 下一步添加 官方 model的封装
		return fmt.Errorf("task type %s is not allow now ", w.Task.TaskType)
	}
	return nil

}

func (w *WorkFlow) wrapperBody(requestBodyTemplate string, keyPairs [][2]string, ctx wrapper.HttpContext) error {

	requestBody, err := w.WrapperDataByTmplStrAndKeys(requestBodyTemplate, keyPairs, ctx)
	if err != nil {
		return fmt.Errorf("wrapper date by tmpl str is %s ,find  err: %v", requestBodyTemplate, err)
	}

	w.Task.Body = requestBody
	return nil
}

func (w *WorkFlow) wrapperToolTask(tool Tool, ctx wrapper.HttpContext) error {
	// 封装cluster
	switch tool.ServiceType {
	default:
		return fmt.Errorf("unknown tool type: %s", tool.ServiceType)
	case ToolServiceTypeStatic:
		w.Task.Cluster = wrapper.FQDNCluster{
			FQDN: tool.ServiceName,
			Port: tool.ServicePort,
		}
	case ToolServiceTypeDomain:
		w.Task.Cluster = wrapper.DnsCluster{
			ServiceName: tool.ServiceName,
			Domain:      tool.ServiceDomain,
			Port:        tool.ServicePort,
		}
	}
	//封装请求body
	err := w.wrapperBody(tool.ServiceBodyTmpl, tool.ServiceBodyReplaceKeys, ctx)
	if err != nil {
		return fmt.Errorf("wrapper body parse failed: %v", err)
	}

	//封装请求Method path headers
	w.Task.Method = tool.ServiceMethod
	w.Task.ServicePath = tool.ServicePath
	w.Task.Headers = tool.ServiceHeaders

	return nil
}

/*
利用模板和替换键值对构造请求
*/

func (w *WorkFlow) WrapperDataByTmplStrAndKeys(tmpl string, keyPairs [][2]string, ctx wrapper.HttpContext) ([]byte, error) {
	var err error
	//不需要替换
	if len(keyPairs) == 0 {
		return []byte(tmpl), nil
	}

	for _, keyPair := range keyPairs {

		target := keyPair[0]
		path := keyPair[1]
		var contextValueRaw []byte
		//获取上下文数据
		if strings.Contains(path, UseContextFlag) {
			pathStr := strings.Split(path, UseContextFlag)
			if len(pathStr) == 2 {
				contextKey := pathStr[0]
				contextBody := ctx.GetContext(contextKey)
				if contextValue, ok := contextBody.([]byte); ok {
					contextValueRaw = contextValue
					path = pathStr[1]
				} else {
					return nil, fmt.Errorf("context value is not []byte,key is %s", contextKey)
				}
			}
		}

		//执行封装 ， `@all`代表全都要
		requestBody := gjson.ParseBytes(contextValueRaw)
		if path == AllFlag {

			tmpl, err = sjson.SetRaw(tmpl, target, requestBody.Raw)
			if err != nil {
				return nil, fmt.Errorf("wrapper body parse failed: %v", err)
			}
			continue
		}
		requestBodyJson := requestBody.Get(path)
		if requestBodyJson.Exists() {
			tmpl, err = sjson.SetRaw(tmpl, target, requestBodyJson.Raw)
			if err != nil {
				return nil, fmt.Errorf("wrapper body parse failed: %v", err)
			}

		} else {
			return nil, fmt.Errorf("wrapper body parse failed: not exists %s", path)
		}
	}
	return []byte(tmpl), nil

}

/*
变量使用`{{str1||str2}}`包裹，使用`||`分隔，str1代表使用前面命名为name的操作(tool/model)。`name-input`或者`name-output` 代表它的输入和输出，str2代表如何取数据，使用gjson的表达式，`@all`代表全都要
*/
func (w *WorkFlow) WrapperDataByTmplStr(tmpl string, body []byte, ctx wrapper.HttpContext) (string, error) {
	//获取模板里的表达式
	TmplKeyAndPath := utils.ParseTmplStr(tmpl)
	if len(TmplKeyAndPath) == 0 {
		return tmpl, nil
	}
	//解析表达式
	for k, path := range TmplKeyAndPath {
		//判断是否需要使用上下文数据
		//变量使用`{{str1||str2}}`包裹，使用`||`分隔，str1代表使用前面命名为name的数据(tool.name/model.name)。
		if strings.Contains(path, UseContextFlag) {
			pathStr := strings.Split(path, UseContextFlag)
			if len(pathStr) == 2 {
				contextKey := pathStr[0]
				contextBody := ctx.GetContext(contextKey)
				if contextValue, ok := contextBody.([]byte); ok {
					body = contextValue
					path = pathStr[1]
				} else {
					return tmpl, fmt.Errorf("context value is not []byte,key is %s", contextKey)
				}
			}
		}
		//执行封装 ， `@all`代表全都要
		requestBody := gjson.ParseBytes(body)
		if path == AllFlag {
			tmpl = strings.Replace(tmpl, k, utils.TrimQuote(requestBody.Raw), -1)
			continue
		}
		requestBodyJson := requestBody.Get(path)
		if requestBodyJson.Exists() {
			tmpl = utils.ReplacedStr(tmpl, map[string]string{k: utils.TrimQuote(requestBodyJson.Raw)})
		} else {
			return tmpl, fmt.Errorf("use path {{%s}} get value is not exists,json is:%s", path, requestBody.Raw)
		}
	}
	return tmpl, nil
}

func (w *WorkFlow) ExecConditional() (bool, error) {

	ConditionalResult, err := utils.ExecConditionalStr(w.Conditional)
	if err != nil {
		return false, fmt.Errorf("exec conditional failed: %v", err)
	}
	return ConditionalResult, nil

}
