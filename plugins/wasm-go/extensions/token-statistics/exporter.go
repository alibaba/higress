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
	"sync"

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

// metricCounter defines the minimal interface we need from metrics.
type metricCounter interface {
	Increment(uint64)
}

// noopCounter is a no-op implementation used in tests or non-wasm environments.
type noopCounter struct{}

func (n noopCounter) Increment(_ uint64) {}

var (
	// 缓存命中计数器
	hitCounter metricCounter = noopCounter{}
	// 缓存未命中计数器
	missCounter metricCounter = noopCounter{}
	// 总请求计数器
	totalCounter  metricCounter = noopCounter{}
	metricCacheMu sync.Mutex
	metricCache   = make(map[string]metricCounter)
)

// getOrDefineCounter will return an existing counter if cached, otherwise
// attempt to define it via proxywasm.DefineCounterMetric. If DefineCounterMetric
// panics or fails (e.g. non-wasm environment), a noopCounter will be cached and
// returned to avoid repeated attempts.
func getOrDefineCounter(name string) metricCounter {
	metricCacheMu.Lock()
	if c, ok := metricCache[name]; ok {
		metricCacheMu.Unlock()
		return c
	}
	// not cached yet
	metricCacheMu.Unlock()

	// attempt to define; protect from hostcall panics
	defer func() {
		if r := recover(); r != nil {
			proxywasm.LogWarnf("[token-statistics] DefineCounterMetric panic for %s: %v", name, r)
		}
	}()

	c := proxywasm.DefineCounterMetric(name)
	// cache the obtained counter
	metricCacheMu.Lock()
	metricCache[name] = c
	metricCacheMu.Unlock()
	return c
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

	// 定义（若尚未定义）并更新指标（懒初始化）
	inputCounter := getOrDefineCounter(inputMetricName)
	outputCounter := getOrDefineCounter(outputMetricName)
	totalCounter := getOrDefineCounter(totalMetricName)

	inputCounter.Increment(uint64(usage.InputTokens))
	outputCounter.Increment(uint64(usage.OutputTokens))
	totalCounter.Increment(uint64(usage.TotalTokens))
}

// 日志输出

func (l *LogExporter) Export(ctx wrapper.HttpContext, model string, usage *TokenUsage) {
	log.Infof("Token usage statistics: model=%s, input_tokens=%d, output_tokens=%d, total_tokens=%d",
		model, usage.InputTokens, usage.OutputTokens, usage.TotalTokens)
}
