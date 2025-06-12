package config

import (
	"errors"
	"fmt"
	"strings"

	"cluster-key-rate-limit/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	re "github.com/wasilibs/go-re2"
	"github.com/zmap/go-iptree/iptree"
)

// LimitRuleItemType 限流规则项类型
type LimitRuleItemType string

// LimitConfigItemType 限流配置项key类型
type LimitConfigItemType string

const (
	LimitByHeaderType      LimitRuleItemType = "limit_by_header"
	LimitByParamType       LimitRuleItemType = "limit_by_param"
	LimitByConsumerType    LimitRuleItemType = "limit_by_consumer"
	LimitByCookieType      LimitRuleItemType = "limit_by_cookie"
	LimitByPerHeaderType   LimitRuleItemType = "limit_by_per_header"
	LimitByPerParamType    LimitRuleItemType = "limit_by_per_param"
	LimitByPerConsumerType LimitRuleItemType = "limit_by_per_consumer"
	LimitByPerCookieType   LimitRuleItemType = "limit_by_per_cookie"
	LimitByPerIpType       LimitRuleItemType = "limit_by_per_ip"

	ExactType  LimitConfigItemType = "exact"  // 精确匹配
	RegexpType LimitConfigItemType = "regexp" // 正则表达式
	AllType    LimitConfigItemType = "*"      // 匹配所有情况
	IpNetType  LimitConfigItemType = "ipNet"  // ip段

	ConsumerHeader = "x-mse-consumer" // LimitByConsumer从该request header获取consumer的名字

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
	RuleName             string           // 限流规则名称
	GlobalThreshold      *GlobalThreshold // 全局限流配置
	RuleItems            []LimitRuleItem  // 限流规则项
	ShowLimitQuotaHeader bool             // 响应头中是否显示X-RateLimit-Limit和X-RateLimit-Remaining
	RejectedCode         uint32           // 当请求超过阈值被拒绝时,返回的HTTP状态码
	RejectedMsg          string           // 当请求超过阈值被拒绝时,返回的响应体
	RedisClient          wrapper.RedisClient
}

type GlobalThreshold struct {
	Count      int64 // 时间窗口内请求数
	TimeWindow int64 // 时间窗口大小(秒)
}

type LimitRuleItem struct {
	LimitType    LimitRuleItemType // 限流类型
	Key          string            // 根据该key值进行限流,limit_by_consumer和limit_by_per_consumer两种类型为ConsumerHeader,其他类型为对应的key值
	LimitByPerIp LimitByPerIp      // 对端ip地址或ip段
	ConfigItems  []LimitConfigItem // 限流配置项
}

type LimitByPerIp struct {
	SourceType string // ip来源类型
	HeaderName string // 根据该请求头获取客户端ip
}

type LimitConfigItem struct {
	ConfigType LimitConfigItemType // 限流配置项key类型
	Key        string              // 限流key
	IpNet      *iptree.IPTree      // 限流key转换的ip地址或者ip段,仅用于itemType为ipNetType
	Regexp     *re.Regexp          // 正则表达式,仅用于itemType为regexpType
	Count      int64               // 指定时间窗口内的总请求数量阈值
	TimeWindow int64               // 时间窗口大小
}

func InitRedisClusterClient(json gjson.Result, config *ClusterKeyRateLimitConfig) error {
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

	config.RedisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: int64(servicePort),
	})
	database := int(redisConfig.Get("database").Int())
	return config.RedisClient.Init(username, password, int64(timeout), wrapper.WithDataBase(database))
}

