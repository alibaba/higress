package main

import (
	"errors"
	"fmt"
	"strings"

	re "regexp"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/zmap/go-iptree/iptree"
)

// 限流规则项类型
type limitRuleItemType string

// 限流配置项key类型
type limitConfigItemType string

const (
	limitByHeaderType      limitRuleItemType = "limit_by_header"
	limitByParamType       limitRuleItemType = "limit_by_param"
	limitByConsumerType    limitRuleItemType = "limit_by_consumer"
	limitByCookieType      limitRuleItemType = "limit_by_cookie"
	limitByPerHeaderType   limitRuleItemType = "limit_by_per_header"
	limitByPerParamType    limitRuleItemType = "limit_by_per_param"
	limitByPerConsumerType limitRuleItemType = "limit_by_per_consumer"
	limitByPerCookieType   limitRuleItemType = "limit_by_per_cookie"
	limitByPerIpType       limitRuleItemType = "limit_by_per_ip"

	exactType  limitConfigItemType = "exact"  // 精确匹配
	regexpType limitConfigItemType = "regexp" // 正则表达式
	allType    limitConfigItemType = "*"      // 匹配所有情况
	ipNetType  limitConfigItemType = "ipNet"  // ip段

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
	"token_per_second": Second,
	"token_per_minute": SecondsPerMinute,
	"token_per_hour":   SecondsPerHour,
	"token_per_day":    SecondsPerDay,
}

type ClusterKeyRateLimitConfig struct {
	ruleName             string          // 限流规则名称
	ruleItems            []LimitRuleItem // 限流规则项
	showLimitQuotaHeader bool            // 响应头中是否显示X-RateLimit-Limit和X-RateLimit-Remaining
	rejectedCode         uint32          // 当请求超过阈值被拒绝时,返回的HTTP状态码
	rejectedMsg          string          // 当请求超过阈值被拒绝时,返回的响应体
	redisClient          wrapper.RedisClient
	counterMetrics       map[string]proxywasm.MetricCounter // Metrics
}

type LimitRuleItem struct {
	limitType    limitRuleItemType // 限流类型
	key          string            // 根据该key值进行限流,limit_by_consumer和limit_by_per_consumer两种类型为ConsumerHeader,其他类型为对应的key值
	limitByPerIp LimitByPerIp      // 对端ip地址或ip段
	configItems  []LimitConfigItem // 限流配置项
}

type LimitByPerIp struct {
	sourceType string // ip来源类型
	headerName string // 根据该请求头获取客户端ip
}

type LimitConfigItem struct {
	configType limitConfigItemType // 限流配置项key类型
	key        string              // 限流key
	ipNet      *iptree.IPTree      // 限流key转换的ip地址或者ip段,仅用于itemType为ipNetType
	regexp     *re.Regexp          // 正则表达式,仅用于itemType为regexpType
	count      int64               // 指定时间窗口内的总请求数量阈值
	timeWindow int64               // 时间窗口大小
}

func initRedisClusterClient(json gjson.Result, config *ClusterKeyRateLimitConfig, log log.Log) error {
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
	database := int(redisConfig.Get("database").Int())
	err := config.redisClient.Init(username, password, int64(timeout), wrapper.WithDataBase(database))
	if config.redisClient.Ready() {
		log.Info("redis init successfully")
	} else {
		log.Error("redis init failed, will try later")
	}
	return err
}

func parseClusterKeyRateLimitConfig(json gjson.Result, config *ClusterKeyRateLimitConfig) error {
	ruleName := json.Get("rule_name")
	if !ruleName.Exists() {
		return errors.New("missing rule_name in config")
	}
	config.ruleName = ruleName.String()

	// 初始化ruleItems
	err := initRuleItems(json, config)
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
	if rejectedMsg.Exists() {
		config.rejectedMsg = rejectedMsg.String()
	} else {
		config.rejectedMsg = DefaultRejectedMsg
	}
	return nil
}

