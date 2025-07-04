// Copyright (c) 2023 Alibaba Group Holding Ltd.
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

package handler

import (
	"time"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/jwt-auth/config"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// jwt-auth 插件认证逻辑与 basic-auth 一致：
// - global_auth == true 开启全局生效：
//   - 若当前 domain/route 未配置 allow 列表，即未配置该插件：则在所有 consumers 中查找，如果找到则认证通过，否则认证失败 (1*)
//   - 若当前 domain/route 配置了该插件：则在 allow 列表中查找，如果找到则认证通过，否则认证失败
//
// - global_auth == false 非全局生效：(2*)
//   - 若当前 domain/route 未配置该插件：则直接放行
//   - 若当前 domain/route 配置了该插件：则在 allow 列表中查找，如果找到则认证通过，否则认证失败
//
// - global_auth 未设置：
//   - 若没有一个 domain/route 配置该插件：则遵循 (1*)
//   - 若有至少一个 domain/route 配置该插件：则遵循 (2*)
//
// https://github.com/alibaba/higress/blob/e09edff827b94fa5bcc149bbeadc905361100c2a/plugins/wasm-go/extensions/basic-auth/main.go#L191
func OnHTTPRequestHeaders(ctx wrapper.HttpContext, config cfg.JWTAuthConfig, log log.Log) types.Action {
	var (
		noAllow            = len(config.Allow) == 0 // 未配置 allow 列表，表示插件在该 domain/route 未生效
		globalAuthNoSet    = config.GlobalAuthCheck() == cfg.GlobalAuthNoSet
		globalAuthSetTrue  = config.GlobalAuthCheck() == cfg.GlobalAuthTrue
		globalAuthSetFalse = config.GlobalAuthCheck() == cfg.GlobalAuthFalse
	)

	// 不需要认证而直接放行的情况：
	// - global_auth == false 且 当前 domain/route 未配置该插件
	// - global_auth 未设置 且 有至少一个 domain/route 配置该插件 且 当前 domain/route 未配置该插件
	if globalAuthSetFalse || (cfg.RuleSet && globalAuthNoSet) {
		if noAllow {
			log.Info("authorization is not required")
			return types.ActionContinue
		}
	}

	header := &proxywasmProvider{}
	actionMap := map[string]func() types.Action{}
	unAuthzConsumer := ""

	// 匹配consumer
	for i := range config.Consumers {
		err := consumerVerify(config.Consumers[i], time.Now(), header, log)
		if err != nil {
			log.Warn(err.Error())
			if v, ok := err.(*ErrDenied); ok {
				actionMap[config.Consumers[i].Name] = v.denied
			}
			continue
		}

		// 全局生效：
		// - global_auth == true 且 当前 domain/route 未配置该插件
		// - global_auth 未设置 且 没有任何一个 domain/route 配置该插件
		if (globalAuthSetTrue && noAllow) || (globalAuthNoSet && !cfg.RuleSet) {
			log.Infof("consumer %q authenticated", config.Consumers[i].Name)
			return authenticated(config.Consumers[i].Name)
		}

		// 全局生效，但当前 domain/route 配置了 allow 列表
		if globalAuthSetTrue && !noAllow {
			if !contains(config.Consumers[i].Name, config.Allow) {
				log.Warnf("jwt verify failed, consumer %q not allow",
					config.Consumers[i].Name)
				actionMap[config.Consumers[i].Name] = deniedUnauthorizedConsumer
				unAuthzConsumer = config.Consumers[i].Name
				continue
			}
			log.Infof("consumer %q authenticated", config.Consumers[i].Name)
			return authenticated(config.Consumers[i].Name)
		}

		// 非全局生效
		if globalAuthSetFalse || (globalAuthNoSet && cfg.RuleSet) {
			if !noAllow { // 配置了 allow 列表
				if !contains(config.Consumers[i].Name, config.Allow) {
					log.Warnf("jwt verify failed, consumer %q not allow",
						config.Consumers[i].Name)
					actionMap[config.Consumers[i].Name] = deniedUnauthorizedConsumer
					unAuthzConsumer = config.Consumers[i].Name
					continue
				}
				log.Infof("consumer %q authenticated", config.Consumers[i].Name)
				return authenticated(config.Consumers[i].Name)
			}
		}

		// switch config.GlobalAuthCheck() {

		// case cfg.GlobalAuthNoSet:
		// 	if !cfg.RuleSet {
		// 		log.Infof("consumer %q authenticated", config.Consumers[i].Name)
		// 		return authenticated(config.Consumers[i].Name)
		// 	}
		// case cfg.GlobalAuthTrue:
		// 	if len(config.Allow) == 0 {
		// 		log.Infof("consumer %q authenticated", config.Consumers[i].Name)
		// 		return authenticated(config.Consumers[i].Name)
		// 	}
		// 	fallthrough // 若 allow 列表不为空，则 fallthrough 到需要检查 allow 列表的逻辑中

		// // 全局生效设置为 false
		// case cfg.GlobalAuthFalse:
		// 	if !contains(config.Consumers[i].Name, config.Allow) {
		// 		log.Warnf("jwt verify failed, consumer %q not allow",
		// 			config.Consumers[i].Name)
		// 		actionMap[config.Consumers[i].Name] = deniedUnauthorizedConsumer
		// 		unAuthzConsumer = config.Consumers[i].Name
		// 		continue
		// 	}
		// 	log.Infof("consumer %q authenticated", config.Consumers[i].Name)
		// 	return authenticated(config.Consumers[i].Name)
		// }
	}

	if len(config.Allow) == 1 {
		if unAuthzConsumer != "" {
			log.Warnf("consumer %q denied", unAuthzConsumer)
			return deniedUnauthorizedConsumer()
		}
		if v, ok := actionMap[config.Allow[0]]; ok {
			log.Warnf("consumer %q denied", config.Allow[0])
			return v()
		}
	}

	// 拒绝兜底
	log.Warnf("all consumers verify failed")
	return deniedNotAllow()
}

func contains(str string, arr []string) bool {
	for _, i := range arr {
		if i == str {
			return true
		}
	}
	return false
}
