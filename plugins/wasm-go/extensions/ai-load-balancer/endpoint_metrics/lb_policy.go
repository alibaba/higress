package endpoint_metrics

import (
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-load-balancer/endpoint_metrics/scheduling"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type MetricsEndpointLoadBalancer struct {
	metricPolicy string
	targetMetric string
}

func NewMetricsEndpointLoadBalancer(json gjson.Result) (MetricsEndpointLoadBalancer, error) {
	lb := MetricsEndpointLoadBalancer{}
	if json.Get("metricPolicy").Exists() {
		lb.metricPolicy = json.Get("metricPolicy").String()
	} else {
		lb.metricPolicy = scheduling.MetricPolicyDefault
	}
	if json.Get("targetMetric").Exists() {
		lb.targetMetric = json.Get("targetMetric").String()
	}
	return lb, nil
}

// Callbacks which are called in request path
func (lb MetricsEndpointLoadBalancer) HandleHttpRequestHeaders(ctx wrapper.HttpContext) types.Action {
	// If return types.ActionContinue, SetUpstreamOverrideHost will not take effect
	return types.HeaderStopIteration
}

func (lb MetricsEndpointLoadBalancer) HandleHttpRequestBody(ctx wrapper.HttpContext, body []byte) types.Action {
	requestModel := gjson.GetBytes(body, "model")
	if !requestModel.Exists() {
		return types.ActionContinue
	}
	llmReq := &scheduling.LLMRequest{
		Model:    requestModel.String(),
		Critical: true,
	}
	hostInfos, err := proxywasm.GetUpstreamHosts()
	if err != nil {
		return types.ActionContinue
	}
	hostMetrics := make(map[string]string)
	for _, hostInfo := range hostInfos {
		if gjson.Get(hostInfo[1], "health_status").String() == "Healthy" {
			hostMetrics[hostInfo[0]] = gjson.Get(hostInfo[1], "metrics").String()
		}
	}
	scheduler, err := scheduling.GetScheduler(hostMetrics, lb.metricPolicy, lb.targetMetric)
	if err != nil {
		log.Debugf("initial scheduler failed: %v", err)
		return types.ActionContinue
	}
	targetPod, err := scheduler.Schedule(llmReq)
	log.Debugf("targetPod: %+v", targetPod.Address)
	if err != nil {
		log.Debugf("pod select failed: %v", err)
		proxywasm.SendHttpResponseWithDetail(429, "limited resources", nil, []byte("limited resources"), 0)
	} else {
		proxywasm.SetUpstreamOverrideHost([]byte(targetPod.Address))
	}
	return types.ActionContinue
}

func (lb MetricsEndpointLoadBalancer) HandleHttpResponseHeaders(ctx wrapper.HttpContext) types.Action {
	ctx.DontReadResponseBody()
	return types.ActionContinue
}

func (lb MetricsEndpointLoadBalancer) HandleHttpStreamingResponseBody(ctx wrapper.HttpContext, data []byte, endOfStream bool) []byte {
	return data
}

func (lb MetricsEndpointLoadBalancer) HandleHttpResponseBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}

func (lb MetricsEndpointLoadBalancer) HandleHttpStreamDone(ctx wrapper.HttpContext) {}
