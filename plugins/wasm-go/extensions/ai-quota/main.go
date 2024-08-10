package main

import (
	"bytes"
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
	pluginName                 = "ai-quota"
	defaultMaxBodyBytes uint32 = 10 * 1024 * 1024
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
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamingBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

type QuotaConfig struct {
	Keys      []string   `yaml:"keys"` // key auth names
	InQuery   bool       `yaml:"in_query,omitempty"`
	InHeader  bool       `yaml:"in_header,omitempty"`
	consumers []Consumer `yaml:"consumers"`

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
	log.Debug("parse config()")
	// init
	config.credential2Name = make(map[string]string)
	// keys
	names := json.Get("keys")
	if !names.Exists() {
		return errors.New("keys is required")
	}
	if len(names.Array()) == 0 {
		return errors.New("keys cannot be empty")
	}

	for _, name := range names.Array() {
		config.Keys = append(config.Keys, name.String())
	}

	// in_query and in_header
	in_query := json.Get("in_query")
	in_header := json.Get("in_header")
	if !in_query.Exists() && !in_header.Exists() {
		return errors.New("must one of in_query/in_header required")
	}
	if in_query.Exists() {
		config.InQuery = in_query.Bool()
	}
	if in_header.Exists() {
		config.InHeader = in_header.Bool()
	}
	// consumers
	consumers := json.Get("consumers")
	if !consumers.Exists() {
		return errors.New("consumers is required")
	}
	if len(consumers.Array()) == 0 {
		return errors.New("consumers cannot be empty")
	}

	for _, item := range consumers.Array() {
		name := item.Get("name")
		if !name.Exists() || name.String() == "" {
			return errors.New("consumer name is required")
		}
		credential := item.Get("credential")
		if !credential.Exists() || credential.String() == "" {
			return errors.New("consumer credential is required")
		}
		if _, ok := config.credential2Name[credential.String()]; ok {
			return errors.New("duplicate consumer credential: " + credential.String())
		}

		consumer := Consumer{
			Name:       name.String(),
			Credential: credential.String(),
		}
		config.consumers = append(config.consumers, consumer)
		config.credential2Name[credential.String()] = name.String()
	}

	// admin
	config.AdminPath = json.Get("admin_path").String()
	config.AdminConsumer = json.Get("admin_consumer").String()
	// Redis
	config.RedisKeyPrefix = json.Get("redis_key_prefix").String()
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

	log.Infof("parse result:%+v", config)
	return config.redisClient.Init(username, password, int64(timeout))
}

func onHttpRequestHeaders(context wrapper.HttpContext, config QuotaConfig, log wrapper.Log) types.Action {
	// get tokens
	var tokens []string
	if config.InHeader {
		// 匹配keys中的 keyname
		for _, key := range config.Keys {
			value, err := proxywasm.GetHttpRequestHeader(key)
			if err == nil && value != "" {
				tokens = append(tokens, value)
			}
		}
	} else if config.InQuery {
		requestUrl, _ := proxywasm.GetHttpRequestHeader(":path")
		url, _ := url.Parse(requestUrl)
		queryValues := url.Query()
		for _, key := range config.Keys {
			values, ok := queryValues[key]
			if ok && len(values) > 0 {
				tokens = append(tokens, values...)
			}
		}
	}
	// header/query
	if len(tokens) > 1 {
		return deniedMultiKeyAuthData()
	} else if len(tokens) <= 0 {
		return deniedNoKeyAuthData()
	}
	// 验证token
	consumer, ok := config.credential2Name[tokens[0]]
	if !ok {
		log.Warnf("credential %q is not configured", tokens[0])
		return deniedUnauthorizedConsumer()
	}

	rawPath := context.Path()
	path, _ := url.Parse(rawPath)
	chatMode, adminMode := getOperationMode(path.Path)
	context.SetContext("chatMode", chatMode)
	context.SetContext("adminMode", adminMode)
	context.SetContext("consumer", consumer)
	if chatMode == ChatModeNone {
		return types.ActionContinue
	}

	if chatMode == ChatModeAdmin {
		context.BufferResponseBody()
		// refresh
		if adminMode == AdminModeRefresh {
			return QueryQuota(context, config, consumer, path, log)
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
		if isDenied {
			util.SendResponse(http.StatusForbidden, "ai-quota.noquota", "text/plain", "Request denied by ai quota check, No quota left")
		}
		proxywasm.ResumeHttpRequest()
	})
	return types.ActionPause
}

func onHttpRequestBody(ctx wrapper.HttpContext, config QuotaConfig, body []byte, log wrapper.Log) types.Action {
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
		return RefreshQuota(ctx, config, adminConsumer, string(body), log)
	}
	if adminMode == AdminModeDelta {
		return DeltaQuota(ctx, config, adminConsumer, string(body), log)
	}

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config QuotaConfig, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if !strings.Contains(contentType, "text/event-stream") {
		ctx.BufferResponseBody()
	}
	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config QuotaConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	chatMode, ok := ctx.GetContext("chatMode").(ChatMode)
	if !ok {
		return data
	}
	if chatMode == ChatModeNone || chatMode == ChatModeAdmin {
		return data
	}

	_, inputToken, outputToken, ok := getUsage(data)
	if !ok {
		return data
	}
	consumer, ok := ctx.GetContext("consumer").(string)
	if ok {
		totalToken := int(inputToken + outputToken)
		config.redisClient.DecrBy(config.RedisKeyPrefix+consumer, totalToken, nil)
	}
	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config QuotaConfig, body []byte, log wrapper.Log) types.Action {
	chatMode, ok := ctx.GetContext("chatMode").(ChatMode)
	if !ok {
		return types.ActionContinue
	}
	if chatMode == ChatModeNone || chatMode == ChatModeAdmin {
		return types.ActionContinue
	}

	_, inputToken, outputToken, ok := getUsage(body)
	if !ok {
		return types.ActionContinue
	}
	consumer, ok := ctx.GetContext("consumer").(string)
	if ok {
		totalToken := int(inputToken + outputToken)
		config.redisClient.DecrBy(config.RedisKeyPrefix+consumer, totalToken, nil)
	}
	return types.ActionContinue
}

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

