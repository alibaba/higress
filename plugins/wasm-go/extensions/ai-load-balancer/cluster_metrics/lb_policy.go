package cluster_metrics

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type ClusterEndpointLoadBalancer struct {
	Mode          string
	ClusterHeader string
	ServiceList   []string
}

const ()

var (
	globalIndex    int
	GaugeMetrics   map[string]proxywasm.MetricGauge
	ServiceToken   map[string]int
	ServiceOngoing map[string]int
	ServiceAvgRT   map[string]float64
	ServiceCount   map[string]int
	Service        map[string]int
)

func generateMetricName(route, cluster, model, consumer, metricName string) string {
	return fmt.Sprintf("route.%s.upstream.%s.model.%s.consumer.%s.metric.%s", route, cluster, model, consumer, metricName)
}

func GaudeAdd(GaugeMetrics map[string]proxywasm.MetricGauge, metricName string, delta int64) {
	if _, exists := GaugeMetrics[metricName]; !exists {
		GaugeMetrics[metricName] = proxywasm.DefineGaugeMetric(metricName)
	}
	GaugeMetrics[metricName].Add(delta)
}

func getRouteName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		return "-", err
	} else {
		return string(raw), nil
	}
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

	for _, svc := range json.Get("serviceList").Array() {
		serviceName := svc.String()
		lb.ServiceList = append(lb.ServiceList, serviceName)
		ServiceToken[serviceName] = 0
		ServiceOngoing[serviceName] = 0
		ServiceAvgRT[serviceName] = 0
		ServiceCount[serviceName] = 0
	}
	return lb, nil
}

// Callbacks which are called in request path
func (lb ClusterEndpointLoadBalancer) HandleHttpRequestHeaders(ctx wrapper.HttpContext) types.Action {
	route, _ := getRouteName()
	ctx.SetContext("route_name", route)
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
	case "RTAndOngoing":
		for svc := range ServiceOngoing {
			// log.Infof("candidate ongoing: %d, candidate rt(avg): %.2f, svc ongoing: %d, svc rt(avg): %.2f", ServiceOngoing[candidate], ServiceAvgRT[candidate], ServiceOngoing[svc], ServiceAvgRT[svc])
			if float64(ServiceOngoing[svc])*ServiceAvgRT[svc] < float64(ServiceOngoing[candidate])*ServiceAvgRT[candidate] {
				candidate = svc
			}
		}
	case "LeastToken":
		for svc, tokenUsage := range ServiceToken {
			if tokenUsage < ServiceToken[candidate] {
				candidate = svc
			}
		}
	}
	log.Infof("candidate: %s, candidate ongoing: %d, candidate rt(avg): %.2f", candidate, ServiceOngoing[candidate], ServiceAvgRT[candidate])
	proxywasm.ReplaceHttpRequestHeader(lb.ClusterHeader, candidate)
	ctx.SetContext(lb.ClusterHeader, candidate)
	metricName := generateMetricName(route, candidate, "none", "none", "ongoing")
	GaudeAdd(GaugeMetrics, metricName, 1)
	ServiceOngoing[candidate] += 1
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpRequestBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpResponseHeaders(ctx wrapper.HttpContext) types.Action {
	ctx.DontReadResponseBody()
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpStreamingResponseBody(ctx wrapper.HttpContext, data []byte, endOfStream bool) []byte {
	candidate := ctx.GetContext(lb.ClusterHeader).(string)
	if _, inputToken, outputToken, ok := getUsage(data); ok {
		ServiceToken[candidate] += int(inputToken + outputToken)
	}
	if endOfStream {
		metricName := generateMetricName(ctx.GetContext("route_name").(string), candidate, "none", "none", "ongoing")
		GaudeAdd(GaugeMetrics, metricName, -1)
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
	candidate := ctx.GetContext(lb.ClusterHeader).(string)
	if _, inputToken, outputToken, ok := getUsage(body); ok {
		ServiceToken[candidate] += int(inputToken + outputToken)
	}

	metricName := generateMetricName(ctx.GetContext("route_name").(string), candidate, "none", "none", "ongoing")
	GaudeAdd(GaugeMetrics, metricName, -1)

	duration := time.Now().UnixMilli() - ctx.GetContext("request_start").(int64)
	oldDuration := ServiceAvgRT[candidate]
	count := ServiceCount[candidate]
	ServiceAvgRT[candidate] = float64(oldDuration*float64(count)+float64(duration)) / float64(count+1)
	ServiceCount[candidate] += 1
	ServiceOngoing[candidate] -= 1
	return types.ActionContinue
}

func (lb ClusterEndpointLoadBalancer) HandleHttpStreamDone(ctx wrapper.HttpContext) {}

func getUsage(data []byte) (model string, inputTokenUsage int64, outputTokenUsage int64, ok bool) {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	for _, chunk := range chunks {
		// the feature strings are used to identify the usage data, like:
		// {"model":"gpt2","usage":{"prompt_tokens":1,"completion_tokens":1}}
		if !bytes.Contains(chunk, []byte("prompt_tokens")) {
			continue
		}
		if !bytes.Contains(chunk, []byte("completion_tokens")) {
			continue
		}
		modelObj := gjson.GetBytes(chunk, "model")
		inputTokenObj := gjson.GetBytes(chunk, "usage.prompt_tokens")
		outputTokenObj := gjson.GetBytes(chunk, "usage.completion_tokens")
		if modelObj.Exists() && inputTokenObj.Exists() && outputTokenObj.Exists() {
			model = modelObj.String()
			inputTokenUsage = inputTokenObj.Int()
			outputTokenUsage = outputTokenObj.Int()
			ok = true
			return
		}
	}
	return
}