func ParseClusterKeyRateLimitConfig(json gjson.Result, config *ClusterKeyRateLimitConfig) error {
	ruleName := json.Get("rule_name")
	if !ruleName.Exists() {
		return errors.New("missing rule_name in config")
	}
	config.RuleName = ruleName.String()

	// 初始化限流规则
	if err := initLimitRule(json, config); err != nil {
		return err
	}

	showLimitQuotaHeader := json.Get("show_limit_quota_header")
	if showLimitQuotaHeader.Exists() {
		config.ShowLimitQuotaHeader = showLimitQuotaHeader.Bool()
	}

	rejectedCode := json.Get("rejected_code")
	if rejectedCode.Exists() {
		config.RejectedCode = uint32(rejectedCode.Uint())
	} else {
		config.RejectedCode = DefaultRejectedCode
	}

	rejectedMsg := json.Get("rejected_msg")
	if rejectedMsg.Exists() {
		config.RejectedMsg = rejectedMsg.String()
	} else {
		config.RejectedMsg = DefaultRejectedMsg
	}
	return nil
}

func initLimitRule(json gjson.Result, config *ClusterKeyRateLimitConfig) error {
	globalThresholdResult := json.Get("global_threshold")
	ruleItemsResult := json.Get("rule_items")

	hasGlobal := globalThresholdResult.Exists()
	hasRule := ruleItemsResult.Exists()
	if !hasGlobal && !hasRule {
		return errors.New("at least one of 'global_threshold' or 'rule_items' must be set")
	} else if hasGlobal && hasRule {
		return errors.New("'global_threshold' and 'rule_items' cannot be set at the same time")
	}

	// 处理全局限流配置
	if hasGlobal {
		threshold, err := parseGlobalThreshold(globalThresholdResult)
		if err != nil {
			return fmt.Errorf("failed to parse global_threshold: %w", err)
		}
		config.GlobalThreshold = threshold
		return nil
	}

	// 处理条件限流规则
	items := ruleItemsResult.Array()
	if len(items) == 0 {
		return errors.New("config rule_items cannot be empty")
	}

	var ruleItems []LimitRuleItem
	for _, item := range items {
		ruleItem, err := parseLimitRuleItem(item)
		if err != nil {
			return fmt.Errorf("failed to parse rule_item in rule_items: %w", err)
		}
		ruleItems = append(ruleItems, *ruleItem)
	}
	config.RuleItems = ruleItems
	return nil
}

func parseGlobalThreshold(item gjson.Result) (*GlobalThreshold, error) {
	for timeWindowKey, duration := range timeWindows {
		q := item.Get(timeWindowKey)
		if q.Exists() && q.Int() > 0 {
			return &GlobalThreshold{
				Count:      q.Int(),
				TimeWindow: duration,
			}, nil
		}
	}
	return nil, errors.New("one of 'query_per_second', 'query_per_minute', 'query_per_hour', or 'query_per_day' must be set for global_threshold")
}

