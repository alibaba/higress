// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	ejson "encoding/json"
	"errors"
	"fmt"
	. "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-workflow/workflow"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"net/http"
)

const (
	maxDepth           uint   = 100
	WorkflowExecStatus string = "workflowExecStatus"
)

func main() {
	wrapper.SetCtx(
		"api-workflow",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

func parseConfig(json gjson.Result, c *PluginConfig, log wrapper.Log) error {

	edges := make([]Edge, 0)
	nodes := make(map[string]Node)
	var err error

	workflow := json.Get("workflow")
	if !workflow.Exists() {
		return errors.New("workflow is empty")
	}
	//workflow.edges
	edges_ := workflow.Get("edges")
	if edges_.Exists() && edges_.IsArray() {
		for _, w := range edges_.Array() {
			task := Task{}
			edge := Edge{}
			edge.Source = w.Get("source").String()
			if edge.Source == "" {
				return errors.New("source is empty")
			}
			edge.Target = w.Get("target").String()
			if edge.Target == "" {
				return errors.New("target is empty")
			}
			edge.Task = &task

			edge.Conditional = w.Get("conditional").String()
			edges = append(edges, edge)
		}
	}
	c.Workflow.Edges = edges

	nodes_ := workflow.Get("nodes")
	if nodes_.Exists() && nodes_.IsArray() {

		for _, value := range nodes_.Array() {
			node := Node{}
			node.Name = value.Get("name").String()
			if node.Name == "" {
				return errors.New("tool name is empty")
			}
			node.ServiceType = value.Get("service_type").String()
			if node.ServiceType == "" {
				return errors.New("tool service type is empty")
			}
			node.ServiceName = value.Get("service_name").String()
			if node.ServiceName == "" {
				return errors.New("tool service name is empty")
			}
			node.ServicePort = value.Get("service_port").Int()
			if node.ServicePort == 0 {
				if node.ServiceType == ToolServiceTypeStatic {
					node.ServicePort = 80
				} else {
					return errors.New("tool service port is empty")
				}

			}
			node.ServiceDomain = value.Get("service_domain").String()
			node.ServicePath = value.Get("service_path").String()
			if node.ServicePath == "" {
				node.ServicePath = "/"
			}
			node.ServiceMethod = value.Get("service_method").String()
			if node.ServiceMethod == "" {
				return errors.New("service_method is empty")
			}
			serviceHeaders := value.Get("service_headers")
			if serviceHeaders.Exists() && serviceHeaders.IsArray() {
				serviceHeaders_ := []ServiceHeader{}
				err = ejson.Unmarshal([]byte(serviceHeaders.Raw), &serviceHeaders_)
				node.ServiceHeaders = serviceHeaders_
			}

			node.ServiceBodyTmpl = value.Get("service_body_tmpl").String()
			serviceBodyReplaceKeys := value.Get("service_body_replace_keys")
			if serviceBodyReplaceKeys.Exists() && serviceBodyReplaceKeys.IsArray() {
				serviceBodyReplaceKeys_ := []BodyReplaceKeyPair{}
				err = ejson.Unmarshal([]byte(serviceBodyReplaceKeys.Raw), &serviceBodyReplaceKeys_)
				node.ServiceBodyReplaceKeys = serviceBodyReplaceKeys_
				if err != nil {
					return fmt.Errorf("unmarshal service body replace keys failed, err:%v", err)
				}
			}

			nodes[node.Name] = node
		}
		c.Workflow.Nodes = nodes
	}
	log.Debugf("config : %v", c)
	return nil
}

func initWorkflowExecStatus(config PluginConfig) (map[string]int, error) {
	result := make(map[string]int)

	for name, _ := range config.Workflow.Nodes {
		result[name] = 0
	}
	for _, edge := range config.Workflow.Edges {

		if edge.Source == TaskStart || edge.Target == TaskContinue || edge.Target == TaskEnd {
			continue
		}

		count, ok := result[edge.Target]
		if !ok {
			return nil, fmt.Errorf("Target %s is not exist in nodes", edge.Target)
		}
		result[edge.Target] = count + 1

	}
	return result, nil
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {

	initHeader := make([][2]string, 0)
	//初始化运行状态
	workflowExecStatus, err := initWorkflowExecStatus(config)
	log.Errorf("init status : %v", workflowExecStatus)
	if err != nil {
		log.Errorf("init workflow exec status failed, err:%v", err)
		return types.ActionContinue
	}
	ctx.SetContext(WorkflowExecStatus, workflowExecStatus)

	//执行工作流
	for _, edge := range config.Workflow.Edges {

		if edge.Source == TaskStart {
			ctx.SetContext(fmt.Sprintf("%s", TaskStart), body)
			err := recursive(edge, initHeader, body, 1, maxDepth, config, log, ctx)
			if err != nil {
				log.Errorf("recursive failed: %v", err)
			}
		}
	}

	return types.ActionPause
}

// 放入符合条件的edge
func recursive(edge Edge, headers [][2]string, body []byte, depth uint, maxDepth uint, config PluginConfig, log wrapper.Log, ctx wrapper.HttpContext) error {

	var err error
	// 防止递归次数太多
	if depth > maxDepth {
		return fmt.Errorf("maximum recursion depth reached")
	}

	//判断是不是end
	if edge.IsEnd() {
		log.Debugf("workflow is end")
		log.Debugf("body is %s", string(body))
		proxywasm.SendHttpResponse(200, headers, body, -1)
		return nil
	}
	//判断是不是continue
	if edge.IsContinue() {
		log.Debugf("workflow is continue")
		proxywasm.ResumeHttpRequest()
		return nil
	}

	// 封装task
	err = edge.WrapperTask(config, ctx)
	if err != nil {
		log.Errorf("workflow exec wrapperTask find error,source is %s,target is %s,error is %v ", edge.Source, edge.Target, err)
		return fmt.Errorf("workflow exec wrapperTask find error,source is %s,target is %s,error is %v ", edge.Source, edge.Target, err)
	}

	//执行task
	log.Debugf("workflow exec task,source is %s,target is %s, body is %s,header is %v", edge.Source, edge.Target, string(edge.Task.Body), edge.Task.Headers)
	err = wrapper.HttpCall(edge.Task.Cluster, edge.Task.Method, edge.Task.ServicePath, edge.Task.Headers, edge.Task.Body, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Debugf("code:%d", statusCode)
		//判断response code
		if statusCode < 400 {

			//存入 这轮返回的body
			ctx.SetContext(fmt.Sprintf("%s", edge.Target), responseBody)

			headers_ := make([][2]string, len(responseHeaders))
			for key, value := range responseHeaders {
				headers_ = append(headers_, [2]string{key, value[0]})
			}
			//判断是否进入下一步
			nextStatus := ctx.GetContext(WorkflowExecStatus).(map[string]int)

			//进入下一步
			for _, next := range config.Workflow.Edges {
				if next.Source == edge.Target {
					//更新workflow status
					if next.Target != TaskContinue && next.Target != TaskEnd {

						nextStatus[next.Target] = nextStatus[next.Target] - 1
						log.Debugf("======source is %s,target is %s,stauts is %v", next.Source, next.Target, nextStatus)
						// 还有没执行完的边
						if nextStatus[next.Target] > 0 {
							ctx.SetContext(WorkflowExecStatus, nextStatus)
							return
						}
						// 执行出了问题
						if nextStatus[next.Target] < 0 {
							log.Errorf("workflow exec status find  error  %v", nextStatus)
							proxywasm.ResumeHttpRequest()
							return
						}
					}
					//判断是否执行
					isPass, err2 := next.IsPass(ctx)
					if err2 != nil {
						log.Errorf("check pass find error:%v", err2)
						proxywasm.ResumeHttpRequest()
						return
					}
					if isPass {
						log.Debugf("workflow is pass ")
						nextStatus := ctx.GetContext(WorkflowExecStatus).(map[string]int)
						nextStatus[next.Target] = nextStatus[next.Target] - 1
						ctx.SetContext(WorkflowExecStatus, nextStatus)
						continue

					}

					//执行下一步
					err = recursive(next, headers_, responseBody, depth+1, maxDepth, config, log, ctx)
					if err != nil {
						log.Errorf("recursive error:%v", err)
						proxywasm.ResumeHttpRequest()
						return
					}
				}
			}

		} else {
			//statusCode >= 400 ,task httpCall执行失败，放行请求，打印错误，结束workflow
			log.Errorf("workflow exec task find error,code is %d,body is %s", statusCode, string(responseBody))
			proxywasm.ResumeHttpRequest()
		}
		return

	}, uint32(maxDepth)*5000)
	if err != nil {
		log.Errorf("httpcall error:%v", err)
	}

	return err
}
