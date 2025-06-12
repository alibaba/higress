package main

import (
	"errors"
	"net"
	"strings"

	"ai-load-balancer/backend"
	"ai-load-balancer/backend/vllm"
	"ai-load-balancer/scheduling"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/prometheus/common/expfmt"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"ai-lbpolicy-leastbusy",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
	)
}

type LBConfig struct {
	criticalModels map[string]struct{}
}

func parseConfig(json gjson.Result, config *LBConfig) error {
	config.criticalModels = make(map[string]struct{})
	for _, model := range json.Get("criticalModels").Array() {
		config.criticalModels[model.String()] = struct{}{}
	}
	return nil
}

// Callbacks which are called in request path
func onHttpRequestHeaders(ctx wrapper.HttpContext, config LBConfig) types.Action {
	// If return types.ActionContinue, SetUpstreamOverrideHost will failed
	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, config LBConfig, body []byte) types.Action {
	requestModel := gjson.GetBytes(body, "model")
	if !requestModel.Exists() {
		return types.ActionContinue
	}
	_, isCritical := config.criticalModels[requestModel.String()]
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
	scheduler, err := GetScheduler(hostMetrics)
	if err != nil {
		log.Debugf("initial scheduler failed: %v", err)
		return types.ActionContinue
	}
	targetPod, err := scheduler.Schedule(llmReq)
	log.Debugf("targetPod: %+v", targetPod.Address)
	if err != nil {
		log.Debugf("pod select failed: %v", err)
		proxywasm.SendHttpResponseWithDetail(429, "from llm-load-balancer", nil, []byte("limited resources"), 0)
	}
	if isValidAddress(targetPod.Address) {
		log.Debugf("override upstream host: %s", targetPod.Address)
		proxywasm.SetUpstreamOverrideHost([]byte(targetPod.Address))
	} else {
		log.Debugf("invalid address: %s", targetPod.Address)
	}
	return types.ActionContinue
}

func GetScheduler(hostMetrics map[string]string) (*scheduling.Scheduler, error) {
	if len(hostMetrics) == 0 {
		return nil, errors.New("backend is not support llm scheduling")
	}
	var pms []*backend.PodMetrics
	for addr, metric := range hostMetrics {
		parser := expfmt.TextParser{}
		metricFamilies, err := parser.TextToMetricFamilies(strings.NewReader(metric))
		if err != nil {
			return nil, err
		}
		pm := &backend.PodMetrics{
			Pod: backend.Pod{
				Name:    addr,
				Address: addr,
			},
			Metrics: backend.Metrics{},
		}
		pm, err = vllm.PromToPodMetrics(metricFamilies, pm)
		if err != nil {
			return nil, err
		}
		pms = append(pms, pm)
	}
	return scheduling.NewScheduler(pms), nil
}

func isValidAddress(s string) bool {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return false
	}

	_, err = net.LookupPort("tcp", port)
	if err != nil {
		return false
	}

	ip := net.ParseIP(host)
	return ip != nil
}
