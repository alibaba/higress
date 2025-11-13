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

type ClusterEndpointLoadBalancer struct {
	// Configurations
	Mode          string
	ClusterHeader string
	ServiceList   []string
	RateLimit     float64
	// Statistic
	ServiceRequestOngoing     map[string]int
	ServiceRequestCount       map[string]int
	FirstTokenLatencyRequests map[string]*utils.FixedQueue[float64]
	TotalLatencyRequests      map[string]*utils.FixedQueue[float64]
}

func NewClusterEndpointLoadBalancer(json gjson.Result) (ClusterEndpointLoadBalancer, error) {
	lb := ClusterEndpointLoadBalancer{}
	lb.ServiceRequestOngoing = make(map[string]int)
	lb.ServiceRequestCount = make(map[string]int)
	lb.FirstTokenLatencyRequests = make(map[string]*utils.FixedQueue[float64])
	lb.TotalLatencyRequests = make(map[string]*utils.FixedQueue[float64])

	lb.Mode = json.Get("mode").String()
	lb.ClusterHeader = json.Get("cluster_header").String()
	lb.RateLimit = json.Get("rate_limit").Float()
	queueSize := int(json.Get("queue_size").Int())

	for _, svc := range json.Get("service_list").Array() {
		serviceName := svc.String()
		lb.ServiceList = append(lb.ServiceList, serviceName)
		lb.ServiceRequestOngoing[serviceName] = 0
		lb.ServiceRequestCount[serviceName] = 0
		lb.FirstTokenLatencyRequests[serviceName] = utils.NewFixedQueue[float64](queueSize)
		lb.TotalLatencyRequests[serviceName] = utils.NewFixedQueue[float64](queueSize)
	}
	return lb, nil
}

func (lb ClusterEndpointLoadBalancer) getRequestRate(serviceName string) float64 {
	totalRequestCount := 0
	for _, v := range lb.ServiceRequestCount {
		totalRequestCount += v
	}
	if totalRequestCount != 0 {
		return float64(lb.ServiceRequestCount[serviceName]) / float64(totalRequestCount)
	}
	return 0
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
	case "LeastBusy":
		for svc, ongoingNum := range lb.ServiceRequestOngoing {
			if candidate == svc {
				continue
			}
			log.Debugf("[candidate: %s] {ongoing request: %d, total request: %d, request rate: %.2f}, [new candidate: %s] {ongoing request: %d, total request: %d, request rate: %.2f}",
				candidate, lb.ServiceRequestOngoing[candidate], lb.ServiceRequestCount[candidate], lb.getRequestRate(candidate),
				svc, lb.ServiceRequestOngoing[svc], lb.ServiceRequestCount[svc], lb.getRequestRate(svc))
			if lb.getRequestRate(candidate) >= lb.RateLimit {
				candidate = svc
			} else if ongoingNum < lb.ServiceRequestOngoing[candidate] && lb.getRequestRate(svc) < lb.RateLimit {
				candidate = svc
			}
		}
	case "LeastFirstTokenLatency":
		candidateTTFT := lb.getServiceTTFT(candidate)
		for _, svc := range lb.ServiceList {
			if candidate == svc {
				continue
			}
			log.Debugf("[candidate: %s] {average ttft: %.2f, total request: %d, request rate: %.2f}, [new candidate: %s] {average ttft: %.2f, total request: %d, request rate: %.2f}",
				candidate, lb.getServiceTTFT(candidate), lb.ServiceRequestCount[candidate], lb.getRequestRate(candidate),
				svc, lb.getServiceTTFT(svc), lb.ServiceRequestCount[svc], lb.getRequestRate(svc))
			if lb.getRequestRate(candidate) >= lb.RateLimit {
				candidate = svc
			} else if lb.getServiceTTFT(svc) < candidateTTFT && lb.getRequestRate(svc) < lb.RateLimit {
				candidate = svc
			}
		}
	case "LeastTotalLatency":
		candidateTotalRT := lb.getServiceTotalRT(candidate)
		for _, svc := range lb.ServiceList {
			if candidate == svc {
				continue
			}
			log.Debugf("[candidate: %s] {average latency: %.2f, total request: %d, request rate: %.2f}, [new candidate: %s] {average latency: %.2f, total request: %d, request rate: %.2f}",
				candidate, lb.getServiceTotalRT(candidate), lb.ServiceRequestCount[candidate], lb.getRequestRate(candidate),
				svc, lb.getServiceTotalRT(svc), lb.ServiceRequestCount[svc], lb.getRequestRate(svc))
			if lb.getRequestRate(candidate) >= lb.RateLimit {
				candidate = svc
			} else if lb.getServiceTotalRT(svc) < candidateTotalRT && lb.getRequestRate(svc) < lb.RateLimit {
				candidate = svc
			}
		}
	}
	proxywasm.ReplaceHttpRequestHeader(lb.ClusterHeader, candidate)
	ctx.SetContext(lb.ClusterHeader, candidate)
	lb.ServiceRequestOngoing[candidate] += 1
	lb.ServiceRequestCount[candidate] += 1
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpRequestBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpResponseHeaders(ctx wrapper.HttpContext) types.Action {
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpStreamingResponseBody(ctx wrapper.HttpContext, data []byte, endOfStream bool) []byte {
	if ctx.GetContext("ttft_recorded") == nil {
		candidate := ctx.GetContext(lb.ClusterHeader).(string)
		duration := time.Now().UnixMilli() - ctx.GetContext("request_start").(int64)
		lb.FirstTokenLatencyRequests[candidate].Enqueue(float64(duration))
		ctx.SetContext("ttft_recorded", struct{}{})
	}
	return data
}

func (lb ClusterEndpointLoadBalancer) HandleHttpResponseBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpStreamDone(ctx wrapper.HttpContext) {
	candidate := ctx.GetContext(lb.ClusterHeader).(string)
	duration := time.Now().UnixMilli() - ctx.GetContext("request_start").(int64)
	lb.TotalLatencyRequests[candidate].Enqueue(float64(duration))
	lb.ServiceRequestOngoing[candidate] -= 1
}