func initRuleItems(json gjson.Result, config *ClusterKeyRateLimitConfig) error {
	ruleItemsResult := json.Get("rule_items")
	if !ruleItemsResult.Exists() {
		return errors.New("missing rule_items in config")
	}
	if len(ruleItemsResult.Array()) == 0 {
		return errors.New("config rule_items cannot be empty")
	}
	var ruleItems []LimitRuleItem
	for _, item := range ruleItemsResult.Array() {
		var ruleItem LimitRuleItem

		// 根据配置区分限流类型
		var limitType limitRuleItemType
		setLimitByKeyIfExists := func(field gjson.Result, limitTypeStr limitRuleItemType) {
			if field.Exists() && field.String() != "" {
				ruleItem.key = field.String()
				limitType = limitTypeStr
			}
		}
		setLimitByKeyIfExists(item.Get("limit_by_header"), limitByHeaderType)
		setLimitByKeyIfExists(item.Get("limit_by_param"), limitByParamType)
		setLimitByKeyIfExists(item.Get("limit_by_cookie"), limitByCookieType)
		setLimitByKeyIfExists(item.Get("limit_by_per_header"), limitByPerHeaderType)
		setLimitByKeyIfExists(item.Get("limit_by_per_param"), limitByPerParamType)
		setLimitByKeyIfExists(item.Get("limit_by_per_cookie"), limitByPerCookieType)

		limitByConsumer := item.Get("limit_by_consumer")
		if limitByConsumer.Exists() {
			ruleItem.key = ConsumerHeader
			limitType = limitByConsumerType
		}
		limitByPerConsumer := item.Get("limit_by_per_consumer")
		if limitByPerConsumer.Exists() {
			ruleItem.key = ConsumerHeader
			limitType = limitByPerConsumerType
		}

		limitByPerIpResult := item.Get("limit_by_per_ip")
		if limitByPerIpResult.Exists() && limitByPerIpResult.String() != "" {
			limitByPerIp := limitByPerIpResult.String()
			ruleItem.key = limitByPerIp
			if strings.HasPrefix(limitByPerIp, "from-header-") {
				headerName := limitByPerIp[len("from-header-"):]
				if headerName == "" {
					return errors.New("limit_by_per_ip parse error: empty after 'from-header-'")
				}
				ruleItem.limitByPerIp = LimitByPerIp{
					sourceType: HeaderSourceType,
					headerName: headerName,
				}
			} else if limitByPerIp == "from-remote-addr" {
				ruleItem.limitByPerIp = LimitByPerIp{
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
		ruleItem.limitType = limitType

		// 初始化configItems
		err := initConfigItems(item, &ruleItem)
		if err != nil {
			return err
		}

		ruleItems = append(ruleItems, ruleItem)
	}
	config.ruleItems = ruleItems
	return nil
}

func initConfigItems(json gjson.Result, rule *LimitRuleItem) error {
	limitKeys := json.Get("limit_keys")
	if !limitKeys.Exists() {
		return errors.New("missing limit_keys in config")
	}
	if len(limitKeys.Array()) == 0 {
		return errors.New("config limit_keys cannot be empty")
	}
	var configItems []LimitConfigItem
	for _, item := range limitKeys.Array() {
		key := item.Get("key")
		if !key.Exists() || key.String() == "" {
			return errors.New("limit_keys key is required")
		}

		var (
			itemKey  = key.String()
			itemType limitConfigItemType
			ipNet    *iptree.IPTree
			regexp   *re.Regexp
		)
		if rule.limitType == limitByPerIpType {
			var err error
			ipNet, err = parseIPNet(itemKey)
			if err != nil {
				return fmt.Errorf("failed to parse IPNet for key '%s': %w", itemKey, err)
			}
			itemType = ipNetType
		} else if rule.limitType == limitByPerHeaderType ||
			rule.limitType == limitByPerParamType ||
			rule.limitType == limitByPerConsumerType ||
			rule.limitType == limitByPerCookieType {
			if itemKey == "*" {
				itemType = allType
			} else if strings.HasPrefix(itemKey, "regexp:") {
				regexpStr := itemKey[len("regexp:"):]
				var err error
				regexp, err = re.Compile(regexpStr)
				if err != nil {
					return fmt.Errorf("failed to compile regex for key '%s': %w", itemKey, err)
				}
				itemType = regexpType
			} else {
				return fmt.Errorf("the '%s' restriction must start with 'regexp:' or be exactly '*'", rule.limitType)
			}
		} else {
			itemType = exactType
		}

		if configItem, err := createConfigItemFromRate(item, itemType, itemKey, ipNet, regexp); err != nil {
			return err
		} else if configItem != nil {
			configItems = append(configItems, *configItem)
		}
	}
	rule.configItems = configItems
	return nil
}

func createConfigItemFromRate(item gjson.Result, itemType limitConfigItemType, key string, ipNet *iptree.IPTree, regexp *re.Regexp) (*LimitConfigItem, error) {
	for timeWindowKey, duration := range timeWindows {
		q := item.Get(timeWindowKey)
		if q.Exists() && q.Int() > 0 {
			return &LimitConfigItem{
				configType: itemType,
				key:        key,
				ipNet:      ipNet,
				regexp:     regexp,
				count:      q.Int(),
				timeWindow: duration,
			}, nil
		}
	}
	return nil, errors.New("one of 'token_per_second', 'token_per_minute', 'token_per_hour', or 'token_per_day' must be set for key: " + key)
}