func parseLimitRuleItem(item gjson.Result) (*LimitRuleItem, error) {
	var ruleItem LimitRuleItem
	// 根据配置区分限流类型
	var limitType LimitRuleItemType

	trySetLimitType := func(field gjson.Result, limitTypeStr LimitRuleItemType) {
		if field.Exists() && field.String() != "" {
			ruleItem.Key = field.String()
			limitType = limitTypeStr
		}
	}
	trySetLimitType(item.Get("limit_by_header"), LimitByHeaderType)
	trySetLimitType(item.Get("limit_by_param"), LimitByParamType)
	trySetLimitType(item.Get("limit_by_cookie"), LimitByCookieType)
	trySetLimitType(item.Get("limit_by_per_header"), LimitByPerHeaderType)
	trySetLimitType(item.Get("limit_by_per_param"), LimitByPerParamType)
	trySetLimitType(item.Get("limit_by_per_cookie"), LimitByPerCookieType)

	limitByConsumer := item.Get("limit_by_consumer")
	if limitByConsumer.Exists() {
		ruleItem.Key = ConsumerHeader
		limitType = LimitByConsumerType
	}
	limitByPerConsumer := item.Get("limit_by_per_consumer")
	if limitByPerConsumer.Exists() {
		ruleItem.Key = ConsumerHeader
		limitType = LimitByPerConsumerType
	}

	limitByPerIpResult := item.Get("limit_by_per_ip")
	if limitByPerIpResult.Exists() && limitByPerIpResult.String() != "" {
		limitByPerIp := limitByPerIpResult.String()
		ruleItem.Key = limitByPerIp
		if strings.HasPrefix(limitByPerIp, "from-header-") {
			headerName := limitByPerIp[len("from-header-"):]
			if headerName == "" {
				return nil, errors.New("limit_by_per_ip parse error: empty after 'from-header-'")
			}
			ruleItem.LimitByPerIp = LimitByPerIp{
				SourceType: HeaderSourceType,
				HeaderName: headerName,
			}
		} else if limitByPerIp == "from-remote-addr" {
			ruleItem.LimitByPerIp = LimitByPerIp{
				SourceType: RemoteAddrSourceType,
				HeaderName: "",
			}
		} else {
			return nil, errors.New("the 'limit_by_per_ip' restriction must start with 'from-header-' or be exactly 'from-remote-addr'")
		}
		limitType = LimitByPerIpType
	}

	if limitType == "" {
		return nil, errors.New("only one of 'limit_by_header' and 'limit_by_param' and 'limit_by_consumer' and 'limit_by_cookie' and 'limit_by_per_header' and 'limit_by_per_param' and 'limit_by_per_consumer' and 'limit_by_per_cookie' and 'limit_by_per_ip' can be set")
	}
	ruleItem.LimitType = limitType

	// 初始化configItems
	err := initConfigItems(item, &ruleItem)
	if err != nil {
		return nil, fmt.Errorf("failed to init config items: %w", err)
	}

	return &ruleItem, nil
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
			itemType LimitConfigItemType
			ipNet    *iptree.IPTree
			regexp   *re.Regexp
		)
		if rule.LimitType == LimitByPerIpType {
			var err error
			ipNet, err = util.ParseIPNet(itemKey)
			if err != nil {
				return fmt.Errorf("failed to parse IPNet for key '%s': %w", itemKey, err)
			}
			itemType = IpNetType
		} else if rule.LimitType == LimitByPerHeaderType ||
			rule.LimitType == LimitByPerParamType ||
			rule.LimitType == LimitByPerConsumerType ||
			rule.LimitType == LimitByPerCookieType {
			if itemKey == "*" {
				itemType = AllType
			} else if strings.HasPrefix(itemKey, "regexp:") {
				regexpStr := itemKey[len("regexp:"):]
				var err error
				regexp, err = re.Compile(regexpStr)
				if err != nil {
					return fmt.Errorf("failed to compile regex for key '%s': %w", itemKey, err)
				}
				itemType = RegexpType
			} else {
				return fmt.Errorf("the '%s' restriction must start with 'regexp:' or be exactly '*'", rule.LimitType)
			}
		} else {
			itemType = ExactType
		}

		if configItem, err := createConfigItemFromRate(item, itemType, itemKey, ipNet, regexp); err != nil {
			return err
		} else if configItem != nil {
			configItems = append(configItems, *configItem)
		}
	}
	rule.ConfigItems = configItems
	return nil
}

func createConfigItemFromRate(item gjson.Result, itemType LimitConfigItemType, key string, ipNet *iptree.IPTree, regexp *re.Regexp) (*LimitConfigItem, error) {
	for timeWindowKey, duration := range timeWindows {
		q := item.Get(timeWindowKey)
		if q.Exists() && q.Int() > 0 {
			return &LimitConfigItem{
				ConfigType: itemType,
				Key:        key,
				IpNet:      ipNet,
				Regexp:     regexp,
				Count:      q.Int(),
				TimeWindow: duration,
			}, nil
		}
	}
	return nil, errors.New("one of 'query_per_second', 'query_per_minute', 'query_per_hour', or 'query_per_day' must be set for key: " + key)
}
