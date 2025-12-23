package workflow

import (
	"fmt"
	"strings"

	"api-workflow/utils"

	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	TaskTypeHTTP   string = "http"
	TaskStart      string = "start"
	TaskEnd        string = "end"
	TaskContinue   string = "continue"
	UseContextFlag string = "||"
	AllFlag        string = "@all"
)

type PluginConfig struct {

	// @Title zh-CN 工作流
	// @Description zh-CN 工作流的具体描述
	Workflow Workflow `json:"workflow" yaml:"workflow"`
	// @Title zh-CN 环境变量
	// @Description zh-CN 用来定义整个工作流的环境变量
	Env Env `json:"env" yaml:"env"`
}

type Env struct {
	// @Title zh-CN 超时时间
	// @Description zh-CN 用来定义工作流的超时时间，单位是毫秒
	Timeout uint32 `json:"timeout" yaml:"timeout"`
	// @Title zh-CN 最大迭代深度
	// @Description zh-CN 用来定义工作流最大的迭代深度，默认是100
	MaxDepth uint32 `json:"max_depth" yaml:"max_depth"`
}
type Workflow struct {
	// @Title zh-CN 边的列表
	// @Description zh-CN 边的列表
	Edges []Edge `json:"edges" yaml:"edges"`
	// @Title zh-CN 节点的列表
	// @Description zh-CN 节点的列表
	Nodes map[string]Node `json:"nodes" yaml:"nodes"`
	// @Title zh-CN 工作流的状态
	// @Description zh-CN 工作流的执行状态，用于记录node之间的相互依赖和执行情况
	WorkflowExecStatus map[string]int `json:"-" yaml:"-"`
}

type Edge struct {
	// @Title zh-CN 上一步节点
	// @Description zh-CN 上一步节点，必须是定义node的name，或者初始化工作流的start
	Source string `json:"source" yaml:"source"`
	// @Title zh-CN 当前执行的节点
	// @Description zh-CN 当前执行节点，必须是定义的node的name，或者结束工作流的关键字 end continue
	Target string `json:"target" yaml:"target"`
	// @Title zh-CN 执行操作
	// @Description zh-CN 执行单元，里面实时封装需要的数据
	Task *Task
	// @Title zh-CN 判断表达式
	// @Description zh-CN 是否执行下一步的判断条件
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

type Node struct {
	// @Title zh-CN 节点名称
	// @Description zh-CN 节点名称全局唯一
	Name string `json:"name" yaml:"name"`
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
	// @Description zh-CN http方法，支持所有可用方法 GET，POST等
	ServiceMethod string `json:"service_method" yaml:"service_method"`
	// @Title zh-CN http 请求头文件
	// @Description zh-CN 请求头文件
	ServiceHeaders []ServiceHeader `json:"service_headers" yaml:"service_headers"`
	// @Title zh-CN http 请求body模板
	// @Description zh-CN 请求body模板，用来构造请求
	ServiceBodyTmpl string `json:"service_body_tmpl" yaml:"service_body_tmpl"`
	// @Title zh-CN http 请求body模板替换键值对
	// @Description zh-CN 请求body模板替换键值对，用来构造请求。to表示填充的位置，from表示数据从哪里，
	// 标识表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	ServiceBodyReplaceKeys []BodyReplaceKeyPair `json:"service_body_replace_keys" yaml:"service_body_replace_keys"`
}
type BodyReplaceKeyPair struct {
	// @Title zh-CN from表示数据从哪里，
	// @Description zh-CN from表示数据从哪里
	// 标识表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	From string `json:"from" yaml:"from"`
	// @Title zh-CN to表示填充的位置
	// @Description zh-CN to表示填充的位置，
	// 标识表达式基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	To string `json:"to" yaml:"to"`
}
type ServiceHeader struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}

func (w *Edge) IsEnd() bool {
	if w.Target == TaskEnd {
		return true
	}
	return false
}
func (w *Edge) IsContinue() bool {
	if w.Target == TaskContinue {
		return true
	}
	return false
}
func (e *Edge) IsPass(ctx wrapper.HttpContext) (bool, error) {
	// 执行判断Conditional
	if e.Conditional != "" {

		var err error
		// 获取模板里的表达式

		e.Conditional, err = e.WrapperDataByTmplStr(e.Conditional, ctx)
		if err != nil {
			return false, fmt.Errorf("workflow WrapperDateByTmplStr %s failed: %v", e.Conditional, err)
		}
		ok, err := e.ExecConditional()
		if err != nil {

			return false, fmt.Errorf("wl exec conditional %s failed: %v", e.Conditional, err)
		}
		return !ok, nil

	}
	return false, nil
}

func (w *Edge) WrapperTask(config PluginConfig, ctx wrapper.HttpContext) error {

	// 判断 node 是否存在
	node, isTool := config.Workflow.Nodes[w.Target]

	if isTool {
		w.Task.TaskType = TaskTypeHTTP
	} else {
		return fmt.Errorf("do not find target :%s", w.Target)
	}

	switch w.Task.TaskType {
	default:
		return fmt.Errorf("unknown node type :%s", w.Task.TaskType)
	case TaskTypeHTTP:
		err := w.wrapperNodeTask(node, ctx)
		if err != nil {
			return err
		}

	}
	return nil

}

