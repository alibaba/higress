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

package backend

import "fmt"

type PodSet map[Pod]bool

type Pod struct {
	Name    string
	Address string
}

func (p Pod) String() string {
	return p.Name + ":" + p.Address
}

type Metrics struct {
	// ActiveModels is a set of models(including LoRA adapters) that are currently cached to GPU.
	ActiveModels map[string]int
	// MaxActiveModels is the maximum number of models that can be loaded to GPU.
	MaxActiveModels         int
	RunningQueueSize        int
	WaitingQueueSize        int
	KVCacheUsagePercent     float64
	KvCacheMaxTokenCapacity int
}

type UserSelectedMetric struct {
	MetricName  string
	MetricValue float64
}

type PodMetrics struct {
	Pod
	Metrics
	UserSelectedMetric
}

func (pm *PodMetrics) String() string {
	return fmt.Sprintf("Pod: %+v; Metrics: %+v, UserSelectedMetric: %+v", pm.Pod, pm.Metrics, pm.UserSelectedMetric)
}

func (pm *PodMetrics) Clone() *PodMetrics {
	cm := make(map[string]int, len(pm.ActiveModels))
	for k, v := range pm.ActiveModels {
		cm[k] = v
	}
	clone := &PodMetrics{
		Pod: pm.Pod,
		Metrics: Metrics{
			ActiveModels:            cm,
			RunningQueueSize:        pm.RunningQueueSize,
			WaitingQueueSize:        pm.WaitingQueueSize,
			KVCacheUsagePercent:     pm.KVCacheUsagePercent,
			KvCacheMaxTokenCapacity: pm.KvCacheMaxTokenCapacity,
		},
		UserSelectedMetric: UserSelectedMetric{
			MetricName:  pm.MetricName,
			MetricValue: pm.MetricValue,
		},
	}
	return clone
}
