// Package scheduling implements request scheduling algorithms.
package scheduling

import (
	"errors"
	"fmt"
	"math/rand"

	"ai-load-balancer/backend"
)

const (
	// TODO(https://github.com/kubernetes-sigs/gateway-api-inference-extension/issues/16) Make this configurable.
	kvCacheThreshold = 0.8
	// TODO(https://github.com/kubernetes-sigs/gateway-api-inference-extension/issues/16) Make this configurable.
	queueThresholdCritical = 5
	// TODO(https://github.com/kubernetes-sigs/gateway-api-inference-extension/issues/16) Make this configurable.
	// the threshold for queued requests to be considered low below which we can prioritize LoRA affinity.
	// The value of 50 is arrived heuristicically based on experiments.
	queueingThresholdLoRA = 50
)

var (
	defaultFilter = &filter{
		name:          "critical request",
		filter:        toFilterFunc(criticalRequestPredicate),
		nextOnSuccess: lowLatencyFilter,
		nextOnFailure: sheddableRequestFilter,
	}

	// queueLoRAAndKVCacheFilter applied least queue -> low cost lora ->  least KV Cache filter
	queueLoRAAndKVCacheFilter = &filter{
		name:   "least queuing",
		filter: leastQueuingFilterFunc,
		nextOnSuccessOrFailure: &filter{
			name:   "low cost LoRA",
			filter: toFilterFunc(lowLoRACostPredicate),
			nextOnSuccessOrFailure: &filter{
				name:   "least KV cache percent",
				filter: leastKVCacheFilterFunc,
			},
		},
	}

	// queueAndKVCacheFilter applies least queue followed by least KV Cache filter
	queueAndKVCacheFilter = &filter{
		name:   "least queuing",
		filter: leastQueuingFilterFunc,
		nextOnSuccessOrFailure: &filter{
			name:   "least KV cache percent",
			filter: leastKVCacheFilterFunc,
		},
	}

	lowLatencyFilter = &filter{
		name:   "low queueing filter",
		filter: toFilterFunc((lowQueueingPodPredicate)),
		nextOnSuccess: &filter{
			name:          "affinity LoRA",
			filter:        toFilterFunc(loRAAffinityPredicate),
			nextOnSuccess: queueAndKVCacheFilter,
			nextOnFailure: &filter{
				name:                   "can accept LoRA Adapter",
				filter:                 toFilterFunc(canAcceptNewLoraPredicate),
				nextOnSuccessOrFailure: queueAndKVCacheFilter,
			},
		},
		nextOnFailure: queueLoRAAndKVCacheFilter,
	}

	sheddableRequestFilter = &filter{
		// When there is at least one model server that's not queuing requests, and still has KV
		// cache below a certain threshold, we consider this model server has capacity to handle
		// a sheddable request without impacting critical requests.
		name:          "has capacity for sheddable requests",
		filter:        toFilterFunc(noQueueAndLessThanKVCacheThresholdPredicate(queueThresholdCritical, kvCacheThreshold)),
		nextOnSuccess: queueLoRAAndKVCacheFilter,
		// If all pods are queuing or running above the KVCache threshold, we drop the sheddable
		// request to make room for critical requests.
		nextOnFailure: &filter{
			name: "drop request",
			filter: func(req *LLMRequest, pods []*backend.PodMetrics) ([]*backend.PodMetrics, error) {
				// api.LogDebugf("Dropping request %v", req)
				return []*backend.PodMetrics{}, errors.New("dropping request due to limited backend resources")
			},
		},
	}
)

func NewScheduler(pm []*backend.PodMetrics) *Scheduler {

	return &Scheduler{
		podMetrics: pm,
		filter:     defaultFilter,
	}
}

type Scheduler struct {
	podMetrics []*backend.PodMetrics
	filter     Filter
}

// Schedule finds the target pod based on metrics and the requested lora adapter.
func (s *Scheduler) Schedule(req *LLMRequest) (targetPod backend.Pod, err error) {
	pods, err := s.filter.Filter(req, s.podMetrics)
	if err != nil || len(pods) == 0 {
		return backend.Pod{}, fmt.Errorf("failed to apply filter, resulted %v pods: %w", len(pods), err)
	}
	i := rand.Intn(len(pods))
	return pods[i].Pod, nil
}
