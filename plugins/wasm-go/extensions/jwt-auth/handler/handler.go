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
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

func OnHTTPRequestHeaders(ctx wrapper.HttpContext, config cfg.JWTAuthConfig, log wrapper.Log) types.Action {
	// 没有为domina/route配置规则且没有全局启用，直接放行
	if !cfg.Enabled {
		log.Info("authorization is not required")
		return types.ActionContinue
	}

	header := &proxywasmProvider{}

	action := deniedUnauthorizedConsumer
	// 匹配consumer
	for i := range config.Consumers {
		err := consumerVerify(config.Consumers[i], time.Now(), header, log)
		if err != nil {
			log.Warn(err.Error())
			if v, ok := err.(*ErrDenied); ok {
				action = v.denied
			}
			continue
		}

		switch config.GlobalAuthCheck() {

		// 全局生效设置为 true 或全局生效设置未配置
		case cfg.GlobalAuthTrue, cfg.GlobalAuthNoSet:
			if len(config.Allow) == 0 {
				log.Infof("consumer %q authenticated", config.Consumers[i].Name)
				return authenticated(config.Consumers[i].Name)
			}
			fallthrough // 若 allow 列表不为空，则 fallthrough 到需要检查 allow 列表的逻辑中

		// 全局生效设置为 false
		case cfg.GlobalAuthFalse:
			if !contains(config.Consumers[i].Name, config.Allow) {
				log.Warnf("jwt verify failed, consumer %q not allow",
					config.Consumers[i].Name)
				action = deniedUnauthorizedConsumer
				continue
			}
			log.Infof("consumer %q authenticated", config.Consumers[i].Name)
			return authenticated(config.Consumers[i].Name)
		}
	}

	// 拒绝兜底
	log.Warnf("all consumers verify failed")
	return action()
}

func contains(str string, arr []string) bool {
	for _, i := range arr {
		if i == str {
			return true
		}
	}
	return false
}
