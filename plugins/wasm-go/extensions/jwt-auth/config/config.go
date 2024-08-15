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

package config

var (
	// DefaultClaimToHeaderOverride 是 claim_to_override 中 override 字段的默认值
	DefaultClaimToHeaderOverride = true

	// DefaultClockSkewSeconds 是 ClockSkewSeconds 的默认值
	DefaultClockSkewSeconds = int64(60)

	// DefaultKeepToken 是 KeepToken 的默认值
	DefaultKeepToken = true

	// DefaultFromHeader 是 from_header 的默认值
	DefaultFromHeader = []FromHeader{{
		Name:        "Authorization",
		ValuePrefix: "Bearer ",
	}}

	// DefaultFromParams 是 from_params 的默认值
	DefaultFromParams = []string{"access_token"}

	// DefaultFromCookies 是 from_cookies 的默认值
	DefaultFromCookies = []string{}
)

// JWTAuthConfig defines the struct of the global config of higress wasm plugin jwt-auth.
// https://higress.io/zh-cn/docs/plugins/jwt-auth
type JWTAuthConfig struct {
	// 全局配置
	//
	// Consumers 配置服务的调用者，用于对请求进行认证
	Consumers []*Consumer `json:"consumers"`

	// 全局配置
	//
	// GlobalAuth 若配置为true，则全局生效认证机制;
	// 若配置为false，则只对做了配置的域名和路由生效认证机制;
	// 若不配置则仅当没有域名和路由配置时全局生效（兼容机制）
	GlobalAuth *bool `json:"global_auth,omitempty"`

	// 域名和路由级配置
	//
	// Allow 对于符合匹配条件的请求，配置允许访问的consumer名称
	Allow []string `json:"allow"`
}

// Consumer 配置服务的调用者，用于对请求进行认证
type Consumer struct {
	// Name 配置该consumer的名称
	Name string `json:"name"`

	// JWKs 指定的json格式字符串，是由验证JWT中签名的公钥（或对称密钥）组成的Json Web Key Set
	//
	// https://www.rfc-editor.org/rfc/rfc7517
	JWKs string `json:"jwks"`

	// Issuer JWT的签发者，需要和payload中的iss字段保持一致
	Issuer string `json:"issuer"`

	// ClaimsToHeaders 抽取JWT的payload中指定字段，设置到指定的请求头中转发给后端
	ClaimsToHeaders *[]ClaimsToHeader `json:"claims_to_headers,omitempty"`

	// FromHeaders 从指定的请求头中抽取JWT
	//
	// 默认值为 [{"name":"Authorization","value_prefix":"Bearer "}]
	//
	// 只有当from_headers,from_params,from_cookies均未配置时，才会使用默认值
	FromHeaders *[]FromHeader `json:"from_headers,omitempty"`

	// FromParams 从指定的URL参数中抽取JWT
	//
	// 默认值为 access_token
	//
	// 只有当from_headers,from_params,from_cookies均未配置时，才会使用默认值
	FromParams *[]string `json:"from_params,omitempty"`

	// FromCookies 从指定的cookie中抽取JWT
	FromCookies *[]string `json:"from_cookies,omitempty"`

	// ClockSkewSeconds 校验JWT的exp和iat字段时允许的时钟偏移量，单位为秒
	//
	// 默认值为 60
	ClockSkewSeconds *int64 `json:"clock_skew_seconds,omitempty"`

	// KeepToken 转发给后端时是否保留JWT
	//
	// 默认值为 true
	KeepToken *bool `json:"keep_token,omitempty"`
}

// ClaimsToHeader 抽取JWT的payload中指定字段，设置到指定的请求头中转发给后端
type ClaimsToHeader struct {
	// Claim JWT payload中的指定字段，要求必须是字符串或无符号整数类型
	Claim string `json:"claim"`

	// Header 从payload取出字段的值设置到这个请求头中，转发给后端
	Header string `json:"header"`

	// Override true时，存在同名请求头会进行覆盖；false时，追加同名请求头
	//
	// 默认值为 true
	Override *bool `json:"override,omitempty"`
}

// FromHeader 从指定的请求头中抽取JWT
type FromHeader struct {
	// Name 抽取JWT的请求header
	Name string `json:"name"`
	// ValuePrefix 对请求header的value去除此前缀，剩余部分作为JWT
	ValuePrefix string `json:"value_prefix"`
}
