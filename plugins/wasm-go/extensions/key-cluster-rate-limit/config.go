package main

import (
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/zmap/go-iptree/iptree"
)

const (
	OriginSourceType = "origin-source"
	HeaderSourceType = "header"

	DefaultRealIpHeader string = "X-Forwarded-For"
	DefaultRejectedCode uint32 = 429
	DefaultRejectedMsg  string = "Too many requests"
	DefaultRedisTimeout int64  = 1000

	Second           int64 = 1
	SecondsPerMinute       = 60 * Second
	SecondsPerHour         = 60 * SecondsPerMinute
	SecondsPerDay          = 24 * SecondsPerHour
)

type KeyClusterRateLimitConfig struct {
	ruleName             string      // 限流规则名称
	ipSourceType         string      // ip来源类型
	ipHeaderName         string      // 根据该请求头获取客户端ip
	limitItems           []LimitItem // 限流配置 key为限流的ip地址或者ip段
	showLimitQuotaHeader bool        // 响应头中是否显示X-RateLimit-Limit和X-RateLimit-Remaining
	rejectedCode         uint32      // 当请求超过阈值被拒绝时,返回的HTTP状态码
	rejectedMsg          string      // 当请求超过阈值被拒绝时,返回的响应体
	client               wrapper.RedisClient
}

type LimitItem struct {
	key        string         // 限流key
	ipNet      *iptree.IPTree // 限流key转换的ip地址或者ip段
	count      int64          // 指定时间窗口内的总请求数量阈值
	timeWindow int64          // 时间窗口大小
}

func initRedisClient(json gjson.Result, config *KeyClusterRateLimitConfig, log wrapper.Log) error {
	serviceSource := json.Get("redis_service_source").String()
	serviceName := json.Get("redis_service_name").String()
	serviceHost := json.Get("redis_service_host").String()
	servicePort := json.Get("redis_service_port").Int()
	if serviceName == "" || servicePort == 0 {
		return errors.New("invalid redis service config")
	}
	switch serviceSource {
	case "ip":
		config.client = wrapper.NewRedisClusterClient(&wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Host:        serviceHost,
			Port:        servicePort,
		})
	case "dns":
		domain := json.Get("redis_service_domain").String()
		if domain == "" {
			return errors.New("missing redis_service_domain in config")
		}
		config.client = wrapper.NewRedisClusterClient(&wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      domain,
		})
	default:
		return errors.New("unknown service source: " + serviceSource)
	}
	username := json.Get("redis_username").String()
	password := json.Get("redis_password").String()
	var timeout int64
	redisTimeout := json.Get("redis_timeout")
	if redisTimeout.Exists() && redisTimeout.Int() > 0 {
		timeout = redisTimeout.Int()
	} else {
		timeout = DefaultRedisTimeout
	}
	err := config.client.Init(username, password, timeout)
	if err != nil {
		return errors.New(fmt.Sprintf("redisClient init error: %v", err))
	}
	return nil
}

func parseClusterRateLimitConfig(json gjson.Result, config *KeyClusterRateLimitConfig, log wrapper.Log) error {
	ruleName := json.Get("rule_name")
	if !ruleName.Exists() {
		return errors.New("missing rule_name in config")
	}
	config.ruleName = ruleName.String()

	sourceType := json.Get("ip_source_type")
	if sourceType.Exists() && sourceType.String() != "" {
		switch sourceType.String() {
		case HeaderSourceType:
			config.ipSourceType = HeaderSourceType
		case OriginSourceType:
		default:
			config.ipSourceType = OriginSourceType
		}
	} else {
		config.ipSourceType = OriginSourceType
	}

	header := json.Get("ip_header_name")
	if header.Exists() && header.String() != "" {
		config.ipHeaderName = header.String()
	} else {
		config.ipHeaderName = DefaultRealIpHeader
	}

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
		ipNet, err := parseIPNet(key.String())
		if err != nil {
			log.Errorf("parseIPNet error: %v", err)
			return err
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

	showLimitQuotaHeader := json.Get("show_limit_quota_header")
	if showLimitQuotaHeader.Exists() {
		config.showLimitQuotaHeader = showLimitQuotaHeader.Bool()
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