func (w *Edge) wrapperBody(requestBodyTemplate string, keyPairs []BodyReplaceKeyPair, ctx wrapper.HttpContext) error {

	requestBody, err := w.WrapperDataByTmplStrAndKeys(requestBodyTemplate, keyPairs, ctx)
	if err != nil {
		return fmt.Errorf("wrapper date by tmpl str is %s ,find  err: %v", requestBodyTemplate, err)
	}

	w.Task.Body = requestBody
	return nil
}

func (w *Edge) wrapperNodeTask(node Node, ctx wrapper.HttpContext) error {
	// 封装cluster
	w.Task.Cluster = wrapper.FQDNCluster{
		Host: node.ServiceDomain,
		FQDN: node.ServiceName,
		Port: node.ServicePort,
	}

	// 封装请求body
	err := w.wrapperBody(node.ServiceBodyTmpl, node.ServiceBodyReplaceKeys, ctx)
	if err != nil {
		return fmt.Errorf("wrapper body parse failed: %v", err)
	}

	// 封装请求Method path headers
	w.Task.Method = node.ServiceMethod
	w.Task.ServicePath = node.ServicePath
	w.Task.Headers = make([][2]string, 0)
	if len(node.ServiceHeaders) > 0 {
		for _, header := range node.ServiceHeaders {
			w.Task.Headers = append(w.Task.Headers, [2]string{header.Key, header.Value})
		}
	}

	return nil
}

// 利用模板和替换键值对构造请求，使用`||`分隔，str1代表使用node是执行结果。tr2代表如何取数据，使用gjson的表达式，`@all`代表全都要
func (w *Edge) WrapperDataByTmplStrAndKeys(tmpl string, keyPairs []BodyReplaceKeyPair, ctx wrapper.HttpContext) ([]byte, error) {
	var err error
	// 不需要替换 node.service_body_replace_keys 为空
	if len(keyPairs) == 0 {
		return []byte(tmpl), nil
	}

	for _, keyPair := range keyPairs {

		jsonPath := keyPair.From
		target := keyPair.To
		var contextValueRaw []byte
		// 获取上下文数据
		if strings.Contains(jsonPath, UseContextFlag) {
			pathStr := strings.Split(jsonPath, UseContextFlag)
			if len(pathStr) == 2 {
				contextKey := pathStr[0]
				contextBody := ctx.GetContext(contextKey)
				if contextValue, ok := contextBody.([]byte); ok {
					contextValueRaw = contextValue
					jsonPath = pathStr[1]
				} else {
					return nil, fmt.Errorf("context value is not []byte,key is %s", contextKey)
				}
			}
		}

		// 执行封装 ， `@all`代表全都要
		requestBody := gjson.ParseBytes(contextValueRaw)
		if jsonPath == AllFlag {

			tmpl, err = sjson.SetRaw(tmpl, target, requestBody.Raw)
			if err != nil {
				return nil, fmt.Errorf("wrapper body parse failed: %v", err)
			}
			continue
		}
		requestBodyJson := requestBody.Get(jsonPath)
		if requestBodyJson.Exists() {
			tmpl, err = sjson.SetRaw(tmpl, target, requestBodyJson.Raw)
			if err != nil {
				return nil, fmt.Errorf("wrapper body parse failed: %v", err)
			}

		} else {
			return nil, fmt.Errorf("wrapper body parse failed: not exists %s", jsonPath)
		}
	}
	return []byte(tmpl), nil

}

// 变量使用`{{str1||str2}}`包裹，使用`||`分隔，str1代表使用node是执行结果。tr2代表如何取数据，使用gjson的表达式，`@all`代表全都要
func (w *Edge) WrapperDataByTmplStr(tmpl string, ctx wrapper.HttpContext) (string, error) {
	var body []byte
	// 获取模板里的表达式
	TmplKeyAndPath := utils.ParseTmplStr(tmpl)
	if len(TmplKeyAndPath) == 0 {
		return tmpl, nil
	}
	// 解析表达式 { "{{str1||str2}}":"str1||str2" }
	for k, path := range TmplKeyAndPath {
		// 变量使用`{{str1||str2}}`包裹，使用`||`分隔，str1代表使用前面命名为name的数据()。
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
			// 执行封装 ， `@all`代表全都要
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
		} else {
			return "", fmt.Errorf("tmpl parse find error: || is not exists %s", path)
		}

	}
	return tmpl, nil
}

func (w *Edge) ExecConditional() (bool, error) {

	ConditionalResult, err := utils.ExecConditionalStr(w.Conditional)
	if err != nil {
		return false, fmt.Errorf("exec conditional failed: %v", err)
	}
	return ConditionalResult, nil

}
