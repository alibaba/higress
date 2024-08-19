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
	maxDepth uint = 100
)

func main() {
	wrapper.SetCtx(
		"ai-workflow",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

func parseConfig(json gjson.Result, c *PluginConfig, log wrapper.Log) error {

	workflows := make([]WorkFlow, 0)
	tools := make(map[string]Tool)

	dsl := json.Get("dsl")
	if !dsl.Exists() {
		return errors.New("dsl is empty")
	}
	//处理dsl.workflow
	workFlows_ := dsl.Get("workflow")
	if workFlows_.Exists() && workFlows_.IsArray() {
		for _, w := range workFlows_.Array() {
			task := Task{}
			workflow := WorkFlow{}
			workflow.Source = w.Get("source").String()
			if workflow.Source == "" {
				return errors.New("source is empty")
			}
			workflow.Target = w.Get("target").String()
			if workflow.Target == "" {
				return errors.New("target is empty")
			}
			workflow.Task = &task
			//workflow.Context = make(map[string]string)
			workflow.Input = w.Get("input").String()
			workflow.Output = w.Get("output").String()
			workflow.Conditional = w.Get("conditional").String()
			workflows = append(workflows, workflow)
		}
	}
	c.DSL.WorkFlow = workflows

	//处理tools
	tools_ := json.Get("tools")
	if tools_.Exists() && tools_.IsArray() {

		for _, value := range tools_.Array() {
			tool := Tool{}
			tool.Name = value.Get("name").String()
			if tool.Name == "" {
				return errors.New("tool name is empty")
			}
			tool.ServiceType = value.Get("service_type").String()
			if tool.ServiceType == "" {
				return errors.New("tool service type is empty")
			}
			tool.ServiceName = value.Get("service_name").String()
			if tool.ServiceName == "" {
				return errors.New("tool service name is empty")
			}
			tool.ServicePort = value.Get("service_port").Int()
			if tool.ServicePort == 0 {
				if tool.ServiceType == ToolServiceTypeStatic {
					tool.ServicePort = 80
				} else {
					return errors.New("tool service port is empty")
				}

			}
			tool.ServiceDomain = value.Get("service_domain").String()
			tool.ServicePath = value.Get("service_path").String()
			if tool.ServicePath == "" {
				tool.ServicePath = "/"
			}
			tool.ServiceMethod = value.Get("service_method").String()
			if tool.ServiceMethod == "" {
				return errors.New("service_method is empty")
			}
			serviceHeaders := value.Get("service_headers")
			if serviceHeaders.Exists() && serviceHeaders.IsArray() {
				tool.ServiceHeaders = make([][2]string, 0)
				for _, serviceHeader := range serviceHeaders.Array() {
					if serviceHeader.IsArray() && len(serviceHeader.Array()) == 2 {
						kv := serviceHeader.Array()
						tool.ServiceHeaders = append(tool.ServiceHeaders, [2]string{kv[0].String(), kv[1].String()})
					} else {
						return errors.New("service_headers is not allow")
					}

				}
			}
			tool.ServiceBodyTmpl = value.Get("service_body_tmpl").String()
			serviceBodyReplaceKeys := value.Get("service_body_replace_keys")
			if serviceBodyReplaceKeys.Exists() && serviceBodyReplaceKeys.IsArray() {
				tool.ServiceBodyReplaceKeys = make([][2]string, 0)
				for _, serviceBodyReplaceKey := range serviceBodyReplaceKeys.Array() {
					if serviceBodyReplaceKey.IsArray() && len(serviceBodyReplaceKey.Array()) == 2 {
						keys := serviceBodyReplaceKey.Array()
						tool.ServiceBodyReplaceKeys = append(tool.ServiceBodyReplaceKeys, [2]string{keys[0].String(), keys[1].String()})
					} else {
						return errors.New("service body replace keys is not allow")
					}
				}
			}
			tools[tool.Name] = tool
		}
		c.Tools = tools
	}
	log.Debugf("config : %v", c)
	return nil
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {

	initHeader := make([][2]string, 0)
	err := recursive(config.DSL.WorkFlow, initHeader, body, 1, maxDepth, config, log, ctx)
	if err != nil {
		log.Errorf("recursive failed: %v", err)
	}
	return types.ActionPause
}

func recursive(workflows []WorkFlow, headers [][2]string, body []byte, depth uint, maxDepth uint, config PluginConfig, log wrapper.Log, ctx wrapper.HttpContext) error {

	var err error
	// 防止递归次数太多
	if depth > maxDepth {
		return fmt.Errorf("maximum recursion depth reached")
	}
	step := depth - 1

	log.Debugf("workflow is %v", workflows[step])
	workflow := workflows[step]

	// 执行判断Conditional
	if workflow.Conditional != "" {
		//填充Conditional
		workflow.Conditional, err = workflow.WrapperDataByTmplStr(workflow.Conditional, body, ctx)
		if err != nil {
			log.Errorf("workflow WrapperDateByTmplStr %s failed: %v", workflow.Conditional, err)
			return fmt.Errorf("workflow WrapperDateByTmplStr %s failed: %v", workflow.Conditional, err)
		}
		log.Debugf("Exec Conditional is %s", workflow.Conditional)
		ok, err := workflow.ExecConditional()
		if err != nil {
			log.Errorf("wl exec conditional %s failed: %v", workflow.Conditional, err)
			return fmt.Errorf("wl exec conditional %s failed: %v", workflow.Conditional, err)
		}
		//如果不通过直接跳过这步
		if !ok {
			log.Debugf("workflow is pass")
			err = recursive(workflows, headers, body, depth+1, maxDepth, config, log, ctx)
			if err != nil {

				return err
			}
			return nil
		}
	}
	//判断是不是end
	if workflow.IsEnd() {
		log.Debugf("workflow is end")
		log.Debugf("body is %s", string(body))
		proxywasm.SendHttpResponse(200, headers, body, -1)
		return nil
	}
	//判断是不是continue
	if workflow.IsContinue() {
		log.Debugf("workflow is continue")
		proxywasm.ResumeHttpRequest()
		return nil
	}

	// 过滤input
	if workflow.Input != "" {
		inputJson := gjson.GetBytes(body, workflow.Input)
		if inputJson.Exists() {
			body = []byte(inputJson.Raw)
		} else {
			return fmt.Errorf("input filter get path %s is not found,json is  %s", workflow.Input, string(body))
		}
	}
	// 存入这轮请求的body
	ctx.SetContext(fmt.Sprintf("%s-input", workflow.Target), body)
	// 封装task
	err = workflow.WrapperTask(config, ctx)
	if err != nil {
		log.Errorf("workflow exec wrapperTask find error,source is %s,target is %s,error is %v ", workflow.Source, workflow.Target, err)
		return fmt.Errorf("workflow exec wrapperTask find error,source is %s,target is %s,error is %v ", workflow.Source, workflow.Target, err)
	}

	//执行task
	log.Debugf("workflow exec task,source is %s,target is %s, body is %s,header is %v", workflow.Source, workflow.Target, string(workflow.Task.Body), workflow.Task.Headers)
	err = wrapper.HttpCall(workflow.Task.Cluster, workflow.Task.Method, workflow.Task.ServicePath, workflow.Task.Headers, workflow.Task.Body, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Debugf("code:%d", statusCode)
		//判断response code
		if statusCode < 400 {
			if workflow.Output != "" {
				out := gjson.GetBytes(responseBody, workflow.Output)
				if out.Exists() {
					responseBody = []byte(out.Raw)
				} else {
					log.Errorf("workflow get path %s exec response body %s not found", workflow.Output, string(responseBody))
					proxywasm.ResumeHttpRequest()
					return
				}
			}
			//存入 这轮返回的body
			ctx.SetContext(fmt.Sprintf("%s-output", workflow.Target), responseBody)

			headers_ := make([][2]string, len(responseHeaders))
			for key, value := range responseHeaders {
				headers_ = append(headers_, [2]string{key, value[0]})
			}
			//进入下一步
			log.Debugf("workflow exec response body %s ", string(responseBody))
			err = recursive(workflows, headers_, responseBody, depth+1, maxDepth, config, log, ctx)

			if err != nil {
				log.Errorf("recursive error:%v", err)
				proxywasm.ResumeHttpRequest()
				return
			}
		} else {
			//statusCode >= 400 ,task httpCall执行失败，放行请求，打印错误，结束workflow
			log.Errorf("workflow exec task find error,code is %d,body is %s", statusCode, string(responseBody))
			proxywasm.ResumeHttpRequest()
		}
		return

	}, uint32(maxDepth-step)*5000)
	if err != nil {
		log.Errorf("httpcall error:%v", err)
	}

	return err
}
