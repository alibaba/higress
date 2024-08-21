package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-quota/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

const (
	pluginName = "ai-quota"
)

type ChatMode string

const (
	ChatModeCompletion ChatMode = "completion"
	ChatModeAdmin      ChatMode = "admin"
	ChatModeNone       ChatMode = "none"
)

type AdminMode string

const (
	AdminModeRefresh AdminMode = "refresh"
	AdminModeQuery   AdminMode = "query"
	AdminModeDelta   AdminMode = "delta"
	AdminModeNone    AdminMode = "none"
)

func main() {
	wrapper.SetCtx(
		pluginName,
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamingResponseBody),
	)
}

type QuotaConfig struct {
	redisInfo       RedisInfo         `yaml:"redis"`
	RedisKeyPrefix  string            `yaml:"redis_key_prefix"`
	AdminConsumer   string            `yaml:"admin_consumer"`
	AdminPath       string            `yaml:"admin_path"`
	credential2Name map[string]string `yaml:"-"`
	redisClient     wrapper.RedisClient
}

type Consumer struct {
	Name       string `yaml:"name"`
	Credential string `yaml:"credential"`
}

type RedisInfo struct {
	ServiceName string `required:"true" yaml:"service_name" json:"service_name"`
	ServicePort int    `required:"false" yaml:"service_port" json:"service_port"`
	Username    string `required:"false" yaml:"username" json:"username"`
	Password    string `required:"false" yaml:"password" json:"password"`
	Timeout     int    `required:"false" yaml:"timeout" json:"timeout"`
}

func parseConfig(json gjson.Result, config *QuotaConfig, log wrapper.Log) error {
	log.Debugf("parse config()")
	// admin
	config.AdminPath = json.Get("admin_path").String()
	config.AdminConsumer = json.Get("admin_consumer").String()
	if config.AdminPath == "" {
		config.AdminPath = "/quota"
	}
	if config.AdminConsumer == "" {
		return errors.New("missing admin_consumer in config")
	}
	// Redis
	config.RedisKeyPrefix = json.Get("redis_key_prefix").String()
	if config.RedisKeyPrefix == "" {
		config.RedisKeyPrefix = "chat_quota:"
	}
	redisConfig := json.Get("redis")
	if !redisConfig.Exists() {
		return errors.New("missing redis in config")
	}
	serviceName := redisConfig.Get("service_name").String()
	if serviceName == "" {
		return errors.New("redis service name must not be empty")
	}
	servicePort := int(redisConfig.Get("service_port").Int())
	if servicePort == 0 {
		if strings.HasSuffix(serviceName, ".static") {
			// use default logic port which is 80 for static service
			servicePort = 80
		} else {
			servicePort = 6379
		}
	}
	username := redisConfig.Get("username").String()
	password := redisConfig.Get("password").String()
	timeout := int(redisConfig.Get("timeout").Int())
	if timeout == 0 {
		timeout = 1000
	}
	config.redisInfo.ServiceName = serviceName
	config.redisInfo.ServicePort = servicePort
	config.redisInfo.Username = username
	config.redisInfo.Password = password
	config.redisInfo.Timeout = timeout
	config.redisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: int64(servicePort),
	})

	return config.redisClient.Init(username, password, int64(timeout))
}

