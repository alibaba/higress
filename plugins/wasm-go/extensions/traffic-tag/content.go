// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/log"
)

func onContentRequestHeaders(conditionGroups []ConditionGroup, log log.Log) bool {
	for _, cg := range conditionGroups {
		if matchCondition(&cg, log) {
			addTagHeader(cg.HeaderName, cg.HeaderValue, log)
			return true
		}
	}

	return false
}

// matchCondition matches the single condition group
func matchCondition(conditionGroup *ConditionGroup, log log.Log) bool {
	for _, condition := range conditionGroup.Conditions {
		conditionKeyValue, err := getConditionValue(condition, log)
		if err != nil {
			log.Debugf("failed to get condition value: %s", err)
			if conditionGroup.Logic == "and" {
				return false
			}
			continue
		}

		switch condition.Operator {
		case Op_Equal:
			if conditionKeyValue == condition.Value[0] && conditionGroup.Logic == "or" {
				log.Debugf("condition match: %s == %s", conditionKeyValue, condition.Value[0])
				return true
			} else if conditionKeyValue != condition.Value[0] && conditionGroup.Logic == "and" {
				return false
			}
		case Op_NotEqual:
			if conditionKeyValue != condition.Value[0] && conditionGroup.Logic == "or" {
				log.Debugf("condition match: %s != %s", conditionKeyValue, condition.Value[0])
				return true
			} else if conditionKeyValue == condition.Value[0] && conditionGroup.Logic == "and" {
				return false
			}
		case Op_Prefix:
			if strings.HasPrefix(conditionKeyValue, condition.Value[0]) && conditionGroup.Logic == "or" {
				log.Debugf("condition match: %s prefix %s", conditionKeyValue, condition.Value[0])
				return true
			} else if !strings.HasPrefix(conditionKeyValue, condition.Value[0]) && conditionGroup.Logic == "and" {
				return false
			}
		case Op_Regex:
			if _, ok := regexCache[condition.Value[0]]; !ok {
				err := compileRegex(condition.Value[0])
				if err != nil {
					log.Warnf("failed to compile regex: %s", err)
					return false
				}
			}
			regex := regexCache[condition.Value[0]]

			if regex.MatchString(conditionKeyValue) && conditionGroup.Logic == "or" {
				log.Debugf("condition match: %s regex %s", conditionKeyValue, condition.Value[0])
				return true
			} else if !regex.MatchString(conditionKeyValue) && conditionGroup.Logic == "and" {
				log.Debugf("condition not match: %s regex %s", conditionKeyValue, condition.Value[0])
				return false
			}
		case Op_In:
			isMatch := false
			for _, v := range condition.Value {
				if v == conditionKeyValue {
					isMatch = true
					break
				}
			}
			if isMatch && conditionGroup.Logic == "or" {
				log.Debugf("condition match: %s in %v", conditionKeyValue, condition.Value)
				return true
			} else if !isMatch && conditionGroup.Logic == "and" {
				return false
			}
		case Op_NotIn:
			isMatch := false
			for _, v := range condition.Value {
				if v == conditionKeyValue {
					isMatch = true
					break
				}
			}
			if !isMatch && conditionGroup.Logic == "or" {
				log.Debugf("condition match: %s not in %v", conditionKeyValue, condition.Value)
				return true
			} else if isMatch && conditionGroup.Logic == "and" {
				return false
			}
		case Op_Percent:
			percentThresholdInt, err := strconv.Atoi(condition.Value[0])
			if err != nil {
				log.Infof("invalid percent threshold config: %s", err)
				return false
			}

			// hash(value) % 100 < percent
			hash := sha256.Sum256([]byte(conditionKeyValue))
			hashInt64 := int64(binary.BigEndian.Uint64(hash[:8]) % 100)
			log.Debugf("hashInt64: %d", hashInt64)

			if hashInt64 < int64(percentThresholdInt) && conditionGroup.Logic == "or" {
				log.Debugf("condition match: %d < %d", hashInt64, percentThresholdInt)
				return true
			} else if hashInt64 >= int64(percentThresholdInt) && conditionGroup.Logic == "and" {
				log.Debugf("condition not match: %d >= %d", hashInt64, percentThresholdInt)
				return false
			}

		default:
			log.Criticalf("invalid operator: %s", condition.Operator)
			return false
		}
	}
	return len(conditionGroup.Conditions) > 0 && conditionGroup.Logic == "and" // all conditions are matched
}

func getConditionValue(condition ConditionRule, log log.Log) (string, error) {
	// log.Debugf("conditionType: %s, key: %s", condition.ConditionType, condition.Key)

	switch condition.ConditionType {
	case Type_Header:
		// log.Debug("Hit header condition")
		log.Debugf("Hit header condition, key: %s", condition.Key)
		return proxywasm.GetHttpRequestHeader(condition.Key)
	case Type_Cookie:
		log.Debugf("Hit cookie condition, key: %s", condition.Key)
		requestCookie, err := proxywasm.GetHttpRequestHeader(Type_Cookie)
		ckv, found := parseCookie(requestCookie, condition.Key)
		if !found {
			return "", errors.New("cookie not found")
		}
		return ckv, err
	case Type_Parameter:
		log.Debugf("Hit parameter condition, key: %s", condition.Key)
		urlStr, err := getFullRequestURL()
		if err != nil {
			return "", err
		}
		return getQueryParameter(urlStr, condition.Key)
	default:
		log.Criticalf("invalid conditionType: %s", condition.ConditionType)
		return "", errors.New("invalid conditionType: " + condition.ConditionType)
	}

}
