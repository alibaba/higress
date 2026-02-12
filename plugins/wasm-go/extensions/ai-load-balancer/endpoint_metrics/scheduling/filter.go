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

package scheduling

import (
	"errors"
	"math"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-load-balancer/endpoint_metrics/backend"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

type Filter interface {
	Name() string
	Filter(req *LLMRequest, pods []*backend.PodMetrics) ([]*backend.PodMetrics, error)
}

// filter applies current filterFunc, and then recursively applies next filters depending success or
// failure of the current filterFunc.
// It can be used to construct a flow chart algorithm.
type filter struct {
	name   string
	filter filterFunc
	// nextOnSuccess filter will be applied after successfully applying the current filter.
	// The filtered results will be passed to the next filter.
	nextOnSuccess *filter
	// nextOnFailure filter will be applied if current filter fails.
	// The original input will be passed to the next filter.
	nextOnFailure *filter
	// nextOnSuccessOrFailure is a convenience field to configure the next filter regardless of the
	// success or failure of the current filter.
	// NOTE: When using nextOnSuccessOrFailure, both nextOnSuccess and nextOnFailure SHOULD be nil.
	// However if that's not the case, nextOnSuccess and nextOnFailure will be used, instead of
	// nextOnSuccessOrFailure,  in the success and failure scenarios, respectively.
	nextOnSuccessOrFailure *filter

	// callbacks api.FilterCallbackHandler
}

func (f *filter) Name() string {
	if f == nil {
		return "nil"
	}
	return f.name
}

func (f *filter) Filter(req *LLMRequest, pods []*backend.PodMetrics) ([]*backend.PodMetrics, error) {
	proxywasm.LogDebugf("Running filter %q on request %v with %v pods", f.name, req, len(pods))
	filtered, err := f.filter(req, pods)

	next := f.nextOnSuccessOrFailure
	if err == nil && len(filtered) > 0 {
		if f.nextOnSuccess == nil && f.nextOnSuccessOrFailure == nil {
			// No succeeding filters to run, return.
			return filtered, err
		}
		if f.nextOnSuccess != nil {
			next = f.nextOnSuccess
		}
		// On success, pass the filtered result to the next filter.
		return next.Filter(req, filtered)
	} else {
		if f.nextOnFailure == nil && f.nextOnSuccessOrFailure == nil {
			// No succeeding filters to run, return.
			return filtered, err
		}
		if f.nextOnFailure != nil {
			next = f.nextOnFailure
		}
		// On failure, pass the initial set of pods to the next filter.
		return next.Filter(req, pods)
	}
}

// filterFunc filters a set of input pods to a subset.
type filterFunc func(req *LLMRequest, pods []*backend.PodMetrics) ([]*backend.PodMetrics, error)

// toFilterFunc is a helper function to convert a per pod filter func to the FilterFunc.
func toFilterFunc(pp podPredicate) filterFunc {
	return func(req *LLMRequest, pods []*backend.PodMetrics) ([]*backend.PodMetrics, error) {
		filtered := []*backend.PodMetrics{}
		for _, pod := range pods {
			pass := pp(req, pod)
			if pass {
				filtered = append(filtered, pod)
			}
		}
		if len(filtered) == 0 {
			return nil, errors.New("no pods left")
		}
		return filtered, nil
	}
}

// leastQueuingFilterFunc finds the max and min queue size of all pods, divides the whole range
// (max-min) by the number of pods, and finds the pods that fall into the first range.
// The intuition is that if there are multiple pods that share similar queue size in the low range,
// we should consider them all instead of the absolute minimum one. This worked better than picking
// the least one as it gives more choices for the next filter, which on aggregate gave better
// results.
// TODO: Compare this strategy with other strategies such as top K.
func leastQueuingFilterFunc(req *LLMRequest, pods []*backend.PodMetrics) ([]*backend.PodMetrics, error) {
	min := math.MaxInt
	max := 0
	filtered := []*backend.PodMetrics{}

	for _, pod := range pods {
		if pod.WaitingQueueSize <= min {
			min = pod.WaitingQueueSize
		}
		if pod.WaitingQueueSize >= max {
			max = pod.WaitingQueueSize
		}
	}

	for _, pod := range pods {
		if pod.WaitingQueueSize >= min && pod.WaitingQueueSize <= min+(max-min)/len(pods) {
			filtered = append(filtered, pod)
		}
	}
	return filtered, nil
}

func lowQueueingPodPredicate(_ *LLMRequest, pod *backend.PodMetrics) bool {
	return pod.WaitingQueueSize < queueingThresholdLoRA
}

// leastKVCacheFilterFunc finds the max and min KV cache of all pods, divides the whole range
// (max-min) by the number of pods, and finds the pods that fall into the first range.
// The intuition is that if there are multiple pods that share similar KV cache in the low range, we
// should consider them all instead of the absolute minimum one. This worked better than picking the
// least one as it gives more choices for the next filter, which on aggregate gave better results.
// TODO: Compare this strategy with other strategies such as top K.
func leastKVCacheFilterFunc(req *LLMRequest, pods []*backend.PodMetrics) ([]*backend.PodMetrics, error) {
	min := math.MaxFloat64
	var max float64 = 0
	filtered := []*backend.PodMetrics{}

	for _, pod := range pods {
		if pod.KVCacheUsagePercent <= min {
			min = pod.KVCacheUsagePercent
		}
		if pod.KVCacheUsagePercent >= max {
			max = pod.KVCacheUsagePercent
		}
	}

	for _, pod := range pods {
		if pod.KVCacheUsagePercent >= min && pod.KVCacheUsagePercent <= min+(max-min)/float64(len(pods)) {
			filtered = append(filtered, pod)
		}
	}
	return filtered, nil
}

// podPredicate is a filter function to check whether a pod is desired.
type podPredicate func(req *LLMRequest, pod *backend.PodMetrics) bool

// We consider serving an adapter low cost it the adapter is active in the model server, or the
// model server has room to load the adapter. The lowLoRACostPredicate ensures weak affinity by
// spreading the load of a LoRA adapter across multiple pods, avoiding "pinning" all requests to
// a single pod. This gave good performance in our initial benchmarking results in the scenario
// where # of lora slots > # of lora adapters.
func lowLoRACostPredicate(req *LLMRequest, pod *backend.PodMetrics) bool {
	_, ok := pod.ActiveModels[req.Model]
	return ok || len(pod.ActiveModels) < pod.MaxActiveModels
}

// loRAAffinityPredicate is a filter function to check whether a pod has affinity to the lora requested.
func loRAAffinityPredicate(req *LLMRequest, pod *backend.PodMetrics) bool {
	_, ok := pod.ActiveModels[req.Model]
	return ok
}

// canAcceptNewLoraPredicate is a filter function to check whether a pod has room to load the adapter.
func canAcceptNewLoraPredicate(req *LLMRequest, pod *backend.PodMetrics) bool {
	return len(pod.ActiveModels) < pod.MaxActiveModels
}

func criticalRequestPredicate(req *LLMRequest, pod *backend.PodMetrics) bool {
	return req.Critical
}

func noQueueAndLessThanKVCacheThresholdPredicate(queueThreshold int, kvCacheThreshold float64) podPredicate {
	return func(req *LLMRequest, pod *backend.PodMetrics) bool {
		return pod.WaitingQueueSize <= queueThreshold && pod.KVCacheUsagePercent <= kvCacheThreshold
	}
}