func onHttpRequestHeaders(context wrapper.HttpContext, config QuotaConfig, log wrapper.Log) types.Action {
	log.Debugf("onHttpRequestHeaders()")
	// get tokens
	consumer, err := proxywasm.GetHttpRequestHeader("x-mse-consumer")
	if err != nil {
		return deniedNoKeyAuthData()
	}
	if consumer == "" {
		return deniedUnauthorizedConsumer()
	}

	rawPath := context.Path()
	path, _ := url.Parse(rawPath)
	chatMode, adminMode := getOperationMode(path.Path, config.AdminPath, log)
	context.SetContext("chatMode", chatMode)
	context.SetContext("adminMode", adminMode)
	context.SetContext("consumer", consumer)
	log.Debugf("chatMode:%s, adminMode:%s, consumer:%s", chatMode, adminMode, consumer)
	if chatMode == ChatModeNone {
		return types.ActionContinue
	}
	if chatMode == ChatModeAdmin {
		// query quota
		if adminMode == AdminModeQuery {
			return queryQuota(context, config, consumer, path, log)
		}
		if adminMode == AdminModeRefresh || adminMode == AdminModeDelta {
			context.BufferRequestBody()
			return types.HeaderStopIteration
		}
		return types.ActionContinue
	}

	// there is no need to read request body when it is on chat completion mode
	context.DontReadRequestBody()
	// check quota here
	config.redisClient.Get(config.RedisKeyPrefix+consumer, func(response resp.Value) {
		isDenied := false
		if err := response.Error(); err != nil {
			isDenied = true
		}
		if response.IsNull() {
			isDenied = true
		}
		if response.Integer() <= 0 {
			isDenied = true
		}
		log.Debugf("get consumer:%s quota:%d isDenied:%t", consumer, response.Integer(), isDenied)
		if isDenied {
			util.SendResponse(http.StatusForbidden, "ai-quota.noquota", "text/plain", "Request denied by ai quota check, No quota left")
			return
		}
		proxywasm.ResumeHttpRequest()
	})
	return types.HeaderStopAllIterationAndWatermark
}

func onHttpRequestBody(ctx wrapper.HttpContext, config QuotaConfig, body []byte, log wrapper.Log) types.Action {
	log.Debugf("onHttpRequestBody()")
	chatMode, ok := ctx.GetContext("chatMode").(ChatMode)
	if !ok {
		return types.ActionContinue
	}
	if chatMode == ChatModeNone || chatMode == ChatModeCompletion {
		return types.ActionContinue
	}
	adminMode, ok := ctx.GetContext("adminMode").(AdminMode)
	if !ok {
		return types.ActionContinue
	}
	adminConsumer, ok := ctx.GetContext("consumer").(string)
	if !ok {
		return types.ActionContinue
	}

	if adminMode == AdminModeRefresh {
		return refreshQuota(ctx, config, adminConsumer, string(body), log)
	}
	if adminMode == AdminModeDelta {
		return deltaQuota(ctx, config, adminConsumer, string(body), log)
	}

	return types.ActionContinue
}

func onHttpStreamingResponseBody(ctx wrapper.HttpContext, config QuotaConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	chatMode, ok := ctx.GetContext("chatMode").(ChatMode)
	if !ok {
		return data
	}
	if chatMode == ChatModeNone || chatMode == ChatModeAdmin {
		return data
	}
	// chat completion mode
	if !endOfStream {
		return data
	}
	inputTokenStr, err := proxywasm.GetProperty([]string{"filter_state", "wasm.input_token"})
	if err != nil {
		return data
	}
	outputTokenStr, err := proxywasm.GetProperty([]string{"filter_state", "wasm.output_token"})
	if err != nil {
		return data
	}
	inputToken, err := strconv.Atoi(string(inputTokenStr))
	if err != nil {
		return data
	}
	outputToken, err := strconv.Atoi(string(outputTokenStr))
	if err != nil {
		return data
	}
	consumer, ok := ctx.GetContext("consumer").(string)
	if ok {
		totalToken := int(inputToken + outputToken)
		log.Debugf("update consumer:%s, totalToken:%d", consumer, totalToken)
		config.redisClient.DecrBy(config.RedisKeyPrefix+consumer, totalToken, nil)
	}
	return data
}

func deniedNoKeyAuthData() types.Action {
	util.SendResponse(http.StatusUnauthorized, "ai-quota.no_key", "text/plain", "Request denied by ai quota check. No Key Authentication information found.")
	return types.ActionContinue
}

func deniedUnauthorizedConsumer() types.Action {
	util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. Unauthorized consumer.")
	return types.ActionContinue
}

func getOperationMode(path string, adminPath string, log wrapper.Log) (ChatMode, AdminMode) {
	fullAdminPath := "/v1/chat/completions" + adminPath
	if strings.HasSuffix(path, fullAdminPath+"/refresh") {
		return ChatModeAdmin, AdminModeRefresh
	}
	if strings.HasSuffix(path, fullAdminPath+"/delta") {
		return ChatModeAdmin, AdminModeDelta
	}
	if strings.HasSuffix(path, fullAdminPath) {
		return ChatModeAdmin, AdminModeQuery
	}
	if strings.HasSuffix(path, "/v1/chat/completions") {
		return ChatModeCompletion, AdminModeNone
	}
	return ChatModeNone, AdminModeNone
}

