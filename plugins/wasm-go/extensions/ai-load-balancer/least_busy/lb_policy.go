package least_busy

import (
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-load-balancer/least_busy/scheduling"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type LeastBusyLoadBalancer struct {
	criticalModels map[string]struct{}
}

func NewLeastBusyLoadBalancer(json gjson.Result) (LeastBusyLoadBalancer, error) {
	lb := LeastBusyLoadBalancer{}
	lb.criticalModels = make(map[string]struct{})
	for _, model := range json.Get("criticalModels").Array() {
		lb.criticalModels[model.String()] = struct{}{}
	}
	return lb, nil
}

// Callbacks which are called in request path
func (lb LeastBusyLoadBalancer) HandleHttpRequestHeaders(ctx wrapper.HttpContext) types.Action {
	// If return types.ActionContinue, SetUpstreamOverrideHost will not take effect
	return types.HeaderStopIteration
}

func (lb LeastBusyLoadBalancer) HandleHttpRequestBody(ctx wrapper.HttpContext, body []byte) types.Action {
	requestModel := gjson.GetBytes(body, "model")
	if !requestModel.Exists() {
		return types.ActionContinue
	}
	_, isCritical := lb.criticalModels[requestModel.String()]
	llmReq := &scheduling.LLMRequest{
		Model:    requestModel.String(),
		Critical: isCritical,
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
	scheduler, err := scheduling.GetScheduler(hostMetrics)
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

func (lb LeastBusyLoadBalancer) HandleHttpResponseHeaders(ctx wrapper.HttpContext) types.Action {
	ctx.DontReadResponseBody()
	return types.ActionContinue
}

func (lb LeastBusyLoadBalancer) HandleHttpStreamingResponseBody(ctx wrapper.HttpContext, data []byte, endOfStream bool) []byte {
	return data
}

func (lb LeastBusyLoadBalancer) HandleHttpResponseBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}

func (lb LeastBusyLoadBalancer) HandleHttpStreamDone(ctx wrapper.HttpContext) {}
