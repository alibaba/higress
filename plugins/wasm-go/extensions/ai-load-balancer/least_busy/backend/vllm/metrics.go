/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package vllm provides vllm specific pod metrics implementation.
package vllm

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-load-balancer/least_busy/backend"

	dto "github.com/prometheus/client_model/go"
	"go.uber.org/multierr"
)

const (
	LoraRequestInfoMetricName                = "vllm:lora_requests_info"
	LoraRequestInfoRunningAdaptersMetricName = "running_lora_adapters"
	LoraRequestInfoMaxAdaptersMetricName     = "max_lora"
	// TODO: Replace these with the num_tokens_running/waiting below once we add those to the fork.
	RunningQueueSizeMetricName = "vllm:num_requests_running"
	WaitingQueueSizeMetricName = "vllm:num_requests_waiting"
	/* TODO: Uncomment this once the following are added to the fork.
	RunningQueueSizeMetricName        = "vllm:num_tokens_running"
	WaitingQueueSizeMetricName        = "vllm:num_tokens_waiting"
	*/
	KVCacheUsagePercentMetricName     = "vllm:gpu_cache_usage_perc"
	KvCacheMaxTokenCapacityMetricName = "vllm:gpu_cache_max_token_capacity"
)

// promToPodMetrics updates internal pod metrics with scraped prometheus metrics.
// A combined error is returned if errors occur in one or more metric processing.
// it returns a new PodMetrics pointer which can be used to atomically update the pod metrics map.
func PromToPodMetrics(
	metricFamilies map[string]*dto.MetricFamily,
	existing *backend.PodMetrics,
) (*backend.PodMetrics, error) {
	var errs error
	updated := existing.Clone()
	runningQueueSize, err := getLatestMetric(metricFamilies, RunningQueueSizeMetricName)
	errs = multierr.Append(errs, err)
	if err == nil {
		updated.RunningQueueSize = int(runningQueueSize.GetGauge().GetValue())
	}
	waitingQueueSize, err := getLatestMetric(metricFamilies, WaitingQueueSizeMetricName)
	errs = multierr.Append(errs, err)
	if err == nil {
		updated.WaitingQueueSize = int(waitingQueueSize.GetGauge().GetValue())
	}
	cachePercent, err := getLatestMetric(metricFamilies, KVCacheUsagePercentMetricName)
	errs = multierr.Append(errs, err)
	if err == nil {
		updated.KVCacheUsagePercent = cachePercent.GetGauge().GetValue()
	}

	loraMetrics, _, err := getLatestLoraMetric(metricFamilies)
	errs = multierr.Append(errs, err)
	/* TODO: uncomment once this is available in vllm.
	kvCap, _, err := getGaugeLatestValue(metricFamilies, KvCacheMaxTokenCapacityMetricName)
	errs = multierr.Append(errs, err)
	if err != nil {
		updated.KvCacheMaxTokenCapacity = int(kvCap)
	}
	*/

	if loraMetrics != nil {
		updated.ActiveModels = make(map[string]int)
		for _, label := range loraMetrics.GetLabel() {
			if label.GetName() == LoraRequestInfoRunningAdaptersMetricName {
				if label.GetValue() != "" {
					adapterList := strings.Split(label.GetValue(), ",")
					for _, adapter := range adapterList {
						updated.ActiveModels[adapter] = 0
					}
				}
			}
			if label.GetName() == LoraRequestInfoMaxAdaptersMetricName {
				if label.GetValue() != "" {
					updated.MaxActiveModels, err = strconv.Atoi(label.GetValue())
					if err != nil {
						errs = multierr.Append(errs, err)
					}
				}
			}
		}

	}

	return updated, errs
}

// getLatestLoraMetric gets latest lora metric series in gauge metric family `vllm:lora_requests_info`
// reason its specially fetched is because each label key value pair permutation generates new series
// and only most recent is useful. The value of each series is the creation timestamp so we can
// retrieve the latest by sorting the value.
func getLatestLoraMetric(metricFamilies map[string]*dto.MetricFamily) (*dto.Metric, time.Time, error) {
	loraRequests, ok := metricFamilies[LoraRequestInfoMetricName]
	if !ok {
		// klog.Warningf("metric family %q not found", LoraRequestInfoMetricName)
		return nil, time.Time{}, fmt.Errorf("metric family %q not found", LoraRequestInfoMetricName)
	}
	var latestTs float64
	var latest *dto.Metric
	for _, m := range loraRequests.GetMetric() {
		if m.GetGauge().GetValue() > latestTs {
			latestTs = m.GetGauge().GetValue()
			latest = m
		}
	}
	return latest, time.Unix(0, int64(latestTs*1000)), nil
}

// getLatestMetric gets the latest metric of a family. This should be used to get the latest Gauge metric.
// Since vllm doesn't set the timestamp in metric, this metric essentially gets the first metric.
func getLatestMetric(metricFamilies map[string]*dto.MetricFamily, metricName string) (*dto.Metric, error) {
	mf, ok := metricFamilies[metricName]
	if !ok {
		// klog.Warningf("metric family %q not found", metricName)
		return nil, fmt.Errorf("metric family %q not found", metricName)
	}
	if len(mf.GetMetric()) == 0 {
		return nil, fmt.Errorf("no metrics available for %q", metricName)
	}
	var latestTs int64
	var latest *dto.Metric
	for _, m := range mf.GetMetric() {
		if m.GetTimestampMs() >= latestTs {
			latestTs = m.GetTimestampMs()
			latest = m
		}
	}
	// klog.V(logutil.TRACE).Infof("Got metric value %+v for metric %v", latest, metricName)
	return latest, nil
}