func refreshQuota(ctx wrapper.HttpContext, config QuotaConfig, adminConsumer string, body string, log wrapper.Log) types.Action {
	// check consumer
	if adminConsumer != config.AdminConsumer {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. Unauthorized admin consumer.")
		return types.ActionContinue
	}

	queryValues, _ := url.ParseQuery(body)
	values := make(map[string]string, len(queryValues))
	for k, v := range queryValues {
		values[k] = v[0]
	}
	queryConsumer := values["consumer"]
	quota, err := strconv.Atoi(values["quota"])
	if queryConsumer == "" || err != nil {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. consumer can't be empty and quota must be integer.")
		return types.ActionContinue
	}
	err2 := config.redisClient.Set(config.RedisKeyPrefix+queryConsumer, quota, func(response resp.Value) {
		log.Debugf("Redis set key = %s quota = %d", config.RedisKeyPrefix+queryConsumer, quota)
		if err := response.Error(); err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return
		}
		util.SendResponse(http.StatusOK, "ai-quota.refreshquota", "text/plain", "refresh quota successful")
	})

	if err2 != nil {
		util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
		return types.ActionContinue
	}

	return types.ActionPause
}
func queryQuota(ctx wrapper.HttpContext, config QuotaConfig, adminConsumer string, url *url.URL, log wrapper.Log) types.Action {
	// check consumer
	if adminConsumer != config.AdminConsumer {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. Unauthorized admin consumer.")
		return types.ActionContinue
	}
	// check url
	queryValues := url.Query()
	values := make(map[string]string, len(queryValues))
	for k, v := range queryValues {
		values[k] = v[0]
	}
	if values["consumer"] == "" {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. consumer can't be empty.")
		return types.ActionContinue
	}
	queryConsumer := values["consumer"]
	err := config.redisClient.Get(config.RedisKeyPrefix+queryConsumer, func(response resp.Value) {
		quota := 0
		if err := response.Error(); err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return
		} else if response.IsNull() {
			quota = 0
		} else {
			quota = response.Integer()
		}
		result := struct {
			Consumer string `json:"consumer"`
			Quota    int    `json:"quota"`
		}{
			Consumer: queryConsumer,
			Quota:    quota,
		}
		body, _ := json.Marshal(result)
		util.SendResponse(http.StatusOK, "ai-quota.queryquota", "application/json", string(body))
	})
	if err != nil {
		util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
		return types.ActionContinue
	}
	return types.ActionPause
}
func deltaQuota(ctx wrapper.HttpContext, config QuotaConfig, adminConsumer string, body string, log wrapper.Log) types.Action {
	// check consumer
	if adminConsumer != config.AdminConsumer {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. Unauthorized admin consumer.")
		return types.ActionContinue
	}

	queryValues, _ := url.ParseQuery(body)
	values := make(map[string]string, len(queryValues))
	for k, v := range queryValues {
		values[k] = v[0]
	}
	queryConsumer := values["consumer"]
	value, err := strconv.Atoi(values["value"])
	if queryConsumer == "" || err != nil {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. consumer can't be empty and value must be integer.")
		return types.ActionContinue
	}

	if value >= 0 {
		err := config.redisClient.IncrBy(config.RedisKeyPrefix+queryConsumer, value, func(response resp.Value) {
			log.Debugf("Redis Incr key = %s value = %d", config.RedisKeyPrefix+queryConsumer, value)
			if err := response.Error(); err != nil {
				util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
				return
			}
			util.SendResponse(http.StatusOK, "ai-quota.deltaquota", "text/plain", "delta quota successful")
		})
		if err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return types.ActionContinue
		}
	} else {
		err := config.redisClient.DecrBy(config.RedisKeyPrefix+queryConsumer, 0-value, func(response resp.Value) {
			log.Debugf("Redis Decr key = %s value = %d", config.RedisKeyPrefix+queryConsumer, 0-value)
			if err := response.Error(); err != nil {
				util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
				return
			}
			util.SendResponse(http.StatusOK, "ai-quota.deltaquota", "text/plain", "delta quota successful")
		})
		if err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return types.ActionContinue
		}
	}

	return types.ActionPause
}
