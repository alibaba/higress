package main

import (
	"errors"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/zmap/go-iptree/iptree"
)

// 限流规则类型
type limitRuleType string

const (
	limitByHeaderType limitRuleType = "limitByHeader"
	limitByParamType  limitRuleType = "limitByParam"
	limitByPerIpType  limitRuleType = "limitByPerIp"

	RemoteAddrSourceType = "remote-addr"
	HeaderSourceType     = "header"

	DefaultRejectedCode uint32 = 429
	DefaultRejectedMsg  string = "Too many requests"

	Second           int64 = 1
	SecondsPerMinute       = 60 * Second
	SecondsPerHour         = 60 * SecondsPerMinute
	SecondsPerDay          = 24 * SecondsPerHour
)

type ClusterKeyRateLimitConfig struct {
	ruleName      string        // 限流规则名称
	limitType     limitRuleType // 限流类型
	limitByHeader string        // 根据http请求头限流
	limitByParam  string        // 根据url参数限流
	limitByPerIp  LimitByPerIp  // 根据对端ip限流
	limitItems    []LimitItem   // 限流配置 key为限流的ip地址或者ip段
	rejectedCode  uint32        // 当请求超过阈值被拒绝时,返回的HTTP状态码
	rejectedMsg   string        // 当请求超过阈值被拒绝时,返回的响应体
	redisClient   wrapper.RedisClient
}

type LimitByPerIp struct {
	sourceType string // ip来源类型
	headerName string // 根据该请求头获取客户端ip
}

type LimitItem struct {
	key        string         // 限流key
	ipNet      *iptree.IPTree // 限流key转换的ip地址或者ip段
	count      int64          // 指定时间窗口内的总请求数量阈值
	timeWindow int64          // 时间窗口大小
}

func initRedisClusterClient(json gjson.Result, config *ClusterKeyRateLimitConfig) error {
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
	config.redisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: int64(servicePort),
	})
	return config.redisClient.Init(username, password, int64(timeout))
}

func parseClusterKeyRateLimitConfig(json gjson.Result, config *ClusterKeyRateLimitConfig, log wrapper.Log) error {
	ruleName := json.Get("rule_name")
	if !ruleName.Exists() {
		return errors.New("missing rule_name in config")
	}
	config.ruleName = ruleName.String()

	// 根据配置区分限流类型
	var limitType limitRuleType
	limitByHeader := json.Get("limit_by_header")
	if limitByHeader.Exists() && limitByHeader.String() != "" {
		config.limitByHeader = limitByHeader.String()
		limitType = limitByHeaderType
	}

	limitByParam := json.Get("limit_by_param")
	if limitByParam.Exists() && limitByParam.String() != "" {
		config.limitByParam = limitByParam.String()
		limitType = limitByParamType
	}

	limitByPerIpResult := json.Get("limit_by_per_ip")
	if limitByPerIpResult.Exists() && limitByPerIpResult.String() != "" {
		limitByPerIp := limitByPerIpResult.String()
		if strings.HasPrefix(limitByPerIp, "from-header-") {
			headerName := limitByPerIp[len("from-header-"):]
			if headerName == "" {
				return errors.New("limit_by_per_ip parse error: empty after 'from-header-'")
			}
			config.limitByPerIp = LimitByPerIp{
				sourceType: HeaderSourceType,
				headerName: headerName,
			}
		} else if limitByPerIp == "from-remote-addr" {
			config.limitByPerIp = LimitByPerIp{
				sourceType: RemoteAddrSourceType,
				headerName: "",
			}
		} else {
			return errors.New("the 'limit_by_per_ip' restriction must start with 'from-header-' or be exactly 'from-remote-addr'")
		}
		limitType = limitByPerIpType
	}
	if limitType == "" {
		return errors.New("only one of 'limit_by_header' and 'limit_by_param' and 'limit_by_per_ip' can be set")
	}
	config.limitType = limitType

	// 初始化LimitItem
	err := initLimitItems(json, config, log)
	if err != nil {
		return err
	}

	rejectedCode := json.Get("rejected_code")
	if rejectedCode.Exists() {
		config.rejectedCode = uint32(rejectedCode.Uint())
	} else {
		config.rejectedCode = DefaultRejectedCode
	}
	rejectedMsg := json.Get("rejected_msg")
	if rejectedCode.Exists() {
		config.rejectedMsg = rejectedMsg.String()
	} else {
		config.rejectedMsg = DefaultRejectedMsg
	}
	return nil
}

func initLimitItems(json gjson.Result, config *ClusterKeyRateLimitConfig, log wrapper.Log) error {
	limitKeys := json.Get("limit_keys")
	if !limitKeys.Exists() {
		return errors.New("missing limit_keys in config")
	}
	if len(limitKeys.Array()) == 0 {
		return errors.New("config limit_keys cannot be empty")
	}
	var limitItems []LimitItem
	for _, item := range limitKeys.Array() {
		key := item.Get("key")
		if !key.Exists() || key.String() == "" {
			return errors.New("limit_keys key is required")
		}
		var ipNet *iptree.IPTree
		if config.limitType == limitByPerIpType {
			var err error
			ipNet, err = parseIPNet(key.String())
			if err != nil {
				log.Errorf("parseIPNet error: %v", err)
				return err
			}
		} else {
			ipNet = nil
		}

		qps := item.Get("query_per_second")
		if qps.Exists() && qps.Int() > 0 {
			limitItems = append(limitItems, LimitItem{
				key:        key.String(),
				ipNet:      ipNet,
				count:      qps.Int(),
				timeWindow: Second,
			})
			continue
		}
		qpm := item.Get("query_per_minute")
		if qpm.Exists() && qpm.Int() > 0 {
			limitItems = append(limitItems, LimitItem{
				key:        key.String(),
				ipNet:      ipNet,
				count:      qpm.Int(),
				timeWindow: SecondsPerMinute,
			})
			continue
		}
		qph := item.Get("query_per_hour")
		if qph.Exists() && qph.Int() > 0 {
			limitItems = append(limitItems, LimitItem{
				key:        key.String(),
				ipNet:      ipNet,
				count:      qph.Int(),
				timeWindow: SecondsPerHour,
			})
			continue
		}
		qpd := item.Get("query_per_day")
		if qpd.Exists() && qpd.Int() > 0 {
			limitItems = append(limitItems, LimitItem{
				key:        key.String(),
				ipNet:      ipNet,
				count:      qpd.Int(),
				timeWindow: SecondsPerDay,
			})
			continue
		}
	}
	config.limitItems = limitItems
	return nil
}
