package cluster_metrics

import (
	"math/rand"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-load-balancer/utils"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

var (
	globalIndex    int
	OnGoingSum     int
	GaugeMetrics   map[string]proxywasm.MetricGauge
	ServiceToken   map[string]int
	ServiceOngoing map[string]int
	ServiceAvgRT   map[string]float64
	ServiceCount   map[string]int
)

type ClusterEndpointLoadBalancer struct {
	Mode                      string
	ClusterHeader             string
	ServiceList               []string
	OnGoingLimit              float64
	FirstTokenLatencyRequests map[string]*utils.FixedQueue[float64]
	TotalLatencyRequests      map[string]*utils.FixedQueue[float64]
	UpperLimit                map[string]int
}

func NewClusterEndpointLoadBalancer(json gjson.Result) (ClusterEndpointLoadBalancer, error) {
	globalIndex = 0
	GaugeMetrics = make(map[string]proxywasm.MetricGauge)
	ServiceToken = make(map[string]int)
	ServiceOngoing = make(map[string]int)
	ServiceAvgRT = make(map[string]float64)
	ServiceCount = make(map[string]int)

	lb := ClusterEndpointLoadBalancer{}
	lb.Mode = json.Get("mode").String()
	lb.ClusterHeader = json.Get("clusterHeader").String()
	lb.OnGoingLimit = json.Get("ongoingLimit").Float()
	queueSize := int(json.Get("queueSize").Int())
	lb.FirstTokenLatencyRequests = make(map[string]*utils.FixedQueue[float64])
	lb.TotalLatencyRequests = make(map[string]*utils.FixedQueue[float64])

	for _, svc := range json.Get("serviceList").Array() {
		serviceName := svc.String()
		lb.ServiceList = append(lb.ServiceList, serviceName)
		ServiceToken[serviceName] = 0
		ServiceOngoing[serviceName] = 0
		ServiceAvgRT[serviceName] = 0
		ServiceCount[serviceName] = 0
		lb.FirstTokenLatencyRequests[serviceName] = utils.NewFixedQueue[float64](queueSize)
		lb.TotalLatencyRequests[serviceName] = utils.NewFixedQueue[float64](queueSize)
	}
	return lb, nil
}

func (lb ClusterEndpointLoadBalancer) getServiceTTFT(serviceName string) float64 {
	queue, ok := lb.FirstTokenLatencyRequests[serviceName]
	if !ok || queue.Size() == 0 {
		return 0
	}
	value := 0.0
	queue.ForEach(func(i int, item float64) {
		value += float64(item)
	})
	return value / float64(queue.Size())
}

func (lb ClusterEndpointLoadBalancer) getServiceTotalRT(serviceName string) float64 {
	queue, ok := lb.TotalLatencyRequests[serviceName]
	if !ok || queue.Size() == 0 {
		return 0
	}
	value := 0.0
	queue.ForEach(func(i int, item float64) {
		value += float64(item)
	})
	return value / float64(queue.Size())
}

// Callbacks which are called in request path
func (lb ClusterEndpointLoadBalancer) HandleHttpRequestHeaders(ctx wrapper.HttpContext) types.Action {
	ctx.SetContext("request_start", time.Now().UnixMilli())
	candidate := lb.ServiceList[rand.Int()%len(lb.ServiceList)]
	switch lb.Mode {
	case "RoundRobin":
		candidate = lb.ServiceList[globalIndex]
		globalIndex = (globalIndex + 1) % len(lb.ServiceList)
	case "LeastBusy":
		for svc, ongoingNum := range ServiceOngoing {
			if ongoingNum < ServiceOngoing[candidate] {
				candidate = svc
			}
		}
	case "LeastFirstTokenLatency":
		for svc := range ServiceOngoing {
			if ServiceAvgRT[svc] < ServiceAvgRT[candidate] {
				candidate = svc
			}
		}
	case "LeastTotalLatency":
		for svc, tokenUsage := range ServiceToken {
			if tokenUsage < ServiceToken[candidate] {
				candidate = svc
			}
		}
	}
	log.Infof("candidate: %s, candidate ongoing: %d, candidate rt(avg): %.2f", candidate, ServiceOngoing[candidate], ServiceAvgRT[candidate])
	proxywasm.ReplaceHttpRequestHeader(lb.ClusterHeader, candidate)
	ctx.SetContext(lb.ClusterHeader, candidate)
	ServiceOngoing[candidate] += 1
	OnGoingSum += 1
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpRequestBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpResponseHeaders(ctx wrapper.HttpContext) types.Action {
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpStreamingResponseBody(ctx wrapper.HttpContext, data []byte, endOfStream bool) []byte {
	candidate := ctx.GetContext(lb.ClusterHeader).(string)
	if endOfStream {
		duration := time.Now().UnixMilli() - ctx.GetContext("request_start").(int64)
		oldDuration := ServiceAvgRT[candidate]
		count := ServiceCount[candidate]
		ServiceAvgRT[candidate] = float64(oldDuration*float64(count)+float64(duration)) / float64(count+1)
		ServiceCount[candidate] += 1
		ServiceOngoing[candidate] -= 1
	}
	return data
}

func (lb ClusterEndpointLoadBalancer) HandleHttpResponseBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpStreamDone(ctx wrapper.HttpContext) {

}
