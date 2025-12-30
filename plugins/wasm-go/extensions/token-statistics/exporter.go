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
	"fmt"

	"github.com/google/martian/log"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

type LogExporter struct {
	level string
}

type PrometheusExporter struct {
	namespace string
	subsystem string
	model     string
}

func NewPrometheusExporter(namespace, subsystem, model string) *PrometheusExporter {
	return &PrometheusExporter{
		namespace: namespace,
		subsystem: subsystem,
		model:     model,
	}
}

func (p *PrometheusExporter) Export(usage *TokenUsage) {
	// 创建指标名称
	inputMetricName := fmt.Sprintf("%s_%s_%s_input_tokens_total", p.namespace, p.subsystem, p.model)
	outputMetricName := fmt.Sprintf("%s_%s_%s_output_tokens_total", p.namespace, p.subsystem, p.model)
	totalMetricName := fmt.Sprintf("%s_%s_%s_total_tokens_total", p.namespace, p.subsystem, p.model)

	// 定义并更新指标
	inputCounter := proxywasm.DefineCounterMetric(inputMetricName)
	outputCounter := proxywasm.DefineCounterMetric(outputMetricName)
	totalCounter := proxywasm.DefineCounterMetric(totalMetricName)

	inputCounter.Increment(uint64(usage.InputTokens))
	outputCounter.Increment(uint64(usage.OutputTokens))
	totalCounter.Increment(uint64(usage.TotalTokens))
}

// 日志输出

func (l *LogExporter) Export(ctx wrapper.HttpContext, model string, usage *TokenUsage) {
	log.Infof("Token usage statistics: model=%s, input_tokens=%d, output_tokens=%d, total_tokens=%d",
		model, usage.InputTokens, usage.OutputTokens, usage.TotalTokens)
}
