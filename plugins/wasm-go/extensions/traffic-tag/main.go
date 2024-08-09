package main

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

const (
	TaggingType   = "taggingType"
	Conditions    = "conditions"
	MatchLogic    = "matchLogic"
	CondKeyType   = "keyType"
	CondKey       = "key"
	CondMatchType = "matchType"
	CondValue     = "value"
	DefaultTagKey = "x-mse-tag"
	DefaultTagVal = "gray"
)

func main() {
	wrapper.SetCtx(
		"traffic-tag",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type TrafficTagConfig struct {
	taggingType string
	percent     uint8
	matchLogic  string
	tagKey      string
	tagValue    string
	conditions  []TagCondition
}

type TagCondition struct {
	keyType   string
	key       string
	matchType string
	value     []string
}

func parseConfig(json gjson.Result, config *TrafficTagConfig, log wrapper.Log) error {
	config.taggingType = json.Get(TaggingType).String()
	// 默认为content类型
	if config.taggingType == "content" || config.taggingType == "" {
		config.matchLogic = strings.ToLower(json.Get(MatchLogic).String())

		if config.matchLogic != "and" && config.matchLogic != "or" {
			return errors.New("invalid matchLogic: " + config.matchLogic)
		}

		// 预留tagKey和tagValue的配置，暂时使用默认值
		// config.tagKey = json.Get("tagKey").String()
		// config.tagValue = json.Get("tagValue").String()
		config.tagKey = DefaultTagKey
		config.tagValue = DefaultTagVal
		if config.tagKey == "" || config.tagValue == "" {
			log.Warn("empty tagKey or tagValue, use default value")
			config.tagKey = DefaultTagKey
			config.tagValue = DefaultTagVal
		}

		conditions := json.Get(Conditions).Array()
		if len(conditions) == 0 {
			return errors.New("empty conditions")
		}
		for _, condition := range conditions {
			keyType := condition.Get(CondKeyType)
			key := condition.Get(CondKey)
			matchType := condition.Get(CondMatchType)
			valueArray := condition.Get(CondValue).Array()
			if !key.Exists() || !matchType.Exists() || !keyType.Exists() || len(valueArray) == 0 {
				log.Criticalf("missing required fields in condition. ")
				return errors.New("missing required fields in condition")
			}

			var valueSlice []string
			for _, item := range valueArray {
				valueSlice = append(valueSlice, item.String())
			}
			config.conditions = append(config.conditions, TagCondition{
				keyType:   keyType.String(),
				key:       key.String(),
				matchType: matchType.String(),
				value:     valueSlice,
			})
		}
	} else if config.taggingType == "percentage" {
		config.percent = uint8(json.Get("percent").Int())

		// 百分比值为0时，不生效
		if config.percent == 0 {
			return errors.New("invalid percentage value")
		}
	} else {
		log.Criticalf("invalid taggingType: %s", config.taggingType)
		return errors.New("invalid taggingType: " + config.taggingType)
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config TrafficTagConfig, log wrapper.Log) types.Action {
	if config.taggingType == "percentage" {
		// TODO 暂时不实现百分比标记
		log.Info("percentage tagging")
		return types.ActionContinue
	}

	for _, condition := range config.conditions {
		conditionKeyValue, err := getConditionValue(condition, log)
		if err != nil {
			log.Debugf("failed to get condition value: %s", err)
			continue
		}

		switch condition.matchType {
		case "==":
			if conditionKeyValue == condition.value[0] && config.matchLogic == "or" {
				addTagHeader(config.tagKey, config.tagValue)
				return types.ActionContinue
			} else if conditionKeyValue != condition.value[0] && config.matchLogic == "and" {
				proxywasm.LogInfof("condition not match: %s != %s", conditionKeyValue, condition.value[0])
				return types.ActionContinue
			}
		case "!=":
			if conditionKeyValue != condition.value[0] && config.matchLogic == "or" {
				addTagHeader(config.tagKey, config.tagValue)
				return types.ActionContinue
			} else if conditionKeyValue == condition.value[0] && config.matchLogic == "and" {
				proxywasm.LogInfof("condition not match: %s == %s", conditionKeyValue, condition.value[0])
				return types.ActionContinue
			}
		case "prefix":
			if strings.HasPrefix(conditionKeyValue, condition.value[0]) && config.matchLogic == "or" {
				addTagHeader(config.tagKey, config.tagValue)
				return types.ActionContinue
			} else if !strings.HasPrefix(conditionKeyValue, condition.value[0]) && config.matchLogic == "and" {
				proxywasm.LogInfof("condition not match: %s not start with %s", conditionKeyValue, condition.value[0])
				return types.ActionContinue
			}
		case "regex":
			// 编译正则表达式
			regex := regexp.MustCompile(condition.value[0])

			if regex.MatchString(conditionKeyValue) && config.matchLogic == "or" {
				addTagHeader(config.tagKey, config.tagValue)
				return types.ActionContinue
			} else if !regex.MatchString(conditionKeyValue) && config.matchLogic == "and" {
				proxywasm.LogInfof("condition not match: %s does not match regex %s", conditionKeyValue, condition.value[0])
				return types.ActionContinue
			}
		case "in":
			isMatch := false
			for _, v := range condition.value {
				if v == conditionKeyValue {
					isMatch = true
					break
				}
			}
			if isMatch && config.matchLogic == "or" {
				addTagHeader(config.tagKey, config.tagValue)
				return types.ActionContinue
			} else if !isMatch && config.matchLogic == "and" {
				proxywasm.LogInfof("condition not match: %s not in %v", conditionKeyValue, condition.value)
				return types.ActionContinue
			}
		case "notIn":
			isMatch := false
			for _, v := range condition.value {
				if v == conditionKeyValue {
					isMatch = true
					break
				}
			}
			if !isMatch && config.matchLogic == "or" {
				addTagHeader(config.tagKey, config.tagValue)
				return types.ActionContinue
			} else if isMatch && config.matchLogic == "and" {
				proxywasm.LogInfof("condition not match: %s in %v", conditionKeyValue, condition.value)
				return types.ActionContinue
			}
		case "percentage":
			// 默认从请求header中获取百分比值
			percentHeaderVal, err := proxywasm.GetHttpRequestHeader(condition.key)
			if err != nil {
				log.Warnf("failed to get percent header value: %s", err)
				return types.ActionContinue
			}
			log.Debugf("condition.value: %v", condition.value)
			percentThresholdInt, err := strconv.Atoi(condition.value[0])
			if err != nil {
				log.Warnf("failed to convert percent threshold to int: %s", err)
				return types.ActionContinue
			}

			// hash(value) % 100 < percent
			hash := sha256.Sum256([]byte(percentHeaderVal))
			hashInt64 := int64(binary.BigEndian.Uint64(hash[:8]) % 100)
			proxywasm.LogDebugf("hashInt: %d, percentThresholdInt: %d", hashInt64, percentThresholdInt)

			if hashInt64 < int64(percentThresholdInt) && config.matchLogic == "or" {
				addTagHeader(config.tagKey, config.tagValue)
				return types.ActionContinue
			} else if hashInt64 >= int64(percentThresholdInt) && config.matchLogic == "and" {
				proxywasm.LogInfof("condition not match: %d >= %d", hashInt64, percentThresholdInt)
				return types.ActionContinue
			}
		default:
			log.Criticalf("invalid matchType: %s", condition.matchType)
			return types.ActionContinue
		}
	}
	// 条件全部匹配
	if config.matchLogic == "and" {
		addTagHeader(config.tagKey, config.tagValue)
	}
	return types.ActionContinue
}

func getConditionValue(condition TagCondition, log wrapper.Log) (string, error) {
	var conditionKeyValue string

	switch condition.keyType {
	case "header":
		log.Debug("Hit header condition")
		conditionKeyValue, _ = proxywasm.GetHttpRequestHeader(condition.key)
	case "cookie":
		log.Debug("Hit cookie condition")
		requestCookie, _ := proxywasm.GetHttpRequestHeader("cookie")
		ckv, found := parseCookie(requestCookie, condition.key)
		if !found {
			return "", errors.New("cookie not found")
		}
		conditionKeyValue = ckv
	case "parameter":
		log.Debug("Hit parameter condition")
		urlStr, err := getFullRequestURL()
		if err != nil {
			return "", err
		}
		conditionKeyValue, err = getQueryParameter(urlStr, condition.key)
		if err != nil {
			return "", err
		}
	}

	return conditionKeyValue, nil
}

func getFullRequestURL() (string, error) {
	path, _ := proxywasm.GetHttpRequestHeader(":path")
	return path, nil
}

func getQueryParameter(urlStr, paramKey string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	values, ok := u.Query()[paramKey]
	if !ok {
		return "", fmt.Errorf("parameter %s not found", paramKey)
	}
	return values[0], nil
}

func parseCookie(cookieHeader string, key string) (string, bool) {
	cookies := strings.Split(cookieHeader, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if strings.HasPrefix(cookie, key+"=") {
			parts := strings.SplitN(cookie, "=", 2)
			if len(parts) == 2 {
				return parts[1], true
			}
		}
	}
	return "", false
}

func addTagHeader(key string, value string) {
	if err := proxywasm.AddHttpRequestHeader(key, value); err != nil {
		proxywasm.LogCriticalf("failed to add tag header: %s", err)
		return
	}
	proxywasm.LogInfof("add tag header: %s, value: %s", key, value)
}
