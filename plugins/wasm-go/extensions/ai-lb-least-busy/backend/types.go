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

type PodMetrics struct {
	Pod
	Metrics
}

func (pm *PodMetrics) String() string {
	return fmt.Sprintf("Pod: %+v; Metrics: %+v", pm.Pod, pm.Metrics)
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
	}
	return clone
}
