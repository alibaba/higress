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
	"encoding/json"
	"fmt"
	"time"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/jwt-auth/config"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

var protectionSpace = "MSE Gateway" // 认证失败时，返回响应头 WWW-Authenticate: JWT realm=MSE Gateway

type ErrDenied struct {
	msg    string
	denied func() types.Action
}

func (e *ErrDenied) Error() string {
	return e.msg
}

func consumerVerify(consumer *cfg.Consumer, verifyTime time.Time, log wrapper.Log) error {
	tokenStr := extractToken(*consumer.KeepToken, consumer, log)
	if tokenStr == "" {
		return &ErrDenied{
			msg:    fmt.Sprintf("jwt is missing, consumer: %s", consumer.Name),
			denied: deniedJWTMissing,
		}
	}

	// 当前版本的higress暂不支持jwe，此处用ParseSigned
	token, err := jwt.ParseSigned(tokenStr,
		[]jose.SignatureAlgorithm{
			jose.EdDSA,
			// HMAC currently unprovided
			// jose.HS256, jose.HS384, jose.HS512,
			jose.RS256, jose.RS384, jose.RS512,
			jose.ES256, jose.ES384, jose.ES512})
	if err != nil {
		return &ErrDenied{
			msg: fmt.Sprintf("jwt parse failed, consumer: %s, token: %s, reason: %s",
				consumer.Name,
				tokenStr,
				err.Error(),
			),
			denied: deniedJWTVerificationFails,
		}
	}

	// 此处可以直接使用 JSON 反序列 jwks
	keys := jose.JSONWebKeySet{}
	err = json.Unmarshal([]byte(consumer.JWKs), &keys)
	if err != nil {
		return &ErrDenied{
			msg: fmt.Sprintf("jwt parse failed, consumer: %s, token: %s, reason: %s",
				consumer.Name,
				tokenStr,
				err.Error(),
			),
			denied: deniedJWTVerificationFails,
		}
	}

	out := jwt.Claims{}
	rawClaims := map[string]any{}

	// Claims 支持直接传入 jose 的 jwks 并自动选择合适的公钥
	// 无需额外调用verify，claims内部已进行验证
	err = token.Claims(keys, &out)
	if err != nil {
		return &ErrDenied{
			msg: fmt.Sprintf("jwt verify failed, consumer: %s, token: %s, reason: %s",
				consumer.Name,
				tokenStr,
				err.Error(),
			),
			denied: deniedJWTVerificationFails,
		}
	}

	if out.Issuer != consumer.Issuer {
		return &ErrDenied{
			msg: fmt.Sprintf("jwt verify failed, consumer: %s, token: %s, reason: issuer does not equal",
				consumer.Name,
				tokenStr,
			),
			denied: deniedJWTVerificationFails,
		}
	}

	// 检查是否过期
	err = out.ValidateWithLeeway(
		jwt.Expected{
			Issuer: consumer.Issuer,
			Time:   verifyTime,
		},
		time.Duration(*consumer.ClockSkewSeconds)*time.Second,
	)
	if err != nil {
		return &ErrDenied{
			msg: fmt.Sprintf("jwt verify failed, consumer: %s, token: %s, reason: %s",
				consumer.Name,
				tokenStr,
				err.Error(),
			),
			denied: deniedJWTExpired,
		}
	}

	if consumer.ClaimsToHeaders != nil {
		claimsToHeader(rawClaims, *consumer.ClaimsToHeaders)
	}
	return nil
}

func deniedJWTMissing() types.Action {
	_ = proxywasm.SendHttpResponse(401, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by JWT Auth check. JWT is missing."), -1)
	return types.ActionContinue
}

func deniedJWTExpired() types.Action {
	_ = proxywasm.SendHttpResponse(401, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by JWT Auth check. JWT is expried."), -1)
	return types.ActionContinue
}

func deniedJWTVerificationFails() types.Action {
	_ = proxywasm.SendHttpResponse(401, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by JWT Auth check. JWT verification fails"), -1)
	return types.ActionContinue
}

func deniedUnauthorizedConsumer() types.Action {
	_ = proxywasm.SendHttpResponse(403, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by JWT Auth check. Unauthorized consumer."), -1)
	return types.ActionContinue
}

func authenticated(name string) types.Action {
	_ = proxywasm.AddHttpRequestHeader("X-Mse-Consumer", name)
	return types.ActionContinue
}

func WWWAuthenticateHeader(realm string) [][2]string {
	return [][2]string{
		{"WWW-Authenticate", fmt.Sprintf("JWT realm=%s", realm)},
	}
}