func deniedMultiKeyAuthData() types.Action {
	util.SendResponse(http.StatusUnauthorized, "ai-quota.multi_key", "text/plain", "Request denied by ai qutota check. Multi Key Authentication information found.")
	return types.ActionContinue
}

func deniedNoKeyAuthData() types.Action {
	util.SendResponse(http.StatusUnauthorized, "ai-quota.no_key", "text/plain", "Request denied by ai quota check. No Key Authentication information found.")
	return types.ActionContinue
}

func deniedUnauthorizedConsumer() types.Action {
	util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. Unauthorized consumer.")
	return types.ActionContinue
}

func getOperationMode(path string) (ChatMode, AdminMode) {
	if strings.HasSuffix(path, "/v1/chat/completions/quota/refresh") {
		return ChatModeAdmin, AdminModeRefresh
	}
	if strings.HasSuffix(path, "/v1/chat/completions/quota/delta") {
		return ChatModeAdmin, AdminModeDelta
	}
	if strings.HasSuffix(path, "/v1/chat/completions/quota") {
		return ChatModeAdmin, AdminModeQuery
	}
	if strings.HasSuffix(path, "/v1/chat/completions") {
		return ChatModeCompletion, AdminModeNone
	}
	return ChatModeNone, AdminModeNone
}

func RefreshQuota(ctx wrapper.HttpContext, config QuotaConfig, adminConsumer string, body string, log wrapper.Log) types.Action {
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
		defer func() {
			proxywasm.ResumeHttpRequest()
		}()
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
func QueryQuota(ctx wrapper.HttpContext, config QuotaConfig, adminConsumer string, url *url.URL, log wrapper.Log) types.Action {
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
		defer func() {
			proxywasm.ResumeHttpRequest()
		}()
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
			consumer string `json:"consumer"`
			quota    int    `json:"quota"`
		}{
			consumer: queryConsumer,
			quota:    quota,
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
func DeltaQuota(ctx wrapper.HttpContext, config QuotaConfig, adminConsumer string, body string, log wrapper.Log) types.Action {
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

	f := func(response resp.Value) {
		defer func() {
			proxywasm.ResumeHttpRequest()
		}()
		if err := response.Error(); err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return
		}
		util.SendResponse(http.StatusOK, "ai-quota.deltaquota", "text/plain", "delta quota successful")
	}

	if value >= 0 {
		err := config.redisClient.IncrBy(config.RedisKeyPrefix+queryConsumer, value, f)
		if err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return types.ActionContinue
		}
	} else {
		err := config.redisClient.DecrBy(config.RedisKeyPrefix+queryConsumer, 0-value, f)
		if err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return types.ActionContinue
		}
	}

	return types.ActionPause
}
