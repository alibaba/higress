package main

import (
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	regexp "github.com/wasilibs/go-re2"
	"github.com/zmap/go-iptree/iptree"
	"strings"
)

// 限流规则类型
type limitRuleType string

// 限流配置项key类型
type limitItemType string

const (
	limitByHeaderType      limitRuleType = "limit_by_header"
	limitByParamType       limitRuleType = "limit_by_param"
	limitByConsumerType    limitRuleType = "limit_by_consumer"
	limitByCookieType      limitRuleType = "limit_by_cookie"
	limitByPerHeaderType   limitRuleType = "limit_by_per_header"
	limitByPerParamType    limitRuleType = "limit_by_per_param"
	limitByPerConsumerType limitRuleType = "limit_by_per_consumer"
	limitByPerCookieType   limitRuleType = "limit_by_per_cookie"
	limitByPerIpType       limitRuleType = "limit_by_per_ip"

	exactType  limitItemType = "exact"  // 精确匹配
	regexpType limitItemType = "regexp" // 正则表达式
	allType    limitItemType = "*"      // 匹配所有情况
	ipNetType  limitItemType = "ipNet"  // ip段

	RemoteAddrSourceType = "remote-addr"
	HeaderSourceType     = "header"

	DefaultRejectedCode uint32 = 429
	DefaultRejectedMsg  string = "Too many requests"

	Second           int64 = 1
	SecondsPerMinute       = 60 * Second
	SecondsPerHour         = 60 * SecondsPerMinute
	SecondsPerDay          = 24 * SecondsPerHour
)

var timeWindows = map[string]int64{
	"query_per_second": Second,
	"query_per_minute": SecondsPerMinute,
	"query_per_hour":   SecondsPerHour,
	"query_per_day":    SecondsPerDay,
}

type ClusterKeyRateLimitConfig struct {
	ruleName             string        // 限流规则名称
	limitType            limitRuleType // 限流类型
	limitByKey           string        // 根据limitType对应的键名:http头名称、url参数名称、cookie名称
	limitByPerIp         LimitByPerIp  // 对端ip地址或ip段
	limitItems           []LimitItem   // 限流配置
	showLimitQuotaHeader bool          // 响应头中是否显示X-RateLimit-Limit和X-RateLimit-Remaining
	rejectedCode         uint32        // 当请求超过阈值被拒绝时,返回的HTTP状态码
	rejectedMsg          string        // 当请求超过阈值被拒绝时,返回的响应体
	redisClient          wrapper.RedisClient
}

type LimitByPerIp struct {
	sourceType string // ip来源类型
	headerName string // 根据该请求头获取客户端ip
}

type LimitItem struct {
	itemType   limitItemType  // 限流配置项key类型
	key        string         // 限流key
	ipNet      *iptree.IPTree // 限流key转换的ip地址或者ip段,仅用于itemType为ipNetType
	re         *regexp.Regexp // 正则表达式,仅用于itemType为regexpType
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

func parseClusterKeyRateLimitConfig(json gjson.Result, config *ClusterKeyRateLimitConfig) error {
	ruleName := json.Get("rule_name")
	if !ruleName.Exists() {
		return errors.New("missing rule_name in config")
	}
	config.ruleName = ruleName.String()

	// 根据配置区分限流类型
	var limitType limitRuleType

	setLimitByKeyIfExists := func(field gjson.Result, limitTypeStr limitRuleType) {
		if field.Exists() && field.String() != "" {
			config.limitByKey = field.String()
			limitType = limitTypeStr
		}
	}
	setLimitByKeyIfExists(json.Get("limit_by_header"), limitByHeaderType)
	setLimitByKeyIfExists(json.Get("limit_by_param"), limitByParamType)
	setLimitByKeyIfExists(json.Get("limit_by_cookie"), limitByCookieType)
	setLimitByKeyIfExists(json.Get("limit_by_per_header"), limitByPerHeaderType)
	setLimitByKeyIfExists(json.Get("limit_by_per_param"), limitByPerParamType)
	setLimitByKeyIfExists(json.Get("limit_by_per_cookie"), limitByPerCookieType)

	limitByConsumer := json.Get("limit_by_consumer")
	if limitByConsumer.Exists() {
		limitType = limitByConsumerType
	}
	limitByPerConsumer := json.Get("")
	if limitByPerConsumer.Exists() {
		limitType = limitByPerConsumerType
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
		return errors.New("only one of 'limit_by_header' and 'limit_by_param' and 'limit_by_consumer' and 'limit_by_cookie' and 'limit_by_per_header' and 'limit_by_per_param' and 'limit_by_per_consumer' and 'limit_by_per_cookie' and 'limit_by_per_ip' can be set")
	}
	config.limitType = limitType

	// 初始化LimitItem
	err := initLimitItems(json, config)
	if err != nil {
		return err
	}

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

func initLimitItems(json gjson.Result, config *ClusterKeyRateLimitConfig) error {
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

		var (
			itemKey  = key.String()
			itemType limitItemType
			ipNet    *iptree.IPTree
			re       *regexp.Regexp
		)
		if config.limitType == limitByPerIpType {
			var err error
			ipNet, err = parseIPNet(itemKey)
			if err != nil {
				return fmt.Errorf("failed to parse IPNet for key '%s': %w", itemKey, err)
			}
			itemType = ipNetType
		} else if config.limitType == limitByPerHeaderType ||
			config.limitType == limitByPerParamType ||
			config.limitType == limitByPerConsumerType ||
			config.limitType == limitByPerCookieType {
			if itemKey == "*" {
				itemType = allType
			} else if strings.HasPrefix(itemKey, "regexp:") {
				regexpStr := itemKey[len("regexp:"):]
				var err error
				re, err = regexp.Compile(regexpStr)
				if err != nil {
					return fmt.Errorf("failed to compile regex for key '%s': %w", itemKey, err)
				}
				itemType = regexpType
			} else {
				return fmt.Errorf("the '%s' restriction must start with 'regexp:' or be exactly '*'", config.limitType)
			}
		} else {
			itemType = exactType
		}

		if limitItem, err := createLimitItemFromRate(item, itemType, itemKey, ipNet, re); err != nil {
			return err
		} else if limitItem != nil {
			limitItems = append(limitItems, *limitItem)
		}
	}
	config.limitItems = limitItems
	return nil
}

func createLimitItemFromRate(item gjson.Result, itemType limitItemType, key string, ipNet *iptree.IPTree, re *regexp.Regexp) (*LimitItem, error) {
	for timeWindowKey, duration := range timeWindows {
		q := item.Get(timeWindowKey)
		if q.Exists() && q.Int() > 0 {
			return &LimitItem{
				itemType:   itemType,
				key:        key,
				ipNet:      ipNet,
				re:         re,
				count:      q.Int(),
				timeWindow: duration,
			}, nil
		}
	}
	return nil, errors.New("one of 'query_per_second', 'query_per_minute', 'query_per_hour', or 'query_per_day' must be set for key: " + key)
}
